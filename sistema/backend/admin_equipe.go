package main

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

// ---- Administração → Equipe (B8) ----

type MembroEquipe struct {
	UsuarioID       string   `json:"usuarioId"`
	Nome            string   `json:"nome"`
	Email           string   `json:"email"`
	AdminPlataforma bool     `json:"adminPlataforma"`
	Organizacoes    []string `json:"organizacoes"`
}

type RespostaEquipeAdmin struct {
	Equipe    []MembroEquipe   `json:"equipe"`
	Registros []RegistroAcesso `json:"registros"`
}

func (a *API) listarEquipeAdmin(w http.ResponseWriter, r *http.Request, _ contextoSessao) {
	linhas, err := a.banco.Query(r.Context(), `
		SELECT u.id::text, u.nome, u.email, u.admin_plataforma,
		       array_agg(o.nome ORDER BY o.nome) AS organizacoes
		FROM usuarios u
		JOIN membros m ON m.usuario_id = u.id
		JOIN organizacoes o ON o.id = m.organizacao_id AND o.da_casa = true
		GROUP BY u.id, u.nome, u.email, u.admin_plataforma
		ORDER BY u.admin_plataforma DESC, lower(u.nome), lower(u.email)`)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	equipe, err := pgx.CollectRows(linhas, func(l pgx.CollectableRow) (MembroEquipe, error) {
		var m MembroEquipe
		err := l.Scan(&m.UsuarioID, &m.Nome, &m.Email, &m.AdminPlataforma, &m.Organizacoes)
		if m.Organizacoes == nil {
			m.Organizacoes = []string{}
		}
		return m, err
	})
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if equipe == nil {
		equipe = []MembroEquipe{}
	}
	registros, err := a.consultarRegistroAcessos(r.Context(), nil, 100)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	responder(w, 200, RespostaEquipeAdmin{
		Equipe:    equipe,
		Registros: registros,
	})
}

func (a *API) atualizarEquipeAdmin(w http.ResponseWriter, r *http.Request, s contextoSessao) {
	var pedido struct {
		AdminPlataforma *bool `json:"adminPlataforma"`
	}
	if !lerJSON(w, r, &pedido) {
		return
	}
	if pedido.AdminPlataforma == nil {
		erro(w, 400, "dados_invalidos", "Informe se a pessoa é administradora da plataforma.")
		return
	}
	alvoID := r.PathValue("usuarioId")

	tx, err := a.banco.Begin(r.Context())
	if err != nil {
		falhaInterna(w, err)
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck

	// Serializa alterações de admin da plataforma para nunca remover o último administrador.
	if _, err := tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(7140002)`); err != nil {
		falhaInterna(w, err)
		return
	}

	var nome, email string
	var atual bool
	var orgsCasa []string
	err = tx.QueryRow(r.Context(), `
		SELECT u.nome, u.email, u.admin_plataforma,
		       COALESCE(array_agg(o.nome ORDER BY o.nome) FILTER (WHERE o.id IS NOT NULL), '{}')
		FROM usuarios u
		LEFT JOIN membros m ON m.usuario_id = u.id
		LEFT JOIN organizacoes o ON o.id = m.organizacao_id AND o.da_casa = true
		WHERE u.id::text = $1
		GROUP BY u.id, u.nome, u.email, u.admin_plataforma`, alvoID).
		Scan(&nome, &email, &atual, &orgsCasa)
	if errors.Is(err, pgx.ErrNoRows) {
		erro(w, 404, "nao_encontrado", "Pessoa não encontrada.")
		return
	}
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if len(orgsCasa) == 0 {
		erro(w, 403, "nao_membro_casa", "Apenas membros das organizações da casa (Laura Growth ou Laura Hub) podem ser administradores da plataforma.")
		return
	}

	novoEstado := *pedido.AdminPlataforma
	if !novoEstado && atual {
		var totalAdmins int
		if err := tx.QueryRow(r.Context(), `SELECT count(*) FROM usuarios WHERE admin_plataforma = true`).Scan(&totalAdmins); err != nil {
			falhaInterna(w, err)
			return
		}
		if totalAdmins <= 1 {
			erro(w, 409, "ultimo_admin", "Não é possível remover o último administrador da plataforma.")
			return
		}
	}

	if novoEstado != atual {
		if _, err := tx.Exec(r.Context(), `UPDATE usuarios SET admin_plataforma = $2 WHERE id::text = $1`, alvoID, novoEstado); err != nil {
			falhaInterna(w, err)
			return
		}
		acao := "admin_plataforma_concedido"
		if !novoEstado {
			acao = "admin_plataforma_revogado"
		}
		if err := registrarAcesso(r.Context(), tx, nil, s.usuario.ID, &alvoID, &email, acao, map[string]bool{
			"adminPlataforma": novoEstado,
		}); err != nil {
			falhaInterna(w, err)
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		falhaInterna(w, err)
		return
	}

	responder(w, 200, MembroEquipe{
		UsuarioID:       alvoID,
		Nome:            nome,
		Email:           email,
		AdminPlataforma: novoEstado,
		Organizacoes:    orgsCasa,
	})
}
