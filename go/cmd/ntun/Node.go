package ntun

import (
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

type Node struct {
	Id                      uuid.UUID
	Name                    string
	allowedToConnectNodeIds map[uuid.UUID]struct{}
	Transporter
	*ConnManager
}

func GenerateNodeId() (id string) {
	return uuid.New().String()
}

func NewNode(id uuid.UUID, name string, transporter Transporter) (n *Node) {
	return &Node{
		Id:                      id,
		Name:                    name,
		allowedToConnectNodeIds: make(map[uuid.UUID]struct{}),
		Transporter:             transporter,
		ConnManager:             NewConnManager(transporter),
	}
}

func (n *Node) String() string {
	return fmt.Sprintf("%s [%s]", n.Id.String(), n.Name)
}

func (n *Node) HasAllowedToConnectNodeId(id uuid.UUID) bool {
	_, ok := n.allowedToConnectNodeIds[id]

	return ok
}

func (n *Node) AddAllowedToConnectNodeId(id uuid.UUID) {
	if n.HasAllowedToConnectNodeId(id) {
		panic(fmt.Errorf("Already has id %s", id.String()))
	}

	n.allowedToConnectNodeIds[id] = struct{}{}

	slog.Debug(fmt.Sprintf("%s allowed to connect node with id %s", n.String(), id.String()))
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
