package main

// Testes da etapa 2 e 2C (kit de marca / DNA da marca). Cobrem isolamento,
// validação dos novos campos (C1), origens e confirmação (C3), versões e histórico (C4),
// concorrência por versaoBase (C5), migração 0005 preservando dados e arquivos (C6)
// e autorização por alçada e organização (C7).

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// PNG de 1x1 pixel.
var pngMinimo, _ = base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==")

func (c *cliente) enviar(caminho, nome string, conteudo []byte, resposta any) int {
	c.t.Helper()
	var corpo bytes.Buffer
	mw := multipart.NewWriter(&corpo)
	fw, _ := mw.CreateFormFile("arquivo", nome)
	_, _ = fw.Write(conteudo)
	_ = mw.Close()
	req, _ := http.NewRequest("POST", c.srv.URL+caminho, &corpo)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res, err := c.c.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	if resposta != nil && res.StatusCode < 300 {
		if err := jsonDecode(res.Body, resposta); err != nil {
			c.t.Fatal(err)
		}
	}
	return res.StatusCode
}

func (c *cliente) baixar(caminho string) (int, []byte, http.Header) {
	c.t.Helper()
	res, err := c.c.Get(c.srv.URL + caminho)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, b, res.Header
}

func entrarComoPaulo(t *testing.T, c *cliente) Sessao {
	t.Helper()
	var s Sessao
	if st := c.pedir("POST", "/api/sessao", map[string]string{"email": "paulo@exemplo.com", "senha": senhaTeste}, &s); st != 200 {
		t.Fatalf("login falhou: %d", st)
	}
	return s
}

func TestEtapa2KitDeMarca(t *testing.T) {
	api := NovaAPI(prepararBanco(t), false, nil)
	c := novoCliente(t, api)
	s := entrarComoPaulo(t, c)
	growth, hub := s.Organizacoes[0], s.Organizacoes[1]

	// Kit vazio no começo.
	var r RespostaKit
	if st := c.pedir("GET", "/api/kit", nil, &r); st != 200 || r.Kit.OrganizacaoID != growth.ID || len(r.Kit.Pilares) != 0 || r.Kit.AtualizadoEm != nil || r.Versao != 0 {
		t.Fatalf("kit inicial inesperado: %d %+v", st, r)
	}

	// Pronto quando: o kit está cadastrado, com 3 a 5 pilares.
	kit := map[string]any{
		"organizacaoId": growth.ID, "site": "https://laura.example", "tomDeVoz": "Próximo e direto",
		"pilares": []string{" Constância ", "Bastidores", "", "IA na prática"}, "cores": []string{"#F26B3A", "#1f2430"},
	}
	if st := c.pedir("PUT", "/api/kit", kit, &r); st != 200 {
		t.Fatalf("salvar kit esperava 200, veio %d", st)
	}
	if len(r.Kit.Pilares) != 3 || r.Kit.Pilares[0] != "Constância" || r.Kit.Cores[0] != "#f26b3a" || r.Kit.AtualizadoEm == nil || r.Versao != 1 {
		t.Fatalf("kit salvo inesperado: %+v", r)
	}

	// Regras: no máximo 5 pilares; site precisa ser endereço completo.
	ruim := map[string]any{"organizacaoId": growth.ID, "versaoBase": r.Versao, "pilares": []string{"a", "b", "c", "d", "e", "f"}}
	if st := c.pedir("PUT", "/api/kit", ruim, nil); st != 400 {
		t.Fatalf("6 pilares esperava 400, veio %d", st)
	}
	if st := c.pedir("PUT", "/api/kit", map[string]any{"organizacaoId": growth.ID, "versaoBase": r.Versao, "site": "javascript:alert(1)"}, nil); st != 400 {
		t.Fatalf("site inválido esperava 400, veio %d", st)
	}

	// Logo: PNG aceito; texto disfarçado de imagem recusado.
	var logo ArquivoMarca
	if st := c.enviar("/api/kit/arquivos?tipo=logo&organizacaoId="+growth.ID, "logo.png", pngMinimo, &logo); st != 201 || logo.Mime != "image/png" {
		t.Fatalf("envio do logo esperava 201, veio %d %+v", st, logo)
	}
	if st := c.enviar("/api/kit/arquivos?tipo=logo&organizacaoId="+growth.ID, "falso.png", []byte("<svg onload=alert(1)>"), nil); st != 415 {
		t.Fatalf("arquivo falso esperava 415, veio %d", st)
	}
	st, corpo, cab := c.baixar("/api/kit/arquivos/" + logo.ID)
	if st != 200 || !bytes.Equal(corpo, pngMinimo) || cab.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("baixar logo falhou: %d", st)
	}
	// Um logo novo substitui o anterior.
	var logo2 ArquivoMarca
	c.enviar("/api/kit/arquivos?tipo=logo&organizacaoId="+growth.ID, "logo2.png", pngMinimo, &logo2)
	c.pedir("GET", "/api/kit", nil, &r)
	if len(r.Arquivos) != 1 || r.Arquivos[0].ID != logo2.ID {
		t.Fatalf("esperava só o logo novo: %+v", r.Arquivos)
	}

	// Pronto quando: o kit de uma organização não aparece na outra.
	c.pedir("PUT", "/api/sessao/organizacao", map[string]string{"organizacaoId": hub.ID}, nil)
	if st := c.pedir("GET", "/api/kit", nil, &r); st != 200 || r.Kit.OrganizacaoID != hub.ID || len(r.Kit.Pilares) != 0 || len(r.Arquivos) != 0 {
		t.Fatalf("Laura Hub enxergou dados da Laura Growth: %+v", r)
	}
	if st, _, _ := c.baixar("/api/kit/arquivos/" + logo2.ID); st != 404 {
		t.Fatalf("logo de outra organização esperava 404, veio %d", st)
	}
	// Tela desatualizada (mostrando a Laura Growth) não grava no kit da Laura Hub.
	if st := c.pedir("PUT", "/api/kit", kit, nil); st != 409 {
		t.Fatalf("gravação com organização trocada esperava 409, veio %d", st)
	}
	if st := c.pedir("DELETE", "/api/kit/arquivos/"+logo2.ID+"?organizacaoId="+hub.ID, nil, nil); st != 404 {
		t.Fatalf("remover arquivo de outra organização esperava 404, veio %d", st)
	}

	// De volta à Laura Growth, o kit e o logo continuam lá; remover funciona.
	c.pedir("PUT", "/api/sessao/organizacao", map[string]string{"organizacaoId": growth.ID}, nil)
	c.pedir("GET", "/api/kit", nil, &r)
	if len(r.Kit.Pilares) != 3 || len(r.Arquivos) != 1 {
		t.Fatalf("kit da Laura Growth sumiu: %+v", r)
	}
	if st := c.pedir("DELETE", "/api/kit/arquivos/"+logo2.ID+"?organizacaoId="+growth.ID, nil, nil); st != 204 {
		t.Fatalf("remover logo esperava 204, veio %d", st)
	}
}

func TestEtapa2CC1CamposAmpliadosEValidacao(t *testing.T) {
	api := NovaAPI(prepararBanco(t), false, nil)
	c := novoCliente(t, api)
	s := entrarComoPaulo(t, c)
	growth := s.Organizacoes[0]

	produtos := []ProdutoKit{
		{Nome: " Diagnóstico de marca ", Prioritario: false},
		{Nome: "Gestão mensal", Prioritario: false},
	}
	canais := []string{"instagram", "linkedin", "google_meu_negocio", "site_blog", "whatsapp", "email", "linkedin"}
	pedido := map[string]any{
		"organizacaoId":   growth.ID,
		"versaoBase":      0,
		"produtos":        produtos,
		"diferencial":     " Atendimento com equipe sênior ",
		"modeloNegocio":   "outro",
		"modeloOutro":     " Franquia regional ",
		"regioes":         " Brasil e Portugal (PT-BR) ",
		"comoCompram":     " Indicação e diagnóstico gratuito ",
		"maiorDesafio":    " Reduzir ciclo de decisão ",
		"objetivo90Dias":  " Gerar 30 reuniões qualificadas ",
		"canais":          canais,
		"canaisOutro":     " Eventos locais ",
		"restricoesSetor": " Não prometer retorno financeiro garantido ",
	}
	var resp RespostaKit
	if st := c.pedir("PUT", "/api/kit", pedido, &resp); st != 200 {
		t.Fatalf("PUT campos novos esperava 200, veio %d", st)
	}

	var lido RespostaKit
	if st := c.pedir("GET", "/api/kit", nil, &lido); st != 200 {
		t.Fatalf("GET /api/kit esperava 200, veio %d", st)
	}
	k := lido.Kit
	if len(k.Produtos) != 2 || k.Produtos[0].Nome != "Diagnóstico de marca" || k.Produtos[0].Prioritario {
		t.Fatalf("produtos inesperados: %+v", k.Produtos)
	}
	if k.Diferencial != "Atendimento com equipe sênior" ||
		k.ModeloNegocio != "outro" || k.ModeloOutro != "Franquia regional" ||
		k.Regioes != "Brasil e Portugal (PT-BR)" ||
		k.ComoCompram != "Indicação e diagnóstico gratuito" ||
		k.MaiorDesafio != "Reduzir ciclo de decisão" ||
		k.Objetivo90Dias != "Gerar 30 reuniões qualificadas" ||
		k.CanaisOutro != "Eventos locais" ||
		k.RestricoesSetor != "Não prometer retorno financeiro garantido" {
		t.Fatalf("campos de texto não conferem: %+v", k)
	}
	canaisEsperados := []string{"instagram", "linkedin", "google_meu_negocio", "site_blog", "whatsapp", "email"}
	if !slices.Equal(k.Canais, canaisEsperados) {
		t.Fatalf("canais esperados %v, veio %v", canaisEsperados, k.Canais)
	}

	if st := c.pedir("PUT", "/api/kit", map[string]any{
		"organizacaoId": growth.ID, "versaoBase": lido.Versao, "modeloNegocio": "b2b", "modeloOutro": "Texto ignorado",
	}, &lido); st != 200 || lido.Kit.ModeloOutro != "" {
		t.Fatalf("modeloNegocio != outro deveria limpar modeloOutro, veio %+v", lido.Kit)
	}

	// Rejeições 400 dados_invalidos: modeloNegocio inválido, canal inválido, textos/produtos acima do limite.
	var produtosDemais []ProdutoKit
	for i := 0; i < 21; i++ {
		produtosDemais = append(produtosDemais, ProdutoKit{Nome: fmt.Sprintf("Produto %d", i)})
	}
	casosInvalidos := []struct {
		nome  string
		corpo map[string]any
	}{
		{"modelo inválido", map[string]any{"organizacaoId": growth.ID, "versaoBase": lido.Versao, "modeloNegocio": "b2g"}},
		{"canal inválido", map[string]any{"organizacaoId": growth.ID, "versaoBase": lido.Versao, "canais": []string{"instagram", "orkut"}}},
		{"diferencial acima de 5000", map[string]any{"organizacaoId": growth.ID, "versaoBase": lido.Versao, "diferencial": strings.Repeat("a", 5001)}},
		{"restricoesSetor acima de 5000", map[string]any{"organizacaoId": growth.ID, "versaoBase": lido.Versao, "restricoesSetor": strings.Repeat("b", 5001)}},
		{"mais de 20 produtos", map[string]any{"organizacaoId": growth.ID, "versaoBase": lido.Versao, "produtos": produtosDemais}},
		{"nome de produto acima de 500", map[string]any{"organizacaoId": growth.ID, "versaoBase": lido.Versao, "produtos": []ProdutoKit{{Nome: strings.Repeat("p", 501)}}}},
		{"versaoBase negativa", map[string]any{"organizacaoId": growth.ID, "versaoBase": -1, "diferencial": "ok"}},
	}
	for _, tc := range casosInvalidos {
		st, body := c.pedirComCorpo("PUT", "/api/kit", tc.corpo)
		if st != 400 || !strings.Contains(string(body), `"dados_invalidos"`) {
			t.Fatalf("%s: esperava 400 dados_invalidos, veio %d (%s)", tc.nome, st, string(body))
		}
	}
}

func TestEtapa2CC3OrigensEConfirmacao(t *testing.T) {
	api := NovaAPI(prepararBanco(t), false, nil)
	c := novoCliente(t, api)
	s := entrarComoPaulo(t, c)
	growth := s.Organizacoes[0]

	// Gravação humana inicial marca apenas o campo alterado como 'confirmado'.
	var r RespostaKit
	if st := c.pedir("PUT", "/api/kit", map[string]any{
		"organizacaoId": growth.ID,
		"versaoBase":    0,
		"publico":       "Empresas B2B de serviços",
	}, &r); st != 200 {
		t.Fatalf("PUT inicial falhou: %d", st)
	}
	if len(r.Origens) != 1 || r.Origens["publico"].Origem != "confirmado" || r.Origens["publico"].DefinidoPor == nil || *r.Origens["publico"].DefinidoPor != "Paulo" {
		t.Fatalf("origens após PUT inicial inesperadas: %+v", r.Origens)
	}
	if _, existe := r.Origens["tomDeVoz"]; existe {
		t.Fatalf("campo vazio não deveria ter origem: %+v", r.Origens["tomDeVoz"])
	}

	// Não há rota HTTP pública para marcar sugestão da IA.
	if st, _ := c.pedirComCorpo("POST", "/api/kit/sugerir", map[string]any{"tomDeVoz": "Teste"}); st != 404 && st != 405 {
		t.Fatalf("não deve haver rota pública de sugestão, veio %d", st)
	}

	// gravarSugestoesIA grava valores, origens sugerido_ia (definidoPor null) e versão autor_tipo 'ia'.
	resIA, err := api.gravarSugestoesIA(context.Background(), growth.ID, map[string]any{
		"tomDeVoz":      "Direto e acolhedor",
		"diferencial":   "Entrega em 48 horas",
		"modeloNegocio": "outro",
		"modeloOutro":   "Assinatura híbrida",
		"canais":        []string{"instagram", "linkedin"},
		"canaisOutro":   "Podcast",
		"produtos":      []ProdutoKit{{Nome: "Plano Pro", Prioritario: true}},
	})
	if err != nil {
		t.Fatalf("gravarSugestoesIA falhou: %v", err)
	}
	if len(resIA.Aplicados) != 5 || len(resIA.Ignorados) != 0 {
		t.Fatalf("retorno gravarSugestoesIA inesperado: %+v", resIA)
	}

	if st := c.pedir("GET", "/api/kit", nil, &r); st != 200 {
		t.Fatalf("GET após IA falhou: %d", st)
	}
	if r.Origens["publico"].Origem != "confirmado" {
		t.Fatalf("campo confirmado anterior deveria permanecer confirmado: %+v", r.Origens["publico"])
	}
	for _, campoIA := range []string{"tomDeVoz", "diferencial", "modeloNegocio", "canais", "produtos"} {
		o, ok := r.Origens[campoIA]
		if !ok || o.Origem != "sugerido_ia" || o.DefinidoPor != nil || o.DefinidoEm.IsZero() {
			t.Fatalf("origem IA inesperada para %s: %+v (ok=%v)", campoIA, o, ok)
		}
	}

	// Edição humana altera só 'diferencial' e 'modeloOutro': só eles viram 'confirmado'; os demais continuam 'sugerido_ia'.
	if st := c.pedir("PUT", "/api/kit", map[string]any{
		"organizacaoId": growth.ID,
		"versaoBase":    r.Versao,
		"publico":       r.Kit.Publico,
		"tomDeVoz":      r.Kit.TomDeVoz,
		"diferencial":   "Entrega em 24 horas",
		"modeloNegocio": r.Kit.ModeloNegocio,
		"modeloOutro":   "Consultoria contínua",
		"canais":        r.Kit.Canais,
		"canaisOutro":   r.Kit.CanaisOutro,
		"produtos":      r.Kit.Produtos,
	}, &r); st != 200 {
		t.Fatalf("PUT parcial sobre sugestões falhou: %d", st)
	}
	if r.Origens["diferencial"].Origem != "confirmado" || *r.Origens["diferencial"].DefinidoPor != "Paulo" {
		t.Fatalf("diferencial editado deveria virar confirmado: %+v", r.Origens["diferencial"])
	}
	if r.Origens["modeloNegocio"].Origem != "confirmado" || *r.Origens["modeloNegocio"].DefinidoPor != "Paulo" {
		t.Fatalf("modeloOutro editado deveria confirmar modeloNegocio: %+v", r.Origens["modeloNegocio"])
	}
	if r.Origens["tomDeVoz"].Origem != "sugerido_ia" || r.Origens["canais"].Origem != "sugerido_ia" || r.Origens["produtos"].Origem != "sugerido_ia" {
		t.Fatalf("campos não alterados não deveriam perder sugerido_ia: %+v", r.Origens)
	}

	// POST /api/kit/confirmar transforma sugestões escolhidas em confirmado e registra versão.
	if st := c.pedir("POST", "/api/kit/confirmar", map[string]any{
		"organizacaoId": growth.ID,
		"campos":        []string{"tomDeVoz", "produtos"},
		"versaoBase":    r.Versao,
	}, &r); st != 200 {
		t.Fatalf("POST /api/kit/confirmar falhou: %d", st)
	}
	for _, confirmado := range []string{"tomDeVoz", "produtos"} {
		o := r.Origens[confirmado]
		if o.Origem != "confirmado" || o.DefinidoPor == nil || *o.DefinidoPor != "Paulo" || o.DefinidoEm.IsZero() {
			t.Fatalf("%s não foi confirmado: %+v", confirmado, o)
		}
	}
	if r.Origens["canais"].Origem != "sugerido_ia" {
		t.Fatalf("canais não confirmado deveria continuar sugerido_ia: %+v", r.Origens["canais"])
	}

	// Confirmar com lista vazia ou campo inválido -> 400 dados_invalidos.
	if st, body := c.pedirComCorpo("POST", "/api/kit/confirmar", map[string]any{"organizacaoId": growth.ID, "campos": []string{}, "versaoBase": r.Versao}); st != 400 || !strings.Contains(string(body), `"dados_invalidos"`) {
		t.Fatalf("confirmar vazio esperava 400 dados_invalidos, veio %d (%s)", st, string(body))
	}
	if st, body := c.pedirComCorpo("POST", "/api/kit/confirmar", map[string]any{"organizacaoId": growth.ID, "campos": []string{"campoInexistente"}, "versaoBase": r.Versao}); st != 400 || !strings.Contains(string(body), `"dados_invalidos"`) {
		t.Fatalf("confirmar campo inválido esperava 400 dados_invalidos, veio %d (%s)", st, string(body))
	}
}

func TestEtapa2CC4VersoesEHistorico(t *testing.T) {
	api := NovaAPI(prepararBanco(t), false, nil)
	c := novoCliente(t, api)
	s := entrarComoPaulo(t, c)
	growth := s.Organizacoes[0]

	// Versão 1: PUT
	var r RespostaKit
	if st := c.pedir("PUT", "/api/kit", map[string]any{
		"organizacaoId": growth.ID, "versaoBase": 0, "tomDeVoz": "Inicial",
	}, &r); st != 200 || r.Versao != 1 {
		t.Fatalf("v1 falhou: %d %+v", st, r.Versao)
	}
	// Versão 2: IA
	if _, err := api.gravarSugestoesIA(context.Background(), growth.ID, map[string]any{"diferencial": "IA sugere"}); err != nil {
		t.Fatalf("v2 IA falhou: %v", err)
	}
	c.pedir("GET", "/api/kit", nil, &r)
	if r.Versao != 2 {
		t.Fatalf("esperava versão 2, veio %d", r.Versao)
	}
	// Versão 3: confirmar
	if st := c.pedir("POST", "/api/kit/confirmar", map[string]any{"organizacaoId": growth.ID, "campos": []string{"diferencial"}, "versaoBase": 2}, &r); st != 200 || r.Versao != 3 {
		t.Fatalf("v3 confirmar falhou: %d %+v", st, r.Versao)
	}
	// Versão 4: upload logo
	var logo ArquivoMarca
	if st := c.enviar("/api/kit/arquivos?tipo=logo&organizacaoId="+growth.ID, "marca.png", pngMinimo, &logo); st != 201 {
		t.Fatalf("v4 upload logo falhou: %d", st)
	}
	// Versão 5: upload referência
	var ref ArquivoMarca
	if st := c.enviar("/api/kit/arquivos?tipo=referencia&organizacaoId="+growth.ID, "manual.png", pngMinimo, &ref); st != 201 {
		t.Fatalf("v5 upload ref falhou: %d", st)
	}
	// Versão 6: remover referência
	if st := c.pedir("DELETE", "/api/kit/arquivos/"+ref.ID+"?organizacaoId="+growth.ID, nil, nil); st != 204 {
		t.Fatalf("v6 remover ref falhou: %d", st)
	}

	// Lista de versões (mais recente primeiro) e leitura de versão antiga.
	var lista RespostaVersoesKit
	if st := c.pedir("GET", "/api/kit/versoes", nil, &lista); st != 200 || lista.Total != 6 || len(lista.Versoes) != 6 {
		t.Fatalf("lista de versões inesperada: %d %+v", st, lista)
	}
	if lista.Versoes[0].Numero != 6 || !slices.Equal(lista.Versoes[0].CamposAlterados, []string{"arquivos:manual.png"}) ||
		lista.Versoes[1].Numero != 5 || !slices.Equal(lista.Versoes[1].CamposAlterados, []string{"arquivos:manual.png"}) ||
		lista.Versoes[2].Numero != 4 || !slices.Equal(lista.Versoes[2].CamposAlterados, []string{"logo:marca.png"}) ||
		lista.Versoes[3].Numero != 3 || !slices.Equal(lista.Versoes[3].CamposAlterados, []string{"diferencial"}) ||
		lista.Versoes[4].Numero != 2 || lista.Versoes[4].AutorTipo != "ia" || lista.Versoes[4].AutorNome != nil ||
		lista.Versoes[5].Numero != 1 || lista.Versoes[5].AutorTipo != "pessoa" || *lista.Versoes[5].AutorNome != "Paulo" {
		t.Fatalf("histórico inesperado: %+v", lista.Versoes)
	}

	var v1 VersaoKit
	if st := c.pedir("GET", "/api/kit/versoes/1", nil, &v1); st != 200 {
		t.Fatalf("GET versao 1 falhou: %d", st)
	}
	if v1.Dados.TomDeVoz != "Inicial" || v1.Dados.Diferencial != "" || v1.Origens["tomDeVoz"].Origem != "confirmado" {
		t.Fatalf("snapshot v1 inesperado: %+v", v1)
	}
	var v2 VersaoKit
	if st := c.pedir("GET", "/api/kit/versoes/2", nil, &v2); st != 200 {
		t.Fatalf("GET versao 2 falhou: %d", st)
	}
	if v2.Dados.Diferencial != "IA sugere" || v2.Origens["diferencial"].Origem != "sugerido_ia" || v2.Origens["diferencial"].DefinidoPor != nil {
		t.Fatalf("snapshot v2 inesperado: %+v", v2)
	}

	// Paginação de 20 por página: cria até 22 versões.
	versao := 6
	for i := 7; i <= 22; i++ {
		if st := c.pedir("PUT", "/api/kit", map[string]any{
			"organizacaoId": growth.ID, "versaoBase": versao, "tomDeVoz": fmt.Sprintf("Tom %d", i),
		}, &r); st != 200 {
			t.Fatalf("PUT %d falhou: %d", i, st)
		}
		versao = r.Versao
	}
	var pag1, pag2 RespostaVersoesKit
	if st := c.pedir("GET", "/api/kit/versoes?pagina=1", nil, &pag1); st != 200 || pag1.Pagina != 1 || pag1.Total != 22 || len(pag1.Versoes) != 20 {
		t.Fatalf("página 1 inesperada: %d %+v", st, pag1)
	}
	if pag1.Versoes[0].Numero != 22 || pag1.Versoes[19].Numero != 3 {
		t.Fatalf("ordem da página 1 errada: primeiro=%d último=%d", pag1.Versoes[0].Numero, pag1.Versoes[19].Numero)
	}
	if st := c.pedir("GET", "/api/kit/versoes?pagina=2", nil, &pag2); st != 200 || pag2.Pagina != 2 || pag2.Total != 22 || len(pag2.Versoes) != 2 {
		t.Fatalf("página 2 inesperada: %d %+v", st, pag2)
	}
	if pag2.Versoes[0].Numero != 2 || pag2.Versoes[1].Numero != 1 {
		t.Fatalf("ordem da página 2 errada: %+v", pag2.Versoes)
	}
	if st, _ := c.pedirComCorpo("GET", "/api/kit/versoes?pagina=0", nil); st != 400 {
		t.Fatalf("pagina=0 esperava 400, veio %d", st)
	}
}

func TestEtapa2CC5ConcorrenciaVersao(t *testing.T) {
	api := NovaAPI(prepararBanco(t), false, nil)
	c := novoCliente(t, api)
	s := entrarComoPaulo(t, c)
	growth := s.Organizacoes[0]

	var r RespostaKit
	if st := c.pedir("PUT", "/api/kit", map[string]any{
		"organizacaoId": growth.ID, "versaoBase": 0, "tomDeVoz": "Original",
	}, &r); st != 200 || r.Versao != 1 {
		t.Fatalf("PUT inicial falhou: %d", st)
	}
	if _, err := api.gravarSugestoesIA(context.Background(), growth.ID, map[string]any{"diferencial": "Sugerido"}); err != nil {
		t.Fatalf("IA falhou: %v", err)
	}
	// Agora versão atual = 2.

	// PUT com versaoBase desatualizada -> 409 kit_desatualizado, nada gravado.
	st, body := c.pedirComCorpo("PUT", "/api/kit", map[string]any{
		"organizacaoId": growth.ID, "versaoBase": 1, "tomDeVoz": "Sobrescreveu",
	})
	if st != 409 || !strings.Contains(string(body), `"kit_desatualizado"`) || !strings.Contains(string(body), msgKitDesatualizado) {
		t.Fatalf("PUT desatualizado esperava 409 kit_desatualizado, veio %d (%s)", st, string(body))
	}

	// Confirmar com organizacaoId de outra organização -> 409 organizacao_trocada, nada gravado.
	st, body = c.pedirComCorpo("POST", "/api/kit/confirmar", map[string]any{
		"organizacaoId": s.Organizacoes[1].ID, "campos": []string{"diferencial"}, "versaoBase": 2,
	})
	if st != 409 || !strings.Contains(string(body), `"organizacao_trocada"`) {
		t.Fatalf("confirmar com outra org esperava 409 organizacao_trocada, veio %d (%s)", st, string(body))
	}

	// Confirmar com versaoBase desatualizada -> 409 kit_desatualizado, origem continua sugerido_ia.
	st, body = c.pedirComCorpo("POST", "/api/kit/confirmar", map[string]any{
		"organizacaoId": growth.ID, "campos": []string{"diferencial"}, "versaoBase": 1,
	})
	if st != 409 || !strings.Contains(string(body), `"kit_desatualizado"`) {
		t.Fatalf("confirmar desatualizado esperava 409 kit_desatualizado, veio %d (%s)", st, string(body))
	}

	c.pedir("GET", "/api/kit", nil, &r)
	if r.Versao != 2 || r.Kit.TomDeVoz != "Original" || r.Origens["diferencial"].Origem != "sugerido_ia" {
		t.Fatalf("estado alterado após 409: %+v", r)
	}

	// Duas gravações simultâneas com versaoBase = 2: apenas uma vence (200) e a outra recebe 409.
	var wg sync.WaitGroup
	inicio := make(chan struct{})
	status := make([]int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-inicio
			stReq, _ := c.pedirComCorpo("PUT", "/api/kit", map[string]any{
				"organizacaoId": growth.ID,
				"versaoBase":    2,
				"tomDeVoz":      fmt.Sprintf("Concorrente %d", idx),
			})
			status[idx] = stReq
		}(i)
	}
	close(inicio)
	wg.Wait()
	slices.Sort(status)
	if status[0] != 200 || status[1] != 409 {
		t.Fatalf("corrida esperava [200, 409], veio %v", status)
	}
	c.pedir("GET", "/api/kit", nil, &r)
	if r.Versao != 3 {
		t.Fatalf("esperava versão 3 após corrida, veio %d", r.Versao)
	}
}

func TestEtapa2CC6Migracao0005PreservaDadosEArquivos(t *testing.T) {
	url := os.Getenv("LG_BANCO_TESTE_URL")
	if url == "" {
		t.Skip("LG_BANCO_TESTE_URL não definida")
	}
	ctx := context.Background()
	banco, err := conectar(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer banco.Close()

	if _, err := banco.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public`); err != nil {
		t.Fatal(err)
	}
	if err := migrarAte(ctx, banco, "0004"); err != nil {
		t.Fatalf("migrarAte 0004 falhou: %v", err)
	}
	if err := semear(ctx, banco, "Paulo", "Paulo@Exemplo.com", senhaTeste); err != nil {
		t.Fatal(err)
	}

	var pauloID, growthID, hubID string
	if err := banco.QueryRow(ctx, `SELECT id::text FROM usuarios WHERE lower(email) = 'paulo@exemplo.com'`).Scan(&pauloID); err != nil {
		t.Fatal(err)
	}
	if err := banco.QueryRow(ctx, `SELECT id::text FROM organizacoes WHERE nome = 'Laura Growth'`).Scan(&growthID); err != nil {
		t.Fatal(err)
	}
	if err := banco.QueryRow(ctx, `SELECT id::text FROM organizacoes WHERE nome = 'Laura Hub'`).Scan(&hubID); err != nil {
		t.Fatal(err)
	}

	// Semeia kit e arquivos na estrutura da migração 0004.
	emAnterior := time.Date(2026, 11, 20, 14, 30, 0, 0, time.UTC)
	if _, err := banco.Exec(ctx, `
		INSERT INTO kits_marca (
			organizacao_id, site, tom_de_voz, nunca_dizer, pilares, cores, atualizado_em, atualizado_por
		) VALUES (
			$1::uuid, 'https://antigo.example', 'Claro e humano', 'Sem promessas irreais',
			ARRAY['Educação', 'Bastidores', 'Casos'], ARRAY['#112233'], $2, $3::uuid
		)`, growthID, emAnterior, pauloID); err != nil {
		t.Fatal(err)
	}
	var logoID, refID string
	refBytes := []byte("%PDF-1.4\n%conteudo-teste\n")
	if err := banco.QueryRow(ctx, `
		INSERT INTO arquivos_marca (organizacao_id, tipo, nome, mime, tamanho, conteudo, criado_por)
		VALUES ($1::uuid, 'logo', 'logo-antigo.png', 'image/png', $2, $3, $4::uuid)
		RETURNING id::text`, growthID, len(pngMinimo), pngMinimo, pauloID).Scan(&logoID); err != nil {
		t.Fatal(err)
	}
	if err := banco.QueryRow(ctx, `
		INSERT INTO arquivos_marca (organizacao_id, tipo, nome, mime, tamanho, conteudo, criado_por)
		VALUES ($1::uuid, 'referencia', 'manual-antigo.pdf', 'application/pdf', $2, $3, $4::uuid)
		RETURNING id::text`, growthID, len(refBytes), refBytes, pauloID).Scan(&refID); err != nil {
		t.Fatal(err)
	}

	// Segundo kit sem atualizado_por para garantir o tratamento de NULL na migração.
	if _, err := banco.Exec(ctx, `
		INSERT INTO kits_marca (organizacao_id, publico, atualizado_em, atualizado_por)
		VALUES ($1::uuid, 'Pequenas empresas', $2, NULL)`, hubID, emAnterior); err != nil {
		t.Fatal(err)
	}

	// Aplica a migração 0005.
	if err := migrar(ctx, banco); err != nil {
		t.Fatalf("migrar 0005 falhou: %v", err)
	}

	api := NovaAPI(banco, false, nil)
	c := novoCliente(t, api)
	entrarComoPaulo(t, c)

	var r RespostaKit
	if st := c.pedir("GET", "/api/kit", nil, &r); st != 200 {
		t.Fatalf("GET /api/kit após 0005 falhou: %d", st)
	}
	if r.Versao != 1 || r.Kit.Site != "https://antigo.example" || r.Kit.TomDeVoz != "Claro e humano" ||
		r.Kit.NuncaDizer != "Sem promessas irreais" || !slices.Equal(r.Kit.Pilares, []string{"Educação", "Bastidores", "Casos"}) ||
		!slices.Equal(r.Kit.Cores, []string{"#112233"}) {
		t.Fatalf("dados do kit após 0005 não conferem: %+v", r)
	}
	if len(r.Arquivos) != 2 {
		t.Fatalf("arquivos esperados=2, veio %d", len(r.Arquivos))
	}
	stLogo, bytesLogo, _ := c.baixar("/api/kit/arquivos/" + logoID)
	if stLogo != 200 || !bytes.Equal(bytesLogo, pngMinimo) {
		t.Fatalf("logo corrompido após 0005: %d", stLogo)
	}
	stRef, bytesRef, _ := c.baixar("/api/kit/arquivos/" + refID)
	if stRef != 200 || !bytes.Equal(bytesRef, refBytes) {
		t.Fatalf("referência corrompida após 0005: %d", stRef)
	}

	for _, preenchido := range []string{"site", "tomDeVoz", "nuncaDizer", "pilares", "cores"} {
		o, ok := r.Origens[preenchido]
		if !ok || o.Origem != "confirmado" || o.DefinidoPor == nil || *o.DefinidoPor != "Paulo" {
			t.Fatalf("origem esperada confirmado por Paulo para %s: %+v (ok=%v)", preenchido, o, ok)
		}
	}
	if _, ok := r.Origens["perfis"]; ok {
		t.Fatalf("campo vazio não deveria ter origem após migração: %+v", r.Origens["perfis"])
	}

	var versoes RespostaVersoesKit
	if st := c.pedir("GET", "/api/kit/versoes", nil, &versoes); st != 200 || versoes.Total != 1 || len(versoes.Versoes) != 1 {
		t.Fatalf("versões após 0005 inesperadas: %d %+v", st, versoes)
	}
	if versoes.Versoes[0].Numero != 1 || versoes.Versoes[0].AutorTipo != "pessoa" || *versoes.Versoes[0].AutorNome != "Paulo" {
		t.Fatalf("versão 1 após 0005 inesperada: %+v", versoes.Versoes[0])
	}
	var v1 VersaoKit
	if st := c.pedir("GET", "/api/kit/versoes/1", nil, &v1); st != 200 {
		t.Fatalf("GET /api/kit/versoes/1 após 0005 falhou: %d", st)
	}
	if v1.Dados.Site != "https://antigo.example" || v1.Origens["site"].Origem != "confirmado" {
		t.Fatalf("snapshot versão 1 após 0005 inesperado: %+v", v1)
	}

	// Laura Hub (atualizado_por = NULL) também recebeu versão 1 e origem confirmado sem autor.
	c.pedir("PUT", "/api/sessao/organizacao", map[string]string{"organizacaoId": hubID}, nil)
	if st := c.pedir("GET", "/api/kit", nil, &r); st != 200 || r.Versao != 1 || r.Kit.Publico != "Pequenas empresas" {
		t.Fatalf("kit Laura Hub após 0005 inesperado: %+v", r)
	}
	if r.Origens["publico"].Origem != "confirmado" || r.Origens["publico"].DefinidoPor != nil {
		t.Fatalf("origem Laura Hub após 0005 inesperada: %+v", r.Origens["publico"])
	}
	var v1Hub VersaoKit
	if st := c.pedir("GET", "/api/kit/versoes/1", nil, &v1Hub); st != 200 || v1Hub.AutorNome != nil || v1Hub.AutorTipo != "pessoa" {
		t.Fatalf("versão 1 Laura Hub inesperada: %d %+v", st, v1Hub)
	}
}

func TestEtapa2CC7AutorizacaoEIsolamento(t *testing.T) {
	api := NovaAPI(prepararBanco(t), false, nil)
	paulo := novoCliente(t, api)
	sessPaulo := entrarComoPaulo(t, paulo)
	growth, hub := sessPaulo.Organizacoes[0], sessPaulo.Organizacoes[1]

	// Paulo cria 2 versões em Laura Hub e 1 versão em Laura Growth.
	paulo.pedir("PUT", "/api/kit", map[string]any{"organizacaoId": growth.ID, "versaoBase": 0, "tomDeVoz": "Growth v1"}, nil)
	paulo.pedir("PUT", "/api/sessao/organizacao", map[string]string{"organizacaoId": hub.ID}, nil)
	paulo.pedir("PUT", "/api/kit", map[string]any{"organizacaoId": hub.ID, "versaoBase": 0, "tomDeVoz": "Hub v1"}, nil)
	paulo.pedir("PUT", "/api/kit", map[string]any{"organizacaoId": hub.ID, "versaoBase": 1, "tomDeVoz": "Hub v2"}, nil)

	// Membro sem alçadas -> 403 em kit, versões e origens.
	criarMembroNaOrg(t, api.banco, growth.ID, "Sem Alcada", "sem-2c@exemplo.com")
	cliSem := novoCliente(t, api)
	entrarComEmail(t, cliSem, "sem-2c@exemplo.com")
	for _, rota := range []string{"/api/kit", "/api/kit/versoes", "/api/kit/versoes/1"} {
		if st := cliSem.pedir("GET", rota, nil, nil); st != 403 {
			t.Fatalf("GET %s sem ver_marca_posts esperava 403, veio %d", rota, st)
		}
	}

	// Membro só com ver_marca_posts -> lê kit e versões, mas recebe 403 ao salvar/confirmar e em arquivos.
	criarMembroNaOrg(t, api.banco, growth.ID, "Leitor", "leitor-2c@exemplo.com", "ver_marca_posts")
	cliLeitor := novoCliente(t, api)
	entrarComEmail(t, cliLeitor, "leitor-2c@exemplo.com")
	paulo.pedir("PUT", "/api/sessao/organizacao", map[string]string{"organizacaoId": growth.ID}, nil)
	var logoGrowth ArquivoMarca
	if st := paulo.enviar("/api/kit/arquivos?tipo=logo&organizacaoId="+growth.ID, "growth-logo.png", pngMinimo, &logoGrowth); st != 201 {
		t.Fatalf("upload do logo por Paulo falhou: %d", st)
	}
	if st := cliLeitor.pedir("GET", "/api/kit", nil, nil); st != 200 {
		t.Fatalf("GET /api/kit com ver_marca_posts esperava 200, veio %d", st)
	}
	if st := cliLeitor.pedir("GET", "/api/kit/versoes", nil, nil); st != 200 {
		t.Fatalf("GET /api/kit/versoes com ver_marca_posts esperava 200, veio %d", st)
	}
	if st := cliLeitor.pedir("GET", "/api/kit/versoes/1", nil, nil); st != 200 {
		t.Fatalf("GET /api/kit/versoes/1 com ver_marca_posts esperava 200, veio %d", st)
	}
	if st := cliLeitor.pedir("PUT", "/api/kit", map[string]any{"organizacaoId": growth.ID, "versaoBase": 2, "tomDeVoz": "Bloqueado"}, nil); st != 403 {
		t.Fatalf("PUT /api/kit sem cmo_dna esperava 403, veio %d", st)
	}
	if st := cliLeitor.pedir("POST", "/api/kit/confirmar", map[string]any{"organizacaoId": growth.ID, "campos": []string{"tomDeVoz"}, "versaoBase": 2}, nil); st != 403 {
		t.Fatalf("POST /api/kit/confirmar sem cmo_dna esperava 403, veio %d", st)
	}
	if st := cliLeitor.enviar("/api/kit/arquivos?tipo=logo&organizacaoId="+growth.ID, "bloqueado.png", pngMinimo, nil); st != 403 {
		t.Fatalf("POST /api/kit/arquivos sem cmo_dna esperava 403, veio %d", st)
	}
	if st := cliLeitor.pedir("DELETE", "/api/kit/arquivos/"+logoGrowth.ID+"?organizacaoId="+growth.ID, nil, nil); st != 403 {
		t.Fatalf("DELETE /api/kit/arquivos sem cmo_dna esperava 403, veio %d", st)
	}

	// Isolamento entre organizações: Laura Growth tem 2 versões (PUT + logo); pedir versão 3 dá 404.
	if st := cliLeitor.pedir("GET", "/api/kit/versoes/3", nil, nil); st != 404 {
		t.Fatalf("versão de outra org esperava 404, veio %d", st)
	}
	var versoesGrowth RespostaVersoesKit
	if st := cliLeitor.pedir("GET", "/api/kit/versoes", nil, &versoesGrowth); st != 200 || versoesGrowth.Total != 2 {
		t.Fatalf("Laura Growth deveria ter 2 versões, veio %+v", versoesGrowth)
	}
	var v1Growth VersaoKit
	if st := cliLeitor.pedir("GET", "/api/kit/versoes/1", nil, &v1Growth); st != 200 || v1Growth.Dados.TomDeVoz != "Growth v1" {
		t.Fatalf("versão 1 de Laura Growth inesperada: %d %+v", st, v1Growth)
	}

	// Organização sem versões recebe 404 ao pedir /api/kit/versoes/1.
	var alheiaID string
	if err := api.banco.QueryRow(context.Background(), `SELECT id::text FROM organizacoes WHERE nome = 'Clínica Alheia'`).Scan(&alheiaID); err != nil {
		t.Fatal(err)
	}
	criarMembroNaOrg(t, api.banco, alheiaID, "Dona Alheia", "dona-alheia@exemplo.com", "ver_marca_posts")
	cliAlheia := novoCliente(t, api)
	entrarComEmail(t, cliAlheia, "dona-alheia@exemplo.com")
	if st := cliAlheia.pedir("GET", "/api/kit/versoes/1", nil, nil); st != 404 {
		t.Fatalf("org sem versões esperava 404 para versão 1, veio %d", st)
	}
}

func TestEtapa2CRevisaoLGPDOrigensVersao(t *testing.T) {
	api := NovaAPI(prepararBanco(t), false, nil)
	paulo := novoCliente(t, api)
	sessPaulo := entrarComoPaulo(t, paulo)
	growth := sessPaulo.Organizacoes[0]

	// Outro membro com cmo_dna grava um campo para gerar a versão 1.
	editorID := criarMembroNaOrg(t, api.banco, growth.ID, "Maria Sigilosa", "maria-lgpd@exemplo.com", "ver_marca_posts", "cmo_dna")
	maria := novoCliente(t, api)
	entrarComEmail(t, maria, "maria-lgpd@exemplo.com")

	var r RespostaKit
	if st := maria.pedir("PUT", "/api/kit", map[string]any{
		"organizacaoId": growth.ID, "versaoBase": 0, "tomDeVoz": "Humano",
	}, &r); st != 200 {
		t.Fatalf("PUT por Maria falhou: %d", st)
	}

	// O JSONB salvo em kit_versoes.origens não deve conter o nome da pessoa (somente definidoPorId).
	var origensBruto string
	if err := api.banco.QueryRow(context.Background(), `
		SELECT origens::text FROM kit_versoes WHERE organizacao_id = $1::uuid AND numero = 1`, growth.ID).Scan(&origensBruto); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(origensBruto, "Maria Sigilosa") || !strings.Contains(origensBruto, editorID) {
		t.Fatalf("kit_versoes.origens não deve guardar nome e deve guardar UUID: %s", origensBruto)
	}

	// Enquanto Maria existe, GET /api/kit/versoes/1 resolve o nome via JOIN.
	var v1 VersaoKit
	if st := paulo.pedir("GET", "/api/kit/versoes/1", nil, &v1); st != 200 {
		t.Fatalf("GET versao 1 falhou: %d", st)
	}
	if v1.AutorNome == nil || *v1.AutorNome != "Maria Sigilosa" ||
		v1.Origens["tomDeVoz"].DefinidoPor == nil || *v1.Origens["tomDeVoz"].DefinidoPor != "Maria Sigilosa" {
		t.Fatalf("esperava nome resolvido na leitura: %+v", v1)
	}

	// CHECK em kit_versoes impede autor_tipo='ia' com autor_id preenchido.
	if _, err := api.banco.Exec(context.Background(), `
		INSERT INTO kit_versoes (organizacao_id, numero, dados, origens, campos_alterados, autor_id, autor_tipo)
		VALUES ($1::uuid, 99, '{}'::jsonb, '{}'::jsonb, '{}', $2::uuid, 'ia')`, growth.ID, editorID); err == nil {
		t.Fatal("esperava falha da restrição CHECK autor_tipo <> 'ia' OR autor_id IS NULL")
	}

	// Após excluir Maria, GET /api/kit/versoes/1 devolve autorNome=null e definidoPor=null.
	if _, err := api.banco.Exec(context.Background(), `DELETE FROM usuarios WHERE id = $1::uuid`, editorID); err != nil {
		t.Fatal(err)
	}
	var v1Apos VersaoKit
	if st := paulo.pedir("GET", "/api/kit/versoes/1", nil, &v1Apos); st != 200 {
		t.Fatalf("GET versao 1 após exclusão falhou: %d", st)
	}
	if v1Apos.AutorNome != nil || v1Apos.Origens["tomDeVoz"].DefinidoPor != nil {
		t.Fatalf("após exclusão do usuário, nomes deveriam vir null: %+v", v1Apos)
	}
}

func TestEtapa2CRevisaoSugestoesIAConservadoraERollback(t *testing.T) {
	api := NovaAPI(prepararBanco(t), false, nil)
	c := novoCliente(t, api)
	s := entrarComoPaulo(t, c)
	growth := s.Organizacoes[0]

	// Campo confirmado pelo usuário e sugestão inicial da IA.
	if st := c.pedir("PUT", "/api/kit", map[string]any{
		"organizacaoId": growth.ID, "versaoBase": 0, "publico": "Clínicas", "modeloNegocio": "b2b",
	}, nil); st != 200 {
		t.Fatal(st)
	}
	res1, err := api.gravarSugestoesIA(context.Background(), growth.ID, map[string]any{
		"tomDeVoz": "Direto",
	})
	if err != nil || !slices.Equal(res1.Aplicados, []string{"tomDeVoz"}) || len(res1.Ignorados) != 0 {
		t.Fatalf("primeira sugestão IA inesperada: %+v %v", res1, err)
	}

	// IA tenta sobrescrever campo confirmado ('publico', 'modeloNegocio'), repetir valor igual ('tomDeVoz') e sugerir um novo ('diferencial').
	res2, err := api.gravarSugestoesIA(context.Background(), growth.ID, map[string]any{
		"publico":       "Sobrescrito pela IA",
		"modeloNegocio": "b2c",
		"tomDeVoz":      " Direto ",
		"diferencial":   "Atendimento ágil",
	})
	if err != nil {
		t.Fatalf("segunda sugestão IA falhou: %v", err)
	}
	if !slices.Equal(res2.Aplicados, []string{"diferencial"}) {
		t.Fatalf("esperava apenas diferencial aplicado, veio %+v", res2)
	}
	if !slices.Equal(res2.Ignorados, []string{"tomDeVoz", "publico", "modeloNegocio"}) {
		t.Fatalf("esperava tomDeVoz/publico/modeloNegocio ignorados, veio %+v", res2)
	}

	var r RespostaKit
	c.pedir("GET", "/api/kit", nil, &r)
	if r.Versao != 3 || r.Kit.Publico != "Clínicas" || r.Kit.ModeloNegocio != "b2b" || r.Kit.Diferencial != "Atendimento ágil" {
		t.Fatalf("campos confirmados foram alterados indevidamente: %+v", r)
	}
	if r.Origens["publico"].Origem != "confirmado" || r.Origens["modeloNegocio"].Origem != "confirmado" {
		t.Fatalf("origens confirmadas alteradas: %+v", r.Origens)
	}

	// Se nada restar para aplicar (só confirmados e/ou inalterados), não cria versão.
	res3, err := api.gravarSugestoesIA(context.Background(), growth.ID, map[string]any{
		"publico":     "Outra tentativa",
		"diferencial": "Atendimento ágil",
	})
	if err != nil {
		t.Fatalf("sugestão no-op não deveria falhar: %v", err)
	}
	if len(res3.Aplicados) != 0 || len(res3.Ignorados) != 2 {
		t.Fatalf("esperava nenhum aplicado e 2 ignorados, veio %+v", res3)
	}
	c.pedir("GET", "/api/kit", nil, &r)
	if r.Versao != 3 {
		t.Fatalf("não deveria criar versão quando nada foi aplicado, veio versão %d", r.Versao)
	}

	// Campo inválido ou valor inválido em gravarSugestoesIA faz rollback total (nada gravado).
	if _, err := api.gravarSugestoesIA(context.Background(), growth.ID, map[string]any{
		"regioes":       "São Paulo",
		"campoInvalido": "valor",
	}); err == nil {
		t.Fatal("esperava erro ao passar campo inválido")
	}
	if _, err := api.gravarSugestoesIA(context.Background(), growth.ID, map[string]any{
		"regioes": "São Paulo",
		"site":    "javascript:alert(1)",
	}); err == nil {
		t.Fatal("esperava erro ao passar site inválido")
	}
	c.pedir("GET", "/api/kit", nil, &r)
	if r.Versao != 3 || r.Kit.Regioes != "" || r.Origens["regioes"].Origem != "" {
		t.Fatalf("rollback de gravarSugestoesIA falhou: %+v", r)
	}
}

func TestEtapa2CRevisaoConfirmarNoOp(t *testing.T) {
	api := NovaAPI(prepararBanco(t), false, nil)
	paulo := novoCliente(t, api)
	sessPaulo := entrarComoPaulo(t, paulo)
	growth := sessPaulo.Organizacoes[0]

	criarMembroNaOrg(t, api.banco, growth.ID, "Editora Ana", "ana-noop@exemplo.com", "ver_marca_posts", "cmo_dna")
	ana := novoCliente(t, api)
	entrarComEmail(t, ana, "ana-noop@exemplo.com")

	// Paulo confirma 'publico'; IA sugere 'tomDeVoz'.
	var r RespostaKit
	if st := paulo.pedir("PUT", "/api/kit", map[string]any{
		"organizacaoId": growth.ID, "versaoBase": 0, "publico": "Empresas",
	}, &r); st != 200 {
		t.Fatal(st)
	}
	emPublico := r.Origens["publico"].DefinidoEm
	if _, err := api.gravarSugestoesIA(context.Background(), growth.ID, map[string]any{"tomDeVoz": "Direto"}); err != nil {
		t.Fatal(err)
	}

	// Ana confirma ['publico', 'tomDeVoz']: 'publico' é no-op (autor/data de Paulo preservados), só 'tomDeVoz' muda e entra na versão.
	if st := ana.pedir("POST", "/api/kit/confirmar", map[string]any{
		"organizacaoId": growth.ID,
		"campos":        []string{"publico", "tomDeVoz"},
		"versaoBase":    2,
	}, &r); st != 200 {
		t.Fatalf("confirmar misto falhou: %d", st)
	}
	if r.Versao != 3 {
		t.Fatalf("esperava versão 3, veio %d", r.Versao)
	}
	if r.Origens["publico"].DefinidoPor == nil || *r.Origens["publico"].DefinidoPor != "Paulo" || !r.Origens["publico"].DefinidoEm.Equal(emPublico) {
		t.Fatalf("campo já confirmado mudou autor/data: %+v (antes=%v)", r.Origens["publico"], emPublico)
	}
	if r.Origens["tomDeVoz"].DefinidoPor == nil || *r.Origens["tomDeVoz"].DefinidoPor != "Editora Ana" {
		t.Fatalf("tomDeVoz deveria ser confirmado por Editora Ana: %+v", r.Origens["tomDeVoz"])
	}
	var v3 VersaoKit
	paulo.pedir("GET", "/api/kit/versoes/3", nil, &v3)
	if !slices.Equal(v3.CamposAlterados, []string{"tomDeVoz"}) {
		t.Fatalf("versão 3 deveria listar apenas tomDeVoz, veio %v", v3.CamposAlterados)
	}

	// Confirmar quando todos os campos já estão confirmados -> 200 com kit atual e NENHUMA versão nova.
	var rNoOp RespostaKit
	if st := ana.pedir("POST", "/api/kit/confirmar", map[string]any{
		"organizacaoId": growth.ID,
		"campos":        []string{"publico", "tomDeVoz"},
		"versaoBase":    3,
	}, &rNoOp); st != 200 {
		t.Fatalf("confirmar no-op esperava 200, veio %d", st)
	}
	if rNoOp.Versao != 3 {
		t.Fatalf("confirmar no-op não deveria criar versão, veio %d", rNoOp.Versao)
	}
}

func TestEtapa2CRevisaoPaginacaoENumeroLimites(t *testing.T) {
	api := NovaAPI(prepararBanco(t), false, nil)
	c := novoCliente(t, api)
	s := entrarComoPaulo(t, c)
	growth := s.Organizacoes[0]

	c.pedir("PUT", "/api/kit", map[string]any{"organizacaoId": growth.ID, "versaoBase": 0, "tomDeVoz": "Ok"}, nil)

	for _, pag := range []string{"0", "-1", "abc", "1000001", "999999999999999999", "99999999999999999999999999"} {
		st, body := c.pedirComCorpo("GET", "/api/kit/versoes?pagina="+pag, nil)
		if st != 400 || !strings.Contains(string(body), `"dados_invalidos"`) {
			t.Fatalf("pagina=%s esperava 400 dados_invalidos, veio %d (%s)", pag, st, string(body))
		}
	}

	for _, num := range []string{"0", "-5", "abc", "9999999999", "99999999999999999999999999"} {
		st, _ := c.pedirComCorpo("GET", "/api/kit/versoes/"+num, nil)
		if st != 404 {
			t.Fatalf("numero=%s esperava 404, veio %d", num, st)
		}
	}
}

func TestEtapa2CRevisaoArquivosVersoesEFalhas(t *testing.T) {
	api := NovaAPI(prepararBanco(t), false, nil)
	c := novoCliente(t, api)
	s := entrarComoPaulo(t, c)
	growth := s.Organizacoes[0]

	// Upload 1: logo inicial -> v1
	var logo1 ArquivoMarca
	if st := c.enviar("/api/kit/arquivos?tipo=logo&organizacaoId="+growth.ID, "logo1.png", pngMinimo, &logo1); st != 201 {
		t.Fatalf("logo1 falhou: %d", st)
	}
	// Upload 2: substituição de logo -> v2
	var logo2 ArquivoMarca
	if st := c.enviar("/api/kit/arquivos?tipo=logo&organizacaoId="+growth.ID, "logo2.png", pngMinimo, &logo2); st != 201 {
		t.Fatalf("logo2 falhou: %d", st)
	}

	// Upload com falha (tamanho > 8 MB e formato inválido) não cria versão.
	grande := make([]byte, maxTamanhoArquivo+10)
	copy(grande, pngMinimo)
	if st := c.enviar("/api/kit/arquivos?tipo=logo&organizacaoId="+growth.ID, "grande.png", grande, nil); st != 413 {
		t.Fatalf("arquivo grande esperava 413, veio %d", st)
	}
	if st := c.enviar("/api/kit/arquivos?tipo=logo&organizacaoId="+growth.ID, "falso.png", []byte("texto puro"), nil); st != 415 {
		t.Fatalf("formato inválido esperava 415, veio %d", st)
	}

	var r RespostaKit
	c.pedir("GET", "/api/kit", nil, &r)
	if r.Versao != 2 {
		t.Fatalf("uploads com falha não deveriam criar versão, veio %d", r.Versao)
	}

	// Remoção de logo -> v3
	if st := c.pedir("DELETE", "/api/kit/arquivos/"+logo2.ID+"?organizacaoId="+growth.ID, nil, nil); st != 204 {
		t.Fatalf("remover logo2 falhou: %d", st)
	}

	var lista RespostaVersoesKit
	if st := c.pedir("GET", "/api/kit/versoes", nil, &lista); st != 200 || lista.Total != 3 {
		t.Fatalf("versões esperadas=3, veio %d %+v", st, lista)
	}
	if !slices.Equal(lista.Versoes[0].CamposAlterados, []string{"logo:logo2.png"}) ||
		!slices.Equal(lista.Versoes[1].CamposAlterados, []string{"logo:logo2.png"}) ||
		!slices.Equal(lista.Versoes[2].CamposAlterados, []string{"logo:logo1.png"}) {
		t.Fatalf("camposAlterados de logo inesperados: %+v", lista.Versoes)
	}
}

func jsonDecode(r io.Reader, v any) error { return json.NewDecoder(r).Decode(v) }
