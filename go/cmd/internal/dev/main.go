package main

import (
	"encoding/hex"
	"encoding/json"
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
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
	"go.yaml.in/yaml/v3"
)

func main() {
	app.Init()
	log.Init()
	os.Setenv("DEVELOPMENT", "true")

	slog.Info(fmt.Sprintf("%s v%s (%s)", app.Name, app.Version, runtime.Version()))
	slog.Info("DEVELOPMENT")

	clientId, serverId := uuid.New(), uuid.New()
	cipherKey := hex.EncodeToString(utils.RandBytes(8))

	/////////////////////

	var iceServers []webrtc.ICEServer

	type IceServersCache struct {
		Time       time.Time
		IceServers []webrtc.ICEServer
	}

	iceServersCacheBuf, err := app.ReadCacheFile("iceServers.yaml")
	if err != nil {
		panic(err)
	}

	var iceServersCache IceServersCache
	if err := yaml.Unmarshal(iceServersCacheBuf, &iceServersCache); err != nil {
		panic(err)
	}

	if time.Since(iceServersCache.Time) < 24*time.Hour {
		iceServers = iceServersCache.IceServers
	} else {
		// iceServer, err := yandex.GetIceServerFromJoinIdOrLink(os.Getenv("DEVELOP_YANDEX_TELEMOST_JOIN_ID_OR_LINK"))
		// slog.Info(fmt.Sprintf("GetIceServerFromJoinIdOrLink %+v", iceServer))
		// if err != nil {
		// 	panic(err)
		// }

		// iceServers := []webrtc.ICEServer{*iceServer}
		// iceServersJson, _ := json.Marshal(&iceServers)
		// slog.Info(fmt.Sprintf("DEVELOP_WEB_RTC_SERVERS=%s", iceServersJson))

		json.Unmarshal([]byte(os.Getenv("DEVELOP_WEB_RTC_SERVERS")), &iceServers)

		iceServersCache := &IceServersCache{Time: time.Now(), IceServers: iceServers}
		iceServersCacheBuf, err := yaml.Marshal(&iceServersCache)

		err = app.WriteCacheFile("iceServers.yaml", iceServersCacheBuf)
		if err != nil {
			panic(err)
		}
	}

	iceServer := &iceServers[0]

	////////////////////////////////////////////////////////////////////////////////////////////////////

	// yandexMailManager, err := yandex.NewYandexMailService(os.Getenv("DEVELOP_YANDEX_MAIL_USER"), os.Getenv("DEVELOP_YANDEX_MAIL_PASSWORD"), cipherKey)
	// go yandexMailManager.Listen()

	// time.Sleep(3 * time.Second)

	// go func() {
	// 	for msg := range yandexMailManager.Mails {
	// 		slog.Info("yandexMailManager.Mails", string(msg))
	// 	}
	// }()

	// time.Sleep(3 * time.Second)

	// for range 2 {
	// 	yandexMailManager.SendMail([]byte(time.Now().Format(time.RFC3339)))

	// 	time.Sleep(30 * time.Second)
	// }

	////////////////////////////////////////////////////////////////////////////////////////////////////

	tr1 := transport.NewWebRTCTransport()
	tr2 := transport.NewWebRTCTransport()

	offerBuf, err := tr1.CreateOffer(iceServer)
	if err != nil {
		panic(err)
	}

	// offerBufMsg, err := GZipCipherBase64Encode(cipher, offerBuf)
	// if err != nil {
	// 	panic(err)
	// }

	// slog.Debug(fmt.Sprintf("GZipCipherBase64Encode: %d / %d bytes", len(offerBufMsg), len(offerBuf)))

	// offerBuf, err = GZipCipherBase64Decode(cipher, offerBufMsg)
	// if err != nil {
	// 	panic(err)
	// }

	answerBuf, err := tr2.CreateAnswer(offerBuf)
	if err != nil {
		panic(err)
	}

	err = tr1.SetAnswer(answerBuf)
	if err != nil {
		panic(err)
	}

	// var wg sync.WaitGroup
	// wg.Add(2)

	// go func() {
	// 	defer wg.Done()

	// 	connA, err := tr1.Transport()
	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	connA.Write([]byte(cipherKey))
	// 	slog.Debug(fmt.Sprintf("connA.Write: %s", cipherKey))
	// }()

	// var connB net.Conn
	// go func() {
	// 	defer wg.Done()

	// 	connB, err = tr2.Transport()
	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	buf := make([]byte, 32)
	// 	for {
	// 		n, err := connB.Read(buf)
	// 		if err != nil {
	// 			return
	// 		}
	// 		slog.Debug(fmt.Sprintf("connB.Read: %s", buf[:n]))
	// 		connB.Close()
	// 	}
	// }()

	// wg.Wait()

	// connB.Close()

	// Client

	const nodeTcpServerConnPort = 8080

	// clientTransport := transport.NewTcpClientTransport(fmt.Sprintf("localhost:%d", nodeTcpServerConnPort))
	clientTransport := tr1
	clientNode := ntun.NewNode(&conf.Config{Id: clientId, Name: "client", Allowed: []uuid.UUID{serverId}, CipherKey: cipherKey}, clientTransport)
	slog.Info(fmt.Sprintf("Client node: %s", clientNode.String()))
	clientNode.Start()

	const proxyServerPort = 8081
	sock5Server := inputs.NewSock5NoAuthServer(clientNode.ConnManager)
	err = sock5Server.ListenAndServe(proxyServerPort)
	if err != nil {
		panic(err)
	}

	// Server

	// serverTransport := transport.NewTcpServerTransport(nodeTcpServerConnPort)
	// err = serverTransport.Listen()
	// if err != nil {
	// 	panic(err)
	// }
	serverTransport := tr2

	serverNode := ntun.NewNode(&conf.Config{Id: serverId, Name: "server", Allowed: []uuid.UUID{clientId}, CipherKey: cipherKey}, serverTransport)
	slog.Info(fmt.Sprintf("Server node: %s", serverNode.String()))
	serverNode.Start()

	// Test

	time.Sleep(3 * time.Second)

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

	result, err = requester.Post(simpleHttpsEchoServerRequestUrl, testStr)
	if err != nil {
		panic(err)
	}

	if result != testStr {
		panic(fmt.Sprintf("result != testStr %s %s", result, testStr))
	}

	ip, err := requester.Get(os.Getenv("DEVELOP_GET_PUBLIC_IP_HTTP_URL"))
	if err != nil {
		panic(err)
	}

	slog.Info(fmt.Sprintf("Public IP %s", ip))

	ipHttps, err := requester.Get(os.Getenv("DEVELOP_GET_PUBLIC_IP_HTTPS_URL"))
	if err != nil {
		panic(err)
	}

	if ip != ipHttps {
		panic(fmt.Sprintf("ip != ipHttps %s %s", ip, ipHttps))
	}

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

	sock5Server.Close()
}
