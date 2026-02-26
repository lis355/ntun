package main

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

func createAndListenSimpleHttpTimeServer(port int, ready chan struct{}) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(time.Now().Format(time.RFC3339)))
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		listener, err := net.Listen("tcp", server.Addr)
		if err != nil {
			panic(err)
		}

		slog.Info(fmt.Sprintf("[SimpleHttpTimeServer]: listening on http://localhost:%d", port))

		close(ready)

		server.Serve(listener)
	}()
}
