package connections

import (
	"net"
	"strings"

	"github.com/open-ch/ja3"
)

const (
	ProtocolTypeHTTP  = "HTTP"
	ProtocolTypeHTTPS = "HTTPS"
)

type Protocoler interface {
	Protocol() string
}

type HttpProtocol struct {
}

func (p *HttpProtocol) Protocol() string {
	return ProtocolTypeHTTP
}

type HttpsProtocol struct {
	Domain string
}

func (p *HttpsProtocol) Protocol() string {
	return ProtocolTypeHTTPS
}

type ProtocolDetectorConn struct {
	net.Conn
	buf     []byte
	active  bool
	cleared bool

	Detected chan Protocoler
	Protocoler
}

func NewProtocolDetectorConn(conn net.Conn) *ProtocolDetectorConn {
	return &ProtocolDetectorConn{
		Conn:     conn,
		active:   true,
		Detected: make(chan Protocoler, 1),
	}
}

func (c *ProtocolDetectorConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)

	if err != nil {
		c.clear()
	}

	return n, err
}

func (c *ProtocolDetectorConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)

	if c.active &&
		n > 0 {
		c.process(b[:n])
	}

	if err != nil {
		c.clear()
	}

	return n, err
}

func (c *ProtocolDetectorConn) Close() error {
	err := c.Conn.Close()

	c.clear()

	return err
}

func (c *ProtocolDetectorConn) clear() {
	if c.cleared {
		return
	}

	c.cleared = true

	close(c.Detected)
	c.Detected = nil
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
			c.detect(&HttpsProtocol{Domain: sni})
		}
	} else if len(c.buf) >= 7 {
		// переводим всё в верхний регистр чтобы учесть варианты типа PoSt
		prefix := strings.ToUpper(string(c.buf[:7]))
		for _, method := range httpMethods {
			if strings.HasPrefix(prefix, method) {
				c.detect(&HttpProtocol{})
				break
			}
		}
	} else if len(c.buf) >= 1024 {
		// ничего не нашли, хватит парсить
		c.detect(nil)
	}
}

func (c *ProtocolDetectorConn) detect(protocol Protocoler) {
	if !c.active {
		return
	}

	c.active = false
	c.buf = nil
	c.Protocoler = protocol

	c.Detected <- protocol

	c.clear()
}
