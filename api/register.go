package api

import (
	"encoding/json"
	"net/http"
	"oculo-pilot-server/auth"
)

// RegisterHandler handles user registration
type RegisterHandler struct {
	authService *auth.Service
}

// NewRegisterHandler creates a new register handler
func NewRegisterHandler(authService *auth.Service) *RegisterHandler {
	return &RegisterHandler{authService: authService}
}

// ServeHTTP handles registration requests
func (h *RegisterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req auth.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.authService.Register(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user": user,
	})
}
