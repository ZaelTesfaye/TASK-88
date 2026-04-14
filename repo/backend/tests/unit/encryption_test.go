package unit

import (
	"bytes"
	"testing"

	"backend/internal/security"
)

func TestAES256GCMEncryptDecrypt(t *testing.T) {
	key, err := security.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	plaintexts := []string{
		"Hello, World!",
		"",
		"Short",
		"A longer message that contains multiple words and special characters! @#$%^&*()",
		"Unicode: \u00e9\u00e8\u00ea \u00fc\u00f6\u00e4 \u4f60\u597d",
	}

	for _, pt := range plaintexts {
		t.Run(pt, func(t *testing.T) {
			plaintext := []byte(pt)

			ciphertext, err := security.EncryptAES256GCM(plaintext, key)
			if err != nil {
				t.Fatalf("EncryptAES256GCM failed: %v", err)
			}

			if len(pt) > 0 && bytes.Equal(ciphertext, plaintext) {
				t.Error("ciphertext should not equal plaintext")
			}

			decrypted, err := security.DecryptAES256GCM(ciphertext, key)
			if err != nil {
				t.Fatalf("DecryptAES256GCM failed: %v", err)
			}

			if !bytes.Equal(decrypted, plaintext) {
				t.Errorf("decrypted text does not match original.\nExpected: %q\nGot: %q",
					pt, string(decrypted))
			}
		})
	}

	// Encrypting the same plaintext twice should produce different ciphertexts (random nonce).
	plaintext := []byte("deterministic test")
	ct1, err := security.EncryptAES256GCM(plaintext, key)
	if err != nil {
		t.Fatalf("first encryption failed: %v", err)
	}
	ct2, err := security.EncryptAES256GCM(plaintext, key)
	if err != nil {
		t.Fatalf("second encryption failed: %v", err)
	}
	if bytes.Equal(ct1, ct2) {
		t.Error("two encryptions of the same plaintext should produce different ciphertexts due to random nonce")
	}
}

func TestAES256GCMWrongKey(t *testing.T) {
	key1, err := security.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey (key1) failed: %v", err)
	}
	key2, err := security.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey (key2) failed: %v", err)
	}

	// Ensure keys are different.
	if bytes.Equal(key1, key2) {
		t.Fatal("two generated keys should be different")
	}

	plaintext := []byte("secret message")
	ciphertext, err := security.EncryptAES256GCM(plaintext, key1)
	if err != nil {
		t.Fatalf("EncryptAES256GCM failed: %v", err)
	}

	// Decrypting with the wrong key should fail.
	_, err = security.DecryptAES256GCM(ciphertext, key2)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong key, got nil")
	}

	// Invalid key length should also fail.
	shortKey := []byte("too-short")
	_, err = security.EncryptAES256GCM(plaintext, shortKey)
	if err == nil {
		t.Fatal("expected error for short key, got nil")
	}

	_, err = security.DecryptAES256GCM(ciphertext, shortKey)
	if err == nil {
		t.Fatal("expected error for short key on decrypt, got nil")
	}

	// Tampered ciphertext should fail.
	tampered := make([]byte, len(ciphertext))
	copy(tampered, ciphertext)
	// Flip a byte in the ciphertext portion (after the 12-byte nonce).
	if len(tampered) > 13 {
		tampered[13] ^= 0xFF
	}
	_, err = security.DecryptAES256GCM(tampered, key1)
	if err == nil {
		t.Fatal("expected error for tampered ciphertext, got nil")
	}
}

func TestKeyGeneration(t *testing.T) {
	key, err := security.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	// Key must be exactly 32 bytes (256 bits).
	if len(key) != 32 {
		t.Fatalf("expected 32-byte key, got %d bytes", len(key))
	}

	// Key should not be all zeros.
	allZero := true
	for _, b := range key {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("generated key should not be all zeros")
	}

	// Two keys should be different.
	key2, err := security.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey (second) failed: %v", err)
	}
	if bytes.Equal(key, key2) {
		t.Error("two generated keys should be different")
	}

	// Key should be usable for encryption.
	plaintext := []byte("test encryption with generated key")
	ct, err := security.EncryptAES256GCM(plaintext, key)
	if err != nil {
		t.Fatalf("encryption with generated key failed: %v", err)
	}
	pt, err := security.DecryptAES256GCM(ct, key)
	if err != nil {
		t.Fatalf("decryption with generated key failed: %v", err)
	}
	if !bytes.Equal(pt, plaintext) {
		t.Error("round-trip encryption/decryption failed with generated key")
	}
}

func TestWrapUnwrapKey(t *testing.T) {
	masterKey, err := security.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey (master) failed: %v", err)
	}

	dataKey, err := security.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey (data) failed: %v", err)
	}

	wrapped, err := security.WrapKey(dataKey, masterKey)
	if err != nil {
		t.Fatalf("WrapKey failed: %v", err)
	}
	if bytes.Equal(wrapped, dataKey) {
		t.Error("wrapped key must not equal plain data key")
	}

	unwrapped, err := security.UnwrapKey(wrapped, masterKey)
	if err != nil {
		t.Fatalf("UnwrapKey failed: %v", err)
	}
	if !bytes.Equal(unwrapped, dataKey) {
		t.Error("unwrapped key must match original data key")
	}
}

func TestWrapKeyBadMasterKeyLength(t *testing.T) {
	dataKey, _ := security.GenerateKey()
	_, err := security.WrapKey(dataKey, []byte("short"))
	if err == nil {
		t.Fatal("WrapKey should reject short master key")
	}

	_, err = security.UnwrapKey([]byte("fake-wrapped"), []byte("short"))
	if err == nil {
		t.Fatal("UnwrapKey should reject short master key")
	}
}
