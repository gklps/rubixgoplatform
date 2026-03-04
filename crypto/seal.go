package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

const (
	sealVersion1  byte   = 0x01
	argon2Time    uint32 = 1
	argon2Memory  uint32 = 64 * 1024
	argon2Threads uint8  = 4
	argon2KeyLen  uint32 = 32
	saltSize             = 16
)

// deriveKeyArgon2id derives a 32-byte AES key using argon2id
func deriveKeyArgon2id(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
}

// deriveKeySHA256 derives a 32-byte AES key using sha256 (legacy)
func deriveKeySHA256(password string) []byte {
	h := sha256.New()
	h.Write([]byte(password))
	return h.Sum(nil)
}

// Seal encrypts data using AES-GCM with argon2id key derivation.
// Output format: [0x01][16-byte salt][nonce][ciphertext]
func Seal(key string, data []byte) ([]byte, error) {
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generating salt: %w", err)
	}

	k := deriveKeyArgon2id(key, salt)
	b, err := aes.NewCipher(k)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(b)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)

	result := make([]byte, 1+saltSize+len(ciphertext))
	result[0] = sealVersion1
	copy(result[1:], salt)
	copy(result[1+saltSize:], ciphertext)
	return result, nil
}

// UnSeal decrypts data. Supports both new (argon2id) and legacy (sha256) formats.
func UnSeal(key string, data []byte) ([]byte, error) {
	if len(data) < 1 {
		return nil, fmt.Errorf("invalid data: too short")
	}

	if data[0] == sealVersion1 && len(data) > 1+saltSize {
		// New format: [0x01][salt][nonce+ciphertext]
		salt := data[1 : 1+saltSize]
		encrypted := data[1+saltSize:]
		k := deriveKeyArgon2id(key, salt)
		b, err := aes.NewCipher(k)
		if err != nil {
			return nil, err
		}
		gcm, err := cipher.NewGCM(b)
		if err != nil {
			return nil, err
		}
		nonceSize := gcm.NonceSize()
		if len(encrypted) < nonceSize {
			return nil, fmt.Errorf("invalid data: encrypted portion too short")
		}
		nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]
		return gcm.Open(nil, nonce, ciphertext, nil)
	}

	// Legacy format: sha256(password) key, no version prefix, just nonce+ciphertext
	k := deriveKeySHA256(key)
	b, err := aes.NewCipher(k)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(b)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("invalid data")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
