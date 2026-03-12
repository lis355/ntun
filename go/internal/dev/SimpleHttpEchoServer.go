package dev

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
)

type SimpleHttpEchoServer struct {
	serv *http.Server
}

func NewSimpleHttpEchoServer() *SimpleHttpEchoServer {
	return &SimpleHttpEchoServer{}
}

func (s *SimpleHttpEchoServer) ListenAndServe(port uint16) error {
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

	serv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	s.serv = serv

	listener, err := net.Listen("tcp", serv.Addr)
	if err != nil {
		slog.Error(fmt.Sprintf("[SimpleHttpEchoServer]: error %s", err))

		return err
	}

	slog.Info(fmt.Sprintf("[SimpleHttpEchoServer]: listening on http://localhost:%d", port))

	go serv.Serve(listener)

	return nil
}

func (s *SimpleHttpEchoServer) Close() error {
	err := s.serv.Close()
	if err != nil {
		slog.Error(fmt.Sprintf("[SimpleHttpEchoServer]: error %s", err))

		return err
	}

	slog.Info("[SimpleHttpEchoServer]: closed")

	return nil
}
