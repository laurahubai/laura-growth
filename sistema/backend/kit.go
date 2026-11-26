package main

// Etapa 2: kit de marca da organização ativa (MKT-01, MKT-03, MKT-04).
// Toda leitura e gravação usa a organização ativa da sessão, conferindo que a
// pessoa é membro dela. Pedidos de gravação trazem também a organização que a
// tela está mostrando: se a pessoa trocou de organização em outra aba, a gravação
// é recusada em vez de cair no kit errado.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
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
)

type KitMarca struct {
	OrganizacaoID string     `json:"organizacaoId"`
	Site          string     `json:"site"`
	Perfis        string     `json:"perfis"`
	TomDeVoz      string     `json:"tomDeVoz"`
	Publico       string     `json:"publico"`
	Temas         string     `json:"temas"`
	NuncaDizer    string     `json:"nuncaDizer"`
	Fontes        string     `json:"fontes"`
	Exemplos      string     `json:"exemplos"`
	Pilares       []string   `json:"pilares"`
	Cores         []string   `json:"cores"`
	AtualizadoEm  *time.Time `json:"atualizadoEm"`
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
	Kit      KitMarca       `json:"kit"`
	Arquivos []ArquivoMarca `json:"arquivos"`
}

func (a *API) rotasKit(m *http.ServeMux) {
	m.HandleFunc("GET /api/kit", a.comSessao(a.verKit))
	m.HandleFunc("PUT /api/kit", a.comSessao(a.salvarKit))
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

func (a *API) verKit(w http.ResponseWriter, r *http.Request, s contextoSessao) {
	org, ok := a.autorizarAlcada(w, r, s, "ver_marca_posts")
	if !ok {
		return
	}
	resp, err := a.carregarKit(r.Context(), org)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	responder(w, 200, resp)
}

func (a *API) carregarKit(ctx context.Context, org string) (RespostaKit, error) {
	k := KitMarca{OrganizacaoID: org, Pilares: []string{}, Cores: []string{}}
	var em time.Time
	err := a.banco.QueryRow(ctx, `
		SELECT site, perfis, tom_de_voz, publico, temas, nunca_dizer, fontes, exemplos, pilares, cores, atualizado_em
		FROM kits_marca WHERE organizacao_id::text = $1`, org).
		Scan(&k.Site, &k.Perfis, &k.TomDeVoz, &k.Publico, &k.Temas, &k.NuncaDizer, &k.Fontes, &k.Exemplos, &k.Pilares, &k.Cores, &em)
	if err == nil {
		k.AtualizadoEm = &em
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return RespostaKit{}, err
	}
	linhas, err := a.banco.Query(ctx, `
		SELECT id::text, tipo, nome, mime, tamanho, criado_em FROM arquivos_marca
		WHERE organizacao_id::text = $1 ORDER BY tipo, criado_em`, org)
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
	return RespostaKit{Kit: k, Arquivos: arqs}, nil
}

var corHex = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// validarKit limpa os textos e devolve a primeira regra quebrada, em linguagem simples.
func validarKit(k *KitMarca) string {
	campos := []struct {
		nome string
		v    *string
	}{
		{"Site", &k.Site}, {"Perfis", &k.Perfis}, {"Tom de voz", &k.TomDeVoz}, {"Público", &k.Publico},
		{"Temas", &k.Temas}, {"O que nunca dizer", &k.NuncaDizer}, {"Fontes", &k.Fontes}, {"Exemplos de posts", &k.Exemplos},
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
	if k.Pilares == nil {
		k.Pilares = []string{}
	}
	if len(k.Cores) > maxCores {
		return fmt.Sprintf("Use no máximo %d cores.", maxCores)
	}
	for i, c := range k.Cores {
		if !corHex.MatchString(c) {
			return "Cor inválida: use o formato #RRGGBB."
		}
		k.Cores[i] = strings.ToLower(c)
	}
	if k.Cores == nil {
		k.Cores = []string{}
	}
	return ""
}

func (a *API) salvarKit(w http.ResponseWriter, r *http.Request, s contextoSessao) {
	org, ok := a.autorizarAlcada(w, r, s, "cmo_dna")
	if !ok {
		return
	}
	var k KitMarca
	if !lerJSON(w, r, &k) {
		return
	}
	if k.OrganizacaoID != org {
		erro(w, 409, "organizacao_trocada", "A organização ativa mudou. Recarregue a página antes de salvar.")
		return
	}
	if msg := validarKit(&k); msg != "" {
		erro(w, 400, "kit_invalido", msg)
		return
	}
	_, err := a.banco.Exec(r.Context(), `
		INSERT INTO kits_marca (organizacao_id, site, perfis, tom_de_voz, publico, temas, nunca_dizer, fontes, exemplos, pilares, cores, atualizado_em, atualizado_por)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, now(), $12)
		ON CONFLICT (organizacao_id) DO UPDATE SET
		  site = EXCLUDED.site, perfis = EXCLUDED.perfis, tom_de_voz = EXCLUDED.tom_de_voz, publico = EXCLUDED.publico,
		  temas = EXCLUDED.temas, nunca_dizer = EXCLUDED.nunca_dizer, fontes = EXCLUDED.fontes, exemplos = EXCLUDED.exemplos,
		  pilares = EXCLUDED.pilares, cores = EXCLUDED.cores, atualizado_em = now(), atualizado_por = EXCLUDED.atualizado_por`,
		org, k.Site, k.Perfis, k.TomDeVoz, k.Publico, k.Temas, k.NuncaDizer, k.Fontes, k.Exemplos, k.Pilares, k.Cores, s.usuario.ID)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	resp, err := a.carregarKit(r.Context(), org)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	responder(w, 200, resp)
}

// Tipos aceitos, conferidos pelo conteúdo do arquivo (não pelo nome nem pelo que o navegador diz).
var mimesAceitos = map[string]map[string]bool{
	"logo":       {"image/png": true, "image/jpeg": true, "image/webp": true},
	"referencia": {"image/png": true, "image/jpeg": true, "image/webp": true, "application/pdf": true},
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
	if tipo == "logo" {
		// O logo novo substitui o anterior.
		if _, err := tx.Exec(r.Context(), `DELETE FROM arquivos_marca WHERE organizacao_id::text = $1 AND tipo = 'logo'`, org); err != nil {
			falhaInterna(w, err)
			return
		}
	} else {
		var n int
		if err := tx.QueryRow(r.Context(), `SELECT count(*) FROM arquivos_marca WHERE organizacao_id::text = $1 AND tipo = 'referencia'`, org).Scan(&n); err != nil {
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
	tag, err := a.banco.Exec(r.Context(), `DELETE FROM arquivos_marca WHERE id::text = $1 AND organizacao_id::text = $2`, r.PathValue("id"), org)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		erro(w, 404, "nao_encontrado", "Arquivo não encontrado.")
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
