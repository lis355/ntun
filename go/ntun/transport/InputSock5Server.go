package transport

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
	"ntun/ntun"
	"strconv"
	"sync"
)

const (
	version            byte = 5
	methodAuthNone     byte = 0    // 0 = No Authentication Required
	methodNoAcceptable byte = 0xFF // 0xFF = No Acceptable Methods
)

type DialFunc func(ctx context.Context, srcAdress, dstAddress string) (net.Conn, error)

type InputSock5Server struct {
	port uint16
	dial DialFunc

	ctx    context.Context
	cancel context.CancelFunc

	listener net.Listener
	mu       sync.Mutex

	running bool
}

func NewInputSock5Server(port uint16, dial DialFunc) (c *InputSock5Server) {
	return &InputSock5Server{
		port: port,
		dial: dial,
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

		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", c.port))
		if err != nil {
			return err
		}

		c.listener = listener

		slog.Info(fmt.Sprintf("[Socks5Server]: listening on http://localhost:%d", c.port))

		go c.serve()

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

func (c *InputSock5Server) serve() {
	for {
		conn, err := c.listener.Accept()
		select {
		case <-c.ctx.Done():
			return
		default:
			if err != nil {
				slog.Error(fmt.Sprintf("[InputSock5Server] accept error: %v", err))
				return
			}

			go c.handleConn(conn)
		}
	}
}

func (c *InputSock5Server) handleConn(srcConn net.Conn) {
	ctx, cancel := context.WithCancel(c.ctx)
	defer cancel()

	go func() {
		<-ctx.Done()
		srcConn.Close()
	}()

	defer srcConn.Close()

	address, err := c.handshakeNoAuth(srcConn)
	if err != nil {
		slog.Error(fmt.Sprintf("[InputSock5Server] handshake error: %v", err))

		return
	}

	slog.Debug(fmt.Sprintf("[InputSock5Server] accept connection from %s, wants to connect to %s", srcConn.RemoteAddr(), address))

	// Connect to target
	destConn, err := c.dial(c.ctx, srcConn.RemoteAddr().String(), address)
	if err != nil {
		// Reply: REP=1 (General failure) [VER, REP, RSV, ATYP, BND.ADDR, BND.PORT]
		srcConn.Write([]byte{version, 1, 0, 1, 0, 0, 0, 0, 0, 0})
		return
	}

	defer destConn.Close()

	// Reply: REP=0 (Success), ATYP=1(tcp), BND.ADDR=0.0.0.0, BND.PORT=0
	if _, err := srcConn.Write([]byte{version, 0, 0, 1, 0, 0, 0, 0, 0, 0}); err != nil {
		return
	}

	slog.Debug(fmt.Sprintf("[InputSock5Server] connected to %s (%s)", address, destConn.RemoteAddr()))

	// srcConn = NewSniffConn(fmt.Sprintf("input socks5 %s-%s", srcConn.RemoteAddr(), address), srcConn)

	ntun.Proxy(srcConn, destConn)
}

func (c *InputSock5Server) handshakeNoAuth(srcConn net.Conn) (string, error) {
	buf := make([]byte, 255) // Max 255 methods

	// Handshake (RFC 1928, Sec 3)
	// Header
	// [VER, NMETHODS] 2 bytes
	if _, err := io.ReadFull(srcConn, buf[:2]); err != nil {
		return "", err
	}

	vrs := buf[0]
	if vrs != version {
		return "", fmt.Errorf("Unsupported version %d", version)
	}

	methodsLen := buf[1]
	methodsBuf := buf[:methodsLen]
	if _, err := io.ReadFull(srcConn, methodsBuf); err != nil {
		return "", err
	}

	if bytes.IndexByte(methodsBuf, methodAuthNone) == -1 {
		if _, err := srcConn.Write([]byte{version, methodNoAcceptable}); err != nil {
			return "", err
		}

		return "", fmt.Errorf("No acceptable method")
	}

	if _, err := srcConn.Write([]byte{version, methodAuthNone}); err != nil {
		return "", err
	}

	// Request (RFC 1928, Sec 4)
	// [VER, CMD, RSV, ATYP]
	if _, err := io.ReadFull(srcConn, buf[:4]); err != nil {
		return "", err
	}

	cmd := buf[1]
	rsv := buf[2]
	// buf[1]=1 (CONNECT command)
	// buf[2]=0 (Reserved)
	if cmd != 1 &&
		rsv != 0 {
		return "", fmt.Errorf("Unsupported command %d", cmd)
	}

	var host string

	// ATYP (Address Type)
	atyp := buf[3]

	switch atyp {
	case 1: // IPv4 (4 bytes)
		ip := make([]byte, 4)
		if _, err := io.ReadFull(srcConn, ip); err != nil {
			return "", err
		}
		host = net.IP(ip).String()
	case 3: // Domain Name (First byte is length)
		if _, err := io.ReadFull(srcConn, buf[:1]); err != nil {
			return "", err
		}
		sz := byte(buf[0])
		if _, err := io.ReadFull(srcConn, buf[:sz]); err != nil {
			return "", err
		}
		host = string(buf[:sz])
	case 4: // IPv6 (16 bytes)
		ip := make([]byte, 16)
		if _, err := io.ReadFull(srcConn, ip); err != nil {
			return "", err
		}
		host = net.IP(ip).String()
	default:
		return "", fmt.Errorf("Unsupported address type %d", atyp)
	}

	// Port (2 bytes, BigEndian)
	if _, err := io.ReadFull(srcConn, buf[:2]); err != nil {
		return "", err
	}

	port := strconv.FormatUint(uint64(binary.BigEndian.Uint16(buf[:2])), 10)

	return net.JoinHostPort(host, port), nil
}
