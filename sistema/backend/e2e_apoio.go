//go:build ignore

// Utilitário exclusivo para preparação e limpeza do banco nos testes E2E (Playwright).
// Exige LG_E2E_BANCO_URL apontando para um banco cujo nome termine com _teste ou _e2e.
// Uso: go run ./e2e_apoio.go [preparar|limpar]
package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

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

func upsertMembroE2E(ctx context.Context, tx pgx.Tx, orgID, nome, email, senhaHash string, alcadas []string) error {
	var usuarioID string
	err := tx.QueryRow(ctx, `
		INSERT INTO usuarios (nome, email, senha_hash, admin_plataforma)
		VALUES ($1, $2, $3, false)
		ON CONFLICT ((lower(email))) DO UPDATE
			SET nome = EXCLUDED.nome,
			    senha_hash = EXCLUDED.senha_hash,
			    admin_plataforma = false
		RETURNING id::text`, nome, email, senhaHash).Scan(&usuarioID)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO membros (organizacao_id, usuario_id, papel)
		VALUES ($1, $2, 'membro')
		ON CONFLICT (organizacao_id, usuario_id) DO UPDATE SET papel = 'membro'`, orgID, usuarioID); err != nil {
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

	email := strings.TrimSpace(os.Getenv("LG_E2E_OPERACAO_EMAIL"))
	if email == "" {
		fmt.Fprintln(os.Stderr, "defina LG_E2E_OPERACAO_EMAIL")
		os.Exit(2)
	}
	semAcessoEmail := strings.TrimSpace(os.Getenv("LG_E2E_SEM_ACESSO_EMAIL"))
	existenteEmail := strings.TrimSpace(os.Getenv("LG_E2E_EXISTENTE_EMAIL"))

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, bancoURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro ao conectar no banco: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	switch acao {
	case "limpar":
		if _, err := pool.Exec(ctx, `DELETE FROM usuarios WHERE lower(email) = lower($1)`, email); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao limpar usuário de teste: %v\n", err)
			os.Exit(1)
		}
		if semAcessoEmail != "" {
			if _, err := pool.Exec(ctx, `DELETE FROM usuarios WHERE lower(email) = lower($1)`, semAcessoEmail); err != nil {
				fmt.Fprintf(os.Stderr, "erro ao limpar usuário sem acesso: %v\n", err)
				os.Exit(1)
			}
		}
		if existenteEmail != "" {
			if _, err := pool.Exec(ctx, `DELETE FROM usuarios WHERE lower(email) = lower($1)`, existenteEmail); err != nil {
				fmt.Fprintf(os.Stderr, "erro ao limpar usuário existente E2E: %v\n", err)
				os.Exit(1)
			}
		}
		if _, err := pool.Exec(ctx, `DELETE FROM convites WHERE lower(email) LIKE 'convidado-e2e-%@exemplo.test'`); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao limpar convites E2E: %v\n", err)
			os.Exit(1)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM usuarios WHERE lower(email) LIKE 'convidado-e2e-%@exemplo.test'`); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao limpar usuários convidados E2E: %v\n", err)
			os.Exit(1)
		}
		return
	case "preparar":
	default:
		fmt.Fprintf(os.Stderr, "ação desconhecida: %s\n", acao)
		os.Exit(2)
	}

	senha := os.Getenv("LG_E2E_OPERACAO_SENHA")
	if len(senha) < 10 {
		fmt.Fprintln(os.Stderr, "defina LG_E2E_OPERACAO_SENHA com pelo menos 10 caracteres")
		os.Exit(2)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(senha), 12)
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro ao gerar hash: %v\n", err)
		os.Exit(1)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro ao iniciar transação: %v\n", err)
		os.Exit(1)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var orgID string
	err = tx.QueryRow(ctx, `SELECT id::text FROM organizacoes WHERE nome = 'Laura Growth'`).Scan(&orgID)
	if errors.Is(err, pgx.ErrNoRows) {
		fmt.Fprintln(os.Stderr, "organização Laura Growth não encontrada; execute semear antes")
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro ao buscar organização: %v\n", err)
		os.Exit(1)
	}

	if err := upsertMembroE2E(ctx, tx, orgID, "Operação E2E", email, string(hash), []string{"ver_marca_posts"}); err != nil {
		fmt.Fprintf(os.Stderr, "erro ao salvar usuário de teste: %v\n", err)
		os.Exit(1)
	}

	if semAcessoEmail != "" {
		semAcessoSenha := os.Getenv("LG_E2E_SEM_ACESSO_SENHA")
		if len(semAcessoSenha) < 10 {
			fmt.Fprintln(os.Stderr, "defina LG_E2E_SEM_ACESSO_SENHA com pelo menos 10 caracteres")
			os.Exit(2)
		}
		hashSemAcesso, err := bcrypt.GenerateFromPassword([]byte(semAcessoSenha), 12)
		if err != nil {
			fmt.Fprintf(os.Stderr, "erro ao gerar hash: %v\n", err)
			os.Exit(1)
		}
		if err := upsertMembroE2E(ctx, tx, orgID, "Sem Acesso E2E", semAcessoEmail, string(hashSemAcesso), nil); err != nil {
			fmt.Fprintf(os.Stderr, "erro ao salvar usuário sem acesso: %v\n", err)
			os.Exit(1)
		}
	}

	if existenteEmail != "" {
		existenteSenha := os.Getenv("LG_E2E_EXISTENTE_SENHA")
		if len(existenteSenha) < 10 {
			fmt.Fprintln(os.Stderr, "defina LG_E2E_EXISTENTE_SENHA com pelo menos 10 caracteres")
			os.Exit(2)
		}
		hashExistente, err := bcrypt.GenerateFromPassword([]byte(existenteSenha), 12)
		if err != nil {
			fmt.Fprintf(os.Stderr, "erro ao gerar hash: %v\n", err)
			os.Exit(1)
		}
		var existenteID string
		if err := tx.QueryRow(ctx, `
			INSERT INTO usuarios (nome, email, senha_hash, admin_plataforma)
			VALUES ('Existente E2E', $1, $2, false)
			ON CONFLICT ((lower(email))) DO UPDATE
				SET nome = EXCLUDED.nome,
				    senha_hash = EXCLUDED.senha_hash,
				    admin_plataforma = false
			RETURNING id::text`, existenteEmail, string(hashExistente)).Scan(&existenteID); err != nil {
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
