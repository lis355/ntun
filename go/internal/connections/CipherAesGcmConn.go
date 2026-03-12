package connections

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

const (
	maxMsgLen    = 4096
	msgHeaderLen = 2
	maxChunkLen  = maxMsgLen - 28 - msgHeaderLen
)

type CipherAesGcmConn struct {
	net.Conn
	aesGCM  cipher.AEAD
	readBuf []byte
}

func NewCipherAesGcmConn(conn net.Conn, key []byte) (*CipherAesGcmConn, error) {
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

	return &CipherAesGcmConn{
		Conn:    conn,
		aesGCM:  aesGCM,
		readBuf: make([]byte, 0),
	}, nil
}

func (c *CipherAesGcmConn) Write(b []byte) (n int, err error) {
	for i := 0; i < len(b); i += maxChunkLen {
		chunk := b[i:min(len(b), i+maxChunkLen)]
		encryptedChunk, err := c.encrypt(chunk)
		if err != nil {
			return n, err
		}

		msgHeaderBuf := make([]byte, msgHeaderLen)
		binary.BigEndian.PutUint16(msgHeaderBuf, uint16(len(encryptedChunk)))

		_, err = c.Conn.Write(msgHeaderBuf)
		if err != nil {
			return n, err
		}

		_, err = c.Conn.Write(encryptedChunk)
		if err != nil {
			return n, err
		}

		n += len(chunk)
	}

	return n, nil
}

func (c *CipherAesGcmConn) Read(b []byte) (n int, err error) {
	if len(c.readBuf) > 0 {
		n = copy(b, c.readBuf)
		c.readBuf = c.readBuf[n:]

		return n, nil
	}

	msgHeaderBuf := make([]byte, msgHeaderLen)
	_, err = io.ReadFull(c.Conn, msgHeaderBuf)
	if err != nil {
		return n, err
	}

	chunkLen := binary.BigEndian.Uint16(msgHeaderBuf)

	cipherBuf := make([]byte, chunkLen)
	_, err = io.ReadFull(c.Conn, cipherBuf)
	if err != nil {
		return n, err
	}

	buf, err := c.decrypt(cipherBuf)
	if err != nil {
		return n, err
	}

	c.readBuf = append(c.readBuf, buf...)

	n = copy(b, c.readBuf)
	c.readBuf = c.readBuf[n:]

	return n, nil
}

func (c *CipherAesGcmConn) encrypt(buf []byte) ([]byte, error) {
	nonce := make([]byte, c.aesGCM.NonceSize())
	rand.Read(nonce)

	cipherBuf := c.aesGCM.Seal(nonce, nonce, buf, nil)

	return cipherBuf, nil
}

func (c *CipherAesGcmConn) decrypt(cipherBuf []byte) ([]byte, error) {
	nonceSize := c.aesGCM.NonceSize()
	if len(cipherBuf) < nonceSize {
		return nil, fmt.Errorf("Bad decryption")
	}

	nonce := cipherBuf[:nonceSize]
	cipherBuf = cipherBuf[nonceSize:]

	buf, err := c.aesGCM.Open(nil, nonce, cipherBuf, nil)
	if err != nil {
		return nil, err
	}

	return buf, nil
}
