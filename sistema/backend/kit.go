package main

// Etapa 2 e 2C: kit de marca / DNA da marca da organização ativa (MKT-01, MKT-03, MKT-04, RF-02, RF-10).
// Toda leitura e gravação usa a organização ativa da sessão, conferindo que a
// pessoa é membro dela e possui a alçada exigida. Pedidos de gravação trazem também
// a organização que a tela está mostrando e a versão base para evitar sobrescrita concorrente.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	maxPilares        = 5
	maxCores          = 5
	maxTextoKit       = 5000
	maxTextoPilar     = 500
	maxTamanhoArquivo = 8 << 20 // 8 MB
	maxReferencias    = 30
	maxProdutos       = 20
	maxNomeProduto    = 500
	versoesPorPagina  = 20
	maxPaginaVersoes  = 1_000_000
)

const msgKitDesatualizado = "O kit foi alterado por outra pessoa. Recarregue para ver a versão atual."

// ---- Tipos (espelham packages/contratos/src/index.ts) ----

type ProdutoKit struct {
	Nome        string `json:"nome"`
	Prioritario bool   `json:"prioritario"`
}

// DadosKitMarca contém os campos do kit sem organizacaoId (espelha KitMarca.omit({ organizacaoId }) e VersaoKit.dados).
type DadosKitMarca struct {
	Site            string       `json:"site"`
	Perfis          string       `json:"perfis"`
	TomDeVoz        string       `json:"tomDeVoz"`
	Publico         string       `json:"publico"`
	Temas           string       `json:"temas"`
	NuncaDizer      string       `json:"nuncaDizer"`
	Fontes          string       `json:"fontes"`
	Exemplos        string       `json:"exemplos"`
	Pilares         []string     `json:"pilares"`
	Cores           []string     `json:"cores"`
	Produtos        []ProdutoKit `json:"produtos"`
	Diferencial     string       `json:"diferencial"`
	ModeloNegocio   string       `json:"modeloNegocio"`
	ModeloOutro     string       `json:"modeloOutro"`
	Regioes         string       `json:"regioes"`
	ComoCompram     string       `json:"comoCompram"`
	MaiorDesafio    string       `json:"maiorDesafio"`
	Objetivo90Dias  string       `json:"objetivo90Dias"`
	Canais          []string     `json:"canais"`
	CanaisOutro     string       `json:"canaisOutro"`
	RestricoesSetor string       `json:"restricoesSetor"`
	AtualizadoEm    *time.Time   `json:"atualizadoEm"`
}

type KitMarca struct {
	OrganizacaoID string `json:"organizacaoId"`
	DadosKitMarca
}

type OrigemCampo struct {
	Origem      string    `json:"origem"`
	DefinidoPor *string   `json:"definidoPor"`
	DefinidoEm  time.Time `json:"definidoEm"`
}

// origemVersaoDB representa o registro gravado em kit_versoes.origens (apenas UUID, sem nomes por LGPD).
type origemVersaoDB struct {
	Origem        string    `json:"origem"`
	DefinidoPorID *string   `json:"definidoPorId"`
	DefinidoEm    time.Time `json:"definidoEm"`
}

type ResultadoSugestoesIA struct {
	Aplicados []string
	Ignorados []string
}

type ArquivoMarca struct {
	ID       string    `json:"id"`
	Tipo     string    `json:"tipo"`
	Nome     string    `json:"nome"`
	Mime     string    `json:"mime"`
	Tamanho  int       `json:"tamanho"`
	CriadoEm time.Time `json:"criadoEm"`
}

type RespostaKit struct {
	Kit      KitMarca               `json:"kit"`
	Arquivos []ArquivoMarca         `json:"arquivos"`
	Origens  map[string]OrigemCampo `json:"origens"`
	Versao   int                    `json:"versao"`
}

type pedidoSalvarKit struct {
	KitMarca
	VersaoBase int `json:"versaoBase"`
}

type PedidoConfirmarKit struct {
	OrganizacaoID string   `json:"organizacaoId"`
	Campos        []string `json:"campos"`
	VersaoBase    int      `json:"versaoBase"`
}

type ResumoVersaoKit struct {
	Numero          int       `json:"numero"`
	AutorTipo       string    `json:"autorTipo"`
	AutorNome       *string   `json:"autorNome"`
	CamposAlterados []string  `json:"camposAlterados"`
	CriadoEm        time.Time `json:"criadoEm"`
}

type RespostaVersoesKit struct {
	Versoes []ResumoVersaoKit `json:"versoes"`
	Pagina  int               `json:"pagina"`
	Total   int               `json:"total"`
}

type VersaoKit struct {
	ResumoVersaoKit
	Dados   DadosKitMarca          `json:"dados"`
	Origens map[string]OrigemCampo `json:"origens"`
}

var modelosNegocioValidos = map[string]bool{
	"":      true,
	"b2b":   true,
	"b2c":   true,
	"b2b2c": true,
	"outro": true,
}

var canaisKitValidos = map[string]bool{
	"instagram":          true,
	"facebook":           true,
	"linkedin":           true,
	"tiktok":             true,
	"youtube":            true,
	"google_meu_negocio": true,
	"site_blog":          true,
	"whatsapp":           true,
	"email":              true,
}

var camposKitOrdenados = []string{
	"site", "perfis", "tomDeVoz", "publico", "temas", "nuncaDizer", "fontes", "exemplos",
	"pilares", "cores", "produtos", "diferencial", "modeloNegocio", "regioes",
	"comoCompram", "maiorDesafio", "objetivo90Dias", "canais", "restricoesSetor",
}

var camposKitValidos = func() map[string]bool {
	m := make(map[string]bool, len(camposKitOrdenados))
	for _, c := range camposKitOrdenados {
		m[c] = true
	}
	return m
}()

func (a *API) rotasKit(m *http.ServeMux) {
	m.HandleFunc("GET /api/kit", a.comSessao(a.verKit))
	m.HandleFunc("PUT /api/kit", a.comSessao(a.salvarKit))
	m.HandleFunc("POST /api/kit/confirmar", a.comSessao(a.confirmarCamposKit))
	m.HandleFunc("GET /api/kit/versoes", a.comSessao(a.listarVersoesKit))
	m.HandleFunc("GET /api/kit/versoes/{numero}", a.comSessao(a.verVersaoKit))
	m.HandleFunc("POST /api/kit/arquivos", a.comSessao(a.enviarArquivo))
	m.HandleFunc("GET /api/kit/arquivos/{id}", a.comSessao(a.baixarArquivo))
	m.HandleFunc("DELETE /api/kit/arquivos/{id}", a.comSessao(a.removerArquivo))
}

var errSemOrganizacao = errors.New("sem organização ativa")

// orgAtiva devolve a organização ativa da sessão, só se a pessoa ainda for membro dela.
func (a *API) orgAtiva(ctx context.Context, s contextoSessao) (string, error) {
	if !uuidValido(s.orgAtiva) || !uuidValido(s.usuario.ID) {
		return "", errSemOrganizacao
	}
	var ok bool
	err := a.banco.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM membros WHERE usuario_id = $1::uuid AND organizacao_id = $2::uuid)`,
		s.usuario.ID, s.orgAtiva).Scan(&ok)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errSemOrganizacao
	}
	return s.orgAtiva, nil
}

type consultaKitSQL interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func kitVazio(org string) KitMarca {
	return KitMarca{
		OrganizacaoID: org,
		DadosKitMarca: DadosKitMarca{
			Pilares:  []string{},
			Cores:    []string{},
			Produtos: []ProdutoKit{},
			Canais:   []string{},
		},
	}
}

func normalizarFatiasDados(d *DadosKitMarca) {
	if d.Pilares == nil {
		d.Pilares = []string{}
	}
	if d.Cores == nil {
		d.Cores = []string{}
	}
	if d.Produtos == nil {
		d.Produtos = []ProdutoKit{}
	}
	if d.Canais == nil {
		d.Canais = []string{}
	}
}

func (a *API) verKit(w http.ResponseWriter, r *http.Request, s contextoSessao) {
	org, ok := a.autorizarAlcada(w, r, s, "ver_marca_posts")
	if !ok {
		return
	}
	resp, err := carregarKitCom(r.Context(), a.banco, org)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	responder(w, 200, resp)
}

func carregarRegistroKit(ctx context.Context, q consultaKitSQL, org string) (KitMarca, bool, error) {
	k := kitVazio(org)
	var em time.Time
	var produtosJSON []byte
	err := q.QueryRow(ctx, `
		SELECT site, perfis, tom_de_voz, publico, temas, nunca_dizer, fontes, exemplos,
		       pilares, cores, produtos, diferencial, modelo_negocio, modelo_outro,
		       regioes, como_compram, maior_desafio, objetivo_90_dias, canais, canais_outro,
		       restricoes_setor, atualizado_em
		FROM kits_marca WHERE organizacao_id = $1::uuid`, org).
		Scan(&k.Site, &k.Perfis, &k.TomDeVoz, &k.Publico, &k.Temas, &k.NuncaDizer, &k.Fontes, &k.Exemplos,
			&k.Pilares, &k.Cores, &produtosJSON, &k.Diferencial, &k.ModeloNegocio, &k.ModeloOutro,
			&k.Regioes, &k.ComoCompram, &k.MaiorDesafio, &k.Objetivo90Dias, &k.Canais, &k.CanaisOutro,
			&k.RestricoesSetor, &em)
	if errors.Is(err, pgx.ErrNoRows) {
		return k, false, nil
	}
	if err != nil {
		return KitMarca{}, false, err
	}
	if len(produtosJSON) > 0 {
		if err := json.Unmarshal(produtosJSON, &k.Produtos); err != nil {
			return KitMarca{}, false, err
		}
	}
	normalizarFatiasDados(&k.DadosKitMarca)
	k.AtualizadoEm = &em
	return k, true, nil
}

func carregarOrigensVersaoDB(ctx context.Context, q consultaKitSQL, org string) (map[string]origemVersaoDB, error) {
	linhas, err := q.Query(ctx, `
		SELECT campo, origem, definido_por::text, definido_em
		FROM kit_origens
		WHERE organizacao_id = $1::uuid`, org)
	if err != nil {
		return nil, err
	}
	defer linhas.Close()
	origens := map[string]origemVersaoDB{}
	for linhas.Next() {
		var campo string
		var o origemVersaoDB
		if err := linhas.Scan(&campo, &o.Origem, &o.DefinidoPorID, &o.DefinidoEm); err != nil {
			return nil, err
		}
		origens[campo] = o
	}
	if err := linhas.Err(); err != nil {
		return nil, err
	}
	return origens, nil
}

func carregarOrigensCom(ctx context.Context, q consultaKitSQL, org string) (map[string]OrigemCampo, error) {
	linhas, err := q.Query(ctx, `
		SELECT ko.campo, ko.origem, u.nome, ko.definido_em
		FROM kit_origens ko
		LEFT JOIN usuarios u ON u.id = ko.definido_por
		WHERE ko.organizacao_id = $1::uuid`, org)
	if err != nil {
		return nil, err
	}
	defer linhas.Close()
	origens := map[string]OrigemCampo{}
	for linhas.Next() {
		var campo string
		var o OrigemCampo
		if err := linhas.Scan(&campo, &o.Origem, &o.DefinidoPor, &o.DefinidoEm); err != nil {
			return nil, err
		}
		origens[campo] = o
	}
	if err := linhas.Err(); err != nil {
		return nil, err
	}
	return origens, nil
}

func versaoAtualKit(ctx context.Context, q consultaKitSQL, org string) (int, error) {
	var v int
	err := q.QueryRow(ctx, `SELECT COALESCE(MAX(numero), 0) FROM kit_versoes WHERE organizacao_id = $1::uuid`, org).Scan(&v)
	return v, err
}

func carregarKitCom(ctx context.Context, q consultaKitSQL, org string) (RespostaKit, error) {
	k, _, err := carregarRegistroKit(ctx, q, org)
	if err != nil {
		return RespostaKit{}, err
	}
	linhas, err := q.Query(ctx, `
		SELECT id::text, tipo, nome, mime, tamanho, criado_em FROM arquivos_marca
		WHERE organizacao_id = $1::uuid ORDER BY tipo, criado_em`, org)
	if err != nil {
		return RespostaKit{}, err
	}
	arqs, err := pgx.CollectRows(linhas, func(l pgx.CollectableRow) (ArquivoMarca, error) {
		var f ArquivoMarca
		err := l.Scan(&f.ID, &f.Tipo, &f.Nome, &f.Mime, &f.Tamanho, &f.CriadoEm)
		return f, err
	})
	if err != nil {
		return RespostaKit{}, err
	}
	if arqs == nil {
		arqs = []ArquivoMarca{}
	}
	origens, err := carregarOrigensCom(ctx, q, org)
	if err != nil {
		return RespostaKit{}, err
	}
	versao, err := versaoAtualKit(ctx, q, org)
	if err != nil {
		return RespostaKit{}, err
	}
	return RespostaKit{Kit: k, Arquivos: arqs, Origens: origens, Versao: versao}, nil
}

var corHex = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// validarKit limpa os textos e devolve a primeira regra quebrada, em linguagem simples.
func validarKit(k *KitMarca) string {
	campos := []struct {
		nome string
		v    *string
	}{
		{"Site", &k.Site},
		{"Perfis", &k.Perfis},
		{"Tom de voz", &k.TomDeVoz},
		{"Público", &k.Publico},
		{"Temas", &k.Temas},
		{"O que nunca dizer", &k.NuncaDizer},
		{"Fontes", &k.Fontes},
		{"Exemplos de posts", &k.Exemplos},
		{"Diferencial", &k.Diferencial},
		{"Outro modelo de negócio", &k.ModeloOutro},
		{"Regiões e idiomas", &k.Regioes},
		{"Como os clientes compram", &k.ComoCompram},
		{"Maior desafio", &k.MaiorDesafio},
		{"Objetivo em 90 dias", &k.Objetivo90Dias},
		{"Outros canais", &k.CanaisOutro},
		{"Restrições do setor", &k.RestricoesSetor},
	}
	for _, c := range campos {
		*c.v = strings.TrimSpace(*c.v)
		if utf8.RuneCountInString(*c.v) > maxTextoKit {
			return fmt.Sprintf("%s: use no máximo %d caracteres.", c.nome, maxTextoKit)
		}
	}
	if k.Site != "" {
		u, err := url.Parse(k.Site)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return "Site: informe o endereço completo, começando com https://."
		}
	}
	var pilares []string
	vistos := map[string]bool{}
	for _, p := range k.Pilares {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if utf8.RuneCountInString(p) > maxTextoPilar {
			return fmt.Sprintf("Cada pilar pode ter no máximo %d caracteres.", maxTextoPilar)
		}
		if vistos[strings.ToLower(p)] {
			return "Há pilares repetidos."
		}
		vistos[strings.ToLower(p)] = true
		pilares = append(pilares, p)
	}
	if len(pilares) > maxPilares {
		return fmt.Sprintf("Use no máximo %d pilares de conteúdo.", maxPilares)
	}
	k.Pilares = pilares

	if len(k.Cores) > maxCores {
		return fmt.Sprintf("Use no máximo %d cores.", maxCores)
	}
	for i, c := range k.Cores {
		if !corHex.MatchString(c) {
			return "Cor inválida: use o formato #RRGGBB."
		}
		k.Cores[i] = strings.ToLower(c)
	}

	var produtos []ProdutoKit
	for _, pr := range k.Produtos {
		nome := strings.TrimSpace(pr.Nome)
		if nome == "" {
			continue
		}
		if utf8.RuneCountInString(nome) > maxNomeProduto {
			return fmt.Sprintf("Cada produto pode ter no máximo %d caracteres.", maxNomeProduto)
		}
		produtos = append(produtos, ProdutoKit{Nome: nome, Prioritario: pr.Prioritario})
	}
	if len(produtos) > maxProdutos {
		return fmt.Sprintf("Use no máximo %d produtos ou serviços.", maxProdutos)
	}
	k.Produtos = produtos

	k.ModeloNegocio = strings.TrimSpace(k.ModeloNegocio)
	if !modelosNegocioValidos[k.ModeloNegocio] {
		return "Modelo de negócio inválido."
	}
	if k.ModeloNegocio != "outro" {
		k.ModeloOutro = ""
	}

	var canais []string
	canaisVistos := map[string]bool{}
	for _, ch := range k.Canais {
		ch = strings.TrimSpace(ch)
		if !canaisKitValidos[ch] {
			return "Canal inválido."
		}
		if canaisVistos[ch] {
			continue
		}
		canaisVistos[ch] = true
		canais = append(canais, ch)
	}
	k.Canais = canais

	normalizarFatiasDados(&k.DadosKitMarca)
	return ""
}

func detectarCamposAlterados(ant, novo KitMarca) []string {
	var alterados []string
	if novo.Site != ant.Site {
		alterados = append(alterados, "site")
	}
	if novo.Perfis != ant.Perfis {
		alterados = append(alterados, "perfis")
	}
	if novo.TomDeVoz != ant.TomDeVoz {
		alterados = append(alterados, "tomDeVoz")
	}
	if novo.Publico != ant.Publico {
		alterados = append(alterados, "publico")
	}
	if novo.Temas != ant.Temas {
		alterados = append(alterados, "temas")
	}
	if novo.NuncaDizer != ant.NuncaDizer {
		alterados = append(alterados, "nuncaDizer")
	}
	if novo.Fontes != ant.Fontes {
		alterados = append(alterados, "fontes")
	}
	if novo.Exemplos != ant.Exemplos {
		alterados = append(alterados, "exemplos")
	}
	if !slices.Equal(novo.Pilares, ant.Pilares) {
		alterados = append(alterados, "pilares")
	}
	if !slices.Equal(novo.Cores, ant.Cores) {
		alterados = append(alterados, "cores")
	}
	if !slices.Equal(novo.Produtos, ant.Produtos) {
		alterados = append(alterados, "produtos")
	}
	if novo.Diferencial != ant.Diferencial {
		alterados = append(alterados, "diferencial")
	}
	if novo.ModeloNegocio != ant.ModeloNegocio || novo.ModeloOutro != ant.ModeloOutro {
		alterados = append(alterados, "modeloNegocio")
	}
	if novo.Regioes != ant.Regioes {
		alterados = append(alterados, "regioes")
	}
	if novo.ComoCompram != ant.ComoCompram {
		alterados = append(alterados, "comoCompram")
	}
	if novo.MaiorDesafio != ant.MaiorDesafio {
		alterados = append(alterados, "maiorDesafio")
	}
	if novo.Objetivo90Dias != ant.Objetivo90Dias {
		alterados = append(alterados, "objetivo90Dias")
	}
	if !slices.Equal(novo.Canais, ant.Canais) || novo.CanaisOutro != ant.CanaisOutro {
		alterados = append(alterados, "canais")
	}
	if novo.RestricoesSetor != ant.RestricoesSetor {
		alterados = append(alterados, "restricoesSetor")
	}
	if alterados == nil {
		return []string{}
	}
	return alterados
}

func travarOrganizacaoKit(ctx context.Context, tx pgx.Tx, org string) error {
	var ok bool
	err := tx.QueryRow(ctx, `SELECT true FROM organizacoes WHERE id = $1::uuid FOR UPDATE`, org).Scan(&ok)
	return err
}

func gravarRegistroKit(ctx context.Context, q consultaKitSQL, org string, k *KitMarca, autorID *string, em time.Time) error {
	produtosJSON, err := json.Marshal(k.Produtos)
	if err != nil {
		return err
	}
	_, err = q.Exec(ctx, `
		INSERT INTO kits_marca (
			organizacao_id, site, perfis, tom_de_voz, publico, temas, nunca_dizer, fontes, exemplos,
			pilares, cores, produtos, diferencial, modelo_negocio, modelo_outro, regioes, como_compram,
			maior_desafio, objetivo_90_dias, canais, canais_outro, restricoes_setor, atualizado_em, atualizado_por
		)
		VALUES (
			$1::uuid, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12::jsonb, $13, $14, $15, $16, $17,
			$18, $19, $20, $21, $22, $23, $24::uuid
		)
		ON CONFLICT (organizacao_id) DO UPDATE SET
			site = EXCLUDED.site,
			perfis = EXCLUDED.perfis,
			tom_de_voz = EXCLUDED.tom_de_voz,
			publico = EXCLUDED.publico,
			temas = EXCLUDED.temas,
			nunca_dizer = EXCLUDED.nunca_dizer,
			fontes = EXCLUDED.fontes,
			exemplos = EXCLUDED.exemplos,
			pilares = EXCLUDED.pilares,
			cores = EXCLUDED.cores,
			produtos = EXCLUDED.produtos,
			diferencial = EXCLUDED.diferencial,
			modelo_negocio = EXCLUDED.modelo_negocio,
			modelo_outro = EXCLUDED.modelo_outro,
			regioes = EXCLUDED.regioes,
			como_compram = EXCLUDED.como_compram,
			maior_desafio = EXCLUDED.maior_desafio,
			objetivo_90_dias = EXCLUDED.objetivo_90_dias,
			canais = EXCLUDED.canais,
			canais_outro = EXCLUDED.canais_outro,
			restricoes_setor = EXCLUDED.restricoes_setor,
			atualizado_em = EXCLUDED.atualizado_em,
			atualizado_por = EXCLUDED.atualizado_por`,
		org, k.Site, k.Perfis, k.TomDeVoz, k.Publico, k.Temas, k.NuncaDizer, k.Fontes, k.Exemplos,
		k.Pilares, k.Cores, string(produtosJSON), k.Diferencial, k.ModeloNegocio, k.ModeloOutro,
		k.Regioes, k.ComoCompram, k.MaiorDesafio, k.Objetivo90Dias, k.Canais, k.CanaisOutro,
		k.RestricoesSetor, em, autorID)
	if err != nil {
		return err
	}
	k.AtualizadoEm = &em
	return nil
}

func upsertOrigensKit(ctx context.Context, q consultaKitSQL, org string, campos []string, origem string, definidoPor *string, em time.Time) error {
	for _, campo := range campos {
		if _, err := q.Exec(ctx, `
			INSERT INTO kit_origens (organizacao_id, campo, origem, definido_por, definido_em)
			VALUES ($1::uuid, $2, $3, $4::uuid, $5)
			ON CONFLICT (organizacao_id, campo) DO UPDATE SET
				origem = EXCLUDED.origem,
				definido_por = EXCLUDED.definido_por,
				definido_em = EXCLUDED.definido_em`,
			org, campo, origem, definidoPor, em); err != nil {
			return err
		}
	}
	return nil
}

func inserirVersaoKit(ctx context.Context, q consultaKitSQL, org string, numero int, dados DadosKitMarca, origens map[string]origemVersaoDB, camposAlterados []string, autorID *string, autorTipo string, em time.Time) error {
	normalizarFatiasDados(&dados)
	if origens == nil {
		origens = map[string]origemVersaoDB{}
	}
	if camposAlterados == nil {
		camposAlterados = []string{}
	}
	dadosJSON, err := json.Marshal(dados)
	if err != nil {
		return err
	}
	origensJSON, err := json.Marshal(origens)
	if err != nil {
		return err
	}
	_, err = q.Exec(ctx, `
		INSERT INTO kit_versoes (organizacao_id, numero, dados, origens, campos_alterados, autor_id, autor_tipo, criado_em)
		VALUES ($1::uuid, $2, $3::jsonb, $4::jsonb, $5, $6::uuid, $7, $8)`,
		org, numero, string(dadosJSON), string(origensJSON), camposAlterados, autorID, autorTipo, em)
	return err
}

func validarOrganizacaoPedido(w http.ResponseWriter, pedidoOrg, orgAtiva string) bool {
	if pedidoOrg != orgAtiva {
		erro(w, 409, "organizacao_trocada", "A organização ativa mudou. Recarregue a página antes de salvar.")
		return false
	}
	return true
}

func (a *API) salvarKit(w http.ResponseWriter, r *http.Request, s contextoSessao) {
	org, ok := a.autorizarAlcada(w, r, s, "cmo_dna")
	if !ok {
		return
	}
	var p pedidoSalvarKit
	if !lerJSON(w, r, &p) {
		return
	}
	if !validarOrganizacaoPedido(w, p.OrganizacaoID, org) {
		return
	}
	if p.VersaoBase < 0 {
		erro(w, 400, "dados_invalidos", "Versão base inválida.")
		return
	}
	if msg := validarKit(&p.KitMarca); msg != "" {
		erro(w, 400, "dados_invalidos", msg)
		return
	}

	tx, err := a.banco.Begin(r.Context())
	if err != nil {
		falhaInterna(w, err)
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck

	if err := travarOrganizacaoKit(r.Context(), tx, org); err != nil {
		falhaInterna(w, err)
		return
	}
	versaoAtual, err := versaoAtualKit(r.Context(), tx, org)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if p.VersaoBase != versaoAtual {
		erro(w, 409, "kit_desatualizado", msgKitDesatualizado)
		return
	}
	anterior, _, err := carregarRegistroKit(r.Context(), tx, org)
	if err != nil {
		falhaInterna(w, err)
		return
	}

	agora := a.agora().UTC()
	autorID := s.usuario.ID
	camposAlterados := detectarCamposAlterados(anterior, p.KitMarca)

	if err := gravarRegistroKit(r.Context(), tx, org, &p.KitMarca, &autorID, agora); err != nil {
		falhaInterna(w, err)
		return
	}
	if err := upsertOrigensKit(r.Context(), tx, org, camposAlterados, "confirmado", &autorID, agora); err != nil {
		falhaInterna(w, err)
		return
	}
	origensDB, err := carregarOrigensVersaoDB(r.Context(), tx, org)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if err := inserirVersaoKit(r.Context(), tx, org, versaoAtual+1, p.KitMarca.DadosKitMarca, origensDB, camposAlterados, &autorID, "pessoa", agora); err != nil {
		falhaInterna(w, err)
		return
	}
	resp, err := carregarKitCom(r.Context(), tx, org)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		falhaInterna(w, err)
		return
	}
	responder(w, 200, resp)
}

func (a *API) confirmarCamposKit(w http.ResponseWriter, r *http.Request, s contextoSessao) {
	org, ok := a.autorizarAlcada(w, r, s, "cmo_dna")
	if !ok {
		return
	}
	var p PedidoConfirmarKit
	if !lerJSON(w, r, &p) {
		return
	}
	if !validarOrganizacaoPedido(w, p.OrganizacaoID, org) {
		return
	}
	if p.VersaoBase < 0 {
		erro(w, 400, "dados_invalidos", "Versão base inválida.")
		return
	}
	if len(p.Campos) == 0 {
		erro(w, 400, "dados_invalidos", "Informe ao menos um campo para confirmar.")
		return
	}
	var campos []string
	vistos := map[string]bool{}
	for _, c := range p.Campos {
		if !camposKitValidos[c] {
			erro(w, 400, "dados_invalidos", "Campo do kit inválido.")
			return
		}
		if vistos[c] {
			continue
		}
		vistos[c] = true
		campos = append(campos, c)
	}

	tx, err := a.banco.Begin(r.Context())
	if err != nil {
		falhaInterna(w, err)
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck

	if err := travarOrganizacaoKit(r.Context(), tx, org); err != nil {
		falhaInterna(w, err)
		return
	}
	versaoAtual, err := versaoAtualKit(r.Context(), tx, org)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if p.VersaoBase != versaoAtual {
		erro(w, 409, "kit_desatualizado", msgKitDesatualizado)
		return
	}
	origensAtuais, err := carregarOrigensVersaoDB(r.Context(), tx, org)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	var camposConfirmar []string
	for _, c := range campos {
		if origensAtuais[c].Origem == "confirmado" {
			continue
		}
		camposConfirmar = append(camposConfirmar, c)
	}
	if len(camposConfirmar) == 0 {
		resp, err := carregarKitCom(r.Context(), tx, org)
		if err != nil {
			falhaInterna(w, err)
			return
		}
		responder(w, 200, resp)
		return
	}

	k, _, err := carregarRegistroKit(r.Context(), tx, org)
	if err != nil {
		falhaInterna(w, err)
		return
	}

	agora := a.agora().UTC()
	autorID := s.usuario.ID
	if err := gravarRegistroKit(r.Context(), tx, org, &k, &autorID, agora); err != nil {
		falhaInterna(w, err)
		return
	}
	if err := upsertOrigensKit(r.Context(), tx, org, camposConfirmar, "confirmado", &autorID, agora); err != nil {
		falhaInterna(w, err)
		return
	}
	origensDB, err := carregarOrigensVersaoDB(r.Context(), tx, org)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if err := inserirVersaoKit(r.Context(), tx, org, versaoAtual+1, k.DadosKitMarca, origensDB, camposConfirmar, &autorID, "pessoa", agora); err != nil {
		falhaInterna(w, err)
		return
	}
	resp, err := carregarKitCom(r.Context(), tx, org)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		falhaInterna(w, err)
		return
	}
	responder(w, 200, resp)
}

func clonarKit(k KitMarca) KitMarca {
	out := k
	out.Pilares = slices.Clone(k.Pilares)
	out.Cores = slices.Clone(k.Cores)
	out.Produtos = slices.Clone(k.Produtos)
	out.Canais = slices.Clone(k.Canais)
	normalizarFatiasDados(&out.DadosKitMarca)
	return out
}

func aplicarSugestaoNoKit(k *KitMarca, chave string, val any) (string, error) {
	switch chave {
	case "site", "perfis", "tomDeVoz", "publico", "temas", "nuncaDizer", "fontes", "exemplos",
		"diferencial", "modeloNegocio", "modeloOutro", "regioes", "comoCompram",
		"maiorDesafio", "objetivo90Dias", "canaisOutro", "restricoesSetor":
		s, ok := val.(string)
		if !ok {
			return "", fmt.Errorf("campo %s exige texto", chave)
		}
		atribuirTextoKit(k, chave, s)
		if chave == "modeloOutro" {
			return "modeloNegocio", nil
		}
		if chave == "canaisOutro" {
			return "canais", nil
		}
		return chave, nil
	case "pilares":
		lista, err := converterListaTexto(val, chave)
		if err != nil {
			return "", err
		}
		k.Pilares = lista
		return chave, nil
	case "cores":
		lista, err := converterListaTexto(val, chave)
		if err != nil {
			return "", err
		}
		k.Cores = lista
		return chave, nil
	case "canais":
		lista, err := converterListaTexto(val, chave)
		if err != nil {
			return "", err
		}
		k.Canais = lista
		return chave, nil
	case "produtos":
		lista, err := converterProdutos(val)
		if err != nil {
			return "", err
		}
		k.Produtos = lista
		return chave, nil
	default:
		return "", fmt.Errorf("campo inválido: %s", chave)
	}
}

// gravarSugestoesIA grava sugestões da IA apenas pelo servidor (sem rota HTTP pública),
// ignorando campos já confirmados ou cujo valor não mudou, e criando uma versão com autor_tipo 'ia'
// apenas se algum campo for efetivamente aplicado.
func (a *API) gravarSugestoesIA(ctx context.Context, org string, campos map[string]any) (ResultadoSugestoesIA, error) {
	if !uuidValido(org) {
		return ResultadoSugestoesIA{}, errors.New("organização inválida")
	}
	if len(campos) == 0 {
		return ResultadoSugestoesIA{}, errors.New("informe ao menos um campo sugerido")
	}

	tx, err := a.banco.Begin(ctx)
	if err != nil {
		return ResultadoSugestoesIA{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := travarOrganizacaoKit(ctx, tx, org); err != nil {
		return ResultadoSugestoesIA{}, err
	}
	versaoAtual, err := versaoAtualKit(ctx, tx, org)
	if err != nil {
		return ResultadoSugestoesIA{}, err
	}
	anterior, _, err := carregarRegistroKit(ctx, tx, org)
	if err != nil {
		return ResultadoSugestoesIA{}, err
	}
	origensAtuais, err := carregarOrigensVersaoDB(ctx, tx, org)
	if err != nil {
		return ResultadoSugestoesIA{}, err
	}

	// Valida todas as entradas antes de decidir o que aplicar ou ignorar.
	validacao := clonarKit(anterior)
	tocados := map[string]bool{}
	for chave, val := range campos {
		campoOrigem, err := aplicarSugestaoNoKit(&validacao, chave, val)
		if err != nil {
			return ResultadoSugestoesIA{}, err
		}
		tocados[campoOrigem] = true
	}
	if msg := validarKit(&validacao); msg != "" {
		return ResultadoSugestoesIA{}, errors.New(msg)
	}

	novo := clonarKit(anterior)
	confirmadosIgnorados := map[string]bool{}
	for chave, val := range campos {
		campoOrigem, _ := aplicarSugestaoNoKit(&novo, chave, val)
		if origensAtuais[campoOrigem].Origem == "confirmado" {
			confirmadosIgnorados[campoOrigem] = true
		}
	}
	// Restaura campos já confirmados para garantir que nunca sejam sobrescritos.
	if confirmadosIgnorados["modeloNegocio"] {
		novo.ModeloNegocio = anterior.ModeloNegocio
		novo.ModeloOutro = anterior.ModeloOutro
	}
	if confirmadosIgnorados["canais"] {
		novo.Canais = slices.Clone(anterior.Canais)
		novo.CanaisOutro = anterior.CanaisOutro
	}
	for c := range confirmadosIgnorados {
		switch c {
		case "site":
			novo.Site = anterior.Site
		case "perfis":
			novo.Perfis = anterior.Perfis
		case "tomDeVoz":
			novo.TomDeVoz = anterior.TomDeVoz
		case "publico":
			novo.Publico = anterior.Publico
		case "temas":
			novo.Temas = anterior.Temas
		case "nuncaDizer":
			novo.NuncaDizer = anterior.NuncaDizer
		case "fontes":
			novo.Fontes = anterior.Fontes
		case "exemplos":
			novo.Exemplos = anterior.Exemplos
		case "pilares":
			novo.Pilares = slices.Clone(anterior.Pilares)
		case "cores":
			novo.Cores = slices.Clone(anterior.Cores)
		case "produtos":
			novo.Produtos = slices.Clone(anterior.Produtos)
		case "diferencial":
			novo.Diferencial = anterior.Diferencial
		case "regioes":
			novo.Regioes = anterior.Regioes
		case "comoCompram":
			novo.ComoCompram = anterior.ComoCompram
		case "maiorDesafio":
			novo.MaiorDesafio = anterior.MaiorDesafio
		case "objetivo90Dias":
			novo.Objetivo90Dias = anterior.Objetivo90Dias
		case "restricoesSetor":
			novo.RestricoesSetor = anterior.RestricoesSetor
		}
	}
	if msg := validarKit(&novo); msg != "" {
		return ResultadoSugestoesIA{}, errors.New(msg)
	}

	alteradosSet := map[string]bool{}
	for _, c := range detectarCamposAlterados(anterior, novo) {
		alteradosSet[c] = true
	}

	aplicados := []string{}
	ignorados := []string{}
	for _, c := range camposKitOrdenados {
		if !tocados[c] {
			continue
		}
		if confirmadosIgnorados[c] || !alteradosSet[c] {
			ignorados = append(ignorados, c)
			continue
		}
		aplicados = append(aplicados, c)
	}

	if len(aplicados) == 0 {
		return ResultadoSugestoesIA{Aplicados: []string{}, Ignorados: ignorados}, nil
	}

	agora := a.agora().UTC()
	if err := gravarRegistroKit(ctx, tx, org, &novo, nil, agora); err != nil {
		return ResultadoSugestoesIA{}, err
	}
	if err := upsertOrigensKit(ctx, tx, org, aplicados, "sugerido_ia", nil, agora); err != nil {
		return ResultadoSugestoesIA{}, err
	}
	origensDB, err := carregarOrigensVersaoDB(ctx, tx, org)
	if err != nil {
		return ResultadoSugestoesIA{}, err
	}
	if err := inserirVersaoKit(ctx, tx, org, versaoAtual+1, novo.DadosKitMarca, origensDB, aplicados, nil, "ia", agora); err != nil {
		return ResultadoSugestoesIA{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ResultadoSugestoesIA{}, err
	}
	return ResultadoSugestoesIA{Aplicados: aplicados, Ignorados: ignorados}, nil
}

func atribuirTextoKit(k *KitMarca, campo, valor string) {
	switch campo {
	case "site":
		k.Site = valor
	case "perfis":
		k.Perfis = valor
	case "tomDeVoz":
		k.TomDeVoz = valor
	case "publico":
		k.Publico = valor
	case "temas":
		k.Temas = valor
	case "nuncaDizer":
		k.NuncaDizer = valor
	case "fontes":
		k.Fontes = valor
	case "exemplos":
		k.Exemplos = valor
	case "diferencial":
		k.Diferencial = valor
	case "modeloNegocio":
		k.ModeloNegocio = valor
	case "modeloOutro":
		k.ModeloOutro = valor
	case "regioes":
		k.Regioes = valor
	case "comoCompram":
		k.ComoCompram = valor
	case "maiorDesafio":
		k.MaiorDesafio = valor
	case "objetivo90Dias":
		k.Objetivo90Dias = valor
	case "canaisOutro":
		k.CanaisOutro = valor
	case "restricoesSetor":
		k.RestricoesSetor = valor
	}
}

func converterListaTexto(val any, campo string) ([]string, error) {
	switch v := val.(type) {
	case []string:
		out := make([]string, len(v))
		copy(out, v)
		return out, nil
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("campo %s exige lista de textos", campo)
			}
			out = append(out, s)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("campo %s exige lista de textos", campo)
	}
}

func converterProdutos(val any) ([]ProdutoKit, error) {
	switch v := val.(type) {
	case []ProdutoKit:
		out := make([]ProdutoKit, len(v))
		copy(out, v)
		return out, nil
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return nil, errors.New("produtos inválidos")
		}
		var out []ProdutoKit
		if err := json.Unmarshal(b, &out); err != nil {
			return nil, errors.New("produtos inválidos")
		}
		return out, nil
	}
}

func (a *API) listarVersoesKit(w http.ResponseWriter, r *http.Request, s contextoSessao) {
	org, ok := a.autorizarAlcada(w, r, s, "ver_marca_posts")
	if !ok {
		return
	}
	pagina := 1
	if pStr := strings.TrimSpace(r.URL.Query().Get("pagina")); pStr != "" {
		n, err := strconv.Atoi(pStr)
		if err != nil || n < 1 || n > maxPaginaVersoes {
			erro(w, 400, "dados_invalidos", "Página inválida.")
			return
		}
		pagina = n
	}

	var total int
	if err := a.banco.QueryRow(r.Context(), `
		SELECT count(*) FROM kit_versoes WHERE organizacao_id = $1::uuid`, org).Scan(&total); err != nil {
		falhaInterna(w, err)
		return
	}

	offset := (pagina - 1) * versoesPorPagina
	linhas, err := a.banco.Query(r.Context(), `
		SELECT kv.numero, kv.autor_tipo, u.nome, kv.campos_alterados, kv.criado_em
		FROM kit_versoes kv
		LEFT JOIN usuarios u ON u.id = kv.autor_id
		WHERE kv.organizacao_id = $1::uuid
		ORDER BY kv.numero DESC
		LIMIT $2 OFFSET $3`, org, versoesPorPagina, offset)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	versoes, err := pgx.CollectRows(linhas, func(l pgx.CollectableRow) (ResumoVersaoKit, error) {
		var v ResumoVersaoKit
		if err := l.Scan(&v.Numero, &v.AutorTipo, &v.AutorNome, &v.CamposAlterados, &v.CriadoEm); err != nil {
			return ResumoVersaoKit{}, err
		}
		if v.CamposAlterados == nil {
			v.CamposAlterados = []string{}
		}
		return v, nil
	})
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if versoes == nil {
		versoes = []ResumoVersaoKit{}
	}
	responder(w, 200, RespostaVersoesKit{
		Versoes: versoes,
		Pagina:  pagina,
		Total:   total,
	})
}

func (a *API) verVersaoKit(w http.ResponseWriter, r *http.Request, s contextoSessao) {
	org, ok := a.autorizarAlcada(w, r, s, "ver_marca_posts")
	if !ok {
		return
	}
	n, err := strconv.ParseInt(r.PathValue("numero"), 10, 32)
	if err != nil || n < 1 {
		erro(w, 404, "nao_encontrado", "Versão não encontrada.")
		return
	}
	numero := int(n)

	var v VersaoKit
	var dadosJSON, origensJSON []byte
	err = a.banco.QueryRow(r.Context(), `
		SELECT kv.numero, kv.autor_tipo, u.nome, kv.campos_alterados, kv.criado_em, kv.dados, kv.origens
		FROM kit_versoes kv
		LEFT JOIN usuarios u ON u.id = kv.autor_id
		WHERE kv.organizacao_id = $1::uuid AND kv.numero = $2`, org, numero).
		Scan(&v.Numero, &v.AutorTipo, &v.AutorNome, &v.CamposAlterados, &v.CriadoEm, &dadosJSON, &origensJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		erro(w, 404, "nao_encontrado", "Versão não encontrada.")
		return
	}
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if err := json.Unmarshal(dadosJSON, &v.Dados); err != nil {
		falhaInterna(w, err)
		return
	}
	normalizarFatiasDados(&v.Dados)
	var origensDB map[string]origemVersaoDB
	if err := json.Unmarshal(origensJSON, &origensDB); err != nil {
		falhaInterna(w, err)
		return
	}
	ids := make([]string, 0, len(origensDB))
	vistosID := map[string]bool{}
	for _, o := range origensDB {
		if o.DefinidoPorID != nil && uuidValido(*o.DefinidoPorID) && !vistosID[*o.DefinidoPorID] {
			vistosID[*o.DefinidoPorID] = true
			ids = append(ids, *o.DefinidoPorID)
		}
	}
	nomesPorID := map[string]string{}
	if len(ids) > 0 {
		linhasNomes, err := a.banco.Query(r.Context(), `SELECT id::text, nome FROM usuarios WHERE id = ANY($1::uuid[])`, ids)
		if err != nil {
			falhaInterna(w, err)
			return
		}
		defer linhasNomes.Close()
		for linhasNomes.Next() {
			var id, nome string
			if err := linhasNomes.Scan(&id, &nome); err != nil {
				falhaInterna(w, err)
				return
			}
			nomesPorID[id] = nome
		}
		if err := linhasNomes.Err(); err != nil {
			falhaInterna(w, err)
			return
		}
	}
	v.Origens = make(map[string]OrigemCampo, len(origensDB))
	for campo, o := range origensDB {
		var definidoPor *string
		if o.DefinidoPorID != nil {
			if nome, ok := nomesPorID[*o.DefinidoPorID]; ok {
				n := nome
				definidoPor = &n
			}
		}
		v.Origens[campo] = OrigemCampo{
			Origem:      o.Origem,
			DefinidoPor: definidoPor,
			DefinidoEm:  o.DefinidoEm,
		}
	}
	if v.CamposAlterados == nil {
		v.CamposAlterados = []string{}
	}
	responder(w, 200, v)
}

// Tipos aceitos, conferidos pelo conteúdo do arquivo (não pelo nome nem pelo que o navegador diz).
var mimesAceitos = map[string]map[string]bool{
	"logo":       {"image/png": true, "image/jpeg": true, "image/webp": true},
	"referencia": {"image/png": true, "image/jpeg": true, "image/webp": true, "application/pdf": true},
}

func campoAlteradoArquivo(tipo, nome string) string {
	if tipo == "logo" {
		return "logo:" + nome
	}
	return "arquivos:" + nome
}

func (a *API) registrarVersaoArquivo(ctx context.Context, tx pgx.Tx, org, autorID, tipo, nome string, em time.Time) error {
	versaoAtual, err := versaoAtualKit(ctx, tx, org)
	if err != nil {
		return err
	}
	k, _, err := carregarRegistroKit(ctx, tx, org)
	if err != nil {
		return err
	}
	if err := gravarRegistroKit(ctx, tx, org, &k, &autorID, em); err != nil {
		return err
	}
	origensDB, err := carregarOrigensVersaoDB(ctx, tx, org)
	if err != nil {
		return err
	}
	return inserirVersaoKit(ctx, tx, org, versaoAtual+1, k.DadosKitMarca, origensDB, []string{campoAlteradoArquivo(tipo, nome)}, &autorID, "pessoa", em)
}

func (a *API) enviarArquivo(w http.ResponseWriter, r *http.Request, s contextoSessao) {
	org, ok := a.autorizarAlcada(w, r, s, "cmo_dna")
	if !ok {
		return
	}
	tipo := r.URL.Query().Get("tipo")
	if mimesAceitos[tipo] == nil {
		erro(w, 400, "tipo_invalido", "Tipo de arquivo inválido.")
		return
	}
	if r.URL.Query().Get("organizacaoId") != org {
		erro(w, 409, "organizacao_trocada", "A organização ativa mudou. Recarregue a página antes de salvar.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxTamanhoArquivo+64<<10)
	arquivo, cab, err := r.FormFile("arquivo")
	if err != nil {
		var grande *http.MaxBytesError
		if errors.As(err, &grande) {
			erro(w, 413, "arquivo_grande", "O arquivo pode ter no máximo 8 MB.")
			return
		}
		erro(w, 400, "sem_arquivo", "Escolha um arquivo para enviar.")
		return
	}
	defer arquivo.Close()
	conteudo, err := io.ReadAll(io.LimitReader(arquivo, maxTamanhoArquivo+1))
	if err != nil {
		erro(w, 400, "sem_arquivo", "Não foi possível ler o arquivo.")
		return
	}
	if len(conteudo) == 0 {
		erro(w, 400, "sem_arquivo", "O arquivo está vazio.")
		return
	}
	if len(conteudo) > maxTamanhoArquivo {
		erro(w, 413, "arquivo_grande", "O arquivo pode ter no máximo 8 MB.")
		return
	}
	mime := http.DetectContentType(conteudo)
	if !mimesAceitos[tipo][mime] {
		if tipo == "logo" {
			erro(w, 415, "formato_nao_aceito", "Logo: envie PNG, JPG ou WebP.")
		} else {
			erro(w, 415, "formato_nao_aceito", "Envie PNG, JPG, WebP ou PDF.")
		}
		return
	}
	nome := limparNomeArquivo(cab.Filename)

	tx, err := a.banco.Begin(r.Context())
	if err != nil {
		falhaInterna(w, err)
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck

	if err := travarOrganizacaoKit(r.Context(), tx, org); err != nil {
		falhaInterna(w, err)
		return
	}
	if tipo == "logo" {
		// O logo novo substitui o anterior.
		if _, err := tx.Exec(r.Context(), `DELETE FROM arquivos_marca WHERE organizacao_id = $1::uuid AND tipo = 'logo'`, org); err != nil {
			falhaInterna(w, err)
			return
		}
	} else {
		var n int
		if err := tx.QueryRow(r.Context(), `SELECT count(*) FROM arquivos_marca WHERE organizacao_id = $1::uuid AND tipo = 'referencia'`, org).Scan(&n); err != nil {
			falhaInterna(w, err)
			return
		}
		if n >= maxReferencias {
			erro(w, 409, "limite_arquivos", fmt.Sprintf("Limite de %d arquivos de referência. Remova algum antes de enviar outro.", maxReferencias))
			return
		}
	}
	var f ArquivoMarca
	err = tx.QueryRow(r.Context(), `
		INSERT INTO arquivos_marca (organizacao_id, tipo, nome, mime, tamanho, conteudo, criado_por)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id::text, tipo, nome, mime, tamanho, criado_em`,
		org, tipo, nome, mime, len(conteudo), conteudo, s.usuario.ID).
		Scan(&f.ID, &f.Tipo, &f.Nome, &f.Mime, &f.Tamanho, &f.CriadoEm)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			erro(w, 409, "logo_em_envio", "Outro logo foi enviado ao mesmo tempo. Tente de novo.")
			return
		}
		falhaInterna(w, err)
		return
	}
	if err := a.registrarVersaoArquivo(r.Context(), tx, org, s.usuario.ID, tipo, nome, a.agora().UTC()); err != nil {
		falhaInterna(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		falhaInterna(w, err)
		return
	}
	responder(w, 201, f)
}

func (a *API) baixarArquivo(w http.ResponseWriter, r *http.Request, s contextoSessao) {
	org, ok := a.autorizarAlcada(w, r, s, "ver_marca_posts")
	if !ok {
		return
	}
	var nome, mime string
	var conteudo []byte
	err := a.banco.QueryRow(r.Context(), `
		SELECT nome, mime, conteudo FROM arquivos_marca WHERE id::text = $1 AND organizacao_id::text = $2`,
		r.PathValue("id"), org).Scan(&nome, &mime, &conteudo)
	if errors.Is(err, pgx.ErrNoRows) {
		erro(w, 404, "nao_encontrado", "Arquivo não encontrado.")
		return
	}
	if err != nil {
		falhaInterna(w, err)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Length", strconv.Itoa(len(conteudo)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'; img-src 'self'")
	w.Header().Set("Cache-Control", "private, no-store")
	disposicao := "inline"
	if r.URL.Query().Get("baixar") == "1" {
		disposicao = "attachment"
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`%s; filename*=UTF-8''%s`, disposicao, url.PathEscape(nome)))
	_, _ = w.Write(conteudo)
}

func (a *API) removerArquivo(w http.ResponseWriter, r *http.Request, s contextoSessao) {
	org, ok := a.autorizarAlcada(w, r, s, "cmo_dna")
	if !ok {
		return
	}
	if r.URL.Query().Get("organizacaoId") != org {
		erro(w, 409, "organizacao_trocada", "A organização ativa mudou. Recarregue a página antes de salvar.")
		return
	}
	tx, err := a.banco.Begin(r.Context())
	if err != nil {
		falhaInterna(w, err)
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck

	if err := travarOrganizacaoKit(r.Context(), tx, org); err != nil {
		falhaInterna(w, err)
		return
	}
	var tipo, nome string
	err = tx.QueryRow(r.Context(), `
		DELETE FROM arquivos_marca WHERE id::text = $1 AND organizacao_id = $2::uuid
		RETURNING tipo, nome`, r.PathValue("id"), org).Scan(&tipo, &nome)
	if errors.Is(err, pgx.ErrNoRows) {
		erro(w, 404, "nao_encontrado", "Arquivo não encontrado.")
		return
	}
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if err := a.registrarVersaoArquivo(r.Context(), tx, org, s.usuario.ID, tipo, nome, a.agora().UTC()); err != nil {
		falhaInterna(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		falhaInterna(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func limparNomeArquivo(nome string) string {
	nome = strings.TrimSpace(nome)
	if i := strings.LastIndexAny(nome, `/\`); i >= 0 {
		nome = nome[i+1:]
	}
	nome = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 || r == '"' {
			return -1
		}
		return r
	}, nome)
	if nome == "" {
		nome = "arquivo"
	}
	if utf8.RuneCountInString(nome) > 120 {
		nome = string([]rune(nome)[:120])
	}
	return nome
}
