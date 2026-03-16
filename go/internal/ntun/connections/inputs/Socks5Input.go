package inputs

import (
	"net"
	"ntun/internal/cfg"
	"ntun/internal/ntun/node"
	"ntun/internal/proxy/socks"
	"strconv"
)

type Socks5Input struct {
	cfg  *cfg.Socks5Input
	node *node.Node
	serv *socks.Sock5NoAuthServer
}

func NewSocks5Input(cfg *cfg.Socks5Input, node *node.Node) *Socks5Input {
	return &Socks5Input{
		cfg:  cfg,
		node: node,
	}
}

func (s *Socks5Input) Listen() error {
	host := s.cfg.Host
	if host == "" {
		host = "localhost"
	}

	address := net.JoinHostPort(host, strconv.Itoa(int(s.cfg.Port)))

	s.serv = socks.NewSock5NoAuthServer(s.node.ConnectionManager)

	return s.serv.Listen(address)
}

func (s *Socks5Input) Close() error {
	err := s.serv.Close()

	s.serv = nil

	return err
}
