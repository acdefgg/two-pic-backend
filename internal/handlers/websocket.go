package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"sync-photo-backend/internal/services"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for MVP
	},
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub          *services.WSHub
	userService  *services.UserService
	pairService  *services.PairService
	photoService *services.PhotoService
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(
	hub *services.WSHub,
	userService *services.UserService,
	pairService *services.PairService,
	photoService *services.PhotoService,
) *WebSocketHandler {
	return &WebSocketHandler{
		hub:          hub,
		userService:  userService,
		pairService:  pairService,
		photoService: photoService,
	}
}

// HandleWebSocket handles WebSocket connections
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get token from query parameter
	token := r.URL.Query().Get("token")
	if token == "" {
		respondError(w, "token required", http.StatusUnauthorized)
		return
	}

	// Validate token
	userID, err := h.userService.ValidateJWT(token)
	if err != nil {
		respondError(w, "invalid token", http.StatusUnauthorized)
		return
	}

	// Upgrade connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upgrade WebSocket connection")
		return
	}
	defer conn.Close()

	// Register connection
	if err := h.hub.Register(userID, conn); err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to register WebSocket connection")
		return
	}
	defer h.hub.Unregister(userID)

	// Get user's pair and notify partner
	ctx := r.Context()
	pair, err := h.pairService.GetPairByUserID(ctx, userID)
	if err == nil && pair != nil {
		// Пара существует - отправить pair_status с данными пары
		partnerID := pair.UserAID
		if partnerID == userID {
			partnerID = pair.UserBID
		}
		h.hub.NotifyPartnerStatus(userID, partnerID, true)

		// Отправить информацию о паре
		pairStatusMsg := services.WSMessage{
			Type: "pair_status",
			Data: map[string]interface{}{
				"has_pair": true,
				"pair_id":  pair.ID,
			},
		}
		if err := h.hub.SendToUser(userID, pairStatusMsg); err != nil {
			log.Error().
				Err(err).
				Str("user_id", userID).
				Msg("Failed to send pair_status message")
		}
	} else {
		// Пары нет - отправить pair_status без пары
		pairStatusMsg := services.WSMessage{
			Type: "pair_status",
			Data: map[string]interface{}{
				"has_pair": false,
			},
		}
		if err := h.hub.SendToUser(userID, pairStatusMsg); err != nil {
			log.Error().
				Err(err).
				Str("user_id", userID).
				Msg("Failed to send pair_status message")
		}
	}

	log.Info().Str("user_id", userID).Msg("WebSocket connection established")

	// Handle messages
	for {
		_, messageBytes, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().Err(err).Str("user_id", userID).Msg("WebSocket error")
			}
			break
		}

		var msg services.WSMessage
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Error().Err(err).Str("user_id", userID).Msg("Failed to parse WebSocket message")
			h.sendError(conn, "Invalid message format")
			continue
		}

		if err := h.handleMessage(ctx, userID, msg); err != nil {
			log.Error().Err(err).Str("user_id", userID).Str("type", msg.Type).Msg("Failed to handle message")
			h.sendError(conn, err.Error())
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (h *WebSocketHandler) handleMessage(ctx context.Context, userID string, msg services.WSMessage) error {
	switch msg.Type {
	case "trigger_photo":
		return h.handleTriggerPhoto(ctx, userID, msg)
	case "photo_uploaded":
		return h.handlePhotoUploaded(ctx, userID, msg)
	default:
		return h.sendErrorToUser(userID, "Unknown message type")
	}
}

// handleTriggerPhoto handles trigger_photo message
func (h *WebSocketHandler) handleTriggerPhoto(ctx context.Context, userID string, msg services.WSMessage) error {
	// Get user's pair
	pair, err := h.pairService.GetPairByUserID(ctx, userID)
	if err != nil {
		return h.sendErrorToUser(userID, "You are not in a pair")
	}

	// Get partner ID
	partnerID := pair.UserAID
	if partnerID == userID {
		partnerID = pair.UserBID
	}

	// Trigger photo
	timestamp := msg.Timestamp
	if timestamp == 0 {
		// Use current timestamp if not provided
		// We'll handle this in the hub
	}

	return h.hub.TriggerPhoto(userID, partnerID, timestamp)
}

// handlePhotoUploaded handles photo_uploaded message
func (h *WebSocketHandler) handlePhotoUploaded(ctx context.Context, userID string, msg services.WSMessage) error {
	if msg.PhotoID == "" || msg.S3URL == "" {
		return h.sendErrorToUser(userID, "photo_id and s3_url are required")
	}

	// Update photo S3 URL
	if err := h.photoService.UpdatePhotoS3URL(ctx, msg.PhotoID, msg.S3URL); err != nil {
		return h.sendErrorToUser(userID, "Failed to update photo")
	}

	log.Info().
		Str("user_id", userID).
		Str("photo_id", msg.PhotoID).
		Msg("Photo uploaded")

	return nil
}

// sendError sends an error message to the WebSocket connection
func (h *WebSocketHandler) sendError(conn *websocket.Conn, message string) {
	msg := services.WSMessage{
		Type:    "error",
		Message: message,
	}
	data, _ := json.Marshal(msg)
	conn.WriteMessage(websocket.TextMessage, data)
}

// sendErrorToUser sends an error message to a user
func (h *WebSocketHandler) sendErrorToUser(userID, message string) error {
	msg := services.WSMessage{
		Type:    "error",
		Message: message,
	}
	return h.hub.SendToUser(userID, msg)
}
