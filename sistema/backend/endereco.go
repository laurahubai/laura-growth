package main

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

type resolvedorIP func(ctx context.Context, host string) ([]netip.Addr, error)

var (
	errRedirecionamentoRecusado = errors.New("redirecionamentos não são permitidos")
	errRedeInternaRecusada      = errors.New("conexão recusada: endereço de rede interna não permitido")
)

var faixasInternas = func() []netip.Prefix {
	cidrs := []string{
		"0.0.0.0/8",
		"10.0.0.0/8",
		"100.64.0.0/10",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"172.16.0.0/12",
		"192.0.0.0/24",
		"192.168.0.0/16",
		"198.18.0.0/15",
		"224.0.0.0/4",
		"240.0.0.0/4",
		"::/128",
		"::1/128",
		"64:ff9b:1::/48",
		"100::/64",
		"fc00::/7",
		"fe80::/10",
		"fec0::/10",
		"ff00::/8",
	}
	out := make([]netip.Prefix, 0, len(cidrs))
	for _, c := range cidrs {
		out = append(out, netip.MustParsePrefix(c))
	}
	return out
}()

func ipPublico(ip netip.Addr) bool {
	if ip.Zone() != "" {
		return false
	}
	ip = ip.Unmap()
	if !ip.IsValid() || !ip.IsGlobalUnicast() || ip.IsPrivate() {
		return false
	}
	for _, faixa := range faixasInternas {
		if faixa.Contains(ip) {
			return false
		}
	}
	return true
}

func resolverDNS(ctx context.Context, host string) ([]netip.Addr, error) {
	return net.DefaultResolver.LookupNetIP(ctx, "ip", host)
}

// validarEndereco valida e normaliza uma URL externa contra SSRF (AC39).
func validarEndereco(ctx context.Context, bruto string) (string, error) {
	return validarEnderecoCom(ctx, bruto, resolverDNS, false)
}

func validarEnderecoCom(ctx context.Context, bruto string, resolver resolvedorIP, permitirRedeInterna bool) (string, error) {
	bruto = strings.TrimSpace(bruto)
	if bruto == "" {
		return "", errors.New("Informe o endereço do fornecedor.")
	}
	if utf8.RuneCountInString(bruto) > 500 {
		return "", errors.New("O endereço pode ter no máximo 500 caracteres.")
	}
	if strings.Contains(bruto, "#") {
		return "", errors.New("O endereço não pode conter fragmento (#).")
	}
	u, err := url.Parse(bruto)
	if err != nil {
		return "", errors.New("Endereço em formato inválido.")
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return "", errors.New("O endereço precisa começar com https://.")
	}
	if u.User != nil {
		return "", errors.New("O endereço não pode conter usuário ou senha.")
	}
	if u.Fragment != "" {
		return "", errors.New("O endereço não pode conter fragmento (#).")
	}
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if host == "" {
		return "", errors.New("Informe um endereço com domínio válido.")
	}
	if !permitirRedeInterna {
		if hostBloqueado(host) {
			return "", errors.New("Endereços de rede interna não são permitidos.")
		}
		if ip, err := netip.ParseAddr(host); err == nil {
			if !ipPublico(ip) {
				return "", errors.New("Endereços de rede interna não são permitidos.")
			}
		} else {
			if resolver == nil {
				resolver = resolverDNS
			}
			ips, err := resolver(ctx, host)
			if err != nil || len(ips) == 0 {
				return "", errors.New("Não foi possível resolver o endereço informado.")
			}
			for _, ip := range ips {
				if !ipPublico(ip) {
					return "", errors.New("O endereço aponta para uma rede interna e não é permitido.")
				}
			}
		}
	}
	u.Scheme = "https"
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath = ""
	return u.String(), nil
}

func hostBloqueado(host string) bool {
	return host == "localhost" ||
		strings.HasSuffix(host, ".localhost") ||
		host == "metadata.google.internal"
}

// novoClienteSeguro cria um cliente HTTP que confere o IP real de cada conexão
// (proteção contra DNS rebinding), exige TLS >= 1.2 e recusa redirecionamentos (AC39).
func novoClienteSeguro(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout: timeout,
		Control: func(_, address string, _ syscall.RawConn) error {
			ap, err := netip.ParseAddrPort(address)
			if err != nil {
				return errors.New("endereço de conexão inválido")
			}
			if !ipPublico(ap.Addr()) {
				return errRedeInternaRecusada
			}
			return nil
		},
	}
	transporte := &http.Transport{
		DialContext:         dialer.DialContext,
		TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12},
		TLSHandshakeTimeout: timeout,
		ForceAttemptHTTP2:   true,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transporte,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return errRedirecionamentoRecusado
		},
	}
}
