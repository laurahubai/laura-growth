//go:build ignore

// Utilitário exclusivo para preparação e limpeza do banco nos testes E2E (Playwright).
// Exige LG_E2E_BANCO_URL apontando para um banco cujo nome termine com _teste ou _e2e.
// Uso:
//
//	go run ./e2e_apoio.go preparar
//	go run ./e2e_apoio.go limpar
//	go run ./e2e_apoio.go expirar-convite <email>
//	go run ./e2e_apoio.go semear-pendentes <orgNome> <quantidade>
//	go run ./e2e_apoio.go sugerir-kit <orgNome> <campoKit> <valorTexto>
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func validarBancoE2E(raw string) (string, error) {
	u := strings.TrimSpace(raw)
	if u == "" {
		return "", errors.New("defina LG_E2E_BANCO_URL com um banco terminado em _teste ou _e2e")
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return "", fmt.Errorf("LG_E2E_BANCO_URL inválida: %w", err)
	}
	nome := strings.TrimPrefix(parsed.Path, "/")
	if !strings.HasSuffix(nome, "_teste") && !strings.HasSuffix(nome, "_e2e") {
		return "", fmt.Errorf("recusado: o banco E2E (%q) precisa terminar com _teste ou _e2e", nome)
	}
	return u, nil
}

func upsertUsuarioE2E(ctx context.Context, tx pgx.Tx, nome, email, senhaHash string) (string, error) {
	var usuarioID string
	err := tx.QueryRow(ctx, `
		INSERT INTO usuarios (nome, email, senha_hash, admin_plataforma)
		VALUES ($1, $2, $3, false)
		ON CONFLICT ((lower(email))) DO UPDATE
			SET nome = EXCLUDED.nome,
			    senha_hash = EXCLUDED.senha_hash,
			    admin_plataforma = false
		RETURNING id::text`, nome, email, senhaHash).Scan(&usuarioID)
	return usuarioID, err
}

func upsertMembroE2E(ctx context.Context, tx pgx.Tx, orgID, nome, email, senhaHash, papel string, alcadas []string) error {
	usuarioID, err := upsertUsuarioE2E(ctx, tx, nome, email, senhaHash)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO membros (organizacao_id, usuario_id, papel)
		VALUES ($1::uuid, $2::uuid, $3)
		ON CONFLICT (organizacao_id, usuario_id) DO UPDATE SET papel = EXCLUDED.papel`, orgID, usuarioID, papel); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM alcadas_membro WHERE organizacao_id = $1::uuid AND usuario_id = $2::uuid`, orgID, usuarioID); err != nil {
		return err
	}
	for _, al := range alcadas {
		if _, err := tx.Exec(ctx, `
			INSERT INTO alcadas_membro (organizacao_id, usuario_id, alcada)
			VALUES ($1::uuid, $2::uuid, $3)
			ON CONFLICT DO NOTHING`, orgID, usuarioID, al); err != nil {
			return err
		}
	}
	return nil
}

func garantirOrganizacaoE2E(ctx context.Context, tx pgx.Tx, nomeOrg string) (string, error) {
	var orgID string
	err := tx.QueryRow(ctx, `SELECT id::text FROM organizacoes WHERE nome = $1`, nomeOrg).Scan(&orgID)
	if err == nil {
		return orgID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	if nomeOrg == "Laura Growth" || nomeOrg == "Laura Hub" {
		return "", errors.New("organização base não encontrada; execute semear antes")
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO organizacoes (nome, tipo, da_casa)
		VALUES ($1, 'empresa', false)
		RETURNING id::text`, nomeOrg).Scan(&orgID)
	return orgID, err
}

func hashSenhaEnv(varNome string) (string, error) {
	senha := os.Getenv(varNome)
	if len(senha) < 10 {
		return "", fmt.Errorf("defina %s com pelo menos 10 caracteres", varNome)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(senha), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func main() {
	bancoURL, err := validarBancoE2E(os.Getenv("LG_E2E_BANCO_URL"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	acao := "preparar"
	if len(os.Args) > 1 {
		acao = os.Args[1]
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, bancoURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro ao conectar no banco: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	switch acao {
	case "expirar-convite":
		if len(os.Args) < 3 || strings.TrimSpace(os.Args[2]) == "" {
			fmt.Fprintln(os.Stderr, "uso: expirar-convite <email>")
			os.Exit(2)
		}
		emailAlvo := strings.TrimSpace(os.Args[2])
		tag, err := pool.Exec(ctx, `
			UPDATE convites
			SET expira_em = now() - interval '1 hour'
			WHERE lower(email) = lower($1)
			  AND aceito_em IS NULL
			  AND revogado_em IS NULL`, emailAlvo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "erro ao expirar convite: %v\n", err)
			os.Exit(1)
		}
		if tag.RowsAffected() == 0 {
			fmt.Fprintf(os.Stderr, "nenhum convite pendente encontrado para %s\n", emailAlvo)
			os.Exit(1)
		}
		return

	case "semear-pendentes":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "uso: semear-pendentes <orgNome> <quantidade>")
			os.Exit(2)
		}
		orgNome := strings.TrimSpace(os.Args[2])
		qtd, err := strconv.Atoi(os.Args[3])
		if err != nil || qtd < 1 || qtd > 50 {
			fmt.Fprintln(os.Stderr, "quantidade inválida (1..50)")
			os.Exit(2)
		}
		if orgNome == "Laura Growth" || orgNome == "Laura Hub" {
			fmt.Fprintln(os.Stderr, "recusado: use organização temporária para semear pendentes")
			os.Exit(2)
		}
		var orgID string
		if err := pool.QueryRow(ctx, `SELECT id::text FROM organizacoes WHERE nome = $1`, orgNome).Scan(&orgID); err != nil {
			fmt.Fprintf(os.Stderr, "organização não encontrada: %v\n", err)
			os.Exit(1)
		}
		for i := 0; i < qtd; i++ {
			sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", orgID, i)))
			emailFicticio := fmt.Sprintf("pendente-e2e-%d@exemplo.test", i)
			if _, err := pool.Exec(ctx, `
				INSERT INTO convites (organizacao_id, email, alcadas, token_hash, expira_em, envio_estado)
				VALUES ($1::uuid, $2, ARRAY['ver_marca_posts']::text[], $3, now() + interval '7 days', 'falhou')`,
				orgID, emailFicticio, sum[:]); err != nil {
				fmt.Fprintf(os.Stderr, "erro ao inserir convite pendente E2E: %v\n", err)
				os.Exit(1)
			}
		}
		return

	case "limpar":
		varsEmails := []string{
			"LG_E2E_OPERACAO_EMAIL",
			"LG_E2E_SEM_ACESSO_EMAIL",
			"LG_E2E_EXISTENTE_EMAIL",
			"LG_E2E_REMOVER_EMAIL",
			"LG_E2E_GERENCIADOR_EMAIL",
			"LG_E2E_ORG_DONO_EMAIL",
		}
		for _, v := range varsEmails {
			if em := strings.TrimSpace(os.Getenv(v)); em != "" {
				if _, err := pool.Exec(ctx, `DELETE FROM usuarios WHERE lower(email) = lower($1)`, em); err != nil {
					fmt.Fprintf(os.Stderr, "erro ao limpar usuário (%s): %v\n", v, err)
					os.Exit(1)
				}
			}
		}
		if _, err := pool.Exec(ctx, `DELETE FROM convites WHERE lower(email) LIKE 'convidado-e2e-%@exemplo.test' OR lower(email) LIKE 'pendente-e2e-%@exemplo.test'`); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao limpar convites E2E: %v\n", err)
			os.Exit(1)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM usuarios WHERE lower(email) LIKE 'convidado-e2e-%@exemplo.test'`); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao limpar usuários convidados E2E: %v\n", err)
			os.Exit(1)
		}
		if orgNome := strings.TrimSpace(os.Getenv("LG_E2E_ORG_NOME")); orgNome != "" && orgNome != "Laura Growth" && orgNome != "Laura Hub" {
			if _, err := pool.Exec(ctx, `DELETE FROM organizacoes WHERE nome = $1`, orgNome); err != nil {
				fmt.Fprintf(os.Stderr, "erro ao limpar organização temporária E2E: %v\n", err)
				os.Exit(1)
			}
		}
		return

	case "sugerir-kit":
		if len(os.Args) < 5 {
			fmt.Fprintln(os.Stderr, "uso: sugerir-kit <orgNome> <campoKit> <valorTexto>")
			os.Exit(2)
		}
		orgNome := strings.TrimSpace(os.Args[2])
		campo := strings.TrimSpace(os.Args[3])
		valor := strings.TrimSpace(os.Args[4])
		if orgNome == "" || campo == "" {
			fmt.Fprintln(os.Stderr, "uso: sugerir-kit <orgNome> <campoKit> <valorTexto>")
			os.Exit(2)
		}
		if utf8.RuneCountInString(valor) > 5000 {
			fmt.Fprintln(os.Stderr, "valorTexto excede 5000 caracteres")
			os.Exit(2)
		}
		colunasTexto := map[string]struct{ coluna, origemCampo string }{
			"site":            {"site", "site"},
			"perfis":          {"perfis", "perfis"},
			"tomDeVoz":        {"tom_de_voz", "tomDeVoz"},
			"publico":         {"publico", "publico"},
			"temas":           {"temas", "temas"},
			"nuncaDizer":      {"nunca_dizer", "nuncaDizer"},
			"fontes":          {"fontes", "fontes"},
			"exemplos":        {"exemplos", "exemplos"},
			"diferencial":     {"diferencial", "diferencial"},
			"modeloNegocio":   {"modelo_negocio", "modeloNegocio"},
			"modeloOutro":     {"modelo_outro", "modeloNegocio"},
			"regioes":         {"regioes", "regioes"},
			"comoCompram":     {"como_compram", "comoCompram"},
			"maiorDesafio":    {"maior_desafio", "maiorDesafio"},
			"objetivo90Dias":  {"objetivo_90_dias", "objetivo90Dias"},
			"canaisOutro":     {"canais_outro", "canais"},
			"restricoesSetor": {"restricoes_setor", "restricoesSetor"},
		}
		alvo, ok := colunasTexto[campo]
		if !ok {
			fmt.Fprintf(os.Stderr, "campoKit de texto desconhecido: %s\n", campo)
			os.Exit(2)
		}
		if campo == "site" && valor != "" {
			u, err := url.Parse(valor)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
				fmt.Fprintln(os.Stderr, "site inválido: informe endereço http:// ou https://")
				os.Exit(2)
			}
		}
		if campo == "modeloNegocio" && valor != "" && valor != "b2b" && valor != "b2c" && valor != "b2b2c" && valor != "outro" {
			fmt.Fprintln(os.Stderr, "modeloNegocio inválido")
			os.Exit(2)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "erro ao iniciar transação: %v\n", err)
			os.Exit(1)
		}
		defer tx.Rollback(ctx) //nolint:errcheck

		var orgID string
		if err := tx.QueryRow(ctx, `SELECT id::text FROM organizacoes WHERE nome = $1 FOR UPDATE`, orgNome).Scan(&orgID); err != nil {
			fmt.Fprintf(os.Stderr, "organização não encontrada: %v\n", err)
			os.Exit(1)
		}
		agora := time.Now().UTC()
		queryKit := fmt.Sprintf(`
			INSERT INTO kits_marca (organizacao_id, %s, atualizado_em, atualizado_por)
			VALUES ($1::uuid, $2, $3, NULL)
			ON CONFLICT (organizacao_id) DO UPDATE SET
				%s = EXCLUDED.%s,
				atualizado_em = EXCLUDED.atualizado_em,
				atualizado_por = NULL`, alvo.coluna, alvo.coluna, alvo.coluna)
		if _, err := tx.Exec(ctx, queryKit, orgID, valor, agora); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao atualizar kit: %v\n", err)
			os.Exit(1)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO kit_origens (organizacao_id, campo, origem, definido_por, definido_em)
			VALUES ($1::uuid, $2, 'sugerido_ia', NULL, $3)
			ON CONFLICT (organizacao_id, campo) DO UPDATE SET
				origem = 'sugerido_ia',
				definido_por = NULL,
				definido_em = EXCLUDED.definido_em`, orgID, alvo.origemCampo, agora); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao gravar origem sugerido_ia: %v\n", err)
			os.Exit(1)
		}
		var dadosJSON, origensJSON []byte
		if err := tx.QueryRow(ctx, `
			SELECT
				jsonb_build_object(
					'site', k.site,
					'perfis', k.perfis,
					'tomDeVoz', k.tom_de_voz,
					'publico', k.publico,
					'temas', k.temas,
					'nuncaDizer', k.nunca_dizer,
					'fontes', k.fontes,
					'exemplos', k.exemplos,
					'pilares', to_jsonb(k.pilares),
					'cores', to_jsonb(k.cores),
					'produtos', k.produtos,
					'diferencial', k.diferencial,
					'modeloNegocio', k.modelo_negocio,
					'modeloOutro', k.modelo_outro,
					'regioes', k.regioes,
					'comoCompram', k.como_compram,
					'maiorDesafio', k.maior_desafio,
					'objetivo90Dias', k.objetivo_90_dias,
					'canais', to_jsonb(k.canais),
					'canaisOutro', k.canais_outro,
					'restricoesSetor', k.restricoes_setor,
					'atualizadoEm', to_jsonb(k.atualizado_em)
				),
				COALESCE(
					(
						SELECT jsonb_object_agg(
							ko.campo,
							jsonb_build_object(
								'origem', ko.origem,
								'definidoPor', u.nome,
								'definidoEm', to_jsonb(ko.definido_em)
							)
						)
						FROM kit_origens ko
						LEFT JOIN usuarios u ON u.id = ko.definido_por
						WHERE ko.organizacao_id = k.organizacao_id
					),
					'{}'::jsonb
				)
			FROM kits_marca k
			WHERE k.organizacao_id = $1::uuid`, orgID).Scan(&dadosJSON, &origensJSON); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao ler estado do kit: %v\n", err)
			os.Exit(1)
		}
		var proximoNumero int
		if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(numero), 0) + 1 FROM kit_versoes WHERE organizacao_id = $1::uuid`, orgID).Scan(&proximoNumero); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao calcular versão do kit: %v\n", err)
			os.Exit(1)
		}
		// Validação rápida de JSON antes de inserir
		if !json.Valid(dadosJSON) || !json.Valid(origensJSON) {
			fmt.Fprintln(os.Stderr, "json inválido ao montar versão")
			os.Exit(1)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO kit_versoes (organizacao_id, numero, dados, origens, campos_alterados, autor_id, autor_tipo, criado_em)
			VALUES ($1::uuid, $2, $3::jsonb, $4::jsonb, $5, NULL, 'ia', $6)`,
			orgID, proximoNumero, string(dadosJSON), string(origensJSON), []string{alvo.origemCampo}, agora); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao gravar versão da IA: %v\n", err)
			os.Exit(1)
		}
		if err := tx.Commit(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao confirmar transação: %v\n", err)
			os.Exit(1)
		}
		return

	case "preparar":
	default:
		fmt.Fprintf(os.Stderr, "ação desconhecida: %s\n", acao)
		os.Exit(2)
	}

	orgAlvoNome := strings.TrimSpace(os.Getenv("LG_E2E_ORG_NOME"))
	if orgAlvoNome == "" {
		orgAlvoNome = "Laura Growth"
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro ao iniciar transação: %v\n", err)
		os.Exit(1)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	orgID, err := garantirOrganizacaoE2E(ctx, tx, orgAlvoNome)
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro ao preparar organização: %v\n", err)
		os.Exit(1)
	}

	if donoOrgEmail := strings.TrimSpace(os.Getenv("LG_E2E_ORG_DONO_EMAIL")); donoOrgEmail != "" {
		hashDono, err := hashSenhaEnv("LG_E2E_ORG_DONO_SENHA")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if err := upsertMembroE2E(ctx, tx, orgID, "Dono Temporário E2E", donoOrgEmail, hashDono, "dono", nil); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao salvar dono da organização E2E: %v\n", err)
			os.Exit(1)
		}
	}

	if email := strings.TrimSpace(os.Getenv("LG_E2E_OPERACAO_EMAIL")); email != "" {
		hash, err := hashSenhaEnv("LG_E2E_OPERACAO_SENHA")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if err := upsertMembroE2E(ctx, tx, orgID, "Operação E2E", email, hash, "membro", []string{"ver_marca_posts"}); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao salvar usuário de teste: %v\n", err)
			os.Exit(1)
		}
	}

	if semAcessoEmail := strings.TrimSpace(os.Getenv("LG_E2E_SEM_ACESSO_EMAIL")); semAcessoEmail != "" {
		hashSemAcesso, err := hashSenhaEnv("LG_E2E_SEM_ACESSO_SENHA")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if err := upsertMembroE2E(ctx, tx, orgID, "Sem Acesso E2E", semAcessoEmail, hashSemAcesso, "membro", nil); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao salvar usuário sem acesso: %v\n", err)
			os.Exit(1)
		}
	}

	if removerEmail := strings.TrimSpace(os.Getenv("LG_E2E_REMOVER_EMAIL")); removerEmail != "" {
		hashRemover, err := hashSenhaEnv("LG_E2E_REMOVER_SENHA")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if err := upsertMembroE2E(ctx, tx, orgID, "Remover E2E", removerEmail, hashRemover, "membro", nil); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao salvar usuário removível E2E: %v\n", err)
			os.Exit(1)
		}
	}

	if gerenciadorEmail := strings.TrimSpace(os.Getenv("LG_E2E_GERENCIADOR_EMAIL")); gerenciadorEmail != "" {
		hashGerenciador, err := hashSenhaEnv("LG_E2E_GERENCIADOR_SENHA")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if err := upsertMembroE2E(ctx, tx, orgID, "Gerenciador E2E", gerenciadorEmail, hashGerenciador, "membro", []string{"gerenciar_pessoas"}); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao salvar gerenciador E2E: %v\n", err)
			os.Exit(1)
		}
	}

	if existenteEmail := strings.TrimSpace(os.Getenv("LG_E2E_EXISTENTE_EMAIL")); existenteEmail != "" {
		hashExistente, err := hashSenhaEnv("LG_E2E_EXISTENTE_SENHA")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		existenteID, err := upsertUsuarioE2E(ctx, tx, "Existente E2E", existenteEmail, hashExistente)
		if err != nil {
			fmt.Fprintf(os.Stderr, "erro ao salvar usuário existente E2E: %v\n", err)
			os.Exit(1)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM membros WHERE usuario_id = $1::uuid`, existenteID); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao limpar vínculos do usuário existente E2E: %v\n", err)
			os.Exit(1)
		}
		var orgHubID string
		if err := tx.QueryRow(ctx, `SELECT id::text FROM organizacoes WHERE nome = 'Laura Hub'`).Scan(&orgHubID); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao buscar organização Laura Hub: %v\n", err)
			os.Exit(1)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO membros (organizacao_id, usuario_id, papel)
			VALUES ($1::uuid, $2::uuid, 'membro')
			ON CONFLICT DO NOTHING`, orgHubID, existenteID); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao vincular usuário existente E2E à Laura Hub: %v\n", err)
			os.Exit(1)
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO ia_fornecedores
			(nome, tipo, cadastro, endereco, chave_final, limite_mensal_usd, aviso_percentual, uso_permitido, ativo)
		VALUES
			('OpenAI (referência E2E)', 'openai', 'chave_api', 'https://api.openai.com/v1', 'e2e0', 10.00, 80, 'todas', false)
		ON CONFLICT ((lower(nome))) DO NOTHING`); err != nil {
		fmt.Fprintf(os.Stderr, "erro ao preparar fornecedor de referência: %v\n", err)
		os.Exit(1)
	}

	if err := tx.Commit(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "erro ao confirmar transação: %v\n", err)
		os.Exit(1)
	}
}
