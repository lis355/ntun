package connections

import (
	"net"
	"ntun/internal/ntun"
)

type Dialer interface {
	Dial(dstAddress string) (net.Conn, error)
}

type Сonnecter interface {
	ntun.Listener
	ntun.Closer
}
