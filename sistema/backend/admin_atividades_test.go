package main

import (
	"context"
	"strings"
	"testing"
	"time"
)

func prepararSimuladosCompletos(t *testing.T, c *cliente) (fornID, modTxt, modFalha, modTxtReserva, modImg, modImgReserva string) {
	t.Helper()
	var forn Fornecedor
	if st := c.pedir("POST", "/api/admin/fornecedores", map[string]any{
		"nome":            "Simulado Atividades",
		"tipo":            "simulado",
		"limiteMensalUsd": 100,
		"avisoPercentual": 80,
	}, &forn); st != 201 {
		t.Fatalf("criar fornecedor falhou: %d", st)
	}
	fornID = forn.ID

	criarMod := func(ident, servico string) string {
		var m Modelo
		if st := c.pedir("POST", "/api/admin/fornecedores/"+fornID+"/modelos", map[string]any{
			"identificador": ident,
			"servico":       servico,
			"origem":        "manual",
		}, &m); st != 201 {
			t.Fatalf("criar modelo %s falhou: %d", ident, st)
		}
		return m.ID
	}
	precoTxt := func(modID string, ent, sai float64) {
		if st := c.pedir("POST", "/api/admin/modelos/"+modID+"/precos", map[string]any{
			"unidade": "token_entrada", "precoUsd": ent, "vigenciaInicio": "2026-11-01",
		}, nil); st != 201 {
			t.Fatalf("preço entrada falhou: %d", st)
		}
		if st := c.pedir("POST", "/api/admin/modelos/"+modID+"/precos", map[string]any{
			"unidade": "token_saida", "precoUsd": sai, "vigenciaInicio": "2026-11-01",
		}, nil); st != 201 {
			t.Fatalf("preço saída falhou: %d", st)
		}
	}
	precoImg := func(modID string, val float64) {
		if st := c.pedir("POST", "/api/admin/modelos/"+modID+"/precos", map[string]any{
			"unidade": "imagem", "precoUsd": val, "vigenciaInicio": "2026-11-01",
		}, nil); st != 201 {
			t.Fatalf("preço imagem falhou: %d", st)
		}
	}

	modTxt = criarMod("simulado-texto", "texto")
	precoTxt(modTxt, 1000, 2000) // 12 * 1000/1e6 + 18 * 2000/1e6 = 0.012 + 0.036 = 0.048 USD

	modFalha = criarMod("simulado-falha", "texto")
	precoTxt(modFalha, 1000, 2000)

	modTxtReserva = criarMod("simulado-texto-reserva", "texto")
	precoTxt(modTxtReserva, 500, 1000) // 12*500/1e6 + 18*1000/1e6 = 0.006 + 0.018 = 0.024 USD

	modImg = criarMod("simulado-imagem", "imagem")
	precoImg(modImg, 0.05)

	modImgReserva = criarMod("simulado-imagem-reserva", "imagem")
	precoImg(modImgReserva, 0.08)

	return
}

func TestAC29AC35TestarModeloTextoEImagem(t *testing.T) {
	api := novaAPIAdminTeste(t)
	agoraFixo := time.Date(2026, 11, 25, 14, 0, 0, 0, fusoSaoPaulo)
	api.agora = func() time.Time { return agoraFixo }

	c := novoCliente(t, api)
	entrarComoPaulo(t, c)

	fornID, modTxt, _, _, modImg, _ := prepararSimuladosCompletos(t, c)

	// Modelo sem preço -> 409 sem_preco.
	var modSemPreco Modelo
	c.pedir("POST", "/api/admin/fornecedores/"+fornID+"/modelos", map[string]any{
		"identificador": "simulado-sem-preco", "servico": "texto", "origem": "manual",
	}, &modSemPreco)
	if st, body := c.pedirComCorpo("POST", "/api/admin/modelos/"+modSemPreco.ID+"/testar", nil); st != 409 || !strings.Contains(string(body), "sem_preco") {
		t.Fatalf("sem preço esperava 409 sem_preco, veio %d (%s)", st, body)
	}

	// AC35 + AC29: testar texto ("Oi, qual modelo é você?").
	var resTxt ResultadoTeste
	if st := c.pedir("POST", "/api/admin/modelos/"+modTxt+"/testar", nil, &resTxt); st != 200 {
		t.Fatalf("testar modelo texto falhou: %d", st)
	}
	if resTxt.Servico != "texto" || resTxt.Texto == nil || !strings.Contains(*resTxt.Texto, promptTesteTexto) ||
		resTxt.ImagemDataUrl != nil || resTxt.TokensEntrada != 12 || resTxt.TokensSaida != 18 ||
		resTxt.CustoUsd != 0.048 || !resTxt.Simulado {
		t.Fatalf("resultado teste texto inesperado: %+v", resTxt)
	}

	// AC35 + AC29: testar imagem ("um círculo azul sobre fundo branco").
	var resImg ResultadoTeste
	if st := c.pedir("POST", "/api/admin/modelos/"+modImg+"/testar", nil, &resImg); st != 200 {
		t.Fatalf("testar modelo imagem falhou: %d", st)
	}
	if resImg.Servico != "imagem" || resImg.ImagemDataUrl == nil || !strings.HasPrefix(*resImg.ImagemDataUrl, "data:image/png;base64,") ||
		resImg.Texto != nil || resImg.Imagens != 1 || resImg.CustoUsd != 0.05 || !resImg.Simulado {
		t.Fatalf("resultado teste imagem inesperado: %+v", resImg)
	}

	// Confere os registros gravados em GET /api/admin/execucoes (mais recentes primeiro).
	var execs RespostaExecucoes
	if st := c.pedir("GET", "/api/admin/execucoes?limite=10", nil, &execs); st != 200 || len(execs.Execucoes) != 2 {
		t.Fatalf("execuções inesperadas: %d %+v", st, execs.Execucoes)
	}
	if execs.Execucoes[0].ID != resImg.ExecucaoID || execs.Execucoes[0].Atividade != "teste" || !execs.Execucoes[0].Simulado || execs.Execucoes[0].CustoUsd != 0.05 {
		t.Fatalf("registro da execução de imagem inesperado: %+v", execs.Execucoes[0])
	}
	if execs.Execucoes[1].ID != resTxt.ExecucaoID || execs.Execucoes[1].TokensEntrada != 12 || execs.Execucoes[1].TokensSaida != 18 || execs.Execucoes[1].CustoUsd != 0.048 {
		t.Fatalf("registro da execução de texto inesperado: %+v", execs.Execucoes[1])
	}

	// Modelo inativo -> 409 modelo_inativo.
	c.pedir("PATCH", "/api/admin/modelos/"+modTxt, map[string]any{"ativo": false}, nil)
	if st, body := c.pedirComCorpo("POST", "/api/admin/modelos/"+modTxt+"/testar", nil); st != 409 || !strings.Contains(string(body), "modelo_inativo") {
		t.Fatalf("modelo inativo esperava 409 modelo_inativo, veio %d (%s)", st, body)
	}
}

func TestAC36AtividadesIncompletasERegrasDeRota(t *testing.T) {
	api := novaAPIAdminTeste(t)
	c := novoCliente(t, api)
	entrarComoPaulo(t, c)

	_, modTxt, _, modTxtRes, modImg, _ := prepararSimuladosCompletos(t, c)

	// Inicialmente todas as 6 atividades vêm incompletas.
	var atvs RespostaAtividades
	if st := c.pedir("GET", "/api/admin/atividades", nil, &atvs); st != 200 || len(atvs.Atividades) != 6 {
		t.Fatalf("GET atividades inesperado: %d len=%d", st, len(atvs.Atividades))
	}
	if atvs.Atividades[0].Pronta || len(atvs.Atividades[0].Falta) != 2 {
		t.Fatalf("atividade inicial deveria faltar principal e reserva: %+v", atvs.Atividades[0])
	}

	// Serviço incompatível -> 400 servico_incompativel; modelos iguais -> 400 modelos_iguais.
	if st, body := c.pedirComCorpo("PUT", "/api/admin/atividades/texto", map[string]any{
		"modeloPrincipalId": modImg, "modeloReservaId": modTxtRes,
	}); st != 400 || !strings.Contains(string(body), "servico_incompativel") {
		t.Fatalf("esperava 400 servico_incompativel, veio %d (%s)", st, body)
	}
	if st, body := c.pedirComCorpo("PUT", "/api/admin/atividades/texto", map[string]any{
		"modeloPrincipalId": modTxt, "modeloReservaId": modTxt,
	}); st != 400 || !strings.Contains(string(body), "modelos_iguais") {
		t.Fatalf("esperava 400 modelos_iguais, veio %d (%s)", st, body)
	}

	// Atividade só com principal (sem reserva) não roda: 409 atividade_incompleta dizendo o que falta.
	var rota RotaAtividade
	if st := c.pedir("PUT", "/api/admin/atividades/texto", map[string]any{
		"modeloPrincipalId": modTxt, "modeloReservaId": nil,
	}, &rota); st != 200 || rota.Pronta || len(rota.Falta) != 1 || rota.Falta[0] != "modelo reserva" {
		t.Fatalf("rota sem reserva inesperada: %d %+v", st, rota)
	}
	st, body := c.pedirComCorpo("POST", "/api/admin/atividades/texto/testar", nil)
	if st != 409 || !strings.Contains(string(body), "atividade_incompleta") || !strings.Contains(string(body), "modelo reserva") {
		t.Fatalf("esperava 409 atividade_incompleta citando reserva, veio %d (%s)", st, body)
	}
}

func TestAC33FallbackPrincipalParaReserva(t *testing.T) {
	api := novaAPIAdminTeste(t)
	c := novoCliente(t, api)
	entrarComoPaulo(t, c)

	_, _, modFalha, modTxtReserva, _, _ := prepararSimuladosCompletos(t, c)

	if st := c.pedir("PUT", "/api/admin/atividades/plano", map[string]any{
		"modeloPrincipalId": modFalha, "modeloReservaId": modTxtReserva,
	}, nil); st != 200 {
		t.Fatalf("configurar rota plano falhou: %d", st)
	}

	var res ResultadoAtividade
	if st := c.pedir("POST", "/api/admin/atividades/plano/testar", nil, &res); st != 200 {
		t.Fatalf("testar plano esperava 200 via reserva, veio %d", st)
	}
	if !res.AtendidoPelaReserva || res.AtendidoPor.ID != modTxtReserva || len(res.Tentativas) != 2 ||
		res.Tentativas[0].Status != "falha" || res.Tentativas[1].Status != "ok" || !res.Simulado {
		t.Fatalf("resultado fallback inesperado: %+v", res)
	}

	// Confere os dois registros em ia_execucoes com a mesma correlação.
	var execs RespostaExecucoes
	if st := c.pedir("GET", "/api/admin/execucoes", nil, &execs); st != 200 || len(execs.Execucoes) != 2 {
		t.Fatalf("execuções esperadas = 2, veio %d (%+v)", st, execs.Execucoes)
	}
	t2, t1 := execs.Execucoes[0], execs.Execucoes[1]
	if t1.CorrelacaoID != res.CorrelacaoID || t2.CorrelacaoID != res.CorrelacaoID {
		t.Fatalf("correlação diferente: %+v %+v", t1, t2)
	}
	if t1.Status != "falha" || t1.Erro == nil || t1.Tentativa != 1 || t1.AtendidoPelaReserva {
		t.Fatalf("tentativa 1 inesperada: %+v", t1)
	}
	if t2.Status != "ok" || t2.Tentativa != 2 || !t2.AtendidoPelaReserva || t2.ModeloIdentificador != "simulado-texto-reserva" {
		t.Fatalf("tentativa 2 inesperada: %+v", t2)
	}
}

func TestAC34LimitesAvisoPuloParadaELimiteDoModelo(t *testing.T) {
	api := novaAPIAdminTeste(t)
	agoraFixo := time.Date(2026, 11, 25, 14, 0, 0, 0, fusoSaoPaulo)
	api.agora = func() time.Time { return agoraFixo }

	c := novoCliente(t, api)
	entrarComoPaulo(t, c)

	fornID, modTxt, _, modTxtReserva, _, _ := prepararSimuladosCompletos(t, c)

	// Limite do fornecedor = 100.00 (aviso 80%).
	// Principal (modTxt) ganha limite próprio = 10.00 (aviso 80%).
	// Reserva (modTxtReserva) fica sem limite próprio (usa os 100.00 do fornecedor).
	if st := c.pedir("PATCH", "/api/admin/modelos/"+modTxt, map[string]any{
		"limiteMensalUsd": 10.00, "avisoPercentual": 80,
	}, nil); st != 200 {
		t.Fatalf("PATCH modTxt falhou: %d", st)
	}

	if st := c.pedir("PUT", "/api/admin/atividades/pauta", map[string]any{
		"modeloPrincipalId": modTxt, "modeloReservaId": modTxtReserva,
	}, nil); st != 200 {
		t.Fatalf("PUT pauta falhou: %d", st)
	}

	ctx := context.Background()
	// 1. Gasta 8.00 em modTxt (>= 80% de 10.00) -> gera aviso em GET /api/admin/avisos, mas ainda roda.
	if _, err := api.banco.Exec(ctx, `
		INSERT INTO ia_execucoes (correlacao_id, atividade, fornecedor_id, modelo_id, fornecedor_nome, modelo_identificador, status, custo_usd, criado_em)
		VALUES (gen_random_uuid(), 'teste', $1, $2, 'Simulado Atividades', 'simulado-texto', 'ok', 8.00, $3)`,
		fornID, modTxt, agoraFixo); err != nil {
		t.Fatal(err)
	}
	var avisos RespostaAvisos
	if st := c.pedir("GET", "/api/admin/avisos", nil, &avisos); st != 200 || len(avisos.Avisos) != 1 || avisos.Avisos[0].Alvo != "modelo" || avisos.Avisos[0].Situacao != "aviso" {
		t.Fatalf("esperava 1 aviso de modelo, veio %d %+v", st, avisos.Avisos)
	}
	var resOk ResultadoAtividade
	if st := c.pedir("POST", "/api/admin/atividades/pauta/testar", nil, &resOk); st != 200 || resOk.AtendidoPelaReserva {
		t.Fatalf("abaixo do limite do modelo deveria atender pela principal, veio %d %+v", st, resOk)
	}

	// 2. Soma +2.00 -> modTxt atinge 10.00+ (limite próprio = 10, bem abaixo dos 100 do fornecedor!).
	//    Testar modelo recusa (409 limite_atingido); atividade pula para a reserva com pulado_limite.
	if _, err := api.banco.Exec(ctx, `
		INSERT INTO ia_execucoes (correlacao_id, atividade, fornecedor_id, modelo_id, fornecedor_nome, modelo_identificador, status, custo_usd, criado_em)
		VALUES (gen_random_uuid(), 'teste', $1, $2, 'Simulado Atividades', 'simulado-texto', 'ok', 2.00, $3)`,
		fornID, modTxt, agoraFixo); err != nil {
		t.Fatal(err)
	}
	if st, body := c.pedirComCorpo("POST", "/api/admin/modelos/"+modTxt+"/testar", nil); st != 409 || !strings.Contains(string(body), "limite_atingido") {
		t.Fatalf("testar modelo no limite esperava 409 limite_atingido, veio %d (%s)", st, body)
	}
	var resPulo ResultadoAtividade
	if st := c.pedir("POST", "/api/admin/atividades/pauta/testar", nil, &resPulo); st != 200 {
		t.Fatalf("pauta com principal no limite esperava 200 pela reserva, veio %d", st)
	}
	if !resPulo.AtendidoPelaReserva || resPulo.Tentativas[0].Status != "pulado_limite" || resPulo.Tentativas[1].Status != "ok" {
		t.Fatalf("pulo para reserva inesperado: %+v", resPulo)
	}

	// 3. Limite do fornecedor (100.00) também estoura -> reserva (sem limite próprio) é pulada -> para com 409 limite_atingido e gera aviso de fornecedor.
	if _, err := api.banco.Exec(ctx, `
		INSERT INTO ia_execucoes (correlacao_id, atividade, fornecedor_id, modelo_id, fornecedor_nome, modelo_identificador, status, custo_usd, criado_em)
		VALUES (gen_random_uuid(), 'teste', $1, $2, 'Simulado Atividades', 'simulado-texto-reserva', 'ok', 95.00, $3)`,
		fornID, modTxtReserva, agoraFixo); err != nil {
		t.Fatal(err)
	}
	if st, body := c.pedirComCorpo("POST", "/api/admin/atividades/pauta/testar", nil); st != 409 || !strings.Contains(string(body), "limite_atingido") {
		t.Fatalf("principal e reserva no limite esperava 409 limite_atingido, veio %d (%s)", st, body)
	}
	if st := c.pedir("GET", "/api/admin/avisos", nil, &avisos); st != 200 || len(avisos.Avisos) != 2 {
		t.Fatalf("esperava avisos para fornecedor e modelo, veio %d %+v", st, avisos.Avisos)
	}

	// 4. Modelo com limite próprio MAIOR (200.00) continua rodando mesmo quando o fornecedor (100.00) já estourou ("limite do modelo vale no lugar do fornecedor").
	var modLimiteMaior Modelo
	c.pedir("POST", "/api/admin/fornecedores/"+fornID+"/modelos", map[string]any{
		"identificador": "simulado-limite-alto", "servico": "texto", "origem": "manual", "limiteMensalUsd": 200.00,
	}, &modLimiteMaior)
	c.pedir("POST", "/api/admin/modelos/"+modLimiteMaior.ID+"/precos", map[string]any{
		"unidade": "token_entrada", "precoUsd": 100, "vigenciaInicio": "2026-11-01",
	}, nil)
	c.pedir("POST", "/api/admin/modelos/"+modLimiteMaior.ID+"/precos", map[string]any{
		"unidade": "token_saida", "precoUsd": 100, "vigenciaInicio": "2026-11-01",
	}, nil)
	if st := c.pedir("POST", "/api/admin/modelos/"+modLimiteMaior.ID+"/testar", nil, nil); st != 200 {
		t.Fatalf("modelo com limite próprio maior que o do fornecedor deveria rodar, veio %d", st)
	}
}

func TestAC30UsoSoDaCasaERotas403S3(t *testing.T) {
	api := novaAPIAdminTeste(t)
	c := novoCliente(t, api)
	entrarComoPaulo(t, c)

	// Cria um fornecedor só da casa (principal) e um para todas (reserva).
	var fornCasa, fornTodas Fornecedor
	c.pedir("POST", "/api/admin/fornecedores", map[string]any{
		"nome": "Simulado Casa", "tipo": "simulado", "usoPermitido": "casa", "limiteMensalUsd": 50,
	}, &fornCasa)
	c.pedir("POST", "/api/admin/fornecedores", map[string]any{
		"nome": "Simulado Todas", "tipo": "simulado", "usoPermitido": "todas", "limiteMensalUsd": 50,
	}, &fornTodas)

	criarComPreco := func(fornID, ident string) string {
		var m Modelo
		c.pedir("POST", "/api/admin/fornecedores/"+fornID+"/modelos", map[string]any{
			"identificador": ident, "servico": "texto", "origem": "manual",
		}, &m)
		c.pedir("POST", "/api/admin/modelos/"+m.ID+"/precos", map[string]any{
			"unidade": "token_entrada", "precoUsd": 10, "vigenciaInicio": "2026-11-01",
		}, nil)
		c.pedir("POST", "/api/admin/modelos/"+m.ID+"/precos", map[string]any{
			"unidade": "token_saida", "precoUsd": 10, "vigenciaInicio": "2026-11-01",
		}, nil)
		return m.ID
	}
	modCasa := criarComPreco(fornCasa.ID, "simulado-casa")
	modTodas := criarComPreco(fornTodas.ID, "simulado-todas")

	c.pedir("PUT", "/api/admin/atividades/chat_cmo", map[string]any{
		"modeloPrincipalId": modCasa, "modeloReservaId": modTodas,
	}, nil)

	var lauraID, alheiaID string
	if err := api.banco.QueryRow(context.Background(), `SELECT id::text FROM organizacoes WHERE nome = 'Laura Growth'`).Scan(&lauraID); err != nil {
		t.Fatal(err)
	}
	if err := api.banco.QueryRow(context.Background(), `SELECT id::text FROM organizacoes WHERE nome = 'Clínica Alheia'`).Scan(&alheiaID); err != nil {
		t.Fatal(err)
	}

	// Organização da casa usa o fornecedor "casa" normalmente.
	resCasa, st, _, _ := api.executarAtividade(context.Background(), "chat_cmo", lauraID, "", "Oi")
	if st != 0 || resCasa.AtendidoPelaReserva || resCasa.AtendidoPor.ID != modCasa {
		t.Fatalf("organização da casa deveria usar principal 'casa', veio st=%d %+v", st, resCasa)
	}

	// Organização de fora da casa NUNCA usa fornecedor "casa": trata como indisponível e segue para a reserva.
	resFora, st, _, _ := api.executarAtividade(context.Background(), "chat_cmo", alheiaID, "", "Oi")
	if st != 0 || !resFora.AtendidoPelaReserva || resFora.AtendidoPor.ID != modTodas || resFora.Tentativas[0].Status != "falha" {
		t.Fatalf("organização fora da casa deveria cair na reserva, veio st=%d %+v", st, resFora)
	}

	// Se reserva também for "casa", recusa para organização de fora da casa (502 atividade_falhou).
	c.pedir("PUT", "/api/admin/atividades/revisao", map[string]any{
		"modeloPrincipalId": modCasa, "modeloReservaId": criarComPreco(fornCasa.ID, "simulado-casa-2"),
	}, nil)
	_, st, cod, _ := api.executarAtividade(context.Background(), "revisao", alheiaID, "", "Oi")
	if st != 502 || cod != "atividade_falhou" {
		t.Fatalf("dois modelos 'casa' para org externa esperava 502 atividade_falhou, veio %d %s", st, cod)
	}

	// Pessoa comum recebe 403 em todas as novas rotas da S3.
	criarUsuarioComum(t, api.banco, "comum-s3@exemplo.com")
	comum := novoCliente(t, api)
	comum.pedir("POST", "/api/sessao", map[string]string{"email": "comum-s3@exemplo.com", "senha": senhaTeste}, nil)
	for _, rt := range []struct{ m, p string }{
		{"POST", "/api/admin/modelos/" + modCasa + "/testar"},
		{"GET", "/api/admin/atividades"},
		{"PUT", "/api/admin/atividades/texto"},
		{"POST", "/api/admin/atividades/texto/testar"},
		{"GET", "/api/admin/execucoes"},
		{"GET", "/api/admin/avisos"},
	} {
		if st, body := comum.pedirComCorpo(rt.m, rt.p, map[string]any{}); st != 403 || !strings.Contains(string(body), "sem_permissao") {
			t.Fatalf("%s %s para comum esperava 403 sem_permissao, veio %d (%s)", rt.m, rt.p, st, body)
		}
	}
}
