package server

import (
	"sync"

	"github.com/JIeeiroSst/utils/gosocket/core"
)

type Room struct {
	name        string
	connections map[string]core.Connection
	mu          sync.RWMutex
}

func NewRoom(name string) *Room {
	return &Room{
		name:        name,
		connections: make(map[string]core.Connection),
	}
}

func (r *Room) Join(conn core.Connection) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.connections[conn.ID()] = conn
	return nil
}

func (r *Room) Leave(conn core.Connection) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.connections, conn.ID())
	return nil
}

func (r *Room) Broadcast(msg *core.Message) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, conn := range r.connections {
		conn.Send(msg)
	}

	return nil
}

func (r *Room) Connections() []core.Connection {
	r.mu.RLock()
	defer r.mu.RUnlock()

	conns := make([]core.Connection, 0, len(r.connections))
	for _, conn := range r.connections {
		conns = append(conns, conn)
	}

	return conns
}

func (r *Room) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.connections)
}
