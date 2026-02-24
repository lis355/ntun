package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

func createAndListenSimpleHttpTimeServer(port int) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(time.Now().Format(time.RFC3339)))
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	slog.Info(fmt.Sprintf("Simple time HTTP server listening on http://localhost:%d", port))

	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}
