package main

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	promptTesteTexto        = "Oi, qual modelo é você?"
	promptTesteImagem       = "um círculo azul sobre fundo branco"
	timeoutImagem           = 120 * time.Second
	timeoutGravacaoExecucao = 5 * time.Second
)

func contextoGravacaoExecucao(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), timeoutGravacaoExecucao)
}

func confirmarTxExecucao(ctx context.Context, tx pgx.Tx) error {
	ctxCommit, cancelar := contextoGravacaoExecucao(ctx)
	defer cancelar()
	return tx.Commit(ctxCommit)
}

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

	prompt := promptTesteTexto
	if cand.servico == "imagem" {
		prompt = promptTesteImagem
	}
	correlacaoID, err := novoUUID()
	if err != nil {
		falhaInterna(w, err)
		return
	}

	res, err := a.executarComTrava(
		r.Context(), cand, precos, correlacaoID, "teste", "", s.usuario.ID, 1, false, prompt, false,
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
		var ei *erroInternoExecucao
		if errors.As(err, &ei) {
			erro(w, 500, "falha_interna", "Algo deu errado. Tente de novo.")
			return
		}
		falhaInterna(w, err)
		return
	}
	if res.puladoLimite {
		erro(w, 409, "limite_atingido", "O limite mensal deste modelo ou fornecedor foi atingido.")
		return
	}

	responder(w, 200, ResultadoTeste{
		ExecucaoID:    res.execucaoID,
		Servico:       cand.servico,
		Modelo:        cand.resumo(),
		Texto:         res.texto,
		ImagemDataUrl: res.imagemURL,
		TokensEntrada: res.tokensIn,
		TokensSaida:   res.tokensOut,
		Imagens:       res.imgs,
		CustoUsd:      res.custoUsd,
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
	res, statusHTTP, codErro, msgErro := a.executarAtividadePlataforma(r.Context(), def.Codigo, s.usuario.ID, prompt)
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

// executarAtividade executa uma atividade com principal e reserva para uma organização,
// gravando uma linha por tentativa em ia_execucoes. Chamada com orgID vazio sempre nega fornecedores "casa".
func (a *API) executarAtividade(ctx context.Context, atividade, orgID, usuarioID, prompt string) (ResultadoAtividade, int, string, string) {
	return a.executarAtividadeInterna(ctx, atividade, orgID, usuarioID, prompt, false)
}

// executarAtividadePlataforma executa o teste administrativo de atividade da plataforma
// (sempre com organizacao_id = NULL e tratado como da casa; uso exclusivo de testarAtividade).
func (a *API) executarAtividadePlataforma(ctx context.Context, atividade, usuarioID, prompt string) (ResultadoAtividade, int, string, string) {
	return a.executarAtividadeInterna(ctx, atividade, "", usuarioID, prompt, true)
}

func (a *API) executarAtividadeInterna(ctx context.Context, atividade, orgID, usuarioID, prompt string, plataformaCasa bool) (ResultadoAtividade, int, string, string) {
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

	orgDaCasa := false
	if plataformaCasa && orgID == "" {
		orgDaCasa = true
	} else if orgID != "" {
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

	type parExecucao struct {
		id      string
		reserva bool
		numero  int
	}
	pares := []parExecucao{
		{id: rota.Principal.ID, reserva: false, numero: 1},
	}
	if rota.Reserva != nil {
		pares = append(pares, parExecucao{id: rota.Reserva.ID, reserva: true, numero: 2})
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
		if cand.usoPermitido == "casa" && !orgDaCasa {
			msg := "Fornecedor restrito às organizações da casa."
			if err := a.registrarTentativaSemExecucaoEm(ctx, a.banco, cand, precos, correlacaoID, def.Codigo, orgID, usuarioID, par.numero, par.reserva, "falha", msg); err != nil {
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
			if err := a.registrarTentativaSemExecucaoEm(ctx, a.banco, cand, precos, correlacaoID, def.Codigo, orgID, usuarioID, par.numero, par.reserva, "falha", msg); err != nil {
				slog.Error("falha ao registrar tentativa", "erro", err)
				return ResultadoAtividade{}, 500, "falha_interna", "Algo deu errado. Tente de novo."
			}
			todosPuladoLimite = false
			tentativas = append(tentativas, Tentativa{Modelo: cand.resumo(), Status: "falha", Erro: &msg})
			continue
		}
		if !precos.completoPara(cand.servico) {
			msg := "Modelo sem preço vigente."
			if err := a.registrarTentativaSemExecucaoEm(ctx, a.banco, cand, precos, correlacaoID, def.Codigo, orgID, usuarioID, par.numero, par.reserva, "falha", msg); err != nil {
				slog.Error("falha ao registrar tentativa", "erro", err)
				return ResultadoAtividade{}, 500, "falha_interna", "Algo deu errado. Tente de novo."
			}
			todosPuladoLimite = false
			tentativas = append(tentativas, Tentativa{Modelo: cand.resumo(), Status: "falha", Erro: &msg})
			continue
		}

		res, err := a.executarComTrava(
			ctx, cand, precos, correlacaoID, def.Codigo, orgID, usuarioID, par.numero, par.reserva, prompt, true,
		)
		if err != nil {
			var ef *erroFornecedor
			var ee *erroEnderecoExecucao
			if !errors.As(err, &ef) && !errors.As(err, &ee) {
				// Falha interna ou ao gravar no banco não cai na reserva.
				return ResultadoAtividade{}, 500, "falha_interna", "Algo deu errado. Tente de novo."
			}
			todosPuladoLimite = false
			msg := err.Error()
			tentativas = append(tentativas, Tentativa{Modelo: cand.resumo(), Status: "falha", Erro: &msg})
			continue
		}
		if res.puladoLimite {
			msg := "Limite mensal atingido."
			tentativas = append(tentativas, Tentativa{Modelo: cand.resumo(), Status: "pulado_limite", Erro: &msg})
			continue
		}

		tentativas = append(tentativas, Tentativa{Modelo: cand.resumo(), Status: "ok"})
		return ResultadoAtividade{
			CorrelacaoID:        correlacaoID,
			Atividade:           def.Codigo,
			AtendidoPor:         cand.resumo(),
			AtendidoPelaReserva: par.reserva,
			Tentativas:          tentativas,
			Texto:               res.texto,
			ImagemDataUrl:       res.imagemURL,
			TokensEntrada:       res.tokensIn,
			TokensSaida:         res.tokensOut,
			Imagens:             res.imgs,
			CustoUsd:            res.custoUsd,
			Simulado:            cand.fornecedorTipo == "simulado",
		}, 0, "", ""
	}

	if todosPuladoLimite {
		msgLimite := "Limite mensal atingido no modelo principal."
		if rota.Reserva != nil {
			msgLimite = "Limite mensal atingido no modelo principal e no modelo reserva."
		}
		return ResultadoAtividade{}, 409, "limite_atingido", msgLimite
	}
	return ResultadoAtividade{}, 502, "atividade_falhou", resumirErroTentativas(tentativas)
}

func resumirErroTentativas(tentativas []Tentativa) string {
	if len(tentativas) == 1 && tentativas[0].Erro != nil {
		return *tentativas[0].Erro
	}
	partes := make([]string, 0, len(tentativas))
	for i, t := range tentativas {
		if t.Erro == nil {
			continue
		}
		papel := "Principal"
		if i > 0 {
			papel = "Reserva"
		}
		partes = append(partes, fmt.Sprintf("%s (%s): %s", papel, t.Modelo.Identificador, *t.Erro))
	}
	if len(partes) == 0 {
		return "Não foi possível concluir a atividade pelos modelos configurados."
	}
	return strings.Join(partes, " ")
}

type executorSQL interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type erroEnderecoExecucao struct {
	mensagem string
}

func (e *erroEnderecoExecucao) Error() string { return e.mensagem }

type erroInternoExecucao struct {
	err error
}

func (e *erroInternoExecucao) Error() string { return "Algo deu errado. Tente de novo." }
func (e *erroInternoExecucao) Unwrap() error { return e.err }

type resultadoExecucao struct {
	execucaoID   string
	texto        *string
	imagemURL    *string
	tokensIn     int
	tokensOut    int
	imgs         int
	custoUsd     float64
	puladoLimite bool
}

func chaveTravaFornecedor(fornecedorID string) (int32, int32) {
	h := fnv.New64a()
	_, _ = h.Write([]byte("laura-growth:ia_limite:" + fornecedorID))
	soma := h.Sum64()
	return int32(soma >> 32), int32(soma)
}

func travarFornecedorTx(ctx context.Context, tx pgx.Tx, fornecedorID string) error {
	k1, k2 := chaveTravaFornecedor(fornecedorID)
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1, $2)`, k1, k2)
	return err
}

// executarComTrava serializa a conferência de limite mensal e a gravação da execução por fornecedor
// usando pg_advisory_xact_lock na transação.
// Nota: a trava consultiva e a conexão do banco ficam presas durante a chamada HTTP ao fornecedor
// (uma conexão em uso por chamada ativa por fornecedor); revisitar com reserva de saldo antes de abrir a clientes.
func (a *API) executarComTrava(
	ctx context.Context,
	cand candidatoExecucao,
	precos precosVigentesTexto,
	correlacaoID, atividade, orgID, usuarioID string,
	tentativa int,
	atendidoPelaReserva bool,
	prompt string,
	registrarPuloLimite bool,
) (resultadoExecucao, error) {
	tx, err := a.banco.Begin(ctx)
	if err != nil {
		slog.Error("falha ao iniciar transação de execução", "atividade", atividade, "modeloId", cand.modeloID, "erro", err)
		return resultadoExecucao{}, &erroInternoExecucao{err: err}
	}
	defer tx.Rollback(context.Background())

	if err := travarFornecedorTx(ctx, tx, cand.fornecedorID); err != nil {
		slog.Error("falha ao travar limite do fornecedor", "fornecedorId", cand.fornecedorID, "erro", err)
		return resultadoExecucao{}, &erroInternoExecucao{err: err}
	}

	noLimite, err := a.candidatoNoLimiteEm(ctx, tx, cand)
	if err != nil {
		slog.Error("falha ao verificar limite", "fornecedorId", cand.fornecedorID, "modeloId", cand.modeloID, "erro", err)
		return resultadoExecucao{}, &erroInternoExecucao{err: err}
	}
	if noLimite {
		if registrarPuloLimite {
			if err := a.registrarTentativaSemExecucaoEm(
				ctx, tx, cand, precos, correlacaoID, atividade, orgID, usuarioID, tentativa, atendidoPelaReserva, "pulado_limite", "Limite mensal atingido.",
			); err != nil {
				slog.Error("falha ao registrar pulo por limite", "atividade", atividade, "modeloId", cand.modeloID, "erro", err)
				return resultadoExecucao{}, &erroInternoExecucao{err: err}
			}
			if errCommit := confirmarTxExecucao(ctx, tx); errCommit != nil {
				slog.Error("falha ao confirmar pulo por limite", "atividade", atividade, "modeloId", cand.modeloID, "erro", errCommit)
				return resultadoExecucao{}, &erroInternoExecucao{err: errCommit}
			}
		}
		return resultadoExecucao{puladoLimite: true}, nil
	}

	res, errExec := a.executarEGerarEm(
		ctx, tx, cand, precos, correlacaoID, atividade, orgID, usuarioID, tentativa, atendidoPelaReserva, prompt,
	)
	if errExec != nil {
		var ei *erroInternoExecucao
		if errors.As(errExec, &ei) {
			return resultadoExecucao{}, errExec
		}
		if errCommit := confirmarTxExecucao(ctx, tx); errCommit != nil {
			slog.Error("falha ao confirmar registro de tentativa", "atividade", atividade, "modeloId", cand.modeloID, "erro", errCommit)
			return resultadoExecucao{}, &erroInternoExecucao{err: errCommit}
		}
		return resultadoExecucao{}, errExec
	}

	if errCommit := confirmarTxExecucao(ctx, tx); errCommit != nil {
		slog.Error("falha ao confirmar registro de execução", "atividade", atividade, "modeloId", cand.modeloID, "erro", errCommit)
		return resultadoExecucao{}, &erroInternoExecucao{err: errCommit}
	}
	return res, nil
}

func (a *API) executarEGerarEm(
	ctx context.Context,
	exec executorSQL,
	cand candidatoExecucao,
	precos precosVigentesTexto,
	correlacaoID, atividade, orgID, usuarioID string,
	tentativa int,
	atendidoPelaReserva bool,
	prompt string,
) (resultadoExecucao, error) {
	var chave string
	endereco := cand.endereco
	if cand.fornecedorTipo != "simulado" {
		enderecoValidado, err := a.resolverEnderecoFornecedor(ctx, cand.fornecedorTipo, endereco)
		if err != nil {
			msg := err.Error()
			if errReg := a.registrarTentativaSemExecucaoEm(ctx, exec, cand, precos, correlacaoID, atividade, orgID, usuarioID, tentativa, atendidoPelaReserva, "falha", msg); errReg != nil {
				slog.Error("falha ao registrar tentativa com endereço inválido", "fornecedorId", cand.fornecedorID, "erro", errReg)
				return resultadoExecucao{}, &erroInternoExecucao{err: errReg}
			}
			return resultadoExecucao{}, &erroEnderecoExecucao{mensagem: msg}
		}
		endereco = enderecoValidado
		if a.cifrador == nil {
			errCfg := errors.New("cifrador não configurado")
			slog.Error("cifrador não configurado ao executar modelo", "fornecedorId", cand.fornecedorID)
			return resultadoExecucao{}, &erroInternoExecucao{err: errCfg}
		}
		chave, err = a.cifrador.decifrar(cand.fornecedorID, cand.chaveCifrada)
		if err != nil {
			slog.Error("falha ao decifrar chave do fornecedor", "fornecedorId", cand.fornecedorID)
			msg := "Não foi possível ler a chave salva; cadastre-a de novo."
			if errReg := a.registrarTentativaSemExecucaoEm(ctx, exec, cand, precos, correlacaoID, atividade, orgID, usuarioID, tentativa, atendidoPelaReserva, "falha", msg); errReg != nil {
				slog.Error("falha ao registrar tentativa com chave ilegível", "fornecedorId", cand.fornecedorID, "erro", errReg)
				return resultadoExecucao{}, &erroInternoExecucao{err: errReg}
			}
			return resultadoExecucao{}, &erroFornecedor{codigo: "fornecedor_erro", mensagem: msg}
		}
	}

	timeout := timeoutFornecedor
	if cand.servico == "imagem" {
		timeout = timeoutImagem
	}
	con, err := novoConector(cand.fornecedorTipo, endereco, chave, a.clienteHTTPFornecedor(timeout))
	if err != nil {
		var ef *erroFornecedor
		if !errors.As(err, &ef) {
			slog.Error("falha interna ao criar conector", "fornecedorId", cand.fornecedorID, "erro", err)
			return resultadoExecucao{}, &erroInternoExecucao{err: err}
		}
		if errReg := a.registrarTentativaSemExecucaoEm(ctx, exec, cand, precos, correlacaoID, atividade, orgID, usuarioID, tentativa, atendidoPelaReserva, "falha", ef.mensagem); errReg != nil {
			slog.Error("falha ao registrar tentativa do conector", "fornecedorId", cand.fornecedorID, "erro", errReg)
			return resultadoExecucao{}, &erroInternoExecucao{err: errReg}
		}
		return resultadoExecucao{}, ef
	}

	var res resultadoExecucao
	if cand.servico == "imagem" {
		resp, err := con.gerarImagem(ctx, cand.identificador, prompt)
		if err != nil {
			var ef *erroFornecedor
			if !errors.As(err, &ef) {
				slog.Error("falha interna ao gerar imagem no fornecedor", "fornecedorId", cand.fornecedorID, "modeloId", cand.modeloID, "erro", err)
				return resultadoExecucao{}, &erroInternoExecucao{err: err}
			}
			if errReg := a.registrarTentativaSemExecucaoEm(ctx, exec, cand, precos, correlacaoID, atividade, orgID, usuarioID, tentativa, atendidoPelaReserva, "falha", ef.mensagem); errReg != nil {
				slog.Error("falha ao registrar tentativa de imagem", "fornecedorId", cand.fornecedorID, "modeloId", cand.modeloID, "erro", errReg)
				return resultadoExecucao{}, &erroInternoExecucao{err: errReg}
			}
			return resultadoExecucao{}, ef
		}
		res.imgs = resp.Imagens
		res.tokensIn = resp.TokensEntrada
		res.tokensOut = resp.TokensSaida
		res.imagemURL = &resp.ImagemDataUrl
	} else {
		resp, err := con.gerarTexto(ctx, cand.identificador, prompt)
		if err != nil {
			var ef *erroFornecedor
			if !errors.As(err, &ef) {
				slog.Error("falha interna ao gerar texto no fornecedor", "fornecedorId", cand.fornecedorID, "modeloId", cand.modeloID, "erro", err)
				return resultadoExecucao{}, &erroInternoExecucao{err: err}
			}
			if errReg := a.registrarTentativaSemExecucaoEm(ctx, exec, cand, precos, correlacaoID, atividade, orgID, usuarioID, tentativa, atendidoPelaReserva, "falha", ef.mensagem); errReg != nil {
				slog.Error("falha ao registrar tentativa de texto", "fornecedorId", cand.fornecedorID, "modeloId", cand.modeloID, "erro", errReg)
				return resultadoExecucao{}, &erroInternoExecucao{err: errReg}
			}
			return resultadoExecucao{}, ef
		}
		res.tokensIn = resp.TokensEntrada
		res.tokensOut = resp.TokensSaida
		res.texto = &resp.Texto
	}

	custoSQL, custoFloat, err := calcularCustoExato(res.tokensIn, res.tokensOut, res.imgs, precos)
	if err != nil {
		slog.Error("falha ao calcular custo da execução", "modeloId", cand.modeloID, "erro", err)
		return resultadoExecucao{}, &erroInternoExecucao{err: err}
	}

	execucaoID, err := a.inserirExecucaoEm(ctx, exec, cand, precos, correlacaoID, atividade, orgID, usuarioID, tentativa, atendidoPelaReserva, "ok", nil, res.tokensIn, res.tokensOut, res.imgs, custoSQL)
	if err != nil {
		slog.Error("falha ao gravar execução de IA", "atividade", atividade, "modeloId", cand.modeloID, "erro", err)
		return resultadoExecucao{}, &erroInternoExecucao{err: err}
	}
	res.execucaoID = execucaoID
	res.custoUsd = custoFloat
	return res, nil
}

func (a *API) registrarTentativaSemExecucaoEm(
	ctx context.Context,
	exec executorSQL,
	cand candidatoExecucao,
	precos precosVigentesTexto,
	correlacaoID, atividade, orgID, usuarioID string,
	tentativa int,
	atendidoPelaReserva bool,
	status, msgErro string,
) error {
	_, err := a.inserirExecucaoEm(ctx, exec, cand, precos, correlacaoID, atividade, orgID, usuarioID, tentativa, atendidoPelaReserva, status, &msgErro, 0, 0, 0, "0.00000000")
	return err
}

func (a *API) inserirExecucaoEm(
	ctx context.Context,
	exec executorSQL,
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
	// Mantém a gravação da execução e do custo mesmo se o cliente desconectar após resposta paga do fornecedor.
	ctxGravacao, cancelar := contextoGravacaoExecucao(ctx)
	defer cancelar()
	agora := a.agora()
	var id string
	err := exec.QueryRow(ctxGravacao, `
		INSERT INTO ia_execucoes (
			correlacao_id, atividade, organizacao_id, usuario_id,
			fornecedor_id, modelo_id, fornecedor_nome, modelo_identificador,
			tentativa, atendido_pela_reserva, status, erro,
			tokens_entrada, tokens_saida, imagens,
			preco_token_entrada, preco_token_saida, preco_imagem,
			custo_usd, custo_estimado, simulado, criado_em
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, false, $20, $21)
		RETURNING id::text`,
		correlacaoID, atividade, orgParam, userParam,
		cand.fornecedorID, cand.modeloID, cand.fornecedorNome, cand.identificador,
		tentativa, atendidoPelaReserva, status, msgErro,
		tokensIn, tokensOut, imgs,
		precos.entrada, precos.saida, precos.imagem,
		custoSQL, cand.fornecedorTipo == "simulado", agora,
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
		if papel == "principal" {
			*falta = append(*falta, "modelo "+papel)
		}
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

// candidatoNoLimiteEm confere o limite mensal no fuso America/Sao_Paulo dentro da consulta/transação informada.
// Quando o modelo tem limite próprio, ele vale no lugar do limite do fornecedor (AC34).
func (a *API) candidatoNoLimiteEm(ctx context.Context, exec executorSQL, cand candidatoExecucao) (bool, error) {
	inicioMes := inicioMesSaoPaulo(a.agora())
	proximoMes := inicioMes.AddDate(0, 1, 0)

	var gastoStr, limiteStr string
	aviso := cand.avisoForn
	if cand.avisoModelo != nil {
		aviso = *cand.avisoModelo
	}

	if cand.limiteModeloStr != nil {
		limiteStr = *cand.limiteModeloStr
		err := exec.QueryRow(ctx, `
			SELECT COALESCE(sum(custo_usd), 0)::text
			FROM ia_execucoes
			WHERE modelo_id::text = $1 AND criado_em >= $2 AND criado_em < $3`,
			cand.modeloID, inicioMes, proximoMes).Scan(&gastoStr)
		if err != nil {
			return false, err
		}
	} else {
		limiteStr = cand.limiteFornStr
		err := exec.QueryRow(ctx, `
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
