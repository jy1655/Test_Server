package websocket

import (
	"log"
	"sync"
)

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients by type
	clients map[ClientType]map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for thread-safe access to clients map
	mu sync.RWMutex
}

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[ClientType]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.clientType] == nil {
				h.clients[client.clientType] = make(map[*Client]bool)
			}
			h.clients[client.clientType][client] = true
			h.mu.Unlock()

			log.Printf("Client registered: type=%s, user=%s (total: %d)",
				client.clientType, client.username, h.GetClientCount())

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.clientType]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					log.Printf("Client unregistered: type=%s, user=%s (total: %d)",
						client.clientType, client.username, h.GetClientCount())
				}
			}
			h.mu.Unlock()
		}
	}
}

// RegisterClient registers a new client
func (h *Hub) RegisterClient(client *Client) {
	h.register <- client
}

// UnregisterClient unregisters a client
func (h *Hub) UnregisterClient(client *Client) {
	h.unregister <- client
}

// BroadcastToType sends a message to all clients of a specific type
func (h *Hub) BroadcastToType(clientType ClientType, message []byte) {
	h.mu.RLock()
	clients := h.clients[clientType]
	h.mu.RUnlock()

	for client := range clients {
		select {
		case client.send <- message:
		default:
			// Client's send buffer is full, unregister it
			go h.UnregisterClient(client)
		}
	}
}

// BroadcastToAll sends a message to all clients
func (h *Hub) BroadcastToAll(message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, clients := range h.clients {
		for client := range clients {
			select {
			case client.send <- message:
			default:
				go h.UnregisterClient(client)
			}
		}
	}
}

// GetClientCount returns the total number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	for _, clients := range h.clients {
		count += len(clients)
	}
	return count
}

// GetClientCountByType returns the number of clients of a specific type
func (h *Hub) GetClientCountByType(clientType ClientType) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.clients[clientType]; ok {
		return len(clients)
	}
	return 0
}

// GetStats returns statistics about connected clients
func (h *Hub) GetStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total"] = h.GetClientCount()
	stats["web"] = len(h.clients[ClientTypeWeb])
	stats["video"] = len(h.clients[ClientTypeVideo])
	stats["control"] = len(h.clients[ClientTypeControl])
	stats["pending"] = len(h.clients[ClientTypePending])

	return stats
}
