package main

import (
	"fmt"
	"log/slog"
	"net"
	"ntun/internal/app"
	"ntun/internal/dev"
	"ntun/internal/log"
	"ntun/ntun/connections/inputs"
	"os"
	"strings"

	"github.com/open-ch/ja3"
)

type ProtocolDetectorConn struct {
	net.Conn
	active bool
	buf    []byte
}

func NewProtocolMonitorConn(conn net.Conn) *ProtocolDetectorConn {
	return &ProtocolDetectorConn{
		Conn:   conn,
		active: true,
	}
}

func (c *ProtocolDetectorConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	if c.active &&
		n > 0 {
		c.process(b[:n])
	}
	return n, err
}

var httpMethods = []string{
	"GET",
	"HEAD",
	"POST",
	"PUT",
	"DELETE",
	"CONNECT",
	"OPTIONS",
	"TRACE",
	"PATCH",
}

func (c *ProtocolDetectorConn) process(b []byte) {
	// будем проверять буфер с начала каждый раз
	// так мы защищены от сегментации tcp пакета
	// главное задачь четкие границы остановки парсинга

	c.buf = append(c.buf, b...)

	if len(c.buf) >= 2 &&
		c.buf[0] == 0x16 &&
		c.buf[1] == 0x03 {
		j, err := ja3.ComputeJA3FromSegment(c.buf)
		if err != nil {
			return
		}

		sni := j.GetSNI()
		if sni != "" {
			slog.Info(fmt.Sprintf("detected HTTPS %s", sni))
		}
		c.stop()
	} else if len(c.buf) >= 7 {
		// переводим всё в верхний регистр чтобы учесть варианты типа PoSt
		prefix := strings.ToUpper(string(c.buf[:7]))
		for _, method := range httpMethods {
			if strings.HasPrefix(prefix, method) {
				slog.Info(fmt.Sprintf("detected HTTP %s", method))
				c.stop()
				break
			}
		}
	} else if len(c.buf) >= 1024 {
		// ничего не нашли, хватит парсить
		c.stop()
	}
}

func (c *ProtocolDetectorConn) stop() {
	c.active = false
	c.buf = nil
}

func dial(srcAddress, dstAddress string) (net.Conn, error) {
	dstConn, err := net.Dial("tcp", dstAddress)
	// dstConn = dev.NewSnifferHexDumpDebugConn(dstConn, fmt.Sprintf("%s <--> %s", srcAddress, dstAddress), true)
	dstConn = NewProtocolMonitorConn(dstConn)

	return dstConn, err
}

func main() {
	app.InitEnv()
	log.Init()

	const proxyServerPort = 8082
	sock5Server := inputs.NewSock5NoAuthServer(dial)
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
