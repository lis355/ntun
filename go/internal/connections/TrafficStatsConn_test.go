package connections

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	mathRand "math/rand"
	"ntun/internal/utils"
	"sync"
	"testing"
)

func TestTrafficStatsConn(t *testing.T) {
	aConn, bConn := utils.SocketPipe()
	aConn = NewTrafficStatsConn(aConn)
	bConn = NewTrafficStatsConn(bConn)

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
		buf := make([]byte, 1024*10)
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

	aConn.Close()
	bConn.Close()

	if !bytes.Equal(writeBuf, readBuf) {
		t.Errorf("writeBuf != readBuf")
	}

	if bConn.(*TrafficStatsConn).Readed() != len(readBuf) {
		t.Errorf("Readed != len(readBuf)")
	}

	if aConn.(*TrafficStatsConn).Written() != len(writeBuf) {
		t.Errorf("Written != len(writeBuf)")
	}
}

func TestTrafficStatsConnResetStats(t *testing.T) {
	aConn, bConn := utils.SocketPipe()
	aConn = NewTrafficStatsConn(aConn)
	bConn = NewTrafficStatsConn(bConn)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		aConn.Write([]byte("hello"))
	}()

	go func() {
		defer wg.Done()

		bConn.Read(make([]byte, 5))
	}()

	wg.Wait()

	if aConn.(*TrafficStatsConn).Written() != 5 {
		t.Errorf("Written != 5")
	}

	if bConn.(*TrafficStatsConn).Readed() != 5 {
		t.Errorf("Readed != 5")
	}

	aConn.(*TrafficStatsConn).ResetStats()
	bConn.(*TrafficStatsConn).ResetStats()

	if aConn.(*TrafficStatsConn).Written() != 0 {
		t.Errorf("Written != 0")
	}

	if bConn.(*TrafficStatsConn).Readed() != 0 {
		t.Errorf("Readed != 0")
	}

	aConn.Close()
	bConn.Close()
}
