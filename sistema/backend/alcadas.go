package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Alçadas fixas da plataforma (Etapa 2B, PRM-01 a PRM-05).
var todasAlcadas = []string{
	"ver_marca_posts",
	"cmo_dna",
	"aprovar_plano",
	"editar_posts",
	"aprovar_posts",
	"conectar_redes",
	"ver_creditos",
	"gerenciar_pessoas",
}

var alcadasValidas = func() map[string]bool {
	m := make(map[string]bool, len(todasAlcadas))
	for _, a := range todasAlcadas {
		m[a] = true
	}
	return m
}()

// ConjuntoAlcadas representa um conjunto pronto fixo em código (apenas pré-preenche caixas na tela).
type ConjuntoAlcadas struct {
	Codigo  string   `json:"codigo"`
	Nome    string   `json:"nome"`
	Alcadas []string `json:"alcadas"`
}

var conjuntosProntos = []ConjuntoAlcadas{
	{
		Codigo:  "somente_leitura",
		Nome:    "Somente leitura",
		Alcadas: []string{"ver_marca_posts", "ver_creditos"},
	},
	{
		Codigo:  "criador_conteudo",
		Nome:    "Criador de conteúdo",
		Alcadas: []string{"ver_marca_posts", "cmo_dna", "editar_posts"},
	},
	{
		Codigo:  "aprovador",
		Nome:    "Aprovador",
		Alcadas: []string{"ver_marca_posts", "aprovar_plano", "aprovar_posts"},
	},
	{
		Codigo: "gestor_marketing",
		Nome:   "Gestor de marketing",
		Alcadas: []string{
			"ver_marca_posts",
			"cmo_dna",
			"aprovar_plano",
			"editar_posts",
			"aprovar_posts",
			"conectar_redes",
			"ver_creditos",
		},
	},
}

func copiarConjuntosProntos() []ConjuntoAlcadas {
	out := make([]ConjuntoAlcadas, len(conjuntosProntos))
	for i, c := range conjuntosProntos {
		alcadas := make([]string, len(c.Alcadas))
		copy(alcadas, c.Alcadas)
		out[i] = ConjuntoAlcadas{
			Codigo:  c.Codigo,
			Nome:    c.Nome,
			Alcadas: alcadas,
		}
	}
	return out
}

func uuidValido(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < 36; i++ {
		c := s[i]
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// ordenarAlcadas devolve as alçadas únicas na ordem canônica definida em todasAlcadas.
func ordenarAlcadas(entrada []string) []string {
	presentes := make(map[string]bool, len(entrada))
	for _, a := range entrada {
		presentes[a] = true
	}
	out := make([]string, 0, len(presentes))
	for _, a := range todasAlcadas {
		if presentes[a] {
			out = append(out, a)
		}
	}
	return out
}

// validarAlcadas valida que todas as alçadas existem e devolve a lista única ordenada.
func validarAlcadas(entrada []string) ([]string, bool) {
	presentes := make(map[string]bool, len(entrada))
	for _, a := range entrada {
		if !alcadasValidas[a] {
			return nil, false
		}
		presentes[a] = true
	}
	out := make([]string, 0, len(presentes))
	for _, a := range todasAlcadas {
		if presentes[a] {
			out = append(out, a)
		}
	}
	return out, true
}

// exigirAlcada verifica no banco, a cada pedido (sem cache), se o usuário possui a alçada
// na organização informada. O papel 'dono' passa sempre.
func (a *API) exigirAlcada(ctx context.Context, org, usuarioID, alcada string) (bool, error) {
	return exigirAlcadaCom(ctx, a.banco, org, usuarioID, alcada, false)
}

func exigirAlcadaCom(ctx context.Context, q executorSQL, org, usuarioID, alcada string, travar bool) (bool, error) {
	if !uuidValido(org) || !uuidValido(usuarioID) || !alcadasValidas[alcada] {
		return false, nil
	}
	consulta := `
		SELECT m.papel,
		       EXISTS (
		         SELECT 1 FROM alcadas_membro am
		         WHERE am.organizacao_id = m.organizacao_id
		           AND am.usuario_id = m.usuario_id
		           AND am.alcada = $3
		       )
		FROM membros m
		WHERE m.organizacao_id = $1::uuid AND m.usuario_id = $2::uuid`
	if travar {
		consulta += ` FOR UPDATE OF m`
	}
	var papel string
	var temAlcada bool
	err := q.QueryRow(ctx, consulta, org, usuarioID, alcada).Scan(&papel, &temAlcada)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if papel == "dono" {
		return true, nil
	}
	return temAlcada, nil
}

// autorizarAlcada garante que há organização ativa da qual a pessoa é membro e que ela possui a alçada.
func (a *API) autorizarAlcada(w http.ResponseWriter, r *http.Request, s contextoSessao, alcada string) (string, bool) {
	org, err := a.orgAtiva(r.Context(), s)
	if errors.Is(err, errSemOrganizacao) {
		erro(w, 404, "sem_organizacao", "Nenhuma organização ativa.")
		return "", false
	}
	if err != nil {
		falhaInterna(w, err)
		return "", false
	}
	ok, err := a.exigirAlcada(r.Context(), org, s.usuario.ID, alcada)
	if err != nil {
		falhaInterna(w, err)
		return "", false
	}
	if !ok {
		slog.Warn("acesso recusado por falta de alçada",
			"usuarioId", s.usuario.ID, "organizacaoId", org, "alcada", alcada,
			"metodo", r.Method, "rota", r.URL.Path)
		erro(w, 403, "sem_permissao", "Você não tem permissão para realizar esta ação.")
		return "", false
	}
	return org, true
}

// alcadasDoUsuarioNaOrg lista as alçadas efetivas do usuário na organização (dono recebe todas as 8).
func (a *API) alcadasDoUsuarioNaOrg(ctx context.Context, orgID, usuarioID string) ([]string, error) {
	if !uuidValido(orgID) || !uuidValido(usuarioID) {
		return []string{}, nil
	}
	var papel string
	err := a.banco.QueryRow(ctx, `
		SELECT papel FROM membros WHERE organizacao_id = $1::uuid AND usuario_id = $2::uuid`,
		orgID, usuarioID).Scan(&papel)
	if errors.Is(err, pgx.ErrNoRows) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	if papel == "dono" {
		out := make([]string, len(todasAlcadas))
		copy(out, todasAlcadas)
		return out, nil
	}
	linhas, err := a.banco.Query(ctx, `
		SELECT alcada FROM alcadas_membro WHERE organizacao_id = $1::uuid AND usuario_id = $2::uuid`,
		orgID, usuarioID)
	if err != nil {
		return nil, err
	}
	lista, err := pgx.CollectRows(linhas, func(l pgx.CollectableRow) (string, error) {
		var al string
		err := l.Scan(&al)
		return al, err
	})
	if err != nil {
		return nil, err
	}
	return ordenarAlcadas(lista), nil
}

type escritorSQL interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func registrarAcesso(ctx context.Context, exec escritorSQL, orgID *string, atorID string, alvoUsuarioID *string, alvoEmail *string, acao string, detalhe any) error {
	detalheJSON := []byte("{}")
	if detalhe != nil {
		b, err := json.Marshal(detalhe)
		if err != nil {
			return err
		}
		detalheJSON = b
	}
	_, err := exec.Exec(ctx, `
		INSERT INTO registro_acessos (organizacao_id, ator_id, alvo_usuario_id, alvo_email, acao, detalhe)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6::jsonb)`,
		orgID, atorID, alvoUsuarioID, alvoEmail, acao, string(detalheJSON))
	return err
}
