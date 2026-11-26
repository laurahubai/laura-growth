package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

const (
	validadeConvite          = 7 * 24 * time.Hour
	limiteConvitesPendentes  = 30
	limiteEnviosPorHora      = 20
	tamanhoTokenConviteBytes = 32
	maxNomeConviteRunas      = 200
	maxNomeOrgEmailRunas     = 120
)

// ---- Formato das respostas (espelha packages/contratos/src/acessos.ts) ----

type RespostaConviteCriado struct {
	Convite       ConvitePendente `json:"convite"`
	Link          string          `json:"link"`
	MensagemEnvio *string         `json:"mensagemEnvio"`
}

type ConvitePublico struct {
	OrganizacaoNome string `json:"organizacaoNome"`
	EmailMascarado  string `json:"emailMascarado"`
	ContaExistente  bool   `json:"contaExistente"`
}

func (a *API) rotasConvites(m *http.ServeMux) {
	m.HandleFunc("POST /api/convites", a.comSessao(a.criarConvite))
	m.HandleFunc("POST /api/convites/{id}/reenviar", a.comSessao(a.reenviarConvite))
	m.HandleFunc("DELETE /api/convites/{id}", a.comSessao(a.revogarConvite))
	m.HandleFunc("GET /api/convites/{token}", a.verConvitePublico)
	m.HandleFunc("POST /api/convites/{token}/aceitar", a.aceitarConvite)
}

func normalizarEValidarEmail(bruto string) (string, bool) {
	email := strings.ToLower(strings.TrimSpace(bruto))
	if len(email) < 3 || len(email) > 254 {
		return "", false
	}
	for _, r := range email {
		if r < 0x20 || r == 0x7f || r == ' ' || r == '\t' || r == '<' || r == '>' || r == ',' || r == ';' || r == '"' {
			return "", false
		}
	}
	if strings.Count(email, "@") != 1 {
		return "", false
	}
	partes := strings.SplitN(email, "@", 2)
	local, dominio := partes[0], partes[1]
	if local == "" || len(local) > 64 || dominio == "" {
		return "", false
	}
	if !strings.Contains(dominio, ".") || strings.HasPrefix(dominio, ".") || strings.HasSuffix(dominio, ".") ||
		strings.HasPrefix(dominio, "-") || strings.HasSuffix(dominio, "-") || strings.Contains(dominio, "..") {
		return "", false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", false
	}
	return email, true
}

func gerarTokenConvite() (tokenStr string, hash []byte, err error) {
	raw := make([]byte, tamanhoTokenConviteBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	sum := sha256.Sum256(raw)
	return base64.RawURLEncoding.EncodeToString(raw), sum[:], nil
}

func hashTokenConvite(tokenStr string) ([]byte, bool) {
	if tokenStr == "" {
		return nil, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(tokenStr)
	if err != nil || len(raw) < tamanhoTokenConviteBytes {
		return nil, false
	}
	sum := sha256.Sum256(raw)
	return sum[:], true
}

func mascararEmail(email string) string {
	partes := strings.SplitN(strings.ToLower(strings.TrimSpace(email)), "@", 2)
	if len(partes) != 2 {
		return "***"
	}
	runas := []rune(partes[0])
	if len(runas) == 0 {
		return "***@" + partes[1]
	}
	prefixo := string(runas[:1])
	if len(runas) > 3 {
		prefixo = string(runas[:2])
	}
	return prefixo + "***@" + partes[1]
}

func limparLinhaTexto(s string, maxRunas int) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			b.WriteByte(' ')
			continue
		}
		b.WriteRune(r)
	}
	limpo := strings.TrimSpace(b.String())
	if maxRunas > 0 {
		runas := []rune(limpo)
		if len(runas) > maxRunas {
			limpo = strings.TrimSpace(string(runas[:maxRunas]))
		}
	}
	return limpo
}

func (a *API) montarLinkConvite(token string) (link string, configurado bool) {
	base := strings.TrimRight(strings.TrimSpace(a.enderecoPublico), "/")
	if base == "" {
		return "/convite/" + token, false
	}
	return base + "/convite/" + token, true
}

func (a *API) emailJaMembroNaOrg(ctx context.Context, q executorSQL, org, email string) (bool, error) {
	var jaMembro bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM membros m
			JOIN usuarios u ON u.id = m.usuario_id
			WHERE m.organizacao_id = $1::uuid AND lower(u.email) = lower($2)
		)`, org, email).Scan(&jaMembro)
	return jaMembro, err
}

func (a *API) verificarLimiteEnviosHora(ctx context.Context, q executorSQL, org string, agora time.Time) (bool, error) {
	var envios int
	err := q.QueryRow(ctx, `
		SELECT count(*)
		FROM registro_acessos
		WHERE organizacao_id = $1::uuid
		  AND acao IN ('convite_criado', 'convite_reenviado')
		  AND em > $2`,
		org, agora.Add(-1*time.Hour)).Scan(&envios)
	if err != nil {
		return false, err
	}
	return envios < limiteEnviosPorHora, nil
}

func (a *API) enviarEmailConvite(ctx context.Context, orgID, conviteID, para, orgNome, link string, linkConfigurado bool) (estado string, mensagem *string) {
	if !linkConfigurado {
		msg := "O endereço público da plataforma (LG_ENDERECO_PUBLICO) não está configurado para envio por e-mail. Copie o link abaixo para enviar manualmente."
		return "falhou", &msg
	}

	nomeOrgLimpo := limparLinhaTexto(orgNome, maxNomeOrgEmailRunas)
	assunto := limparLinhaTexto(fmt.Sprintf("Convite para %s na Laura Growth", nomeOrgLimpo), 0)
	texto := fmt.Sprintf(
		"Olá,\n\nVocê recebeu um convite para acessar a organização %s na Plataforma Laura Growth.\n\n"+
			"Para aceitar o convite, acesse o link abaixo (válido por 7 dias e de uso único):\n%s\n\n"+
			"Se você não esperava este convite, pode ignorar este e-mail.",
		nomeOrgLimpo, link,
	)

	var err error
	if a.email == nil {
		err = errEmailNaoConfigurado
	} else {
		err = a.email.enviar(ctx, para, assunto, texto)
	}
	if err == nil {
		return "enviado", nil
	}

	slog.Warn("falha ao enviar e-mail de convite",
		"organizacaoId", orgID, "conviteId", conviteID, "motivo", err.Error())

	var msg string
	if errors.Is(err, errEmailNaoConfigurado) {
		msg = "O envio automático de e-mail não está configurado. Copie o link abaixo para enviar o convite manualmente."
	} else {
		msg = "Não foi possível enviar o e-mail de convite agora. O convite foi salvo; copie o link abaixo ou clique em Reenviar mais tarde."
	}
	return "falhou", &msg
}

func (a *API) atualizarEstadoEnvioConvite(conviteID string, tokenHash []byte, estado string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := a.banco.Exec(ctx, `
		UPDATE convites
		SET envio_estado = $1
		WHERE id = $2::uuid
		  AND token_hash = $3
		  AND aceito_em IS NULL
		  AND revogado_em IS NULL`,
		estado, conviteID, tokenHash); err != nil {
		slog.Warn("falha ao atualizar estado de envio do convite", "conviteId", conviteID, "erro", err)
		return "pendente"
	}
	return estado
}

func revogarConvitesPendentesDoCriador(ctx context.Context, tx pgx.Tx, orgID, atorID, criadorID, motivo string, agora time.Time) error {
	linhas, err := tx.Query(ctx, `
		UPDATE convites
		SET revogado_em = $3
		WHERE organizacao_id = $1::uuid
		  AND criado_por = $2::uuid
		  AND aceito_em IS NULL
		  AND revogado_em IS NULL
		  AND expira_em > $3
		RETURNING id::text, email`,
		orgID, criadorID, agora)
	if err != nil {
		return err
	}
	type conviteRevogado struct {
		id    string
		email string
	}
	revogados, err := pgx.CollectRows(linhas, func(l pgx.CollectableRow) (conviteRevogado, error) {
		var r conviteRevogado
		err := l.Scan(&r.id, &r.email)
		return r, err
	})
	if err != nil {
		return err
	}
	for _, item := range revogados {
		if err := registrarAcessoEm(ctx, tx, &orgID, &atorID, nil, &item.email, "convite_revogado", map[string]any{
			"conviteId": item.id,
			"motivo":    motivo,
		}, agora); err != nil {
			return err
		}
	}
	return nil
}

func (a *API) criarConvite(w http.ResponseWriter, r *http.Request, s contextoSessao) {
	org, ok := a.autorizarAlcada(w, r, s, "gerenciar_pessoas")
	if !ok {
		return
	}

	var pedido struct {
		Email   string   `json:"email"`
		Alcadas []string `json:"alcadas"`
	}
	if !lerJSON(w, r, &pedido) {
		return
	}
	email, emailValido := normalizarEValidarEmail(pedido.Email)
	if !emailValido || pedido.Alcadas == nil {
		erro(w, 400, "dados_invalidos", "Informe um e-mail válido e a lista de alçadas.")
		return
	}
	alcadas, alcadasValidasOk := validarAlcadas(pedido.Alcadas)
	if !alcadasValidasOk {
		erro(w, 400, "dados_invalidos", "Há uma alçada inválida na lista.")
		return
	}

	agora := a.agora()
	tx, ok := a.iniciarEscritaPessoas(w, r, s, org)
	if !ok {
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck

	jaMembro, err := a.emailJaMembroNaOrg(r.Context(), tx, org, email)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if jaMembro {
		erro(w, 409, "ja_membro", "Esta pessoa já faz parte da organização.")
		return
	}

	dentroLimiteHora, err := a.verificarLimiteEnviosHora(r.Context(), tx, org, agora)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if !dentroLimiteHora {
		erro(w, 429, "limite_envios_hora", "Limite de 20 envios de convite por hora atingido. Aguarde antes de enviar outro.")
		return
	}

	var pendentesOutros int
	if err := tx.QueryRow(r.Context(), `
		SELECT count(*)
		FROM convites
		WHERE organizacao_id = $1::uuid
		  AND aceito_em IS NULL
		  AND revogado_em IS NULL
		  AND expira_em > $2
		  AND lower(email) <> $3`,
		org, agora, email).Scan(&pendentesOutros); err != nil {
		falhaInterna(w, err)
		return
	}
	if pendentesOutros >= limiteConvitesPendentes {
		erro(w, 429, "limite_convites_pendentes", "Limite de 30 convites pendentes nesta organização atingido. Revogue um convite pendente para enviar outro.")
		return
	}

	linhasSubstituidos, err := tx.Query(r.Context(), `
		UPDATE convites
		SET revogado_em = $3
		WHERE organizacao_id = $1::uuid
		  AND lower(email) = $2
		  AND aceito_em IS NULL
		  AND revogado_em IS NULL
		  AND expira_em > $3
		RETURNING id::text`,
		org, email, agora)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	substituidos, err := pgx.CollectRows(linhasSubstituidos, func(l pgx.CollectableRow) (string, error) {
		var idAntigo string
		err := l.Scan(&idAntigo)
		return idAntigo, err
	})
	if err != nil {
		falhaInterna(w, err)
		return
	}
	for _, idAntigo := range substituidos {
		if err := registrarAcessoEm(r.Context(), tx, &org, &s.usuario.ID, nil, &email, "convite_revogado", map[string]any{
			"conviteId": idAntigo,
			"motivo":    "substituido",
		}, agora); err != nil {
			falhaInterna(w, err)
			return
		}
	}

	token, tokenHash, err := gerarTokenConvite()
	if err != nil {
		falhaInterna(w, err)
		return
	}
	expiraEm := agora.Add(validadeConvite)

	var conviteID, orgNome string
	if err := tx.QueryRow(r.Context(), `
		INSERT INTO convites (organizacao_id, email, alcadas, token_hash, criado_por, criado_em, expira_em, envio_estado)
		VALUES ($1::uuid, $2, $3, $4, $5::uuid, $6, $7, 'pendente')
		RETURNING id::text, (SELECT nome FROM organizacoes WHERE id = $1::uuid)`,
		org, email, alcadas, tokenHash, s.usuario.ID, agora, expiraEm).
		Scan(&conviteID, &orgNome); err != nil {
		falhaInterna(w, err)
		return
	}

	if err := registrarAcessoEm(r.Context(), tx, &org, &s.usuario.ID, nil, &email, "convite_criado", map[string]any{
		"conviteId": conviteID,
		"alcadas":   alcadas,
	}, agora); err != nil {
		falhaInterna(w, err)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		falhaInterna(w, err)
		return
	}

	link, linkConfigurado := a.montarLinkConvite(token)
	estado, msgEnvio := a.enviarEmailConvite(r.Context(), org, conviteID, email, orgNome, link, linkConfigurado)
	estado = a.atualizarEstadoEnvioConvite(conviteID, tokenHash, estado)

	criadoPorNome := s.usuario.Nome
	responder(w, 201, RespostaConviteCriado{
		Convite: ConvitePendente{
			ID:            conviteID,
			Email:         email,
			Alcadas:       alcadas,
			EnvioEstado:   estado,
			CriadoPorNome: &criadoPorNome,
			CriadoEm:      agora,
			ExpiraEm:      expiraEm,
		},
		Link:          link,
		MensagemEnvio: msgEnvio,
	})
}

func (a *API) reenviarConvite(w http.ResponseWriter, r *http.Request, s contextoSessao) {
	org, ok := a.autorizarAlcada(w, r, s, "gerenciar_pessoas")
	if !ok {
		return
	}
	conviteID := r.PathValue("id")
	if !uuidValido(conviteID) {
		erro(w, 404, "nao_encontrado", "Convite não encontrado.")
		return
	}

	agora := a.agora()
	tx, ok := a.iniciarEscritaPessoas(w, r, s, org)
	if !ok {
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck

	var email, orgNome string
	var alcadas []string
	var criadoPorNome *string
	var criadoEm time.Time
	err := tx.QueryRow(r.Context(), `
		SELECT c.email, c.alcadas, u.nome, c.criado_em, o.nome
		FROM convites c
		JOIN organizacoes o ON o.id = c.organizacao_id
		LEFT JOIN usuarios u ON u.id = c.criado_por
		WHERE c.id = $1::uuid
		  AND c.organizacao_id = $2::uuid
		  AND c.aceito_em IS NULL
		  AND c.revogado_em IS NULL
		FOR UPDATE OF c`,
		conviteID, org).
		Scan(&email, &alcadas, &criadoPorNome, &criadoEm, &orgNome)
	if errors.Is(err, pgx.ErrNoRows) {
		erro(w, 404, "nao_encontrado", "Convite não encontrado.")
		return
	}
	if err != nil {
		falhaInterna(w, err)
		return
	}

	jaMembro, err := a.emailJaMembroNaOrg(r.Context(), tx, org, email)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if jaMembro {
		erro(w, 409, "ja_membro", "Esta pessoa já faz parte da organização.")
		return
	}

	dentroLimiteHora, err := a.verificarLimiteEnviosHora(r.Context(), tx, org, agora)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	if !dentroLimiteHora {
		erro(w, 429, "limite_envios_hora", "Limite de 20 envios de convite por hora atingido. Aguarde antes de reenviar.")
		return
	}

	token, tokenHash, err := gerarTokenConvite()
	if err != nil {
		falhaInterna(w, err)
		return
	}
	expiraEm := agora.Add(validadeConvite)
	if _, err := tx.Exec(r.Context(), `
		UPDATE convites
		SET token_hash = $1, expira_em = $2, envio_estado = 'pendente'
		WHERE id = $3::uuid`,
		tokenHash, expiraEm, conviteID); err != nil {
		falhaInterna(w, err)
		return
	}

	if err := registrarAcessoEm(r.Context(), tx, &org, &s.usuario.ID, nil, &email, "convite_reenviado", map[string]any{
		"conviteId": conviteID,
	}, agora); err != nil {
		falhaInterna(w, err)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		falhaInterna(w, err)
		return
	}

	link, linkConfigurado := a.montarLinkConvite(token)
	estado, msgEnvio := a.enviarEmailConvite(r.Context(), org, conviteID, email, orgNome, link, linkConfigurado)
	estado = a.atualizarEstadoEnvioConvite(conviteID, tokenHash, estado)

	responder(w, 200, RespostaConviteCriado{
		Convite: ConvitePendente{
			ID:            conviteID,
			Email:         email,
			Alcadas:       ordenarAlcadas(alcadas),
			EnvioEstado:   estado,
			CriadoPorNome: criadoPorNome,
			CriadoEm:      criadoEm,
			ExpiraEm:      expiraEm,
		},
		Link:          link,
		MensagemEnvio: msgEnvio,
	})
}

func (a *API) revogarConvite(w http.ResponseWriter, r *http.Request, s contextoSessao) {
	org, ok := a.autorizarAlcada(w, r, s, "gerenciar_pessoas")
	if !ok {
		return
	}
	conviteID := r.PathValue("id")
	if !uuidValido(conviteID) {
		erro(w, 404, "nao_encontrado", "Convite não encontrado.")
		return
	}

	agora := a.agora()
	tx, ok := a.iniciarEscritaPessoas(w, r, s, org)
	if !ok {
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck

	var email string
	err := tx.QueryRow(r.Context(), `
		SELECT email
		FROM convites
		WHERE id = $1::uuid
		  AND organizacao_id = $2::uuid
		  AND aceito_em IS NULL
		  AND revogado_em IS NULL
		FOR UPDATE`,
		conviteID, org).Scan(&email)
	if errors.Is(err, pgx.ErrNoRows) {
		erro(w, 404, "nao_encontrado", "Convite não encontrado.")
		return
	}
	if err != nil {
		falhaInterna(w, err)
		return
	}

	if _, err := tx.Exec(r.Context(), `
		UPDATE convites SET revogado_em = $1 WHERE id = $2::uuid`,
		agora, conviteID); err != nil {
		falhaInterna(w, err)
		return
	}

	if err := registrarAcessoEm(r.Context(), tx, &org, &s.usuario.ID, nil, &email, "convite_revogado", map[string]any{
		"conviteId": conviteID,
	}, agora); err != nil {
		falhaInterna(w, err)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		falhaInterna(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) verConvitePublico(w http.ResponseWriter, r *http.Request) {
	tokenHash, ok := hashTokenConvite(r.PathValue("token"))
	if !ok {
		erro(w, 404, "convite_invalido", "Este convite não é válido ou já expirou.")
		return
	}

	var orgNome, email string
	var contaExistente bool
	err := a.banco.QueryRow(r.Context(), `
		SELECT o.nome, c.email,
		       EXISTS (SELECT 1 FROM usuarios u WHERE lower(u.email) = lower(c.email))
		FROM convites c
		JOIN organizacoes o ON o.id = c.organizacao_id
		JOIN membros cm ON cm.organizacao_id = c.organizacao_id AND cm.usuario_id = c.criado_por
		WHERE c.token_hash = $1
		  AND c.aceito_em IS NULL
		  AND c.revogado_em IS NULL
		  AND c.expira_em > $2
		  AND (
		    cm.papel = 'dono'
		    OR EXISTS (
		      SELECT 1 FROM alcadas_membro cam
		      WHERE cam.organizacao_id = cm.organizacao_id
		        AND cam.usuario_id = cm.usuario_id
		        AND cam.alcada = 'gerenciar_pessoas'
		    )
		  )`,
		tokenHash, a.agora()).
		Scan(&orgNome, &email, &contaExistente)
	if errors.Is(err, pgx.ErrNoRows) {
		erro(w, 404, "convite_invalido", "Este convite não é válido ou já expirou.")
		return
	}
	if err != nil {
		falhaInterna(w, err)
		return
	}

	responder(w, 200, ConvitePublico{
		OrganizacaoNome: orgNome,
		EmailMascarado:  mascararEmail(email),
		ContaExistente:  contaExistente,
	})
}

type pedidoAceitarConvite struct {
	Nome  *string `json:"nome"`
	Senha *string `json:"senha"`
}

func (a *API) aceitarConvite(w http.ResponseWriter, r *http.Request) {
	tokenHash, ok := hashTokenConvite(r.PathValue("token"))
	if !ok {
		erro(w, 404, "convite_invalido", "Este convite não é válido ou já expirou.")
		return
	}

	var pedido pedidoAceitarConvite
	if !lerJSON(w, r, &pedido) {
		return
	}

	sessaoAtual, _, sessaoValida, err := a.lerSessaoDeCookie(r)
	if err != nil {
		falhaInterna(w, err)
		return
	}

	agora := a.agora()
	tx, err := a.banco.Begin(r.Context())
	if err != nil {
		falhaInterna(w, err)
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck

	var conviteID, orgID, email string
	var alcadas []string
	var criadoPor *string
	err = tx.QueryRow(r.Context(), `
		SELECT c.id::text, c.organizacao_id::text, lower(c.email), c.alcadas, c.criado_por::text
		FROM convites c
		WHERE c.token_hash = $1
		  AND c.aceito_em IS NULL
		  AND c.revogado_em IS NULL
		  AND c.expira_em > $2
		FOR UPDATE OF c`,
		tokenHash, agora).
		Scan(&conviteID, &orgID, &email, &alcadas, &criadoPor)
	if errors.Is(err, pgx.ErrNoRows) {
		erro(w, 404, "convite_invalido", "Este convite não é válido ou já expirou.")
		return
	}
	if err != nil {
		falhaInterna(w, err)
		return
	}

	// S-1: o convite só pode ser aceito enquanto quem o criou continuar dono ou membro com gerenciar_pessoas.
	criadorAutorizado := false
	if criadoPor != nil {
		criadorAutorizado, err = exigirAlcadaCom(r.Context(), tx, orgID, *criadoPor, "gerenciar_pessoas", false)
		if err != nil {
			falhaInterna(w, err)
			return
		}
	}
	if !criadorAutorizado {
		erro(w, 404, "convite_invalido", "Este convite não é válido ou já expirou.")
		return
	}

	// Serializa criação/vinculação do mesmo e-mail para evitar corridas.
	if _, err := tx.Exec(r.Context(), `SELECT pg_advisory_xact_lock(hashtext('laura-growth:usuario:' || $1))`, email); err != nil {
		falhaInterna(w, err)
		return
	}

	var usuarioID string
	err = tx.QueryRow(r.Context(), `SELECT id::text FROM usuarios WHERE lower(email) = $1 FOR UPDATE`, email).Scan(&usuarioID)
	contaExistente := true
	if errors.Is(err, pgx.ErrNoRows) {
		contaExistente = false
	} else if err != nil {
		falhaInterna(w, err)
		return
	}

	var sessaoTokenBruto []byte
	var sessaoHash []byte

	if contaExistente {
		if !sessaoValida {
			erro(w, 401, "login_necessario", "Entre com sua conta antes de aceitar o convite.")
			return
		}
		if !strings.EqualFold(sessaoAtual.usuario.Email, email) || sessaoAtual.usuario.ID != usuarioID {
			erro(w, 403, "email_diferente", "Você está conectado com outro e-mail. Entre com o e-mail convidado para aceitar.")
			return
		}
		var jaMembro bool
		if err := tx.QueryRow(r.Context(), `
			SELECT EXISTS (SELECT 1 FROM membros WHERE organizacao_id = $1::uuid AND usuario_id = $2::uuid)`,
			orgID, usuarioID).Scan(&jaMembro); err != nil {
			falhaInterna(w, err)
			return
		}
		if jaMembro {
			if _, err := tx.Exec(r.Context(), `
				UPDATE convites SET revogado_em = $1 WHERE id = $2::uuid`,
				agora, conviteID); err != nil {
				falhaInterna(w, err)
				return
			}
			if err := registrarAcessoEm(r.Context(), tx, &orgID, &usuarioID, &usuarioID, &email, "convite_revogado", map[string]any{
				"conviteId": conviteID,
				"motivo":    "ja_membro",
			}, agora); err != nil {
				falhaInterna(w, err)
				return
			}
			if err := tx.Commit(r.Context()); err != nil {
				falhaInterna(w, err)
				return
			}
			erro(w, 409, "ja_membro", "Esta pessoa já faz parte da organização.")
			return
		}
		if _, err := tx.Exec(r.Context(), `
			UPDATE sessoes SET organizacao_id = $1::uuid WHERE token_hash = $2`,
			orgID, sessaoAtual.tokenHash); err != nil {
			falhaInterna(w, err)
			return
		}
		sessaoHash = sessaoAtual.tokenHash
	} else {
		var errInterno error
		var statusErro int
		var codErro, msgErro string
		usuarioID, sessaoTokenBruto, sessaoHash, statusErro, codErro, msgErro, errInterno = criarContaESessaoConvite(r.Context(), tx, orgID, email, pedido, agora)
		if errInterno != nil {
			falhaInterna(w, errInterno)
			return
		}
		if statusErro != 0 {
			erro(w, statusErro, codErro, msgErro)
			return
		}
	}

	if _, err := tx.Exec(r.Context(), `
		INSERT INTO membros (organizacao_id, usuario_id, papel)
		VALUES ($1::uuid, $2::uuid, 'membro')`,
		orgID, usuarioID); err != nil {
		falhaInterna(w, err)
		return
	}

	alcadasOrdenadas := ordenarAlcadas(alcadas)
	for _, al := range alcadasOrdenadas {
		if _, err := tx.Exec(r.Context(), `
			INSERT INTO alcadas_membro (organizacao_id, usuario_id, alcada, concedida_por, concedida_em)
			VALUES ($1::uuid, $2::uuid, $3, $4::uuid, $5)`,
			orgID, usuarioID, al, criadoPor, agora); err != nil {
			falhaInterna(w, err)
			return
		}
	}

	if _, err := tx.Exec(r.Context(), `
		UPDATE convites SET aceito_em = $1, aceito_por = $2::uuid WHERE id = $3::uuid`,
		agora, usuarioID, conviteID); err != nil {
		falhaInterna(w, err)
		return
	}

	if err := registrarAcessoEm(r.Context(), tx, &orgID, &usuarioID, &usuarioID, &email, "convite_aceito", map[string]any{
		"conviteId": conviteID,
		"alcadas":   alcadasOrdenadas,
		"contaNova": !contaExistente,
	}, agora); err != nil {
		falhaInterna(w, err)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		falhaInterna(w, err)
		return
	}

	if len(sessaoTokenBruto) > 0 {
		http.SetCookie(w, &http.Cookie{
			Name:     nomeCookie,
			Value:    base64.RawURLEncoding.EncodeToString(sessaoTokenBruto),
			Path:     "/",
			HttpOnly: true,
			Secure:   a.cookieSeguro,
			SameSite: http.SameSiteLaxMode,
			Expires:  agora.Add(duracaoSessao),
		})
	}

	sessao, err := a.montarSessao(r.Context(), sessaoHash)
	if err != nil {
		falhaInterna(w, err)
		return
	}
	responder(w, 200, sessao)
}

func criarContaESessaoConvite(ctx context.Context, tx pgx.Tx, orgID, email string, pedido pedidoAceitarConvite, agora time.Time) (usuarioID string, tokenBruto, sessaoHash []byte, statusErro int, codErro, msgErro string, err error) {
	nome := ""
	if pedido.Nome != nil {
		nome = strings.TrimSpace(*pedido.Nome)
	}
	if nome == "" || utf8.RuneCountInString(nome) > maxNomeConviteRunas {
		return "", nil, nil, 400, "dados_invalidos", "Informe seu nome para criar a conta.", nil
	}
	senha := ""
	if pedido.Senha != nil {
		senha = *pedido.Senha
	}
	if errSenha := validarSenhaNova(senha); errSenha != nil {
		return "", nil, nil, 400, "dados_invalidos", "A senha precisa ter entre 10 e 72 caracteres.", nil
	}
	senhaHash, err := gerarHashSenha(senha)
	if err != nil {
		return "", nil, nil, 0, "", "", err
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO usuarios (nome, email, senha_hash)
		VALUES ($1, $2, $3)
		RETURNING id::text`,
		nome, email, senhaHash).Scan(&usuarioID); err != nil {
		return "", nil, nil, 0, "", "", err
	}

	tokenBruto = make([]byte, 32)
	if _, err := rand.Read(tokenBruto); err != nil {
		return "", nil, nil, 0, "", "", err
	}
	sum := sha256.Sum256(tokenBruto)
	sessaoHash = sum[:]
	if _, err := tx.Exec(ctx, `
		INSERT INTO sessoes (token_hash, usuario_id, organizacao_id, expira_em)
		VALUES ($1, $2::uuid, $3::uuid, $4)`,
		sessaoHash, usuarioID, orgID, agora.Add(duracaoSessao)); err != nil {
		return "", nil, nil, 0, "", "", err
	}
	return usuarioID, tokenBruto, sessaoHash, 0, "", "", nil
}
