package mux

// import (
// 	"bytes"
// 	"encoding/binary"
// 	"fmt"
// 	"io"
// 	"ntun/internal/connections"
// 	"ntun/internal/utils"
// 	"slices"
// 	"sync"
// 	"testing"
// )

// func TestMux(t *testing.T) {
// 	aConn, bConn := utils.SocketPipe()

// 	aTrafficStatsConn, bTrafficStatsConn := connections.NewTrafficStatsConn(aConn), connections.NewTrafficStatsConn(bConn)
// 	aConn, bConn = aTrafficStatsConn, bTrafficStatsConn

// 	key := []byte("my key")
// 	aConn, _ = connections.NewCipherAesGcmConn(aConn, key)
// 	bConn, _ = connections.NewCipherAesGcmConn(bConn, key)

// 	aMux, _, bMux, _ := NewMux(aConn), NewMux(bConn)
// 	_ = bMux

// 	aMuxListener, _ := aMux.Listen()
// 	bMuxListener, _ := bMux.Listen()

// 	const streamsAmount = 3

// 	var wg sync.WaitGroup
// 	wg.Add(streamsAmount)

// 	for i := range streamsAmount {
// 		streamA, _ := aMux.CreateStream()
// 		streamB, err := bMuxListener.Accept()
// 		if err != nil {
// 			t.Fatalf("bad Accept")
// 		}

// 		go func() {
// 			tWriteBuf, tReadBuf := make([]byte, 0), make([]byte, 0)

// 			var wgReadWrite sync.WaitGroup
// 			wgReadWrite.Add(2)

// 			write := func() {
// 				defer wgReadWrite.Done()

// 				addressBuf := []byte(fmt.Sprintf("ADDR-%d", i))
// 				addressLenBuf := make([]byte, 4)
// 				binary.BigEndian.PutUint32(addressLenBuf, uint32(len(addressBuf)))

// 				dataBuf := []byte(fmt.Sprintf("DATA-%d", i))

// 				streamA.Write(addressLenBuf)
// 				streamA.Write(addressBuf)
// 				streamA.Write(dataBuf)

// 				tWriteBuf = slices.Concat(tWriteBuf, addressLenBuf, addressBuf, dataBuf)

// 				streamA.Close()
// 			}

// 			read := func() {
// 				defer wgReadWrite.Done()

// 				addressLenBuf := make([]byte, 4)
// 				io.ReadFull(streamB, addressLenBuf)

// 				addressLen := binary.BigEndian.Uint32(addressLenBuf)
// 				addressBuf := make([]byte, addressLen)
// 				io.ReadFull(streamB, addressBuf)

// 				address := string(addressBuf)
// 				_ = address

// 				tReadBuf = slices.Concat(tReadBuf, addressLenBuf, addressBuf)

// 				readBuf := make([]byte, 1024)
// 				for {
// 					n, err := streamB.Read(readBuf)
// 					if n > 0 {
// 						tReadBuf = append(tReadBuf, readBuf[:n]...)
// 					}
// 					if err != nil {
// 						return
// 					}
// 				}
// 			}

// 			write()
// 			read()

// 			wgReadWrite.Wait()

// 			if !bytes.Equal(tWriteBuf, tReadBuf) {
// 				t.Errorf("tWriteBuf (%s) != tReadBuf (%s)", tWriteBuf, tReadBuf)
// 			}

// 			wg.Done()
// 		}()
// 	}

// 	wg.Wait()

// 	aMuxListener.Close()
// 	bMuxListener.Close()

// 	aConn.Close()
// 	bConn.Close()

// 	if aTrafficStatsConn.Written() != bTrafficStatsConn.Readed() {
// 		t.Errorf("Written != Readed")
// 	}
// }
