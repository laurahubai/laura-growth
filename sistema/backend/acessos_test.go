package main

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"
)

func criarMembroNaOrg(t *testing.T, banco *Banco, orgID, nome, email string, alcadas ...string) string {
	t.Helper()
	hash, err := gerarHashSenha(senhaTeste)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	var usuarioID string
	err = banco.QueryRow(ctx, `
		INSERT INTO usuarios (nome, email, senha_hash) VALUES ($1, $2, $3)
		RETURNING id::text`, nome, email, hash).Scan(&usuarioID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := banco.Exec(ctx, `
		INSERT INTO membros (organizacao_id, usuario_id, papel) VALUES ($1::uuid, $2::uuid, 'membro')`,
		orgID, usuarioID); err != nil {
		t.Fatal(err)
	}
	for _, al := range alcadas {
		if _, err := banco.Exec(ctx, `
			INSERT INTO alcadas_membro (organizacao_id, usuario_id, alcada) VALUES ($1::uuid, $2::uuid, $3)`,
			orgID, usuarioID, al); err != nil {
			t.Fatal(err)
		}
	}
	return usuarioID
}

func entrarComEmail(t *testing.T, c *cliente, email string) Sessao {
	t.Helper()
	var s Sessao
	if st := c.pedir("POST", "/api/sessao", map[string]string{"email": email, "senha": senhaTeste}, &s); st != 200 {
		t.Fatalf("login %s falhou: %d", email, st)
	}
	return s
}

func TestAcessosB3AlteracaoImediataESessao(t *testing.T) {
	api := NovaAPI(prepararBanco(t), false, nil)
	paulo := novoCliente(t, api)
	sessPaulo := entrarComoPaulo(t, paulo)
	growth, hub := sessPaulo.Organizacoes[0], sessPaulo.Organizacoes[1]

	// Membro sem alçadas em Laura Growth e com 'ver_marca_posts' em Laura Hub.
	membroID := criarMembroNaOrg(t, api.banco, growth.ID, "Ana", "ana-b3@exemplo.com")
	if _, err := api.banco.Exec(context.Background(), `
		INSERT INTO membros (organizacao_id, usuario_id, papel) VALUES ($1::uuid, $2::uuid, 'membro')`,
		hub.ID, membroID); err != nil {
		t.Fatal(err)
	}

	ana := novoCliente(t, api)
	sessAna := entrarComEmail(t, ana, "ana-b3@exemplo.com")
	if sessAna.OrganizacaoAtiva.ID != growth.ID || len(sessAna.Alcadas) != 0 {
		t.Fatalf("esperava 0 alçadas em Laura Growth, veio %+v", sessAna.Alcadas)
	}

	// Cabeçalho da organização continua aberto a qualquer membro (mesmo com 0 alçadas).
	var det OrganizacaoDetalhe
	if st := ana.pedir("GET", "/api/organizacao", nil, &det); st != 200 || det.Papel != "membro" {
		t.Fatalf("GET /api/organizacao esperava 200 para membro sem alçadas, veio %d", st)
	}
	if st := ana.pedir("GET", "/api/kit", nil, nil); st != 403 {
		t.Fatalf("GET /api/kit sem alçada esperava 403, veio %d", st)
	}

	// Paulo concede ver_marca_posts -> efeito imediato no próximo pedido da Ana (sem cache).
	if st := paulo.pedir("PUT", "/api/pessoas/"+membroID+"/alcadas", map[string]any{
		"alcadas": []string{"ver_marca_posts"},
	}, nil); st != 200 {
		t.Fatalf("conceder ver_marca_posts falhou: %d", st)
	}
	if st := ana.pedir("GET", "/api/sessao", nil, &sessAna); st != 200 || !slices.Equal(sessAna.Alcadas, []string{"ver_marca_posts"}) {
		t.Fatalf("GET /api/sessao não refletiu ver_marca_posts imediato: %d %+v", st, sessAna.Alcadas)
	}
	if st := ana.pedir("GET", "/api/kit", nil, nil); st != 200 {
		t.Fatalf("GET /api/kit com ver_marca_posts esperava 200, veio %d", st)
	}
	if st := ana.pedir("PUT", "/api/kit", map[string]any{"organizacaoId": growth.ID, "tomDeVoz": "Direto"}, nil); st != 403 {
		t.Fatalf("PUT /api/kit sem cmo_dna esperava 403, veio %d", st)
	}

	// Paulo concede também cmo_dna -> Ana edita kit e envia arquivo.
	if st := paulo.pedir("PUT", "/api/pessoas/"+membroID+"/alcadas", map[string]any{
		"alcadas": []string{"ver_marca_posts", "cmo_dna"},
	}, nil); st != 200 {
		t.Fatalf("conceder cmo_dna falhou: %d", st)
	}
	if st := ana.pedir("PUT", "/api/kit", map[string]any{"organizacaoId": growth.ID, "tomDeVoz": "Direto"}, nil); st != 200 {
		t.Fatalf("PUT /api/kit com cmo_dna esperava 200, veio %d", st)
	}
	var logo ArquivoMarca
	if st := ana.enviar("/api/kit/arquivos?tipo=logo&organizacaoId="+growth.ID, "logo.png", pngMinimo, &logo); st != 201 {
		t.Fatalf("POST /api/kit/arquivos com cmo_dna esperava 201, veio %d", st)
	}
	if st, _, _ := ana.baixar("/api/kit/arquivos/" + logo.ID); st != 200 {
		t.Fatalf("GET /api/kit/arquivos com ver_marca_posts esperava 200, veio %d", st)
	}

	// Paulo revoga todas as alçadas -> Ana perde acesso imediatamente no próximo pedido.
	if st := paulo.pedir("PUT", "/api/pessoas/"+membroID+"/alcadas", map[string]any{"alcadas": []string{}}, nil); st != 200 {
		t.Fatalf("revogar alçadas falhou: %d", st)
	}
	if st := ana.pedir("GET", "/api/sessao", nil, &sessAna); st != 200 || len(sessAna.Alcadas) != 0 {
		t.Fatalf("esperava 0 alçadas após revogação, veio %+v", sessAna.Alcadas)
	}
	if st := ana.pedir("GET", "/api/kit", nil, nil); st != 403 {
		t.Fatalf("GET /api/kit após revogação esperava 403, veio %d", st)
	}
	if st, _, _ := ana.baixar("/api/kit/arquivos/" + logo.ID); st != 403 {
		t.Fatalf("baixar arquivo após revogação esperava 403, veio %d", st)
	}
	if st := ana.pedir("DELETE", "/api/kit/arquivos/"+logo.ID+"?organizacaoId="+growth.ID, nil, nil); st != 403 {
		t.Fatalf("remover arquivo após revogação esperava 403, veio %d", st)
	}

	// Trocar para Laura Hub onde o Paulo concede editar_posts reflete só as alçadas da Laura Hub.
	if st := paulo.pedir("PUT", "/api/sessao/organizacao", map[string]string{"organizacaoId": hub.ID}, nil); st != 200 {
		t.Fatal(st)
	}
	if st := paulo.pedir("PUT", "/api/pessoas/"+membroID+"/alcadas", map[string]any{"alcadas": []string{"editar_posts"}}, nil); st != 200 {
		t.Fatal(st)
	}
	if st := ana.pedir("PUT", "/api/sessao/organizacao", map[string]string{"organizacaoId": hub.ID}, &sessAna); st != 200 || !slices.Equal(sessAna.Alcadas, []string{"editar_posts"}) {
		t.Fatalf("alçadas ao trocar para Laura Hub inesperadas: %d %+v", st, sessAna.Alcadas)
	}
}

func TestAcessosB4Bloqueio403E404OutraOrg(t *testing.T) {
	api := NovaAPI(prepararBanco(t), false, nil)
	paulo := novoCliente(t, api)
	sessPaulo := entrarComoPaulo(t, paulo)
	growth, hub := sessPaulo.Organizacoes[0], sessPaulo.Organizacoes[1]

	// Cria um arquivo na Laura Hub (para testar AC18) e um usuário na Clínica Alheia.
	if st := paulo.pedir("PUT", "/api/sessao/organizacao", map[string]string{"organizacaoId": hub.ID}, nil); st != 200 {
		t.Fatal(st)
	}
	var arquivoHub ArquivoMarca
	if st := paulo.enviar("/api/kit/arquivos?tipo=logo&organizacaoId="+hub.ID, "hub.png", pngMinimo, &arquivoHub); st != 201 {
		t.Fatalf("enviar arquivo Hub falhou: %d", st)
	}
	if st := paulo.pedir("PUT", "/api/sessao/organizacao", map[string]string{"organizacaoId": growth.ID}, nil); st != 200 {
		t.Fatal(st)
	}

	var usuarioAlheioID string
	if err := api.banco.QueryRow(context.Background(), `SELECT id::text FROM usuarios WHERE email = 'outra@exemplo.com'`).Scan(&usuarioAlheioID); err != nil {
		t.Fatal(err)
	}

	// Membro da Laura Growth sem alçadas: todas as rotas protegidas retornam 403 (mesmo com IDs forjados de outra org).
	criarMembroNaOrg(t, api.banco, growth.ID, "Beto", "beto-b4@exemplo.com")
	beto := novoCliente(t, api)
	entrarComEmail(t, beto, "beto-b4@exemplo.com")

	casos403 := []struct {
		metodo string
		rota   string
		corpo  any
	}{
		{"GET", "/api/kit", nil},
		{"GET", "/api/kit/arquivos/" + arquivoHub.ID, nil},
		{"PUT", "/api/kit", map[string]any{"organizacaoId": hub.ID, "tomDeVoz": "hack"}},
		{"DELETE", "/api/kit/arquivos/" + arquivoHub.ID + "?organizacaoId=" + hub.ID, nil},
		{"GET", "/api/pessoas", nil},
		{"GET", "/api/pessoas/registro", nil},
		{"PUT", "/api/pessoas/" + usuarioAlheioID + "/alcadas", map[string]any{"alcadas": []string{"ver_marca_posts"}}},
		{"DELETE", "/api/pessoas/" + usuarioAlheioID, nil},
	}
	for _, caso := range casos403 {
		if st := beto.pedir(caso.metodo, caso.rota, caso.corpo, nil); st != 403 {
			t.Fatalf("%s %s sem alçada esperava 403, veio %d", caso.metodo, caso.rota, st)
		}
	}
	if st := beto.enviar("/api/kit/arquivos?tipo=logo&organizacaoId="+hub.ID, "x.png", pngMinimo, nil); st != 403 {
		t.Fatalf("POST /api/kit/arquivos sem alçada esperava 403, veio %d", st)
	}

	// Membro COM alçadas na Laura Growth tentando acessar recurso da Laura Hub (AC18) -> 404.
	criarMembroNaOrg(t, api.banco, growth.ID, "Carla", "carla-b4@exemplo.com", "ver_marca_posts", "cmo_dna", "gerenciar_pessoas")
	carla := novoCliente(t, api)
	entrarComEmail(t, carla, "carla-b4@exemplo.com")

	if st, _, _ := carla.baixar("/api/kit/arquivos/" + arquivoHub.ID); st != 404 {
		t.Fatalf("arquivo de outra org esperava 404, veio %d", st)
	}
	if st := carla.pedir("DELETE", "/api/kit/arquivos/"+arquivoHub.ID+"?organizacaoId="+growth.ID, nil, nil); st != 404 {
		t.Fatalf("remover arquivo de outra org esperava 404, veio %d", st)
	}
	if st := carla.pedir("PUT", "/api/pessoas/"+usuarioAlheioID+"/alcadas", map[string]any{"alcadas": []string{"ver_marca_posts"}}, nil); st != 404 {
		t.Fatalf("editar pessoa de outra org esperava 404, veio %d", st)
	}
	if st := carla.pedir("DELETE", "/api/pessoas/"+usuarioAlheioID, nil, nil); st != 404 {
		t.Fatalf("remover pessoa de outra org esperava 404, veio %d", st)
	}
	if st := carla.pedir("PUT", "/api/pessoas/id-malformado/alcadas", map[string]any{"alcadas": []string{}}, nil); st != 404 {
		t.Fatalf("id malformado esperava 404 (não 500), veio %d", st)
	}
}

func TestAcessosB5DonoIntocavelESemAutoEdicao(t *testing.T) {
	api := NovaAPI(prepararBanco(t), false, nil)
	paulo := novoCliente(t, api)
	sessPaulo := entrarComoPaulo(t, paulo)
	growth := sessPaulo.Organizacoes[0]

	// Dono tem todas as 8 alçadas na sessão mesmo sem linhas em alcadas_membro.
	if len(sessPaulo.Alcadas) != 8 {
		t.Fatalf("dono esperava 8 alçadas na sessão, veio %d: %+v", len(sessPaulo.Alcadas), sessPaulo.Alcadas)
	}

	// Membro com gerenciar_pessoas.
	gerenteID := criarMembroNaOrg(t, api.banco, growth.ID, "Daniel", "daniel-b5@exemplo.com", "gerenciar_pessoas")
	gerente := novoCliente(t, api)
	entrarComEmail(t, gerente, "daniel-b5@exemplo.com")

	// Não pode editar as próprias alçadas nem remover a si mesmo.
	if st := gerente.pedir("PUT", "/api/pessoas/"+gerenteID+"/alcadas", map[string]any{"alcadas": []string{"gerenciar_pessoas", "cmo_dna"}}, nil); st != 403 {
		t.Fatalf("autoedição de alçadas esperava 403, veio %d", st)
	}
	if st := gerente.pedir("DELETE", "/api/pessoas/"+gerenteID, nil, nil); st != 403 {
		t.Fatalf("autorremoção esperava 403, veio %d", st)
	}

	// Não pode editar alçadas do dono nem remover o dono.
	if st := gerente.pedir("PUT", "/api/pessoas/"+sessPaulo.Usuario.ID+"/alcadas", map[string]any{"alcadas": []string{}}, nil); st != 403 {
		t.Fatalf("editar alçadas do dono esperava 403, veio %d", st)
	}
	if st := gerente.pedir("DELETE", "/api/pessoas/"+sessPaulo.Usuario.ID, nil, nil); st != 403 {
		t.Fatalf("remover dono esperava 403, veio %d", st)
	}
	// Nem o próprio dono pode alterar/remover a si mesmo por essas rotas.
	if st := paulo.pedir("PUT", "/api/pessoas/"+sessPaulo.Usuario.ID+"/alcadas", map[string]any{"alcadas": []string{}}, nil); st != 403 {
		t.Fatalf("dono editar próprias alçadas esperava 403, veio %d", st)
	}
	if st := paulo.pedir("DELETE", "/api/pessoas/"+sessPaulo.Usuario.ID, nil, nil); st != 403 {
		t.Fatalf("dono remover a si mesmo esperava 403, veio %d", st)
	}
}

func TestAcessosB6RegistroAuditoriaERemocao(t *testing.T) {
	api := NovaAPI(prepararBanco(t), false, nil)
	paulo := novoCliente(t, api)
	sessPaulo := entrarComoPaulo(t, paulo)
	growth, hub := sessPaulo.Organizacoes[0], sessPaulo.Organizacoes[1]

	eduardoID := criarMembroNaOrg(t, api.banco, growth.ID, "Eduardo", "eduardo-b6@exemplo.com")
	// Também adiciona Eduardo em Laura Hub para testar isolamento do registro por organização.
	if _, err := api.banco.Exec(context.Background(), `
		INSERT INTO membros (organizacao_id, usuario_id, papel) VALUES ($1::uuid, $2::uuid, 'membro')`,
		hub.ID, eduardoID); err != nil {
		t.Fatal(err)
	}

	// Convite pendente e convite expirado em convites para conferir listagem em GET /api/pessoas.
	if _, err := api.banco.Exec(context.Background(), `
		INSERT INTO convites (organizacao_id, email, alcadas, token_hash, criado_por, expira_em, envio_estado)
		VALUES
		  ($1::uuid, 'pendente@exemplo.com', ARRAY['ver_marca_posts'], decode('010203', 'hex'), $2::uuid, $3, 'enviado'),
		  ($1::uuid, 'expirado@exemplo.com', ARRAY['ver_marca_posts'], decode('040506', 'hex'), $2::uuid, $4, 'enviado')`,
		growth.ID, sessPaulo.Usuario.ID, time.Now().Add(48*time.Hour), time.Now().Add(-1*time.Hour)); err != nil {
		t.Fatal(err)
	}

	// 1. Concede 2 alçadas.
	if st := paulo.pedir("PUT", "/api/pessoas/"+eduardoID+"/alcadas", map[string]any{
		"alcadas": []string{"ver_marca_posts", "editar_posts"},
	}, nil); st != 200 {
		t.Fatalf("conceder inicial falhou: %d", st)
	}
	// 2. Atualiza para manter editar_posts, revogar ver_marca_posts e conceder aprovar_posts.
	if st := paulo.pedir("PUT", "/api/pessoas/"+eduardoID+"/alcadas", map[string]any{
		"alcadas": []string{"editar_posts", "aprovar_posts"},
	}, nil); st != 200 {
		t.Fatalf("atualizar por diff falhou: %d", st)
	}

	var respPessoas RespostaPessoas
	if st := paulo.pedir("GET", "/api/pessoas", nil, &respPessoas); st != 200 {
		t.Fatalf("GET /api/pessoas falhou: %d", st)
	}
	if len(respPessoas.Membros) != 2 || len(respPessoas.Convites) != 1 || respPessoas.Convites[0].Email != "pendente@exemplo.com" {
		t.Fatalf("GET /api/pessoas inesperado: %+v", respPessoas)
	}
	if respPessoas.Membros[0].Papel != "dono" || len(respPessoas.Membros[0].Alcadas) != 8 {
		t.Fatalf("dono esperado primeiro com 8 alçadas em GET /api/pessoas: %+v", respPessoas.Membros[0])
	}
	if respPessoas.Membros[1].UsuarioID != eduardoID || respPessoas.Membros[1].Papel != "membro" || !slices.Equal(respPessoas.Membros[1].Alcadas, []string{"editar_posts", "aprovar_posts"}) {
		t.Fatalf("membro esperado com alçadas exatas em GET /api/pessoas: %+v", respPessoas.Membros[1])
	}

	// 3. Remove Eduardo da Laura Growth.
	eduardo := novoCliente(t, api)
	entrarComEmail(t, eduardo, "eduardo-b6@exemplo.com")
	if st := paulo.pedir("DELETE", "/api/pessoas/"+eduardoID, nil, nil); st != 204 {
		t.Fatalf("DELETE /api/pessoas/{id} esperava 204, veio %d", st)
	}
	// Eduardo perde acesso à Laura Growth imediatamente.
	if st := eduardo.pedir("GET", "/api/organizacao", nil, nil); st != 404 {
		t.Fatalf("removido da org ativa esperava 404, veio %d", st)
	}
	var sessEduardo Sessao
	if st := eduardo.pedir("GET", "/api/sessao", nil, &sessEduardo); st != 200 || sessEduardo.OrganizacaoAtiva.ID != hub.ID {
		t.Fatalf("após remoção esperava cair na Laura Hub, veio %d %+v", st, sessEduardo.OrganizacaoAtiva)
	}

	// Registro de acessos da Laura Growth mostra exatamente 5 eventos (2 concessões, 1 revogação, 1 concessão, 1 remoção).
	var reg RespRegistroParaTeste
	if st := paulo.pedir("GET", "/api/pessoas/registro", nil, &reg); st != 200 {
		t.Fatalf("GET /api/pessoas/registro falhou: %d", st)
	}
	if len(reg.Registros) != 5 {
		t.Fatalf("esperava 5 registros de auditoria (por diff + remoção), veio %d: %+v", len(reg.Registros), reg.Registros)
	}
	for _, r := range reg.Registros {
		if r.AtorID == nil || *r.AtorID != sessPaulo.Usuario.ID || r.AlvoEmail == nil || *r.AlvoEmail != "eduardo-b6@exemplo.com" {
			t.Fatalf("registro sem ator/alvo completos: %+v", r)
		}
	}
	if reg.Registros[0].Acao != "membro_removido" {
		t.Fatalf("registro mais recente esperava membro_removido, veio %+v", reg.Registros[0])
	}

	// Validação do parâmetro ?limite= em GET /api/pessoas/registro.
	for _, invalido := range []string{"0", "201", "abc"} {
		if st := paulo.pedir("GET", "/api/pessoas/registro?limite="+invalido, nil, nil); st != 400 {
			t.Fatalf("limite=%s esperava 400, veio %d", invalido, st)
		}
	}
	if st := paulo.pedir("GET", "/api/pessoas/registro?limite=2", nil, &reg); st != 200 || len(reg.Registros) != 2 {
		t.Fatalf("limite=2 esperava 2 registros, veio %d (%d itens)", st, len(reg.Registros))
	}

	// Registro de acessos é somente inserção (append-only) no próprio banco.
	if _, err := api.banco.Exec(context.Background(), `UPDATE registro_acessos SET acao = 'adulterado'`); err == nil {
		t.Fatal("UPDATE em registro_acessos deveria falhar no banco")
	}
	if _, err := api.banco.Exec(context.Background(), `DELETE FROM registro_acessos`); err == nil {
		t.Fatal("DELETE em registro_acessos deveria falhar no banco")
	}

	// Registro da Laura Hub está vazio (isolamento por organização).
	if st := paulo.pedir("PUT", "/api/sessao/organizacao", map[string]string{"organizacaoId": hub.ID}, nil); st != 200 {
		t.Fatal(st)
	}
	if st := paulo.pedir("GET", "/api/pessoas/registro", nil, &reg); st != 200 || len(reg.Registros) != 0 {
		t.Fatalf("registro da Laura Hub deveria estar vazio, veio %d %+v", st, reg.Registros)
	}
}

type RespRegistroParaTeste = RespostaRegistroAcessos

func TestAcessosB7ConjuntosProntos(t *testing.T) {
	api := NovaAPI(prepararBanco(t), false, nil)
	paulo := novoCliente(t, api)
	sessPaulo := entrarComoPaulo(t, paulo)
	growth := sessPaulo.Organizacoes[0]

	fabioID := criarMembroNaOrg(t, api.banco, growth.ID, "Fábio", "fabio-b7@exemplo.com")

	var resp RespostaPessoas
	if st := paulo.pedir("GET", "/api/pessoas", nil, &resp); st != 200 {
		t.Fatalf("GET /api/pessoas falhou: %d", st)
	}
	if len(resp.Conjuntos) != 4 {
		t.Fatalf("esperava 4 conjuntos prontos, veio %d", len(resp.Conjuntos))
	}

	var gestor ConjuntoAlcadas
	for _, c := range resp.Conjuntos {
		if c.Codigo == "gestor_marketing" {
			gestor = c
		}
	}
	if len(gestor.Alcadas) != 7 || slices.Contains(gestor.Alcadas, "gerenciar_pessoas") {
		t.Fatalf("conjunto gestor_marketing inesperado: %+v", gestor)
	}

	// Aplica as alçadas do conjunto "gestor_marketing" e depois desmarca apenas "conectar_redes".
	var salvo PessoaMembro
	if st := paulo.pedir("PUT", "/api/pessoas/"+fabioID+"/alcadas", map[string]any{"alcadas": gestor.Alcadas}, &salvo); st != 200 || len(salvo.Alcadas) != 7 {
		t.Fatalf("salvar conjunto falhou: %d %+v", st, salvo)
	}
	reduzidas := make([]string, 0, len(salvo.Alcadas))
	for _, al := range salvo.Alcadas {
		if al != "conectar_redes" {
			reduzidas = append(reduzidas, al)
		}
	}
	if st := paulo.pedir("PUT", "/api/pessoas/"+fabioID+"/alcadas", map[string]any{"alcadas": reduzidas}, &salvo); st != 200 || len(salvo.Alcadas) != 6 || slices.Contains(salvo.Alcadas, "conectar_redes") {
		t.Fatalf("ajustar caixa após conjunto falhou: %d %+v", st, salvo)
	}

	// Validações de entrada.
	if st := paulo.pedir("PUT", "/api/pessoas/"+fabioID+"/alcadas", map[string]any{"alcadas": []string{"inexistente"}}, nil); st != 400 {
		t.Fatalf("alçada inválida esperava 400, veio %d", st)
	}
	if st := paulo.pedir("PUT", "/api/pessoas/"+fabioID+"/alcadas", map[string]any{}, nil); st != 400 {
		t.Fatalf("lista ausente esperava 400, veio %d", st)
	}
	// Campo extra desconhecido (ex.: tentar vincular conjuntoId) é recusado.
	if st := paulo.pedir("PUT", "/api/pessoas/"+fabioID+"/alcadas", map[string]any{"alcadas": []string{"ver_marca_posts"}, "conjuntoId": "gestor_marketing"}, nil); st != 400 {
		t.Fatalf("campo desconhecido esperava 400, veio %d", st)
	}
}

func TestAcessosB8AdminEquipe(t *testing.T) {
	api := NovaAPI(prepararBanco(t), false, nil)
	paulo := novoCliente(t, api)
	sessPaulo := entrarComoPaulo(t, paulo)
	growth, hub := sessPaulo.Organizacoes[0], sessPaulo.Organizacoes[1]

	// Membro da casa (Laura Growth e Laura Hub) e membro fora da casa (Clínica Alheia).
	membroCasaID := criarMembroNaOrg(t, api.banco, growth.ID, "Gabi", "gabi-b8@exemplo.com")
	if _, err := api.banco.Exec(context.Background(), `
		INSERT INTO membros (organizacao_id, usuario_id, papel) VALUES ($1::uuid, $2::uuid, 'membro')`,
		hub.ID, membroCasaID); err != nil {
		t.Fatal(err)
	}
	membroForaID := criarUsuarioComum(t, api.banco, "fora-b8@exemplo.com")

	// Não-admin não acessa /api/admin/equipe.
	gabi := novoCliente(t, api)
	entrarComEmail(t, gabi, "gabi-b8@exemplo.com")
	if st := gabi.pedir("GET", "/api/admin/equipe", nil, nil); st != 403 {
		t.Fatalf("não-admin GET /api/admin/equipe esperava 403, veio %d", st)
	}
	if st := gabi.pedir("PUT", "/api/admin/equipe/"+membroCasaID, map[string]any{"adminPlataforma": true}, nil); st != 403 {
		t.Fatalf("não-admin PUT /api/admin/equipe esperava 403, veio %d", st)
	}

	// Admin vê apenas membros das organizações da casa (sem duplicar quem está nas duas).
	var resp RespostaEquipeAdmin
	if st := paulo.pedir("GET", "/api/admin/equipe", nil, &resp); st != 200 {
		t.Fatalf("GET /api/admin/equipe falhou: %d", st)
	}
	if len(resp.Equipe) != 2 {
		t.Fatalf("esperava 2 membros da casa (Paulo e Gabi), veio %d: %+v", len(resp.Equipe), resp.Equipe)
	}
	for _, m := range resp.Equipe {
		if m.UsuarioID == membroForaID {
			t.Fatalf("membro fora da casa apareceu na equipe: %+v", m)
		}
		if m.UsuarioID == membroCasaID && len(m.Organizacoes) != 2 {
			t.Fatalf("Gabi deveria aparecer 1 vez com 2 orgs da casa, veio %+v", m)
		}
	}

	// Recusa tornar admin alguém que não é de organização da casa.
	if st, body := paulo.pedirComCorpo("PUT", "/api/admin/equipe/"+membroForaID, map[string]any{"adminPlataforma": true}); st != 403 || !strings.Contains(string(body), "nao_membro_casa") {
		t.Fatalf("promover membro fora da casa esperava 403 nao_membro_casa, veio %d (%s)", st, body)
	}

	// Não permite remover o último administrador (Paulo).
	if st, body := paulo.pedirComCorpo("PUT", "/api/admin/equipe/"+sessPaulo.Usuario.ID, map[string]any{"adminPlataforma": false}); st != 409 || !strings.Contains(string(body), "ultimo_admin") {
		t.Fatalf("remover último admin esperava 409 ultimo_admin, veio %d (%s)", st, body)
	}

	// Concede admin da plataforma para Gabi -> auditado com organizacao_id NULL.
	var atualizado MembroEquipe
	if st := paulo.pedir("PUT", "/api/admin/equipe/"+membroCasaID, map[string]any{"adminPlataforma": true}, &atualizado); st != 200 || !atualizado.AdminPlataforma {
		t.Fatalf("promover Gabi falhou: %d %+v", st, atualizado)
	}
	// Gabi passa a acessar administração imediatamente.
	if st := gabi.pedir("GET", "/api/admin/equipe", nil, &resp); st != 200 {
		t.Fatalf("Gabi já deveria acessar /api/admin/equipe, veio %d", st)
	}

	// Agora que há 2 admins, pode revogar Paulo (e depois re-conceder Paulo e revogar Gabi).
	if st := gabi.pedir("PUT", "/api/admin/equipe/"+sessPaulo.Usuario.ID, map[string]any{"adminPlataforma": false}, &atualizado); st != 200 || atualizado.AdminPlataforma {
		t.Fatalf("revogar Paulo com 2 admins falhou: %d %+v", st, atualizado)
	}
	// Com Gabi como única admin, tentar revogá-la falha com 409.
	if st := gabi.pedir("PUT", "/api/admin/equipe/"+membroCasaID, map[string]any{"adminPlataforma": false}, nil); st != 409 {
		t.Fatalf("remover última admin esperava 409, veio %d", st)
	}

	// Confere auditoria em registro_acessos com organizacao_id NULL e sem vazar para /api/pessoas/registro.
	if st := gabi.pedir("GET", "/api/admin/equipe", nil, &resp); st != 200 || len(resp.Registros) != 2 {
		t.Fatalf("esperava 2 registros de auditoria de plataforma, veio %d %+v", st, resp.Registros)
	}
	if resp.Registros[0].Acao != "admin_plataforma_revogado" || resp.Registros[1].Acao != "admin_plataforma_concedido" {
		t.Fatalf("ações de auditoria inesperadas: %+v", resp.Registros)
	}
	var nullCount int
	if err := api.banco.QueryRow(context.Background(), `SELECT count(*) FROM registro_acessos WHERE organizacao_id IS NULL`).Scan(&nullCount); err != nil {
		t.Fatal(err)
	}
	if nullCount != 2 {
		t.Fatalf("esperava 2 linhas em registro_acessos com organizacao_id NULL, veio %d", nullCount)
	}

	// Garante que registros de plataforma (organizacao_id NULL) não aparecem no registro da organização.
	var regOrg RespostaRegistroAcessos
	if st := paulo.pedir("GET", "/api/pessoas/registro", nil, &regOrg); st != 200 || len(regOrg.Registros) != 0 {
		t.Fatalf("registro da organização não deve incluir registros de plataforma: %d %+v", st, regOrg.Registros)
	}

	// G-1 / S-1: admin fora das organizações da casa aparece na lista e pode ser revogado,
	// mas não pode voltar a receber admin sem entrar em uma organização da casa.
	if st := gabi.pedir("PUT", "/api/admin/equipe/"+sessPaulo.Usuario.ID, map[string]any{"adminPlataforma": true}, &atualizado); st != 200 {
		t.Fatalf("repromover Paulo falhou: %d", st)
	}
	if st := paulo.pedir("PUT", "/api/admin/equipe/"+membroCasaID, map[string]any{"adminPlataforma": true}, nil); st != 200 {
		t.Fatalf("manter Gabi admin falhou: %d", st)
	}
	// Remove Gabi das duas organizações da casa (Laura Growth e Laura Hub), mantendo-a apenas com admin_plataforma=true.
	if st := paulo.pedir("DELETE", "/api/pessoas/"+membroCasaID, nil, nil); st != 204 {
		t.Fatalf("remover Gabi da Laura Growth falhou: %d", st)
	}
	if st := paulo.pedir("PUT", "/api/sessao/organizacao", map[string]string{"organizacaoId": hub.ID}, nil); st != 200 {
		t.Fatal(st)
	}
	if st := paulo.pedir("DELETE", "/api/pessoas/"+membroCasaID, nil, nil); st != 204 {
		t.Fatalf("remover Gabi da Laura Hub falhou: %d", st)
	}
	if st := paulo.pedir("GET", "/api/admin/equipe", nil, &resp); st != 200 {
		t.Fatalf("GET /api/admin/equipe falhou: %d", st)
	}
	encontrouGabiSemCasa := false
	for _, m := range resp.Equipe {
		if m.UsuarioID == membroCasaID && m.AdminPlataforma && len(m.Organizacoes) == 0 {
			encontrouGabiSemCasa = true
		}
	}
	if !encontrouGabiSemCasa {
		t.Fatalf("admin fora das orgs da casa deveria continuar visível em GET /api/admin/equipe: %+v", resp.Equipe)
	}
	if st := paulo.pedir("PUT", "/api/admin/equipe/"+membroCasaID, map[string]any{"adminPlataforma": false}, &atualizado); st != 200 || atualizado.AdminPlataforma {
		t.Fatalf("revogar admin fora das orgs da casa esperava 200, veio %d %+v", st, atualizado)
	}
	if st := paulo.pedir("PUT", "/api/admin/equipe/"+membroCasaID, map[string]any{"adminPlataforma": true}, nil); st != 403 {
		t.Fatalf("conceder admin a quem não está em org da casa esperava 403, veio %d", st)
	}
	if st := paulo.pedir("PUT", "/api/admin/equipe/id-invalido", map[string]any{"adminPlataforma": false}, nil); st != 404 {
		t.Fatalf("id inválido em /api/admin/equipe esperava 404, veio %d", st)
	}

	// JSON marshaling sanity check (garante detalhe válido).
	raw, _ := json.Marshal(resp.Registros[0].Detalhe)
	if !strings.Contains(string(raw), "adminPlataforma") {
		t.Fatalf("detalhe inesperado: %s", raw)
	}
}
