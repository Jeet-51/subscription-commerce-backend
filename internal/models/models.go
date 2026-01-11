package models

import "time"

// User represents a user in the system
type User struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SubscriptionStatus represents valid subscription states
type SubscriptionStatus string

const (
	StatusActive    SubscriptionStatus = "active"
	StatusCancelled SubscriptionStatus = "cancelled"
	StatusExpired   SubscriptionStatus = "expired"
	StatusPending   SubscriptionStatus = "pending"
)

// Subscription represents a user subscription
type Subscription struct {
	ID          int                `json:"id"`
	UserID      int                `json:"user_id"`
	Status      SubscriptionStatus `json:"status"`
	StartDate   time.Time          `json:"start_date"`
	EndDate     time.Time          `json:"end_date"`
	CancelledAt *time.Time         `json:"cancelled_at,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

// GiftStatus represents valid gift states
type GiftStatus string

const (
	GiftPending  GiftStatus = "pending"
	GiftRedeemed GiftStatus = "redeemed"
	GiftExpired  GiftStatus = "expired"
)

// Gift represents a subscription gift
type Gift struct {
	ID             int        `json:"id"`
	GifterID       int        `json:"gifter_id"`
	RecipientEmail string     `json:"recipient_email"`
	RecipientID    *int       `json:"recipient_id,omitempty"`
	Status         GiftStatus `json:"status"`
	DurationMonths int        `json:"duration_months"`
	RedeemedAt     *time.Time `json:"redeemed_at,omitempty"`
	ExpiresAt      time.Time  `json:"expires_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

// Transaction represents an idempotent operation record
type Transaction struct {
	ID             int       `json:"id"`
	IdempotencyKey string    `json:"idempotency_key"`
	OperationType  string    `json:"operation_type"`
	EntityType     string    `json:"entity_type"`
	EntityID       int       `json:"entity_id"`
	Metadata       string    `json:"metadata,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// API Request/Response types

type SubscribeRequest struct {
	UserID         int    `json:"user_id"`
	Plan           string `json:"plan"`
	DurationMonths int    `json:"duration_months"`
}

type RenewRequest struct {
	SubscriptionID int `json:"subscription_id"`
	DurationMonths int `json:"duration_months"`
}

type CancelRequest struct {
	SubscriptionID int `json:"subscription_id"`
}

type GiftRequest struct {
	GifterID       int    `json:"gifter_id"`
	RecipientEmail string `json:"recipient_email"`
	DurationMonths int    `json:"duration_months"`
}

type RedeemGiftRequest struct {
	GiftID int `json:"gift_id"`
	UserID int `json:"user_id"`
}
