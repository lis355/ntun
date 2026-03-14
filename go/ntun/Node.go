package ntun

import (
	"fmt"
	"log/slog"
	"ntun/internal/conf"
	"ntun/internal/log"
	"ntun/ntun/connections/outputs"
	"ntun/ntun/transport"

	"github.com/google/uuid"
)

type Node struct {
	Config *conf.Config
	transport.Transporter
	*ConnManager
}

func NewNode(config *conf.Config, transporter transport.Transporter) *Node {
	node := &Node{
		Config:      config,
		Transporter: transporter,
	}

	node.ConnManager = NewConnManager(node, outputs.NewDirectOutput())

	return node
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
