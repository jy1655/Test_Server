package api

import (
	"encoding/json"
	"net/http"
)

// RegisterRequest represents user registration request
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterService interface for user registration
type RegisterService interface {
	Register(req *RegisterRequest) (interface{}, error)
}

// RegisterHandler handles user registration
type RegisterHandler struct {
	registerService RegisterService
}

// NewRegisterHandler creates a new register handler
func NewRegisterHandler(registerService RegisterService) *RegisterHandler {
	return &RegisterHandler{registerService: registerService}
}

// ServeHTTP handles registration requests
func (h *RegisterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.registerService.Register(&req)
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
