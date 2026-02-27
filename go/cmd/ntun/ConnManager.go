package ntun

import (
	"context"
	"fmt"
	"net"
)

type ConnId string

type ConnManager struct {
	transporter Transporter

	conn            map[ConnId]net.Conn
	connMultiplexer *ConnMultiplexer
}

func NewConnManager(transporter Transporter) *ConnManager {
	return &ConnManager{
		transporter: transporter,
		conn:        make(map[ConnId]net.Conn),
	}
}

func (m *ConnManager) Start() error {
	go m.process()

	return nil
}

func (m *ConnManager) Stop() error {
	return nil
}

func (m *ConnManager) process() {
	for baseConn := range m.transporter.Transport() {
		m.connMultiplexer = NewConnMultiplexer(baseConn)
	}
}

func (m *ConnManager) Dial(ctx context.Context, address string) (net.Conn, error) {
	// id := GetId(conn)

	// if _, ok := m.conn[id]; ok {
	// 	panic("[ConnManager] id already exists")
	// }

	// m.conn[id] = conn

	// go m.processConnection(id)

	return nil, nil
}

func GetId(conn net.Conn) ConnId {
	return ConnId(fmt.Sprintf("%s -- %s", conn.LocalAddr(), conn.RemoteAddr()))
}

func (m *ConnManager) processConnection(id ConnId) {
	conn, ok := m.conn[id]
	if !ok {
		return
	}

	_ = conn
}
