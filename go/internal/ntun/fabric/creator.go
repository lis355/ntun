package fabric

import (
	"fmt"
	"ntun/internal/cfg"
	"ntun/internal/ntun/connections"
	"ntun/internal/ntun/connections/inputs"
	"ntun/internal/ntun/connections/outputs"
	"ntun/internal/ntun/node"
	"ntun/internal/ntun/transport"
	"ntun/internal/ntun/transport/yandex"
)

func CreateInput(node *node.Node) (connections.Сonnecter, error) {
	if node.Config.Input == nil {
		return nil, nil
	}

	switch config := node.Config.Input.(type) {
	case *cfg.Socks5Input:
		return inputs.NewSocks5Input(config, node), nil
	default:
		panic(fmt.Errorf("unknown input type %v", config))
	}
}

func CreateOutput(node *node.Node) (connections.Сonnecter, error) {
	if node.Config.Output == nil {
		return nil, nil
	}

	switch config := node.Config.Output.(type) {
	case *cfg.DirectOutput:
		return outputs.NewDirectOutput(), nil
	default:
		panic(fmt.Errorf("unknown output type %v", config))
	}
}

func CreateTransporter(node *node.Node) (transport.Transporter, error) {
	if node.Config.Transport == nil {
		panic(fmt.Errorf("nil transport"))
	}

	switch config := node.Config.Transport.(type) {
	case *cfg.TcpClientTransport:
		return transport.NewTcpClientTransport(config), nil
	case *cfg.TcpServerTransport:
		return transport.NewTcpServerTransport(config), nil
	case *cfg.YandexWebRTCTransport:
		return yandex.NewYandexWebRTCTransport(config, node)
	default:
		panic(fmt.Errorf("unknown transport type %v", config))
	}
}
