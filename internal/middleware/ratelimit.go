package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/jeet-patel/subscription-commerce-backend/internal/cache"
)

const (
	RateLimit       = 10              // requests per window
	RateLimitWindow = 1 * time.Minute // window duration
)

func RateLimiter(redisClient *cache.Redis) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use IP address as identifier (in production, use user ID)
			clientIP := r.RemoteAddr

			key := fmt.Sprintf("ratelimit:%s", clientIP)

			// Increment request count
			count, err := redisClient.Incr(key)
			if err != nil {
				// If Redis fails, allow the request
				next.ServeHTTP(w, r)
				return
			}

			// Set expiry on first request
			if count == 1 {
				redisClient.Expire(key, RateLimitWindow)
			}

			// Check if over limit
			if count > RateLimit {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", RateLimit))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"Rate limit exceeded. Try again later."}`))
				return
			}

			// Add rate limit headers
			remaining := RateLimit - int(count)
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", RateLimit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

			next.ServeHTTP(w, r)
		})
	}
}
