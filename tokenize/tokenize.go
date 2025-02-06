package tokenize

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
)

type TokenVault struct {
	mu   sync.RWMutex
	data map[string][]byte
	key  []byte           
}

func NewTokenVault(key []byte) *TokenVault {
	return &TokenVault{
		data: make(map[string][]byte),
		key:  key,
	}
}

func (v *TokenVault) Tokenize(plaintext []byte) (string, error) {
	// Encrypt the data
	encrypted, err := v.encrypt(plaintext)
	if err != nil {
		return "", err
	}

	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(tokenBytes)

	v.mu.Lock()
	defer v.mu.Unlock()
	v.data[token] = encrypted

	return token, nil
}

func (v *TokenVault) Retrieve(token string) ([]byte, error) {
	v.mu.RLock()
	encrypted, exists := v.data[token]
	v.mu.RUnlock()

	if !exists {
		return nil, errors.New("token not found")
	}

	return v.decrypt(encrypted)
}

func (v *TokenVault) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(v.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (v *TokenVault) decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(v.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("invalid ciphertext")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func (v *TokenVault) ValidateToken(token string) bool {
	v.mu.RLock()
	_, exists := v.data[token]
	v.mu.RUnlock()
	return exists
}
