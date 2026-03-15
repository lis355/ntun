package node

import (
	"fmt"
	"log/slog"
	"ntun/internal/conf"
	"ntun/internal/connections"
	"ntun/internal/connections/outputs"
	"ntun/internal/log"
	"ntun/internal/transport"

	"github.com/google/uuid"
)

type Node struct {
	Config *conf.Config
	transport.Transporter
	*ConnManager
}

func NewNode(config *conf.Config) *Node {
	return &Node{
		Config: config,
	}
}

func (n *Node) String() string {
	return fmt.Sprintf("%s [%s]", n.Config.Id, n.Config.Name)
	// return n.Config.Name
}

func (n *Node) HasAllowedToConnectNodeId(id uuid.UUID) bool {
	for _, allowedId := range n.Config.Allowed {
		if allowedId == id {
			return true
		}
	}

	return false
}

// TODO hack сделать абстракцию
func (n *Node) CreateConnManager(transporter transport.Transporter, outputDialer connections.Dialer) {
	n.Transporter = transporter
	n.ConnManager = NewConnManager(n, outputs.NewDirectOutput())
}

func (n *Node) Start() error {
	slog.Debug(fmt.Sprintf("%s: starting", log.ObjName(n)))
	defer slog.Debug(fmt.Sprintf("%s: started", log.ObjName(n)))

	err := n.ConnManager.Start()
	if err != nil {
		return err
	}

	// err = n.Transporter.Start()
	// if err != nil {
	// 	return err
	// }

	return err
}

func (n *Node) Stop() error {
	slog.Debug(fmt.Sprintf("%s: stopping", log.ObjName(n)))
	defer slog.Debug(fmt.Sprintf("%s: stopped", log.ObjName(n)))

	err := n.ConnManager.Stop()
	if err != nil {
		return err
	}

	// err = n.Transporter.Stop()
	// if err != nil {
	// 	return err
	// }

	return err
}
