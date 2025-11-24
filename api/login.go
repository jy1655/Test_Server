package api

import (
	"encoding/json"
	"net/http"
	"oculo-pilot-server/auth"
)

// LoginHandler handles user login
type LoginHandler struct {
	authService *auth.Service
}

// NewLoginHandler creates a new login handler
func NewLoginHandler(authService *auth.Service) *LoginHandler {
	return &LoginHandler{authService: authService}
}

// ServeHTTP handles login requests
func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req auth.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, err := h.authService.Login(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
