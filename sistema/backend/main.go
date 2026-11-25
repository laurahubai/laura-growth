// Servidor da Plataforma Laura Growth.
//
// Uso: go run . migrar | semear | servir
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "uso: laura-growth-api migrar | semear | servir")
		os.Exit(2)
	}
	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parar()
	if err := executar(ctx, os.Args[1]); err != nil {
		slog.Error("falhou", "comando", os.Args[1], "erro", err)
		os.Exit(1)
	}
}

func executar(ctx context.Context, comando string) error {
	banco, err := conectar(ctx, exigirEnv("LG_BANCO_URL"))
	if err != nil {
		return err
	}
	defer banco.Close()

	switch comando {
	case "migrar":
		return migrar(ctx, banco)
	case "semear":
		return semear(ctx, banco, os.Getenv("LG_SEMENTE_NOME"), os.Getenv("LG_SEMENTE_EMAIL"), os.Getenv("LG_SEMENTE_SENHA"))
	case "servir":
		return servir(ctx, banco)
	default:
		return fmt.Errorf("comando desconhecido: %s", comando)
	}
}

func servir(ctx context.Context, banco *Banco) error {
	cifrador, err := novoCifrador(os.Getenv("LG_CHAVE_CIFRAGEM"))
	if err != nil {
		return err
	}
	endereco := os.Getenv("LG_API_ENDERECO")
	if endereco == "" {
		endereco = "127.0.0.1:8090"
	}
	api := NovaAPI(banco, os.Getenv("LG_COOKIE_SEGURO") == "true", cifrador)
	srv := &http.Server{
		Addr:              endereco,
		Handler:           api.Rotas(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		desligar, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelar()
		_ = srv.Shutdown(desligar)
	}()
	slog.Info("API no ar", "endereco", endereco)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func exigirEnv(nome string) string {
	v := os.Getenv(nome)
	if v == "" {
		fmt.Fprintf(os.Stderr, "variável %s não definida (veja .env.example)\n", nome)
		os.Exit(2)
	}
	return v
}
