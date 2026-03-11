package ntun

import (
	"context"
	"net"
)

type ConnManager struct {
	transporter Transporter

	connMux *ConnMux
}

func NewConnManager(transporter Transporter) *ConnManager {
	return &ConnManager{
		transporter: transporter,
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
		m.connMux = NewConnMux(baseConn)
	}
}

func (m *ConnManager) Dial(ctx context.Context, srcAddress, dstAddress string) (net.Conn, error) {
	return net.Dial("tcp", dstAddress)

	// select {
	// case <-ctx.Done():
	// 	return nil, fmt.Errorf("Ctx is done")
	// default:
	// 	return m.connMux.createConn(srcAddress, dstAddress)
	// }
}
