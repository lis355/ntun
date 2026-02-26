package ntun

import "net"

const (
	MsgConnect = iota
	MsgDisconnect
	MsgData
)

type ConnMultiplexer struct {
	conn net.Conn
}

func NewConnMultiplexer(baseConn *net.Conn) (c *ConnMultiplexer) {
	return &ConnMultiplexer{
		conn: *baseConn,
	}
}

func (c *ConnMultiplexer) Process() {
	
}
