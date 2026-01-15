package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"sync-photo-backend/internal/middleware"
	"sync-photo-backend/internal/services"

	"github.com/rs/zerolog/log"
)

// PhotoHandler handles photo-related HTTP requests
type PhotoHandler struct {
	photoService *services.PhotoService
}

// NewPhotoHandler creates a new photo handler
func NewPhotoHandler(photoService *services.PhotoService) *PhotoHandler {
	return &PhotoHandler{
		photoService: photoService,
	}
}

// GetPhotos handles GET /api/v1/photos
func (h *PhotoHandler) GetPhotos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	// Parse query parameters
	limit := 50
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil {
			offset = parsedOffset
		}
	}

	photos, total, err := h.photoService.GetPhotosByPair(ctx, userID, limit, offset)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Msg("Failed to get photos")

		statusCode := http.StatusInternalServerError
		if err.Error() == "user is not in a pair" {
			statusCode = http.StatusNotFound
		}

		respondError(w, err.Error(), statusCode)
		return
	}

	response := map[string]interface{}{
		"photos": photos,
		"total":  total,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// UploadPhoto handles POST /api/v1/photos/upload
func (h *PhotoHandler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	var req services.UploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Filename == "" {
		respondError(w, "filename is required", http.StatusBadRequest)
		return
	}

	if req.ContentType == "" {
		req.ContentType = "image/jpeg" // Default
	}

	response, err := h.photoService.GetPreSignedURL(ctx, userID, req.Filename, req.ContentType)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", userID).
			Str("filename", req.Filename).
			Msg("Failed to generate pre-signed URL")

		statusCode := http.StatusInternalServerError
		if err.Error() == "user is not in a pair" {
			statusCode = http.StatusNotFound
		}

		respondError(w, err.Error(), statusCode)
		return
	}

	log.Info().
		Str("user_id", userID).
		Str("photo_id", response.PhotoID).
		Str("filename", req.Filename).
		Msg("Pre-signed URL generated")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
