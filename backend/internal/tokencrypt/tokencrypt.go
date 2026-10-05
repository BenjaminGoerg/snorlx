// Package tokencrypt encrypts GitHub access tokens before they are written to the database.
//
// The key is derived from SESSION_SECRET with HKDF-SHA256, so operators configure a single secret.
// Values are stored as "enc:v1:<base64(nonce || ciphertext)>" using AES-256-GCM. A stored value
// without the prefix is treated as legacy plaintext and is re-encrypted on the next write.
package tokencrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

const (
	prefix    = "enc:v1:"
	keyLength = 32
	hkdfInfo  = "snorlx github token encryption v1"
)

// ErrInvalidCiphertext is returned when a stored value carries the encryption prefix but cannot be decrypted.
var ErrInvalidCiphertext = errors.New("tokencrypt: invalid ciphertext")

// Cipher encrypts and decrypts token strings.
type Cipher struct {
	aead cipher.AEAD
}

// New derives an AES-256-GCM key from secret.
func New(secret string) (*Cipher, error) {
	if secret == "" {
		return nil, errors.New("tokencrypt: secret must not be empty")
	}
	key, err := hkdf.Key(sha256.New, []byte(secret), nil, hkdfInfo, keyLength)
	if err != nil {
		return nil, fmt.Errorf("tokencrypt: derive key: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("tokencrypt: new cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("tokencrypt: new gcm: %w", err)
	}
	return &Cipher{aead: aead}, nil
}

// Encrypt returns the prefixed, base64 encoded ciphertext for plaintext. Empty input stays empty.
func (c *Cipher) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("tokencrypt: nonce: %w", err)
	}
	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return prefix + base64.RawStdEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt. Values without the prefix are returned unchanged (legacy plaintext).
func (c *Cipher) Decrypt(stored string) (string, error) {
	if !strings.HasPrefix(stored, prefix) {
		return stored, nil
	}
	raw, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(stored, prefix))
	if err != nil {
		return "", ErrInvalidCiphertext
	}
	nonceSize := c.aead.NonceSize()
	if len(raw) < nonceSize+c.aead.Overhead() {
		return "", ErrInvalidCiphertext
	}
	plaintext, err := c.aead.Open(nil, raw[:nonceSize], raw[nonceSize:], nil)
	if err != nil {
		return "", ErrInvalidCiphertext
	}
	return string(plaintext), nil
}

// IsEncrypted reports whether stored already carries the encryption prefix.
func IsEncrypted(stored string) bool {
	return strings.HasPrefix(stored, prefix)
}
