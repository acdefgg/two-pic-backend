package handlers

import (
	"encoding/json"
	"net/http"

	"sync-photo-backend/internal/services"

	"github.com/rs/zerolog/log"
)

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	userService *services.UserService
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// CreateUser handles POST /api/v1/users
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, err := h.userService.CreateUser(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create user")
		respondError(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	log.Info().
		Str("user_id", user.ID).
		Str("code", user.Code).
		Msg("User created")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}
