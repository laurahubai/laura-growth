package main

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func TestAdminOrganizacoesB11(t *testing.T) {
	api, fake := novaAPIConvitesTeste(t)

	paulo := novoCliente(t, api)
	sessPaulo := entrarComoPaulo(t, paulo)
	growth := sessPaulo.Organizacoes[0]

	// Descobre a organização alheia (Clínica Alheia) e seu dono; Paulo NÃO é membro dela.
	var clinicaID, clinicaDonoID string
	if err := api.banco.QueryRow(context.Background(), `
		SELECT o.id::text, m.usuario_id::text
		FROM organizacoes o
		JOIN membros m ON m.organizacao_id = o.id AND m.papel = 'dono'
		WHERE o.nome = 'Clínica Alheia'`).Scan(&clinicaID, &clinicaDonoID); err != nil {
		t.Fatal(err)
	}

	// Cria um membro comum com gerenciar_pessoas em Laura Growth (NÃO é admin_plataforma).
	gerenteID := criarMembroNaOrg(t, api.banco, growth.ID, "Gerente Comum", "gerente-b11@exemplo.com", "gerenciar_pessoas")
	gerente := novoCliente(t, api)
	entrarComEmail(t, gerente, "gerente-b11@exemplo.com")

	// Cria membro na Clínica Alheia para testar remoção e edição via admin da plataforma.
	membroClinicaID := criarMembroNaOrg(t, api.banco, clinicaID, "Membro Clínica", "membro-clinica-b11@exemplo.com", "ver_marca_posts")

	// 1. Não-admin recebe 403 em TODAS as rotas /api/admin/organizacoes* (e anônimo recebe 401).
	anon := novoCliente(t, api)
	const idQualquer = "00000000-0000-4000-8000-000000000111"
	rotasAdminOrgs := []struct {
		metodo string
		rota   string
		corpo  any
	}{
		{"GET", "/api/admin/organizacoes", nil},
		{"GET", "/api/admin/organizacoes/" + clinicaID + "/pessoas", nil},
		{"GET", "/api/admin/organizacoes/" + clinicaID + "/pessoas/registro", nil},
		{"PUT", "/api/admin/organizacoes/" + clinicaID + "/pessoas/" + membroClinicaID + "/alcadas", map[string]any{"alcadas": []string{"ver_marca_posts"}}},
		{"DELETE", "/api/admin/organizacoes/" + clinicaID + "/pessoas/" + membroClinicaID, nil},
		{"POST", "/api/admin/organizacoes/" + clinicaID + "/convites", map[string]any{"email": "bloqueado-b11@exemplo.com", "alcadas": []string{"ver_marca_posts"}}},
		{"POST", "/api/admin/organizacoes/" + clinicaID + "/convites/" + idQualquer + "/reenviar", nil},
		{"DELETE", "/api/admin/organizacoes/" + clinicaID + "/convites/" + idQualquer, nil},
		{"GET", "/api/admin/organizacoes/rota-desconhecida", nil},
	}
	for _, r := range rotasAdminOrgs {
		if st := anon.pedir(r.metodo, r.rota, r.corpo, nil); st != 401 {
			t.Fatalf("anônimo %s %s esperava 401, veio %d", r.metodo, r.rota, st)
		}
		if st := gerente.pedir(r.metodo, r.rota, r.corpo, nil); st != 403 {
			t.Fatalf("não-admin %s %s esperava 403, veio %d", r.metodo, r.rota, st)
		}
	}
	// Mesmo dono da Clínica Alheia (não-admin da plataforma) recebe 403 nas rotas /api/admin/organizacoes*.
	var emailDonoClinica string
	if err := api.banco.QueryRow(context.Background(), `SELECT email FROM usuarios WHERE id = $1::uuid`, clinicaDonoID).Scan(&emailDonoClinica); err != nil {
		t.Fatal(err)
	}
	hashDono, err := gerarHashSenha(senhaTeste)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := api.banco.Exec(context.Background(), `UPDATE usuarios SET senha_hash = $1 WHERE id = $2::uuid`, hashDono, clinicaDonoID); err != nil {
		t.Fatal(err)
	}
	donoClinica := novoCliente(t, api)
	entrarComEmail(t, donoClinica, emailDonoClinica)
	if st := donoClinica.pedir("GET", "/api/admin/organizacoes/"+clinicaID+"/pessoas", nil, nil); st != 403 {
		t.Fatalf("dono da org sem admin_plataforma esperava 403, veio %d", st)
	}

	// 2. Organização desconhecida ou UUID malformado retorna 404 para admin da plataforma.
	const orgInexistente = "00000000-0000-4000-8000-000000000999"
	for _, orgTeste := range []string{"id-invalido", orgInexistente} {
		casos404 := []struct {
			metodo string
			rota   string
			corpo  any
		}{
			{"GET", "/api/admin/organizacoes/" + orgTeste + "/pessoas", nil},
			{"GET", "/api/admin/organizacoes/" + orgTeste + "/pessoas/registro", nil},
			{"PUT", "/api/admin/organizacoes/" + orgTeste + "/pessoas/" + sessPaulo.Usuario.ID + "/alcadas", map[string]any{"alcadas": []string{}}},
			{"DELETE", "/api/admin/organizacoes/" + orgTeste + "/pessoas/" + sessPaulo.Usuario.ID, nil},
			{"POST", "/api/admin/organizacoes/" + orgTeste + "/convites", map[string]any{"email": "x@exemplo.com", "alcadas": []string{"ver_marca_posts"}}},
			{"POST", "/api/admin/organizacoes/" + orgTeste + "/convites/" + idQualquer + "/reenviar", nil},
			{"DELETE", "/api/admin/organizacoes/" + orgTeste + "/convites/" + idQualquer, nil},
		}
		for _, c := range casos404 {
			if st := paulo.pedir(c.metodo, c.rota, c.corpo, nil); st != 404 {
				t.Fatalf("org desconhecida (%s) %s %s esperava 404, veio %d", orgTeste, c.metodo, c.rota, st)
			}
		}
	}

	// 3. Admin da plataforma (não membro da Clínica Alheia) lista organizações e vê id, nome e contagem de membros.
	var respOrgs RespostaOrganizacoesAdmin
	if st := paulo.pedir("GET", "/api/admin/organizacoes", nil, &respOrgs); st != 200 {
		t.Fatalf("GET /api/admin/organizacoes falhou: %d", st)
	}
	if len(respOrgs.Organizacoes) != 3 {
		t.Fatalf("esperava 3 organizações, veio %d: %+v", len(respOrgs.Organizacoes), respOrgs.Organizacoes)
	}
	var achouClinica bool
	for _, o := range respOrgs.Organizacoes {
		if o.ID == clinicaID {
			achouClinica = true
			if o.Nome != "Clínica Alheia" || o.Membros != 2 || o.DaCasa {
				t.Fatalf("dados inesperados da Clínica Alheia: %+v", o)
			}
		}
	}
	if !achouClinica {
		t.Fatalf("Clínica Alheia não apareceu na listagem: %+v", respOrgs.Organizacoes)
	}

	// 4. Admin não membro lista pessoas da Clínica Alheia.
	var respPessoas RespostaPessoas
	if st := paulo.pedir("GET", "/api/admin/organizacoes/"+clinicaID+"/pessoas", nil, &respPessoas); st != 200 {
		t.Fatalf("GET pessoas da Clínica Alheia falhou: %d", st)
	}
	if len(respPessoas.Membros) != 2 {
		t.Fatalf("esperava 2 membros na Clínica Alheia, veio %d", len(respPessoas.Membros))
	}

	// 5. Regra do dono protegida via admin da plataforma.
	if st, body := paulo.pedirComCorpo("PUT", "/api/admin/organizacoes/"+clinicaID+"/pessoas/"+clinicaDonoID+"/alcadas", map[string]any{"alcadas": []string{}}); st != 403 || !strings.Contains(string(body), "dono_protegido") {
		t.Fatalf("editar alçadas do dono via admin esperava 403 dono_protegido, veio %d (%s)", st, body)
	}
	if st, body := paulo.pedirComCorpo("DELETE", "/api/admin/organizacoes/"+clinicaID+"/pessoas/"+clinicaDonoID, nil); st != 403 || !strings.Contains(string(body), "dono_protegido") {
		t.Fatalf("remover dono via admin esperava 403 dono_protegido, veio %d (%s)", st, body)
	}

	// 6. Regra de autoedição e autorremoção mantida (admin que É membro não-dono de Laura Growth não altera/remove a si mesmo).
	if _, err := api.banco.Exec(context.Background(), `UPDATE usuarios SET admin_plataforma = true WHERE id = $1::uuid`, gerenteID); err != nil {
		t.Fatal(err)
	}
	if st, body := gerente.pedirComCorpo("PUT", "/api/admin/organizacoes/"+growth.ID+"/pessoas/"+gerenteID+"/alcadas", map[string]any{"alcadas": []string{"ver_marca_posts"}}); st != 403 || !strings.Contains(string(body), "auto_edicao") {
		t.Fatalf("autoedição via admin esperava 403 auto_edicao, veio %d (%s)", st, body)
	}
	if st, body := gerente.pedirComCorpo("DELETE", "/api/admin/organizacoes/"+growth.ID+"/pessoas/"+gerenteID, nil); st != 403 || !strings.Contains(string(body), "auto_remocao") {
		t.Fatalf("autorremoção via admin esperava 403 auto_remocao, veio %d (%s)", st, body)
	}

	// 7. Admin não membro convida pessoa para Clínica Alheia, reenvia, cancela outro e convidado aceita.
	var convCriado RespostaConviteCriado
	if st := paulo.pedir("POST", "/api/admin/organizacoes/"+clinicaID+"/convites", map[string]any{
		"email":   "convidado-b11@exemplo.com",
		"alcadas": []string{"ver_marca_posts"},
	}, &convCriado); st != 201 {
		t.Fatalf("criar convite via admin falhou: %d", st)
	}
	if fake.total() != 1 {
		t.Fatalf("esperava 1 envio no adaptador de e-mail, veio %d", fake.total())
	}

	var convReenviado RespostaConviteCriado
	if st := paulo.pedir("POST", "/api/admin/organizacoes/"+clinicaID+"/convites/"+convCriado.Convite.ID+"/reenviar", nil, &convReenviado); st != 200 {
		t.Fatalf("reenviar convite via admin falhou: %d", st)
	}
	tokenNovo := extrairTokenDeLink(t, convReenviado.Link)

	// Convite criado por admin da plataforma (não membro) pode ser lido e aceito normalmente.
	convidado := novoCliente(t, api)
	var convPub ConvitePublico
	if st := convidado.pedir("GET", "/api/convites/"+tokenNovo, nil, &convPub); st != 200 || convPub.OrganizacaoNome != "Clínica Alheia" {
		t.Fatalf("GET convite público criado por admin falhou: %d %+v", st, convPub)
	}
	var sessConvidado Sessao
	if st := convidado.pedir("POST", "/api/convites/"+tokenNovo+"/aceitar", map[string]any{
		"nome":  "Convidado B11",
		"senha": "senha-segura-123",
	}, &sessConvidado); st != 200 {
		t.Fatalf("aceitar convite criado por admin falhou: %d", st)
	}
	if sessConvidado.OrganizacaoAtiva.ID != clinicaID || !slices.Equal(sessConvidado.Alcadas, []string{"ver_marca_posts"}) {
		t.Fatalf("sessão do convidado inesperada: %+v", sessConvidado)
	}

	// Outro convite é cancelado via admin.
	var convCancelar RespostaConviteCriado
	if st := paulo.pedir("POST", "/api/admin/organizacoes/"+clinicaID+"/convites", map[string]any{
		"email":   "cancelar-b11@exemplo.com",
		"alcadas": []string{"ver_marca_posts"},
	}, &convCancelar); st != 201 {
		t.Fatal(st)
	}
	if st := paulo.pedir("DELETE", "/api/admin/organizacoes/"+clinicaID+"/convites/"+convCancelar.Convite.ID, nil, nil); st != 204 {
		t.Fatalf("cancelar convite via admin esperava 204, veio %d", st)
	}

	// 8. Admin não membro altera alçadas de membro da Clínica Alheia (efeito imediato) e remove outro membro.
	var salvo PessoaMembro
	if st := paulo.pedir("PUT", "/api/admin/organizacoes/"+clinicaID+"/pessoas/"+sessConvidado.Usuario.ID+"/alcadas", map[string]any{
		"alcadas": []string{"ver_marca_posts", "cmo_dna"},
	}, &salvo); st != 200 || !slices.Equal(salvo.Alcadas, []string{"ver_marca_posts", "cmo_dna"}) {
		t.Fatalf("editar alçadas via admin falhou: %d %+v", st, salvo)
	}
	if st := convidado.pedir("PUT", "/api/kit", map[string]any{"organizacaoId": clinicaID, "tomDeVoz": "Direto"}, nil); st != 200 {
		t.Fatalf("nova alçada via admin deveria valer no próximo pedido, veio %d", st)
	}

	if st := paulo.pedir("DELETE", "/api/admin/organizacoes/"+clinicaID+"/pessoas/"+membroClinicaID, nil, nil); st != 204 {
		t.Fatalf("remover membro via admin falhou: %d", st)
	}

	// 9. Auditoria na Clínica Alheia tem ator = Paulo (admin da plataforma) e via_admin_plataforma = true.
	var reg RespostaRegistroAcessos
	if st := paulo.pedir("GET", "/api/admin/organizacoes/"+clinicaID+"/pessoas/registro", nil, &reg); st != 200 {
		t.Fatalf("GET registro via admin falhou: %d", st)
	}
	if len(reg.Registros) < 6 {
		t.Fatalf("esperava pelo menos 6 registros na Clínica Alheia, veio %d", len(reg.Registros))
	}
	var acoesAdmin []string
	for _, r := range reg.Registros {
		if r.Acao == "convite_aceito" {
			if r.AtorID == nil || *r.AtorID != sessConvidado.Usuario.ID {
				t.Fatalf("convite_aceito deveria ter o convidado como ator: %+v", r)
			}
			continue
		}
		acoesAdmin = append(acoesAdmin, r.Acao)
		if r.AtorID == nil || *r.AtorID != sessPaulo.Usuario.ID {
			t.Fatalf("registro %s deveria ter Paulo como ator, veio %+v", r.Acao, r)
		}
		if v, ok := r.Detalhe["via_admin_plataforma"].(bool); !ok || !v {
			raw, _ := json.Marshal(r.Detalhe)
			t.Fatalf("registro %s sem via_admin_plataforma=true: %s", r.Acao, raw)
		}
	}
	for _, esperada := range []string{"convite_criado", "convite_reenviado", "convite_revogado", "alcada_concedida", "membro_removido"} {
		if !slices.Contains(acoesAdmin, esperada) {
			t.Fatalf("ação %s ausente nos registros via admin: %+v", esperada, acoesAdmin)
		}
	}
}
