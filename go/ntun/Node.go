package ntun

import (
	"log/slog"
	"ntun/internal/conf"

	"github.com/google/uuid"
)

type Node struct {
	Config *conf.Config
	Transporter
	*ConnManager
}

func NewNode(config *conf.Config, transporter Transporter) *Node {
	node := &Node{
		Config:      config,
		Transporter: transporter,
	}

	node.ConnManager = NewConnManager(node)

	return node
}

func (n *Node) String() string {
	//DEBUG
	// return fmt.Sprintf("%s [%s]", n.Config.Id.String(), n.Config.Name)
	return n.Config.Name
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
	slog.Debug("[Node] starting")
	defer slog.Debug("[Node] started")

	err := n.ConnManager.Start()
	if err != nil {
		return err
	}

	err = n.Transporter.Start()
	if err != nil {
		return err
	}

	return err
}

func (n *Node) Stop() error {
	slog.Debug("[Node] stopping")
	defer slog.Debug("[Node] stopped")

	err := n.ConnManager.Stop()
	if err != nil {
		return err
	}

	err = n.Transporter.Stop()
	if err != nil {
		return err
	}

	return err
}
