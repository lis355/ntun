package main

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"ntun/internal/app"
	"ntun/internal/log"
	"os"
)

func main() {
	app.InitEnv()
	os.Setenv("LOG_LEVEL", "debug")
	log.Init()

	var offer, answer []byte

	mux := http.NewServeMux()

	mux.HandleFunc("/offer", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if offer != nil {
				w.WriteHeader(http.StatusOK)
				w.Write(offer)
				offer = nil
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		case http.MethodPost:
			offer, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/answer", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if answer != nil {
				w.WriteHeader(http.StatusOK)
				w.Write(answer)
				answer = nil
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		case http.MethodPost:
			answer, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	port := 8060

	serv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	listener, err := net.Listen("tcp", serv.Addr)
	if err != nil {
		panic(fmt.Sprintf("[SignalServer]: error %s", err))
	}

	slog.Info(fmt.Sprintf("[SignalServer]: listening on http://localhost:%d", port))

	serv.Serve(listener)
}
