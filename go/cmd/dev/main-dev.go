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

	clientTransport := ntun.NewTcpClientTransport(fmt.Sprintf("localhost:%d", nodeTcpServerConnPort))
	clientNode := ntun.NewNode(uuid.New(), "client", clientTransport)
	slog.Info(fmt.Sprintf("Client node: %s", clientNode.String()))
	clientNode.Start()

	serverTransport := ntun.NewTcpServerTransport(nodeTcpServerConnPort)
	serverNode := ntun.NewNode(uuid.New(), "server", serverTransport)
	slog.Info(fmt.Sprintf("Server node: %s", serverNode.String()))
	serverNode.Start()

	clientNode.AddAllowedToConnectNodeId(serverNode.Id)
	serverNode.AddAllowedToConnectNodeId(clientNode.Id)

	const proxyServerPort = 8082

	inputSock5Server := ntun.NewInputSock5Server(proxyServerPort, func(ctx context.Context, scrConn net.Conn, address string) (net.Conn, error) {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("Ctx is done")
		default:
			return net.Dial("tcp", address)
		}
	})
	inputSock5Server.Start()

	const simpleHttpTimeServerPort = 8081
	var simpleHttpTimeServerRequestUrl = fmt.Sprintf("http://localhost:%d", simpleHttpTimeServerPort)
	simpleHttpTimeServerReady := make(chan struct{})
	go createAndListenSimpleHttpTimeServer(simpleHttpTimeServerPort, simpleHttpTimeServerReady)
	<-simpleHttpTimeServerReady

	socks5ProxyAddress := fmt.Sprintf("localhost:%d", proxyServerPort)

	os.Getenv("DEVELOPMENT")

	requestViaSocks5Proxy := func(url string) string {
		return request(socks5ProxyAddress, url)
	}

	requestViaSocks5Proxy(simpleHttpTimeServerRequestUrl)
	requestViaSocks5Proxy(os.Getenv("DEVELOP_GET_PUBLIC_IP_HTTP_URL"))
	requestViaSocks5Proxy(os.Getenv("DEVELOP_GET_PUBLIC_IP_HTTPS_URL"))

	// time.Sleep(time.Second * 3)

	// inputSock5Server.Stop()

	select {}
}
