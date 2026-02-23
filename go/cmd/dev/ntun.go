package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"ntun/cmd/app"
	"ntun/utils/log"
	"strings"
	"time"

	"github.com/armon/go-socks5"
	"golang.org/x/net/proxy"
)

func request(proxyAddress, url string) string {
	dialer, err := proxy.SOCKS5("tcp", proxyAddress, nil, proxy.Direct)
	if err != nil {
		panic(err)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			Dial: dialer.Dial,
		},
		Timeout: 10 * time.Second,
	}

	slog.Debug(fmt.Sprintf("[%s] %s", proxyAddress, url))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("User-Agent", "curl")

	resp, err := httpClient.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	bodyStr := string(body)

	slog.Debug(fmt.Sprintf("[%s] %s %s %s", proxyAddress, url, resp.Status, bodyStr))

	return bodyStr
}

const needPrintHexDump = false

type observableConn struct {
	net.Conn
}

func newObservableConn(conn net.Conn) *observableConn {
	return &observableConn{Conn: conn}
}

func (c *observableConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	if n > 0 {
		slog.Debug(fmt.Sprintf("[%s <-- %s] read %d bytes", c.Conn.LocalAddr(), c.Conn.RemoteAddr(), n))
		if needPrintHexDump {
			printHexDump(b[:n])
		}
	}
	return n, err
}

func (c *observableConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	if n > 0 {
		slog.Debug(fmt.Sprintf("[%s --> %s] write %d bytes", c.Conn.LocalAddr(), c.Conn.RemoteAddr(), n))
		if needPrintHexDump {
			printHexDump(b[:n])
		}
	}
	return n, err
}

func printHexDump(data []byte) {
	for i := 0; i < len(data); i += 32 {
		fmt.Printf("%08X  ", i)

		ascii := strings.Builder{}
		ascii.WriteString(" |")

		for j := 0; j < 32; j++ {
			if i+j < len(data) {
				fmt.Printf("%02X ", data[i+j])

				b := data[i+j]
				if b >= 32 && b <= 126 {
					ascii.WriteByte(b)
				} else {
					ascii.WriteByte('.')
				}
			} else {
				fmt.Print("   ")
				ascii.WriteByte(' ')
			}

			if (j+1)%8 == 0 {
				fmt.Print(" ")
			}
		}

		fmt.Printf(" %s\n", ascii.String())
	}
}

func createAndListenSocks5Server(proxyServerPort int, socks5ServerReady chan bool) {
	socks5ProxyAddress := fmt.Sprintf("socks5://localhost:%d", proxyServerPort)

	conf := &socks5.Config{
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			slog.Debug(fmt.Sprintf("Connection with %s via [%s]", address, socks5ProxyAddress))

			conn, err := net.Dial(network, address)
			if err != nil {
				return nil, err
			}

			if tcpConn, ok := conn.(*net.TCPConn); ok {
				conn = newObservableConn(tcpConn)
			} else {
				panic(fmt.Errorf("Strange conn"))
			}

			return conn, nil
		},
	}

	server, err := socks5.New(conf)
	if err != nil {
		panic(err)
	}

	socks5ServerReady <- true

	if err := server.ListenAndServe("tcp", fmt.Sprintf(":%d", proxyServerPort)); err != nil {
		panic(err)
	}
}

func main() {
	app.Initialize()
	log.Initialize()

	slog.Info("DEVELOPMENT")

	slog.Info(fmt.Sprintf("%s v%s", app.Name, app.Version))
	if app.IsDevelopment {
		slog.Warn("Development mode")
	}

	const proxyServerPort = 8080

	socks5ServerReady := make(chan bool, 1)
	go createAndListenSocks5Server(proxyServerPort, socks5ServerReady)
	<-socks5ServerReady

	socks5ProxyAddress := fmt.Sprintf("localhost:%d", proxyServerPort)

	request(socks5ProxyAddress, "http://ifconfig.me/ip")
	request(socks5ProxyAddress, "https://ifconfig.me/ip")
}
