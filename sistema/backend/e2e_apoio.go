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

	var usuarioID string
	err = tx.QueryRow(ctx, `
		INSERT INTO usuarios (nome, email, senha_hash, admin_plataforma)
		VALUES ('Operação E2E', $1, $2, false)
		ON CONFLICT ((lower(email))) DO UPDATE
			SET nome = EXCLUDED.nome,
			    senha_hash = EXCLUDED.senha_hash,
			    admin_plataforma = false
		RETURNING id::text`, email, string(hash)).Scan(&usuarioID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro ao salvar usuário de teste: %v\n", err)
		os.Exit(1)
	}

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

	if _, err := tx.Exec(ctx, `
		INSERT INTO membros (organizacao_id, usuario_id, papel)
		VALUES ($1, $2, 'membro')
		ON CONFLICT DO NOTHING`, orgID, usuarioID); err != nil {
		fmt.Fprintf(os.Stderr, "erro ao vincular membro: %v\n", err)
		os.Exit(1)
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
