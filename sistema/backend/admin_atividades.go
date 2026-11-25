package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	promptTesteTexto  = "Oi, qual modelo é você?"
	promptTesteImagem = "um círculo azul sobre fundo branco"
)

type definicaoAtividade struct {
	Codigo  string
	Nome    string
	Servico string
}

var definicoesAtividades = []definicaoAtividade{
	{Codigo: "chat_cmo", Nome: "Chat do CMO", Servico: "texto"},
	{Codigo: "plano", Nome: "Plano de marketing", Servico: "texto"},
	{Codigo: "pauta", Nome: "Pauta", Servico: "texto"},
	{Codigo: "texto", Nome: "Texto", Servico: "texto"},
	{Codigo: "imagem", Nome: "Imagem", Servico: "imagem"},
	{Codigo: "revisao", Nome: "Revisão (Guardian)", Servico: "texto"},
}

var mapaAtividades = func() map[string]definicaoAtividade {
	m := make(map[string]definicaoAtividade, len(definicoesAtividades))
	for _, d := range definicoesAtividades {
		m[d.Codigo] = d
	}
	return m
}()

// ---- Formato das respostas (espelha packages/contratos/src/admin.ts) ----

type ModeloResumo struct {
	ID             string `json:"id"`
	Identificador  string `json:"identificador"`
	FornecedorNome string `json:"fornecedorNome"`
	Simulado       bool   `json:"simulado"`
	Ativo          bool   `json:"ativo"`
}

type ResultadoTeste struct {
	ExecucaoID    string       `json:"execucaoId"`
	Servico       string       `json:"servico"`
	Modelo        ModeloResumo `json:"modelo"`
	Texto         *string      `json:"texto"`
	ImagemDataUrl *string      `json:"imagemDataUrl"`
	TokensEntrada int          `json:"tokensEntrada"`
	TokensSaida   int          `json:"tokensSaida"`
	Imagens       int          `json:"imagens"`
	CustoUsd      float64      `json:"custoUsd"`
	Simulado      bool         `json:"simulado"`
}

type Execucao struct {
	ID                  string    `json:"id"`
	CorrelacaoID        string    `json:"correlacaoId"`
	CriadoEm            time.Time `json:"criadoEm"`
	Atividade           string    `json:"atividade"`
	OrganizacaoNome     *string   `json:"organizacaoNome"`
	FornecedorNome      string    `json:"fornecedorNome"`
	ModeloIdentificador string    `json:"modeloIdentificador"`
	Tentativa           int       `json:"tentativa"`
	AtendidoPelaReserva bool      `json:"atendidoPelaReserva"`
	Status              string    `json:"status"`
	Erro                *string   `json:"erro"`
	TokensEntrada       int       `json:"tokensEntrada"`
	TokensSaida         int       `json:"tokensSaida"`
	Imagens             int       `json:"imagens"`
	CustoUsd            float64   `json:"custoUsd"`
	CustoEstimado       bool      `json:"custoEstimado"`
	Simulado            bool      `json:"simulado"`
}

type RespostaExecucoes struct {
	Execucoes []Execucao `json:"execucoes"`
}

type RotaAtividade struct {
	Atividade string        `json:"atividade"`
	Nome      string        `json:"nome"`
	Servico   string        `json:"servico"`
	Principal *ModeloResumo `json:"principal"`
	Reserva   *ModeloResumo `json:"reserva"`
	Pronta    bool          `json:"pronta"`
	Falta     []string      `json:"falta"`
}

type RespostaAtividades struct {
	Atividades []RotaAtividade `json:"atividades"`
}

type PedidoRotaAtividade struct {
	ModeloPrincipalID *string `json:"modeloPrincipalId"`
	ModeloReservaID   *string `json:"modeloReservaId"`
}

type Tentativa struct {
	Modelo ModeloResumo `json:"modelo"`
	Status string       `json:"status"`
	Erro   *string      `json:"erro"`
}

type ResultadoAtividade struct {
	CorrelacaoID        string       `json:"correlacaoId"`
	Atividade           string       `json:"atividade"`
	AtendidoPor         ModeloResumo `json:"atendidoPor"`
	AtendidoPelaReserva bool         `json:"atendidoPelaReserva"`
	Tentativas          []Tentativa  `json:"tentativas"`
	Texto               *string      `json:"texto"`
	ImagemDataUrl       *string      `json:"imagemDataUrl"`
	TokensEntrada       int          `json:"tokensEntrada"`
	TokensSaida         int          `json:"tokensSaida"`
	Imagens             int          `json:"imagens"`
	CustoUsd            float64      `json:"custoUsd"`
	Simulado            bool         `json:"simulado"`
}

type AvisoLimite struct {
	Alvo            string  `json:"alvo"`
	ID              string  `json:"id"`
	Nome            string  `json:"nome"`
	Situacao        string  `json:"situacao"`
	GastoMesUsd     float64 `json:"gastoMesUsd"`
	LimiteMensalUsd float64 `json:"limiteMensalUsd"`
	AvisoPercentual int     `json:"avisoPercentual"`
}

type RespostaAvisos struct {
	Avisos []AvisoLimite `json:"avisos"`
}

// ---- Estruturas internas do executor ----

type candidatoExecucao struct {
	modeloID        string
	identificador   string
	servico         string
	modeloAtivo     bool
	limiteModeloStr *string
	avisoModelo     *int
	fornecedorID    string
	fornecedorNome  string
	fornecedorTipo  string
	endereco        string
	chaveCifrada    []byte
	usoPermitido    string
	limiteFornStr   string
	avisoForn       int
	fornecedorAtivo bool
}

func (c candidatoExecucao) resumo() ModeloResumo {
	return ModeloResumo{
		ID:             c.modeloID,
		Identificador:  c.identificador,
		FornecedorNome: c.fornecedorNome,
		Simulado:       c.fornecedorTipo == "simulado",
		Ativo:          c.modeloAtivo,
	}
}

type precosVigentesTexto struct {
	entrada *string
	saida   *string
	imagem  *string
}

func (p precosVigentesTexto) completoPara(servico string) bool {
	if servico == "imagem" {
		return p.imagem != nil
	}
	return p.entrada != nil && p.saida != nil
}

// ---- Rotas de Atividades, Testar, Execuções e Avisos ----

func (a *API) listarAtividades(w http.ResponseWriter, r *http.Request, _ contextoSessao) {
	atividades := make([]RotaAtividade, 0, len(definicoesAtividades))
	for _, def := range definicoesAtividades {
		rota, err := a.carregarRotaAtividade(r.Context(), def)
		if err != nil {
			falhaInterna(w, err)
			return
		}
		atividades = append(atividades, rota)
	}
	responder(w, 200, RespostaAtividades{Atividades: atividades})
}

func (a *API) salvarRotaAtividade(w http.ResponseWriter, r *http.Request, _ contextoSessao) {
	def, ok := mapaAtividades[r.PathValue("atividade")]
	if !ok {
		erro(w, 404, "nao_encontrado", "Atividade não encontrada.")
		return
	}
	var p PedidoRotaAtividade
	if !lerJSON(w, r, &p) {
		return
	}
	if p.ModeloPrincipalID != nil && p.ModeloReservaID != nil && *p.ModeloPrincipalID == *p.ModeloReservaID {
		erro(w, 400, "modelos_iguais", "Escolha modelos diferentes para principal e reserva.")
		return
	}
	for _, modID := range []*string{p.ModeloPrincipalID, p.ModeloReservaID} {
		if modID == nil {
			continue
		}
		var servico string
		err := a.banco.QueryRow(r.Context(), `SELECT servico FROM ia_modelos WHERE id::text = $1`, *modID).Scan(&servico)
		if errors.Is(err, pgx.ErrNoRows) {
			erro(w, 404, "nao_encontrado", "Modelo não encontrado.")
			return
		}
		if err != nil {
			falhaInterna(w, err)
			return
		}
		if servico != def.Servico {
			erro(w, 400, "servico_incompativel", fmt.Sprintf("Este modelo é de %s e a atividade exige %s.", servico, def.Servico))
			return
		}
	}

	if _, err := a.banco.Exec(r.Context(), `
		INSERT INTO ia_rotas (atividade, modelo_principal_id, modelo_reserva_id, atualizado_em)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (atividade) DO UPDATE SET
			modelo_principal_id = EXCLUDED.modelo_principal_id,
			modelo_reserva_id = EXCLUDED.modelo_reserva_id,
			atualizado_em = now()`,
		def.Codigo, p.ModeloPrincipalID, p.ModeloReservaID); err != nil {
		falhaInterna(w, err)
		return
	}
	rota, err := a.carregarRotaAtividade(r.Context(), def)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	responder(w, 200, rota)
}

func (a *API) testarModelo(w http.ResponseWriter, r *http.Request, s contextoSessao) {
	modeloID := r.PathValue("id")
	cand, err := a.buscarCandidato(r.Context(), modeloID)
	if errors.Is(err, pgx.ErrNoRows) {
		erro(w, 404, "nao_encontrado", "Modelo não encontrado.")
		return
	}
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if !cand.fornecedorAtivo {
		erro(w, 409, "fornecedor_inativo", "Este fornecedor está desativado. Ative antes de testar.")
		return
	}
	if !cand.modeloAtivo {
		erro(w, 409, "modelo_inativo", "Este modelo está desativado. Ative antes de testar.")
		return
	}

	hoje := a.agora().In(fusoSaoPaulo).Format("2006-01-02")
	precos, err := a.carregarPrecosVigentesTexto(r.Context(), cand.modeloID, hoje)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if !precos.completoPara(cand.servico) {
		erro(w, 409, "sem_preco", "Cadastre o preço vigente deste modelo antes de testar.")
		return
	}

	noLimite, err := a.candidatoNoLimite(r.Context(), cand)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if noLimite {
		erro(w, 409, "limite_atingido", "O limite mensal deste modelo ou fornecedor foi atingido.")
		return
	}

	prompt := promptTesteTexto
	if cand.servico == "imagem" {
		prompt = promptTesteImagem
	}
	correlacaoID, err := novoUUID()
	if err != nil {
		falhaInterna(w, err)
		return
	}

	execucaoID, texto, imagemURL, tokensIn, tokensOut, imgs, custoUsd, err := a.executarEGerar(
		r.Context(), cand, precos, correlacaoID, "teste", s.orgAtiva, s.usuario.ID, 1, false, prompt,
	)
	if err != nil {
		var ef *erroFornecedor
		if errors.As(err, &ef) {
			erro(w, 502, ef.codigo, ef.mensagem)
			return
		}
		var ee *erroEnderecoExecucao
		if errors.As(err, &ee) {
			erro(w, 400, "endereco_invalido", ee.mensagem)
			return
		}
		falhaInterna(w, err)
		return
	}

	responder(w, 200, ResultadoTeste{
		ExecucaoID:    execucaoID,
		Servico:       cand.servico,
		Modelo:        cand.resumo(),
		Texto:         texto,
		ImagemDataUrl: imagemURL,
		TokensEntrada: tokensIn,
		TokensSaida:   tokensOut,
		Imagens:       imgs,
		CustoUsd:      custoUsd,
		Simulado:      cand.fornecedorTipo == "simulado",
	})
}

func (a *API) testarAtividade(w http.ResponseWriter, r *http.Request, s contextoSessao) {
	def, ok := mapaAtividades[r.PathValue("atividade")]
	if !ok {
		erro(w, 404, "nao_encontrado", "Atividade não encontrada.")
		return
	}
	prompt := promptTesteTexto
	if def.Servico == "imagem" {
		prompt = promptTesteImagem
	}
	res, statusHTTP, codErro, msgErro := a.executarAtividade(r.Context(), def.Codigo, s.orgAtiva, s.usuario.ID, prompt)
	if statusHTTP != 0 {
		erro(w, statusHTTP, codErro, msgErro)
		return
	}
	responder(w, 200, res)
}

func (a *API) listarExecucoes(w http.ResponseWriter, r *http.Request, _ contextoSessao) {
	limite := 50
	if bruto := r.URL.Query().Get("limite"); bruto != "" {
		v, err := strconv.Atoi(bruto)
		if err != nil || v <= 0 {
			erro(w, 400, "dados_invalidos", "Informe um limite válido.")
			return
		}
		if v > 200 {
			v = 200
		}
		limite = v
	}

	linhas, err := a.banco.Query(r.Context(), `
		SELECT
			e.id::text, e.correlacao_id::text, e.criado_em, e.atividade,
			o.nome, e.fornecedor_nome, e.modelo_identificador,
			e.tentativa, e.atendido_pela_reserva, e.status, e.erro,
			e.tokens_entrada, e.tokens_saida, e.imagens,
			e.custo_usd::text, e.custo_estimado, e.simulado
		FROM ia_execucoes e
		LEFT JOIN organizacoes o ON o.id = e.organizacao_id
		ORDER BY e.criado_em DESC, e.tentativa DESC
		LIMIT $1`, limite)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	lista, err := pgx.CollectRows(linhas, func(l pgx.CollectableRow) (Execucao, error) {
		var ex Execucao
		var custoStr string
		if err := l.Scan(
			&ex.ID, &ex.CorrelacaoID, &ex.CriadoEm, &ex.Atividade,
			&ex.OrganizacaoNome, &ex.FornecedorNome, &ex.ModeloIdentificador,
			&ex.Tentativa, &ex.AtendidoPelaReserva, &ex.Status, &ex.Erro,
			&ex.TokensEntrada, &ex.TokensSaida, &ex.Imagens,
			&custoStr, &ex.CustoEstimado, &ex.Simulado,
		); err != nil {
			return ex, err
		}
		v, ok := numeroDeTexto(custoStr)
		if !ok {
			return ex, errors.New("custo inválido no banco")
		}
		ex.CustoUsd = v
		return ex, nil
	})
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if lista == nil {
		lista = []Execucao{}
	}
	responder(w, 200, RespostaExecucoes{Execucoes: lista})
}

func (a *API) listarAvisos(w http.ResponseWriter, r *http.Request, _ contextoSessao) {
	avisos := []AvisoLimite{}

	fornecedores, err := a.consultarFornecedores(r.Context(), "")
	if err != nil {
		falhaInterna(w, err)
		return
	}
	for _, f := range fornecedores {
		if !f.Ativo || (f.Situacao != "aviso" && f.Situacao != "limite") {
			continue
		}
		avisos = append(avisos, AvisoLimite{
			Alvo:            "fornecedor",
			ID:              f.ID,
			Nome:            f.Nome,
			Situacao:        f.Situacao,
			GastoMesUsd:     f.GastoMesUsd,
			LimiteMensalUsd: f.LimiteMensalUsd,
			AvisoPercentual: f.AvisoPercentual,
		})
	}

	modelos, err := a.consultarModelos(r.Context(), "", "")
	if err != nil {
		falhaInterna(w, err)
		return
	}
	for _, m := range modelos {
		if !m.Ativo || m.LimiteMensalUsd == nil || (m.Situacao != "aviso" && m.Situacao != "limite") {
			continue
		}
		avisoPct := 80
		if m.AvisoPercentual != nil {
			avisoPct = *m.AvisoPercentual
		} else {
			for _, f := range fornecedores {
				if f.ID == m.FornecedorID {
					avisoPct = f.AvisoPercentual
					break
				}
			}
		}
		avisos = append(avisos, AvisoLimite{
			Alvo:            "modelo",
			ID:              m.ID,
			Nome:            m.Identificador,
			Situacao:        m.Situacao,
			GastoMesUsd:     m.GastoMesUsd,
			LimiteMensalUsd: *m.LimiteMensalUsd,
			AvisoPercentual: avisoPct,
		})
	}

	responder(w, 200, RespostaAvisos{Avisos: avisos})
}

// ---- Executor de atividades ----

// executarAtividade executa uma atividade com principal e reserva, gravando uma
// linha por tentativa em ia_execucoes. Reutilizada pelo endpoint de teste e pelas etapas seguintes.
func (a *API) executarAtividade(ctx context.Context, atividade, orgID, usuarioID, prompt string) (ResultadoAtividade, int, string, string) {
	def, ok := mapaAtividades[atividade]
	if !ok {
		return ResultadoAtividade{}, 404, "nao_encontrado", "Atividade não encontrada."
	}
	rota, err := a.carregarRotaAtividade(ctx, def)
	if err != nil {
		slog.Error("falha ao carregar rota", "atividade", atividade, "erro", err)
		return ResultadoAtividade{}, 500, "falha_interna", "Algo deu errado. Tente de novo."
	}
	if !rota.Pronta {
		return ResultadoAtividade{}, 409, "atividade_incompleta", "Atividade incompleta. Falta: " + strings.Join(rota.Falta, ", ") + "."
	}

	orgDaCasa := true
	if orgID != "" {
		err := a.banco.QueryRow(ctx, `SELECT da_casa FROM organizacoes WHERE id::text = $1`, orgID).Scan(&orgDaCasa)
		if errors.Is(err, pgx.ErrNoRows) {
			orgDaCasa = false
		} else if err != nil {
			slog.Error("falha ao verificar organização da casa", "erro", err)
			return ResultadoAtividade{}, 500, "falha_interna", "Algo deu errado. Tente de novo."
		}
	}

	correlacaoID, err := novoUUID()
	if err != nil {
		slog.Error("falha ao gerar correlação", "erro", err)
		return ResultadoAtividade{}, 500, "falha_interna", "Algo deu errado. Tente de novo."
	}

	hoje := a.agora().In(fusoSaoPaulo).Format("2006-01-02")
	tentativas := make([]Tentativa, 0, 2)
	todosPuladoLimite := true

	pares := []struct {
		id      string
		reserva bool
		numero  int
	}{
		{id: rota.Principal.ID, reserva: false, numero: 1},
		{id: rota.Reserva.ID, reserva: true, numero: 2},
	}

	for _, par := range pares {
		cand, err := a.buscarCandidato(ctx, par.id)
		if err != nil {
			slog.Error("falha ao buscar modelo da atividade", "modeloId", par.id, "erro", err)
			return ResultadoAtividade{}, 500, "falha_interna", "Algo deu errado. Tente de novo."
		}
		precos, err := a.carregarPrecosVigentesTexto(ctx, cand.modeloID, hoje)
		if err != nil {
			slog.Error("falha ao buscar preços do modelo", "modeloId", par.id, "erro", err)
			return ResultadoAtividade{}, 500, "falha_interna", "Algo deu errado. Tente de novo."
		}

		// Verificação de uso restrito à casa (AC30).
		if cand.usoPermitido == "casa" && orgID != "" && !orgDaCasa {
			msg := "Fornecedor restrito às organizações da casa."
			if err := a.registrarTentativaSemExecucao(ctx, cand, precos, correlacaoID, def.Codigo, orgID, usuarioID, par.numero, par.reserva, "falha", msg); err != nil {
				slog.Error("falha ao registrar tentativa", "erro", err)
				return ResultadoAtividade{}, 500, "falha_interna", "Algo deu errado. Tente de novo."
			}
			todosPuladoLimite = false
			tentativas = append(tentativas, Tentativa{Modelo: cand.resumo(), Status: "falha", Erro: &msg})
			continue
		}

		// Verificação de modelo/fornecedor ativo ou sem preço.
		if !cand.fornecedorAtivo || !cand.modeloAtivo {
			msg := "Modelo ou fornecedor desativado."
			if err := a.registrarTentativaSemExecucao(ctx, cand, precos, correlacaoID, def.Codigo, orgID, usuarioID, par.numero, par.reserva, "falha", msg); err != nil {
				slog.Error("falha ao registrar tentativa", "erro", err)
				return ResultadoAtividade{}, 500, "falha_interna", "Algo deu errado. Tente de novo."
			}
			todosPuladoLimite = false
			tentativas = append(tentativas, Tentativa{Modelo: cand.resumo(), Status: "falha", Erro: &msg})
			continue
		}
		if !precos.completoPara(cand.servico) {
			msg := "Modelo sem preço vigente."
			if err := a.registrarTentativaSemExecucao(ctx, cand, precos, correlacaoID, def.Codigo, orgID, usuarioID, par.numero, par.reserva, "falha", msg); err != nil {
				slog.Error("falha ao registrar tentativa", "erro", err)
				return ResultadoAtividade{}, 500, "falha_interna", "Algo deu errado. Tente de novo."
			}
			todosPuladoLimite = false
			tentativas = append(tentativas, Tentativa{Modelo: cand.resumo(), Status: "falha", Erro: &msg})
			continue
		}

		// Verificação de limite mensal (AC34).
		noLimite, err := a.candidatoNoLimite(ctx, cand)
		if err != nil {
			slog.Error("falha ao verificar limite", "erro", err)
			return ResultadoAtividade{}, 500, "falha_interna", "Algo deu errado. Tente de novo."
		}
		if noLimite {
			msg := "Limite mensal atingido."
			if err := a.registrarTentativaSemExecucao(ctx, cand, precos, correlacaoID, def.Codigo, orgID, usuarioID, par.numero, par.reserva, "pulado_limite", msg); err != nil {
				slog.Error("falha ao registrar pulo por limite", "erro", err)
				return ResultadoAtividade{}, 500, "falha_interna", "Algo deu errado. Tente de novo."
			}
			tentativas = append(tentativas, Tentativa{Modelo: cand.resumo(), Status: "pulado_limite", Erro: &msg})
			continue
		}

		todosPuladoLimite = false
		_, texto, imagemURL, tokensIn, tokensOut, imgs, custoUsd, err := a.executarEGerar(
			ctx, cand, precos, correlacaoID, def.Codigo, orgID, usuarioID, par.numero, par.reserva, prompt,
		)
		if err != nil {
			msg := err.Error()
			tentativas = append(tentativas, Tentativa{Modelo: cand.resumo(), Status: "falha", Erro: &msg})
			continue
		}
		tentativas = append(tentativas, Tentativa{Modelo: cand.resumo(), Status: "ok"})
		return ResultadoAtividade{
			CorrelacaoID:        correlacaoID,
			Atividade:           def.Codigo,
			AtendidoPor:         cand.resumo(),
			AtendidoPelaReserva: par.reserva,
			Tentativas:          tentativas,
			Texto:               texto,
			ImagemDataUrl:       imagemURL,
			TokensEntrada:       tokensIn,
			TokensSaida:         tokensOut,
			Imagens:             imgs,
			CustoUsd:            custoUsd,
			Simulado:            cand.fornecedorTipo == "simulado",
		}, 0, "", ""
	}

	if todosPuladoLimite {
		return ResultadoAtividade{}, 409, "limite_atingido", "Limite mensal atingido no modelo principal e no modelo reserva."
	}
	return ResultadoAtividade{}, 502, "atividade_falhou", "Não foi possível concluir a atividade pelos modelos configurados."
}

type erroEnderecoExecucao struct {
	mensagem string
}

func (e *erroEnderecoExecucao) Error() string { return e.mensagem }

func (a *API) executarEGerar(
	ctx context.Context,
	cand candidatoExecucao,
	precos precosVigentesTexto,
	correlacaoID, atividade, orgID, usuarioID string,
	tentativa int,
	atendidoPelaReserva bool,
	prompt string,
) (execucaoID string, texto, imagemURL *string, tokensIn, tokensOut, imgs int, custoUsd float64, errRet error) {
	var chave string
	endereco := cand.endereco
	if cand.fornecedorTipo != "simulado" {
		enderecoValidado, err := a.resolverEnderecoFornecedor(ctx, cand.fornecedorTipo, endereco)
		if err != nil {
			_ = a.registrarTentativaSemExecucao(ctx, cand, precos, correlacaoID, atividade, orgID, usuarioID, tentativa, atendidoPelaReserva, "falha", err.Error())
			return "", nil, nil, 0, 0, 0, 0, &erroEnderecoExecucao{mensagem: err.Error()}
		}
		endereco = enderecoValidado
		if a.cifrador == nil {
			return "", nil, nil, 0, 0, 0, 0, errors.New("cifrador não configurado")
		}
		chave, err = a.cifrador.decifrar(cand.fornecedorID, cand.chaveCifrada)
		if err != nil {
			return "", nil, nil, 0, 0, 0, 0, err
		}
	}

	con, err := novoConector(cand.fornecedorTipo, endereco, chave, a.clienteHTTPFornecedor(timeoutFornecedor))
	if err != nil {
		_ = a.registrarTentativaSemExecucao(ctx, cand, precos, correlacaoID, atividade, orgID, usuarioID, tentativa, atendidoPelaReserva, "falha", err.Error())
		return "", nil, nil, 0, 0, 0, 0, err
	}

	if cand.servico == "imagem" {
		resp, err := con.gerarImagem(ctx, cand.identificador, prompt)
		if err != nil {
			_ = a.registrarTentativaSemExecucao(ctx, cand, precos, correlacaoID, atividade, orgID, usuarioID, tentativa, atendidoPelaReserva, "falha", err.Error())
			return "", nil, nil, 0, 0, 0, 0, err
		}
		imgs = resp.Imagens
		tokensIn = resp.TokensEntrada
		tokensOut = resp.TokensSaida
		imagemURL = &resp.ImagemDataUrl
	} else {
		resp, err := con.gerarTexto(ctx, cand.identificador, prompt)
		if err != nil {
			_ = a.registrarTentativaSemExecucao(ctx, cand, precos, correlacaoID, atividade, orgID, usuarioID, tentativa, atendidoPelaReserva, "falha", err.Error())
			return "", nil, nil, 0, 0, 0, 0, err
		}
		tokensIn = resp.TokensEntrada
		tokensOut = resp.TokensSaida
		texto = &resp.Texto
	}

	custoSQL, custoFloat, err := calcularCustoExato(tokensIn, tokensOut, imgs, precos)
	if err != nil {
		return "", nil, nil, 0, 0, 0, 0, err
	}

	execucaoID, err = a.inserirExecucao(ctx, cand, precos, correlacaoID, atividade, orgID, usuarioID, tentativa, atendidoPelaReserva, "ok", nil, tokensIn, tokensOut, imgs, custoSQL)
	if err != nil {
		return "", nil, nil, 0, 0, 0, 0, err
	}
	return execucaoID, texto, imagemURL, tokensIn, tokensOut, imgs, custoFloat, nil
}

func (a *API) registrarTentativaSemExecucao(
	ctx context.Context,
	cand candidatoExecucao,
	precos precosVigentesTexto,
	correlacaoID, atividade, orgID, usuarioID string,
	tentativa int,
	atendidoPelaReserva bool,
	status, msgErro string,
) error {
	_, err := a.inserirExecucao(ctx, cand, precos, correlacaoID, atividade, orgID, usuarioID, tentativa, atendidoPelaReserva, status, &msgErro, 0, 0, 0, "0.00000000")
	return err
}

func (a *API) inserirExecucao(
	ctx context.Context,
	cand candidatoExecucao,
	precos precosVigentesTexto,
	correlacaoID, atividade, orgID, usuarioID string,
	tentativa int,
	atendidoPelaReserva bool,
	status string,
	msgErro *string,
	tokensIn, tokensOut, imgs int,
	custoSQL string,
) (string, error) {
	var orgParam, userParam *string
	if orgID != "" {
		orgParam = &orgID
	}
	if usuarioID != "" {
		userParam = &usuarioID
	}
	var id string
	err := a.banco.QueryRow(ctx, `
		INSERT INTO ia_execucoes (
			correlacao_id, atividade, organizacao_id, usuario_id,
			fornecedor_id, modelo_id, fornecedor_nome, modelo_identificador,
			tentativa, atendido_pela_reserva, status, erro,
			tokens_entrada, tokens_saida, imagens,
			preco_token_entrada, preco_token_saida, preco_imagem,
			custo_usd, custo_estimado, simulado
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, false, $20)
		RETURNING id::text`,
		correlacaoID, atividade, orgParam, userParam,
		cand.fornecedorID, cand.modeloID, cand.fornecedorNome, cand.identificador,
		tentativa, atendidoPelaReserva, status, msgErro,
		tokensIn, tokensOut, imgs,
		precos.entrada, precos.saida, precos.imagem,
		custoSQL, cand.fornecedorTipo == "simulado",
	).Scan(&id)
	return id, err
}

func (a *API) carregarRotaAtividade(ctx context.Context, def definicaoAtividade) (RotaAtividade, error) {
	rota := RotaAtividade{
		Atividade: def.Codigo,
		Nome:      def.Nome,
		Servico:   def.Servico,
		Falta:     []string{},
	}
	var principalID, reservaID *string
	err := a.banco.QueryRow(ctx, `
		SELECT modelo_principal_id::text, modelo_reserva_id::text
		FROM ia_rotas WHERE atividade = $1`, def.Codigo).
		Scan(&principalID, &reservaID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return rota, err
	}

	hoje := a.agora().In(fusoSaoPaulo).Format("2006-01-02")

	var errFalta error
	rota.Principal, errFalta = a.avaliarModeloRota(ctx, principalID, "principal", def.Servico, hoje, &rota.Falta)
	if errFalta != nil {
		return rota, errFalta
	}
	rota.Reserva, errFalta = a.avaliarModeloRota(ctx, reservaID, "reserva", def.Servico, hoje, &rota.Falta)
	if errFalta != nil {
		return rota, errFalta
	}
	rota.Pronta = len(rota.Falta) == 0
	return rota, nil
}

func (a *API) avaliarModeloRota(ctx context.Context, id *string, papel, servico, hoje string, falta *[]string) (*ModeloResumo, error) {
	if id == nil {
		*falta = append(*falta, "modelo "+papel)
		return nil, nil
	}
	cand, err := a.buscarCandidato(ctx, *id)
	if errors.Is(err, pgx.ErrNoRows) {
		*falta = append(*falta, "modelo "+papel)
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	resumo := cand.resumo()
	if !cand.fornecedorAtivo {
		*falta = append(*falta, "fornecedor do modelo "+papel+" inativo")
	} else if !cand.modeloAtivo {
		*falta = append(*falta, "modelo "+papel+" inativo")
	}
	precos, err := a.carregarPrecosVigentesTexto(ctx, cand.modeloID, hoje)
	if err != nil {
		return nil, err
	}
	if !precos.completoPara(servico) {
		*falta = append(*falta, "preço do modelo "+papel)
	}
	return &resumo, nil
}

func (a *API) buscarCandidato(ctx context.Context, modeloID string) (candidatoExecucao, error) {
	var c candidatoExecucao
	err := a.banco.QueryRow(ctx, `
		SELECT
			m.id::text, m.identificador, m.servico, m.ativo, m.limite_mensal_usd::text, m.aviso_percentual,
			f.id::text, f.nome, f.tipo, f.endereco, f.chave_cifrada, f.uso_permitido, f.limite_mensal_usd::text, f.aviso_percentual, f.ativo
		FROM ia_modelos m
		JOIN ia_fornecedores f ON f.id = m.fornecedor_id
		WHERE m.id::text = $1`, modeloID).
		Scan(
			&c.modeloID, &c.identificador, &c.servico, &c.modeloAtivo, &c.limiteModeloStr, &c.avisoModelo,
			&c.fornecedorID, &c.fornecedorNome, &c.fornecedorTipo, &c.endereco, &c.chaveCifrada,
			&c.usoPermitido, &c.limiteFornStr, &c.avisoForn, &c.fornecedorAtivo,
		)
	return c, err
}

func (a *API) carregarPrecosVigentesTexto(ctx context.Context, modeloID, dataISO string) (precosVigentesTexto, error) {
	var p precosVigentesTexto
	linhas, err := a.banco.Query(ctx, `
		SELECT DISTINCT ON (unidade) unidade, preco_usd::text
		FROM ia_precos
		WHERE modelo_id::text = $1 AND vigencia_inicio <= $2::date
		ORDER BY unidade, vigencia_inicio DESC, criado_em DESC`, modeloID, dataISO)
	if err != nil {
		return p, err
	}
	defer linhas.Close()
	for linhas.Next() {
		var unidade, val string
		if err := linhas.Scan(&unidade, &val); err != nil {
			return p, err
		}
		copia := val
		switch unidade {
		case "token_entrada":
			p.entrada = &copia
		case "token_saida":
			p.saida = &copia
		case "imagem":
			p.imagem = &copia
		}
	}
	return p, linhas.Err()
}

// candidatoNoLimite confere o limite mensal no fuso America/Sao_Paulo.
// Quando o modelo tem limite próprio, ele vale no lugar do limite do fornecedor (AC34).
func (a *API) candidatoNoLimite(ctx context.Context, cand candidatoExecucao) (bool, error) {
	inicioMes := inicioMesSaoPaulo(a.agora())
	proximoMes := inicioMes.AddDate(0, 1, 0)

	var gastoStr, limiteStr string
	aviso := cand.avisoForn
	if cand.avisoModelo != nil {
		aviso = *cand.avisoModelo
	}

	if cand.limiteModeloStr != nil {
		limiteStr = *cand.limiteModeloStr
		err := a.banco.QueryRow(ctx, `
			SELECT COALESCE(sum(custo_usd), 0)::text
			FROM ia_execucoes
			WHERE modelo_id::text = $1 AND criado_em >= $2 AND criado_em < $3`,
			cand.modeloID, inicioMes, proximoMes).Scan(&gastoStr)
		if err != nil {
			return false, err
		}
	} else {
		limiteStr = cand.limiteFornStr
		err := a.banco.QueryRow(ctx, `
			SELECT COALESCE(sum(custo_usd), 0)::text
			FROM ia_execucoes
			WHERE fornecedor_id::text = $1 AND criado_em >= $2 AND criado_em < $3`,
			cand.fornecedorID, inicioMes, proximoMes).Scan(&gastoStr)
		if err != nil {
			return false, err
		}
	}

	situacao, _, _, err := calcularSituacaoLimite(gastoStr, limiteStr, aviso)
	if err != nil {
		return false, err
	}
	return situacao == "limite", nil
}

// calcularCustoExato calcula o custo em US$ usando aritmética decimal exata (math/big.Rat).
// Tokens: US$ por 1 milhão de tokens; imagem: US$ por imagem.
func calcularCustoExato(tokensIn, tokensOut, imgs int, precos precosVigentesTexto) (string, float64, error) {
	total := new(big.Rat)
	umMilhao := big.NewRat(1_000_000, 1)

	if tokensIn > 0 && precos.entrada != nil {
		p, ok := new(big.Rat).SetString(*precos.entrada)
		if !ok {
			return "", 0, errors.New("preço de entrada inválido")
		}
		parc := new(big.Rat).Mul(p, big.NewRat(int64(tokensIn), 1))
		parc.Quo(parc, umMilhao)
		total.Add(total, parc)
	}
	if tokensOut > 0 && precos.saida != nil {
		p, ok := new(big.Rat).SetString(*precos.saida)
		if !ok {
			return "", 0, errors.New("preço de saída inválido")
		}
		parc := new(big.Rat).Mul(p, big.NewRat(int64(tokensOut), 1))
		parc.Quo(parc, umMilhao)
		total.Add(total, parc)
	}
	if imgs > 0 && precos.imagem != nil {
		p, ok := new(big.Rat).SetString(*precos.imagem)
		if !ok {
			return "", 0, errors.New("preço de imagem inválido")
		}
		parc := new(big.Rat).Mul(p, big.NewRat(int64(imgs), 1))
		total.Add(total, parc)
	}

	s := total.FloatString(8)
	arredondado, _ := new(big.Rat).SetString(s)
	f, _ := arredondado.Float64()
	return s, f, nil
}
