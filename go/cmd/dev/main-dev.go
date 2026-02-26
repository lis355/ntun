package main

import (
	"fmt"
	"log/slog"
	"ntun/cmd/app"
	"ntun/cmd/ntun"
	"ntun/utils/log"
	"time"

	"github.com/google/uuid"
)

func main() {
	app.Initialize()
	log.Initialize()

	slog.Info("DEVELOPMENT")

	slog.Info(fmt.Sprintf("%s v%s", app.Name, app.Version))
	if app.IsDevelopment {
		slog.Warn("Development mode")
	}

	const nodeServerPort = 8080

	clientNode := ntun.NewNode(uuid.New(), "client")
	slog.Info(fmt.Sprintf("Client node: %s", clientNode.String()))
	clientNode.Conn = ntun.NewTcpClientConn(fmt.Sprintf("localhost:%d", nodeServerPort))
	// clientNode.Conn.Start()

	for {
		slog.Info(fmt.Sprintf("Start"))
		clientNode.Conn.Start()

		time.Sleep(2 * time.Second)

		slog.Info(fmt.Sprintf("Stop"))
		clientNode.Conn.Stop()

		time.Sleep(1 * time.Second)
	}

	// DEBUG
	select {}

	// serverNode := ntun.NewNode(uuid.New(), "server")
	// slog.Info(fmt.Sprintf("Server node: %s", serverNode.String()))
	// serverNode.Conn = ntun.NewTcpServerConn(nodeServerPort)
	// serverNode.Conn.Start()

	// clientNode.AddAllowedToConnectNodeId(serverNode.Id)
	// serverNode.AddAllowedToConnectNodeId(clientNode.Id)

	// const simpleHttpTimeServerPort = 8081
	// var simpleHttpTimeServerRequestUrl = fmt.Sprintf("http://localhost:%d", simpleHttpTimeServerPort)
	// simpleHttpTimeServerReady := make(chan struct{})
	// go createAndListenSimpleHttpTimeServer(simpleHttpTimeServerPort, simpleHttpTimeServerReady)
	// <-simpleHttpTimeServerReady

	// const proxyServerPort = 8082

	// socks5ServerReady := make(chan struct{})
	// ntun.CreateAndListenSocks5Server(proxyServerPort, clientNode.Conn.(ntun.ConnIn), socks5ServerReady)
	// <-socks5ServerReady

	// socks5ProxyAddress := fmt.Sprintf("localhost:%d", proxyServerPort)

	// os.Getenv("DEVELOPMENT")

	// requestViaSocks5Proxy := func(url string) string {
	// 	return request(socks5ProxyAddress, url)
	// }

	// requestViaSocks5Proxy(simpleHttpTimeServerRequestUrl)
	// requestViaSocks5Proxy(os.Getenv("DEVELOP_GET_PUBLIC_IP_HTTP_URL"))
	// requestViaSocks5Proxy(os.Getenv("DEVELOP_GET_PUBLIC_IP_HTTPS_URL"))
}
