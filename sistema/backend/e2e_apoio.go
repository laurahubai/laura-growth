//go:build ignore

// Utilitário exclusivo para preparação do banco nos testes E2E (Playwright).
// Cria/atualiza um usuário comum (admin_plataforma = false) e um fornecedor OpenAI
// inativo para validar os campos de chave na interface sem chamadas externas.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	bancoURL := os.Getenv("LG_BANCO_URL")
	email := strings.TrimSpace(os.Getenv("LG_E2E_OPERACAO_EMAIL"))
	senha := os.Getenv("LG_E2E_OPERACAO_SENHA")
	if bancoURL == "" || email == "" || senha == "" {
		fmt.Fprintln(os.Stderr, "defina LG_BANCO_URL, LG_E2E_OPERACAO_EMAIL e LG_E2E_OPERACAO_SENHA")
		os.Exit(2)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, bancoURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "erro ao conectar no banco: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

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
