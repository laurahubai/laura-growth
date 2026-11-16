package main

import (
	"errors"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const custoBcrypt = 12

func validarSenhaNova(senha string) error {
	if len(senha) < 10 {
		return errors.New("a senha precisa ter pelo menos 10 caracteres")
	}
	if len(senha) > 72 {
		return errors.New("a senha pode ter no máximo 72 caracteres")
	}
	return nil
}

func gerarHashSenha(senha string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(senha), custoBcrypt)
	return string(h), err
}

// hashFalso é comparado quando o e-mail não existe, para a resposta levar o mesmo
// tempo e não revelar quais e-mails têm conta.
var hashFalso = func() []byte {
	h, _ := bcrypt.GenerateFromPassword([]byte("senha-que-nunca-confere"), custoBcrypt)
	return h
}()

func senhaConfere(hash, senha string) bool {
	if hash == "" {
		_ = bcrypt.CompareHashAndPassword(hashFalso, []byte(senha))
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(senha)) == nil
}

// limitador barra tentativas de login repetidas para o mesmo e-mail:
// depois de maxFalhas erros dentro da janela, recusa até a janela passar.
type limitador struct {
	mu       sync.Mutex
	falhas   map[string][]time.Time
	maxFalhas int
	janela   time.Duration
}

func novoLimitador() *limitador {
	return &limitador{falhas: map[string][]time.Time{}, maxFalhas: 8, janela: 15 * time.Minute}
}

func (l *limitador) bloqueado(chave string, agora time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.recentes(chave, agora)) >= l.maxFalhas
}

func (l *limitador) registrarFalha(chave string, agora time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.falhas[chave] = append(l.recentes(chave, agora), agora)
}

func (l *limitador) limpar(chave string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.falhas, chave)
}

func (l *limitador) recentes(chave string, agora time.Time) []time.Time {
	var r []time.Time
	for _, t := range l.falhas[chave] {
		if agora.Sub(t) < l.janela {
			r = append(r, t)
		}
	}
	return r
}
