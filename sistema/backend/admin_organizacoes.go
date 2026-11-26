package main

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

// ---- Administração → Organizações (B11) ----

type OrganizacaoAdmin struct {
	ID       string    `json:"id"`
	Nome     string    `json:"nome"`
	Tipo     string    `json:"tipo"`
	DaCasa   bool      `json:"daCasa"`
	CriadaEm time.Time `json:"criadaEm"`
	Membros  int       `json:"membros"`
}

type RespostaOrganizacoesAdmin struct {
	Organizacoes []OrganizacaoAdmin `json:"organizacoes"`
}

func (a *API) listarOrganizacoesAdmin(w http.ResponseWriter, r *http.Request, _ contextoSessao) {
	linhas, err := a.banco.Query(r.Context(), `
		SELECT o.id::text, o.nome, o.tipo, o.da_casa, o.criada_em,
		       count(m.usuario_id)::int AS membros
		FROM organizacoes o
		LEFT JOIN membros m ON m.organizacao_id = o.id
		GROUP BY o.id, o.nome, o.tipo, o.da_casa, o.criada_em
		ORDER BY o.da_casa DESC, lower(o.nome), o.criada_em`)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	orgs, err := pgx.CollectRows(linhas, func(l pgx.CollectableRow) (OrganizacaoAdmin, error) {
		var o OrganizacaoAdmin
		err := l.Scan(&o.ID, &o.Nome, &o.Tipo, &o.DaCasa, &o.CriadaEm, &o.Membros)
		return o, err
	})
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if orgs == nil {
		orgs = []OrganizacaoAdmin{}
	}
	responder(w, 200, RespostaOrganizacoesAdmin{Organizacoes: orgs})
}

func (a *API) resolverEscopoPessoasAdmin(w http.ResponseWriter, r *http.Request) (escopoGestaoPessoas, bool) {
	orgID := r.PathValue("orgId")
	if !uuidValido(orgID) {
		erro(w, 404, "nao_encontrado", "Organização não encontrada.")
		return escopoGestaoPessoas{}, false
	}
	var existe bool
	err := a.banco.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM organizacoes WHERE id = $1::uuid)`, orgID).Scan(&existe)
	if err != nil {
		falhaInterna(w, err)
		return escopoGestaoPessoas{}, false
	}
	if !existe {
		erro(w, 404, "nao_encontrado", "Organização não encontrada.")
		return escopoGestaoPessoas{}, false
	}
	return escopoGestaoPessoas{
		orgID:              orgID,
		viaAdminPlataforma: true,
	}, true
}

func (a *API) comEscopoPessoasAdmin(h func(http.ResponseWriter, *http.Request, contextoSessao, escopoGestaoPessoas)) manipulador {
	return func(w http.ResponseWriter, r *http.Request, s contextoSessao) {
		escopo, ok := a.resolverEscopoPessoasAdmin(w, r)
		if !ok {
			return
		}
		h(w, r, s, escopo)
	}
}
