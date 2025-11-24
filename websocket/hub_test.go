package websocket

import (
	"testing"
)

// TestNewHub tests hub creation
func TestNewHub(t *testing.T) {
	hub := NewHub()

	if hub == nil {
		t.Fatal("NewHub() returned nil")
	}

	if hub.clients == nil {
		t.Error("Hub clients map not initialized")
	}

	if hub.register == nil {
		t.Error("Hub register channel not initialized")
	}

	if hub.unregister == nil {
		t.Error("Hub unregister channel not initialized")
	}
}

// TestHubGetClientCount tests client counting
func TestHubGetClientCount(t *testing.T) {
	hub := NewHub()

	count := hub.GetClientCount()
	if count != 0 {
		t.Errorf("Expected 0 clients in new hub, got %d", count)
	}
}

// TestHubGetClientCountByType tests client counting by type
func TestHubGetClientCountByType(t *testing.T) {
	hub := NewHub()

	types := []ClientType{
		ClientTypeWeb,
		ClientTypeControl,
		ClientTypeVideo,
		ClientTypeTelemetry,
		ClientTypePending,
	}

	for _, clientType := range types {
		count := hub.GetClientCountByType(clientType)
		if count != 0 {
			t.Errorf("Expected 0 %s clients, got %d", clientType, count)
		}
	}
}

// TestHubGetStats tests statistics retrieval
func TestHubGetStats(t *testing.T) {
	hub := NewHub()

	stats := hub.GetStats()

	// Check stats structure
	if stats == nil {
		t.Fatal("GetStats() returned nil")
	}

	// Check required fields exist
	requiredFields := []string{"total", "web", "video", "control", "telemetry", "pending"}
	for _, field := range requiredFields {
		if _, ok := stats[field]; !ok {
			t.Errorf("Stats missing required field: %s", field)
		}
	}

	// Check initial values are 0
	if stats["total"] != 0 {
		t.Errorf("Expected total 0, got %v", stats["total"])
	}
}

// TestClientTypes tests client type constants
func TestClientTypes(t *testing.T) {
	types := map[ClientType]string{
		ClientTypeWeb:       "web",
		ClientTypeVideo:     "video",
		ClientTypeControl:   "control",
		ClientTypeTelemetry: "telemetry",
		ClientTypePending:   "pending",
	}

	for clientType, expectedString := range types {
		if string(clientType) != expectedString {
			t.Errorf("Expected %s to equal '%s', got '%s'", clientType, expectedString, string(clientType))
		}
	}
}

