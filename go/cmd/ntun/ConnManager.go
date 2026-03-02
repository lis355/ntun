package ntun

import (
	"context"
	"net"

	"github.com/google/uuid"
)

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
	id := ConnId(uuid.New().String())

	conn, connMux := net.Pipe()

	m.conn[id] = connMux

	m.connMultiplexer.SendMsgConnect(id, address)

	// go m.processConnection(id)

	return conn, nil
}

func (m *ConnManager) processConnection(id ConnId) {
	conn, ok := m.conn[id]
	if !ok {
		return
	}

	_ = conn
}
