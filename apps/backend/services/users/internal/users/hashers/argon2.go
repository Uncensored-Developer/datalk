package hashers

import (
	"context"

	"github.com/matthewhartstonge/argon2"
)

type Argon2Hasher struct {
	config argon2.Config
}

func NewArgon2Hasher() *Argon2Hasher {
	return &Argon2Hasher{
		config: argon2.DefaultConfig(),
	}
}

func (a *Argon2Hasher) Hash(ctx context.Context, password string) (string, error) {
	encoded, err := a.config.HashEncoded([]byte(password))
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func (a *Argon2Hasher) Verify(ctx context.Context, password, hash string) (bool, error) {
	return argon2.VerifyEncoded([]byte(password), []byte(hash))
}
