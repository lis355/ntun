package dev

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
)

func ListenAndServeSimpleHttpEchoServer(port int) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(body)
		}
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		slog.Error(fmt.Sprintf("[SimpleHttpEchoServer]: error %s", err))
		return
	}

	slog.Info(fmt.Sprintf("[SimpleHttpEchoServer]: listening on http://localhost:%d", port))

	go server.Serve(listener)
}
