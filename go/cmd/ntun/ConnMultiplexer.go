package ntun

import (
	"encoding/binary"
	"net"
)

const (
	MsgConnect byte = iota
	MsgDisconnect
	MsgData
)

type ConnMultiplexer struct {
	conn net.Conn
}

func NewConnMultiplexer(conn net.Conn) (c *ConnMultiplexer) {
	return &ConnMultiplexer{
		conn: conn,
	}
}

func (c *ConnMultiplexer) SendMsgConnect(connId uint32, address string) {
	addressBuf := []byte(address)
	msgBuf := make([]byte, 1+4+4+len(addressBuf))

	msgBuf[0] = MsgConnect
	binary.BigEndian.PutUint32(msgBuf[1:5], uint32(len(addressBuf)))
	binary.BigEndian.PutUint32(msgBuf[5:9], uint32(len(addressBuf)))
	copy(msgBuf[9:], addressBuf)

	c.conn.Write(msgBuf)
}

func (c *ConnMultiplexer) SendMsgDisconnect(connId uint32) {
	msgBuf := make([]byte, 1+4)

	msgBuf[0] = MsgDisconnect
	binary.BigEndian.PutUint32(msgBuf[1:5], connId)

	c.conn.Write(msgBuf)
}

func (c *ConnMultiplexer) SendMsgData(connId uint32, data []byte) {
	msgBuf := make([]byte, 1+4+len(data))

	msgBuf[0] = MsgData
	binary.BigEndian.PutUint32(msgBuf[1:5], connId)
	copy(msgBuf[5:], data)

	c.conn.Write(msgBuf)
}
