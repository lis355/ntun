package ntun

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/armon/go-socks5"
)

type ConnWrap struct {
	net.Conn
	remoteAddr *net.TCPAddr
	localAddr  *net.TCPAddr
}

func (c *ConnWrap) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *ConnWrap) LocalAddr() net.Addr {
	return c.localAddr
}

func CreateAndListenSocks5Server(port int, dialer Dialer, ready chan struct{}) {

}

type InputSock5Server struct {
	port   uint16
	dialer Dialer

	ctx    context.Context
	cancel context.CancelFunc

	listener net.Listener
	server   *socks5.Server
	mu       sync.Mutex

	running bool
}

func NewInputSock5Server(port uint16, dialer Dialer) (c *InputSock5Server) {
	return &InputSock5Server{
		port: port,
	}
}

func (c *InputSock5Server) Start() error {
	slog.Debug("[InputSock5Server] starting")
	defer slog.Debug("[InputSock5Server] started")

	c.mu.Lock()
	err := func() error {
		if c.running {
			return fmt.Errorf("[InputSock5Server] already started")
		}

		c.ctx, c.cancel = context.WithCancel(context.Background())

		socks5ProxyAddress := fmt.Sprintf("socks5://localhost:%d", c.port)

		conf := &socks5.Config{
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				if network != "tcp" {
					return nil, fmt.Errorf("[InputSock5Server] only tcp network is supported")
				}

				slog.Debug(fmt.Sprintf("[InputSock5Server]: connection with %s via [%s]", address, socks5ProxyAddress))

				conn, err := c.dialer.Dial(ctx, address)
				if err != nil {
					return nil, err
				}

				// библиотека armon/go-socks5 зачем то хочет узнать net.TCPAddr
				// кастует target.LocalAddr().(*net.TCPAddr), использует LocalAddr() и все ломается, поэтому сделаем хак для работы
				if _, ok := conn.(*net.TCPConn); !ok {
					conn = &ConnWrap{
						Conn:       conn,
						localAddr:  &net.TCPAddr{IP: net.IPv4zero, Port: 0},
						remoteAddr: &net.TCPAddr{IP: net.IPv4zero, Port: 0},
					}
				}

				return conn, nil
			},
		}

		server, err := socks5.New(conf)
		if err != nil {
			return err
		}

		c.server = server

		c.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", c.port))
		if err != nil {
			return err
		}

		slog.Info(fmt.Sprintf("[Socks5Server]: listening on http://localhost:%d", c.port))

		go c.server.Serve(c.listener)

		c.running = true

		return nil
	}()
	c.mu.Unlock()

	return err
}

func (c *InputSock5Server) Stop() error {
	slog.Debug("[InputSock5Server] stopping")
	defer slog.Debug("[InputSock5Server] stopped")

	c.mu.Lock()
	err := func() error {
		if !c.running {
			return fmt.Errorf("[InputSock5Server] already stopped")
		}

		c.cancel()

		c.listener.Close()

		c.running = false

		return nil
	}()
	c.mu.Unlock()

	return err
}
