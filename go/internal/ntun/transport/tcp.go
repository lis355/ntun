package transport

import (
	"fmt"
	"log/slog"
	"net"
	"ntun/internal/cfg"
	"ntun/internal/log"
	"strconv"
	"time"
)

type TcpServerTransport struct {
	cfg *cfg.TcpServerTransport

	// ctx    context.Context
	// cancel context.CancelFunc

	listener net.Listener
	// conn     net.Conn // только 1 активное соединиение, других отклоняем
	// connMu   sync.Mutex
	// connChan chan net.Conn

	// running bool
}

func NewTcpServerTransport(cfg *cfg.TcpServerTransport) (c *TcpServerTransport) {
	return &TcpServerTransport{
		cfg: cfg,
		// connChan: make(chan net.Conn),
	}
}

func (c *TcpServerTransport) Transport() (net.Conn, error) {
	if c.listener == nil {
		return nil, fmt.Errorf("listener is not started")
	}

	conn, err := c.listener.Accept()
	if err != nil {
		slog.Debug(fmt.Sprintf("%s: connection failed, waiting", log.ObjName(c)))

		return nil, err
	}

	slog.Debug(fmt.Sprintf("%s: connected successfull with %s", log.ObjName(c), conn.RemoteAddr().String()))

	return conn, nil
}

// func (c *TcpServerTransport) Start() error {
// 	slog.Debug("[TcpServerTransport] starting")
// 	defer slog.Debug("[TcpServerTransport] started")

// 	c.connMu.Lock()
// 	err := func() error {
// 		if c.running {
// 			return fmt.Errorf("[TcpServerTransport] already started")
// 		}

// 		c.ctx, c.cancel = context.WithCancel(context.Background())

// 		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", c.port))
// 		if err != nil {
// 			return err
// 		}

// 		c.listener = listener

// 		go c.listen()

// 		c.running = true

// 		return nil
// 	}()
// 	c.connMu.Unlock()

// 	return err
// }

// func (c *TcpServerTransport) Stop() error {
// 	slog.Debug("[TcpServerTransport] stopping")
// 	defer slog.Debug("[TcpServerTransport] stopped")

// 	c.connMu.Lock()
// 	err := func() error {
// 		if !c.running {
// 			return fmt.Errorf("[TcpServerTransport] already stopped")
// 		}

// 		c.cancel()

// 		c.listener.Close()
// 		if c.conn != nil {
// 			c.conn.Close()
// 		}

// 		c.running = false

// 		return nil
// 	}()
// 	c.connMu.Unlock()

// 	return err
// }

func (c *TcpServerTransport) Listen() error {
	host := c.cfg.Host
	if host == "" {
		host = "localhost"
	}

	address := net.JoinHostPort(host, strconv.Itoa(int(c.cfg.Port)))

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	c.listener = listener

	slog.Debug(fmt.Sprintf("%s: listening %s", log.ObjName(c), address))

	return nil

	// for {
	// 	conn, err := c.listener.Accept()
	// 	if err != nil {
	// 		select {
	// 		case <-c.ctx.Done():
	// 			return
	// 		default:
	// 			if err != io.EOF {
	// 				slog.Error(err.Error())
	// 			}

	// 			continue
	// 		}
	// 	}

	// 	hasActiveConn := false

	// 	c.connMu.Lock()
	// 	if c.conn != nil {
	// 		hasActiveConn = true
	// 	} else {
	// 		c.conn = conn
	// 	}
	// 	c.connMu.Unlock()

	// 	if hasActiveConn {
	// 		conn.Close()
	// 		continue
	// 	}

	// 	slog.Debug(fmt.Sprintf("[TcpServerTransport] connected successfull to %s", c.conn.RemoteAddr().String()))

	// 	c.connChan <- c.conn

	// 	c.connMu.Lock()
	// 	c.conn = nil
	// 	c.connMu.Unlock()
	// }
}

func (c *TcpServerTransport) Close() error {
	// TODO robust handle
	return c.listener.Close()
}

const TcpClientDialTimeout = 10 * time.Second
const TcpClientReconnectTimeout = 1 * time.Second

type TcpClientTransport struct {
	cfg *cfg.TcpClientTransport

	// ctx    context.Context
	// cancel context.CancelFunc

	// conn     net.Conn
	// connMu   sync.Mutex
	// connChan chan net.Conn

	// running bool

	dialer net.Dialer
}

func NewTcpClientTransport(cfg *cfg.TcpClientTransport) (c *TcpClientTransport) {
	return &TcpClientTransport{
		cfg: cfg,
		// connChan: make(chan net.Conn),
		dialer: net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 3 * time.Minute,
		},
	}
}

func (c *TcpClientTransport) Transport() (net.Conn, error) {
	address := net.JoinHostPort(c.cfg.Host, strconv.Itoa(int(c.cfg.Port)))

	slog.Debug(fmt.Sprintf("%s: trying to connect to %s", log.ObjName(c), address))

	conn, err := c.dialer.Dial("tcp", address)
	if err != nil {
		slog.Debug(fmt.Sprintf("%s: connect failed", log.ObjName(c)))

		return nil, err
	}

	slog.Debug(fmt.Sprintf("%s: connected successfull to %s", log.ObjName(c), address))

	return conn, nil
}

func (c *TcpClientTransport) Listen() error {
	return nil
}

func (c *TcpClientTransport) Close() error {
	// TODO close current connection
	return nil
}

// func (c *TcpClientTransport) Start() error {
// 	slog.Debug("[TcpClientTransport] starting")
// 	defer slog.Debug("[TcpClientTransport] started")

// 	c.connMu.Lock()
// 	err := func() error {
// 		if c.running {
// 			return fmt.Errorf("[TcpClientTransport] already started")
// 		}

// 		c.ctx, c.cancel = context.WithCancel(context.Background())

// 		go c.reconnect()
// 		c.running = true

// 		return nil
// 	}()
// 	c.connMu.Unlock()

// 	return err
// }

// func (c *TcpClientTransport) Stop() error {
// 	slog.Debug("[TcpClientTransport] stopping")
// 	defer slog.Debug("[TcpClientTransport] stopped")

// 	c.connMu.Lock()
// 	err := func() error {
// 		if !c.running {
// 			return fmt.Errorf("[TcpClientTransport] already stopped")
// 		}

// 		c.cancel()
// 		c.conn.Close()
// 		c.running = false

// 		return nil
// 	}()
// 	c.connMu.Unlock()

// 	return err
// }

// func (c *TcpClientTransport) reconnect() {
// 	for {
// 		slog.Debug(fmt.Sprintf("[TcpClientTransport] trying to connect to %s", c.address))

// 		conn, err := c.dial()
// 		if err != nil {
// 			slog.Debug("[TcpClientTransport] connect failed, waiting")

// 			select {
// 			case <-c.ctx.Done():
// 				return
// 			case <-time.After(TcpClientReconnectTimeout):
// 				continue
// 			}
// 		}

// 		c.connMu.Lock()
// 		c.conn = conn
// 		c.connMu.Unlock()

// 		slog.Debug(fmt.Sprintf("[TcpClientTransport] connected successfull to %s", c.address))

// 		c.connChan <- c.conn

// 		c.connMu.Lock()
// 		c.conn = nil
// 		c.connMu.Unlock()

// 		select {
// 		case <-c.ctx.Done():
// 			return
// 		default:
// 			continue
// 		}
// 	}
// }

// func (c *TcpClientTransport) dial() (net.Conn, error) {
// 	ctx, cancel := context.WithTimeout(c.ctx, TcpClientDialTimeout)
// 	defer cancel()

// 	return c.dialer.DialContext(ctx, "tcp", c.address)
// }

// func (c *TcpClientTransport) Dial(ctx context.Context, address string) (net.Conn, error) {
// 	c.connMu.Lock()
// 	conn := c.conn
// 	c.connMu.Unlock()

// 	if conn == nil {
// 		return nil, fmt.Errorf("[TcpClientTransport] not connected")
// 	}

// 	clientSide, muxSide := net.Pipe()
// 	_ = muxSide

// 	// streamID := m.nextID()
// 	// go m.handleVirtualStream(streamID, address, muxSide)

// 	wbuf := make([]byte, 1024)
// 	copy(wbuf, []byte(address)) // Копирует байты из s в начало buf
// 	c.conn.Write(wbuf)

// 	go func() {
// 		bytes := make([]byte, defaultDataBufferSize)

// 		for {
// 			n, err := muxSide.Read(bytes)
// 			if err != nil {
// 				return
// 			}

// 			// вначале соединения всегда читается 0, мб это прикол либы github.com/armon/go-socks5 ?
// 			if n == 0 {
// 				continue
// 			}

// 			slog.Debug(fmt.Sprintf("muxSide read %d", n))
// 		}
// 	}()

// 	return clientSide, nil
// }
