package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jeet-patel/subscription-commerce-backend/internal/cache"
	"github.com/jeet-patel/subscription-commerce-backend/internal/database"
	"github.com/jeet-patel/subscription-commerce-backend/internal/handlers"
)

var testDB *database.DB
var testRedis *cache.Redis

func setupTest(t *testing.T) func() {
	var err error

	// Connect to test database
	testDB, err = database.New()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Connect to Redis
	testRedis, err = cache.NewRedis()
	if err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Clean up tables
	testDB.Exec("DELETE FROM transactions")
	testDB.Exec("DELETE FROM gifts")
	testDB.Exec("DELETE FROM subscriptions")
	testDB.Exec("DELETE FROM users")

	// Create test users
	testDB.Exec("INSERT INTO users (id, email) VALUES (100, 'testuser@test.com')")
	testDB.Exec("INSERT INTO users (id, email) VALUES (101, 'recipient@test.com')")

	return func() {
		testDB.Exec("DELETE FROM transactions")
		testDB.Exec("DELETE FROM gifts")
		testDB.Exec("DELETE FROM subscriptions")
		testDB.Exec("DELETE FROM users WHERE id IN (100, 101)")
		testDB.Close()
		testRedis.Close()
	}
}

func TestSubscribeHappyPath(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	handler := handlers.NewSubscriptionHandler(testDB)

	body := `{"user_id": 100, "plan": "monthly", "duration_months": 1}`
	req := httptest.NewRequest(http.MethodPost, "/subscribe", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-sub-001")

	rr := httptest.NewRecorder()
	handler.Subscribe(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &response)

	if response["status"] != "active" {
		t.Errorf("Expected status 'active', got %v", response["status"])
	}

	if response["user_id"].(float64) != 100 {
		t.Errorf("Expected user_id 100, got %v", response["user_id"])
	}
}

func TestSubscribeDuplicateUser(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	handler := handlers.NewSubscriptionHandler(testDB)

	// First subscription
	body := `{"user_id": 100, "plan": "monthly", "duration_months": 1}`
	req := httptest.NewRequest(http.MethodPost, "/subscribe", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-sub-002")

	rr := httptest.NewRecorder()
	handler.Subscribe(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("First subscription failed: %s", rr.Body.String())
	}

	// Try duplicate subscription
	req2 := httptest.NewRequest(http.MethodPost, "/subscribe", bytes.NewBufferString(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", "test-sub-003")

	rr2 := httptest.NewRecorder()
	handler.Subscribe(rr2, req2)

	if rr2.Code != http.StatusConflict {
		t.Errorf("Expected status 409 Conflict, got %d: %s", rr2.Code, rr2.Body.String())
	}
}

func TestRenewHappyPath(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	handler := handlers.NewSubscriptionHandler(testDB)

	// Create subscription first
	body := `{"user_id": 100, "plan": "monthly", "duration_months": 1}`
	req := httptest.NewRequest(http.MethodPost, "/subscribe", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-sub-004")

	rr := httptest.NewRecorder()
	handler.Subscribe(rr, req)

	var subResponse map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &subResponse)
	subID := int(subResponse["id"].(float64))

	// Renew subscription
	renewBody := `{"subscription_id": ` + string(rune(subID+'0')) + `, "duration_months": 1}`
	renewReq := httptest.NewRequest(http.MethodPost, "/renew", bytes.NewBufferString(renewBody))
	renewReq.Header.Set("Content-Type", "application/json")
	renewReq.Header.Set("Idempotency-Key", "test-renew-001")

	renewRR := httptest.NewRecorder()
	handler.Renew(renewRR, renewReq)

	if renewRR.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", renewRR.Code, renewRR.Body.String())
	}
}

func TestCancelHappyPath(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	handler := handlers.NewSubscriptionHandler(testDB)

	// Create subscription first
	body := `{"user_id": 100, "plan": "monthly", "duration_months": 1}`
	req := httptest.NewRequest(http.MethodPost, "/subscribe", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-sub-005")

	rr := httptest.NewRecorder()
	handler.Subscribe(rr, req)

	var subResponse map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &subResponse)
	subID := int(subResponse["id"].(float64))

	// Cancel subscription
	cancelBody := `{"subscription_id": ` + string(rune(subID+'0')) + `}`
	cancelReq := httptest.NewRequest(http.MethodPost, "/cancel", bytes.NewBufferString(cancelBody))
	cancelReq.Header.Set("Content-Type", "application/json")
	cancelReq.Header.Set("Idempotency-Key", "test-cancel-001")

	cancelRR := httptest.NewRecorder()
	handler.Cancel(cancelRR, cancelReq)

	if cancelRR.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", cancelRR.Code, cancelRR.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(cancelRR.Body.Bytes(), &response)

	if response["status"] != "cancelled" {
		t.Errorf("Expected status 'cancelled', got %v", response["status"])
	}
}

func TestGiftHappyPath(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	handler := handlers.NewGiftHandler(testDB)

	body := `{"gifter_id": 100, "recipient_email": "friend@test.com", "duration_months": 3}`
	req := httptest.NewRequest(http.MethodPost, "/gift", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-gift-001")

	rr := httptest.NewRecorder()
	handler.CreateGift(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &response)

	if response["status"] != "pending" {
		t.Errorf("Expected status 'pending', got %v", response["status"])
	}

	if response["duration_months"].(float64) != 3 {
		t.Errorf("Expected duration_months 3, got %v", response["duration_months"])
	}
}

func TestMissingIdempotencyKey(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	handler := handlers.NewSubscriptionHandler(testDB)

	body := `{"user_id": 100, "plan": "monthly", "duration_months": 1}`
	req := httptest.NewRequest(http.MethodPost, "/subscribe", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	// No Idempotency-Key header

	rr := httptest.NewRecorder()
	handler.Subscribe(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestSubscribeUserNotFound(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	handler := handlers.NewSubscriptionHandler(testDB)

	body := `{"user_id": 9999, "plan": "monthly", "duration_months": 1}`
	req := httptest.NewRequest(http.MethodPost, "/subscribe", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-sub-notfound")

	rr := httptest.NewRecorder()
	handler.Subscribe(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestResponseTime(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	handler := handlers.NewSubscriptionHandler(testDB)

	body := `{"user_id": 100, "plan": "monthly", "duration_months": 1}`
	req := httptest.NewRequest(http.MethodPost, "/subscribe", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-perf-001")

	start := time.Now()
	rr := httptest.NewRecorder()
	handler.Subscribe(rr, req)
	elapsed := time.Since(start)

	if elapsed > 150*time.Millisecond {
		t.Errorf("Response time %v exceeded 150ms target", elapsed)
	}

	t.Logf("Subscribe response time: %v", elapsed)
}
