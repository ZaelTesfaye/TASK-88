package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

const (
	aes256KeySize = 32 // 256 bits
	gcmNonceSize  = 12 // 96 bits, standard for AES-GCM
)

// EncryptAES256GCM encrypts plaintext using AES-256-GCM.
// The key must be exactly 32 bytes (256 bits).
// Returns nonce (12 bytes) + ciphertext + GCM tag appended.
func EncryptAES256GCM(plaintext []byte, key []byte) ([]byte, error) {
	if len(key) != aes256KeySize {
		return nil, fmt.Errorf("key must be %d bytes, got %d", aes256KeySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate a random 12-byte nonce
	nonce := make([]byte, gcmNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Seal appends the ciphertext+tag to the nonce prefix
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptAES256GCM decrypts AES-256-GCM encrypted data.
// Expects input in the format: nonce (12 bytes) + ciphertext + GCM tag.
func DecryptAES256GCM(ciphertext []byte, key []byte) ([]byte, error) {
	if len(key) != aes256KeySize {
		return nil, fmt.Errorf("key must be %d bytes, got %d", aes256KeySize, len(key))
	}

	if len(ciphertext) < gcmNonceSize {
		return nil, fmt.Errorf("ciphertext too short: must be at least %d bytes", gcmNonceSize)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := ciphertext[:gcmNonceSize]
	encryptedData := ciphertext[gcmNonceSize:]

	plaintext, err := aesGCM.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// GenerateKey generates a new cryptographically secure AES-256 key (32 bytes).
func GenerateKey() ([]byte, error) {
	key := make([]byte, aes256KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

// WrapKey wraps (encrypts) a key for storage using a master key.
// Uses AES-256-GCM to encrypt the data key with the master key.
func WrapKey(key []byte, masterKey []byte) ([]byte, error) {
	if len(masterKey) != aes256KeySize {
		return nil, fmt.Errorf("master key must be %d bytes, got %d", aes256KeySize, len(masterKey))
	}
	return EncryptAES256GCM(key, masterKey)
}

// UnwrapKey unwraps (decrypts) a stored key using the master key.
func UnwrapKey(wrappedKey []byte, masterKey []byte) ([]byte, error) {
	if len(masterKey) != aes256KeySize {
		return nil, fmt.Errorf("master key must be %d bytes, got %d", aes256KeySize, len(masterKey))
	}
	return DecryptAES256GCM(wrappedKey, masterKey)
}
