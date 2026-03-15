package cipher

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
)

type CipherAesGcm struct {
	aesGCM cipher.AEAD
}

func NewCipherAesGcm(key []byte) (*CipherAesGcm, error) {
	// hash the key to 32 bytes, to not to depend on the key length
	keyHash32 := sha256.Sum256(key)
	keyHash := keyHash32[:]

	block, err := aes.NewCipher(keyHash)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &CipherAesGcm{
		aesGCM: aesGCM,
	}, nil
}

func (c *CipherAesGcm) Encrypt(buf []byte) ([]byte, error) {
	nonce := make([]byte, c.aesGCM.NonceSize())
	rand.Read(nonce)

	cipherBuf := c.aesGCM.Seal(nonce, nonce, buf, nil)

	return cipherBuf, nil
}

func (c *CipherAesGcm) Decrypt(cipherBuf []byte) ([]byte, error) {
	nonceSize := c.aesGCM.NonceSize()
	if len(cipherBuf) < nonceSize {
		return nil, fmt.Errorf("bad decryption")
	}

	nonce := cipherBuf[:nonceSize]
	cipherBuf = cipherBuf[nonceSize:]

	buf, err := c.aesGCM.Open(nil, nonce, cipherBuf, nil)
	if err != nil {
		return nil, err
	}

	return buf, nil
}
