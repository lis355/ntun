package main

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"ntun/internal/app"
	"ntun/internal/dev"
	"ntun/internal/log"
	"ntun/internal/utils"
	"ntun/ntun"
	"ntun/ntun/connections/inputs"
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

	const simpleHttpEchoServerPort = 8081
	var simpleHttpTimeServerRequestUrl = fmt.Sprintf("http://localhost:%d", simpleHttpEchoServerPort)
	simpleHttpEchoServer := dev.NewSimpleHttpEchoServer()
	simpleHttpEchoServer.ListenAndServe(simpleHttpEchoServerPort)

	const proxyServerPort = 8082
	sock5Server := inputs.NewSock5NoAuthServer(clientNode.ConnManager.Dial)
	err := sock5Server.ListenAndServe(proxyServerPort)
	if err != nil {
		panic(err)
	}

	socks5ProxyAddress := fmt.Sprintf("localhost:%d", proxyServerPort)

	requester, err := dev.NewRequester(socks5ProxyAddress)
	if err != nil {
		panic(err)
	}

	testStr := hex.EncodeToString(utils.RandBytes(8))
	result, err := requester.Post(simpleHttpTimeServerRequestUrl, testStr)
	if err != nil {
		panic(err)
	}

	if result != testStr {
		slog.Error(fmt.Sprintf("result != testStr %s %s", result, testStr))

		return
	}

	// ip, err := requester.Get(os.Getenv("DEVELOP_GET_PUBLIC_IP_HTTP_URL"))
	// if err != nil {
	// 	panic(err)
	// }

	// slog.Info(fmt.Sprintf("Public IP %s", ip))

	// ipHttps, err := requester.Get(os.Getenv("DEVELOP_GET_PUBLIC_IP_HTTPS_URL"))
	// if err != nil {
	// 	panic(err)
	// }

	// if ip != ipHttps {
	// 	slog.Error(fmt.Sprintf("ip != ipHttps %s %s", ip, ipHttps))

	// 	return
	// }

	sock5Server.Close()
}
