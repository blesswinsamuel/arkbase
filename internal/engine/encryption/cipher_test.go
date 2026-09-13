package encryption_test

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/blesswinsamuel/arkbase/internal/engine/encryption"
)

func TestEncryptionRoundtrip(t *testing.T) {
	passphrase := "my-ultra-secret-backup-passphrase"
	originalData := []byte("PostgreSQL database dump content: CREATE TABLE users (id serial, name text); INSERT INTO users VALUES (1, 'alice');")

	encrypted, err := encryption.EncryptBytes(passphrase, originalData)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	if bytes.Equal(encrypted, originalData) {
		t.Fatalf("encrypted data matches original")
	}

	decrypted, err := encryption.DecryptBytes(passphrase, encrypted)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if !bytes.Equal(decrypted, originalData) {
		t.Fatalf("decrypted data does not match original")
	}
}

func TestWrongPassphrase(t *testing.T) {
	passphrase := "correct-passphrase"
	wrongPass := "wrong-passphrase"
	data := []byte("important database secrets")

	encrypted, err := encryption.EncryptBytes(passphrase, data)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	_, err = encryption.DecryptBytes(wrongPass, encrypted)
	if err == nil {
		t.Fatalf("expected decryption error with wrong passphrase, got nil")
	}
}

func TestLargeStreamingEncryption(t *testing.T) {
	passphrase := "stream-key"
	// Generate 1MB of random data (spanning multiple 64KB chunks)
	largeData := make([]byte, 1024*1024)
	if _, err := rand.Read(largeData); err != nil {
		t.Fatalf("failed to generate random data: %v", err)
	}

	var encryptedBuf bytes.Buffer
	if err := encryption.EncryptStream(passphrase, bytes.NewReader(largeData), &encryptedBuf); err != nil {
		t.Fatalf("stream encryption failed: %v", err)
	}

	var decryptedBuf bytes.Buffer
	if err := encryption.DecryptStream(passphrase, &encryptedBuf, &decryptedBuf); err != nil {
		t.Fatalf("stream decryption failed: %v", err)
	}

	if !bytes.Equal(decryptedBuf.Bytes(), largeData) {
		t.Fatalf("large decrypted stream does not match original")
	}
}
