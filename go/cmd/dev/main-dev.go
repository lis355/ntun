package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"ntun/cmd/app"
	"ntun/cmd/ntun"
	"ntun/utils/log"
	"os"

	"github.com/armon/go-socks5"
	"github.com/google/uuid"
)

func createAndListenSocks5Server(proxyServerPort int, socks5ServerReady chan struct{}) {
	socks5ProxyAddress := fmt.Sprintf("socks5://localhost:%d", proxyServerPort)

	conf := &socks5.Config{
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			slog.Debug(fmt.Sprintf("Connection with %s via [%s]", address, socks5ProxyAddress))

			conn, err := net.Dial(network, address)
			if err != nil {
				return nil, err
			}

			if tcpConn, ok := conn.(*net.TCPConn); ok {
				conn = newObservableConn(tcpConn)
			} else {
				panic(fmt.Errorf("Strange conn"))
			}

			return conn, nil
		},
	}

	server, err := socks5.New(conf)
	if err != nil {
		panic(err)
	}

	socks5ServerReady <- struct{}{}

	if err := server.ListenAndServe("tcp", fmt.Sprintf(":%d", proxyServerPort)); err != nil {
		panic(err)
	}
}

func main() {
	app.Initialize()
	log.Initialize()

	slog.Info("DEVELOPMENT")

	slog.Info(fmt.Sprintf("%s v%s", app.Name, app.Version))
	if app.IsDevelopment {
		slog.Warn("Development mode")
	}

	const simpleHttpTimeServerPort = 8081
	var simpleHttpTimeServerRequestUrl = fmt.Sprintf("http://localhost:%d", simpleHttpTimeServerPort)
	go createAndListenSimpleHttpTimeServer(simpleHttpTimeServerPort)

	const proxyServerPort = 8080

	socks5ServerReady := make(chan struct{})
	go createAndListenSocks5Server(proxyServerPort, socks5ServerReady)
	<-socks5ServerReady

	socks5ProxyAddress := fmt.Sprintf("localhost:%d", proxyServerPort)

	os.Getenv("DEVELOPMENT")

	requestViaSocks5Proxy := func(url string) string {
		return request(socks5ProxyAddress, url)
	}

	requestViaSocks5Proxy(simpleHttpTimeServerRequestUrl)
	// requestViaSocks5Proxy(os.Getenv("DEVELOP_GET_PUBLIC_IP_HTTP_URL"))
	// requestViaSocks5Proxy(os.Getenv("DEVELOP_GET_PUBLIC_IP_HTTPS_URL"))

	const nodeServerPort = 8082

	clientNode := ntun.NewNode(uuid.New(), "client")
	slog.Info(fmt.Sprintf("Client node: %s", clientNode.String()))
	clientNode.Conn = ntun.NewTcpClientConn(fmt.Sprintf("localhost:%d", nodeServerPort))
	clientNode.Conn.Start()

	serverNode := ntun.NewNode(uuid.New(), "server")
	slog.Info(fmt.Sprintf("Server node: %s", serverNode.String()))
	serverNode.Conn = ntun.NewTcpServerConn(nodeServerPort)
	serverNode.Conn.Start()

	clientNode.AddAllowedToConnectNodeId(serverNode.Id)
	serverNode.AddAllowedToConnectNodeId(clientNode.Id)
}
