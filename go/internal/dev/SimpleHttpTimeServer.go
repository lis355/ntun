package dev

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

func CreateAndListenSimpleHttpTimeServer(port int) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(time.Now().Format(time.RFC3339)))
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		slog.Error(fmt.Sprintf("[SimpleHttpTimeServer]: error %s", err))
		return
	}

	slog.Info(fmt.Sprintf("[SimpleHttpTimeServer]: listening on http://localhost:%d", port))

	go server.Serve(listener)
}
