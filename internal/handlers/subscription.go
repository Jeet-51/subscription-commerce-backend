package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/jeet-patel/subscription-commerce-backend/internal/database"
	"github.com/jeet-patel/subscription-commerce-backend/internal/models"
)

type SubscriptionHandler struct {
	db *database.DB
}

func NewSubscriptionHandler(db *database.DB) *SubscriptionHandler {
	return &SubscriptionHandler{db: db}
}

// Subscribe handles POST /subscribe
func (h *SubscriptionHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		writeError(w, http.StatusBadRequest, "Idempotency-Key header is required")
		return
	}

	var req models.SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.UserID <= 0 {
		writeError(w, http.StatusBadRequest, "Valid user_id is required")
		return
	}

	if req.DurationMonths <= 0 {
		req.DurationMonths = 1 // Default to 1 month
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

	// Check for existing active subscription
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

	// Create subscription
	sub, err := h.db.CreateSubscriptionTx(tx, req.UserID, req.DurationMonths, idempotencyKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create subscription")
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	writeJSON(w, http.StatusCreated, sub)
}

// Renew handles POST /renew
func (h *SubscriptionHandler) Renew(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		writeError(w, http.StatusBadRequest, "Idempotency-Key header is required")
		return
	}

	var req models.RenewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.SubscriptionID <= 0 {
		writeError(w, http.StatusBadRequest, "Valid subscription_id is required")
		return
	}

	if req.DurationMonths <= 0 {
		req.DurationMonths = 1
	}

	// Check if subscription exists and is active
	existing, err := h.db.GetSubscriptionByID(req.SubscriptionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "Subscription not found")
		return
	}
	if existing.Status != models.StatusActive {
		writeError(w, http.StatusConflict, "Subscription is not active")
		return
	}

	// Begin transaction
	tx, err := h.db.BeginTx()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Renew subscription
	sub, err := h.db.RenewSubscriptionTx(tx, req.SubscriptionID, req.DurationMonths, idempotencyKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to renew subscription")
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	writeJSON(w, http.StatusOK, sub)
}

// Cancel handles POST /cancel
func (h *SubscriptionHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		writeError(w, http.StatusBadRequest, "Idempotency-Key header is required")
		return
	}

	var req models.CancelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.SubscriptionID <= 0 {
		writeError(w, http.StatusBadRequest, "Valid subscription_id is required")
		return
	}

	// Check if subscription exists and is active
	existing, err := h.db.GetSubscriptionByID(req.SubscriptionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "Subscription not found")
		return
	}
	if existing.Status != models.StatusActive {
		writeError(w, http.StatusConflict, "Subscription is not active")
		return
	}

	// Begin transaction
	tx, err := h.db.BeginTx()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	// Cancel subscription
	sub, err := h.db.CancelSubscriptionTx(tx, req.SubscriptionID, idempotencyKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to cancel subscription")
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	writeJSON(w, http.StatusOK, sub)
}

// GetUserSubscriptions handles GET /subscriptions/{user_id}
func (h *SubscriptionHandler) GetUserSubscriptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract user_id from path
	path := strings.TrimPrefix(r.URL.Path, "/subscriptions/")
	userID, err := strconv.Atoi(path)
	if err != nil || userID <= 0 {
		writeError(w, http.StatusBadRequest, "Valid user_id is required")
		return
	}

	// Get subscriptions
	subs, err := h.db.GetUserSubscriptions(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}

	response := map[string]interface{}{
		"user_id":       userID,
		"subscriptions": subs,
	}

	writeJSON(w, http.StatusOK, response)
}

// Helper functions
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
