package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jeet-patel/subscription-commerce-backend/internal/models"
)

// CreateUser creates a new user
func (db *DB) CreateUser(email string) (*models.User, error) {
	var user models.User
	err := db.QueryRow(
		`INSERT INTO users (email) VALUES ($1) 
		 RETURNING id, email, created_at, updated_at`,
		email,
	).Scan(&user.ID, &user.Email, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &user, nil
}

// GetUserByID retrieves a user by ID
func (db *DB) GetUserByID(id int) (*models.User, error) {
	var user models.User
	err := db.QueryRow(
		`SELECT id, email, created_at, updated_at FROM users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Email, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (db *DB) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := db.QueryRow(
		`SELECT id, email, created_at, updated_at FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// GetActiveSubscription retrieves active subscription for a user
func (db *DB) GetActiveSubscription(userID int) (*models.Subscription, error) {
	var sub models.Subscription
	err := db.QueryRow(
		`SELECT id, user_id, status, start_date, end_date, cancelled_at, created_at, updated_at 
		 FROM subscriptions 
		 WHERE user_id = $1 AND status = 'active'`,
		userID,
	).Scan(&sub.ID, &sub.UserID, &sub.Status, &sub.StartDate, &sub.EndDate,
		&sub.CancelledAt, &sub.CreatedAt, &sub.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}
	return &sub, nil
}

// GetSubscriptionByID retrieves a subscription by ID
func (db *DB) GetSubscriptionByID(id int) (*models.Subscription, error) {
	var sub models.Subscription
	err := db.QueryRow(
		`SELECT id, user_id, status, start_date, end_date, cancelled_at, created_at, updated_at 
		 FROM subscriptions 
		 WHERE id = $1`,
		id,
	).Scan(&sub.ID, &sub.UserID, &sub.Status, &sub.StartDate, &sub.EndDate,
		&sub.CancelledAt, &sub.CreatedAt, &sub.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}
	return &sub, nil
}

// GetUserSubscriptions retrieves all subscriptions for a user
func (db *DB) GetUserSubscriptions(userID int) ([]models.Subscription, error) {
	rows, err := db.Query(
		`SELECT id, user_id, status, start_date, end_date, cancelled_at, created_at, updated_at 
		 FROM subscriptions 
		 WHERE user_id = $1 
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscriptions: %w", err)
	}
	defer rows.Close()

	var subscriptions []models.Subscription
	for rows.Next() {
		var sub models.Subscription
		err := rows.Scan(&sub.ID, &sub.UserID, &sub.Status, &sub.StartDate, &sub.EndDate,
			&sub.CancelledAt, &sub.CreatedAt, &sub.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subscription: %w", err)
		}
		subscriptions = append(subscriptions, sub)
	}
	return subscriptions, nil
}

// CreateSubscriptionTx creates a subscription within a transaction
func (db *DB) CreateSubscriptionTx(tx *sql.Tx, userID int, durationMonths int, idempotencyKey string) (*models.Subscription, error) {
	startDate := time.Now()
	endDate := startDate.AddDate(0, durationMonths, 0)

	var sub models.Subscription
	err := tx.QueryRow(
		`INSERT INTO subscriptions (user_id, status, start_date, end_date) 
		 VALUES ($1, 'active', $2, $3) 
		 RETURNING id, user_id, status, start_date, end_date, cancelled_at, created_at, updated_at`,
		userID, startDate, endDate,
	).Scan(&sub.ID, &sub.UserID, &sub.Status, &sub.StartDate, &sub.EndDate,
		&sub.CancelledAt, &sub.CreatedAt, &sub.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	// Record transaction
	_, err = tx.Exec(
		`INSERT INTO transactions (idempotency_key, operation_type, entity_type, entity_id) 
		 VALUES ($1, 'create', 'subscription', $2)`,
		idempotencyKey, sub.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to record transaction: %w", err)
	}

	return &sub, nil
}

// RenewSubscriptionTx renews a subscription within a transaction
func (db *DB) RenewSubscriptionTx(tx *sql.Tx, subscriptionID int, durationMonths int, idempotencyKey string) (*models.Subscription, error) {
	var sub models.Subscription
	err := tx.QueryRow(
		`UPDATE subscriptions 
		 SET end_date = end_date + interval '1 month' * $1, updated_at = NOW() 
		 WHERE id = $2 AND status = 'active'
		 RETURNING id, user_id, status, start_date, end_date, cancelled_at, created_at, updated_at`,
		durationMonths, subscriptionID,
	).Scan(&sub.ID, &sub.UserID, &sub.Status, &sub.StartDate, &sub.EndDate,
		&sub.CancelledAt, &sub.CreatedAt, &sub.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to renew subscription: %w", err)
	}

	// Record transaction
	_, err = tx.Exec(
		`INSERT INTO transactions (idempotency_key, operation_type, entity_type, entity_id) 
		 VALUES ($1, 'renew', 'subscription', $2)`,
		idempotencyKey, sub.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to record transaction: %w", err)
	}

	return &sub, nil
}

// CancelSubscriptionTx cancels a subscription within a transaction
func (db *DB) CancelSubscriptionTx(tx *sql.Tx, subscriptionID int, idempotencyKey string) (*models.Subscription, error) {
	var sub models.Subscription
	err := tx.QueryRow(
		`UPDATE subscriptions 
		 SET status = 'cancelled', cancelled_at = NOW(), updated_at = NOW() 
		 WHERE id = $1 AND status = 'active'
		 RETURNING id, user_id, status, start_date, end_date, cancelled_at, created_at, updated_at`,
		subscriptionID,
	).Scan(&sub.ID, &sub.UserID, &sub.Status, &sub.StartDate, &sub.EndDate,
		&sub.CancelledAt, &sub.CreatedAt, &sub.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to cancel subscription: %w", err)
	}

	// Record transaction
	_, err = tx.Exec(
		`INSERT INTO transactions (idempotency_key, operation_type, entity_type, entity_id) 
		 VALUES ($1, 'cancel', 'subscription', $2)`,
		idempotencyKey, sub.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to record transaction: %w", err)
	}

	return &sub, nil
}

// GetGiftByID retrieves a gift by ID
func (db *DB) GetGiftByID(id int) (*models.Gift, error) {
	var gift models.Gift
	err := db.QueryRow(
		`SELECT id, gifter_id, recipient_email, recipient_id, status, duration_months, redeemed_at, expires_at, created_at 
		 FROM gifts 
		 WHERE id = $1`,
		id,
	).Scan(&gift.ID, &gift.GifterID, &gift.RecipientEmail, &gift.RecipientID,
		&gift.Status, &gift.DurationMonths, &gift.RedeemedAt, &gift.ExpiresAt, &gift.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get gift: %w", err)
	}
	return &gift, nil
}

// CreateGiftTx creates a gift within a transaction
func (db *DB) CreateGiftTx(tx *sql.Tx, gifterID int, recipientEmail string, durationMonths int, idempotencyKey string) (*models.Gift, error) {
	expiresAt := time.Now().AddDate(0, 0, 30) // Gift expires in 30 days

	var gift models.Gift
	err := tx.QueryRow(
		`INSERT INTO gifts (gifter_id, recipient_email, status, duration_months, expires_at) 
		 VALUES ($1, $2, 'pending', $3, $4) 
		 RETURNING id, gifter_id, recipient_email, recipient_id, status, duration_months, redeemed_at, expires_at, created_at`,
		gifterID, recipientEmail, durationMonths, expiresAt,
	).Scan(&gift.ID, &gift.GifterID, &gift.RecipientEmail, &gift.RecipientID,
		&gift.Status, &gift.DurationMonths, &gift.RedeemedAt, &gift.ExpiresAt, &gift.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create gift: %w", err)
	}

	// Record transaction
	_, err = tx.Exec(
		`INSERT INTO transactions (idempotency_key, operation_type, entity_type, entity_id) 
		 VALUES ($1, 'create', 'gift', $2)`,
		idempotencyKey, gift.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to record transaction: %w", err)
	}

	return &gift, nil
}

// RedeemGiftTx redeems a gift and creates subscription within a transaction
func (db *DB) RedeemGiftTx(tx *sql.Tx, giftID int, userID int, idempotencyKey string) (*models.Subscription, *models.Gift, error) {
	// Update gift status
	var gift models.Gift
	err := tx.QueryRow(
		`UPDATE gifts 
		 SET status = 'redeemed', recipient_id = $1, redeemed_at = NOW() 
		 WHERE id = $2 AND status = 'pending' AND expires_at > NOW()
		 RETURNING id, gifter_id, recipient_email, recipient_id, status, duration_months, redeemed_at, expires_at, created_at`,
		userID, giftID,
	).Scan(&gift.ID, &gift.GifterID, &gift.RecipientEmail, &gift.RecipientID,
		&gift.Status, &gift.DurationMonths, &gift.RedeemedAt, &gift.ExpiresAt, &gift.CreatedAt)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to redeem gift: %w", err)
	}

	// Create subscription for recipient
	startDate := time.Now()
	endDate := startDate.AddDate(0, gift.DurationMonths, 0)

	var sub models.Subscription
	err = tx.QueryRow(
		`INSERT INTO subscriptions (user_id, status, start_date, end_date) 
		 VALUES ($1, 'active', $2, $3) 
		 RETURNING id, user_id, status, start_date, end_date, cancelled_at, created_at, updated_at`,
		userID, startDate, endDate,
	).Scan(&sub.ID, &sub.UserID, &sub.Status, &sub.StartDate, &sub.EndDate,
		&sub.CancelledAt, &sub.CreatedAt, &sub.UpdatedAt)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create subscription from gift: %w", err)
	}

	// Record transaction
	_, err = tx.Exec(
		`INSERT INTO transactions (idempotency_key, operation_type, entity_type, entity_id, metadata) 
		 VALUES ($1, 'redeem', 'gift', $2, $3)`,
		idempotencyKey, gift.ID, fmt.Sprintf(`{"subscription_id": %d}`, sub.ID),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to record transaction: %w", err)
	}

	return &sub, &gift, nil
}

// BeginTx starts a new database transaction
func (db *DB) BeginTx() (*sql.Tx, error) {
	return db.Begin()
}
