package models

import "time"

// User represents a user in the system
type User struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Token     string    `json:"token"`
	PushToken *string   `json:"push_token,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Pair represents a pair of users
type Pair struct {
	ID        string    `json:"id"`
	UserAID   string    `json:"user_a_id"`
	UserBID   string    `json:"user_b_id"`
	CreatedAt time.Time `json:"created_at"`
}

// Photo represents a photo taken by a user in a pair
type Photo struct {
	ID        string    `json:"id"`
	PairID    string    `json:"pair_id"`
	UserID    string    `json:"user_id"`
	S3URL     string    `json:"s3_url"`
	TakenAt   time.Time `json:"taken_at"`
	CreatedAt time.Time `json:"created_at"`
}
