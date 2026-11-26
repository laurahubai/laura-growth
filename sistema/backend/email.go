package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

const (
	enderecoZohoPadrao  = "http://127.0.0.1:8030"
	tempoLimiteEmail    = 45 * time.Second
	limiteRespostaEmail = int64(64 << 10)
)

var errEmailNaoConfigurado = errors.New("serviço de e-mail não configurado")

// enviadorEmail abstrai o envio de e-mails transacionais (ex.: convites).
type enviadorEmail interface {
	enviar(ctx context.Context, para, assunto, texto string) error
}

// enviadorEmailZoho envia e-mails pelo adaptador local Zoho Mail compartilhado.
type enviadorEmailZoho struct {
	urlBase string
	chave   string
	cliente *http.Client
}

func novoEnviadorEmailZoho(urlBase, chave string) *enviadorEmailZoho {
	return novoEnviadorEmailZohoComTimeout(urlBase, chave, tempoLimiteEmail)
}

func urlZohoSegura(urlBase string) bool {
	u, err := url.Parse(urlBase)
	if err != nil || u.Host == "" {
		return false
	}
	if strings.EqualFold(u.Scheme, "https") {
		return true
	}
	if !strings.EqualFold(u.Scheme, "http") {
		return false
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if host == "localhost" {
		return true
	}
	ip, err := netip.ParseAddr(host)
	return err == nil && ip.Unmap().IsLoopback()
}

func novoEnviadorEmailZohoComTimeout(urlBase, chave string, timeout time.Duration) *enviadorEmailZoho {
	base := strings.TrimRight(strings.TrimSpace(urlBase), "/")
	if base == "" {
		base = enderecoZohoPadrao
	}
	chaveLimpa := strings.TrimSpace(chave)
	if !urlZohoSegura(base) {
		chaveLimpa = ""
	}
	return &enviadorEmailZoho{
		urlBase: base,
		chave:   chaveLimpa,
		cliente: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return errRedirecionamentoRecusado
			},
		},
	}
}

func (z *enviadorEmailZoho) enviar(ctx context.Context, para, assunto, texto string) error {
	if z == nil || z.chave == "" || !urlZohoSegura(z.urlBase) {
		return errEmailNaoConfigurado
	}

	corpo, err := json.Marshal(struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
		Text    string `json:"text"`
	}{
		To:      para,
		Subject: assunto,
		Text:    texto,
	})
	if err != nil {
		return errors.New("não foi possível preparar o e-mail")
	}

	destino := strings.TrimRight(z.urlBase, "/") + "/v1/email/send"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, destino, bytes.NewReader(corpo))
	if err != nil {
		return errors.New("não foi possível preparar o pedido ao serviço de e-mail")
	}
	req.Header.Set("Authorization", "Bearer "+z.chave)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")

	res, err := z.cliente.Do(req)
	if err != nil {
		if errors.Is(err, errRedirecionamentoRecusado) {
			return errors.New("serviço de e-mail tentou redirecionar a requisição")
		}
		return errors.New("serviço de e-mail indisponível")
	}
	defer res.Body.Close()

	bruto, err := io.ReadAll(io.LimitReader(res.Body, limiteRespostaEmail))
	if err != nil {
		return errors.New("falha ao ler resposta do serviço de e-mail")
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("serviço de e-mail respondeu HTTP %d", res.StatusCode)
	}

	var resp struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(bruto, &resp); err != nil || resp.Status != "sent" {
		return errors.New("resposta inesperada do serviço de e-mail")
	}
	return nil
}
