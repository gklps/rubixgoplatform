package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"io"
	"testing"
)

func TestSealUnseal_NewFormat(t *testing.T) {
	plaintext := []byte("hello rubix ledger")
	key := "test-password-123"

	sealed, err := Seal(key, plaintext)
	if err != nil {
		t.Fatalf("Seal() error: %v", err)
	}

	// Verify version byte
	if sealed[0] != sealVersion1 {
		t.Fatalf("expected version byte 0x01, got 0x%02x", sealed[0])
	}

	// Verify total length: 1 + 16 (salt) + 12 (nonce) + len(plaintext) + 16 (GCM tag)
	expectedMin := 1 + saltSize + 12 + len(plaintext) + 16
	if len(sealed) < expectedMin {
		t.Fatalf("sealed output too short: got %d, want >= %d", len(sealed), expectedMin)
	}

	// Round-trip
	got, err := UnSeal(key, sealed)
	if err != nil {
		t.Fatalf("UnSeal() error: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("UnSeal() got %q, want %q", got, plaintext)
	}
}

func TestSealUnseal_WrongKey(t *testing.T) {
	plaintext := []byte("secret data")
	sealed, err := Seal("correct-key", plaintext)
	if err != nil {
		t.Fatalf("Seal() error: %v", err)
	}
	_, err = UnSeal("wrong-key", sealed)
	if err == nil {
		t.Fatal("UnSeal() with wrong key should return error, got nil")
	}
}

// TestUnSeal_LegacyFormat verifies that data encrypted with the old sha256-based
// Seal can still be decrypted by the new UnSeal.
func TestUnSeal_LegacyFormat(t *testing.T) {
	plaintext := []byte("legacy encrypted data")
	key := "old-password"

	// Produce legacy-format ciphertext (nonce || ciphertext, sha256 key)
	legacy, err := sealLegacyForTest(key, plaintext)
	if err != nil {
		t.Fatalf("sealLegacyForTest() error: %v", err)
	}

	// The new UnSeal must handle it
	got, err := UnSeal(key, legacy)
	if err != nil {
		t.Fatalf("UnSeal() on legacy format error: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("UnSeal() on legacy got %q, want %q", got, plaintext)
	}
}

// TestUnSeal_LegacyStartsWith0x01 verifies backward-compat fallback when a
// legacy ciphertext's first byte is 0x01 (1/256 chance in production).
func TestUnSeal_LegacyStartsWith0x01(t *testing.T) {
	key := "password"
	// Craft a legacy ciphertext whose first byte is forced to 0x01
	// by trying until we get one (probabilistic but fast in practice).
	var legacy []byte
	for i := 0; i < 10000; i++ {
		candidate, err := sealLegacyForTest(key, []byte("data"))
		if err != nil {
			t.Fatalf("sealLegacyForTest: %v", err)
		}
		if candidate[0] == 0x01 {
			legacy = candidate
			break
		}
	}
	if legacy == nil {
		t.Skip("could not generate legacy ciphertext starting with 0x01 in 10000 tries (extremely unlikely)")
	}

	got, err := UnSeal(key, legacy)
	if err != nil {
		t.Fatalf("UnSeal() on legacy starting with 0x01: %v", err)
	}
	if !bytes.Equal(got, []byte("data")) {
		t.Fatalf("UnSeal() got %q, want %q", got, []byte("data"))
	}
}

// sealLegacyForTest produces ciphertext in the old sha256 format for testing.
func sealLegacyForTest(key string, data []byte) ([]byte, error) {
	h := sha256.New()
	h.Write([]byte(key))
	k := h.Sum(nil)
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
	return gcm.Seal(nonce, nonce, data, nil), nil
}
