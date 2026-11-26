package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

type registroEmailFake struct {
	Para    string
	Assunto string
	Texto   string
}

type adaptadorEmailFake struct {
	mu       sync.Mutex
	enviados []registroEmailFake
	erro     error
}

func (f *adaptadorEmailFake) enviar(_ context.Context, para, assunto, texto string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.erro != nil {
		return f.erro
	}
	f.enviados = append(f.enviados, registroEmailFake{
		Para:    para,
		Assunto: assunto,
		Texto:   texto,
	})
	return nil
}

func (f *adaptadorEmailFake) total() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.enviados)
}

func (f *adaptadorEmailFake) ultimo() registroEmailFake {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.enviados[len(f.enviados)-1]
}

func (f *adaptadorEmailFake) definirErro(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.erro = err
}

func novaAPIConvitesTeste(t *testing.T) (*API, *adaptadorEmailFake) {
	t.Helper()
	t.Setenv("LG_EMAIL_ZOHO_CHAVE", "")
	t.Setenv("LG_EMAIL_ZOHO_URL", "")
	t.Setenv("LG_ENDERECO_PUBLICO", "")
	api := NovaAPI(prepararBanco(t), false, nil)
	fake := &adaptadorEmailFake{}
	api.email = fake
	api.enderecoPublico = "https://app.lauragrowth.test"
	return api, fake
}

func extrairTokenDeLink(t *testing.T, link string) string {
	t.Helper()
	const marcador = "/convite/"
	idx := strings.LastIndex(link, marcador)
	if idx < 0 {
		t.Fatalf("link sem %q: %q", marcador, link)
	}
	token := link[idx+len(marcador):]
	if token == "" {
		t.Fatalf("token vazio no link: %q", link)
	}
	return token
}

func TestAdaptadorEmailZohoContrato(t *testing.T) {
	const chaveSecreta = "zoho-chave-super-secreta-123456"

	t.Run("sucesso", func(t *testing.T) {
		var metodoRecebido, caminhoRecebido, authRecebido string
		var corpoRecebido map[string]string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			metodoRecebido = r.Method
			caminhoRecebido = r.URL.Path
			authRecebido = r.Header.Get("Authorization")
			_ = json.NewDecoder(r.Body).Decode(&corpoRecebido)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"status":"sent"}`))
		}))
		defer srv.Close()

		adp := novoEnviadorEmailZoho(srv.URL, chaveSecreta)
		if adp.cliente.Timeout != 45*time.Second {
			t.Fatalf("timeout esperado 45s, veio %v", adp.cliente.Timeout)
		}
		if err := adp.enviar(context.Background(), "pessoa@exemplo.com", "Assunto", "Corpo com link"); err != nil {
			t.Fatalf("envio válido falhou: %v", err)
		}
		if metodoRecebido != http.MethodPost || caminhoRecebido != "/v1/email/send" {
			t.Fatalf("requisição inesperada: %s %s", metodoRecebido, caminhoRecebido)
		}
		if authRecebido != "Bearer "+chaveSecreta {
			t.Fatalf("Authorization inesperado: %q", authRecebido)
		}
		if corpoRecebido["to"] != "pessoa@exemplo.com" || corpoRecebido["subject"] != "Assunto" || corpoRecebido["text"] != "Corpo com link" {
			t.Fatalf("corpo inesperado: %+v", corpoRecebido)
		}
	})

	t.Run("status nao 200 sem vazar chave", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(502)
			_, _ = w.Write([]byte(`{"error":"bad gateway ` + chaveSecreta + `"}`))
		}))
		defer srv.Close()

		adp := novoEnviadorEmailZoho(srv.URL, chaveSecreta)
		err := adp.enviar(context.Background(), "a@b.com", "s", "t")
		if err == nil {
			t.Fatal("esperava erro para HTTP 502")
		}
		if strings.Contains(err.Error(), chaveSecreta) {
			t.Fatalf("erro vazou a chave secreta: %v", err)
		}
	})

	t.Run("corpo invalido ou status diferente de sent", func(t *testing.T) {
		for _, payload := range []string{`{"status":"queued"}`, `nao-e-json`} {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
				_, _ = w.Write([]byte(payload))
			}))
			adp := novoEnviadorEmailZoho(srv.URL, chaveSecreta)
			err := adp.enviar(context.Background(), "a@b.com", "s", "t")
			srv.Close()
			if err == nil {
				t.Fatalf("esperava erro para payload %q", payload)
			}
			if strings.Contains(err.Error(), chaveSecreta) {
				t.Fatalf("erro vazou a chave: %v", err)
			}
		}
	})

	t.Run("recusa redirecionamento", func(t *testing.T) {
		destinoChamado := false
		destino := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			destinoChamado = true
			_, _ = w.Write([]byte(`{"status":"sent"}`))
		}))
		defer destino.Close()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, destino.URL, http.StatusFound)
		}))
		defer srv.Close()

		adp := novoEnviadorEmailZoho(srv.URL, chaveSecreta)
		err := adp.enviar(context.Background(), "a@b.com", "s", "t")
		if err == nil {
			t.Fatal("esperava erro ao tentar redirecionar")
		}
		if destinoChamado {
			t.Fatal("destino do redirecionamento não deveria ser chamado")
		}
		if strings.Contains(err.Error(), chaveSecreta) {
			t.Fatalf("erro vazou a chave: %v", err)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(120 * time.Millisecond)
			_, _ = w.Write([]byte(`{"status":"sent"}`))
		}))
		defer srv.Close()

		adp := novoEnviadorEmailZohoComTimeout(srv.URL, chaveSecreta, 25*time.Millisecond)
		err := adp.enviar(context.Background(), "a@b.com", "s", "t")
		if err == nil {
			t.Fatal("esperava erro de timeout")
		}
		if strings.Contains(err.Error(), chaveSecreta) {
			t.Fatalf("erro vazou a chave: %v", err)
		}
	})

	t.Run("nao configurado", func(t *testing.T) {
		adp := novoEnviadorEmailZoho("", "")
		if adp.urlBase != "http://127.0.0.1:8030" {
			t.Fatalf("URL padrão inesperada: %q", adp.urlBase)
		}
		err := adp.enviar(context.Background(), "a@b.com", "s", "t")
		if !errors.Is(err, errEmailNaoConfigurado) {
			t.Fatalf("esperava errEmailNaoConfigurado, veio %v", err)
		}
	})
}

func TestConvitesB1CriarPermissoesEEnvio(t *testing.T) {
	api, fake := novaAPIConvitesTeste(t)
	paulo := novoCliente(t, api)
	sessPaulo := entrarComoPaulo(t, paulo)
	growth, hub := sessPaulo.Organizacoes[0], sessPaulo.Organizacoes[1]

	// Membro sem gerenciar_pessoas -> 403 em todas as rotas de gestão de convite.
	criarMembroNaOrg(t, api.banco, growth.ID, "Sem Acesso", "sem-acesso@exemplo.com", "ver_marca_posts")
	semAcesso := novoCliente(t, api)
	entrarComEmail(t, semAcesso, "sem-acesso@exemplo.com")

	if st := semAcesso.pedir("POST", "/api/convites", map[string]any{
		"email":   "novo@exemplo.com",
		"alcadas": []string{"ver_marca_posts"},
	}, nil); st != 403 {
		t.Fatalf("POST /api/convites sem alçada esperava 403, veio %d", st)
	}

	// Membro COM gerenciar_pessoas pode convidar na organização ativa.
	criarMembroNaOrg(t, api.banco, growth.ID, "Gestor", "gestor-b1@exemplo.com", "gerenciar_pessoas")
	gestor := novoCliente(t, api)
	entrarComEmail(t, gestor, "gestor-b1@exemplo.com")

	// Validações de entrada -> 400.
	for _, invalido := range []map[string]any{
		{"email": "invalido", "alcadas": []string{"ver_marca_posts"}},
		{"email": "valido@exemplo.com"},
		{"email": "valido@exemplo.com", "alcadas": []string{"alcada_que_nao_existe"}},
		{"email": "valido@exemplo.com", "alcadas": []string{}, "extra": true},
	} {
		if st := gestor.pedir("POST", "/api/convites", invalido, nil); st != 400 {
			t.Fatalf("payload inválido %+v esperava 400, veio %d", invalido, st)
		}
	}

	// Convidar quem já é membro (sem diferenciar maiúsculas) -> 409 ja_membro.
	if st, body := gestor.pedirComCorpo("POST", "/api/convites", map[string]any{
		"email":   "PAULO@exemplo.com",
		"alcadas": []string{"ver_marca_posts"},
	}); st != 409 || !strings.Contains(string(body), "ja_membro") {
		t.Fatalf("convidar membro existente esperava 409 ja_membro, veio %d (%s)", st, body)
	}

	// Convite válido: 201, e-mail em minúsculas, 1 envio com o link em PT-BR.
	var criado RespostaConviteCriado
	if st := gestor.pedir("POST", "/api/convites", map[string]any{
		"email":   "  Maria.Silva@Exemplo.COM ",
		"alcadas": []string{"editar_posts", "ver_marca_posts"},
	}, &criado); st != 201 {
		t.Fatalf("POST /api/convites esperava 201, veio %d", st)
	}
	if criado.Convite.Email != "maria.silva@exemplo.com" {
		t.Fatalf("e-mail não normalizado: %q", criado.Convite.Email)
	}
	if criado.Convite.EnvioEstado != "enviado" || criado.MensagemEnvio != nil {
		t.Fatalf("estado de envio inesperado: %+v (%v)", criado.Convite.EnvioEstado, criado.MensagemEnvio)
	}
	if !strings.HasPrefix(criado.Link, "https://app.lauragrowth.test/convite/") {
		t.Fatalf("link inesperado: %q", criado.Link)
	}
	if !slices.Equal(criado.Convite.Alcadas, []string{"ver_marca_posts", "editar_posts"}) {
		t.Fatalf("alçadas inesperadas: %+v", criado.Convite.Alcadas)
	}
	if fake.total() != 1 {
		t.Fatalf("esperava 1 e-mail enviado, veio %d", fake.total())
	}
	envio := fake.ultimo()
	if envio.Para != "maria.silva@exemplo.com" || !strings.Contains(envio.Texto, criado.Link) {
		t.Fatalf("e-mail enviado não contém destinatário ou link: %+v", envio)
	}

	// Sem gerenciar_pessoas não consegue reenviar nem revogar convite existente -> 403.
	if st := semAcesso.pedir("POST", "/api/convites/"+criado.Convite.ID+"/reenviar", nil, nil); st != 403 {
		t.Fatalf("reenviar sem alçada esperava 403, veio %d", st)
	}
	if st := semAcesso.pedir("DELETE", "/api/convites/"+criado.Convite.ID, nil, nil); st != 403 {
		t.Fatalf("revogar sem alçada esperava 403, veio %d", st)
	}

	// Reconvidar o mesmo e-mail pendente substitui o convite anterior e invalida o token antigo.
	tokenAntigo := extrairTokenDeLink(t, criado.Link)
	var substituido RespostaConviteCriado
	if st := gestor.pedir("POST", "/api/convites", map[string]any{
		"email":   "maria.silva@exemplo.com",
		"alcadas": []string{"aprovar_posts"},
	}, &substituido); st != 201 {
		t.Fatalf("reconvidar e-mail pendente esperava 201, veio %d", st)
	}
	if substituido.Convite.ID == criado.Convite.ID {
		t.Fatal("esperava novo ID de convite ao substituir")
	}
	if st := novoCliente(t, api).pedir("GET", "/api/convites/"+tokenAntigo, nil, nil); st != 404 {
		t.Fatalf("token antigo após substituição esperava 404, veio %d", st)
	}
	var pessoas RespPessoasParaTeste
	if st := gestor.pedir("GET", "/api/pessoas", nil, &pessoas); st != 200 || len(pessoas.Convites) != 1 || pessoas.Convites[0].ID != substituido.Convite.ID {
		t.Fatalf("esperava apenas 1 convite pendente após substituição: %d %+v", st, pessoas.Convites)
	}

	// Isolamento entre organizações (AC18): convite na Laura Growth dá 404 quando acessado a partir da Laura Hub.
	if st := paulo.pedir("PUT", "/api/sessao/organizacao", map[string]string{"organizacaoId": hub.ID}, nil); st != 200 {
		t.Fatal(st)
	}
	if st := paulo.pedir("POST", "/api/convites/"+substituido.Convite.ID+"/reenviar", nil, nil); st != 404 {
		t.Fatalf("reenviar convite de outra org esperava 404, veio %d", st)
	}
	if st := paulo.pedir("DELETE", "/api/convites/"+substituido.Convite.ID, nil, nil); st != 404 {
		t.Fatalf("revogar convite de outra org esperava 404, veio %d", st)
	}

	// Revogar na org certa -> 204 e invalida token.
	if st := gestor.pedir("DELETE", "/api/convites/"+substituido.Convite.ID, nil, nil); st != 204 {
		t.Fatalf("DELETE /api/convites/{id} esperava 204, veio %d", st)
	}
	tokenNovo := extrairTokenDeLink(t, substituido.Link)
	if st := novoCliente(t, api).pedir("GET", "/api/convites/"+tokenNovo, nil, nil); st != 404 {
		t.Fatalf("token revogado esperava 404, veio %d", st)
	}
	if st := gestor.pedir("DELETE", "/api/convites/"+substituido.Convite.ID, nil, nil); st != 404 {
		t.Fatalf("revogar convite já revogado esperava 404, veio %d", st)
	}
}

type RespPessoasParaTeste = RespostaPessoas

func TestConvitesB2AceiteTokenUnicoEExpiracao(t *testing.T) {
	api, _ := novaAPIConvitesTeste(t)
	agoraFixo := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	api.agora = func() time.Time { return agoraFixo }

	paulo := novoCliente(t, api)
	sessPaulo := entrarComoPaulo(t, paulo)
	growth, hub := sessPaulo.Organizacoes[0], sessPaulo.Organizacoes[1]

	// 1. Convite para conta NOVA.
	var convNova RespostaConviteCriado
	if st := paulo.pedir("POST", "/api/convites", map[string]any{
		"email":   "lucia.nova@exemplo.com",
		"alcadas": []string{"ver_marca_posts", "editar_posts"},
	}, &convNova); st != 201 {
		t.Fatalf("criar convite conta nova falhou: %d", st)
	}
	tokenNova := extrairTokenDeLink(t, convNova.Link)
	rawToken, err := base64.RawURLEncoding.DecodeString(tokenNova)
	if err != nil || len(rawToken) < 32 {
		t.Fatalf("token precisa ter >= 32 bytes aleatórios, veio len=%d err=%v", len(rawToken), err)
	}

	// Token fica guardado apenas como hash no banco (nunca em texto puro).
	var hashBanco []byte
	if err := api.banco.QueryRow(context.Background(), `SELECT token_hash FROM convites WHERE id = $1::uuid`, convNova.Convite.ID).Scan(&hashBanco); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(hashBanco, rawToken) || bytes.Equal(hashBanco, []byte(tokenNova)) || len(hashBanco) != 32 {
		t.Fatalf("token_hash inválido ou armazenado em claro: len=%d", len(hashBanco))
	}

	// GET público devolve apenas organização, e-mail mascarado e contaExistente=false.
	publico := novoCliente(t, api)
	var info ConvitePublico
	if st := publico.pedir("GET", "/api/convites/"+tokenNova, nil, &info); st != 200 {
		t.Fatalf("GET /api/convites/{token} esperava 200, veio %d", st)
	}
	if info.OrganizacaoNome != "Laura Growth" || info.ContaExistente || strings.Contains(info.EmailMascarado, "lucia.nova") {
		t.Fatalf("ConvitePublico inesperado: %+v", info)
	}
	if !strings.HasSuffix(info.EmailMascarado, "@exemplo.com") || !strings.Contains(info.EmailMascarado, "***") {
		t.Fatalf("máscara de e-mail inesperada: %q", info.EmailMascarado)
	}

	// Token malformado ou inexistente -> 404 convite_invalido.
	if st, body := publico.pedirComCorpo("GET", "/api/convites/token-curto", nil); st != 404 || !strings.Contains(string(body), "convite_invalido") {
		t.Fatalf("token inválido esperava 404 convite_invalido, veio %d (%s)", st, body)
	}

	// Aceite sem nome ou senha fraca -> 400 dados_invalidos, sem consumir convite.
	if st := publico.pedir("POST", "/api/convites/"+tokenNova+"/aceitar", map[string]any{"nome": "", "senha": "senha-forte-123"}, nil); st != 400 {
		t.Fatalf("nome vazio esperava 400, veio %d", st)
	}
	if st := publico.pedir("POST", "/api/convites/"+tokenNova+"/aceitar", map[string]any{"nome": "Lúcia", "senha": "curta"}, nil); st != 400 {
		t.Fatalf("senha curta esperava 400, veio %d", st)
	}

	// Aceite válido cria conta, inicia sessão na organização do convite com exatamente as alçadas escolhidas.
	var sessLucia Sessao
	if st := publico.pedir("POST", "/api/convites/"+tokenNova+"/aceitar", map[string]any{
		"nome":  "Lúcia Nova",
		"senha": "senha-segura-lucia-123",
	}, &sessLucia); st != 200 {
		t.Fatalf("aceitar convite conta nova esperava 200, veio %d", st)
	}
	if sessLucia.OrganizacaoAtiva.ID != growth.ID || !slices.Equal(sessLucia.Alcadas, []string{"ver_marca_posts", "editar_posts"}) {
		t.Fatalf("sessão após aceite inesperada: %+v", sessLucia)
	}
	// Verifica que entrou como membro e com acesso imediato conforme alçadas.
	if st := publico.pedir("GET", "/api/kit", nil, nil); st != 200 {
		t.Fatalf("GET /api/kit com ver_marca_posts esperava 200, veio %d", st)
	}
	if st := publico.pedir("GET", "/api/pessoas", nil, nil); st != 403 {
		t.Fatalf("GET /api/pessoas sem gerenciar_pessoas esperava 403, veio %d", st)
	}
	var papelLucia string
	if err := api.banco.QueryRow(context.Background(), `SELECT papel FROM membros WHERE organizacao_id = $1::uuid AND usuario_id = $2::uuid`, growth.ID, sessLucia.Usuario.ID).Scan(&papelLucia); err != nil || papelLucia != "membro" {
		t.Fatalf("papel esperado 'membro', veio %q (%v)", papelLucia, err)
	}

	// Uso único: tentar ler ou aceitar novamente -> 404 convite_invalido.
	if st := novoCliente(t, api).pedir("GET", "/api/convites/"+tokenNova, nil, nil); st != 404 {
		t.Fatalf("reuso GET esperava 404, veio %d", st)
	}
	if st := novoCliente(t, api).pedir("POST", "/api/convites/"+tokenNova+"/aceitar", map[string]any{"nome": "X", "senha": "senha-segura-lucia-123"}, nil); st != 404 {
		t.Fatalf("reuso POST aceitar esperava 404, veio %d", st)
	}

	// 2. Convite para conta EXISTENTE (Lúcia convidada agora para Laura Hub).
	if st := paulo.pedir("PUT", "/api/sessao/organizacao", map[string]string{"organizacaoId": hub.ID}, nil); st != 200 {
		t.Fatal(st)
	}
	var convExistente RespostaConviteCriado
	if st := paulo.pedir("POST", "/api/convites", map[string]any{
		"email":   "lucia.nova@exemplo.com",
		"alcadas": []string{"aprovar_plano"},
	}, &convExistente); st != 201 {
		t.Fatalf("criar convite conta existente falhou: %d", st)
	}
	tokenExistente := extrairTokenDeLink(t, convExistente.Link)

	anon := novoCliente(t, api)
	if st := anon.pedir("GET", "/api/convites/"+tokenExistente, nil, &info); st != 200 || !info.ContaExistente {
		t.Fatalf("esperava contaExistente=true, veio %d %+v", st, info)
	}
	// Sem sessão -> 401 login_necessario.
	if st, body := anon.pedirComCorpo("POST", "/api/convites/"+tokenExistente+"/aceitar", map[string]any{}); st != 401 || !strings.Contains(string(body), "login_necessario") {
		t.Fatalf("sem sessão esperava 401 login_necessario, veio %d (%s)", st, body)
	}
	// Sessão de outro e-mail -> 403 email_diferente.
	if st, body := paulo.pedirComCorpo("POST", "/api/convites/"+tokenExistente+"/aceitar", map[string]any{}); st != 403 || !strings.Contains(string(body), "email_diferente") {
		t.Fatalf("sessão de outro e-mail esperava 403 email_diferente, veio %d (%s)", st, body)
	}
	// Sessão da própria Lúcia -> 200 e organização ativa passa a ser Laura Hub.
	if st := publico.pedir("POST", "/api/convites/"+tokenExistente+"/aceitar", map[string]any{}, &sessLucia); st != 200 {
		t.Fatalf("aceite conta existente esperava 200, veio %d", st)
	}
	if sessLucia.OrganizacaoAtiva.ID != hub.ID || !slices.Equal(sessLucia.Alcadas, []string{"aprovar_plano"}) {
		t.Fatalf("sessão após aceite em Laura Hub inesperada: %+v", sessLucia)
	}
	// Uso único novamente.
	if st := publico.pedir("POST", "/api/convites/"+tokenExistente+"/aceitar", map[string]any{}, nil); st != 404 {
		t.Fatalf("reuso conta existente esperava 404, veio %d", st)
	}

	// 3. Expiração em 7 dias.
	var convExpira RespostaConviteCriado
	if st := paulo.pedir("POST", "/api/convites", map[string]any{
		"email":   "expira@exemplo.com",
		"alcadas": []string{"ver_marca_posts"},
	}, &convExpira); st != 201 {
		t.Fatal(st)
	}
	tokenExpira := extrairTokenDeLink(t, convExpira.Link)
	agoraFixo = agoraFixo.Add(7*24*time.Hour + time.Minute)
	if st := anon.pedir("GET", "/api/convites/"+tokenExpira, nil, nil); st != 404 {
		t.Fatalf("convite expirado GET esperava 404, veio %d", st)
	}
	if st := anon.pedir("POST", "/api/convites/"+tokenExpira+"/aceitar", map[string]any{"nome": "Expirado", "senha": "senha-segura-123"}, nil); st != 404 {
		t.Fatalf("convite expirado POST aceitar esperava 404, veio %d", st)
	}

	// 4. Auditoria em registro_acessos (convite_criado e convite_aceito presentes na Laura Hub).
	var reg RespostaRegistroAcessos
	if st := paulo.pedir("GET", "/api/pessoas/registro", nil, &reg); st != 200 {
		t.Fatal(st)
	}
	var acoes []string
	for _, r := range reg.Registros {
		acoes = append(acoes, r.Acao)
	}
	if !slices.Contains(acoes, "convite_criado") || !slices.Contains(acoes, "convite_aceito") {
		t.Fatalf("auditoria da Laura Hub incompleta: %+v", acoes)
	}
}

func TestConvitesB9AdaptadorIndisponivelEReenviar(t *testing.T) {
	// 1. Adaptador não configurado (sem chave).
	apiNaoConfig, _ := novaAPIConvitesTeste(t)
	apiNaoConfig.email = novoEnviadorEmailZoho("", "")
	pauloNC := novoCliente(t, apiNaoConfig)
	entrarComoPaulo(t, pauloNC)

	var respNC RespostaConviteCriado
	if st := pauloNC.pedir("POST", "/api/convites", map[string]any{
		"email":   "sem-zoho@exemplo.com",
		"alcadas": []string{"ver_marca_posts"},
	}, &respNC); st != 201 {
		t.Fatalf("esperava 201 mesmo sem adaptador configurado, veio %d", st)
	}
	if respNC.Convite.EnvioEstado != "falhou" || respNC.MensagemEnvio == nil || *respNC.MensagemEnvio == "" || respNC.Link == "" {
		t.Fatalf("esperava envioEstado=falhou + mensagemEnvio + link, veio %+v", respNC)
	}

	// 2. Adaptador Zoho com chave real configurada, mas fora do ar (500): chave nunca vaza em resposta nem em log.
	const chaveSecreta = "zoho-segredo-b9-nao-vazar-999"
	var bufLogs bytes.Buffer
	loggerAntigo := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&bufLogs, nil)))
	defer slog.SetDefault(loggerAntigo)

	zohoFalhando := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("erro com segredo " + chaveSecreta))
	}))
	defer zohoFalhando.Close()

	api, _ := novaAPIConvitesTeste(t)
	api.email = novoEnviadorEmailZoho(zohoFalhando.URL, chaveSecreta)
	paulo := novoCliente(t, api)
	entrarComoPaulo(t, paulo)

	st, corpo := paulo.pedirComCorpo("POST", "/api/convites", map[string]any{
		"email":   "b9@exemplo.com",
		"alcadas": []string{"ver_marca_posts"},
	})
	if st != 201 {
		t.Fatalf("esperava 201 com adaptador fora do ar, veio %d (%s)", st, corpo)
	}
	if strings.Contains(string(corpo), chaveSecreta) {
		t.Fatalf("resposta HTTP vazou chave Zoho: %s", corpo)
	}
	if strings.Contains(bufLogs.String(), chaveSecreta) {
		t.Fatalf("logs vazaram chave Zoho: %s", bufLogs.String())
	}

	var criado RespostaConviteCriado
	if err := json.Unmarshal(corpo, &criado); err != nil {
		t.Fatal(err)
	}
	if criado.Convite.EnvioEstado != "falhou" || criado.MensagemEnvio == nil {
		t.Fatalf("esperava falhou + mensagemEnvio, veio %+v", criado)
	}
	tokenAntigo := extrairTokenDeLink(t, criado.Link)

	// GET /api/pessoas mostra o convite como falhou e nunca inclui o link.
	stPessoas, corpoPessoas := paulo.pedirComCorpo("GET", "/api/pessoas", nil)
	if stPessoas != 200 || strings.Contains(string(corpoPessoas), tokenAntigo) || strings.Contains(string(corpoPessoas), `"link"`) {
		t.Fatalf("GET /api/pessoas não deve expor link/token: %d (%s)", stPessoas, corpoPessoas)
	}

	// 3. Reenviar após recuperação do adaptador: estado vira enviado, novo token emitido, antigo invalidado.
	fakeRecuperado := &adaptadorEmailFake{}
	api.email = fakeRecuperado
	var reenviado RespostaConviteCriado
	if st := paulo.pedir("POST", "/api/convites/"+criado.Convite.ID+"/reenviar", nil, &reenviado); st != 200 {
		t.Fatalf("reenviar esperava 200, veio %d", st)
	}
	if reenviado.Convite.EnvioEstado != "enviado" || reenviado.MensagemEnvio != nil {
		t.Fatalf("esperava enviado após reenvio, veio %+v", reenviado)
	}
	tokenNovo := extrairTokenDeLink(t, reenviado.Link)
	if tokenNovo == tokenAntigo {
		t.Fatal("reenvio precisa emitir um novo token")
	}
	if fakeRecuperado.total() != 1 || !strings.Contains(fakeRecuperado.ultimo().Texto, reenviado.Link) {
		t.Fatalf("e-mail de reenvio não continha novo link: %+v", fakeRecuperado.ultimo())
	}

	anon := novoCliente(t, api)
	if st := anon.pedir("GET", "/api/convites/"+tokenAntigo, nil, nil); st != 404 {
		t.Fatalf("token antigo após reenvio esperava 404, veio %d", st)
	}
	if st := anon.pedir("GET", "/api/convites/"+tokenNovo, nil, nil); st != 200 {
		t.Fatalf("novo token após reenvio esperava 200, veio %d", st)
	}
}

func TestConvitesB10LimitesAbuso(t *testing.T) {
	api, fake := novaAPIConvitesTeste(t)
	agoraFixo := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	api.agora = func() time.Time { return agoraFixo }

	paulo := novoCliente(t, api)
	sessPaulo := entrarComoPaulo(t, paulo)
	hub := sessPaulo.Organizacoes[1]

	// Alterna sucesso e falha no adaptador para provar que tentativas com falha também contam no limite por hora.
	var primeiroConviteID string
	for i := 0; i < 15; i++ {
		if i%2 == 1 {
			fake.definirErro(errors.New("falha temporária"))
		} else {
			fake.definirErro(nil)
		}
		var resp RespostaConviteCriado
		email := "lote1-" + string(rune('a'+i)) + "@exemplo.com"
		if st := paulo.pedir("POST", "/api/convites", map[string]any{
			"email":   email,
			"alcadas": []string{"ver_marca_posts"},
		}, &resp); st != 201 {
			t.Fatalf("convite %d falhou: %d", i, st)
		}
		if i == 0 {
			primeiroConviteID = resp.Convite.ID
		}
	}
	fake.definirErro(nil)
	// Mais 5 reenvios completam 20 tentativas de envio na mesma hora.
	for i := 0; i < 5; i++ {
		if st := paulo.pedir("POST", "/api/convites/"+primeiroConviteID+"/reenviar", nil, nil); st != 200 {
			t.Fatalf("reenvio %d falhou: %d", i, st)
		}
	}

	// 21ª tentativa (criar ou reenviar) -> 429 limite_envios_hora.
	if st, body := paulo.pedirComCorpo("POST", "/api/convites", map[string]any{
		"email":   "excedente-hora@exemplo.com",
		"alcadas": []string{"ver_marca_posts"},
	}); st != 429 || !strings.Contains(string(body), "limite_envios_hora") {
		t.Fatalf("21º envio esperava 429 limite_envios_hora, veio %d (%s)", st, body)
	}
	if st, body := paulo.pedirComCorpo("POST", "/api/convites/"+primeiroConviteID+"/reenviar", nil); st != 429 || !strings.Contains(string(body), "limite_envios_hora") {
		t.Fatalf("21º envio por reenvio esperava 429 limite_envios_hora, veio %d (%s)", st, body)
	}

	// Outra organização (Laura Hub) não é afetada pelo limite da Laura Growth.
	if st := paulo.pedir("PUT", "/api/sessao/organizacao", map[string]string{"organizacaoId": hub.ID}, nil); st != 200 {
		t.Fatal(st)
	}
	if st := paulo.pedir("POST", "/api/convites", map[string]any{
		"email":   "hub@exemplo.com",
		"alcadas": []string{"ver_marca_posts"},
	}, nil); st != 201 {
		t.Fatalf("outra org deveria continuar enviando normalmente, veio %d", st)
	}
	if st := paulo.pedir("PUT", "/api/sessao/organizacao", map[string]string{"organizacaoId": sessPaulo.Organizacoes[0].ID}, nil); st != 200 {
		t.Fatal(st)
	}

	// Avança > 1 hora para liberar a janela de envios e preencher os 30 pendentes (já há 15).
	agoraFixo = agoraFixo.Add(61 * time.Minute)
	var ultimoID string
	for i := 0; i < 15; i++ {
		var resp RespostaConviteCriado
		email := "lote2-" + string(rune('a'+i)) + "@exemplo.com"
		if st := paulo.pedir("POST", "/api/convites", map[string]any{
			"email":   email,
			"alcadas": []string{"ver_marca_posts"},
		}, &resp); st != 201 {
			t.Fatalf("convite lote2 %d falhou: %d", i, st)
		}
		ultimoID = resp.Convite.ID
	}

	// 31º convite pendente para novo e-mail -> 429 limite_convites_pendentes.
	if st, body := paulo.pedirComCorpo("POST", "/api/convites", map[string]any{
		"email":   "trigesimo-primeiro@exemplo.com",
		"alcadas": []string{"ver_marca_posts"},
	}); st != 429 || !strings.Contains(string(body), "limite_convites_pendentes") {
		t.Fatalf("31º convite pendente esperava 429 limite_convites_pendentes, veio %d (%s)", st, body)
	}

	// Reconvidar um e-mail que já está pendente substitui sem ultrapassar 30 -> 201.
	if st := paulo.pedir("POST", "/api/convites", map[string]any{
		"email":   "lote2-a@exemplo.com",
		"alcadas": []string{"editar_posts"},
	}, nil); st != 201 {
		t.Fatalf("substituir pendente com 30 pendentes esperava 201, veio %d", st)
	}

	// Revogar 1 convite abre vaga para um novo -> 201.
	if st := paulo.pedir("DELETE", "/api/convites/"+ultimoID, nil, nil); st != 204 {
		t.Fatal(st)
	}
	if st := paulo.pedir("POST", "/api/convites", map[string]any{
		"email":   "abriu-vaga@exemplo.com",
		"alcadas": []string{"ver_marca_posts"},
	}, nil); st != 201 {
		t.Fatalf("convidar após revogar 1 esperava 201, veio %d", st)
	}
}
