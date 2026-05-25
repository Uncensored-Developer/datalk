package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"strings"

	"github.com/mdobak/go-xerrors"
)

const ciphertextPrefix = "v1:"

type Cipher interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

type AESCipher struct {
	aead cipher.AEAD
}

func NewAESCipher(secret string) (*AESCipher, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, xerrors.New("provider config secret is required")
	}

	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, xerrors.Newf("failed to create aes cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, xerrors.Newf("failed to create aes-gcm cipher: %w", err)
	}

	return &AESCipher{aead: aead}, nil
}

func (c *AESCipher) Encrypt(plaintext string) (string, error) {
	if c == nil || c.aead == nil {
		return "", xerrors.New("cipher is not configured")
	}

	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", xerrors.Newf("failed to generate nonce: %w", err)
	}

	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return ciphertextPrefix + base64.StdEncoding.EncodeToString(sealed), nil
}

func (c *AESCipher) Decrypt(ciphertext string) (string, error) {
	if c == nil || c.aead == nil {
		return "", xerrors.New("cipher is not configured")
	}
	if !strings.HasPrefix(ciphertext, ciphertextPrefix) {
		return ciphertext, nil
	}

	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(ciphertext, ciphertextPrefix))
	if err != nil {
		return "", xerrors.Newf("failed to decode ciphertext: %w", err)
	}
	if len(raw) < c.aead.NonceSize() {
		return "", xerrors.New("ciphertext is too short")
	}

	nonce := raw[:c.aead.NonceSize()]
	encrypted := raw[c.aead.NonceSize():]
	plaintext, err := c.aead.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", xerrors.Newf("failed to decrypt ciphertext: %w", err)
	}

	return string(plaintext), nil
}

type PlaintextCipher struct{}

func (PlaintextCipher) Encrypt(plaintext string) (string, error) {
	return plaintext, nil
}

func (PlaintextCipher) Decrypt(ciphertext string) (string, error) {
	return ciphertext, nil
}
