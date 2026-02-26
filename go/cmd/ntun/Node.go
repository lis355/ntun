package ntun

import (
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

type Node struct {
	Id   uuid.UUID
	Name string
	Conn
	allowedToConnectNodeIds map[uuid.UUID]struct{}
}

func GenerateNodeId() (id string) {
	return uuid.New().String()
}

func NewNode(id uuid.UUID, name string) (n *Node) {
	return &Node{
		Id:                      id,
		Name:                    name,
		allowedToConnectNodeIds: make(map[uuid.UUID]struct{}),
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
