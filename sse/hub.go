package sse

import (
	"sync"

	"github.com/google/uuid"
)

type Hub struct {
	mu        sync.RWMutex
	queueSize int
	byClient  map[string]*Connection
	byUser    map[string]map[string]*Connection
}

type Connection struct {
	mu       sync.Mutex
	id       string
	identity Identity
	events   chan Event
	closed   bool
}

func NewHub(queueSize int) *Hub {
	if queueSize <= 0 {
		queueSize = defaultQueueSize
	}

	return &Hub{
		queueSize: queueSize,
		byClient:  map[string]*Connection{},
		byUser:    map[string]map[string]*Connection{},
	}
}

func (hub *Hub) Add(identity Identity) *Connection {
	connection := &Connection{
		id:       uuid.NewString(),
		identity: identity,
		events:   make(chan Event, hub.queueSize),
	}

	hub.mu.Lock()
	defer hub.mu.Unlock()

	hub.byClient[connection.id] = connection
	if _, ok := hub.byUser[identity.UserID]; !ok {
		hub.byUser[identity.UserID] = map[string]*Connection{}
	}
	hub.byUser[identity.UserID][connection.id] = connection

	return connection
}

func (hub *Hub) Remove(clientID string) {
	hub.mu.Lock()
	connection, ok := hub.byClient[clientID]
	if ok {
		delete(hub.byClient, clientID)
		if connections, exists := hub.byUser[connection.identity.UserID]; exists {
			delete(connections, clientID)
			if len(connections) == 0 {
				delete(hub.byUser, connection.identity.UserID)
			}
		}
	}
	hub.mu.Unlock()

	if ok {
		connection.Close()
	}
}

func (hub *Hub) Deliver(event Event) int {
	hub.mu.RLock()
	connections := hub.byUser[event.UserID]
	targets := make([]*Connection, 0, len(connections))
	for _, connection := range connections {
		if event.ClientID != "" && event.ClientID != connection.id {
			continue
		}
		targets = append(targets, connection)
	}
	hub.mu.RUnlock()

	delivered := 0
	for _, connection := range targets {
		if connection.identity.UserID != event.UserID {
			continue
		}
		if event.ClientID != "" && event.ClientID != connection.id {
			continue
		}
		if connection.Enqueue(event) {
			delivered++
			continue
		}
		hub.Remove(connection.id)
	}

	return delivered
}

func (hub *Hub) UserConnectionCount(userID string) int {
	hub.mu.RLock()
	defer hub.mu.RUnlock()

	return len(hub.byUser[userID])
}

func (connection *Connection) ID() string {
	return connection.id
}

func (connection *Connection) Events() <-chan Event {
	return connection.events
}

func (connection *Connection) Enqueue(event Event) bool {
	connection.mu.Lock()
	defer connection.mu.Unlock()

	if connection.closed {
		return false
	}

	select {
	case connection.events <- event:
		return true
	default:
		return false
	}
}

func (connection *Connection) Close() {
	connection.mu.Lock()
	defer connection.mu.Unlock()

	if connection.closed {
		return
	}
	connection.closed = true
	close(connection.events)
}
