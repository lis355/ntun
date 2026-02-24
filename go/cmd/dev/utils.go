package main

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"

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
	m sync.Mutex // for sync printing tcp hex data
}

func newObservableConn(conn net.Conn) *observableConn {
	return &observableConn{Conn: conn}
}

func (c *observableConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)

	if n > 0 {
		slog.Debug(fmt.Sprintf("[%s <-- %s] read %d bytes", c.Conn.LocalAddr(), c.Conn.RemoteAddr(), n))
		if needPrintHexDump {
			c.m.Lock()
			printHexDump(b[:n])
			c.m.Unlock()
		}
	}

	return n, err
}

func (c *observableConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)

	if n > 0 {
		slog.Debug(fmt.Sprintf("[%s --> %s] write %d bytes", c.Conn.LocalAddr(), c.Conn.RemoteAddr(), n))
		if needPrintHexDump {
			c.m.Lock()
			printHexDump(b[:n])
			c.m.Unlock()
		}
	}

	return n, err
}

func printHexDump(data []byte) {
	// fmt.Printf("%s", strings.Repeat(" ", 10))
	// for i := range 32 {
	// 	fmt.Printf("%02X ", i)

	// 	if (i+1)%8 == 0 {
	// 		fmt.Print(" ")
	// 	}
	// }
	// fmt.Print("\n")

	// fmt.Printf("%s", strings.Repeat(" ", 10))
	// for i := range 32 {
	// 	fmt.Printf("--")

	// 	if i < 31 {
	// 		fmt.Print("-")
	// 	}

	// 	if (i+1)%8 == 0 &&
	// 		i+1 != 32 {
	// 		fmt.Print("-")
	// 	}
	// }
	// fmt.Print("\n")

	for i := 0; i < len(data); i += 32 {
		fmt.Printf("%08X| ", i)

		ascii := strings.Builder{}
		ascii.WriteString("|")

		for j := 0; j < 32; j++ {
			if i+j < len(data) {
				fmt.Printf("%02X ", data[i+j])

				b := data[i+j]
				if b >= 32 &&
					b <= 126 {
					ascii.WriteByte(b)
				} else {
					ascii.WriteByte('.')
				}
			} else {
				fmt.Print("   ")
				ascii.WriteByte(' ')
			}

			if (j+1)%8 == 0 &&
				j+1 != 32 {
				fmt.Print(" ")
			}
		}

		fmt.Print(ascii.String())
		fmt.Print("\n")
	}
}
