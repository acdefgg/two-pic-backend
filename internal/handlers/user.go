package handlers

import (
	"encoding/json"
	"net/http"

	"sync-photo-backend/internal/middleware"
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

// UpdatePushToken handles PUT /api/v1/users/push-token
func (h *UserHandler) UpdatePushToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	var req struct {
		PushToken string `json:"push_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.PushToken == "" {
		respondError(w, "push_token is required", http.StatusBadRequest)
		return
	}

	if err := h.userService.UpdatePushToken(ctx, userID, req.PushToken); err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to update push token")
		respondError(w, "Failed to update push token", http.StatusInternalServerError)
		return
	}

	log.Info().Str("user_id", userID).Msg("Push token updated")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
