package ntun

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"

	"github.com/vmihailenco/msgpack/v5"
)

const (
	MsgTypeConnect byte = iota
	MsgTypeDisconnect
	MsgTypeData
)

type ConnId string

func NewConnId(srcAddress, dstAddress string) ConnId {
	return ConnId(fmt.Sprintf("%s--%s", srcAddress, dstAddress))
}

type MuxConn struct {
	net.Conn
	id ConnId
}

type ConnMux struct {
	conn        net.Conn
	connections map[ConnId]net.Conn
}

type MsgConn struct {
	Id ConnId
}

type MsgConnect struct {
	MsgConn
	Address string
}

func NewConnMux(transportConn net.Conn) (c *ConnMux) {
	transportConn = NewSniffConn("transportConn", transportConn)

	return &ConnMux{
		conn:        transportConn,
		connections: make(map[ConnId]net.Conn),
	}
}

func (c *ConnMux) createConn(srcAddress, dstAddress string) (net.Conn, error) {
	id := NewConnId(srcAddress, dstAddress)

	slog.Debug(fmt.Sprintf("[%T]: createConn id=%s", c, id))

	msgBytes, err := msgpack.Marshal(&MsgConnect{MsgConn{Id: id}, dstAddress})
	if err != nil {
		return nil, err
	}

	c.WriteMsg(MsgTypeConnect, msgBytes)

	muxConn := &MuxConn{
		Conn: &net.IPConn{}, // DEBUG Mock
		id:   id,
	}

	return muxConn, nil
}

func (c *ConnMux) WriteMsg(msgType byte, msg []byte) error {
	_, err := c.conn.Write([]byte{msgType})
	if err != nil {
		return err
	}

	msgLenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(msgLenBuf, uint32(len(msg)))
	_, err = c.conn.Write(msgLenBuf)
	if err != nil {
		return err
	}

	_, err = c.conn.Write(msg)
	if err != nil {
		return err
	}

	return nil
}
