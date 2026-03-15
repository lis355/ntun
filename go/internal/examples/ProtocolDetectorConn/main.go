package main

import (
	"fmt"
	"log/slog"
	"net"
	"ntun/internal/app"
	"ntun/internal/connections"
	"ntun/internal/connections/inputs"
	"ntun/internal/dev"
	"ntun/internal/log"
	"os"
)

type Dialer struct{}

func (d *Dialer) Dial(dstAddress string) (net.Conn, error) {
	dstConn, err := net.Dial("tcp", dstAddress)
	// dstConn = dev.NewSnifferHexDumpDebugConn(dstConn, fmt.Sprintf("%s <--> %s", srcAddress, dstAddress), true)
	protocolDetectorConn := connections.NewProtocolDetectorConn(dstConn)
	dstConn = protocolDetectorConn

	go func() {
		protocol := <-protocolDetectorConn.Detected
		switch pr := protocol.(type) {
		case *connections.HttpProtocol:
			slog.Info(fmt.Sprintf("detected %s protocol", pr.Protocol()))
		case *connections.HttpsProtocol:
			slog.Info(fmt.Sprintf("detected %s protocol %s", pr.Protocol(), pr.Domain))
		}
	}()

	return dstConn, err
}

func main() {
	app.Init()
	log.Init()

	const proxyServerPort = 8082
	sock5Server := inputs.NewSock5NoAuthServer(&Dialer{})
	err := sock5Server.ListenAndServe(proxyServerPort)
	if err != nil {
		panic(err)
	}

	socks5ProxyAddress := fmt.Sprintf("localhost:%d", proxyServerPort)

	requester, err := dev.NewRequester(socks5ProxyAddress)
	if err != nil {
		panic(err)
	}

	_, err = requester.Get(os.Getenv("DEVELOP_GET_PUBLIC_IP_HTTP_URL"))
	if err != nil {
		panic(err)
	}

	_, err = requester.Get(os.Getenv("DEVELOP_GET_PUBLIC_IP_HTTPS_URL"))
	if err != nil {
		panic(err)
	}

	sock5Server.Close()
}
