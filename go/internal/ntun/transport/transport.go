package transport

import (
	"net"
	"ntun/internal/cfg"
	"ntun/internal/ntun"
)

type Transporter interface {
	ntun.Listener
	ntun.Closer
	Transport() (net.Conn, error) // TODO попробовать net.Conn заменить на свой интерфейс Duplex?
	RateLimit() *cfg.Rate
}
