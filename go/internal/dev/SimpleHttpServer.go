package dev

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

type SimpleHttpEchoServer struct {
	serv *http.Server
	tls  bool
}

func NewSimpleHttpEchoServer() *SimpleHttpEchoServer {
	return &SimpleHttpEchoServer{}
}

func NewSimpleHttpsEchoServer() *SimpleHttpEchoServer {
	return &SimpleHttpEchoServer{tls: true}
}

func (s *SimpleHttpEchoServer) ListenAndServe(port uint16) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
		} else {
			n, err := strconv.Atoi(string(body))
			if err == nil &&
				n > 0 {
				time.Sleep(time.Duration(n) * time.Millisecond)
			}

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

	var protocol string
	if s.tls {
		protocol = "https"

		go serv.ServeTLS(listener, os.Getenv("DEVELOP_HTTPS_TEST_SERVER_CERT"), os.Getenv("DEVELOP_HTTPS_TEST_SERVER_KEY"))
	} else {
		protocol = "http"

		go serv.Serve(listener)
	}

	slog.Info(fmt.Sprintf("[SimpleHttpEchoServer]: listening on %s://localhost:%d", protocol, port))

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
