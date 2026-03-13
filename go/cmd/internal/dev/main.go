package main

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"ntun/internal/app"
	"ntun/internal/conf"
	"ntun/internal/dev"
	"ntun/internal/log"
	"ntun/internal/utils"
	"ntun/ntun"
	"ntun/ntun/connections/inputs"
	"ntun/ntun/transport"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
)

func main() {
	app.InitEnv()
	log.Init()
	os.Setenv("DEVELOPMENT", "true")

	slog.Info(fmt.Sprintf("%s v%s (%s)", app.Name, app.Version, runtime.Version()))
	slog.Info("DEVELOPMENT")

	clientId, serverId := uuid.New(), uuid.New()
	cipherKey := hex.EncodeToString(utils.RandBytes(8))

	// Client

	const nodeTcpServerConnPort = 8080

	clientTransport := transport.NewTcpClientTransport(fmt.Sprintf("localhost:%d", nodeTcpServerConnPort))
	clientNode := ntun.NewNode(&conf.Config{Id: clientId, Name: "client", Allowed: []uuid.UUID{serverId}, CipherKey: cipherKey}, clientTransport)
	slog.Info(fmt.Sprintf("Client node: %s", clientNode.String()))
	clientNode.Start()

	const proxyServerPort = 8081
	sock5Server := inputs.NewSock5NoAuthServer(clientNode.ConnManager.Dial)
	err := sock5Server.ListenAndServe(proxyServerPort)
	if err != nil {
		panic(err)
	}

	// Server

	serverTransport := transport.NewTcpServerTransport(nodeTcpServerConnPort)
	serverNode := ntun.NewNode(&conf.Config{Id: serverId, Name: "server", Allowed: []uuid.UUID{clientId}, CipherKey: cipherKey}, serverTransport)
	slog.Info(fmt.Sprintf("Server node: %s", serverNode.String()))
	serverNode.Start()

	// Test

	time.Sleep(200 * time.Millisecond)

	const simpleHttpEchoServerPort = 8082
	var simpleHttpEchoServerRequestUrl = fmt.Sprintf("http://localhost:%d", simpleHttpEchoServerPort)
	simpleHttpEchoServer := dev.NewSimpleHttpEchoServer()
	simpleHttpEchoServer.ListenAndServe(simpleHttpEchoServerPort)

	const simpleHttpsEchoServerPort = 8083
	var simpleHttpsEchoServerRequestUrl = fmt.Sprintf("https://localhost:%d", simpleHttpsEchoServerPort)
	simpleHttpsEchoServer := dev.NewSimpleHttpsEchoServer()
	simpleHttpsEchoServer.ListenAndServe(simpleHttpsEchoServerPort)

	socks5ProxyAddress := fmt.Sprintf("localhost:%d", proxyServerPort)

	requester, err := dev.NewRequester(socks5ProxyAddress)
	if err != nil {
		panic(err)
	}

	testStr := hex.EncodeToString(utils.RandBytes(8))
	// result, err := requester.Post(simpleHttpTimeServerRequestUrl, testStr)
	// if err != nil {
	// 	panic(err)
	// }

	// if result != testStr {
	// 	slog.Error(fmt.Sprintf("result != testStr %s %s", result, testStr))

	// 	return
	// }

	// ip, err := requester.Get(os.Getenv("DEVELOP_GET_PUBLIC_IP_HTTP_URL"))
	// if err != nil {
	// 	panic(err)
	// }

	// slog.Info(fmt.Sprintf("Public IP %s", ip))

	const n = 5
	var wg sync.WaitGroup
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()

			ipHttps, err := requester.Post(simpleHttpsEchoServerRequestUrl, testStr)
			// ipHttps, err := requester.Get(os.Getenv("DEVELOP_GET_PUBLIC_IP_HTTPS_URL"))
			if err != nil {
				slog.Error(err.Error())
			}

			_ = ipHttps
		}()
	}
	wg.Wait()

	select {}

	// if ip != ipHttps {
	// 	slog.Error(fmt.Sprintf("ip != ipHttps %s %s", ip, ipHttps))

	// 	return
	// }

	_ = simpleHttpEchoServerRequestUrl

	sock5Server.Close()
}
