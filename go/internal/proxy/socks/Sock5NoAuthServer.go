package socks

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"ntun/internal/log"
	"ntun/internal/ntun/connections"
	"ntun/internal/proxy"
	"strconv"
)

const (
	version            byte = 5
	methodAuthNone     byte = 0    // 0 = No Authentication Required
	methodNoAcceptable byte = 0xFF // 0xFF = No Acceptable Methods
)

type Sock5NoAuthServer struct {
	dialer   connections.Dialer
	listener net.Listener
}

func NewSock5NoAuthServer(dialer connections.Dialer) (c *Sock5NoAuthServer) {
	return &Sock5NoAuthServer{
		dialer: dialer,
	}
}

func (s *Sock5NoAuthServer) Listen(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	s.listener = listener

	slog.Info(fmt.Sprintf("%s: listening on socks5://%s", log.ObjName(s), address))

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}

				slog.Error(fmt.Sprintf("%s: accept error: %v", log.ObjName(s), err))

				continue
			}

			go s.handleConn(conn)
		}
	}()

	return nil
}

func (s *Sock5NoAuthServer) Close() error {
	err := s.listener.Close()
	if err != nil {
		slog.Error(fmt.Sprintf("[%s]: error %s", log.ObjName(s), err))

		return err
	}

	slog.Info(fmt.Sprintf("[%s]: closed", log.ObjName(s)))

	return nil
}

func (s *Sock5NoAuthServer) handleConn(srcConn net.Conn) {
	address, err := s.handshakeNoAuth(srcConn)
	if err != nil {
		slog.Error(fmt.Sprintf("[%s]: handshake error: %v", log.ObjName(s), err))

		srcConn.Close()

		return
	}

	dstConn, err := s.handleDial(srcConn, address)
	if err != nil {
		slog.Error(fmt.Sprintf("[%s]: dial error: %v", log.ObjName(s), err))

		srcConn.Close()

		return
	}

	// slog.Debug(fmt.Sprintf("[%s]: connected %s -- %s (%s)", srcConn.RemoteAddr(), dstConn.RemoteAddr(), address))

	// DEBUG
	// dstConn = dev.NewSnifferHexDumpDebugConn(dstConn, fmt.Sprintf("socks"), false)

	// DEBUG
	// protocolDetectorConn := connections.NewProtocolDetectorConn(dstConn)
	// dstConn = protocolDetectorConn

	// var protocolWg sync.WaitGroup
	// protocolWg.Add(1)

	// go func() {
	// 	defer protocolWg.Done()

	// 	protocol := <-protocolDetectorConn.Detected
	// 	switch pr := protocol.(type) {
	// 	case *connections.HttpProtocol:
	// 		slog.Info(fmt.Sprintf("detected %s protocol", pr.Protocol()))
	// 	case *connections.HttpsProtocol:
	// 		slog.Info(fmt.Sprintf("detected %s protocol %s", pr.Protocol(), pr.Domain))
	// 	}
	// }()

	proxy.Proxy(srcConn, dstConn)

	// protocolWg.Wait()

	// slog.Debug(fmt.Sprintf("[%s]: disconnected %s -- %s (%s)", srcConn.RemoteAddr(), dstConn.RemoteAddr(), address))
}

func (s *Sock5NoAuthServer) handshakeNoAuth(srcConn net.Conn) (string, error) {
	buf := make([]byte, 255) // Max 255 methods

	// Handshake (RFC 1928, Sec 3)
	// Header
	// [VER, NMETHODS] 2 bytes
	if _, err := io.ReadFull(srcConn, buf[:2]); err != nil {
		return "", err
	}

	vrs := buf[0]
	if vrs != version {
		return "", fmt.Errorf("unsupported version %d", version)
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

		return "", fmt.Errorf("no acceptable method")
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
		return "", fmt.Errorf("unsupported command %d", cmd)
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
		return "", fmt.Errorf("unsupported address type %d", atyp)
	}

	// Port (2 bytes, BigEndian)
	if _, err := io.ReadFull(srcConn, buf[:2]); err != nil {
		return "", err
	}

	port := strconv.FormatUint(uint64(binary.BigEndian.Uint16(buf[:2])), 10)

	return net.JoinHostPort(host, port), nil
}

func (s *Sock5NoAuthServer) handleDial(srcConn net.Conn, address string) (net.Conn, error) {
	// slog.Debug(fmt.Sprintf("[%s]: accept connection from %s, wants to connect to %s", srcConn.RemoteAddr(), address))

	// Connect to target
	dstConn, err := s.dialer.Dial(address)
	if err != nil {
		// Reply: REP=1 (General failure) [VER, REP, RSV, ATYP, BND.ADDR, BND.PORT]
		srcConn.Write([]byte{version, 1, 0, 1, 0, 0, 0, 0, 0, 0})

		return nil, err
	}

	// Reply: REP=0 (Success), ATYP=1(tcp), BND.ADDR=0.0.0.0, BND.PORT=0
	if _, err := srcConn.Write([]byte{version, 0, 0, 1, 0, 0, 0, 0, 0, 0}); err != nil {
		dstConn.Close()

		return nil, err
	}

	return dstConn, nil
}
