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
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]string{{"id": "claude-sonnet-4-5"}},
			})
			return
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

	// OpenAI 401 -> 502 fornecedor_recusou_chave sem vazar chave na resposta nem no log.
	modoOpenAI.Store(1)
	st, body := c.pedirComCorpo("POST", "/api/admin/fornecedores/"+fornOAI.ID+"/ler-modelos", nil)
	if st != 502 || !strings.Contains(string(body), "fornecedor_recusou_chave") {
		t.Fatalf("OpenAI 401 esperava 502 fornecedor_recusou_chave, veio %d (%s)", st, body)
	}
	if bytes.Contains(body, []byte(chaveAPITeste)) {
		t.Fatalf("resposta 401 vazou a chave: %s", body)
	}

	// Anthropic: troca TLS de teste para srvAnthropic -> lista ok, 401 -> recusou_chave, 503/fechado -> indisponivel.
	configurarTLSRedeInterna(api, srvAnthropic)
	if st := c.pedir("POST", "/api/admin/fornecedores/"+fornAnt.ID+"/ler-modelos", nil, &lidos); st != 200 {
		t.Fatalf("ler-modelos Anthropic esperava 200, veio %d", st)
	}
	if apiKeyAnthropic != chaveAPINovaTeste || versaoHeader != versaoAnthropic || len(lidos.Modelos) != 1 || lidos.Modelos[0].Identificador != "claude-sonnet-4-5" {
		t.Fatalf("Anthropic inesperado: key=%q ver=%q modelos=%+v", apiKeyAnthropic, versaoHeader, lidos.Modelos)
	}

	modoAnthropic.Store(1)
	st, body = c.pedirComCorpo("POST", "/api/admin/fornecedores/"+fornAnt.ID+"/ler-modelos", nil)
	if st != 502 || !strings.Contains(string(body), "fornecedor_recusou_chave") || bytes.Contains(body, []byte(chaveAPINovaTeste)) {
		t.Fatalf("Anthropic 401 inesperado: %d (%s)", st, body)
	}

	modoAnthropic.Store(2)
	st, body = c.pedirComCorpo("POST", "/api/admin/fornecedores/"+fornAnt.ID+"/ler-modelos", nil)
	if st != 502 || !strings.Contains(string(body), "fornecedor_indisponivel") {
		t.Fatalf("Anthropic 503 esperava 502 fornecedor_indisponivel, veio %d (%s)", st, body)
	}

	// Servidor fora do ar (fechado) -> 502 fornecedor_indisponivel.
	srvAnthropic.Close()
	st, body = c.pedirComCorpo("POST", "/api/admin/fornecedores/"+fornAnt.ID+"/ler-modelos", nil)
	if st != 502 || !strings.Contains(string(body), "fornecedor_indisponivel") || bytes.Contains(body, []byte(chaveAPINovaTeste)) {
		t.Fatalf("Anthropic fora do ar esperava 502 fornecedor_indisponivel, veio %d (%s)", st, body)
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
	if err != nil || !strings.HasPrefix(riSim.ImagemDataUrl, "data:image/png;base64,") || riSim.Imagens != 1 {
		t.Fatalf("simulado imagem inesperado: %+v err=%v", riSim, err)
	}
	if _, err := sim.gerarTexto(ctx, "simulado-falha", "Oi"); err == nil {
		t.Fatal("modelo simulado com 'falha' deveria falhar no texto")
	}
	if _, err := sim.gerarImagem(ctx, "simulado-imagem-falha", "Oi"); err == nil {
		t.Fatal("modelo simulado com 'falha' deveria falhar na imagem")
	}

	// OpenAI TLS mock para chat/completions e images/generations.
	srvOAI := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/chat/completions":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{{"message": map[string]string{"content": "Sou gpt-4o"}}},
				"usage":   map[string]int{"prompt_tokens": 10, "completion_tokens": 25},
			})
		case "/v1/images/generations":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data":  []map[string]string{{"b64_json": imgBase64}},
				"usage": map[string]int{"input_tokens": 5, "output_tokens": 0},
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
	riOAI, err := conOAI.gerarImagem(ctx, "gpt-image-1", "círculo")
	if err != nil || riOAI.ImagemDataUrl != "data:image/png;base64,"+imgBase64 || riOAI.Imagens != 1 || riOAI.TokensEntrada != 5 {
		t.Fatalf("OpenAI gerarImagem inesperado: %+v err=%v", riOAI, err)
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
}
