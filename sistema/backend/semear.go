package main

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Organizações da fase 1A: a própria Laura usa a plataforma para as duas marcas.
var organizacoesIniciais = []struct{ nome, tipo string }{
	{"Laura Growth", "empresa"},
	{"Laura Hub", "empresa"},
}

// semear cria (ou atualiza) a primeira pessoa e lhe dá acesso às organizações iniciais.
// Pode rodar várias vezes sem duplicar nada.
func semear(ctx context.Context, banco *Banco, nome, email, senha string) error {
	email = strings.TrimSpace(email)
	if nome == "" || email == "" || senha == "" {
		return errors.New("defina LG_SEMENTE_NOME, LG_SEMENTE_EMAIL e LG_SEMENTE_SENHA")
	}
	if err := validarSenhaNova(senha); err != nil {
		return err
	}
	hash, err := gerarHashSenha(senha)
	if err != nil {
		return err
	}
	tx, err := banco.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var usuarioID string
	err = tx.QueryRow(ctx, `
		INSERT INTO usuarios (nome, email, senha_hash) VALUES ($1, $2, $3)
		ON CONFLICT ((lower(email))) DO UPDATE SET nome = EXCLUDED.nome, senha_hash = EXCLUDED.senha_hash
		RETURNING id`, nome, email, hash).Scan(&usuarioID)
	if err != nil {
		return err
	}
	for _, o := range organizacoesIniciais {
		var orgID string
		err := tx.QueryRow(ctx, `SELECT id FROM organizacoes WHERE nome = $1`, o.nome).Scan(&orgID)
		if errors.Is(err, pgx.ErrNoRows) {
			if err := tx.QueryRow(ctx, `INSERT INTO organizacoes (nome, tipo) VALUES ($1, $2) RETURNING id`, o.nome, o.tipo).Scan(&orgID); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO membros (organizacao_id, usuario_id, papel) VALUES ($1, $2, 'dono')
			ON CONFLICT DO NOTHING`, orgID, usuarioID); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	slog.Info("semente aplicada", "organizacoes", len(organizacoesIniciais))
	return nil
}
