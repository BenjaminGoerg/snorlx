package tokencrypt

import (
	"errors"
	"strings"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	c, err := New("a-very-long-session-secret-used-for-tests")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	stored, err := c.Encrypt("gho_plaintext_token")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if !strings.HasPrefix(stored, "enc:v1:") {
		t.Fatalf("expected prefix, got %q", stored)
	}
	if strings.Contains(stored, "gho_plaintext_token") {
		t.Fatal("ciphertext leaks plaintext")
	}
	got, err := c.Decrypt(stored)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != "gho_plaintext_token" {
		t.Fatalf("Decrypt = %q", got)
	}
}

func TestEncrypt_RandomNonce(t *testing.T) {
	c, _ := New("a-very-long-session-secret-used-for-tests")
	a, _ := c.Encrypt("same")
	b, _ := c.Encrypt("same")
	if a == b {
		t.Fatal("two encryptions of the same plaintext must differ")
	}
}

func TestDecrypt_LegacyPlaintextPassthrough(t *testing.T) {
	c, _ := New("a-very-long-session-secret-used-for-tests")
	got, err := c.Decrypt("gho_legacy")
	if err != nil || got != "gho_legacy" {
		t.Fatalf("legacy passthrough failed: %q %v", got, err)
	}
	if IsEncrypted("gho_legacy") {
		t.Fatal("legacy value reported as encrypted")
	}
}

func TestDecrypt_WrongKeyFails(t *testing.T) {
	c1, _ := New("secret-one-that-is-long-enough-for-tests")
	c2, _ := New("secret-two-that-is-long-enough-for-tests")
	stored, _ := c1.Encrypt("token")
	if _, err := c2.Decrypt(stored); !errors.Is(err, ErrInvalidCiphertext) {
		t.Fatalf("expected ErrInvalidCiphertext, got %v", err)
	}
}

func TestDecrypt_Tampered(t *testing.T) {
	c, _ := New("a-very-long-session-secret-used-for-tests")
	stored, _ := c.Encrypt("token")
	tampered := stored[:len(stored)-2] + "AA"
	if _, err := c.Decrypt(tampered); !errors.Is(err, ErrInvalidCiphertext) {
		t.Fatalf("expected ErrInvalidCiphertext, got %v", err)
	}
	if _, err := c.Decrypt("enc:v1:!!!"); !errors.Is(err, ErrInvalidCiphertext) {
		t.Fatalf("expected ErrInvalidCiphertext for bad base64, got %v", err)
	}
	if _, err := c.Decrypt("enc:v1:AAAA"); !errors.Is(err, ErrInvalidCiphertext) {
		t.Fatalf("expected ErrInvalidCiphertext for short payload, got %v", err)
	}
}

func TestEmpty(t *testing.T) {
	c, _ := New("a-very-long-session-secret-used-for-tests")
	stored, err := c.Encrypt("")
	if err != nil || stored != "" {
		t.Fatalf("empty plaintext should stay empty: %q %v", stored, err)
	}
	if _, err := New(""); err == nil {
		t.Fatal("empty secret must be rejected")
	}
}
