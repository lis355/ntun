package node

import (
	"fmt"
	"ntun/internal/cfg"
	"ntun/internal/ntun/connections"
	"ntun/internal/ntun/transport"

	"github.com/google/uuid"
)

type Node struct {
	Config        *cfg.Config
	Input, Output connections.Сonnecter
	transport.Transporter
	*ConnectionManager
}

func NewNode(config *cfg.Config) *Node {
	return &Node{
		Config: config,
	}
}

func (n *Node) String() string {
	s := n.Config.Id.String()
	if n.Config.Name != "" {
		s += fmt.Sprintf(" [%s]", n.Config.Name)
	}

	return s
}

func (n *Node) HasAllowedToConnectNodeId(id uuid.UUID) bool {
	for _, allowedId := range n.Config.Allowed {
		if allowedId == id {
			return true
		}
	}

	return false
}

func (n *Node) AssignComponents(input, output connections.Сonnecter, transporter transport.Transporter) {
	// TODO hack сделать абстракцию
	outputDialer, _ := output.(connections.Dialer)

	n.Input = input
	n.Output = output
	n.Transporter = transporter
	n.ConnectionManager = NewConnManager(n, outputDialer)
}

func (n *Node) Start() error {
	// slog.Debug(fmt.Sprintf("%s: starting", log.ObjName(n)))
	// defer slog.Debug(fmt.Sprintf("%s: started", log.ObjName(n)))

	err := n.ConnectionManager.Start()
	if err != nil {
		return err
	}

	return err
}

func (n *Node) Stop() error {
	// slog.Debug(fmt.Sprintf("%s: stopping", log.ObjName(n)))
	// defer slog.Debug(fmt.Sprintf("%s: stopped", log.ObjName(n)))

	err := n.ConnectionManager.Stop()
	if err != nil {
		return err
	}

	return err
}
