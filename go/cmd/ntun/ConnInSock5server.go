package ntun

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/armon/go-socks5"
)

type PipeTCPConn struct {
	net.Conn
	remoteAddr *net.TCPAddr
	localAddr  *net.TCPAddr
}

func (c *PipeTCPConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *PipeTCPConn) LocalAddr() net.Addr {
	return c.localAddr
}

func CreateAndListenSocks5Server(port int, connId ConnIn, ready chan struct{}) {
	socks5ProxyAddress := fmt.Sprintf("socks5://localhost:%d", port)

	conf := &socks5.Config{
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			slog.Debug(fmt.Sprintf("[Socks5Server]: Connection with %s via [%s]", address, socks5ProxyAddress))

			conn, err := connId.HandleInputConn(address)
			if err != nil {
				return nil, err
			}

			// библиотека armon/go-socks5 зачем то хочет узнать net.TCPAddr
			// кастует target.LocalAddr().(*net.TCPAddr) и все ломается, поэтому сделаем мок для LocalAddr()
			conn = &PipeTCPConn{
				Conn:       conn,
				localAddr:  &net.TCPAddr{IP: net.IPv4zero, Port: 0},
				remoteAddr: &net.TCPAddr{IP: net.IPv4zero, Port: 0},
			}

			return conn, nil
		},
	}

	server, err := socks5.New(conf)
	if err != nil {
		panic(err)
	}

	go func() {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			panic(err)
		}

		slog.Info(fmt.Sprintf("[Socks5Server]: listening on http://localhost:%d", port))

		close(ready)

		server.Serve(listener)
	}()
}
