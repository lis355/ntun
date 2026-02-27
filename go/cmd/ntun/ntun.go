package ntun

import (
	"context"
	"net"
)

const (
	Byte     = 1
	Kilobyte = Byte * 1024
	Megabyte = Kilobyte * 1024
	Gigabyte = Megabyte * 1024
	Terabyte = Gigabyte * 1024
)

type Service interface {
	Start() error
	Stop() error
}

type Transporter interface {
	Service
	Transport() <-chan net.Conn
}

type Dialer interface {
	Dial(ctx context.Context, address string) (net.Conn, error)
}

type ConnDial struct {
	ctx     context.Context
	address string
}

type Input interface {
}

type Output interface {
}
