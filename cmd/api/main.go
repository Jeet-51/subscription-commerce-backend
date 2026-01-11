package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jeet-patel/subscription-commerce-backend/internal/cache"
	"github.com/jeet-patel/subscription-commerce-backend/internal/database"
	"github.com/jeet-patel/subscription-commerce-backend/internal/handlers"
	"github.com/jeet-patel/subscription-commerce-backend/internal/middleware"
)

var db *database.DB
var redisClient *cache.Redis

type HealthResponse struct {
	Status    string `json:"status"`
	Database  string `json:"database"`
	Redis     string `json:"redis"`
	Timestamp string `json:"timestamp"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	dbStatus := "connected"
	if err := db.Ping(); err != nil {
		dbStatus = "disconnected"
	}

	redisStatus := "connected"
	if err := redisClient.Ping(); err != nil {
		redisStatus = "disconnected"
	}

	response := HealthResponse{
		Status:    "healthy",
		Database:  dbStatus,
		Redis:     redisStatus,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func main() {
	var err error

	// Connect to database
	db, err = database.New()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	err = db.RunMigrations("internal/database/migrations")
	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Connect to Redis
	redisClient, err = cache.NewRedis()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize handlers
	subHandler := handlers.NewSubscriptionHandler(db)
	giftHandler := handlers.NewGiftHandler(db)

	// Setup routes
	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/health", healthHandler)

	// Subscription endpoints
	mux.HandleFunc("/subscribe", subHandler.Subscribe)
	mux.HandleFunc("/renew", subHandler.Renew)
	mux.HandleFunc("/cancel", subHandler.Cancel)
	mux.HandleFunc("/subscriptions/", subHandler.GetUserSubscriptions)

	// Gift endpoints
	mux.HandleFunc("/gift", giftHandler.CreateGift)
	mux.HandleFunc("/gift/redeem", giftHandler.RedeemGift)

	// Apply middleware
	handler := middleware.RateLimiter(redisClient)(
		middleware.Idempotency(redisClient)(mux),
	)

	// Custom handler to skip idempotency for GET requests
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || strings.HasPrefix(r.URL.Path, "/health") {
			mux.ServeHTTP(w, r)
			return
		}
		handler.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:         ":8080",
		Handler:      finalHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("Starting server on :8080")
	log.Println("Available endpoints:")
	log.Println("  GET  /health")
	log.Println("  POST /subscribe")
	log.Println("  POST /renew")
	log.Println("  POST /cancel")
	log.Println("  POST /gift")
	log.Println("  POST /gift/redeem")
	log.Println("  GET  /subscriptions/{user_id}")

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
