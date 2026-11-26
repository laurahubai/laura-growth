package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migracoes/*.sql
var arquivosMigracao embed.FS

type Banco = pgxpool.Pool

func conectar(ctx context.Context, url string) (*Banco, error) {
	banco, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("conectar ao banco: %w", err)
	}
	if err := banco.Ping(ctx); err != nil {
		banco.Close()
		return nil, fmt.Errorf("banco não respondeu: %w", err)
	}
	return banco, nil
}

// migrar aplica, em ordem, os arquivos de migracoes/ que ainda não foram aplicados.
// Cada arquivo roda numa transação; uma trava impede duas migrações ao mesmo tempo.
func migrar(ctx context.Context, banco *Banco) error {
	return migrarAte(ctx, banco, "")
}

// migrarAte aplica as migrações pendentes até o prefixo ou nome informado (inclusive).
// Se limite for vazio, aplica todas as migrações disponíveis.
func migrarAte(ctx context.Context, banco *Banco, limite string) error {
	conn, err := banco.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock(7140001)`); err != nil {
		return err
	}
	defer conn.Exec(context.Background(), `SELECT pg_advisory_unlock(7140001)`) //nolint:errcheck

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS migracoes_aplicadas (
		nome text PRIMARY KEY, aplicada_em timestamptz NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	nomes, err := fs.Glob(arquivosMigracao, "migracoes/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(nomes)
	for _, caminho := range nomes {
		nome := strings.TrimPrefix(caminho, "migracoes/")
		var existe bool
		if err := conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM migracoes_aplicadas WHERE nome = $1)`, nome).Scan(&existe); err != nil {
			return err
		}
		if !existe {
			sql, err := arquivosMigracao.ReadFile(caminho)
			if err != nil {
				return err
			}
			tx, err := conn.Begin(ctx)
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, string(sql)); err != nil {
				_ = tx.Rollback(ctx)
				return fmt.Errorf("migração %s: %w", nome, err)
			}
			if _, err := tx.Exec(ctx, `INSERT INTO migracoes_aplicadas (nome) VALUES ($1)`, nome); err != nil {
				_ = tx.Rollback(ctx)
				return err
			}
			if err := tx.Commit(ctx); err != nil {
				return err
			}
			slog.Info("migração aplicada", "nome", nome)
		}
		if limite != "" && (nome == limite || strings.HasPrefix(nome, limite+"_")) {
			break
		}
	}
	return nil
}
