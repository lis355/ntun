package connections

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	mathRand "math/rand"
	"net"
	"ntun/internal/utils"
	"sync"
	"testing"
)

func TestCipherAesGcmConnKeyLen(t *testing.T) {
	aConn, _ := net.Pipe()

	for _, keyLen := range []int{0, 1, 16, 24, 32, 13, 17, 25, 33} {
		_, err := NewCipherAesGcmConn(aConn, utils.RandBytes(keyLen))
		if err != nil {
			t.Errorf("bad key length: %v", err)
		}
	}
}

func TestCipherAesGcmConn(t *testing.T) {
	key := utils.RandBytes(32)

	aConn, bConn := utils.SocketPipe()
	aConn, _ = NewCipherAesGcmConn(aConn, key)
	bConn, _ = NewCipherAesGcmConn(bConn, key)

	var wg sync.WaitGroup
	wg.Add(2)

	writeBuf, readBuf := make([]byte, 0), make([]byte, 0)

	go func() {
		for range 2 {
			buf := utils.RandBytes(mathRand.Int() % 4096)
			writeBuf = append(writeBuf, buf...)
			aConn.Write(buf)
		}

		aConn.Close()

		wg.Done()
	}()

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := bConn.Read(buf)
			if n > 0 {
				readBuf = append(readBuf, buf[:n]...)
			}
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}

				fmt.Printf("read error: %v\n", err)
				break
			}
		}

		wg.Done()
	}()

	wg.Wait()

	if !bytes.Equal(writeBuf, readBuf) {
		t.Errorf("writeBuf != readBuf")
	}

	aConn.Close()
	bConn.Close()
}
