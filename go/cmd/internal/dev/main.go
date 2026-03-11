package main

import (
	"fmt"
	"log/slog"
	"ntun/internal/app"
	"ntun/internal/dev"
	"ntun/internal/log"
	"ntun/ntun"
	"ntun/ntun/transport"
	"os"
	"runtime"

	"github.com/google/uuid"
)

func main() {
	app.InitEnv()
	log.Init()
	os.Setenv("DEVELOPMENT", "true")

	slog.Info(fmt.Sprintf("%s v%s (%s)", app.Name, app.Version, runtime.Version()))
	slog.Info("DEVELOPMENT")

	const nodeTcpServerConnPort = 8080

	clientTransport := transport.NewTcpClientTransport(fmt.Sprintf("localhost:%d", nodeTcpServerConnPort))
	clientNode := ntun.NewNode(uuid.New(), "client", clientTransport)
	slog.Info(fmt.Sprintf("Client node: %s", clientNode.String()))
	clientNode.Start()

	serverTransport := transport.NewTcpServerTransport(nodeTcpServerConnPort)
	serverNode := ntun.NewNode(uuid.New(), "server", serverTransport)
	slog.Info(fmt.Sprintf("Server node: %s", serverNode.String()))
	serverNode.Start()

	clientNode.AddAllowedToConnectNodeId(serverNode.Id)
	serverNode.AddAllowedToConnectNodeId(clientNode.Id)

	const proxyServerPort = 8082

	inputSock5Server := transport.NewInputSock5Server(proxyServerPort, clientNode.ConnManager.Dial)
	inputSock5Server.Start()

	const simpleHttpTimeServerPort = 8081
	var simpleHttpTimeServerRequestUrl = fmt.Sprintf("http://localhost:%d", simpleHttpTimeServerPort)
	dev.CreateAndListenSimpleHttpTimeServer(simpleHttpTimeServerPort)

	socks5ProxyAddress := fmt.Sprintf("localhost:%d", proxyServerPort)

	requester, err := dev.NewRequester(socks5ProxyAddress)
	if err != nil {
		panic(err)
	}

	requester.Get(simpleHttpTimeServerRequestUrl)
	// requester.Get(os.Getenv("DEVELOP_GET_PUBLIC_IP_HTTP_URL"))
	// requester.Get(os.Getenv("DEVELOP_GET_PUBLIC_IP_HTTPS_URL"))

	// inputSock5Server.Stop()
}
