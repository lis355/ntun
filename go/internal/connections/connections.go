package connections

import "net"

type Dialer interface {
	Dial(dstAddress string) (net.Conn, error)
}
