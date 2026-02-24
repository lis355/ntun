package ntun

import (
	"bufio"
	"fmt"
	"log"
	"log/slog"
	"net"
	"strings"
)

const defaultDataBufferSize = 4096

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
	close(c.quit)

	if c.listener != nil {
		return c.listener.Close()
	}

	return nil
}

type TcpClientConn struct {
	address string
	conn    net.Conn
	quit    chan struct{}
	bytes   []byte
}

func NewTcpClientConn(address string) (c *TcpClientConn) {
	return &TcpClientConn{
		address: address,
		quit:    make(chan struct{}),
		bytes:   make([]byte, defaultDataBufferSize),
	}
}

func (c *TcpClientConn) Start() error {
	if c.conn != nil {
		return fmt.Errorf("Connection already started")
	}

	conn, err := net.Dial("tcp", c.address)
	if err != nil {
		return err
	}

	c.conn = conn

	go func() {
		for {
			n, err := conn.Read(c.bytes)
			select {
			case <-c.quit:
				c.conn = nil
				return
			default:
				if n > 0 {

				}

				if err != nil {
					slog.Error(err.Error())

					conn.Close()
					return
				}
			}
		}
	}()

	return nil
}

func (c *TcpClientConn) Stop() error {
	if c.conn == nil {
		return fmt.Errorf("Connection already stopped")
	}

	close(c.quit)

	c.conn.Close()
	c.conn = nil

	return nil
}

func (c *TcpClientConn) HandleInputConn(conn net.Conn) error {
	return nil
}
