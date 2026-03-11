package transport

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"ntun/ntun"
	"sync"
	"time"
)

const defaultDataBufferSize = 4 * ntun.Kilobyte

type TcpServerTransport struct {
	port int

	ctx    context.Context
	cancel context.CancelFunc

	listener net.Listener
	conn     net.Conn // только 1 активное соединиение, других отклоняем
	connMu   sync.Mutex
	connChan chan net.Conn

	running bool
}

func NewTcpServerTransport(port int) (c *TcpServerTransport) {
	return &TcpServerTransport{
		port: port,
	}
}

func (c *TcpServerTransport) Transport() <-chan net.Conn {
	return c.connChan
}

func (c *TcpServerTransport) Start() error {
	slog.Debug("[TcpServerConn] starting")
	defer slog.Debug("[TcpServerConn] started")

	c.connMu.Lock()
	err := func() error {
		if c.running {
			return fmt.Errorf("[TcpServerConn] already started")
		}

		c.ctx, c.cancel = context.WithCancel(context.Background())

		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", c.port))
		if err != nil {
			return err
		}

		c.listener = listener

		go c.listen()

		c.running = true

		return nil
	}()
	c.connMu.Unlock()

	return err
}

func (c *TcpServerTransport) Stop() error {
	slog.Debug("[TcpServerConn] stopping")
	defer slog.Debug("[TcpServerConn] stopped")

	c.connMu.Lock()
	err := func() error {
		if !c.running {
			return fmt.Errorf("[TcpServerConn] already stopped")
		}

		c.cancel()

		c.listener.Close()
		if c.conn != nil {
			c.conn.Close()
		}

		c.running = false

		return nil
	}()
	c.connMu.Unlock()

	return err
}

func (c *TcpServerTransport) listen() {
	for {
		conn, err := c.listener.Accept()
		if err != nil {
			select {
			case <-c.ctx.Done():
				return
			default:
				if err != io.EOF {
					slog.Error(err.Error())
				}

				continue
			}
		}

		hasActiveConn := false

		c.connMu.Lock()
		if c.conn != nil {
			hasActiveConn = true
		} else {
			c.conn = conn
		}
		c.connMu.Unlock()

		if hasActiveConn {
			conn.Close()
			continue
		}

		c.processConnection()

		c.connMu.Lock()
		c.conn = nil
		c.connMu.Unlock()
	}
}

func (c *TcpServerTransport) processConnection() {
	defer c.conn.Close()

	// DEBUG
	io.Copy(c.conn, c.conn)
}

const TcpClientDialTimeout = 10 * time.Second
const TcpClientReconnectTimeout = 1 * time.Second

type TcpClientTransport struct {
	address string

	ctx    context.Context
	cancel context.CancelFunc

	conn     net.Conn
	connMu   sync.Mutex
	connChan chan net.Conn

	running bool

	dialer net.Dialer
}

func NewTcpClientTransport(address string) (c *TcpClientTransport) {
	return &TcpClientTransport{
		address: address,
		dialer: net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 1 * time.Hour,
		},
	}
}

func (c *TcpClientTransport) Transport() <-chan net.Conn {
	return c.connChan
}

func (c *TcpClientTransport) Start() error {
	slog.Debug("[TcpClientConn] starting")
	defer slog.Debug("[TcpClientConn] started")

	c.connMu.Lock()
	err := func() error {
		if c.running {
			return fmt.Errorf("[TcpClientConn] already started")
		}

		c.ctx, c.cancel = context.WithCancel(context.Background())

		go c.reconnect()
		c.running = true

		return nil
	}()
	c.connMu.Unlock()

	return err
}

func (c *TcpClientTransport) Stop() error {
	slog.Debug("[TcpClientConn] stopping")
	defer slog.Debug("[TcpClientConn] stopped")

	c.connMu.Lock()
	err := func() error {
		if !c.running {
			return fmt.Errorf("[TcpClientConn] already stopped")
		}

		c.cancel()
		c.conn.Close()
		c.running = false

		return nil
	}()
	c.connMu.Unlock()

	return err
}

func (c *TcpClientTransport) reconnect() {
	for {
		slog.Debug(fmt.Sprintf("[TcpClientConn] trying to connect to %s", c.address))

		conn, err := c.dial()
		if err != nil {
			slog.Debug("[TcpClientConn] connect failed, waiting")

			select {
			case <-c.ctx.Done():
				return
			case <-time.After(TcpClientReconnectTimeout):
				continue
			}
		}

		c.connMu.Lock()
		c.conn = conn
		c.connMu.Unlock()

		c.processConnection()

		c.connMu.Lock()
		c.conn = nil
		c.connMu.Unlock()

		select {
		case <-c.ctx.Done():
			return
		default:
			continue
		}
	}
}

func (c *TcpClientTransport) dial() (net.Conn, error) {
	ctx, cancel := context.WithTimeout(c.ctx, TcpClientDialTimeout)
	defer cancel()

	return c.dialer.DialContext(ctx, "tcp", c.address)
}

func (c *TcpClientTransport) processConnection() {
	slog.Debug(fmt.Sprintf("[TcpClientConn] connected successfull to %s", c.address))

	defer c.conn.Close()

	c.connChan <- c.conn
}

func (c *TcpClientTransport) Dial(ctx context.Context, address string) (net.Conn, error) {
	c.connMu.Lock()
	conn := c.conn
	c.connMu.Unlock()

	if conn == nil {
		return nil, fmt.Errorf("[TcpClientConn] not connected")
	}

	clientSide, muxSide := net.Pipe()
	_ = muxSide

	// streamID := m.nextID()
	// go m.handleVirtualStream(streamID, address, muxSide)

	wbuf := make([]byte, 1024)
	copy(wbuf, []byte(address)) // Копирует байты из s в начало buf
	c.conn.Write(wbuf)

	go func() {
		bytes := make([]byte, defaultDataBufferSize)

		for {
			n, err := muxSide.Read(bytes)
			if err != nil {
				return
			}

			// вначале соединения всегда читается 0, мб это прикол либы github.com/armon/go-socks5 ?
			if n == 0 {
				continue
			}

			slog.Debug(fmt.Sprintf("muxSide read %d", n))
		}
	}()

	return clientSide, nil
}
