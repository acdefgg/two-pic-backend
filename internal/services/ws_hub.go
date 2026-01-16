package services

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type        string      `json:"type"`
	Timestamp   int64       `json:"timestamp,omitempty"`
	InitiatorID string      `json:"initiator_id,omitempty"`
	PhotoID     string      `json:"photo_id,omitempty"`
	S3URL       string      `json:"s3_url,omitempty"`
	Online      *bool       `json:"online,omitempty"`
	Message     string      `json:"message,omitempty"`
	Data        interface{} `json:"data,omitempty"`
}

// WSHub manages WebSocket connections
type WSHub struct {
	mu          sync.RWMutex
	connections map[string]*websocket.Conn
	pairService *PairService
}

// NewWSHub creates a new WebSocket hub
func NewWSHub(pairService *PairService) *WSHub {
	return &WSHub{
		connections: make(map[string]*websocket.Conn),
		pairService: pairService,
	}
}

// Register registers a new WebSocket connection for a user
func (h *WSHub) Register(userID string, conn *websocket.Conn) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Close existing connection if any
	if existingConn, exists := h.connections[userID]; exists {
		existingConn.Close()
	}

	h.connections[userID] = conn

	log.Info().Str("user_id", userID).Msg("WebSocket connection registered")

	// Notify partner about online status
	go h.notifyPartnerStatus(userID, true)

	return nil
}

// Unregister removes a WebSocket connection for a user
func (h *WSHub) Unregister(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conn, exists := h.connections[userID]; exists {
		conn.Close()
		delete(h.connections, userID)
		log.Info().Str("user_id", userID).Msg("WebSocket connection unregistered")
	}

	// Notify partner about offline status
	go h.notifyPartnerStatus(userID, false)
}

// SendToUser sends a message to a specific user
func (h *WSHub) SendToUser(userID string, message WSMessage) error {
	h.mu.RLock()
	conn, exists := h.connections[userID]
	h.mu.RUnlock()

	if !exists {
		return fmt.Errorf("user %s is not connected", userID)
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		h.Unregister(userID)
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// IsOnline checks if a user is online
func (h *WSHub) IsOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, exists := h.connections[userID]
	return exists
}

// GetPartnerID gets the partner ID for a user
func (h *WSHub) GetPartnerID(userID string) (string, error) {
	// This will be called from handler with context, so we need to pass context
	// For now, we'll use a simple approach - the handler will get the partner ID
	return "", fmt.Errorf("use GetPartnerID from handler context")
}

// notifyPartnerStatus notifies the partner about online/offline status
func (h *WSHub) notifyPartnerStatus(userID string, online bool) {
	// This needs to be called with context to get the pair
	// We'll handle this in the handler
}

// TriggerPhoto handles trigger_photo message
func (h *WSHub) TriggerPhoto(initiatorID, partnerID string, timestamp int64) error {
	// Check if partner is online
	if !h.IsOnline(partnerID) {
		message := WSMessage{
			Type:    "error",
			Message: "Partner is offline",
		}
		return h.SendToUser(initiatorID, message)
	}

	// Use current timestamp if not provided
	if timestamp == 0 {
		timestamp = time.Now().UnixMilli()
	}

	// Send take_photo message to both users
	takePhotoMsg := WSMessage{
		Type:        "take_photo",
		InitiatorID: initiatorID,
		Timestamp:   timestamp,
	}

	if err := h.SendToUser(initiatorID, takePhotoMsg); err != nil {
		log.Error().Err(err).Str("user_id", initiatorID).Msg("Failed to send take_photo to initiator")
	}

	if err := h.SendToUser(partnerID, takePhotoMsg); err != nil {
		log.Error().Err(err).Str("user_id", partnerID).Msg("Failed to send take_photo to partner")
		return err
	}

	log.Info().
		Str("initiator_id", initiatorID).
		Str("partner_id", partnerID).
		Int64("timestamp", timestamp).
		Msg("Photo triggered")

	return nil
}

// NotifyPartnerStatus notifies partner about online/offline status
func (h *WSHub) NotifyPartnerStatus(userID, partnerID string, online bool) {
	if partnerID == "" {
		return
	}

	message := WSMessage{
		Type:   "partner_status",
		Online: &online,
	}

	if err := h.SendToUser(partnerID, message); err != nil {
		log.Error().
			Err(err).
			Str("user_id", partnerID).
			Msg("Failed to notify partner status")
	}
}

// NotifyPairCreated notifies the second user when a pair is created
func (h *WSHub) NotifyPairCreated(partnerID string, pairID, userAID, userBID string, createdAt time.Time) error {
	message := WSMessage{
		Type: "pair_created",
		Data: map[string]interface{}{
			"pair_id":    pairID,
			"user_a_id":  userAID,
			"user_b_id":  userBID,
			"created_at": createdAt,
		},
	}
	return h.SendToUser(partnerID, message)
}

// NotifyPairDeleted notifies the partner when a pair is deleted
func (h *WSHub) NotifyPairDeleted(partnerID string) error {
	message := WSMessage{
		Type: "pair_deleted",
	}
	return h.SendToUser(partnerID, message)
}
