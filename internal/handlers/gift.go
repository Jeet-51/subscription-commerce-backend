package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jeet-patel/subscription-commerce-backend/internal/database"
	"github.com/jeet-patel/subscription-commerce-backend/internal/models"
)

type GiftHandler struct {
	db *database.DB
}

func NewGiftHandler(db *database.DB) *GiftHandler {
	return &GiftHandler{db: db}
}

// CreateGift handles POST /gift
func (h *GiftHandler) CreateGift(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		writeError(w, http.StatusBadRequest, "Idempotency-Key header is required")
		return
	}

	var req models.GiftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.GifterID <= 0 {
		writeError(w, http.StatusBadRequest, "Valid gifter_id is required")
		return
	}

	if req.RecipientEmail == "" {
		writeError(w, http.StatusBadRequest, "recipient_email is required")
		return
	}

	if req.DurationMonths <= 0 {
		req.DurationMonths = 1
	}

	// Check if gifter exists
	gifter, err := h.db.GetUserByID(req.GifterID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if gifter == nil {
		writeError(w, http.StatusNotFound, "Gifter not found")
		return
	}

	// Begin transaction
	tx, err := h.db.BeginTx()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Create gift
	gift, err := h.db.CreateGiftTx(tx, req.GifterID, req.RecipientEmail, req.DurationMonths, idempotencyKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create gift")
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	writeJSON(w, http.StatusCreated, gift)
}

// RedeemGift handles POST /gift/redeem
func (h *GiftHandler) RedeemGift(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		writeError(w, http.StatusBadRequest, "Idempotency-Key header is required")
		return
	}

	var req models.RedeemGiftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.GiftID <= 0 {
		writeError(w, http.StatusBadRequest, "Valid gift_id is required")
		return
	}

	if req.UserID <= 0 {
		writeError(w, http.StatusBadRequest, "Valid user_id is required")
		return
	}

	// Check if user exists
	user, err := h.db.GetUserByID(req.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	// Check if gift exists and is pending
	gift, err := h.db.GetGiftByID(req.GiftID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if gift == nil {
		writeError(w, http.StatusNotFound, "Gift not found")
		return
	}
	if gift.Status != models.GiftPending {
		writeError(w, http.StatusConflict, "Gift is not available for redemption")
		return
	}

	// Check if user already has active subscription
	existing, err := h.db.GetActiveSubscription(req.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if existing != nil {
		writeError(w, http.StatusConflict, "User already has an active subscription")
		return
	}

	// Begin transaction
	tx, err := h.db.BeginTx()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Redeem gift
	sub, redeemedGift, err := h.db.RedeemGiftTx(tx, req.GiftID, req.UserID, idempotencyKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to redeem gift")
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	response := map[string]interface{}{
		"subscription_id": sub.ID,
		"gift_id":         redeemedGift.ID,
		"status":          redeemedGift.Status,
		"start_date":      sub.StartDate,
		"end_date":        sub.EndDate,
	}

	writeJSON(w, http.StatusOK, response)
}
