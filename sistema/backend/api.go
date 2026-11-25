package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	nomeCookie    = "lg_sessao"
	duracaoSessao = 30 * 24 * time.Hour
)

type API struct {
	banco               *Banco
	cookieSeguro        bool
	cifrador            *Cifrador
	limite              *limitador
	agora               func() time.Time
	resolverIP          resolvedorIP
	permitirRedeInterna bool        // usado apenas em testes locais com httptest.NewTLSServer
	tlsTeste            *tls.Config // usado apenas em testes locais com httptest.NewTLSServer
}

func NovaAPI(banco *Banco, cookieSeguro bool, cifrador *Cifrador) *API {
	return &API{
		banco:        banco,
		cookieSeguro: cookieSeguro,
		cifrador:     cifrador,
		limite:       novoLimitador(),
		agora:        time.Now,
		resolverIP:   resolverDNS,
	}
}

// ---- Formato das respostas (espelha packages/contratos) ----

type Usuario struct {
	ID              string `json:"id"`
	Nome            string `json:"nome"`
	Email           string `json:"email"`
	AdminPlataforma bool   `json:"adminPlataforma"`
}

type Organizacao struct {
	ID   string `json:"id"`
	Nome string `json:"nome"`
	Tipo string `json:"tipo"`
}

type Sessao struct {
	Usuario          Usuario       `json:"usuario"`
	Organizacoes     []Organizacao `json:"organizacoes"`
	OrganizacaoAtiva Organizacao   `json:"organizacaoAtiva"`
}

type OrganizacaoDetalhe struct {
	Organizacao
	CriadaEm time.Time `json:"criadaEm"`
	Papel    string    `json:"papel"`
	Membros  int       `json:"membros"`
}

// ---- Rotas ----

func (a *API) Rotas() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /api/saude", func(w http.ResponseWriter, r *http.Request) { responder(w, 200, map[string]string{"estado": "ok"}) })
	m.HandleFunc("POST /api/sessao", a.entrar)
	m.HandleFunc("GET /api/sessao", a.comSessao(a.verSessao))
	m.HandleFunc("DELETE /api/sessao", a.comSessao(a.sair))
	m.HandleFunc("PUT /api/sessao/organizacao", a.comSessao(a.trocarOrganizacao))
	m.HandleFunc("GET /api/organizacao", a.comSessao(a.verOrganizacao))
	a.rotasKit(m)
	a.rotasAdmin(m)
	return protegerOrigem(m)
}

// protegerOrigem recusa alterações vindas de outro site (proteção contra CSRF),
// além do SameSite do cookie. A tela (Next.js) repassa os pedidos para cá e informa
// o endereço que o navegador usou em X-Forwarded-Host; um site malicioso não consegue
// definir esse cabeçalho num pedido entre sites sem a permissão (CORS) que não damos.
func protegerOrigem(proximo http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if origem := r.Header.Get("Origin"); origem != "" {
				esperado := r.Header.Get("X-Forwarded-Host")
				if esperado == "" {
					esperado = r.Host
				}
				u, err := url.Parse(origem)
				if err != nil || !strings.EqualFold(u.Host, esperado) {
					erro(w, 403, "origem_recusada", "Pedido recusado.")
					return
				}
			}
		}
		proximo.ServeHTTP(w, r)
	})
}

// ---- Sessão ----

type contextoSessao struct {
	tokenHash []byte
	usuario   Usuario
	orgAtiva  string // pode vir vazio se a organização ativa deixou de existir
}

type manipulador func(w http.ResponseWriter, r *http.Request, s contextoSessao)

func (a *API) comSessao(h manipulador) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(nomeCookie)
		if err != nil || c.Value == "" {
			erro(w, 401, "sem_sessao", "Entre com seu e-mail e senha.")
			return
		}
		token, err := base64.RawURLEncoding.DecodeString(c.Value)
		if err != nil || len(token) != 32 {
			a.apagarCookie(w)
			erro(w, 401, "sem_sessao", "Entre com seu e-mail e senha.")
			return
		}
		hash := sha256.Sum256(token)
		var s contextoSessao
		var org *string
		err = a.banco.QueryRow(r.Context(), `
			SELECT u.id::text, u.nome, u.email, s.organizacao_id::text
			FROM sessoes s JOIN usuarios u ON u.id = s.usuario_id
			WHERE s.token_hash = $1 AND s.expira_em > $2`, hash[:], a.agora()).
			Scan(&s.usuario.ID, &s.usuario.Nome, &s.usuario.Email, &org)
		if errors.Is(err, pgx.ErrNoRows) {
			a.apagarCookie(w)
			erro(w, 401, "sem_sessao", "Sua sessão terminou. Entre de novo.")
			return
		}
		if err != nil {
			falhaInterna(w, err)
			return
		}
		s.tokenHash = hash[:]
		if org != nil {
			s.orgAtiva = *org
		}
		h(w, r, s)
	}
}

func (a *API) entrar(w http.ResponseWriter, r *http.Request) {
	var pedido struct {
		Email string `json:"email"`
		Senha string `json:"senha"`
	}
	if !lerJSON(w, r, &pedido) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(pedido.Email))
	if email == "" || pedido.Senha == "" {
		erro(w, 400, "dados_incompletos", "Informe e-mail e senha.")
		return
	}
	agora := a.agora()
	if a.limite.bloqueado(email, agora) {
		erro(w, 429, "muitas_tentativas", "Muitas tentativas. Aguarde 15 minutos e tente de novo.")
		return
	}

	var usuarioID, hash string
	err := a.banco.QueryRow(r.Context(), `SELECT id::text, senha_hash FROM usuarios WHERE lower(email) = $1`, email).Scan(&usuarioID, &hash)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		falhaInterna(w, err)
		return
	}
	if !senhaConfere(hash, pedido.Senha) {
		a.limite.registrarFalha(email, agora)
		erro(w, 401, "credenciais_invalidas", "E-mail ou senha não conferem.")
		return
	}
	a.limite.limpar(email)

	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		falhaInterna(w, err)
		return
	}
	hashToken := sha256.Sum256(token)
	// A organização ativa começa na primeira em ordem alfabética.
	_, err = a.banco.Exec(r.Context(), `
		INSERT INTO sessoes (token_hash, usuario_id, organizacao_id, expira_em)
		VALUES ($1, $2, (SELECT o.id FROM membros m JOIN organizacoes o ON o.id = m.organizacao_id
		                 WHERE m.usuario_id = $2 ORDER BY o.nome LIMIT 1), $3)`,
		hashToken[:], usuarioID, agora.Add(duracaoSessao))
	if err != nil {
		falhaInterna(w, err)
		return
	}
	// Limpeza oportuna de sessões vencidas desta pessoa.
	_, _ = a.banco.Exec(r.Context(), `DELETE FROM sessoes WHERE usuario_id = $1 AND expira_em <= $2`, usuarioID, agora)

	http.SetCookie(w, &http.Cookie{
		Name: nomeCookie, Value: base64.RawURLEncoding.EncodeToString(token), Path: "/",
		HttpOnly: true, Secure: a.cookieSeguro, SameSite: http.SameSiteLaxMode,
		Expires: agora.Add(duracaoSessao),
	})
	sessao, err := a.montarSessao(r.Context(), hashToken[:])
	if err != nil {
		falhaInterna(w, err)
		return
	}
	responder(w, 200, sessao)
}

func (a *API) verSessao(w http.ResponseWriter, r *http.Request, s contextoSessao) {
	sessao, err := a.montarSessao(r.Context(), s.tokenHash)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	responder(w, 200, sessao)
}

func (a *API) sair(w http.ResponseWriter, r *http.Request, s contextoSessao) {
	if _, err := a.banco.Exec(r.Context(), `DELETE FROM sessoes WHERE token_hash = $1`, s.tokenHash); err != nil {
		falhaInterna(w, err)
		return
	}
	a.apagarCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// trocarOrganizacao só aceita organizações das quais a pessoa é membro (ORG-01, ORG-03).
func (a *API) trocarOrganizacao(w http.ResponseWriter, r *http.Request, s contextoSessao) {
	var pedido struct {
		OrganizacaoID string `json:"organizacaoId"`
	}
	if !lerJSON(w, r, &pedido) {
		return
	}
	tag, err := a.banco.Exec(r.Context(), `
		UPDATE sessoes SET organizacao_id = m.organizacao_id
		FROM membros m
		WHERE sessoes.token_hash = $1 AND m.usuario_id = $2 AND m.organizacao_id::text = $3`,
		s.tokenHash, s.usuario.ID, pedido.OrganizacaoID)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		erro(w, 403, "sem_acesso", "Você não tem acesso a esta organização.")
		return
	}
	a.verSessao(w, r, s)
}

// verOrganizacao devolve só a organização ativa da sessão, e só se a pessoa ainda for membro.
func (a *API) verOrganizacao(w http.ResponseWriter, r *http.Request, s contextoSessao) {
	var d OrganizacaoDetalhe
	err := a.banco.QueryRow(r.Context(), `
		SELECT o.id::text, o.nome, o.tipo, o.criada_em, m.papel,
		       (SELECT count(*) FROM membros t WHERE t.organizacao_id = o.id)
		FROM organizacoes o JOIN membros m ON m.organizacao_id = o.id AND m.usuario_id = $1
		WHERE o.id::text = $2`, s.usuario.ID, s.orgAtiva).
		Scan(&d.ID, &d.Nome, &d.Tipo, &d.CriadaEm, &d.Papel, &d.Membros)
	if errors.Is(err, pgx.ErrNoRows) {
		erro(w, 404, "sem_organizacao", "Nenhuma organização ativa.")
		return
	}
	if err != nil {
		falhaInterna(w, err)
		return
	}
	responder(w, 200, d)
}

func (a *API) montarSessao(ctx context.Context, tokenHash []byte) (Sessao, error) {
	var s Sessao
	var ativa *string
	err := a.banco.QueryRow(ctx, `
		SELECT u.id::text, u.nome, u.email, u.admin_plataforma, s.organizacao_id::text
		FROM sessoes s JOIN usuarios u ON u.id = s.usuario_id WHERE s.token_hash = $1`, tokenHash).
		Scan(&s.Usuario.ID, &s.Usuario.Nome, &s.Usuario.Email, &s.Usuario.AdminPlataforma, &ativa)
	if err != nil {
		return s, err
	}
	linhas, err := a.banco.Query(ctx, `
		SELECT o.id::text, o.nome, o.tipo FROM membros m JOIN organizacoes o ON o.id = m.organizacao_id
		WHERE m.usuario_id = $1 ORDER BY o.nome`, s.Usuario.ID)
	if err != nil {
		return s, err
	}
	s.Organizacoes, err = pgx.CollectRows(linhas, func(l pgx.CollectableRow) (Organizacao, error) {
		var o Organizacao
		err := l.Scan(&o.ID, &o.Nome, &o.Tipo)
		return o, err
	})
	if err != nil {
		return s, err
	}
	for _, o := range s.Organizacoes {
		if ativa != nil && o.ID == *ativa {
			s.OrganizacaoAtiva = o
		}
	}
	// Se a organização ativa sumiu (ou a pessoa perdeu o acesso), cai na primeira disponível.
	if s.OrganizacaoAtiva.ID == "" && len(s.Organizacoes) > 0 {
		s.OrganizacaoAtiva = s.Organizacoes[0]
		_, err = a.banco.Exec(ctx, `UPDATE sessoes SET organizacao_id = $2 WHERE token_hash = $1`, tokenHash, s.OrganizacaoAtiva.ID)
	}
	if s.Organizacoes == nil {
		s.Organizacoes = []Organizacao{}
	}
	return s, err
}

func (a *API) apagarCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: nomeCookie, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: a.cookieSeguro, SameSite: http.SameSiteLaxMode})
}

// ---- Utilidades ----

func lerJSON(w http.ResponseWriter, r *http.Request, destino any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		erro(w, 415, "formato_invalido", "Envie os dados em JSON.")
		return false
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(destino); err != nil {
		erro(w, 400, "formato_invalido", "Dados em formato inválido.")
		return false
	}
	return true
}

func responder(w http.ResponseWriter, status int, corpo any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(corpo)
}

func erro(w http.ResponseWriter, status int, codigo, mensagem string) {
	responder(w, status, map[string]string{"codigo": codigo, "mensagem": mensagem})
}

func falhaInterna(w http.ResponseWriter, err error) {
	slog.Error("falha interna", "erro", err)
	erro(w, 500, "falha_interna", "Algo deu errado. Tente de novo.")
}
