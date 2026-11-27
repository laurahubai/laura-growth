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
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func configurarTLSRedeInterna(api *API, srv *httptest.Server) {
	api.permitirRedeInterna = true
	tr := srv.Client().Transport.(*http.Transport)
	api.tlsTeste = tr.TLSClientConfig
}

func TestAC38LerModelosOpenAIAnthropicESimulado(t *testing.T) {
	api := novaAPIAdminTeste(t)
	c := novoCliente(t, api)
	entrarComoPaulo(t, c)

	var modoOpenAI atomic.Int32 // 0=ok, 1=401
	var authRecebido string
	srvOpenAI := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authRecebido = r.Header.Get("Authorization")
		if modoOpenAI.Load() == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			// Propositalmente inclui a chave no corpo do fornecedor para provar que não vaza.
			_, _ = w.Write([]byte(`{"error":{"message":"Incorrect API key: ` + chaveAPITeste + `"}}`))
			return
		}
		if r.URL.Path == "/v1/models" && r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]string{{"id": "gpt-4o-mini"}, {"id": "gpt-image-1"}},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srvOpenAI.Close()

	var modoAnthropic atomic.Int32 // 0=ok, 1=401, 2=503
	var apiKeyAnthropic, versaoHeader string
	srvAnthropic := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKeyAnthropic = r.Header.Get("x-api-key")
		versaoHeader = r.Header.Get("anthropic-version")
		switch modoAnthropic.Load() {
		case 1:
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"invalid x-api-key ` + chaveAPINovaTeste + `"}}`))
			return
		case 2:
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if r.URL.Path == "/v1/models" && r.Method == http.MethodGet {
			after := r.URL.Query().Get("after_id")
			if after == "" {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data":     []map[string]string{{"id": "claude-sonnet-4-5"}},
					"has_more": true,
					"last_id":  "claude-sonnet-4-5",
				})
				return
			}
			if after == "claude-sonnet-4-5" {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data":     []map[string]string{{"id": "claude-haiku-4-5"}},
					"has_more": false,
					"last_id":  "claude-haiku-4-5",
				})
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srvAnthropic.Close()

	configurarTLSRedeInterna(api, srvOpenAI)

	var bufLog bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&bufLog, nil)))
	defer slog.SetDefault(anterior)

	// Cadastra fornecedores OpenAI (apontando para srvOpenAI), Anthropic (apontando para srvAnthropic) e Simulado.
	var fornOAI, fornAnt, fornSim Fornecedor
	if st := c.pedir("POST", "/api/admin/fornecedores", map[string]any{
		"nome":            "OpenAI Teste",
		"tipo":            "compativel_openai",
		"endereco":        srvOpenAI.URL + "/v1",
		"chave":           chaveAPITeste,
		"limiteMensalUsd": 100,
	}, &fornOAI); st != 201 {
		t.Fatalf("criar OpenAI teste falhou: %d", st)
	}
	if st := c.pedir("POST", "/api/admin/fornecedores", map[string]any{
		"nome":            "Anthropic Teste",
		"tipo":            "anthropic",
		"endereco":        srvAnthropic.URL + "/v1",
		"chave":           chaveAPINovaTeste,
		"limiteMensalUsd": 100,
	}, &fornAnt); st != 201 {
		t.Fatalf("criar Anthropic teste falhou: %d", st)
	}
	if st := c.pedir("POST", "/api/admin/fornecedores", map[string]any{
		"nome":            "Simulado Teste",
		"tipo":            "simulado",
		"limiteMensalUsd": 10,
	}, &fornSim); st != 201 {
		t.Fatalf("criar Simulado teste falhou: %d", st)
	}

	// OpenAI: lista ok e marca jaCadastrado=false -> depois de cadastrar um modelo, jaCadastrado=true.
	var lidos RespostaLerModelos
	if st := c.pedir("POST", "/api/admin/fornecedores/"+fornOAI.ID+"/ler-modelos", nil, &lidos); st != 200 {
		t.Fatalf("ler-modelos OpenAI esperava 200, veio %d", st)
	}
	if authRecebido != "Bearer "+chaveAPITeste {
		t.Fatalf("cabeçalho Authorization inesperado: %q", authRecebido)
	}
	if len(lidos.Modelos) != 2 || lidos.Modelos[0].Identificador != "gpt-4o-mini" || lidos.Modelos[0].JaCadastrado {
		t.Fatalf("lista OpenAI inesperada: %+v", lidos.Modelos)
	}
	if st := c.pedir("POST", "/api/admin/fornecedores/"+fornOAI.ID+"/modelos", map[string]any{
		"identificador": "gpt-4o-mini",
		"servico":       "texto",
		"origem":        "lido",
	}, nil); st != 201 {
		t.Fatalf("cadastrar modelo lido esperava 201, veio %d", st)
	}
	if st := c.pedir("POST", "/api/admin/fornecedores/"+fornOAI.ID+"/ler-modelos", nil, &lidos); st != 200 || !lidos.Modelos[0].JaCadastrado || lidos.Modelos[1].JaCadastrado {
		t.Fatalf("jaCadastrado esperado [true, false], veio %+v", lidos.Modelos)
	}

	// OpenAI 401 -> 422 fornecedor_recusou_chave sem vazar chave na resposta nem no log.
	modoOpenAI.Store(1)
	st, body := c.pedirComCorpo("POST", "/api/admin/fornecedores/"+fornOAI.ID+"/ler-modelos", nil)
	if st != 422 || !strings.Contains(string(body), "fornecedor_recusou_chave") {
		t.Fatalf("OpenAI 401 esperava 422 fornecedor_recusou_chave, veio %d (%s)", st, body)
	}
	if bytes.Contains(body, []byte(chaveAPITeste)) {
		t.Fatalf("resposta 401 vazou a chave: %s", body)
	}

	// Anthropic: troca TLS de teste para srvAnthropic -> lista ok, 401 -> recusou_chave, 503/fechado -> indisponivel.
	configurarTLSRedeInterna(api, srvAnthropic)
	if st := c.pedir("POST", "/api/admin/fornecedores/"+fornAnt.ID+"/ler-modelos", nil, &lidos); st != 200 {
		t.Fatalf("ler-modelos Anthropic esperava 200, veio %d", st)
	}
	if apiKeyAnthropic != chaveAPINovaTeste || versaoHeader != versaoAnthropic || len(lidos.Modelos) != 2 ||
		lidos.Modelos[0].Identificador != "claude-sonnet-4-5" || lidos.Modelos[1].Identificador != "claude-haiku-4-5" {
		t.Fatalf("Anthropic paginado inesperado: key=%q ver=%q modelos=%+v", apiKeyAnthropic, versaoHeader, lidos.Modelos)
	}

	modoAnthropic.Store(1)
	st, body = c.pedirComCorpo("POST", "/api/admin/fornecedores/"+fornAnt.ID+"/ler-modelos", nil)
	if st != 422 || !strings.Contains(string(body), "fornecedor_recusou_chave") || bytes.Contains(body, []byte(chaveAPINovaTeste)) {
		t.Fatalf("Anthropic 401 inesperado: %d (%s)", st, body)
	}

	modoAnthropic.Store(2)
	st, body = c.pedirComCorpo("POST", "/api/admin/fornecedores/"+fornAnt.ID+"/ler-modelos", nil)
	if st != 422 || !strings.Contains(string(body), "fornecedor_indisponivel") {
		t.Fatalf("Anthropic 503 esperava 422 fornecedor_indisponivel, veio %d (%s)", st, body)
	}

	// Servidor fora do ar (fechado) -> 422 fornecedor_indisponivel.
	srvAnthropic.Close()
	st, body = c.pedirComCorpo("POST", "/api/admin/fornecedores/"+fornAnt.ID+"/ler-modelos", nil)
	if st != 422 || !strings.Contains(string(body), "fornecedor_indisponivel") || bytes.Contains(body, []byte(chaveAPINovaTeste)) {
		t.Fatalf("Anthropic fora do ar esperava 422 fornecedor_indisponivel, veio %d (%s)", st, body)
	}

	// Simulado funciona offline.
	if st := c.pedir("POST", "/api/admin/fornecedores/"+fornSim.ID+"/ler-modelos", nil, &lidos); st != 200 || len(lidos.Modelos) != 2 {
		t.Fatalf("ler-modelos simulado inesperado: %d %+v", st, lidos.Modelos)
	}

	// Fornecedor inativo -> 409 fornecedor_inativo.
	if st := c.pedir("PATCH", "/api/admin/fornecedores/"+fornSim.ID, map[string]any{"ativo": false}, nil); st != 200 {
		t.Fatalf("desativar fornecedor falhou: %d", st)
	}
	if st, body := c.pedirComCorpo("POST", "/api/admin/fornecedores/"+fornSim.ID+"/ler-modelos", nil); st != 409 || !strings.Contains(string(body), "fornecedor_inativo") {
		t.Fatalf("fornecedor inativo esperava 409 fornecedor_inativo, veio %d (%s)", st, body)
	}

	// Sem permissão para pessoa comum nas rotas da S2.
	criarUsuarioComum(t, api.banco, "comum-s2@exemplo.com")
	comum := novoCliente(t, api)
	comum.pedir("POST", "/api/sessao", map[string]string{"email": "comum-s2@exemplo.com", "senha": senhaTeste}, nil)
	for _, rt := range []struct{ m, p string }{
		{"POST", "/api/admin/fornecedores/" + fornOAI.ID + "/ler-modelos"},
		{"GET", "/api/admin/fornecedores/" + fornOAI.ID + "/modelos"},
		{"POST", "/api/admin/fornecedores/" + fornOAI.ID + "/modelos"},
		{"PATCH", "/api/admin/modelos/00000000-0000-0000-0000-000000000000"},
		{"GET", "/api/admin/modelos/00000000-0000-0000-0000-000000000000/precos"},
		{"POST", "/api/admin/modelos/00000000-0000-0000-0000-000000000000/precos"},
	} {
		if st, _ := comum.pedirComCorpo(rt.m, rt.p, nil); st != 403 {
			t.Fatalf("%s %s para pessoa comum esperava 403, veio %d", rt.m, rt.p, st)
		}
	}

	// Se proteção de rede interna estiver ligada e endereço no banco for 127.0.0.1, recusa na conexão.
	api.permitirRedeInterna = false
	if st, body := c.pedirComCorpo("POST", "/api/admin/fornecedores/"+fornOAI.ID+"/ler-modelos", nil); st != 400 || !strings.Contains(string(body), "endereco_invalido") {
		t.Fatalf("endereço interno na conexão esperava 400 endereco_invalido, veio %d (%s)", st, body)
	}

	logs := bufLog.String()
	if strings.Contains(logs, chaveAPITeste) || strings.Contains(logs, chaveAPINovaTeste) {
		t.Fatalf("logs vazaram chave de API: %s", logs)
	}
}

func TestConectoresGeracaoTextoEImagem(t *testing.T) {
	ctx := context.Background()
	imgBase64 := base64.StdEncoding.EncodeToString(pngMinimo)

	// Simulado: texto, imagem e modelo com "falha" no nome.
	sim, err := novoConector("simulado", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	rtSim, err := sim.gerarTexto(ctx, "simulado-texto", "Oi")
	if err != nil || rtSim.Texto == "" || rtSim.TokensEntrada <= 0 || rtSim.TokensSaida <= 0 {
		t.Fatalf("simulado texto inesperado: %+v err=%v", rtSim, err)
	}
	riSim, err := sim.gerarImagem(ctx, "simulado-imagem", "círculo azul")
	if err != nil || !strings.HasPrefix(riSim.ImagemDataUrl, "data:image/png;base64,") || riSim.Imagens != 1 ||
		riSim.TokensEntrada != 12 || riSim.TokensEntradaImagem != 0 || riSim.TokensSaida != 1258 {
		t.Fatalf("simulado imagem inesperado: %+v err=%v", riSim, err)
	}
	if _, err := sim.gerarTexto(ctx, "simulado-falha", "Oi"); err == nil {
		t.Fatal("modelo simulado com 'falha' deveria falhar no texto")
	}
	if _, err := sim.gerarImagem(ctx, "simulado-imagem-falha", "Oi"); err == nil {
		t.Fatal("modelo simulado com 'falha' deveria falhar na imagem")
	}

	// OpenAI TLS mock para chat/completions, images/generations e limites de resposta.
	var paginasInfinitas atomic.Int32
	srvOAI := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/chat/completions":
			if r.URL.Query().Get("grande") == "1" {
				// > 2 MB para texto deve ser recusado.
				_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"` + strings.Repeat("a", int(limiteRespostaPadrao)+100) + `"}}]}`))
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{{"message": map[string]string{"content": "Sou gpt-4o"}}},
				"usage":   map[string]int{"prompt_tokens": 10, "completion_tokens": 25},
			})
		case "/v1/images/generations":
			var reqBody map[string]any
			_ = json.NewDecoder(r.Body).Decode(&reqBody)
			prompt, _ := reqBody["prompt"].(string)
			modeloReq, _ := reqBody["model"].(string)
			_, temResponseFormat := reqBody["response_format"]
			if strings.HasPrefix(strings.ToLower(modeloReq), "dall-e-") {
				if reqBody["response_format"] != "b64_json" {
					w.WriteHeader(400)
					return
				}
			} else if temResponseFormat {
				w.WriteHeader(400)
				_, _ = w.Write([]byte(`{"error":{"message":"Unknown parameter: 'response_format'."}}`))
				return
			}
			switch prompt {
			case "nao-imagem":
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": []map[string]string{{"b64_json": base64.StdEncoding.EncodeToString([]byte("<html>nao-e-imagem</html>"))}},
				})
				return
			case "jpeg":
				// Cabeçalho JPEG mínimo (SOI + APP0) para detecção de image/jpeg.
				jpegBytes := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xff, 0xd9}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": []map[string]string{{"b64_json": base64.StdEncoding.EncodeToString(jpegBytes)}},
				})
				return
			case "grande-ok":
				// > 2 MB (e < 20 MB): PNG válido com padding > 2.2 MB para provar que imagem aceita até 20 MB.
				pngGrande := append(append([]byte(nil), pngMinimo...), bytes.Repeat([]byte{0x00}, 2300*1024)...)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": []map[string]string{{"b64_json": base64.StdEncoding.EncodeToString(pngGrande)}},
				})
				return
			case "maior-20mb":
				_, _ = w.Write([]byte(`{"data":[{"b64_json":"` + strings.Repeat("A", int(limiteRespostaImagem)+100) + `"}]}`))
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data":  []map[string]string{{"b64_json": imgBase64}},
				"usage": map[string]int{"input_tokens": 5, "output_tokens": 0},
			})
		case "/v1/models":
			p := paginasInfinitas.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data":     []map[string]string{{"id": "claude-pag-" + string(rune('a'+p%26))}},
				"has_more": true,
				"last_id":  "id-" + string(rune('a'+p%26)),
			})
		default:
			w.WriteHeader(404)
		}
	}))
	defer srvOAI.Close()

	conOAI, err := novoConector("openai", srvOAI.URL+"/v1", chaveAPITeste, srvOAI.Client())
	if err != nil {
		t.Fatal(err)
	}
	rtOAI, err := conOAI.gerarTexto(ctx, "gpt-4o", "Oi")
	if err != nil || rtOAI.Texto != "Sou gpt-4o" || rtOAI.TokensEntrada != 10 || rtOAI.TokensSaida != 25 {
		t.Fatalf("OpenAI gerarTexto inesperado: %+v err=%v", rtOAI, err)
	}
	// Apenas dall-e-* envia response_format: b64_json; chatgpt-image-latest e gpt-image-1 não enviam response_format (J1).
	riChatGPT, err := conOAI.gerarImagem(ctx, "chatgpt-image-latest", "círculo")
	if err != nil || riChatGPT.ImagemDataUrl != "data:image/png;base64,"+imgBase64 || riChatGPT.Imagens != 1 || riChatGPT.TokensEntrada != 5 {
		t.Fatalf("OpenAI chatgpt-image-latest gerarImagem inesperado: %+v err=%v", riChatGPT, err)
	}
	riOAI, err := conOAI.gerarImagem(ctx, "gpt-image-1", "círculo")
	if err != nil || riOAI.ImagemDataUrl != "data:image/png;base64,"+imgBase64 || riOAI.Imagens != 1 || riOAI.TokensEntrada != 5 {
		t.Fatalf("OpenAI gpt-image-1 gerarImagem inesperado: %+v err=%v", riOAI, err)
	}
	riDalle, err := conOAI.gerarImagem(ctx, "dall-e-3", "jpeg")
	if err != nil || !strings.HasPrefix(riDalle.ImagemDataUrl, "data:image/jpeg;base64,") {
		t.Fatalf("OpenAI dall-e-3 JPEG inesperado: %+v err=%v", riDalle, err)
	}
	// Imagem > 2 MB (<= 20 MB) é aceita.
	if riGrande, err := conOAI.gerarImagem(ctx, "gpt-image-1", "grande-ok"); err != nil || !strings.HasPrefix(riGrande.ImagemDataUrl, "data:image/png;base64,") {
		t.Fatalf("imagem > 2MB e <= 20MB deveria passar, veio err=%v", err)
	}
	// Imagem > 20 MB devolve fornecedor_erro com mensagem exata.
	var efImg *erroFornecedor
	if _, err := conOAI.gerarImagem(ctx, "gpt-image-1", "maior-20mb"); !errors.As(err, &efImg) || efImg.codigo != "fornecedor_erro" || efImg.mensagem != "resposta do fornecedor maior que o permitido" {
		t.Fatalf("imagem > 20MB esperava erro de tamanho, veio %v", err)
	}
	// Conteúdo decodificado não-imagem (HTML) é recusado.
	if _, err := conOAI.gerarImagem(ctx, "gpt-image-1", "nao-imagem"); !errors.As(err, &efImg) || efImg.codigo != "fornecedor_erro" {
		t.Fatalf("b64_json não-imagem deveria ser recusado, veio %v", err)
	}
	// Texto > 2 MB é recusado com mensagem exata.
	conTextoGrande := &conectorOpenAI{endereco: srvOAI.URL + "/v1/chat/completions?grande=1", chave: chaveAPITeste, cliente: srvOAI.Client()}
	var respDummy map[string]any
	if err := fazerPedidoJSON(ctx, conTextoGrande.cliente, http.MethodPost, conTextoGrande.endereco, nil, map[string]string{}, &respDummy, limiteRespostaPadrao); !errors.As(err, &efImg) || efImg.codigo != "fornecedor_erro" || efImg.mensagem != "resposta do fornecedor maior que o permitido" {
		t.Fatalf("resposta > 2MB esperava erro de tamanho, veio %v", err)
	}
	// Anthropic para no teto de 20 páginas.
	conPag, _ := novoConector("anthropic", srvOAI.URL+"/v1", chaveAPITeste, srvOAI.Client())
	if _, err := conPag.listarModelos(ctx); err != nil {
		t.Fatalf("paginação Anthropic até teto falhou: %v", err)
	}
	if paginasInfinitas.Load() != maxPaginasAnthropic {
		t.Fatalf("esperava teto de %d páginas, consultou %d", maxPaginasAnthropic, paginasInfinitas.Load())
	}
	if timeoutImagem != 120*time.Second {
		t.Fatalf("timeoutImagem esperado 120s, veio %v", timeoutImagem)
	}

	// Anthropic TLS mock: texto ok; imagem recusada.
	srvAnt := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/messages" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"content": []map[string]string{{"type": "text", "text": "Sou Claude"}},
				"usage":   map[string]int{"input_tokens": 14, "output_tokens": 22},
			})
			return
		}
		w.WriteHeader(404)
	}))
	defer srvAnt.Close()

	conAnt, err := novoConector("anthropic", srvAnt.URL+"/v1", chaveAPINovaTeste, srvAnt.Client())
	if err != nil {
		t.Fatal(err)
	}
	rtAnt, err := conAnt.gerarTexto(ctx, "claude-sonnet-4-5", "Oi")
	if err != nil || rtAnt.Texto != "Sou Claude" || rtAnt.TokensEntrada != 14 || rtAnt.TokensSaida != 22 {
		t.Fatalf("Anthropic gerarTexto inesperado: %+v err=%v", rtAnt, err)
	}
	var ef *erroFornecedor
	if _, err := conAnt.gerarImagem(ctx, "claude-sonnet-4-5", "círculo"); !errors.As(err, &ef) || ef.codigo != "fornecedor_erro" {
		t.Fatalf("Anthropic gerarImagem deveria recusar, veio %v", err)
	}
}

func TestValidacaoTokensFornecedor(t *testing.T) {
	ctx := context.Background()
	imgBase64 := base64.StdEncoding.EncodeToString(pngMinimo)

	// Limites unitários (0 e 10.000.000 aceitos; negativos ou > 10.000.000 recusados com fornecedor_erro).
	for _, tc := range []struct {
		ent, sai int
		valido   bool
	}{
		{0, 0, true},
		{maxTokensFornecedor, maxTokensFornecedor, true},
		{-1, 10, false},
		{10, -1, false},
		{maxTokensFornecedor + 1, 10, false},
		{10, maxTokensFornecedor + 1, false},
	} {
		err := validarTokensFornecedor(tc.ent, tc.sai)
		if tc.valido && err != nil {
			t.Fatalf("validarTokensFornecedor(%d, %d) deveria aceitar, veio %v", tc.ent, tc.sai, err)
		}
		if !tc.valido {
			var ef *erroFornecedor
			if !errors.As(err, &ef) || ef.codigo != "fornecedor_erro" {
				t.Fatalf("validarTokensFornecedor(%d, %d) esperava fornecedor_erro, veio %v", tc.ent, tc.sai, err)
			}
		}
	}

	var promptTok, compTok atomic.Int64
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/chat/completions":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{{"message": map[string]string{"content": "ok"}}},
				"usage":   map[string]int64{"prompt_tokens": promptTok.Load(), "completion_tokens": compTok.Load()},
			})
		case "/v1/images/generations":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data":  []map[string]string{{"b64_json": imgBase64}},
				"usage": map[string]int64{"input_tokens": promptTok.Load(), "output_tokens": compTok.Load()},
			})
		case "/v1/messages":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"content": []map[string]string{{"type": "text", "text": "ok"}},
				"usage":   map[string]int64{"input_tokens": promptTok.Load(), "output_tokens": compTok.Load()},
			})
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	conOAI, _ := novoConector("openai", srv.URL+"/v1", chaveAPITeste, srv.Client())
	conAnt, _ := novoConector("anthropic", srv.URL+"/v1", chaveAPITeste, srv.Client())

	promptTok.Store(maxTokensFornecedor + 1)
	compTok.Store(10)
	var ef *erroFornecedor
	if _, err := conOAI.gerarTexto(ctx, "gpt-4o", "Oi"); !errors.As(err, &ef) || ef.codigo != "fornecedor_erro" {
		t.Fatalf("OpenAI texto >10M tokens esperava fornecedor_erro, veio %v", err)
	}
	if _, err := conOAI.gerarImagem(ctx, "gpt-image-1", "Oi"); !errors.As(err, &ef) || ef.codigo != "fornecedor_erro" {
		t.Fatalf("OpenAI imagem >10M tokens esperava fornecedor_erro, veio %v", err)
	}
	if _, err := conAnt.gerarTexto(ctx, "claude-sonnet-4-5", "Oi"); !errors.As(err, &ef) || ef.codigo != "fornecedor_erro" {
		t.Fatalf("Anthropic texto >10M tokens esperava fornecedor_erro, veio %v", err)
	}

	promptTok.Store(0)
	compTok.Store(-5)
	if _, err := conOAI.gerarTexto(ctx, "gpt-4o", "Oi"); !errors.As(err, &ef) || ef.codigo != "fornecedor_erro" {
		t.Fatalf("OpenAI texto tokens negativos esperava fornecedor_erro, veio %v", err)
	}
	if _, err := conAnt.gerarTexto(ctx, "claude-sonnet-4-5", "Oi"); !errors.As(err, &ef) || ef.codigo != "fornecedor_erro" {
		t.Fatalf("Anthropic texto tokens negativos esperava fornecedor_erro, veio %v", err)
	}

	promptTok.Store(maxTokensFornecedor)
	compTok.Store(0)
	if rt, err := conOAI.gerarTexto(ctx, "gpt-4o", "Oi"); err != nil || rt.TokensEntrada != maxTokensFornecedor {
		t.Fatalf("OpenAI exatamente 10M tokens deveria passar, veio %+v err=%v", rt, err)
	}
}

func TestModelosEPrecosRegrasEVigencia(t *testing.T) {
	api := novaAPIAdminTeste(t)
	agoraFixo := time.Date(2026, 11, 25, 14, 0, 0, 0, fusoSaoPaulo)
	api.agora = func() time.Time { return agoraFixo }

	c := novoCliente(t, api)
	entrarComoPaulo(t, c)

	var fornAnt, fornSim Fornecedor
	if st := c.pedir("POST", "/api/admin/fornecedores", map[string]any{
		"nome":            "Anthropic Modelos",
		"tipo":            "anthropic",
		"chave":           chaveAPITeste,
		"limiteMensalUsd": 100,
	}, &fornAnt); st != 201 {
		t.Fatalf("criar Anthropic falhou: %d", st)
	}
	if st := c.pedir("POST", "/api/admin/fornecedores", map[string]any{
		"nome":            "Simulado Modelos",
		"tipo":            "simulado",
		"limiteMensalUsd": 50,
		"avisoPercentual": 80,
	}, &fornSim); st != 201 {
		t.Fatalf("criar Simulado falhou: %d", st)
	}

	// Anthropic não aceita serviço imagem -> 400 servico_invalido.
	if st, body := c.pedirComCorpo("POST", "/api/admin/fornecedores/"+fornAnt.ID+"/modelos", map[string]any{
		"identificador": "claude-img",
		"servico":       "imagem",
		"origem":        "manual",
	}); st != 400 || !strings.Contains(string(body), "servico_invalido") {
		t.Fatalf("Anthropic imagem esperava 400 servico_invalido, veio %d (%s)", st, body)
	}

	// Cadastro de modelo com limite próprio e sem limite próprio.
	var modTexto, modImg Modelo
	if st := c.pedir("POST", "/api/admin/fornecedores/"+fornSim.ID+"/modelos", map[string]any{
		"identificador":   "simulado-texto",
		"servico":         "texto",
		"origem":          "lido",
		"limiteMensalUsd": 10,
		"avisoPercentual": 50,
	}, &modTexto); st != 201 {
		t.Fatalf("criar modelo texto esperava 201, veio %d", st)
	}
	if modTexto.LimiteMensalUsd == nil || *modTexto.LimiteMensalUsd != 10 || modTexto.AvisoPercentual == nil || *modTexto.AvisoPercentual != 50 || !modTexto.Simulado {
		t.Fatalf("modelo texto inesperado: %+v", modTexto)
	}
	if st := c.pedir("POST", "/api/admin/fornecedores/"+fornSim.ID+"/modelos", map[string]any{
		"identificador": "simulado-imagem",
		"servico":       "imagem",
		"origem":        "manual",
	}, &modImg); st != 201 {
		t.Fatalf("criar modelo imagem esperava 201, veio %d", st)
	}
	if modImg.LimiteMensalUsd != nil || modImg.AvisoPercentual != nil {
		t.Fatalf("modelo imagem esperava limite/aviso nil, veio %+v", modImg)
	}

	// Modelo repetido -> 409 modelo_repetido.
	if st, body := c.pedirComCorpo("POST", "/api/admin/fornecedores/"+fornSim.ID+"/modelos", map[string]any{
		"identificador": "simulado-texto",
		"servico":       "texto",
		"origem":        "manual",
	}); st != 409 || !strings.Contains(string(body), "modelo_repetido") {
		t.Fatalf("modelo repetido esperava 409 modelo_repetido, veio %d (%s)", st, body)
	}

	// Gasto de $6 em modTexto: com limite próprio 10 (aviso 50%) fica em "aviso";
	// após PATCH com limiteMensalUsd=null e avisoPercentual=null, volta a usar o do fornecedor (50/80% -> "normal").
	if _, err := api.banco.Exec(context.Background(), `
		INSERT INTO ia_execucoes (correlacao_id, atividade, fornecedor_id, modelo_id, fornecedor_nome, modelo_identificador, status, custo_usd, criado_em)
		VALUES (gen_random_uuid(), 'teste', $1, $2, $3, $4, 'ok', 6.00, $5)`,
		fornSim.ID, modTexto.ID, fornSim.Nome, modTexto.Identificador, agoraFixo); err != nil {
		t.Fatal(err)
	}
	var listaMod RespostaModelos
	if st := c.pedir("GET", "/api/admin/fornecedores/"+fornSim.ID+"/modelos", nil, &listaMod); st != 200 || len(listaMod.Modelos) != 2 {
		t.Fatalf("GET modelos inesperado: %d %+v", st, listaMod)
	}
	// Ordenado por identificador: simulado-imagem, simulado-texto.
	if listaMod.Modelos[1].ID != modTexto.ID || listaMod.Modelos[1].Situacao != "aviso" {
		t.Fatalf("esperava situacao=aviso com limite próprio 10/50%%, veio %+v", listaMod.Modelos[1])
	}

	// PATCH com null explícito limpa limite e aviso próprios; ativo=false desativa sem apagar.
	var modEditado Modelo
	if st := c.pedir("PATCH", "/api/admin/modelos/"+modTexto.ID, map[string]any{
		"limiteMensalUsd": nil,
		"avisoPercentual": nil,
		"ativo":           false,
	}, &modEditado); st != 200 {
		t.Fatalf("PATCH modelo esperava 200, veio %d", st)
	}
	if modEditado.LimiteMensalUsd != nil || modEditado.AvisoPercentual != nil || modEditado.Ativo || modEditado.Situacao != "normal" {
		t.Fatalf("PATCH modelo com null/desativar inesperado: %+v", modEditado)
	}

	// Preços: histórico preservado, preço futuro não é vigente ainda, preco_repetido dá 409.
	var p1, p2, p3, pImg Preco
	if st := c.pedir("POST", "/api/admin/modelos/"+modTexto.ID+"/precos", map[string]any{
		"unidade":        "token_entrada",
		"precoUsd":       1.5,
		"vigenciaInicio": "2026-10-01",
	}, &p1); st != 201 {
		t.Fatalf("criar preço 1 falhou: %d", st)
	}
	if st := c.pedir("POST", "/api/admin/modelos/"+modTexto.ID+"/precos", map[string]any{
		"unidade":        "token_entrada",
		"precoUsd":       2.25,
		"vigenciaInicio": "2026-11-25",
	}, &p2); st != 201 {
		t.Fatalf("criar preço 2 falhou: %d", st)
	}
	if st := c.pedir("POST", "/api/admin/modelos/"+modTexto.ID+"/precos", map[string]any{
		"unidade":        "token_entrada",
		"precoUsd":       9.99,
		"vigenciaInicio": "2026-12-01", // futuro em relação a agoraFixo (2026-11-25)
	}, &p3); st != 201 {
		t.Fatalf("criar preço futuro falhou: %d", st)
	}
	if st := c.pedir("POST", "/api/admin/modelos/"+modTexto.ID+"/precos", map[string]any{
		"unidade":        "token_saida",
		"precoUsd":       0,
		"vigenciaInicio": "2026-11-01",
	}, nil); st != 201 {
		t.Fatalf("criar preço saída = 0 falhou: %d", st)
	}
	if st := c.pedir("POST", "/api/admin/modelos/"+modImg.ID+"/precos", map[string]any{
		"unidade":        "imagem",
		"precoUsd":       0.04,
		"vigenciaInicio": "2026-11-01",
	}, &pImg); st != 201 {
		t.Fatalf("criar preço imagem falhou: %d", st)
	}

	// Preço repetido (mesma unidade e vigência) -> 409 preco_repetido.
	if st, body := c.pedirComCorpo("POST", "/api/admin/modelos/"+modTexto.ID+"/precos", map[string]any{
		"unidade":        "token_entrada",
		"precoUsd":       3.0,
		"vigenciaInicio": "2026-11-25",
	}); st != 409 || !strings.Contains(string(body), "preco_repetido") {
		t.Fatalf("preço repetido esperava 409 preco_repetido, veio %d (%s)", st, body)
	}

	// Lista de preços vem mais recente primeiro (2026-12-01, 2026-11-25, 2026-11-01, 2026-10-01).
	var respPrecos RespostaPrecos
	if st := c.pedir("GET", "/api/admin/modelos/"+modTexto.ID+"/precos", nil, &respPrecos); st != 200 || len(respPrecos.Precos) != 4 {
		t.Fatalf("GET preços inesperado: %d %+v", st, respPrecos.Precos)
	}
	if respPrecos.Precos[0].VigenciaInicio != "2026-12-01" || respPrecos.Precos[3].VigenciaInicio != "2026-10-01" {
		t.Fatalf("ordenação de preços inesperada: %+v", respPrecos.Precos)
	}

	// precosVigentes do modTexto considera vigência <= 2026-11-25 (2.25 entrada, 0 saída, nil imagem) e modImg (0.04 imagem).
	if st := c.pedir("GET", "/api/admin/fornecedores/"+fornSim.ID+"/modelos", nil, &listaMod); st != 200 {
		t.Fatalf("GET modelos falhou: %d", st)
	}
	gotImg, gotTxt := listaMod.Modelos[0], listaMod.Modelos[1]
	if gotImg.PrecosVigentes.Imagem == nil || *gotImg.PrecosVigentes.Imagem != 0.04 {
		t.Fatalf("preço vigente de imagem inesperado: %+v", gotImg.PrecosVigentes)
	}
	if gotTxt.PrecosVigentes.TokenEntrada == nil || *gotTxt.PrecosVigentes.TokenEntrada != 2.25 ||
		gotTxt.PrecosVigentes.TokenSaida == nil || *gotTxt.PrecosVigentes.TokenSaida != 0 ||
		gotTxt.PrecosVigentes.Imagem != nil {
		t.Fatalf("preços vigentes de texto inesperados: %+v", gotTxt.PrecosVigentes)
	}

	// P1: modelo de texto recusa imagem e token_entrada_imagem (400 formato_invalido);
	// modelo de imagem aceita token_entrada, token_entrada_imagem, token_saida e imagem.
	if st, body := c.pedirComCorpo("POST", "/api/admin/modelos/"+modTexto.ID+"/precos", map[string]any{
		"unidade": "imagem", "precoUsd": 0.1, "vigenciaInicio": "2026-11-25",
	}); st != 400 || !strings.Contains(string(body), "formato_invalido") {
		t.Fatalf("unidade imagem em modelo texto esperava 400 formato_invalido, veio %d (%s)", st, body)
	}
	if st, body := c.pedirComCorpo("POST", "/api/admin/modelos/"+modTexto.ID+"/precos", map[string]any{
		"unidade": "token_entrada_imagem", "precoUsd": 0.1, "vigenciaInicio": "2026-11-25",
	}); st != 400 || !strings.Contains(string(body), "formato_invalido") {
		t.Fatalf("unidade token_entrada_imagem em modelo texto esperava 400 formato_invalido, veio %d (%s)", st, body)
	}
	if st, body := c.pedirComCorpo("POST", "/api/admin/modelos/"+modImg.ID+"/precos", map[string]any{
		"unidade": "invalida", "precoUsd": 0.1, "vigenciaInicio": "2026-11-25",
	}); st != 400 || !strings.Contains(string(body), "dados_invalidos") {
		t.Fatalf("unidade desconhecida esperava 400 dados_invalidos, veio %d (%s)", st, body)
	}
	if st := c.pedir("POST", "/api/admin/modelos/"+modImg.ID+"/precos", map[string]any{
		"unidade": "token_entrada", "precoUsd": 2.0, "vigenciaInicio": "2026-11-25",
	}, nil); st != 201 {
		t.Fatalf("modelo imagem deveria aceitar token_entrada, veio %d", st)
	}
	if st := c.pedir("POST", "/api/admin/modelos/"+modImg.ID+"/precos", map[string]any{
		"unidade": "token_entrada_imagem", "precoUsd": 10.0, "vigenciaInicio": "2026-11-25",
	}, nil); st != 201 {
		t.Fatalf("modelo imagem deveria aceitar token_entrada_imagem, veio %d", st)
	}
	if st := c.pedir("POST", "/api/admin/modelos/"+modImg.ID+"/precos", map[string]any{
		"unidade": "token_saida", "precoUsd": 40.0, "vigenciaInicio": "2026-11-25",
	}, nil); st != 201 {
		t.Fatalf("modelo imagem deveria aceitar token_saida, veio %d", st)
	}

	if st := c.pedir("GET", "/api/admin/fornecedores/"+fornSim.ID+"/modelos", nil, &listaMod); st != 200 {
		t.Fatalf("GET modelos após novos preços de imagem falhou: %d", st)
	}
	gotImg = listaMod.Modelos[0]
	if gotImg.PrecosVigentes.TokenEntrada == nil || *gotImg.PrecosVigentes.TokenEntrada != 2.0 ||
		gotImg.PrecosVigentes.TokenEntradaImagem == nil || *gotImg.PrecosVigentes.TokenEntradaImagem != 10.0 ||
		gotImg.PrecosVigentes.TokenSaida == nil || *gotImg.PrecosVigentes.TokenSaida != 40.0 ||
		gotImg.PrecosVigentes.Imagem == nil || *gotImg.PrecosVigentes.Imagem != 0.04 {
		t.Fatalf("preços vigentes de imagem completos inesperados: %+v", gotImg.PrecosVigentes)
	}

	// Modelo SEM limite próprio (modImg) usa o gasto do fornecedor inteiro contra o limite do fornecedor (50.00, aviso 80% = 40.00).
	// modTexto já gastou 6.00; somando 35.00 em modTexto -> fornecedor gasta 41.00 (>= 80% de 50.00).
	if _, err := api.banco.Exec(context.Background(), `
		INSERT INTO ia_execucoes (correlacao_id, atividade, fornecedor_id, modelo_id, fornecedor_nome, modelo_identificador, status, custo_usd, criado_em)
		VALUES (gen_random_uuid(), 'teste', $1, $2, $3, $4, 'ok', 35.00, $5)`,
		fornSim.ID, modTexto.ID, fornSim.Nome, modTexto.Identificador, agoraFixo); err != nil {
		t.Fatal(err)
	}
	if st := c.pedir("GET", "/api/admin/fornecedores/"+fornSim.ID+"/modelos", nil, &listaMod); st != 200 {
		t.Fatalf("GET modelos falhou: %d", st)
	}
	gotImg = listaMod.Modelos[0]
	if gotImg.GastoMesUsd != 0 || gotImg.Situacao != "aviso" {
		t.Fatalf("modelo sem limite próprio esperava gastoMesUsd=0 e situacao=aviso pelo gasto do fornecedor, veio %+v", gotImg)
	}

	// PATCH do modelo recusa troca de serviço se estiver em uso em alguma atividade (409 servico_incompativel).
	if st := c.pedir("PUT", "/api/admin/atividades/imagem", map[string]any{
		"modeloPrincipalId": modImg.ID, "modeloReservaId": nil,
	}, nil); st != 200 {
		t.Fatalf("PUT atividade imagem falhou: %d", st)
	}
	if st, body := c.pedirComCorpo("PATCH", "/api/admin/modelos/"+modImg.ID, map[string]any{
		"servico": "texto",
	}); st != 409 || !strings.Contains(string(body), "servico_incompativel") || !strings.Contains(string(body), "Imagem") {
		t.Fatalf("trocar serviço de modelo em uso esperava 409 servico_incompativel citando Imagem, veio %d (%s)", st, body)
	}
}

func TestP2OpenAIImagemUsageComESemDetalhes(t *testing.T) {
	ctx := context.Background()
	imgBase64 := base64.StdEncoding.EncodeToString(pngMinimo)

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/generations" {
			w.WriteHeader(404)
			return
		}
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		prompt, _ := req["prompt"].(string)
		switch prompt {
		case "com-detalhes":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]string{{"b64_json": imgBase64}},
				"usage": map[string]any{
					"input_tokens":  352,
					"output_tokens": 1258,
					"input_tokens_details": map[string]int{
						"text_tokens":  12,
						"image_tokens": 340,
					},
				},
			})
		case "sem-detalhes":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]string{{"b64_json": imgBase64}},
				"usage": map[string]any{
					"input_tokens":  25,
					"output_tokens": 900,
				},
			})
		case "so-image-tokens":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]string{{"b64_json": imgBase64}},
				"usage": map[string]any{
					"input_tokens":  100,
					"output_tokens": 800,
					"input_tokens_details": map[string]int{
						"image_tokens": 40,
					},
				},
			})
		case "inconsistente-soma":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]string{{"b64_json": imgBase64}},
				"usage": map[string]any{
					"input_tokens":  100,
					"output_tokens": 800,
					"input_tokens_details": map[string]int{
						"text_tokens":  12,
						"image_tokens": 40,
					},
				},
			})
		case "inconsistente-image-maior":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]string{{"b64_json": imgBase64}},
				"usage": map[string]any{
					"input_tokens":  30,
					"output_tokens": 800,
					"input_tokens_details": map[string]int{
						"image_tokens": 40,
					},
				},
			})
		case "inconsistente-text-diferente":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]string{{"b64_json": imgBase64}},
				"usage": map[string]any{
					"input_tokens":  30,
					"output_tokens": 800,
					"input_tokens_details": map[string]int{
						"text_tokens": 20,
					},
				},
			})
		case "detalhes-negativos":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]string{{"b64_json": imgBase64}},
				"usage": map[string]any{
					"input_tokens":  10,
					"output_tokens": 100,
					"input_tokens_details": map[string]int{
						"text_tokens":  -1,
						"image_tokens": 10,
					},
				},
			})
		case "total-acima-limite":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]string{{"b64_json": imgBase64}},
				"usage": map[string]any{
					"input_tokens":  maxTokensFornecedor + 1,
					"output_tokens": 100,
					"input_tokens_details": map[string]int{
						"text_tokens":  10,
						"image_tokens": 0,
					},
				},
			})
		default:
			w.WriteHeader(400)
		}
	}))
	defer srv.Close()

	con, err := novoConector("openai", srv.URL+"/v1", chaveAPITeste, srv.Client())
	if err != nil {
		t.Fatal(err)
	}

	comDet, err := con.gerarImagem(ctx, "chatgpt-image-latest", "com-detalhes")
	if err != nil {
		t.Fatalf("usage com detalhes deveria passar: %v", err)
	}
	if comDet.TokensEntrada != 12 || comDet.TokensEntradaImagem != 340 || comDet.TokensSaida != 1258 || comDet.Imagens != 1 {
		t.Fatalf("usage com detalhes inesperado: %+v", comDet)
	}

	semDet, err := con.gerarImagem(ctx, "gpt-image-1", "sem-detalhes")
	if err != nil {
		t.Fatalf("usage sem detalhes deveria passar: %v", err)
	}
	if semDet.TokensEntrada != 25 || semDet.TokensEntradaImagem != 0 || semDet.TokensSaida != 900 || semDet.Imagens != 1 {
		t.Fatalf("usage sem detalhes inesperado: %+v", semDet)
	}

	soImg, err := con.gerarImagem(ctx, "gpt-image-1", "so-image-tokens")
	if err != nil {
		t.Fatalf("usage só com image_tokens deveria derivar text_tokens: %v", err)
	}
	if soImg.TokensEntrada != 60 || soImg.TokensEntradaImagem != 40 || soImg.TokensSaida != 800 || soImg.Imagens != 1 {
		t.Fatalf("usage só com image_tokens inesperado: %+v", soImg)
	}

	var ef *erroFornecedor
	for _, caso := range []string{"inconsistente-soma", "inconsistente-image-maior", "inconsistente-text-diferente"} {
		if _, err := con.gerarImagem(ctx, "gpt-image-1", caso); !errors.As(err, &ef) || ef.codigo != "fornecedor_erro" ||
			ef.mensagem != "O fornecedor retornou um uso de tokens inconsistente." {
			t.Fatalf("%s esperava fornecedor_erro inconsistente, veio %v", caso, err)
		}
	}
	if _, err := con.gerarImagem(ctx, "gpt-image-1", "detalhes-negativos"); !errors.As(err, &ef) || ef.codigo != "fornecedor_erro" {
		t.Fatalf("detalhes negativos esperava fornecedor_erro, veio %v", err)
	}
	if _, err := con.gerarImagem(ctx, "gpt-image-1", "total-acima-limite"); !errors.As(err, &ef) || ef.codigo != "fornecedor_erro" {
		t.Fatalf("total acima do limite esperava fornecedor_erro, veio %v", err)
	}
}
