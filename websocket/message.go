package websocket

import (
	"encoding/json"
	"log"
)

// Message represents a WebSocket message
type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// HandshakeResponse represents handshake response from client
type HandshakeResponse struct {
	Type       string     `json:"type"`
	ClientType ClientType `json:"client_type"`
}

// RouteMessage routes a message from sender to appropriate recipients
func (h *Hub) RouteMessage(sender *Client, rawMessage []byte) {
	var msg Message
	if err := json.Unmarshal(rawMessage, &msg); err != nil {
		log.Printf("Invalid message format from %s: %v", sender.clientType, err)
		return
	}

	log.Printf("Message received: type=%s from client_type=%s user=%s",
		msg.Type, sender.clientType, sender.username)

	switch msg.Type {
	case "handshake_response":
		h.handleHandshake(sender, rawMessage)

	case "ping":
		h.handlePing(sender, rawMessage)

	case "pong":
		// Just log pong messages
		log.Printf("Pong received from %s", sender.clientType)

	case "control_command":
		// Control commands from web clients go to control clients
		if sender.clientType == ClientTypeWeb {
			h.BroadcastToType(ClientTypeControl, rawMessage)
			log.Printf("Routed control command to %d control clients",
				h.GetClientCountByType(ClientTypeControl))
		}

	case "control_response":
		// Control responses from control clients go back to web clients
		if sender.clientType == ClientTypeControl {
			h.BroadcastToType(ClientTypeWeb, rawMessage)
			log.Printf("Routed control response to %d web clients",
				h.GetClientCountByType(ClientTypeWeb))
		}

	case "offer", "answer", "ice-candidate":
		// WebRTC signaling
		h.handleWebRTCSignaling(sender, msg.Type, rawMessage)

	case "video_client_ready":
		// Video client is ready, notify web clients
		h.BroadcastToType(ClientTypeWeb, rawMessage)
		log.Printf("Notified %d web clients that video is ready",
			h.GetClientCountByType(ClientTypeWeb))

	case "emergency_stop":
		// Emergency stop broadcasts to all control clients
		h.BroadcastToType(ClientTypeControl, rawMessage)
		log.Printf("🚨 Emergency stop broadcast to %d control clients",
			h.GetClientCountByType(ClientTypeControl))

	case "route_update", "location_update":
		// Telemetry updates go to web clients
		h.BroadcastToType(ClientTypeWeb, rawMessage)
		log.Printf("Forwarded %s to %d web clients",
			msg.Type, h.GetClientCountByType(ClientTypeWeb))

	default:
		// Unknown message type - broadcast to all except sender
		log.Printf("Unknown message type: %s, broadcasting to all", msg.Type)
		h.broadcastExceptSender(sender, rawMessage)
	}
}

// handleHandshake processes handshake response from client
func (h *Hub) handleHandshake(client *Client, rawMessage []byte) {
	var handshake HandshakeResponse
	if err := json.Unmarshal(rawMessage, &handshake); err != nil {
		log.Printf("Invalid handshake response: %v", err)
		return
	}

	// Update client type if it was pending
	if client.clientType == ClientTypePending {
		h.mu.Lock()
		// Remove from pending
		if clients, ok := h.clients[ClientTypePending]; ok {
			delete(clients, client)
		}

		// Add to new type
		client.clientType = handshake.ClientType
		if h.clients[client.clientType] == nil {
			h.clients[client.clientType] = make(map[*Client]bool)
		}
		h.clients[client.clientType][client] = true
		h.mu.Unlock()

		log.Printf("Client type updated: %s (user=%s)", client.clientType, client.username)

		// Send confirmation
		response := map[string]interface{}{
			"type":        "handshake_confirmed",
			"client_type": client.clientType,
		}
		client.SendJSON(response)
	}
}

// handlePing responds to ping messages with pong
func (h *Hub) handlePing(client *Client, rawMessage []byte) {
	var pingMsg map[string]interface{}
	if err := json.Unmarshal(rawMessage, &pingMsg); err != nil {
		return
	}

	pongMsg := map[string]interface{}{
		"type":      "pong",
		"timestamp": pingMsg["timestamp"],
	}

	client.SendJSON(pongMsg)
}

// handleWebRTCSignaling routes WebRTC signaling messages
func (h *Hub) handleWebRTCSignaling(sender *Client, msgType string, rawMessage []byte) {
	switch sender.clientType {
	case ClientTypeWeb:
		// Web client's offer/ice-candidate goes to video client
		h.BroadcastToType(ClientTypeVideo, rawMessage)
		log.Printf("Routed %s from web to %d video clients",
			msgType, h.GetClientCountByType(ClientTypeVideo))

	case ClientTypeVideo:
		// Video client's answer/ice-candidate goes to web clients
		h.BroadcastToType(ClientTypeWeb, rawMessage)
		log.Printf("Routed %s from video to %d web clients",
			msgType, h.GetClientCountByType(ClientTypeWeb))

	default:
		log.Printf("Unexpected WebRTC signaling from %s", sender.clientType)
	}
}

// broadcastExceptSender sends message to all clients except the sender
func (h *Hub) broadcastExceptSender(sender *Client, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, clients := range h.clients {
		for client := range clients {
			if client != sender {
				select {
				case client.send <- message:
				default:
					go h.UnregisterClient(client)
				}
			}
		}
	}
}
