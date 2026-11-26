package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"
)

const (
	chaveCifragemTeste = "chave-de-cifragem-apenas-para-testes-123456"
	chaveAPITeste      = "sk-proj-segredo-falso-de-teste-abcd1234"
	chaveAPINovaTeste  = "sk-proj-segredo-falso-trocado-wxyz9876"
)

func novaAPIAdminTeste(t *testing.T) *API {
	t.Helper()
	t.Setenv("LG_CHAVE_CIFRAGEM", chaveCifragemTeste)
	cifrador, err := novoCifrador(chaveCifragemTeste)
	if err != nil {
		t.Fatal(err)
	}
	api := NovaAPI(prepararBanco(t), false, cifrador)
	api.resolverIP = func(_ context.Context, _ string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("104.18.7.192")}, nil
	}
	return api
}

func (c *cliente) pedirComCorpo(metodo, caminho string, corpo any) (int, []byte) {
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
	b, err := io.ReadAll(res.Body)
	if err != nil {
		c.t.Fatal(err)
	}
	return res.StatusCode, b
}

func criarUsuarioComum(t *testing.T, banco *Banco, email string) string {
	t.Helper()
	hash, err := gerarHashSenha(senhaTeste)
	if err != nil {
		t.Fatal(err)
	}
	var id string
	err = banco.QueryRow(context.Background(), `
		WITH u AS (
			INSERT INTO usuarios (nome, email, senha_hash) VALUES ('Comum', $1, $2)
			RETURNING id
		),
		o AS (SELECT id FROM organizacoes WHERE nome = 'Clínica Alheia' LIMIT 1),
		m AS (INSERT INTO membros (organizacao_id, usuario_id, papel) SELECT o.id, u.id, 'membro' FROM o, u)
		SELECT id::text FROM u`, email, hash).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestAC37AC30PermissaoAdminESemente(t *testing.T) {
	api := novaAPIAdminTeste(t)
	ctx := context.Background()

	// Migração e semente rodam uma segunda vez sem erro (idempotência).
	if err := migrar(ctx, api.banco); err != nil {
		t.Fatalf("segunda migração falhou: %v", err)
	}
	if err := semear(ctx, api.banco, "Paulo", "Paulo@Exemplo.com", senhaTeste); err != nil {
		t.Fatalf("segunda semente falhou: %v", err)
	}

	// Semente marca Laura Growth e Laura Hub como organizações da casa, e não a alheia.
	linhas, err := api.banco.Query(ctx, `SELECT nome, da_casa FROM organizacoes ORDER BY nome`)
	if err != nil {
		t.Fatal(err)
	}
	defer linhas.Close()
	daCasa := map[string]bool{}
	for linhas.Next() {
		var nome string
		var casa bool
		if err := linhas.Scan(&nome, &casa); err != nil {
			t.Fatal(err)
		}
		daCasa[nome] = casa
	}
	if !daCasa["Laura Growth"] || !daCasa["Laura Hub"] || daCasa["Clínica Alheia"] {
		t.Fatalf("da_casa inesperado nas organizações: %+v", daCasa)
	}

	// Sem sessão -> 401 nas rotas admin.
	anonimo := novoCliente(t, api)
	if st, _ := anonimo.pedirComCorpo("GET", "/api/admin/fornecedores", nil); st != 401 {
		t.Fatalf("sem sessão esperava 401, veio %d", st)
	}

	// Paulo tem adminPlataforma = true no POST e GET /api/sessao.
	paulo := novoCliente(t, api)
	var sessPaulo Sessao
	if st := paulo.pedir("POST", "/api/sessao", map[string]string{"email": "paulo@exemplo.com", "senha": senhaTeste}, &sessPaulo); st != 200 || !sessPaulo.Usuario.AdminPlataforma {
		t.Fatalf("login Paulo esperava adminPlataforma=true, veio %d %+v", st, sessPaulo.Usuario)
	}
	if st := paulo.pedir("GET", "/api/sessao", nil, &sessPaulo); st != 200 || !sessPaulo.Usuario.AdminPlataforma {
		t.Fatalf("GET /api/sessao Paulo esperava adminPlataforma=true, veio %d %+v", st, sessPaulo.Usuario)
	}

	// Cria um fornecedor para testar o PATCH depois com a usuária comum.
	var criado Fornecedor
	if st := paulo.pedir("POST", "/api/admin/fornecedores", map[string]any{
		"nome":            "Simulado Permissão",
		"tipo":            "simulado",
		"limiteMensalUsd": 10,
	}, &criado); st != 201 {
		t.Fatalf("criar fornecedor esperava 201, veio %d", st)
	}

	// Pessoa comum (membro sem marca admin_plataforma) recebe adminPlataforma=false e 403 em todas as rotas /api/admin/*.
	comumID := criarUsuarioComum(t, api.banco, "comum@exemplo.com")
	comum := novoCliente(t, api)
	var sessComum Sessao
	if st := comum.pedir("POST", "/api/sessao", map[string]string{"email": "comum@exemplo.com", "senha": senhaTeste}, &sessComum); st != 200 || sessComum.Usuario.AdminPlataforma {
		t.Fatalf("login comum esperava adminPlataforma=false, veio %d %+v", st, sessComum.Usuario)
	}
	if st := comum.pedir("GET", "/api/sessao", nil, &sessComum); st != 200 || sessComum.Usuario.AdminPlataforma {
		t.Fatalf("GET /api/sessao comum esperava adminPlataforma=false, veio %d %+v", st, sessComum.Usuario)
	}

	var bufLog bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&bufLog, nil)))
	defer slog.SetDefault(anterior)

	rotas := []struct {
		metodo  string
		caminho string
		corpo   any
	}{
		{"GET", "/api/admin/fornecedores", nil},
		{"POST", "/api/admin/fornecedores", map[string]any{"nome": "X", "tipo": "simulado", "limiteMensalUsd": 5}},
		{"PATCH", "/api/admin/fornecedores/" + criado.ID, map[string]any{"nome": "Y"}},
		{"DELETE", "/api/admin/fornecedores", nil},
		{"GET", "/api/admin/rota-que-nao-existe", nil},
		{"POST", "/api/admin", nil},
	}
	for _, rt := range rotas {
		st, body := comum.pedirComCorpo(rt.metodo, rt.caminho, rt.corpo)
		if st != 403 {
			t.Fatalf("%s %s para pessoa comum esperava 403, veio %d (%s)", rt.metodo, rt.caminho, st, body)
		}
		var errResp map[string]string
		if err := json.Unmarshal(body, &errResp); err != nil || errResp["codigo"] != "sem_permissao" {
			t.Fatalf("%s %s esperava codigo=sem_permissao, veio %s", rt.metodo, rt.caminho, body)
		}
	}
	// Para administrador, rota desconhecida dá 404 e método não previsto dá 405.
	if st, _ := paulo.pedirComCorpo("GET", "/api/admin/rota-que-nao-existe", nil); st != 404 {
		t.Fatalf("rota desconhecida para admin esperava 404, veio %d", st)
	}
	if st, _ := paulo.pedirComCorpo("DELETE", "/api/admin/fornecedores", nil); st != 405 {
		t.Fatalf("método não previsto para admin esperava 405, veio %d", st)
	}
	logs := bufLog.String()
	if !strings.Contains(logs, comumID) || !strings.Contains(logs, "/api/admin/fornecedores") {
		t.Fatalf("log de recusa deveria conter usuarioId e rota, veio: %s", logs)
	}

	// Se a marca admin_plataforma for removida no banco, a sessão já aberta passa a ser recusada na hora.
	if _, err := api.banco.Exec(ctx, `UPDATE usuarios SET admin_plataforma = false WHERE id::text = $1`, sessPaulo.Usuario.ID); err != nil {
		t.Fatal(err)
	}
	if st, _ := paulo.pedirComCorpo("GET", "/api/admin/fornecedores", nil); st != 403 {
		t.Fatalf("após remover admin_plataforma esperava 403, veio %d", st)
	}
}

func TestAC32CifragemESigiloDeChave(t *testing.T) {
	// Validação de subida do servidor sem LG_CHAVE_CIFRAGEM ou com chave curta.
	for _, segredo := range []string{"", "   ", "curta-demais-123"} {
		t.Setenv("LG_CHAVE_CIFRAGEM", segredo)
		if _, err := novoCifrador(segredo); err == nil {
			t.Fatalf("novoCifrador(%q) deveria falhar", segredo)
		}
		err := servir(context.Background(), nil)
		if err == nil {
			t.Fatalf("servir deveria recusar subir com LG_CHAVE_CIFRAGEM=%q", segredo)
		}
		if segredo != "" && strings.TrimSpace(segredo) != "" && strings.Contains(err.Error(), segredo) {
			t.Fatalf("erro de servir não pode vazar o segredo: %v", err)
		}
	}

	// Cifrar/decifrar ida e volta, AAD e integridade.
	cif, err := novoCifrador(chaveCifragemTeste)
	if err != nil {
		t.Fatal(err)
	}
	cifrado, err := cif.cifrar("forn-1", chaveAPITeste)
	if err != nil {
		t.Fatal(err)
	}
	if cifrado[0] != 0x01 {
		t.Fatalf("versão esperada 0x01, veio %x", cifrado[0])
	}
	if bytes.Contains(cifrado, []byte(chaveAPITeste)) {
		t.Fatal("texto cifrado contém a chave em claro")
	}
	decifrado, err := cif.decifrar("forn-1", cifrado)
	if err != nil || decifrado != chaveAPITeste {
		t.Fatalf("ida e volta falhou: err=%v decifrado=%q", err, decifrado)
	}
	if _, err := cif.decifrar("forn-2", cifrado); err == nil {
		t.Fatal("decifrar com AAD de outro fornecedor deveria falhar")
	}
	adulterado := append([]byte(nil), cifrado...)
	adulterado[len(adulterado)-1] ^= 0xff
	if _, err := cif.decifrar("forn-1", adulterado); err == nil {
		t.Fatal("decifrar com byte alterado deveria falhar")
	}

	// Fluxo completo na API capturando logs e conferindo respostas + banco.
	api := novaAPIAdminTeste(t)
	var bufLog bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&bufLog, nil)))
	defer slog.SetDefault(anterior)

	c := novoCliente(t, api)
	entrarComoPaulo(t, c)

	st, bodyPost := c.pedirComCorpo("POST", "/api/admin/fornecedores", map[string]any{
		"nome":            "OpenAI Principal",
		"tipo":            "openai",
		"chave":           chaveAPITeste,
		"limiteMensalUsd": 100,
	})
	if st != 201 {
		t.Fatalf("POST fornecedor esperava 201, veio %d (%s)", st, bodyPost)
	}
	if bytes.Contains(bodyPost, []byte(chaveAPITeste)) {
		t.Fatalf("resposta POST vazou a chave: %s", bodyPost)
	}
	var criado Fornecedor
	if err := json.Unmarshal(bodyPost, &criado); err != nil {
		t.Fatal(err)
	}
	if criado.ChaveFinal == nil || *criado.ChaveFinal != "1234" {
		t.Fatalf("chaveFinal inesperada: %v", criado.ChaveFinal)
	}

	st, bodyGet := c.pedirComCorpo("GET", "/api/admin/fornecedores", nil)
	if st != 200 || bytes.Contains(bodyGet, []byte(chaveAPITeste)) {
		t.Fatalf("GET fornecedores falhou ou vazou chave: %d %s", st, bodyGet)
	}

	// Banco guarda cifrado, não em texto claro, e decifra com AAD = id.
	var noBanco []byte
	if err := api.banco.QueryRow(context.Background(), `SELECT chave_cifrada FROM ia_fornecedores WHERE id::text = $1`, criado.ID).Scan(&noBanco); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(noBanco, []byte(chaveAPITeste)) {
		t.Fatal("coluna chave_cifrada no banco contém a chave em claro")
	}
	decBanco, err := cif.decifrar(criado.ID, noBanco)
	if err != nil || decBanco != chaveAPITeste {
		t.Fatalf("decifrar coluna do banco falhou: err=%v", err)
	}

	// PATCH trocando a chave recifra com o mesmo AAD e nunca vaza.
	st, bodyPatch := c.pedirComCorpo("PATCH", "/api/admin/fornecedores/"+criado.ID, map[string]any{
		"chave": chaveAPINovaTeste,
	})
	if st != 200 || bytes.Contains(bodyPatch, []byte(chaveAPINovaTeste)) || bytes.Contains(bodyPatch, []byte(chaveAPITeste)) {
		t.Fatalf("PATCH fornecedor falhou ou vazou chave: %d %s", st, bodyPatch)
	}
	var atualizado Fornecedor
	if err := json.Unmarshal(bodyPatch, &atualizado); err != nil {
		t.Fatal(err)
	}
	if atualizado.ChaveFinal == nil || *atualizado.ChaveFinal != "9876" {
		t.Fatalf("chaveFinal após PATCH inesperada: %v", atualizado.ChaveFinal)
	}
	if err := api.banco.QueryRow(context.Background(), `SELECT chave_cifrada FROM ia_fornecedores WHERE id::text = $1`, criado.ID).Scan(&noBanco); err != nil {
		t.Fatal(err)
	}
	if decNova, err := cif.decifrar(criado.ID, noBanco); err != nil || decNova != chaveAPINovaTeste {
		t.Fatalf("decifrar chave trocada falhou: err=%v", err)
	}

	logs := bufLog.String()
	if strings.Contains(logs, chaveAPITeste) || strings.Contains(logs, chaveAPINovaTeste) || strings.Contains(logs, chaveCifragemTeste) {
		t.Fatalf("logs contêm segredo: %s", logs)
	}
}

func TestAC39ProtecaoDeEnderecoEClienteSeguro(t *testing.T) {
	ctx := context.Background()
	resPublico := func(_ context.Context, _ string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("104.18.7.192")}, nil
	}

	recusados := []string{
		"http://api.openai.com/v1",
		"https://127.0.0.1",
		"https://10.0.0.5",
		"https://172.16.0.1",
		"https://192.168.0.1",
		"https://169.254.169.254",
		"https://100.64.0.1",
		"https://0.0.0.1",
		"https://[::1]",
		"https://[fc00::1]",
		"https://[fe80::1]",
		"https://[::ffff:127.0.0.1]",
		"https://[64:ff9b::127.0.0.1]",
		"https://[2002:7f00:0001::1]",
		"https://[2001:0000::1]",
		"https://[::127.0.0.1]",
		"https://localhost",
		"https://metadata.google.internal",
		"https://usuario:senha@api.openai.com/v1",
		"https://api.openai.com/v1#frag",
	}
	for _, u := range recusados {
		if norm, err := validarEnderecoCom(ctx, u, resPublico, false); err == nil {
			t.Fatalf("esperava recusa para %q, mas aceitou %q", u, norm)
		}
	}

	// DNS resolvendo para IP interno também é recusado.
	resPrivado := func(_ context.Context, _ string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("104.18.7.192"), netip.MustParseAddr("10.0.0.5")}, nil
	}
	if norm, err := validarEnderecoCom(ctx, "https://api.openai.com/v1", resPrivado, false); err == nil {
		t.Fatalf("DNS com IP privado deveria ser recusado, aceitou %q", norm)
	}

	// Endereço público válido é aceito e remove barra final.
	norm, err := validarEnderecoCom(ctx, "https://api.openai.com/v1/", resPublico, false)
	if err != nil || norm != "https://api.openai.com/v1" {
		t.Fatalf("esperava https://api.openai.com/v1, veio norm=%q err=%v", norm, err)
	}

	// API recusa cadastro com endereço interno (400 endereco_invalido).
	api := novaAPIAdminTeste(t)
	c := novoCliente(t, api)
	entrarComoPaulo(t, c)
	st, body := c.pedirComCorpo("POST", "/api/admin/fornecedores", map[string]any{
		"nome":            "Compatível Interno",
		"tipo":            "compativel_openai",
		"endereco":        "https://169.254.169.254/latest",
		"chave":           chaveAPITeste,
		"limiteMensalUsd": 20,
	})
	if st != 400 {
		t.Fatalf("endereço interno na API esperava 400, veio %d (%s)", st, body)
	}
	var errResp map[string]string
	_ = json.Unmarshal(body, &errResp)
	if errResp["codigo"] != "endereco_invalido" {
		t.Fatalf("codigo esperado endereco_invalido, veio %s", body)
	}

	// Cliente seguro recusa conectar em servidor 127.0.0.1 e recusa redirecionamentos.
	tlsSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://api.openai.com/v1", http.StatusFound)
	}))
	defer tlsSrv.Close()

	clienteSeg := novoClienteSeguro(2 * time.Second)
	tr, ok := clienteSeg.Transport.(*http.Transport)
	if !ok || tr.TLSClientConfig == nil || tr.TLSClientConfig.MinVersion < tls.VersionTLS12 {
		t.Fatalf("TLS mínimo esperado >= 1.2, veio %+v", tr)
	}
	// Faz o cliente confiar no certificado do servidor de teste para provar que a recusa vem do bloqueio de 127.0.0.1.
	tr.TLSClientConfig = tlsSrv.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	tr.TLSClientConfig.MinVersion = tls.VersionTLS12
	if _, err := clienteSeg.Get(tlsSrv.URL); !errors.Is(err, errRedeInternaRecusada) {
		t.Fatalf("novoClienteSeguro deveria recusar 127.0.0.1 por rede interna, veio: %v", err)
	}
	reqFake, _ := http.NewRequest("GET", "https://api.openai.com/v1", nil)
	if err := clienteSeg.CheckRedirect(reqFake, []*http.Request{reqFake}); !errors.Is(err, errRedirecionamentoRecusado) {
		t.Fatal("novoClienteSeguro deveria recusar redirecionamento")
	}

	// DNS rebinding através do cliente de produção (novoClienteSeguro):
	// resolvedor devolve 127.0.0.1 só na hora da conexão; erro deve ser errRedeInternaRecusada.
	dnsAddr := iniciarDNSRebindingTeste(t)
	srvRebind, poolRebind := iniciarTLSComDominioTeste(t, "rebind.exemplo")
	defer srvRebind.Close()
	_, porta, _ := net.SplitHostPort(strings.TrimPrefix(srvRebind.URL, "https://"))

	anteriorResolver := resolvedorDial
	resolvedorDial = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "udp", dnsAddr)
		},
	}
	defer func() { resolvedorDial = anteriorResolver }()

	clienteProd := novoClienteSeguro(2 * time.Second)
	trProd := clienteProd.Transport.(*http.Transport)
	trProd.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: poolRebind}

	if _, err := clienteProd.Get("https://rebind.exemplo:" + porta); !errors.Is(err, errRedeInternaRecusada) {
		t.Fatalf("DNS rebinding deveria falhar com errRedeInternaRecusada, veio: %v", err)
	}
}

func iniciarDNSRebindingTeste(t *testing.T) string {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pc.Close() })
	go func() {
		buf := make([]byte, 512)
		for {
			n, addr, err := pc.ReadFrom(buf)
			if err != nil {
				return
			}
			if n < 17 {
				continue
			}
			// Encontra o fim do QNAME na pergunta.
			i := 12
			for i < n && buf[i] != 0 {
				i += int(buf[i]) + 1
			}
			if i+5 > n {
				continue
			}
			qend := i + 5
			qtype := uint16(buf[i+1])<<8 | uint16(buf[i+2])
			resp := make([]byte, 0, qend+16)
			resp = append(resp, buf[0], buf[1], 0x81, 0x80, 0, 1, 0, 0, 0, 0, 0, 0)
			resp = append(resp, buf[12:qend]...)
			if qtype == 1 { // A -> 127.0.0.1
				resp[7] = 1
				resp = append(resp, 0xc0, 0x0c, 0, 1, 0, 1, 0, 0, 0, 1, 0, 4, 127, 0, 0, 1)
			}
			_, _ = pc.WriteTo(resp, addr)
		}
	}()
	return pc.LocalAddr().String()
}

func iniciarTLSComDominioTeste(t *testing.T, dominio string) (*httptest.Server, *x509.CertPool) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: dominio},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:              []string{dominio},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatal(err)
	}
	certParsed, _ := x509.ParseCertificate(der)
	pool := x509.NewCertPool()
	pool.AddCert(certParsed)

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	srv.TLS = &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: priv}},
	}
	srv.StartTLS()
	return srv, pool
}

func TestFornecedoresRegrasLimitesEFuso(t *testing.T) {
	api := novaAPIAdminTeste(t)
	agoraFixo := time.Date(2026, 11, 25, 14, 0, 0, 0, fusoSaoPaulo)
	api.agora = func() time.Time { return agoraFixo }

	c := novoCliente(t, api)
	entrarComoPaulo(t, c)

	// 1. Simulado sem chave e sem endereço.
	var sim Fornecedor
	if st := c.pedir("POST", "/api/admin/fornecedores", map[string]any{
		"nome":            "Simulado Local",
		"tipo":            "simulado",
		"limiteMensalUsd": 10.00,
	}, &sim); st != 201 || !sim.Simulado || sim.ChaveFinal != nil || sim.Endereco != "" || sim.Situacao != "normal" {
		t.Fatalf("cadastro simulado inesperado: %d %+v", st, sim)
	}

	// 2. OpenAI e Anthropic com endereço vazio preenchem padrão.
	var oai, ant Fornecedor
	if st := c.pedir("POST", "/api/admin/fornecedores", map[string]any{
		"nome":            "OpenAI Padrão",
		"tipo":            "openai",
		"chave":           chaveAPITeste,
		"limiteMensalUsd": 50.00,
	}, &oai); st != 201 || oai.Endereco != "https://api.openai.com/v1" || oai.Simulado {
		t.Fatalf("OpenAI padrão inesperado: %d %+v", st, oai)
	}
	if st := c.pedir("POST", "/api/admin/fornecedores", map[string]any{
		"nome":            "Anthropic Padrão",
		"tipo":            "anthropic",
		"chave":           chaveAPITeste,
		"usoPermitido":    "casa",
		"limiteMensalUsd": 100.00,
		"avisoPercentual": 80,
	}, &ant); st != 201 || ant.Endereco != "https://api.anthropic.com/v1" || ant.UsoPermitido != "casa" {
		t.Fatalf("Anthropic padrão inesperado: %d %+v", st, ant)
	}

	// 3. Compatível exige endereço; não simulado exige chave; nome repetido dá 409.
	st, b := c.pedirComCorpo("POST", "/api/admin/fornecedores", map[string]any{
		"nome":            "Compatível Sem URL",
		"tipo":            "compativel_openai",
		"chave":           chaveAPITeste,
		"limiteMensalUsd": 25,
	})
	if st != 400 || !strings.Contains(string(b), "endereco_invalido") {
		t.Fatalf("compatível sem endereço esperava 400 endereco_invalido, veio %d (%s)", st, b)
	}
	st, b = c.pedirComCorpo("POST", "/api/admin/fornecedores", map[string]any{
		"nome":            "OpenAI Sem Chave",
		"tipo":            "openai",
		"limiteMensalUsd": 25,
	})
	if st != 400 || !strings.Contains(string(b), "chave_obrigatoria") {
		t.Fatalf("sem chave esperava 400 chave_obrigatoria, veio %d (%s)", st, b)
	}
	st, b = c.pedirComCorpo("POST", "/api/admin/fornecedores", map[string]any{
		"nome":            "openai padrão",
		"tipo":            "simulado",
		"limiteMensalUsd": 25,
	})
	if st != 409 || !strings.Contains(string(b), "nome_repetido") {
		t.Fatalf("nome repetido esperava 409 nome_repetido, veio %d (%s)", st, b)
	}

	// 4. Gasto do mês em America/Sao_Paulo e cálculo exato de situação (normal -> aviso -> limite).
	ctx := context.Background()
	var modID string
	if err := api.banco.QueryRow(ctx, `
		INSERT INTO ia_modelos (fornecedor_id, identificador, servico, origem)
		VALUES ($1, 'simulado-texto', 'texto', 'manual') RETURNING id::text`, sim.ID).Scan(&modID); err != nil {
		t.Fatal(err)
	}
	// Execução no mês anterior em São Paulo (2026-10-31 23:59:59 -03:00 -> 2026-11-01 02:59:59 UTC): NÃO entra no mês de novembro.
	mesAnterior := time.Date(2026, 10, 31, 23, 59, 59, 0, fusoSaoPaulo)
	if _, err := api.banco.Exec(ctx, `
		INSERT INTO ia_execucoes (correlacao_id, atividade, fornecedor_id, modelo_id, fornecedor_nome, modelo_identificador, status, custo_usd, criado_em)
		VALUES (gen_random_uuid(), 'teste', $1, $2, $3, 'simulado-texto', 'ok', 99.00, $4)`,
		sim.ID, modID, sim.Nome, mesAnterior); err != nil {
		t.Fatal(err)
	}
	// Execução em novembro no limite exato de aviso (8.00 de 10.00 = 80%).
	noMes := time.Date(2026, 11, 1, 0, 0, 0, 0, fusoSaoPaulo)
	if _, err := api.banco.Exec(ctx, `
		INSERT INTO ia_execucoes (correlacao_id, atividade, fornecedor_id, modelo_id, fornecedor_nome, modelo_identificador, status, custo_usd, criado_em)
		VALUES (gen_random_uuid(), 'teste', $1, $2, $3, 'simulado-texto', 'ok', 8.00, $4)`,
		sim.ID, modID, sim.Nome, noMes); err != nil {
		t.Fatal(err)
	}

	var lista RespostaFornecedores
	if st := c.pedir("GET", "/api/admin/fornecedores", nil, &lista); st != 200 {
		t.Fatalf("GET fornecedores falhou: %d", st)
	}
	var simLido Fornecedor
	for _, f := range lista.Fornecedores {
		if f.ID == sim.ID {
			simLido = f
		}
	}
	if simLido.GastoMesUsd != 8 || simLido.Situacao != "aviso" {
		t.Fatalf("esperava gasto=8 situacao=aviso (ignorando mês anterior), veio %+v", simLido)
	}

	// Mais 2.00 -> atinge 10.00 = limite.
	if _, err := api.banco.Exec(ctx, `
		INSERT INTO ia_execucoes (correlacao_id, atividade, fornecedor_id, modelo_id, fornecedor_nome, modelo_identificador, status, custo_usd, criado_em)
		VALUES (gen_random_uuid(), 'teste', $1, $2, $3, 'simulado-texto', 'ok', 2.00, $4)`,
		sim.ID, modID, sim.Nome, agoraFixo); err != nil {
		t.Fatal(err)
	}
	// Desativa via PATCH (nada é apagado) e confere situacao=limite.
	var simDesativado Fornecedor
	if st := c.pedir("PATCH", "/api/admin/fornecedores/"+sim.ID, map[string]any{"ativo": false}, &simDesativado); st != 200 || simDesativado.Ativo || simDesativado.Situacao != "limite" || simDesativado.GastoMesUsd != 10 {
		t.Fatalf("PATCH desativar inesperado: %d %+v", st, simDesativado)
	}

	// Regras de PATCH: endereço interno (AC39), nome repetido (409), limite inválido (400) e id inexistente (404).
	if st, body := c.pedirComCorpo("PATCH", "/api/admin/fornecedores/"+oai.ID, map[string]any{"endereco": "https://127.0.0.1/v1"}); st != 400 || !strings.Contains(string(body), "endereco_invalido") {
		t.Fatalf("PATCH endereço interno esperava 400 endereco_invalido, veio %d (%s)", st, body)
	}
	if st, body := c.pedirComCorpo("PATCH", "/api/admin/fornecedores/"+oai.ID, map[string]any{"nome": "anthropic padrão"}); st != 409 || !strings.Contains(string(body), "nome_repetido") {
		t.Fatalf("PATCH nome repetido esperava 409 nome_repetido, veio %d (%s)", st, body)
	}
	if st, body := c.pedirComCorpo("PATCH", "/api/admin/fornecedores/"+oai.ID, map[string]any{"limiteMensalUsd": 0}); st != 400 || !strings.Contains(string(body), "dados_invalidos") {
		t.Fatalf("PATCH limite inválido esperava 400 dados_invalidos, veio %d (%s)", st, body)
	}
	if st, body := c.pedirComCorpo("PATCH", "/api/admin/fornecedores/00000000-0000-0000-0000-000000000000", map[string]any{"nome": "X"}); st != 404 || !strings.Contains(string(body), "nao_encontrado") {
		t.Fatalf("PATCH id inexistente esperava 404 nao_encontrado, veio %d (%s)", st, body)
	}

	// Validação de chave: mínimo 12 caracteres, apenas ASCII visível (sem espaços internos ou quebras de linha); pontas aparadas.
	for _, ruim := range []string{"curta-12345", "sk-espaco interno 123", "sk-quebra\nlinha123", "sk-não-ascii-12345"} {
		if st, body := c.pedirComCorpo("POST", "/api/admin/fornecedores", map[string]any{
			"nome": "Chave Ruim", "tipo": "openai", "chave": ruim, "limiteMensalUsd": 10,
		}); st != 400 || !strings.Contains(string(body), "formato_invalido") {
			t.Fatalf("POST chave inválida %q esperava 400 formato_invalido, veio %d (%s)", ruim, st, body)
		}
		if st, body := c.pedirComCorpo("PATCH", "/api/admin/fornecedores/"+oai.ID, map[string]any{"chave": ruim}); st != 400 || !strings.Contains(string(body), "formato_invalido") {
			t.Fatalf("PATCH chave inválida %q esperava 400 formato_invalido, veio %d (%s)", ruim, st, body)
		}
	}
	var aparada Fornecedor
	if st := c.pedir("PATCH", "/api/admin/fornecedores/"+oai.ID, map[string]any{"chave": "  " + chaveAPINovaTeste + "\t "}, &aparada); st != 200 || aparada.ChaveFinal == nil || *aparada.ChaveFinal != "9876" {
		t.Fatalf("PATCH chave com espaços nas pontas falhou: %d %+v", st, aparada)
	}

	// Aritmética exata com frações decimais (0.1 + 0.2 = 0.30 de 0.30 -> limite; 0.24 de 0.30 -> aviso).
	sit, _, _, err := calcularSituacaoLimite("0.30", "0.30", 80)
	if err != nil || sit != "limite" {
		t.Fatalf("esperava limite para 0.30/0.30, veio %q %v", sit, err)
	}
	sit, _, _, err = calcularSituacaoLimite("0.24", "0.30", 80)
	if err != nil || sit != "aviso" {
		t.Fatalf("esperava aviso para 0.24/0.30 (80%%), veio %q %v", sit, err)
	}
	sit, _, _, err = calcularSituacaoLimite("0.23999999", "0.30", 80)
	if err != nil || sit != "normal" {
		t.Fatalf("esperava normal abaixo de 80%%, veio %q %v", sit, err)
	}
}
