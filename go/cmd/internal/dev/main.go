package main

import (
	"fmt"
	"log/slog"
	"ntun/internal/app"
	"ntun/internal/cfg"
	"ntun/internal/dev"
	"ntun/internal/log"
	"ntun/internal/ntun/fabric"
	"ntun/internal/ntun/node"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
)

func main() {
	app.InitEnv()
	os.Setenv("LOG_LEVEL", "debug")
	log.Init()
	app.PrintLogo()
	app.PrintHeader()

	clientId, serverId := uuid.MustParse("4e82bc58-e39d-4aaf-86d5-54cfbe27ec53"), uuid.MustParse("145fe7cb-efef-46e3-afc0-f907759ec17c") // uuid.New(), uuid.New()
	cipherKey := "key"

	clientCfg := &cfg.Config{
		Name:      "client",
		Id:        clientId,
		Allowed:   []uuid.UUID{serverId},
		CipherKey: cipherKey,
		Input: &cfg.Socks5Input{
			Port: 8080,
		},
		// Transport: &cfg.TcpClientTransport{
		// 	Host: "localhost",
		// 	Port: 8081,
		// },
		Transport: &cfg.YandexWebRTCTransport{
			JoinId:   os.Getenv("DEVELOP_YANDEX_TELEMOST_JOIN_ID_OR_LINK"),
			MailUser: os.Getenv("DEVELOP_YANDEX_MAIL_USER"),
			MailPass: os.Getenv("DEVELOP_YANDEX_MAIL_PASSWORD"),
		},
	}

	serverCfg := &cfg.Config{
		Name:      "server",
		Id:        serverId,
		Allowed:   []uuid.UUID{clientId},
		CipherKey: cipherKey,
		Output:    &cfg.DirectOutput{},
		// Transport: &cfg.TcpServerTransport{
		// 	Host:      "0.0.0.0",
		// 	Port:      8081,
		// 	RateLimit: cfg.Rate{Value: 5 * 1024 * 1024 / 8, Interval: time.Second},
		// },
		Transport: &cfg.YandexWebRTCTransport{
			JoinId:    os.Getenv("DEVELOP_YANDEX_TELEMOST_JOIN_ID_OR_LINK"),
			MailUser:  os.Getenv("DEVELOP_YANDEX_MAIL_USER"),
			MailPass:  os.Getenv("DEVELOP_YANDEX_MAIL_PASSWORD"),
			RateLimit: cfg.Rate{Value: 5 * 1024 * 1024 / 8, Interval: time.Second},
		},
	}

	createAndStartNode := func(config *cfg.Config) *node.Node {
		n := node.NewNode(config)
		slog.Info(fmt.Sprintf("Node: %s", n.String()))
		input, _ := fabric.CreateInput(n)
		output, _ := fabric.CreateOutput(n)
		transporter, _ := fabric.CreateTransporter(n)
		n.AssignComponents(input, output, transporter)

		if n.Input != nil {
			if err := n.Input.Listen(); err != nil {
				panic(err)
			}
		}

		if n.Output != nil {
			if err := n.Output.Listen(); err != nil {
				panic(err)
			}
		}

		n.Start()

		return n
	}

	clientNode := createAndStartNode(clientCfg)
	time.Sleep(5 * time.Second)
	serverNode := createAndStartNode(serverCfg)
	time.Sleep(20 * time.Second)

	// Test

	const simpleHttpEchoServerPort = 8082
	var simpleHttpEchoServerRequestUrl = fmt.Sprintf("http://localhost:%d", simpleHttpEchoServerPort)
	simpleHttpEchoServer := dev.NewSimpleHttpEchoServer()
	simpleHttpEchoServer.ListenAndServe(simpleHttpEchoServerPort)

	// const simpleHttpsEchoServerPort = 8083
	// var simpleHttpsEchoServerRequestUrl = fmt.Sprintf("https://localhost:%d", simpleHttpsEchoServerPort)
	// simpleHttpsEchoServer := dev.NewSimpleHttpsEchoServer()
	// simpleHttpsEchoServer.ListenAndServe(simpleHttpsEchoServerPort)

	proxyServerPort := clientCfg.Input.(*cfg.Socks5Input).Port
	socks5ProxyAddress := fmt.Sprintf("localhost:%d", proxyServerPort)

	time.Sleep(5 * time.Second)

	requester, err := dev.NewRequester(socks5ProxyAddress)
	if err != nil {
		panic(err)
	}

	// for range 15 {
	// 	testStr := strconv.Itoa(1000)
	// 	requester.Post(simpleHttpEchoServerRequestUrl, testStr)
	// }

	testStr := strconv.Itoa(1000)
	result, err := requester.Post(simpleHttpEchoServerRequestUrl, testStr)
	if err != nil {
		panic(err)
	}

	if result != testStr {
		panic(fmt.Sprintf("result != testStr %s %s", result, testStr))
	}

	// result, err = requester.Post(simpleHttpsEchoServerRequestUrl, testStr)
	// if err != nil {
	// 	panic(err)
	// }

	// if result != testStr {
	// 	panic(fmt.Sprintf("result != testStr %s %s", result, testStr))
	// }

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
	// 	panic(fmt.Sprintf("ip != ipHttps %s %s", ip, ipHttps))
	// }

	// const n = 5
	// var wg sync.WaitGroup
	// wg.Add(n)
	// for range n {
	// 	go func() {
	// 		defer wg.Done()

	// 		ipHttps, err := requester.Post(simpleHttpsEchoServerRequestUrl, testStr)
	// 		// ipHttps, err := requester.Get(os.Getenv("DEVELOP_GET_PUBLIC_IP_HTTPS_URL"))
	// 		if err != nil {
	// 			slog.Error(err.Error())
	// 		}

	// 		_ = ipHttps
	// 	}()
	// }
	// wg.Wait()

	// _ = simpleHttpEchoServerRequestUrl

	select {}

	clientNode.Transporter.Close()
	serverNode.Transporter.Close()

	clientNode.Input.Close()
	serverNode.Output.Close()
}
