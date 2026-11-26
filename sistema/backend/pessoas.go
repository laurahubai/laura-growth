package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

// ---- Formato das respostas (espelha packages/contratos/src/acessos.ts) ----

type PessoaMembro struct {
	UsuarioID string    `json:"usuarioId"`
	Nome      string    `json:"nome"`
	Email     string    `json:"email"`
	Papel     string    `json:"papel"`
	Alcadas   []string  `json:"alcadas"`
	CriadoEm  time.Time `json:"criadoEm"`
}

type ConvitePendente struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	Alcadas       []string  `json:"alcadas"`
	EnvioEstado   string    `json:"envioEstado"`
	CriadoPorNome *string   `json:"criadoPorNome"`
	CriadoEm      time.Time `json:"criadoEm"`
	ExpiraEm      time.Time `json:"expiraEm"`
}

type RespostaPessoas struct {
	Membros   []PessoaMembro    `json:"membros"`
	Convites  []ConvitePendente `json:"convites"`
	Conjuntos []ConjuntoAlcadas `json:"conjuntos"`
}

type RegistroAcesso struct {
	ID            string         `json:"id"`
	Acao          string         `json:"acao"`
	AtorID        *string        `json:"atorId"`
	AtorNome      *string        `json:"atorNome"`
	AtorEmail     *string        `json:"atorEmail"`
	AlvoUsuarioID *string        `json:"alvoUsuarioId"`
	AlvoNome      *string        `json:"alvoNome"`
	AlvoEmail     *string        `json:"alvoEmail"`
	Detalhe       map[string]any `json:"detalhe"`
	Em            time.Time      `json:"em"`
}

type RespostaRegistroAcessos struct {
	Registros []RegistroAcesso `json:"registros"`
}

type escopoGestaoPessoas struct {
	orgID              string
	viaAdminPlataforma bool
}

func detalheAcesso(escopo escopoGestaoPessoas, campos map[string]any) map[string]any {
	out := make(map[string]any, len(campos)+1)
	for k, v := range campos {
		out[k] = v
	}
	if escopo.viaAdminPlataforma {
		out["via_admin_plataforma"] = true
	}
	return out
}

func (a *API) resolverEscopoPessoasOrg(w http.ResponseWriter, r *http.Request, s contextoSessao) (escopoGestaoPessoas, bool) {
	org, ok := a.autorizarAlcada(w, r, s, "gerenciar_pessoas")
	if !ok {
		return escopoGestaoPessoas{}, false
	}
	return escopoGestaoPessoas{orgID: org}, true
}

func (a *API) comEscopoPessoasOrg(h func(http.ResponseWriter, *http.Request, contextoSessao, escopoGestaoPessoas)) manipulador {
	return func(w http.ResponseWriter, r *http.Request, s contextoSessao) {
		escopo, ok := a.resolverEscopoPessoasOrg(w, r, s)
		if !ok {
			return
		}
		h(w, r, s, escopo)
	}
}

func (a *API) rotasPessoas(m *http.ServeMux) {
	m.HandleFunc("GET /api/pessoas", a.comSessao(a.comEscopoPessoasOrg(a.listarPessoasCom)))
	m.HandleFunc("GET /api/pessoas/registro", a.comSessao(a.comEscopoPessoasOrg(a.listarRegistroAcessosCom)))
	m.HandleFunc("PUT /api/pessoas/{usuarioId}/alcadas", a.comSessao(a.comEscopoPessoasOrg(a.salvarAlcadasPessoaCom)))
	m.HandleFunc("DELETE /api/pessoas/{usuarioId}", a.comSessao(a.comEscopoPessoasOrg(a.removerPessoaCom)))
}

func (a *API) listarPessoasCom(w http.ResponseWriter, r *http.Request, _ contextoSessao, escopo escopoGestaoPessoas) {
	org := escopo.orgID

	linhasMembros, err := a.banco.Query(r.Context(), `
		SELECT m.usuario_id::text, u.nome, u.email, m.papel, m.criado_em,
		       COALESCE(array_agg(am.alcada) FILTER (WHERE am.alcada IS NOT NULL), '{}')
		FROM membros m
		JOIN usuarios u ON u.id = m.usuario_id
		LEFT JOIN alcadas_membro am
		       ON am.organizacao_id = m.organizacao_id AND am.usuario_id = m.usuario_id
		WHERE m.organizacao_id = $1::uuid
		GROUP BY m.usuario_id, u.nome, u.email, m.papel, m.criado_em
		ORDER BY (m.papel = 'dono') DESC, lower(u.nome), lower(u.email)`, org)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	membros, err := pgx.CollectRows(linhasMembros, func(l pgx.CollectableRow) (PessoaMembro, error) {
		var p PessoaMembro
		var alcadas []string
		if err := l.Scan(&p.UsuarioID, &p.Nome, &p.Email, &p.Papel, &p.CriadoEm, &alcadas); err != nil {
			return p, err
		}
		if p.Papel == "dono" {
			p.Alcadas = make([]string, len(todasAlcadas))
			copy(p.Alcadas, todasAlcadas)
		} else {
			p.Alcadas = ordenarAlcadas(alcadas)
		}
		return p, nil
	})
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if membros == nil {
		membros = []PessoaMembro{}
	}

	linhasConvites, err := a.banco.Query(r.Context(), `
		SELECT c.id::text, c.email, c.alcadas, c.envio_estado, u.nome, c.criado_em, c.expira_em
		FROM convites c
		LEFT JOIN usuarios u ON u.id = c.criado_por
		WHERE c.organizacao_id = $1::uuid
		  AND c.aceito_em IS NULL
		  AND c.revogado_em IS NULL
		  AND c.expira_em > $2
		ORDER BY c.criado_em DESC`, org, a.agora())
	if err != nil {
		falhaInterna(w, err)
		return
	}
	convites, err := pgx.CollectRows(linhasConvites, func(l pgx.CollectableRow) (ConvitePendente, error) {
		var c ConvitePendente
		var alcadas []string
		if err := l.Scan(&c.ID, &c.Email, &alcadas, &c.EnvioEstado, &c.CriadoPorNome, &c.CriadoEm, &c.ExpiraEm); err != nil {
			return c, err
		}
		c.Alcadas = ordenarAlcadas(alcadas)
		return c, nil
	})
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if convites == nil {
		convites = []ConvitePendente{}
	}

	responder(w, 200, RespostaPessoas{
		Membros:   membros,
		Convites:  convites,
		Conjuntos: copiarConjuntosProntos(),
	})
}

// iniciarEscritaPessoas abre transação serializada por organização e revalida a permissão
// dentro da própria transação (evita corrida entre revogação do ator e gravação).
func (a *API) iniciarEscritaPessoas(w http.ResponseWriter, r *http.Request, s contextoSessao, escopo escopoGestaoPessoas) (pgx.Tx, bool) {
	tx, err := a.banco.Begin(r.Context())
	if err != nil {
		falhaInterna(w, err)
		return nil, false
	}
	if _, err := tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtext('laura-growth:pessoas:' || $1))`, escopo.orgID); err != nil {
		_ = tx.Rollback(r.Context())
		falhaInterna(w, err)
		return nil, false
	}
	if escopo.viaAdminPlataforma {
		var ehAdmin bool
		if err := tx.QueryRow(r.Context(), `SELECT admin_plataforma FROM usuarios WHERE id = $1::uuid FOR SHARE`, s.usuario.ID).Scan(&ehAdmin); err != nil {
			_ = tx.Rollback(r.Context())
			falhaInterna(w, err)
			return nil, false
		}
		if !ehAdmin {
			_ = tx.Rollback(r.Context())
			slog.Warn("acesso administrativo recusado na transação",
				"usuarioId", s.usuario.ID, "organizacaoId", escopo.orgID,
				"metodo", r.Method, "rota", r.URL.Path)
			erro(w, 403, "sem_permissao", "Acesso restrito ao administrador da plataforma.")
			return nil, false
		}
		var existe bool
		if err := tx.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM organizacoes WHERE id = $1::uuid)`, escopo.orgID).Scan(&existe); err != nil {
			_ = tx.Rollback(r.Context())
			falhaInterna(w, err)
			return nil, false
		}
		if !existe {
			_ = tx.Rollback(r.Context())
			erro(w, 404, "nao_encontrado", "Organização não encontrada.")
			return nil, false
		}
		return tx, true
	}
	permitido, err := exigirAlcadaCom(r.Context(), tx, escopo.orgID, s.usuario.ID, "gerenciar_pessoas", true)
	if err != nil {
		_ = tx.Rollback(r.Context())
		falhaInterna(w, err)
		return nil, false
	}
	if !permitido {
		_ = tx.Rollback(r.Context())
		slog.Warn("acesso recusado na transação por falta de alçada",
			"usuarioId", s.usuario.ID, "organizacaoId", escopo.orgID, "alcada", "gerenciar_pessoas",
			"metodo", r.Method, "rota", r.URL.Path)
		erro(w, 403, "sem_permissao", "Você não tem permissão para realizar esta ação.")
		return nil, false
	}
	return tx, true
}

func (a *API) salvarAlcadasPessoaCom(w http.ResponseWriter, r *http.Request, s contextoSessao, escopo escopoGestaoPessoas) {
	org := escopo.orgID
	alvoID := r.PathValue("usuarioId")
	if alvoID == s.usuario.ID {
		erro(w, 403, "auto_edicao", "Você não pode alterar as próprias alçadas.")
		return
	}

	var pedido struct {
		Alcadas []string `json:"alcadas"`
	}
	if !lerJSON(w, r, &pedido) {
		return
	}
	if pedido.Alcadas == nil {
		erro(w, 400, "dados_invalidos", "Informe a lista de alçadas.")
		return
	}
	novas, validas := validarAlcadas(pedido.Alcadas)
	if !validas {
		erro(w, 400, "alcada_invalida", "Há uma alçada inválida na lista.")
		return
	}
	if !uuidValido(alvoID) {
		erro(w, 404, "nao_encontrado", "Pessoa não encontrada nesta organização.")
		return
	}

	tx, ok := a.iniciarEscritaPessoas(w, r, s, escopo)
	if !ok {
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck

	var papel, alvoNome, alvoEmail string
	var criadoEm time.Time
	err := tx.QueryRow(r.Context(), `
		SELECT m.papel, u.nome, u.email, m.criado_em
		FROM membros m
		JOIN usuarios u ON u.id = m.usuario_id
		WHERE m.organizacao_id = $1::uuid AND m.usuario_id = $2::uuid
		FOR UPDATE OF m`, org, alvoID).
		Scan(&papel, &alvoNome, &alvoEmail, &criadoEm)
	if errors.Is(err, pgx.ErrNoRows) {
		erro(w, 404, "nao_encontrado", "Pessoa não encontrada nesta organização.")
		return
	}
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if papel == "dono" {
		erro(w, 403, "dono_protegido", "Não é possível alterar as alçadas do dono da organização.")
		return
	}

	linhasAtuais, err := tx.Query(r.Context(), `
		SELECT alcada FROM alcadas_membro
		WHERE organizacao_id = $1::uuid AND usuario_id = $2::uuid`, org, alvoID)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	atuais, err := pgx.CollectRows(linhasAtuais, func(l pgx.CollectableRow) (string, error) {
		var al string
		err := l.Scan(&al)
		return al, err
	})
	if err != nil {
		falhaInterna(w, err)
		return
	}

	atuaisSet := make(map[string]bool, len(atuais))
	for _, al := range atuais {
		atuaisSet[al] = true
	}
	novasSet := make(map[string]bool, len(novas))
	for _, al := range novas {
		novasSet[al] = true
	}

	for _, al := range todasAlcadas {
		estava := atuaisSet[al]
		fica := novasSet[al]
		if estava && !fica {
			if _, err := tx.Exec(r.Context(), `
				DELETE FROM alcadas_membro
				WHERE organizacao_id = $1::uuid AND usuario_id = $2::uuid AND alcada = $3`,
				org, alvoID, al); err != nil {
				falhaInterna(w, err)
				return
			}
			if err := registrarAcesso(r.Context(), tx, &org, s.usuario.ID, &alvoID, &alvoEmail, "alcada_revogada", detalheAcesso(escopo, map[string]any{"alcada": al})); err != nil {
				falhaInterna(w, err)
				return
			}
			if al == "gerenciar_pessoas" {
				if err := revogarConvitesPendentesDoCriador(r.Context(), tx, org, s.usuario.ID, alvoID, "alcada_criador_revogada", escopo.viaAdminPlataforma, a.agora()); err != nil {
					falhaInterna(w, err)
					return
				}
			}
		} else if !estava && fica {
			if _, err := tx.Exec(r.Context(), `
				INSERT INTO alcadas_membro (organizacao_id, usuario_id, alcada, concedida_por)
				VALUES ($1::uuid, $2::uuid, $3, $4::uuid)
				ON CONFLICT DO NOTHING`,
				org, alvoID, al, s.usuario.ID); err != nil {
				falhaInterna(w, err)
				return
			}
			if err := registrarAcesso(r.Context(), tx, &org, s.usuario.ID, &alvoID, &alvoEmail, "alcada_concedida", detalheAcesso(escopo, map[string]any{"alcada": al})); err != nil {
				falhaInterna(w, err)
				return
			}
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		falhaInterna(w, err)
		return
	}

	responder(w, 200, PessoaMembro{
		UsuarioID: alvoID,
		Nome:      alvoNome,
		Email:     alvoEmail,
		Papel:     papel,
		Alcadas:   novas,
		CriadoEm:  criadoEm,
	})
}

func (a *API) removerPessoaCom(w http.ResponseWriter, r *http.Request, s contextoSessao, escopo escopoGestaoPessoas) {
	org := escopo.orgID
	alvoID := r.PathValue("usuarioId")
	if alvoID == s.usuario.ID {
		erro(w, 403, "auto_remocao", "Você não pode remover a si mesmo da organização.")
		return
	}
	if !uuidValido(alvoID) {
		erro(w, 404, "nao_encontrado", "Pessoa não encontrada nesta organização.")
		return
	}

	tx, ok := a.iniciarEscritaPessoas(w, r, s, escopo)
	if !ok {
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck

	var papel, alvoEmail string
	err := tx.QueryRow(r.Context(), `
		SELECT m.papel, u.email
		FROM membros m
		JOIN usuarios u ON u.id = m.usuario_id
		WHERE m.organizacao_id = $1::uuid AND m.usuario_id = $2::uuid
		FOR UPDATE OF m`, org, alvoID).
		Scan(&papel, &alvoEmail)
	if errors.Is(err, pgx.ErrNoRows) {
		erro(w, 404, "nao_encontrado", "Pessoa não encontrada nesta organização.")
		return
	}
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if papel == "dono" {
		erro(w, 403, "dono_protegido", "Não é possível remover o dono da organização.")
		return
	}

	linhasAlcadas, err := tx.Query(r.Context(), `
		SELECT alcada FROM alcadas_membro
		WHERE organizacao_id = $1::uuid AND usuario_id = $2::uuid`, org, alvoID)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	alcadasAnteriores, err := pgx.CollectRows(linhasAlcadas, func(l pgx.CollectableRow) (string, error) {
		var al string
		err := l.Scan(&al)
		return al, err
	})
	if err != nil {
		falhaInterna(w, err)
		return
	}

	if _, err := tx.Exec(r.Context(), `
		DELETE FROM membros WHERE organizacao_id = $1::uuid AND usuario_id = $2::uuid`,
		org, alvoID); err != nil {
		falhaInterna(w, err)
		return
	}
	if _, err := tx.Exec(r.Context(), `
		UPDATE sessoes SET organizacao_id = NULL
		WHERE usuario_id = $1::uuid AND organizacao_id = $2::uuid`,
		alvoID, org); err != nil {
		falhaInterna(w, err)
		return
	}
	if err := revogarConvitesPendentesDoCriador(r.Context(), tx, org, s.usuario.ID, alvoID, "criador_removido", escopo.viaAdminPlataforma, a.agora()); err != nil {
		falhaInterna(w, err)
		return
	}
	if err := registrarAcesso(r.Context(), tx, &org, s.usuario.ID, &alvoID, &alvoEmail, "membro_removido", detalheAcesso(escopo, map[string]any{
		"alcadasAnteriores": ordenarAlcadas(alcadasAnteriores),
	})); err != nil {
		falhaInterna(w, err)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		falhaInterna(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) listarRegistroAcessosCom(w http.ResponseWriter, r *http.Request, _ contextoSessao, escopo escopoGestaoPessoas) {
	org := escopo.orgID
	limite := 100
	if bruto := r.URL.Query().Get("limite"); bruto != "" {
		v, err := strconv.Atoi(bruto)
		if err != nil || v < 1 || v > 200 {
			erro(w, 400, "dados_invalidos", "Informe um limite entre 1 e 200.")
			return
		}
		limite = v
	}
	registros, err := a.consultarRegistroAcessos(r.Context(), &org, limite)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	responder(w, 200, RespostaRegistroAcessos{Registros: registros})
}

func (a *API) consultarRegistroAcessos(ctx context.Context, orgID *string, limite int) ([]RegistroAcesso, error) {
	var linhas pgx.Rows
	var err error
	const consultaBase = `
		SELECT r.id::text, r.acao,
		       r.ator_id::text, ator.nome, ator.email,
		       r.alvo_usuario_id::text, alvo.nome, COALESCE(alvo.email, r.alvo_email),
		       r.detalhe::text, r.em
		FROM registro_acessos r
		LEFT JOIN usuarios ator ON ator.id = r.ator_id
		LEFT JOIN usuarios alvo ON alvo.id = r.alvo_usuario_id
	`
	if orgID != nil {
		linhas, err = a.banco.Query(ctx, consultaBase+`
			WHERE r.organizacao_id = $1::uuid
			ORDER BY r.em DESC, r.id DESC
			LIMIT $2`, *orgID, limite)
	} else {
		linhas, err = a.banco.Query(ctx, consultaBase+`
			WHERE r.organizacao_id IS NULL
			ORDER BY r.em DESC, r.id DESC
			LIMIT $1`, limite)
	}
	if err != nil {
		return nil, err
	}
	registros, err := pgx.CollectRows(linhas, func(l pgx.CollectableRow) (RegistroAcesso, error) {
		var reg RegistroAcesso
		var detalheBruto string
		if err := l.Scan(
			&reg.ID, &reg.Acao,
			&reg.AtorID, &reg.AtorNome, &reg.AtorEmail,
			&reg.AlvoUsuarioID, &reg.AlvoNome, &reg.AlvoEmail,
			&detalheBruto, &reg.Em,
		); err != nil {
			return reg, err
		}
		reg.Detalhe = map[string]any{}
		if detalheBruto != "" {
			if err := json.Unmarshal([]byte(detalheBruto), &reg.Detalhe); err != nil {
				return reg, err
			}
		}
		return reg, nil
	})
	if err != nil {
		return nil, err
	}
	if registros == nil {
		registros = []RegistroAcesso{}
	}
	return registros, nil
}
