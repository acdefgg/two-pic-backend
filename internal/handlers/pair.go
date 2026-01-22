package handlers

import (
	"encoding/json"
	"net/http"

	"sync-photo-backend/internal/middleware"
	"sync-photo-backend/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// PairHandler handles pair-related HTTP requests
type PairHandler struct {
	pairService *services.PairService
	wsHub       *services.WSHub
}

// NewPairHandler creates a new pair handler
func NewPairHandler(pairService *services.PairService, wsHub *services.WSHub) *PairHandler {
	return &PairHandler{
		pairService: pairService,
		wsHub:       wsHub,
	}
}

// CreatePairRequest represents the request body for creating a pair
type CreatePairRequest struct {
	PartnerCode string `json:"partner_code"`
}

// CreatePair handles POST /api/v1/pairs
func (h *PairHandler) CreatePair(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	var req CreatePairRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate partner code
	if req.PartnerCode == "" {
		respondError(w, "partner_code is required", http.StatusBadRequest)
		return
	}

	if len(req.PartnerCode) != 6 {
		respondError(w, "partner_code must be 6 characters", http.StatusBadRequest)
		return
	}

	pair, err := h.pairService.CreatePair(ctx, userID, req.PartnerCode)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Str("partner_code", req.PartnerCode).
			Msg("Failed to create pair")

		statusCode := http.StatusInternalServerError
		if err.Error() == "partner not found" {
			statusCode = http.StatusNotFound
		} else if err.Error() == "cannot create pair with yourself" ||
			err.Error() == "user is already in a pair" ||
			err.Error() == "partner is already in a pair" {
			statusCode = http.StatusConflict
		}

		respondError(w, err.Error(), statusCode)
		return
	}

	log.Info().
		Str("user_id", userID).
		Str("partner_code", req.PartnerCode).
		Str("pair_id", pair.ID).
		Msg("Pair created")

	// Определить ID партнера
	partnerID := pair.UserBID
	if pair.UserBID == userID {
		partnerID = pair.UserAID
	}

	// Отправить уведомление обоим пользователям через WebSocket (если они онлайн)
	// Уведомление инициатору создания пары
	if h.wsHub.IsOnline(userID) {
		if err := h.wsHub.NotifyPairCreated(userID, pair.ID, pair.UserAID, pair.UserBID, pair.CreatedAt); err != nil {
			log.Error().
				Err(err).
				Str("user_id", userID).
				Msg("Failed to notify user about pair creation")
			// Не возвращаем ошибку, так как пара уже создана
		}
	}

	// Уведомление партнеру
	if h.wsHub.IsOnline(partnerID) {
		if err := h.wsHub.NotifyPairCreated(partnerID, pair.ID, pair.UserAID, pair.UserBID, pair.CreatedAt); err != nil {
			log.Error().
				Err(err).
				Str("partner_id", partnerID).
				Msg("Failed to notify partner about pair creation")
			// Не возвращаем ошибку, так как пара уже создана
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(pair)
}

// DeletePair handles DELETE /api/v1/pairs/:pair_id
func (h *PairHandler) DeletePair(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)
	pairID := chi.URLParam(r, "pair_id")

	if pairID == "" {
		respondError(w, "pair_id is required", http.StatusBadRequest)
		return
	}

	// Получить информацию о паре для определения партнера перед удалением
	pair, err := h.pairService.GetPairByID(ctx, pairID)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Str("pair_id", pairID).
			Msg("Failed to get pair")

		statusCode := http.StatusInternalServerError
		if err.Error() == "pair not found" {
			statusCode = http.StatusNotFound
		}

		respondError(w, err.Error(), statusCode)
		return
	}

	// Проверить, что пользователь является членом пары
	if pair.UserAID != userID && pair.UserBID != userID {
		respondError(w, "user is not a member of this pair", http.StatusForbidden)
		return
	}

	// Определить ID партнера
	partnerID := pair.UserAID
	if pair.UserAID == userID {
		partnerID = pair.UserBID
	}

	// Удалить пару
	err = h.pairService.DeletePair(ctx, pairID, userID)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Str("pair_id", pairID).
			Msg("Failed to delete pair")

		statusCode := http.StatusInternalServerError
		if err.Error() == "pair not found" {
			statusCode = http.StatusNotFound
		} else if err.Error() == "user is not a member of this pair" {
			statusCode = http.StatusForbidden
		}

		respondError(w, err.Error(), statusCode)
		return
	}

	log.Info().
		Str("user_id", userID).
		Str("pair_id", pairID).
		Msg("Pair deleted")

	// Отправить уведомление обоим пользователям через WebSocket (если они онлайн)
	// Уведомление инициатору удаления пары
	if h.wsHub.IsOnline(userID) {
		if err := h.wsHub.NotifyPairDeleted(userID); err != nil {
			log.Error().
				Err(err).
				Str("user_id", userID).
				Msg("Failed to notify user about pair deletion")
			// Не возвращаем ошибку, так как пара уже удалена
		}
	}

	// Уведомление партнеру
	if h.wsHub.IsOnline(partnerID) {
		if err := h.wsHub.NotifyPairDeleted(partnerID); err != nil {
			log.Error().
				Err(err).
				Str("partner_id", partnerID).
				Msg("Failed to notify partner about pair deletion")
			// Не возвращаем ошибку, так как пара уже удалена
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
