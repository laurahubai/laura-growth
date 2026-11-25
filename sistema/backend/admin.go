package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	enderecosPadrao = map[string]string{
		"openai":    "https://api.openai.com/v1",
		"anthropic": "https://api.anthropic.com/v1",
	}
	tiposFornecedorValidos = map[string]bool{
		"openai":            true,
		"compativel_openai": true,
		"anthropic":         true,
		"simulado":          true,
	}
	usosPermitidosValidos = map[string]bool{
		"todas": true,
		"casa":  true,
	}
	fusoSaoPaulo = func() *time.Location {
		loc, err := time.LoadLocation("America/Sao_Paulo")
		if err != nil {
			return time.FixedZone("America/Sao_Paulo", -3*3600)
		}
		return loc
	}()
)

// ---- Formato das respostas (espelha packages/contratos/src/admin.ts) ----

type Fornecedor struct {
	ID              string    `json:"id"`
	Nome            string    `json:"nome"`
	Tipo            string    `json:"tipo"`
	Cadastro        string    `json:"cadastro"`
	Endereco        string    `json:"endereco"`
	ChaveFinal      *string   `json:"chaveFinal"`
	UsoPermitido    string    `json:"usoPermitido"`
	LimiteMensalUsd float64   `json:"limiteMensalUsd"`
	AvisoPercentual int       `json:"avisoPercentual"`
	Ativo           bool      `json:"ativo"`
	GastoMesUsd     float64   `json:"gastoMesUsd"`
	Situacao        string    `json:"situacao"`
	Simulado        bool      `json:"simulado"`
	CriadoEm        time.Time `json:"criadoEm"`
}

type RespostaFornecedores struct {
	Fornecedores []Fornecedor `json:"fornecedores"`
}

type PedidoNovoFornecedor struct {
	Nome            string  `json:"nome"`
	Tipo            string  `json:"tipo"`
	Endereco        string  `json:"endereco"`
	Chave           string  `json:"chave"`
	UsoPermitido    string  `json:"usoPermitido"`
	LimiteMensalUsd float64 `json:"limiteMensalUsd"`
	AvisoPercentual *int    `json:"avisoPercentual"`
}

type PedidoEditarFornecedor struct {
	Nome            *string  `json:"nome"`
	Endereco        *string  `json:"endereco"`
	Chave           *string  `json:"chave"`
	UsoPermitido    *string  `json:"usoPermitido"`
	LimiteMensalUsd *float64 `json:"limiteMensalUsd"`
	AvisoPercentual *int     `json:"avisoPercentual"`
	Ativo           *bool    `json:"ativo"`
}

func (a *API) rotasAdmin(m *http.ServeMux) {
	m.HandleFunc("GET /api/admin/fornecedores", a.comAdmin(a.listarFornecedores))
	m.HandleFunc("POST /api/admin/fornecedores", a.comAdmin(a.criarFornecedor))
	m.HandleFunc("PATCH /api/admin/fornecedores/{id}", a.comAdmin(a.editarFornecedor))
	m.HandleFunc("POST /api/admin/fornecedores/{id}/ler-modelos", a.comAdmin(a.lerModelosFornecedor))
	m.HandleFunc("GET /api/admin/fornecedores/{id}/modelos", a.comAdmin(a.listarModelos))
	m.HandleFunc("POST /api/admin/fornecedores/{id}/modelos", a.comAdmin(a.criarModelo))
	m.HandleFunc("PATCH /api/admin/modelos/{id}", a.comAdmin(a.editarModelo))
	m.HandleFunc("GET /api/admin/modelos/{id}/precos", a.comAdmin(a.listarPrecos))
	m.HandleFunc("POST /api/admin/modelos/{id}/precos", a.comAdmin(a.criarPreco))
}

// comAdmin exige sessão válida e confere admin_plataforma no banco a cada pedido (AC30, AC37).
func (a *API) comAdmin(h manipulador) http.HandlerFunc {
	return a.comSessao(func(w http.ResponseWriter, r *http.Request, s contextoSessao) {
		var ehAdmin bool
		err := a.banco.QueryRow(r.Context(), `SELECT admin_plataforma FROM usuarios WHERE id::text = $1`, s.usuario.ID).Scan(&ehAdmin)
		if err != nil {
			falhaInterna(w, err)
			return
		}
		if !ehAdmin {
			slog.Warn("acesso administrativo recusado", "usuarioId", s.usuario.ID, "metodo", r.Method, "rota", r.URL.Path)
			erro(w, 403, "sem_permissao", "Acesso restrito ao administrador da plataforma.")
			return
		}
		s.usuario.AdminPlataforma = true
		h(w, r, s)
	})
}

func (a *API) listarFornecedores(w http.ResponseWriter, r *http.Request, _ contextoSessao) {
	fornecedores, err := a.consultarFornecedores(r.Context(), "")
	if err != nil {
		falhaInterna(w, err)
		return
	}
	responder(w, 200, RespostaFornecedores{Fornecedores: fornecedores})
}

func (a *API) criarFornecedor(w http.ResponseWriter, r *http.Request, _ contextoSessao) {
	var p PedidoNovoFornecedor
	if !lerJSON(w, r, &p) {
		return
	}
	nome, errMsg := validarNomeFornecedor(p.Nome)
	if errMsg != "" {
		erro(w, 400, "dados_invalidos", errMsg)
		return
	}
	if !tiposFornecedorValidos[p.Tipo] {
		erro(w, 400, "dados_invalidos", "Tipo de fornecedor inválido.")
		return
	}
	uso := p.UsoPermitido
	if uso == "" {
		uso = "todas"
	}
	if !usosPermitidosValidos[uso] {
		erro(w, 400, "dados_invalidos", "Uso permitido inválido.")
		return
	}
	limiteStr, ok := formatarLimiteUsd(p.LimiteMensalUsd)
	if !ok {
		erro(w, 400, "dados_invalidos", "Informe um limite mensal maior que zero.")
		return
	}
	aviso := 80
	if p.AvisoPercentual != nil {
		aviso = *p.AvisoPercentual
	}
	if aviso < 1 || aviso > 100 {
		erro(w, 400, "dados_invalidos", "O percentual de aviso deve ficar entre 1 e 100.")
		return
	}

	endereco, err := a.resolverEnderecoFornecedor(r.Context(), p.Tipo, p.Endereco)
	if err != nil {
		erro(w, 400, "endereco_invalido", err.Error())
		return
	}

	id, err := novoUUID()
	if err != nil {
		falhaInterna(w, err)
		return
	}

	var chaveCifrada []byte
	var chaveFinal *string
	if p.Tipo != "simulado" {
		chave := strings.TrimSpace(p.Chave)
		if chave == "" {
			erro(w, 400, "chave_obrigatoria", "Informe a chave de API.")
			return
		}
		if utf8.RuneCountInString(chave) > 500 {
			erro(w, 400, "dados_invalidos", "A chave pode ter no máximo 500 caracteres.")
			return
		}
		if a.cifrador == nil {
			falhaInterna(w, errors.New("cifrador não configurado"))
			return
		}
		chaveCifrada, err = a.cifrador.cifrar(id, chave)
		if err != nil {
			falhaInterna(w, err)
			return
		}
		ultimos := ultimosQuatro(chave)
		chaveFinal = &ultimos
	}

	_, err = a.banco.Exec(r.Context(), `
		INSERT INTO ia_fornecedores
			(id, nome, tipo, cadastro, endereco, chave_cifrada, chave_final, limite_mensal_usd, aviso_percentual, uso_permitido, ativo)
		VALUES ($1, $2, $3, 'chave_api', $4, $5, $6, $7, $8, $9, true)`,
		id, nome, p.Tipo, endereco, chaveCifrada, chaveFinal, limiteStr, aviso, uso)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			erro(w, 409, "nome_repetido", "Já existe um fornecedor com este nome.")
			return
		}
		falhaInterna(w, err)
		return
	}

	f, err := a.carregarFornecedor(r.Context(), id)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	responder(w, 201, f)
}

func (a *API) editarFornecedor(w http.ResponseWriter, r *http.Request, _ contextoSessao) {
	id := r.PathValue("id")
	var p PedidoEditarFornecedor
	if !lerJSON(w, r, &p) {
		return
	}

	var nome, tipo, endereco, uso, limiteStr string
	var chaveCifrada []byte
	var chaveFinal *string
	var aviso int
	var ativo bool
	err := a.banco.QueryRow(r.Context(), `
		SELECT nome, tipo, endereco, chave_cifrada, chave_final, uso_permitido, limite_mensal_usd::text, aviso_percentual, ativo
		FROM ia_fornecedores WHERE id::text = $1`, id).
		Scan(&nome, &tipo, &endereco, &chaveCifrada, &chaveFinal, &uso, &limiteStr, &aviso, &ativo)
	if errors.Is(err, pgx.ErrNoRows) {
		erro(w, 404, "nao_encontrado", "Fornecedor não encontrado.")
		return
	}
	if err != nil {
		falhaInterna(w, err)
		return
	}

	if p.Nome != nil {
		var errMsg string
		nome, errMsg = validarNomeFornecedor(*p.Nome)
		if errMsg != "" {
			erro(w, 400, "dados_invalidos", errMsg)
			return
		}
	}
	if p.Endereco != nil {
		endereco, err = a.resolverEnderecoFornecedor(r.Context(), tipo, *p.Endereco)
		if err != nil {
			erro(w, 400, "endereco_invalido", err.Error())
			return
		}
	}
	if p.Chave != nil {
		if tipo == "simulado" {
			erro(w, 400, "dados_invalidos", "Fornecedor simulado não usa chave de API.")
			return
		}
		chave := strings.TrimSpace(*p.Chave)
		if chave == "" {
			erro(w, 400, "dados_invalidos", "Informe a chave de API.")
			return
		}
		if utf8.RuneCountInString(chave) > 500 {
			erro(w, 400, "dados_invalidos", "A chave pode ter no máximo 500 caracteres.")
			return
		}
		if a.cifrador == nil {
			falhaInterna(w, errors.New("cifrador não configurado"))
			return
		}
		chaveCifrada, err = a.cifrador.cifrar(id, chave)
		if err != nil {
			falhaInterna(w, err)
			return
		}
		ultimos := ultimosQuatro(chave)
		chaveFinal = &ultimos
	}
	if p.UsoPermitido != nil {
		if !usosPermitidosValidos[*p.UsoPermitido] {
			erro(w, 400, "dados_invalidos", "Uso permitido inválido.")
			return
		}
		uso = *p.UsoPermitido
	}
	if p.LimiteMensalUsd != nil {
		var ok bool
		limiteStr, ok = formatarLimiteUsd(*p.LimiteMensalUsd)
		if !ok {
			erro(w, 400, "dados_invalidos", "Informe um limite mensal maior que zero.")
			return
		}
	}
	if p.AvisoPercentual != nil {
		if *p.AvisoPercentual < 1 || *p.AvisoPercentual > 100 {
			erro(w, 400, "dados_invalidos", "O percentual de aviso deve ficar entre 1 e 100.")
			return
		}
		aviso = *p.AvisoPercentual
	}
	if p.Ativo != nil {
		ativo = *p.Ativo
	}

	_, err = a.banco.Exec(r.Context(), `
		UPDATE ia_fornecedores
		SET nome = $2, endereco = $3, chave_cifrada = $4, chave_final = $5,
		    uso_permitido = $6, limite_mensal_usd = $7, aviso_percentual = $8, ativo = $9, atualizado_em = now()
		WHERE id::text = $1`,
		id, nome, endereco, chaveCifrada, chaveFinal, uso, limiteStr, aviso, ativo)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			erro(w, 409, "nome_repetido", "Já existe um fornecedor com este nome.")
			return
		}
		falhaInterna(w, err)
		return
	}

	f, err := a.carregarFornecedor(r.Context(), id)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	responder(w, 200, f)
}

func (a *API) carregarFornecedor(ctx context.Context, id string) (Fornecedor, error) {
	lista, err := a.consultarFornecedores(ctx, id)
	if err != nil {
		return Fornecedor{}, err
	}
	if len(lista) == 0 {
		return Fornecedor{}, pgx.ErrNoRows
	}
	return lista[0], nil
}

func (a *API) consultarFornecedores(ctx context.Context, filtroID string) ([]Fornecedor, error) {
	inicioMes := inicioMesSaoPaulo(a.agora())
	proximoMes := inicioMes.AddDate(0, 1, 0)

	query := `
		SELECT
			f.id::text, f.nome, f.tipo, f.cadastro, f.endereco, f.chave_final,
			f.uso_permitido, f.limite_mensal_usd::text, f.aviso_percentual, f.ativo, f.criado_em,
			COALESCE((
				SELECT sum(e.custo_usd)
				FROM ia_execucoes e
				WHERE e.fornecedor_id = f.id AND e.criado_em >= $1 AND e.criado_em < $2
			), 0)::text AS gasto_mes_usd
		FROM ia_fornecedores f`
	args := []any{inicioMes, proximoMes}
	if filtroID != "" {
		query += ` WHERE f.id::text = $3`
		args = append(args, filtroID)
	} else {
		query += ` ORDER BY lower(f.nome), f.criado_em`
	}

	linhas, err := a.banco.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	lista, err := pgx.CollectRows(linhas, func(l pgx.CollectableRow) (Fornecedor, error) {
		var f Fornecedor
		var limiteStr, gastoStr string
		if err := l.Scan(
			&f.ID, &f.Nome, &f.Tipo, &f.Cadastro, &f.Endereco, &f.ChaveFinal,
			&f.UsoPermitido, &limiteStr, &f.AvisoPercentual, &f.Ativo, &f.CriadoEm, &gastoStr,
		); err != nil {
			return f, err
		}
		situacao, gasto, limite, err := calcularSituacaoLimite(gastoStr, limiteStr, f.AvisoPercentual)
		if err != nil {
			return f, err
		}
		f.Situacao = situacao
		f.GastoMesUsd = gasto
		f.LimiteMensalUsd = limite
		f.Simulado = f.Tipo == "simulado"
		return f, nil
	})
	if err != nil {
		return nil, err
	}
	if lista == nil {
		lista = []Fornecedor{}
	}
	return lista, nil
}

func (a *API) resolverEnderecoFornecedor(ctx context.Context, tipo, bruto string) (string, error) {
	bruto = strings.TrimSpace(bruto)
	if tipo == "simulado" {
		if bruto != "" {
			return "", errors.New("Fornecedor simulado não usa endereço.")
		}
		return "", nil
	}
	if bruto == "" {
		bruto = enderecosPadrao[tipo]
		if bruto == "" {
			return "", errors.New("Informe o endereço do fornecedor compatível com OpenAI.")
		}
	}
	return validarEnderecoCom(ctx, bruto, a.resolverIP, a.permitirRedeInterna)
}

func validarNomeFornecedor(bruto string) (string, string) {
	nome := strings.TrimSpace(bruto)
	n := utf8.RuneCountInString(nome)
	if n == 0 {
		return "", "Informe o nome."
	}
	if n > 100 {
		return "", "O nome pode ter no máximo 100 caracteres."
	}
	return nome, ""
}

func formatarLimiteUsd(v float64) (string, bool) {
	if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 || v > 9999999999.99 {
		return "", false
	}
	r := new(big.Rat).SetFloat64(v)
	if r == nil {
		return "", false
	}
	s := r.FloatString(2)
	arredondado, ok := new(big.Rat).SetString(s)
	if !ok || arredondado.Sign() <= 0 {
		return "", false
	}
	return s, true
}

// inicioMesSaoPaulo devolve o início do mês corrente no fuso America/Sao_Paulo.
func inicioMesSaoPaulo(agora time.Time) time.Time {
	local := agora.In(fusoSaoPaulo)
	return time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, fusoSaoPaulo)
}

// calcularSituacaoLimite compara gasto e limite usando aritmética decimal exata (math/big.Rat).
// Reutilizada por fornecedores (S1) e modelos/executor (S2/S3).
func calcularSituacaoLimite(gastoStr, limiteStr string, avisoPercentual int) (situacao string, gasto, limite float64, err error) {
	gastoRat, ok := new(big.Rat).SetString(gastoStr)
	if !ok {
		return "", 0, 0, fmt.Errorf("valor de gasto inválido: %q", gastoStr)
	}
	limiteRat, ok := new(big.Rat).SetString(limiteStr)
	if !ok || limiteRat.Sign() <= 0 {
		return "", 0, 0, fmt.Errorf("valor de limite inválido: %q", limiteStr)
	}
	gasto, _ = gastoRat.Float64()
	limite, _ = limiteRat.Float64()

	if gastoRat.Cmp(limiteRat) >= 0 {
		return "limite", gasto, limite, nil
	}
	limiarAviso := new(big.Rat).Mul(limiteRat, big.NewRat(int64(avisoPercentual), 100))
	if gastoRat.Cmp(limiarAviso) >= 0 {
		return "aviso", gasto, limite, nil
	}
	return "normal", gasto, limite, nil
}

func ultimosQuatro(chave string) string {
	r := []rune(chave)
	if len(r) <= 4 {
		return chave
	}
	return string(r[len(r)-4:])
}

func (a *API) clienteHTTPFornecedor(timeout time.Duration) *http.Client {
	if !a.permitirRedeInterna {
		return novoClienteSeguro(timeout)
	}
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if a.tlsTeste != nil {
		tlsCfg = a.tlsTeste.Clone()
		if tlsCfg.MinVersion < tls.VersionTLS12 {
			tlsCfg.MinVersion = tls.VersionTLS12
		}
	}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig:     tlsCfg,
			TLSHandshakeTimeout: timeout,
			ForceAttemptHTTP2:   true,
		},
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errRedirecionamentoRecusado
		},
	}
}

func novoUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
