package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/hkdf"
)

const (
	versaoCifragem  byte = 0x01
	infoHKDF             = "laura-growth/ia-chaves/v1"
	tamanhoChave         = 32
	minSegredoChars      = 32
)

// Cifrador protege chaves de API em repouso com AES-256-GCM e AAD por fornecedor (AC32).
type Cifrador struct {
	chave []byte
}

// String evita que a chave derivada apareça se o cifrador for formatado por acidente.
func (c *Cifrador) String() string   { return "[cifrador]" }
func (c *Cifrador) GoString() string { return "[cifrador]" }

// novoCifrador deriva uma chave AES-256 usando HKDF-SHA256 a partir de LG_CHAVE_CIFRAGEM.
func novoCifrador(segredo string) (*Cifrador, error) {
	if strings.TrimSpace(segredo) == "" || utf8.RuneCountInString(segredo) < minSegredoChars {
		return nil, errors.New("defina LG_CHAVE_CIFRAGEM com pelo menos 32 caracteres")
	}
	leitor := hkdf.New(sha256.New, []byte(segredo), nil, []byte(infoHKDF))
	chave := make([]byte, tamanhoChave)
	if _, err := io.ReadFull(leitor, chave); err != nil {
		return nil, errors.New("não foi possível preparar a chave de cifragem")
	}
	return &Cifrador{chave: chave}, nil
}

// cifrar grava no formato 0x01 || nonce(12) || texto cifrado, vinculando ao fornecedorID (AAD).
func (c *Cifrador) cifrar(fornecedorID, textoPlano string) ([]byte, error) {
	if c == nil || len(c.chave) != tamanhoChave {
		return nil, errors.New("cifrador não configurado")
	}
	if fornecedorID == "" {
		return nil, errors.New("identificador do fornecedor obrigatório para cifrar")
	}
	bloco, err := aes.NewCipher(c.chave)
	if err != nil {
		return nil, errors.New("não foi possível cifrar a chave")
	}
	gcm, err := cipher.NewGCM(bloco)
	if err != nil {
		return nil, errors.New("não foi possível cifrar a chave")
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, errors.New("não foi possível cifrar a chave")
	}
	cifrado := gcm.Seal(nil, nonce, []byte(textoPlano), []byte(fornecedorID))
	saida := make([]byte, 0, 1+len(nonce)+len(cifrado))
	saida = append(saida, versaoCifragem)
	saida = append(saida, nonce...)
	saida = append(saida, cifrado...)
	return saida, nil
}

// decifrar valida versão, nonce e AAD sem expor conteúdo cifrado ou segredos em caso de falha.
func (c *Cifrador) decifrar(fornecedorID string, dados []byte) (string, error) {
	if c == nil || len(c.chave) != tamanhoChave {
		return "", errors.New("cifrador não configurado")
	}
	bloco, err := aes.NewCipher(c.chave)
	if err != nil {
		return "", errors.New("não foi possível decifrar a chave")
	}
	gcm, err := cipher.NewGCM(bloco)
	if err != nil {
		return "", errors.New("não foi possível decifrar a chave")
	}
	minimo := 1 + gcm.NonceSize() + gcm.Overhead()
	if len(dados) < minimo {
		return "", errors.New("chave cifrada inválida")
	}
	if dados[0] != versaoCifragem {
		return "", errors.New("versão de cifragem não suportada")
	}
	inicioCipher := 1 + gcm.NonceSize()
	nonce := dados[1:inicioCipher]
	cifrado := dados[inicioCipher:]
	plano, err := gcm.Open(nil, nonce, cifrado, []byte(fornecedorID))
	if err != nil {
		return "", errors.New("não foi possível decifrar a chave")
	}
	return string(plano), nil
}
