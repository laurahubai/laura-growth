package main

// Testes da etapa 2 (kit de marca). Cobrem o "pronto quando" do roteiro que não
// depende de IA: kit por organização, isolamento, pilares e envio de arquivos.

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"testing"
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
	api := NovaAPI(prepararBanco(t), false)
	c := novoCliente(t, api)
	s := entrarComoPaulo(t, c)
	growth, hub := s.Organizacoes[0], s.Organizacoes[1]

	// Kit vazio no começo.
	var r RespostaKit
	if st := c.pedir("GET", "/api/kit", nil, &r); st != 200 || r.Kit.OrganizacaoID != growth.ID || len(r.Kit.Pilares) != 0 || r.Kit.AtualizadoEm != nil {
		t.Fatalf("kit inicial inesperado: %d %+v", st, r.Kit)
	}

	// Pronto quando: o kit está cadastrado, com 3 a 5 pilares.
	kit := map[string]any{
		"organizacaoId": growth.ID, "site": "https://laura.example", "tomDeVoz": "Próximo e direto",
		"pilares": []string{" Constância ", "Bastidores", "", "IA na prática"}, "cores": []string{"#F26B3A", "#1f2430"},
	}
	if st := c.pedir("PUT", "/api/kit", kit, &r); st != 200 {
		t.Fatalf("salvar kit esperava 200, veio %d", st)
	}
	if len(r.Kit.Pilares) != 3 || r.Kit.Pilares[0] != "Constância" || r.Kit.Cores[0] != "#f26b3a" || r.Kit.AtualizadoEm == nil {
		t.Fatalf("kit salvo inesperado: %+v", r.Kit)
	}

	// Regras: no máximo 5 pilares; site precisa ser endereço completo.
	ruim := map[string]any{"organizacaoId": growth.ID, "pilares": []string{"a", "b", "c", "d", "e", "f"}}
	if st := c.pedir("PUT", "/api/kit", ruim, nil); st != 400 {
		t.Fatalf("6 pilares esperava 400, veio %d", st)
	}
	if st := c.pedir("PUT", "/api/kit", map[string]any{"organizacaoId": growth.ID, "site": "javascript:alert(1)"}, nil); st != 400 {
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

func jsonDecode(r io.Reader, v any) error { return json.NewDecoder(r).Decode(v) }
