package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"ntun/internal/app"
	"ntun/internal/log"
	"ntun/internal/ntun/transport"
	"ntun/internal/utils"
	"os"
	"strings"
	"sync"

	"github.com/pion/webrtc/v3"
)

func main() {
	app.InitEnv()
	os.Setenv("LOG_LEVEL", "debug")
	log.Init()

	var iceServers []*webrtc.ICEServer
	var iceServer *webrtc.ICEServer
	err := json.Unmarshal([]byte(os.Getenv("DEVELOP_WEB_RTC_SERVERS")), &iceServers)
	if err != nil {
		panic(err)
	}

	for _, currentIceServer := range iceServers {
		for _, url := range currentIceServer.URLs {
			if strings.HasPrefix(url, "turn:") {
				// strings.Contains(url, "transport=tcp") {

				iceServer = currentIceServer
				break
			}
		}
	}

	if iceServer == nil {
		panic(fmt.Errorf("No turn servers"))
	}

	slog.Info(fmt.Sprintf("iceServer %s", iceServer.URLs[0]))

	webRtc1 := transport.NewWebRTCTransport()
	offerBuf, err := webRtc1.CreateOffer(iceServer)
	if err != nil {
		panic(fmt.Sprintf("failed to create offer %v", err))
	}

	webRtc2 := transport.NewWebRTCTransport()
	answerBuf, err := webRtc2.CreateAnswer(offerBuf)
	if err != nil {
		panic(fmt.Sprintf("failed to create answer %v", err))
	}

	err = webRtc1.SetAnswer(answerBuf)
	if err != nil {
		panic(fmt.Sprintf("failed to ser answer %v", err))
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		conn, err := webRtc1.Transport()
		if err != nil {
			panic(fmt.Sprintf("failed transport %v", err))
		}

		readBuf := make([]byte, 1024)
		for {
			n, err := conn.Read(readBuf)
			if err != nil {
				if errors.Is(err, io.EOF) ||
					errors.Is(err, io.ErrClosedPipe) {
					return
				}

				panic(err)
			}

			slog.Info(fmt.Sprintf("r %s", hex.EncodeToString(readBuf[:n])))
		}
	}()

	go func() {
		defer wg.Done()

		conn, err := webRtc2.Transport()
		if err != nil {
			panic(fmt.Sprintf("failed transport %v", err))
		}

		buf := utils.RandBytes(32)
		n, err := conn.Write(buf)
		if err != nil {

			panic(err)
		}

		slog.Info(fmt.Sprintf("w %s", hex.EncodeToString(buf[:n])))

		conn.Close()
	}()

	wg.Wait()
}
