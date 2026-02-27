package main

import (
	"fmt"
	"log/slog"
	"ntun/cmd/app"
	"ntun/cmd/ntun"
	"ntun/utils/log"
	"os"

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

	const nodeTcpServerConnPort = 8080

	clientConn := ntun.NewTcpClientConn(fmt.Sprintf("localhost:%d", nodeTcpServerConnPort))
	clientNode := ntun.NewNode(uuid.New(), "client", clientConn)
	slog.Info(fmt.Sprintf("Client node: %s", clientNode.String()))
	clientNode.Start()

	serverConn := ntun.NewTcpServerConn(nodeTcpServerConnPort)
	serverNode := ntun.NewNode(uuid.New(), "server", serverConn)
	slog.Info(fmt.Sprintf("Server node: %s", serverNode.String()))
	serverNode.Start()

	clientNode.AddAllowedToConnectNodeId(serverNode.Id)
	serverNode.AddAllowedToConnectNodeId(clientNode.Id)

	const simpleHttpTimeServerPort = 8081
	var simpleHttpTimeServerRequestUrl = fmt.Sprintf("http://localhost:%d", simpleHttpTimeServerPort)
	simpleHttpTimeServerReady := make(chan struct{})
	go createAndListenSimpleHttpTimeServer(simpleHttpTimeServerPort, simpleHttpTimeServerReady)
	<-simpleHttpTimeServerReady

	const proxyServerPort = 8082

	inputSock5Server := ntun.NewInputSock5Server(proxyServerPort, clientNode.ConnManager)
	inputSock5Server.Start()

	socks5ProxyAddress := fmt.Sprintf("localhost:%d", proxyServerPort)

	os.Getenv("DEVELOPMENT")

	requestViaSocks5Proxy := func(url string) string {
		return request(socks5ProxyAddress, url)
	}

	requestViaSocks5Proxy(simpleHttpTimeServerRequestUrl)
	// // requestViaSocks5Proxy(os.Getenv("DEVELOP_GET_PUBLIC_IP_HTTP_URL"))
	// // requestViaSocks5Proxy(os.Getenv("DEVELOP_GET_PUBLIC_IP_HTTPS_URL"))

	// DEBUG
	select {}
}
