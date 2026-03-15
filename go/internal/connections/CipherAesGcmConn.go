package connections

import (
	"encoding/binary"
	"io"
	"net"
	"ntun/internal/cipher"
)

const (
	maxMsgLen    = 4096
	msgHeaderLen = 2
	maxChunkLen  = maxMsgLen - 28 - msgHeaderLen
)

type CipherAesGcmConn struct {
	net.Conn
	cipher  *cipher.CipherAesGcm
	readBuf []byte
}

func NewCipherAesGcmConn(conn net.Conn, key []byte) (*CipherAesGcmConn, error) {
	cipher, err := cipher.NewCipherAesGcm(key)
	if err != nil {
		return nil, err
	}

	return &CipherAesGcmConn{
		Conn:    conn,
		cipher:  cipher,
		readBuf: make([]byte, 0),
	}, nil
}

func (c *CipherAesGcmConn) Write(b []byte) (n int, err error) {
	for i := 0; i < len(b); i += maxChunkLen {
		chunk := b[i:min(len(b), i+maxChunkLen)]
		encryptedChunk, err := c.cipher.Encrypt(chunk)
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

	buf, err := c.cipher.Decrypt(cipherBuf)
	if err != nil {
		return n, err
	}

	c.readBuf = append(c.readBuf, buf...)

	n = copy(b, c.readBuf)
	c.readBuf = c.readBuf[n:]

	return n, nil
}
