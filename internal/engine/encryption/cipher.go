package encryption

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	chunkSize   = 64 * 1024 // 64 KB chunks for low-memory streaming
	magicHeader = "ARK1"
	saltSize    = 16
	nonceSize   = 12
)

// EncryptStream encrypts data from src to dst using AES-256-GCM.
func EncryptStream(passphrase string, src io.Reader, dst io.Writer) error {
	if passphrase == "" {
		return errors.New("passphrase cannot be empty")
	}

	// 1. Generate random salt
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return fmt.Errorf("generate salt: %w", err)
	}

	// 2. Derive 32-byte key using PBKDF2 with SHA-256
	key := pbkdf2.Key([]byte(passphrase), salt, 100_000, 32, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("create GCM: %w", err)
	}

	// 3. Write magic header + salt
	if _, err := dst.Write([]byte(magicHeader)); err != nil {
		return err
	}
	if _, err := dst.Write(salt); err != nil {
		return err
	}

	// 4. Stream and encrypt chunks
	buf := make([]byte, chunkSize)
	var chunkIndex uint64

	for {
		n, err := io.ReadFull(src, buf)
		if n > 0 {
			nonce := make([]byte, nonceSize)
			binary.BigEndian.PutUint64(nonce[:8], chunkIndex)

			ciphertext := aead.Seal(nil, nonce, buf[:n], nil)

			// Write 4-byte chunk length
			lenBuf := make([]byte, 4)
			binary.BigEndian.PutUint32(lenBuf, uint32(len(ciphertext)))
			if _, err := dst.Write(lenBuf); err != nil {
				return err
			}
			if _, err := dst.Write(ciphertext); err != nil {
				return err
			}
			chunkIndex++
		}

		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read chunk: %w", err)
		}
	}

	// Write 0 length to signal end of stream
	endBuf := make([]byte, 4)
	_, err = dst.Write(endBuf)
	return err
}

// DecryptStream decrypts AES-256-GCM encrypted data from src to dst.
func DecryptStream(passphrase string, src io.Reader, dst io.Writer) error {
	if passphrase == "" {
		return errors.New("passphrase cannot be empty")
	}

	// 1. Read and verify magic header
	magic := make([]byte, len(magicHeader))
	if _, err := io.ReadFull(src, magic); err != nil {
		return fmt.Errorf("read magic header: %w", err)
	}
	if string(magic) != magicHeader {
		return errors.New("invalid file format or corrupt header")
	}

	// 2. Read salt
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(src, salt); err != nil {
		return fmt.Errorf("read salt: %w", err)
	}

	// 3. Derive key
	key := pbkdf2.Key([]byte(passphrase), salt, 100_000, 32, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("create GCM: %w", err)
	}

	var chunkIndex uint64
	lenBuf := make([]byte, 4)

	for {
		if _, err := io.ReadFull(src, lenBuf); err != nil {
			return fmt.Errorf("read chunk length: %w", err)
		}
		cLen := binary.BigEndian.Uint32(lenBuf)
		if cLen == 0 {
			// End of stream
			break
		}

		ciphertext := make([]byte, cLen)
		if _, err := io.ReadFull(src, ciphertext); err != nil {
			return fmt.Errorf("read ciphertext: %w", err)
		}

		nonce := make([]byte, nonceSize)
		binary.BigEndian.PutUint64(nonce[:8], chunkIndex)

		plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return fmt.Errorf("decryption failed (wrong passphrase or corrupted data): %w", err)
		}

		if _, err := dst.Write(plaintext); err != nil {
			return fmt.Errorf("write plaintext: %w", err)
		}

		chunkIndex++
	}

	return nil
}

// EncryptBytes encrypts in-memory bytes
func EncryptBytes(passphrase string, data []byte) ([]byte, error) {
	var out bytes.Buffer
	if err := EncryptStream(passphrase, bytes.NewReader(data), &out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// DecryptBytes decrypts in-memory bytes
func DecryptBytes(passphrase string, data []byte) ([]byte, error) {
	var out bytes.Buffer
	if err := DecryptStream(passphrase, bytes.NewReader(data), &out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
