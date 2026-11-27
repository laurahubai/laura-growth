package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const timeoutFornecedor = 15 * time.Second

var (
	servicosValidos = map[string]bool{"texto": true, "imagem": true}
	origensValidas  = map[string]bool{"lido": true, "manual": true}
	unidadesValidas = map[string]bool{"token_entrada": true, "token_saida": true, "imagem": true}
	regexDataISO    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// ---- Formato das respostas (espelha packages/contratos/src/admin.ts) ----

type ModeloLido struct {
	Identificador string `json:"identificador"`
	JaCadastrado  bool   `json:"jaCadastrado"`
}

type RespostaLerModelos struct {
	Modelos []ModeloLido `json:"modelos"`
}

type PrecosVigentes struct {
	TokenEntrada *float64 `json:"tokenEntrada"`
	TokenSaida   *float64 `json:"tokenSaida"`
	Imagem       *float64 `json:"imagem"`
}

type Modelo struct {
	ID              string         `json:"id"`
	FornecedorID    string         `json:"fornecedorId"`
	FornecedorNome  string         `json:"fornecedorNome"`
	Identificador   string         `json:"identificador"`
	Servico         string         `json:"servico"`
	Origem          string         `json:"origem"`
	LimiteMensalUsd *float64       `json:"limiteMensalUsd"`
	AvisoPercentual *int           `json:"avisoPercentual"`
	Ativo           bool           `json:"ativo"`
	GastoMesUsd     float64        `json:"gastoMesUsd"`
	Situacao        string         `json:"situacao"`
	Simulado        bool           `json:"simulado"`
	PrecosVigentes  PrecosVigentes `json:"precosVigentes"`
}

type RespostaModelos struct {
	Modelos []Modelo `json:"modelos"`
}

type PedidoNovoModelo struct {
	Identificador   string   `json:"identificador"`
	Servico         string   `json:"servico"`
	Origem          string   `json:"origem"`
	LimiteMensalUsd *float64 `json:"limiteMensalUsd"`
	AvisoPercentual *int     `json:"avisoPercentual"`
}

// campoOpcional diferencia campo omitido de null explícito em PATCH.
type campoOpcional[T any] struct {
	definido bool
	valor    *T
}

func (c *campoOpcional[T]) UnmarshalJSON(data []byte) error {
	c.definido = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		c.valor = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	c.valor = &v
	return nil
}

type PedidoEditarModelo struct {
	Servico         *string                `json:"servico"`
	LimiteMensalUsd campoOpcional[float64] `json:"limiteMensalUsd"`
	AvisoPercentual campoOpcional[int]     `json:"avisoPercentual"`
	Ativo           *bool                  `json:"ativo"`
}

type Preco struct {
	ID             string    `json:"id"`
	ModeloID       string    `json:"modeloId"`
	Unidade        string    `json:"unidade"`
	PrecoUsd       float64   `json:"precoUsd"`
	VigenciaInicio string    `json:"vigenciaInicio"`
	CriadoEm       time.Time `json:"criadoEm"`
}

type RespostaPrecos struct {
	Precos []Preco `json:"precos"`
}

type PedidoNovoPreco struct {
	Unidade        string   `json:"unidade"`
	PrecoUsd       *float64 `json:"precoUsd"`
	VigenciaInicio string   `json:"vigenciaInicio"`
}

// ---- Ler modelos disponíveis (AC38) ----

func (a *API) lerModelosFornecedor(w http.ResponseWriter, r *http.Request, _ contextoSessao) {
	id := r.PathValue("id")
	var tipo, endereco string
	var chaveCifrada []byte
	var ativo bool
	err := a.banco.QueryRow(r.Context(), `
		SELECT tipo, endereco, chave_cifrada, ativo
		FROM ia_fornecedores WHERE id::text = $1`, id).
		Scan(&tipo, &endereco, &chaveCifrada, &ativo)
	if errors.Is(err, pgx.ErrNoRows) {
		erro(w, 404, "nao_encontrado", "Fornecedor não encontrado.")
		return
	}
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if !ativo {
		erro(w, 409, "fornecedor_inativo", "Este fornecedor está desativado. Ative antes de ler os modelos.")
		return
	}

	var chave string
	if tipo != "simulado" {
		enderecoValidado, err := a.resolverEnderecoFornecedor(r.Context(), tipo, endereco)
		if err != nil {
			erro(w, 400, "endereco_invalido", err.Error())
			return
		}
		endereco = enderecoValidado
		if a.cifrador == nil {
			falhaInterna(w, errors.New("cifrador não configurado"))
			return
		}
		chave, err = a.cifrador.decifrar(id, chaveCifrada)
		if err != nil {
			falhaInterna(w, err)
			return
		}
	}

	con, err := novoConector(tipo, endereco, chave, a.clienteHTTPFornecedor(timeoutFornecedor))
	if err != nil {
		responderErroFornecedor(w, id, err)
		return
	}
	ids, err := con.listarModelos(r.Context())
	if err != nil {
		responderErroFornecedor(w, id, err)
		return
	}

	linhas, err := a.banco.Query(r.Context(), `SELECT identificador FROM ia_modelos WHERE fornecedor_id::text = $1`, id)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	cadastrados := map[string]bool{}
	for linhas.Next() {
		var ident string
		if err := linhas.Scan(&ident); err != nil {
			linhas.Close()
			falhaInterna(w, err)
			return
		}
		cadastrados[ident] = true
	}
	if err := linhas.Err(); err != nil {
		linhas.Close()
		falhaInterna(w, err)
		return
	}
	linhas.Close()

	modelos := make([]ModeloLido, 0, len(ids))
	for _, ident := range ids {
		modelos = append(modelos, ModeloLido{
			Identificador: ident,
			JaCadastrado:  cadastrados[ident],
		})
	}
	responder(w, 200, RespostaLerModelos{Modelos: modelos})
}

func responderErroFornecedor(w http.ResponseWriter, fornecedorID string, err error) {
	var ef *erroFornecedor
	if errors.As(err, &ef) {
		slog.Warn("falha ao consultar fornecedor", "fornecedorId", fornecedorID, "codigo", ef.codigo)
		erro(w, 422, ef.codigo, ef.mensagem)
		return
	}
	slog.Warn("falha inesperada ao consultar fornecedor", "fornecedorId", fornecedorID)
	erro(w, 422, "fornecedor_erro", "Não foi possível completar a consulta no fornecedor.")
}

// ---- Modelos ----

func (a *API) listarModelos(w http.ResponseWriter, r *http.Request, _ contextoSessao) {
	fornecedorID := r.PathValue("id")
	var existe bool
	if err := a.banco.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM ia_fornecedores WHERE id::text = $1)`, fornecedorID).Scan(&existe); err != nil {
		falhaInterna(w, err)
		return
	}
	if !existe {
		erro(w, 404, "nao_encontrado", "Fornecedor não encontrado.")
		return
	}
	modelos, err := a.consultarModelos(r.Context(), fornecedorID, "")
	if err != nil {
		falhaInterna(w, err)
		return
	}
	responder(w, 200, RespostaModelos{Modelos: modelos})
}

func (a *API) criarModelo(w http.ResponseWriter, r *http.Request, _ contextoSessao) {
	fornecedorID := r.PathValue("id")
	var tipoFornecedor string
	err := a.banco.QueryRow(r.Context(), `SELECT tipo FROM ia_fornecedores WHERE id::text = $1`, fornecedorID).Scan(&tipoFornecedor)
	if errors.Is(err, pgx.ErrNoRows) {
		erro(w, 404, "nao_encontrado", "Fornecedor não encontrado.")
		return
	}
	if err != nil {
		falhaInterna(w, err)
		return
	}

	var p PedidoNovoModelo
	if !lerJSON(w, r, &p) {
		return
	}
	identificador := strings.TrimSpace(p.Identificador)
	if n := utf8.RuneCountInString(identificador); n == 0 || n > 200 {
		erro(w, 400, "dados_invalidos", "Informe o identificador do modelo (até 200 caracteres).")
		return
	}
	if !servicosValidos[p.Servico] {
		erro(w, 400, "dados_invalidos", "Serviço inválido. Use texto ou imagem.")
		return
	}
	if tipoFornecedor == "anthropic" && p.Servico == "imagem" {
		erro(w, 400, "servico_invalido", "A Anthropic não gera imagens.")
		return
	}
	if !origensValidas[p.Origem] {
		erro(w, 400, "dados_invalidos", "Origem inválida. Use lido ou manual.")
		return
	}

	var limiteSQL *string
	if p.LimiteMensalUsd != nil {
		s, ok := formatarLimiteUsd(*p.LimiteMensalUsd)
		if !ok {
			erro(w, 400, "dados_invalidos", "Informe um limite mensal maior que zero.")
			return
		}
		limiteSQL = &s
	}
	if p.AvisoPercentual != nil && (*p.AvisoPercentual < 1 || *p.AvisoPercentual > 100) {
		erro(w, 400, "dados_invalidos", "O percentual de aviso deve ficar entre 1 e 100.")
		return
	}

	var modeloID string
	err = a.banco.QueryRow(r.Context(), `
		INSERT INTO ia_modelos (fornecedor_id, identificador, servico, origem, limite_mensal_usd, aviso_percentual, ativo)
		VALUES ($1, $2, $3, $4, $5, $6, true)
		RETURNING id::text`,
		fornecedorID, identificador, p.Servico, p.Origem, limiteSQL, p.AvisoPercentual).Scan(&modeloID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			erro(w, 409, "modelo_repetido", "Este modelo já está cadastrado neste fornecedor.")
			return
		}
		falhaInterna(w, err)
		return
	}

	m, err := a.carregarModelo(r.Context(), modeloID)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	responder(w, 201, m)
}

func (a *API) editarModelo(w http.ResponseWriter, r *http.Request, _ contextoSessao) {
	id := r.PathValue("id")
	var p PedidoEditarModelo
	if !lerJSON(w, r, &p) {
		return
	}

	var servico, tipoFornecedor string
	var limiteStr *string
	var aviso *int
	var ativo bool
	err := a.banco.QueryRow(r.Context(), `
		SELECT m.servico, f.tipo, m.limite_mensal_usd::text, m.aviso_percentual, m.ativo
		FROM ia_modelos m JOIN ia_fornecedores f ON f.id = m.fornecedor_id
		WHERE m.id::text = $1`, id).
		Scan(&servico, &tipoFornecedor, &limiteStr, &aviso, &ativo)
	if errors.Is(err, pgx.ErrNoRows) {
		erro(w, 404, "nao_encontrado", "Modelo não encontrado.")
		return
	}
	if err != nil {
		falhaInterna(w, err)
		return
	}

	if p.Servico != nil {
		if !servicosValidos[*p.Servico] {
			erro(w, 400, "dados_invalidos", "Serviço inválido. Use texto ou imagem.")
			return
		}
		if tipoFornecedor == "anthropic" && *p.Servico == "imagem" {
			erro(w, 400, "servico_invalido", "A Anthropic não gera imagens.")
			return
		}
		if *p.Servico != servico {
			linhas, err := a.banco.Query(r.Context(), `
				SELECT atividade FROM ia_rotas
				WHERE modelo_principal_id::text = $1 OR modelo_reserva_id::text = $1
				ORDER BY atividade`, id)
			if err != nil {
				falhaInterna(w, err)
				return
			}
			var emUso []string
			for linhas.Next() {
				var cod string
				if err := linhas.Scan(&cod); err != nil {
					linhas.Close()
					falhaInterna(w, err)
					return
				}
				nomeAtv := cod
				if def, ok := mapaAtividades[cod]; ok {
					nomeAtv = def.Nome
				}
				emUso = append(emUso, nomeAtv)
			}
			if err := linhas.Err(); err != nil {
				linhas.Close()
				falhaInterna(w, err)
				return
			}
			linhas.Close()
			if len(emUso) > 0 {
				erro(w, 409, "servico_incompativel", "Não é possível trocar o serviço: modelo em uso em "+strings.Join(emUso, ", ")+".")
				return
			}
		}
		servico = *p.Servico
	}
	if p.LimiteMensalUsd.definido {
		if p.LimiteMensalUsd.valor == nil {
			limiteStr = nil
		} else {
			s, ok := formatarLimiteUsd(*p.LimiteMensalUsd.valor)
			if !ok {
				erro(w, 400, "dados_invalidos", "Informe um limite mensal maior que zero.")
				return
			}
			limiteStr = &s
		}
	}
	if p.AvisoPercentual.definido {
		if p.AvisoPercentual.valor == nil {
			aviso = nil
		} else {
			v := *p.AvisoPercentual.valor
			if v < 1 || v > 100 {
				erro(w, 400, "dados_invalidos", "O percentual de aviso deve ficar entre 1 e 100.")
				return
			}
			aviso = &v
		}
	}
	if p.Ativo != nil {
		ativo = *p.Ativo
	}

	if _, err := a.banco.Exec(r.Context(), `
		UPDATE ia_modelos
		SET servico = $2, limite_mensal_usd = $3, aviso_percentual = $4, ativo = $5
		WHERE id::text = $1`,
		id, servico, limiteStr, aviso, ativo); err != nil {
		falhaInterna(w, err)
		return
	}

	m, err := a.carregarModelo(r.Context(), id)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	responder(w, 200, m)
}

func (a *API) carregarModelo(ctx context.Context, modeloID string) (Modelo, error) {
	lista, err := a.consultarModelos(ctx, "", modeloID)
	if err != nil {
		return Modelo{}, err
	}
	if len(lista) == 0 {
		return Modelo{}, pgx.ErrNoRows
	}
	return lista[0], nil
}

func (a *API) consultarModelos(ctx context.Context, fornecedorID, modeloID string) ([]Modelo, error) {
	agora := a.agora()
	inicioMes := inicioMesSaoPaulo(agora)
	proximoMes := inicioMes.AddDate(0, 1, 0)
	hoje := agora.In(fusoSaoPaulo).Format("2006-01-02")

	query := `
		SELECT
			m.id::text, m.fornecedor_id::text, f.nome, f.tipo,
			m.identificador, m.servico, m.origem,
			m.limite_mensal_usd::text, m.aviso_percentual, m.ativo,
			f.limite_mensal_usd::text, f.aviso_percentual,
			COALESCE((
				SELECT sum(e.custo_usd)
				FROM ia_execucoes e
				WHERE e.modelo_id = m.id AND e.criado_em >= $1 AND e.criado_em < $2
			), 0)::text AS gasto_mes_usd,
			COALESCE((
				SELECT sum(ef.custo_usd)
				FROM ia_execucoes ef
				WHERE ef.fornecedor_id = f.id AND ef.criado_em >= $1 AND ef.criado_em < $2
			), 0)::text AS gasto_fornecedor_mes_usd
		FROM ia_modelos m
		JOIN ia_fornecedores f ON f.id = m.fornecedor_id`
	args := []any{inicioMes, proximoMes}
	if modeloID != "" {
		query += ` WHERE m.id::text = $3`
		args = append(args, modeloID)
	} else if fornecedorID != "" {
		query += ` WHERE m.fornecedor_id::text = $3 ORDER BY lower(m.identificador), m.criado_em`
		args = append(args, fornecedorID)
	}

	linhas, err := a.banco.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	type modeloBruto struct {
		modelo      Modelo
		gastoModStr string
		gastoSitStr string
		limiteEfStr string
		avisoEf     int
	}
	brutos, err := pgx.CollectRows(linhas, func(l pgx.CollectableRow) (modeloBruto, error) {
		var b modeloBruto
		var tipoForn, limFornStr, gastoModStr, gastoFornStr string
		var limModStr *string
		var avisoForn int
		var avisoMod *int
		if err := l.Scan(
			&b.modelo.ID, &b.modelo.FornecedorID, &b.modelo.FornecedorNome, &tipoForn,
			&b.modelo.Identificador, &b.modelo.Servico, &b.modelo.Origem,
			&limModStr, &avisoMod, &b.modelo.Ativo,
			&limFornStr, &avisoForn, &gastoModStr, &gastoFornStr,
		); err != nil {
			return b, err
		}
		b.modelo.Simulado = tipoForn == "simulado"
		b.gastoModStr = gastoModStr
		if limModStr != nil {
			v, ok := numeroDeTexto(*limModStr)
			if !ok {
				return b, errors.New("limite do modelo inválido")
			}
			b.modelo.LimiteMensalUsd = &v
			b.limiteEfStr = *limModStr
			b.gastoSitStr = gastoModStr
		} else {
			b.limiteEfStr = limFornStr
			b.gastoSitStr = gastoFornStr
		}
		b.modelo.AvisoPercentual = avisoMod
		b.avisoEf = avisoForn
		if avisoMod != nil {
			b.avisoEf = *avisoMod
		}
		return b, nil
	})
	if err != nil {
		return nil, err
	}

	modelos := make([]Modelo, 0, len(brutos))
	for _, b := range brutos {
		gastoMod, ok := numeroDeTexto(b.gastoModStr)
		if !ok {
			return nil, errors.New("gasto do modelo inválido")
		}
		situacao, _, _, err := calcularSituacaoLimite(b.gastoSitStr, b.limiteEfStr, b.avisoEf)
		if err != nil {
			return nil, err
		}
		precos, err := a.carregarPrecosVigentes(ctx, b.modelo.ID, hoje)
		if err != nil {
			return nil, err
		}
		b.modelo.GastoMesUsd = gastoMod
		b.modelo.Situacao = situacao
		b.modelo.PrecosVigentes = precos
		modelos = append(modelos, b.modelo)
	}
	return modelos, nil
}

func (a *API) carregarPrecosVigentes(ctx context.Context, modeloID, dataISO string) (PrecosVigentes, error) {
	var pv PrecosVigentes
	linhas, err := a.banco.Query(ctx, `
		SELECT DISTINCT ON (unidade) unidade, preco_usd::text
		FROM ia_precos
		WHERE modelo_id::text = $1 AND vigencia_inicio <= $2::date
		ORDER BY unidade, vigencia_inicio DESC, criado_em DESC`,
		modeloID, dataISO)
	if err != nil {
		return pv, err
	}
	defer linhas.Close()

	for linhas.Next() {
		var unidade, precoStr string
		if err := linhas.Scan(&unidade, &precoStr); err != nil {
			return pv, err
		}
		val, ok := numeroDeTexto(precoStr)
		if !ok {
			return pv, errors.New("preço inválido no banco")
		}
		switch unidade {
		case "token_entrada":
			pv.TokenEntrada = &val
		case "token_saida":
			pv.TokenSaida = &val
		case "imagem":
			pv.Imagem = &val
		}
	}
	return pv, linhas.Err()
}

// ---- Preços ----

func (a *API) listarPrecos(w http.ResponseWriter, r *http.Request, _ contextoSessao) {
	modeloID := r.PathValue("id")
	var existe bool
	if err := a.banco.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM ia_modelos WHERE id::text = $1)`, modeloID).Scan(&existe); err != nil {
		falhaInterna(w, err)
		return
	}
	if !existe {
		erro(w, 404, "nao_encontrado", "Modelo não encontrado.")
		return
	}

	linhas, err := a.banco.Query(r.Context(), `
		SELECT id::text, modelo_id::text, unidade, preco_usd::text, to_char(vigencia_inicio, 'YYYY-MM-DD'), criado_em
		FROM ia_precos
		WHERE modelo_id::text = $1
		ORDER BY vigencia_inicio DESC, criado_em DESC`, modeloID)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	precos, err := pgx.CollectRows(linhas, func(l pgx.CollectableRow) (Preco, error) {
		var p Preco
		var precoStr string
		if err := l.Scan(&p.ID, &p.ModeloID, &p.Unidade, &precoStr, &p.VigenciaInicio, &p.CriadoEm); err != nil {
			return p, err
		}
		val, ok := numeroDeTexto(precoStr)
		if !ok {
			return p, errors.New("preço inválido no banco")
		}
		p.PrecoUsd = val
		return p, nil
	})
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if precos == nil {
		precos = []Preco{}
	}
	responder(w, 200, RespostaPrecos{Precos: precos})
}

func (a *API) criarPreco(w http.ResponseWriter, r *http.Request, _ contextoSessao) {
	modeloID := r.PathValue("id")
	var servicoModelo string
	err := a.banco.QueryRow(r.Context(), `SELECT servico FROM ia_modelos WHERE id::text = $1`, modeloID).Scan(&servicoModelo)
	if errors.Is(err, pgx.ErrNoRows) {
		erro(w, 404, "nao_encontrado", "Modelo não encontrado.")
		return
	}
	if err != nil {
		falhaInterna(w, err)
		return
	}

	var p PedidoNovoPreco
	if !lerJSON(w, r, &p) {
		return
	}
	if !unidadesValidas[p.Unidade] {
		erro(w, 400, "dados_invalidos", "Unidade de preço inválida.")
		return
	}
	if servicoModelo == "texto" && p.Unidade == "imagem" {
		erro(w, 400, "formato_invalido", "Modelo de texto não aceita preço por imagem.")
		return
	}
	if servicoModelo == "imagem" && p.Unidade == "token_saida" {
		erro(w, 400, "formato_invalido", "Modelo de imagem aceita apenas preço por imagem ou token de entrada.")
		return
	}
	if p.PrecoUsd == nil {
		erro(w, 400, "dados_invalidos", "Informe o preço em US$.")
		return
	}
	precoSQL, ok := formatarPrecoUsd(*p.PrecoUsd)
	if !ok {
		erro(w, 400, "dados_invalidos", "Informe um preço válido maior ou igual a zero.")
		return
	}
	if !regexDataISO.MatchString(p.VigenciaInicio) {
		erro(w, 400, "dados_invalidos", "Use o formato AAAA-MM-DD na vigência.")
		return
	}
	if _, err := time.Parse("2006-01-02", p.VigenciaInicio); err != nil {
		erro(w, 400, "dados_invalidos", "Data de vigência inválida.")
		return
	}

	var criado Preco
	var precoStr string
	err = a.banco.QueryRow(r.Context(), `
		INSERT INTO ia_precos (modelo_id, unidade, preco_usd, vigencia_inicio)
		VALUES ($1, $2, $3, $4::date)
		RETURNING id::text, modelo_id::text, unidade, preco_usd::text, to_char(vigencia_inicio, 'YYYY-MM-DD'), criado_em`,
		modeloID, p.Unidade, precoSQL, p.VigenciaInicio).
		Scan(&criado.ID, &criado.ModeloID, &criado.Unidade, &precoStr, &criado.VigenciaInicio, &criado.CriadoEm)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			erro(w, 409, "preco_repetido", "Já existe preço cadastrado para esta unidade nesta data de vigência.")
			return
		}
		falhaInterna(w, err)
		return
	}
	val, ok := numeroDeTexto(precoStr)
	if !ok {
		falhaInterna(w, errors.New("preço gravado inválido"))
		return
	}
	criado.PrecoUsd = val
	responder(w, 201, criado)
}

func formatarPrecoUsd(v float64) (string, bool) {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 9999999999.99999999 {
		return "", false
	}
	r := new(big.Rat).SetFloat64(v)
	if r == nil {
		return "", false
	}
	return r.FloatString(8), true
}

func numeroDeTexto(s string) (float64, bool) {
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return 0, false
	}
	f, _ := r.Float64()
	return f, true
}
