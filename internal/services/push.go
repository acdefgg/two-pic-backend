package services

import (
	"fmt"

	"sync-photo-backend/internal/config"

	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/payload"
	"github.com/sideshow/apns2/token"

	"github.com/rs/zerolog/log"
)

// PushService handles sending push notifications via APNs
type PushService struct {
	client   *apns2.Client
	bundleID string
}

// NewPushService creates a new push service with token-based APNs auth
func NewPushService(cfg config.APNsConfig) (*PushService, error) {
	authKey, err := token.AuthKeyFromFile(cfg.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load APNs auth key from %s: %w", cfg.KeyPath, err)
	}

	tkn := &token.Token{
		AuthKey: authKey,
		KeyID:   cfg.KeyID,
		TeamID:  cfg.TeamID,
	}

	var client *apns2.Client
	if cfg.Production {
		client = apns2.NewTokenClient(tkn).Production()
	} else {
		client = apns2.NewTokenClient(tkn).Development()
	}

	log.Info().
		Bool("production", cfg.Production).
		Str("bundle_id", cfg.BundleID).
		Msg("APNs push service initialized")

	return &PushService{
		client:   client,
		bundleID: cfg.BundleID,
	}, nil
}

// SendCallNotification sends a "partner is calling" push notification
func (s *PushService) SendCallNotification(pushToken string) error {
	p := payload.NewPayload().
		AlertTitle("TwoPic").
		AlertBody("Тебя ждут в TwoPic! Открой приложение 📸").
		Sound("default").
		MutableContent()

	notification := &apns2.Notification{
		DeviceToken: pushToken,
		Topic:       s.bundleID,
		Payload:     p,
	}

	res, err := s.client.Push(notification)
	if err != nil {
		return fmt.Errorf("failed to send push notification: %w", err)
	}

	if !res.Sent() {
		log.Warn().
			Int("status", res.StatusCode).
			Str("reason", res.Reason).
			Str("device_token", pushToken[:min(10, len(pushToken))]+"...").
			Msg("APNs push not sent")
		return fmt.Errorf("push notification not sent: %s (status %d)", res.Reason, res.StatusCode)
	}

	log.Debug().
		Str("apns_id", res.ApnsID).
		Msg("Push notification sent successfully")

	return nil
}
