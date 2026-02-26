package ntun

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"
)

const defaultDataBufferSize = 4 * Kilobyte

type TcpServerConn struct {
	port     int
	listener net.Listener
	quit     chan struct{}
}

func NewTcpServerConn(port int) (c *TcpServerConn) {
	return &TcpServerConn{
		port:     port,
		listener: nil,
		quit:     make(chan struct{}),
	}
}

func (c *TcpServerConn) Start() error {
	slog.Debug("[TcpServerConn] starting")
	defer slog.Debug("[TcpServerConn] started")

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", c.port))
	if err != nil {
		return err
	}

	c.listener = listener

	go func() {
		for {
			conn, err := c.listener.Accept()
			if err != nil {
				select {
				case <-c.quit:
					c.listener = nil
					return
				default:
					slog.Error(err.Error())
					continue
				}
			}

			go (func(conn net.Conn) {
				defer conn.Close()

				reader := bufio.NewReader(conn)

				message, err := reader.ReadString('\n')
				if err != nil {
					log.Printf("Read error: %v", err)
					return
				}

				ackMsg := strings.ToUpper(strings.TrimSpace(message))
				response := fmt.Sprintf("ACK: %s\n", ackMsg)
				_, err = conn.Write([]byte(response))
				if err != nil {
					log.Printf("Server write error: %v", err)
				}
			})(conn)
		}
	}()

	return nil
}

func (c *TcpServerConn) Stop() error {
	slog.Debug("[TcpServerConn] stopping")
	defer slog.Debug("[TcpServerConn] stopped")

	close(c.quit)

	if c.listener != nil {
		return c.listener.Close()
	}

	return nil
}

const TcpClientDialTimeout = 10 * time.Second
const TcpClientReconnectTimeout = 1 * time.Second

type TcpClientConn struct {
	address string

	ctx     context.Context
	cancel  context.CancelFunc
	conn    net.Conn
	connMu  sync.Mutex
	running bool

	dialer net.Dialer
}

func NewTcpClientConn(address string) (c *TcpClientConn) {
	return &TcpClientConn{
		address: address,
		dialer: net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 1 * time.Hour,
		},
	}
}

func (c *TcpClientConn) Start() error {
	slog.Debug("[TcpClientConn] starting")
	defer slog.Debug("[TcpClientConn] started")

	c.connMu.Lock()
	err := func() error {
		if c.running {
			return fmt.Errorf("Connection already started")
		}

		c.ctx, c.cancel = context.WithCancel(context.Background())

		go c.reconnect()
		c.running = true

		return nil
	}()
	c.connMu.Unlock()

	return err
}

func (c *TcpClientConn) Stop() error {
	slog.Debug("[TcpClientConn] stopping")
	defer slog.Debug("[TcpClientConn] stopped")

	c.connMu.Lock()
	err := func() error {
		if !c.running {
			return fmt.Errorf("Connection already stopped")
		}

		c.cancel()
		c.conn.Close()
		c.running = false

		return nil
	}()
	c.connMu.Unlock()

	return err
}

func (c *TcpClientConn) reconnect() {
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

		select {
		case <-c.ctx.Done():
			return
		default:
			continue
		}
	}
}

func (c *TcpClientConn) dial() (net.Conn, error) {
	ctx, cancel := context.WithTimeout(c.ctx, TcpClientDialTimeout)
	defer cancel()

	return c.dialer.DialContext(ctx, "tcp", c.address)
}

func (c *TcpClientConn) processConnection() {
	slog.Debug(fmt.Sprintf("[TcpClientConn] connected successfull to %s", c.address))

	defer c.conn.Close()

	bytes := make([]byte, defaultDataBufferSize)

	for {
		// DEBUG
		c.conn.Write([]byte(time.Now().Format(time.RFC3339)))
		time.Sleep(time.Second)

		n, err := c.conn.Read(bytes)
		select {
		case <-c.ctx.Done():
			return
		default:
			if err != nil {
				slog.Error(err.Error())

				return
			}

			buffer := bytes[:n]

			slog.Debug(fmt.Sprintf("[TcpClientConn] read %d bytes %s", len(buffer), buffer))
		}
	}
}

func (c *TcpClientConn) HandleInputConn(address string) (net.Conn, error) {
	// TODO check conn transport is connected

	clientSide, muxSide := net.Pipe()
	_ = muxSide

	// streamID := m.nextID()
	// go m.handleVirtualStream(streamID, address, muxSide)

	// go func() {
	// 	for {
	// 		n, err := muxSide.Read(c.bytes)
	// 		if err != nil {
	// 			return
	// 		}

	// 		slog.Debug(fmt.Sprintf("muxSide read %d", n))
	// 	}
	// }()

	return clientSide, nil
}
