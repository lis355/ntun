package ntun

import "net"

type Conn interface {
	Start() error
	Stop() error
}

type ConnIn interface {
	Conn

	HandleInputConn(address string) (net.Conn, error)
}
