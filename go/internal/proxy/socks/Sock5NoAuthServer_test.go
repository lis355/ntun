package socks

import (
	"encoding/hex"
	"fmt"
	"ntun/internal/dev"
	"ntun/internal/ntun/connections/outputs"
	"ntun/internal/utils"
	"testing"
)

func TestSock5NoAuthServer(t *testing.T) {
	const simpleHttpEchoServerPort = 8081
	var simpleHttpTimeServerRequestUrl = fmt.Sprintf("http://localhost:%d", simpleHttpEchoServerPort)
	simpleHttpEchoServer := dev.NewSimpleHttpEchoServer()
	simpleHttpEchoServer.ListenAndServe(simpleHttpEchoServerPort)

	sock5Server := NewSock5NoAuthServer(outputs.NewDirectOutput())

	const proxyServerPort = 8082
	socks5ProxyAddress := fmt.Sprintf("localhost:%d", proxyServerPort)

	err := sock5Server.Listen(socks5ProxyAddress)
	if err != nil {
		t.Fatal(err)
	}

	requester, err := dev.NewRequester(socks5ProxyAddress)
	if err != nil {
		t.Fatal(err)
	}

	testStr := hex.EncodeToString(utils.RandBytes(8))
	result, err := requester.Post(simpleHttpTimeServerRequestUrl, testStr)
	if err != nil {
		t.Fatal(err)
	}

	if result != testStr {
		t.Fatalf("result != testStr %s %s", result, testStr)
	}

	sock5Server.Close()
	simpleHttpEchoServer.Close()
}
