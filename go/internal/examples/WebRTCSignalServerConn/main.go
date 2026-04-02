package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"ntun/internal/app"
	"ntun/internal/log"
	"ntun/internal/ntun/transport"
	"ntun/internal/utils"
	"os"
	"strings"
	"time"

	"github.com/pion/webrtc/v3"
)

func main() {
	app.InitEnv()
	os.Setenv("LOG_LEVEL", "debug")
	log.Init()

	isClient := os.Getenv("DEVELOP_WEB_RTC_CALLER") == "true"
	signalServerUrl := os.Getenv("DEVELOP_WEB_RTC_SIGNAL_SERVER_URL")

	webRtc := transport.NewWebRTCTransport()

	if isClient {
		var iceServers []*webrtc.ICEServer
		var iceServer *webrtc.ICEServer
		err := json.Unmarshal([]byte(os.Getenv("DEVELOP_WEB_RTC_SERVERS")), &iceServers)
		if err != nil {
			panic(err)
		}

		for _, currentIceServer := range iceServers {
			for _, url := range currentIceServer.URLs {
				if strings.HasPrefix(url, "turn:") {
					iceServer = currentIceServer
					break
				}
			}
		}

		if iceServer == nil {
			panic(fmt.Errorf("No turn servers"))
		}

		slog.Info(fmt.Sprintf("iceServer %s", iceServer.URLs[0]))

		offerBuf, err := webRtc.CreateOffer(iceServer)
		if err != nil {
			panic(fmt.Sprintf("failed to create offer %v", err))
		}

		slog.Info(fmt.Sprintf("created offer %d", len(offerBuf)))

		offerUrl, _ := url.JoinPath(signalServerUrl, "offer")
		http.Post(offerUrl, "text/plain", bytes.NewReader(offerBuf))

		slog.Info("check answer")

		for {
			answerUrl, _ := url.JoinPath(signalServerUrl, "answer")
			res, err := http.Get(answerUrl)
			if err != nil {
				panic(err)
			}

			if res.StatusCode != http.StatusOK {
				time.Sleep(1 * time.Second)

				continue
			}

			body, _ := io.ReadAll(res.Body)
			res.Body.Close()

			slog.Info(fmt.Sprintf("set answer %d", len(body)))

			err = webRtc.SetAnswer(body)
			if err != nil {
				panic(fmt.Sprintf("failed to ser answer %v", err))
			}

			break
		}

		slog.Info("create transport")

		conn, err := webRtc.Transport()
		if err != nil {
			panic(fmt.Sprintf("failed transport %v", err))
		}

		buf := utils.RandBytes(32)
		n, err := conn.Write(buf)
		if err != nil {

			panic(err)
		}

		slog.Info(fmt.Sprintf("w %s", hex.EncodeToString(buf[:n])))

		time.Sleep(1 * time.Second)

		conn.Close()
	} else {
		slog.Info("check offer")

		for {
			offerUrl, _ := url.JoinPath(signalServerUrl, "offer")
			res, err := http.Get(offerUrl)
			if err != nil {
				panic(err)
			}

			if res.StatusCode != http.StatusOK {
				time.Sleep(1 * time.Second)

				continue
			}

			body, _ := io.ReadAll(res.Body)
			res.Body.Close()

			slog.Info(fmt.Sprintf("set offer %d", len(body)))

			answerBuf, err := webRtc.CreateAnswer(body)
			if err != nil {
				panic(fmt.Sprintf("failed to create answer %v", err))
			}

			slog.Info(fmt.Sprintf("created answer %d", len(answerBuf)))

			answerUrl, _ := url.JoinPath(signalServerUrl, "answer")
			http.Post(answerUrl, "text/plain", bytes.NewReader(answerBuf))

			break
		}

		slog.Info("create transport")

		conn, err := webRtc.Transport()
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
	}
}
