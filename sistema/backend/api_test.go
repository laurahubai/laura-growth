package main

// Testes da etapa 1 contra um PostgreSQL de verdade (banco laura_growth_teste).
// Cobrem o "pronto quando" do roteiro: entrar, ver as duas organizações,
// trocar entre elas e não enxergar organização alheia.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

const senhaTeste = "senha-de-teste-123"

func prepararBanco(t *testing.T) *Banco {
	t.Helper()
	url := os.Getenv("LG_BANCO_TESTE_URL")
	if url == "" {
		t.Skip("LG_BANCO_TESTE_URL não definida")
	}
	ctx := context.Background()
	banco, err := conectar(ctx, url)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "3D000" {
		// Primeira execução: cria o banco de testes a partir do banco principal.
		principal, err2 := conectar(ctx, os.Getenv("LG_BANCO_URL"))
		if err2 != nil {
			t.Fatal(err2)
		}
		_, err2 = principal.Exec(ctx, "CREATE DATABASE laura_growth_teste")
		principal.Close()
		if err2 != nil {
			t.Fatal(err2)
		}
		banco, err = conectar(ctx, url)
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(banco.Close)
	if _, err := banco.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public`); err != nil {
		t.Fatal(err)
	}
	if err := migrar(ctx, banco); err != nil {
		t.Fatal(err)
	}
	if err := semear(ctx, banco, "Paulo", "Paulo@Exemplo.com", senhaTeste); err != nil {
		t.Fatal(err)
	}
	// Uma organização de outra pessoa, que o Paulo não pode ver.
	if _, err := banco.Exec(ctx, `
		WITH u AS (INSERT INTO usuarios (nome, email, senha_hash) VALUES ('Outra', 'outra@exemplo.com', 'x') RETURNING id),
		     o AS (INSERT INTO organizacoes (nome, tipo) VALUES ('Clínica Alheia', 'clinica_estetica') RETURNING id)
		INSERT INTO membros (organizacao_id, usuario_id) SELECT o.id, u.id FROM o, u`); err != nil {
		t.Fatal(err)
	}
	return banco
}

type cliente struct {
	t   *testing.T
	srv *httptest.Server
	c   *http.Client
}

func novoCliente(t *testing.T, api *API) *cliente {
	srv := httptest.NewServer(api.Rotas())
	t.Cleanup(srv.Close)
	jar, _ := cookiejar.New(nil)
	return &cliente{t: t, srv: srv, c: &http.Client{Jar: jar}}
}

func (c *cliente) pedir(metodo, caminho string, corpo any, resposta any) int {
	c.t.Helper()
	var leitor *bytes.Reader
	if corpo != nil {
		b, _ := json.Marshal(corpo)
		leitor = bytes.NewReader(b)
	} else {
		leitor = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(metodo, c.srv.URL+caminho, leitor)
	if corpo != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.c.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	if resposta != nil && res.StatusCode < 300 {
		if err := json.NewDecoder(res.Body).Decode(resposta); err != nil {
			c.t.Fatal(err)
		}
	}
	return res.StatusCode
}

func TestEtapa1AcessoEOrganizacoes(t *testing.T) {
	api := NovaAPI(prepararBanco(t), false)
	c := novoCliente(t, api)

	if st := c.pedir("GET", "/api/sessao", nil, nil); st != 401 {
		t.Fatalf("sem login esperava 401, veio %d", st)
	}
	if st := c.pedir("POST", "/api/sessao", map[string]string{"email": "paulo@exemplo.com", "senha": "errada"}, nil); st != 401 {
		t.Fatalf("senha errada esperava 401, veio %d", st)
	}

	// Pronto quando: você entra na plataforma com sua conta (e-mail sem diferenciar maiúsculas).
	var s Sessao
	if st := c.pedir("POST", "/api/sessao", map[string]string{"email": " PAULO@exemplo.com ", "senha": senhaTeste}, &s); st != 200 {
		t.Fatalf("login esperava 200, veio %d", st)
	}
	// Pronto quando: existem duas organizações, Laura Growth e Laura Hub.
	if len(s.Organizacoes) != 2 || s.Organizacoes[0].Nome != "Laura Growth" || s.Organizacoes[1].Nome != "Laura Hub" {
		t.Fatalf("organizações inesperadas: %+v", s.Organizacoes)
	}
	if s.OrganizacaoAtiva.Nome != "Laura Growth" {
		t.Fatalf("organização ativa inicial: %+v", s.OrganizacaoAtiva)
	}

	// Pronto quando: você troca de uma para a outra e cada uma mostra só o que é dela.
	hub := s.Organizacoes[1]
	if st := c.pedir("PUT", "/api/sessao/organizacao", map[string]string{"organizacaoId": hub.ID}, &s); st != 200 || s.OrganizacaoAtiva.ID != hub.ID {
		t.Fatalf("troca para Laura Hub falhou: %d %+v", st, s.OrganizacaoAtiva)
	}
	var d OrganizacaoDetalhe
	if st := c.pedir("GET", "/api/organizacao", nil, &d); st != 200 || d.Nome != "Laura Hub" || d.Papel != "dono" || d.Membros != 1 {
		t.Fatalf("organização ativa errada: %d %+v", st, d)
	}

	// Isolamento: não dá para ativar a organização de outra pessoa.
	var alheia string
	if err := api.banco.QueryRow(context.Background(), `SELECT id::text FROM organizacoes WHERE nome = 'Clínica Alheia'`).Scan(&alheia); err != nil {
		t.Fatal(err)
	}
	if st := c.pedir("PUT", "/api/sessao/organizacao", map[string]string{"organizacaoId": alheia}, nil); st != 403 {
		t.Fatalf("organização alheia esperava 403, veio %d", st)
	}
	if st := c.pedir("PUT", "/api/sessao/organizacao", map[string]string{"organizacaoId": "nao-e-um-id"}, nil); st != 403 {
		t.Fatalf("id inválido esperava 403, veio %d", st)
	}
	if st := c.pedir("GET", "/api/organizacao", nil, &d); st != 200 || d.Nome != "Laura Hub" {
		t.Fatalf("depois da recusa a organização ativa mudou: %+v", d)
	}

	// Sair encerra a sessão.
	if st := c.pedir("DELETE", "/api/sessao", nil, nil); st != 204 {
		t.Fatalf("sair esperava 204, veio %d", st)
	}
	if st := c.pedir("GET", "/api/sessao", nil, nil); st != 401 {
		t.Fatalf("depois de sair esperava 401, veio %d", st)
	}
}

func TestPedidoDeOutroSiteERecusado(t *testing.T) {
	api := NovaAPI(prepararBanco(t), false)
	srv := httptest.NewServer(api.Rotas())
	defer srv.Close()
	req, _ := http.NewRequest("POST", srv.URL+"/api/sessao", bytes.NewBufferString(`{"email":"paulo@exemplo.com","senha":"`+senhaTeste+`"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://site-malicioso.example")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 403 {
		t.Fatalf("esperava 403, veio %d", res.StatusCode)
	}
}

func TestMuitasTentativasBloqueiam(t *testing.T) {
	api := NovaAPI(prepararBanco(t), false)
	c := novoCliente(t, api)
	for i := 0; i < 8; i++ {
		c.pedir("POST", "/api/sessao", map[string]string{"email": "paulo@exemplo.com", "senha": "errada"}, nil)
	}
	if st := c.pedir("POST", "/api/sessao", map[string]string{"email": "paulo@exemplo.com", "senha": senhaTeste}, nil); st != 429 {
		t.Fatalf("esperava 429 depois de 8 erros, veio %d", st)
	}
}
