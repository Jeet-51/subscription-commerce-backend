package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jeet-patel/subscription-commerce-backend/internal/cache"
)

const (
	IdempotencyKeyHeader = "Idempotency-Key"
	IdempotencyTTL       = 24 * time.Hour
)

type cachedResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           &bytes.Buffer{},
	}
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func Idempotency(redisClient *cache.Redis) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only apply to POST, PUT, DELETE
			if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodDelete {
				next.ServeHTTP(w, r)
				return
			}

			idempotencyKey := r.Header.Get(IdempotencyKeyHeader)
			if idempotencyKey == "" {
				http.Error(w, `{"error":"Idempotency-Key header is required"}`, http.StatusBadRequest)
				return
			}

			cacheKey := "idempotency:" + idempotencyKey

			// Check if we have a cached response
			cached, err := redisClient.Get(cacheKey)
			if err == nil && cached != "" {
				var resp cachedResponse
				if err := json.Unmarshal([]byte(cached), &resp); err == nil {
					for k, v := range resp.Headers {
						w.Header().Set(k, v)
					}
					w.Header().Set("X-Idempotency-Replayed", "true")
					w.WriteHeader(resp.StatusCode)
					w.Write([]byte(resp.Body))
					return
				}
			}

			// Record the response
			recorder := newResponseRecorder(w)
			next.ServeHTTP(recorder, r)

			// Cache the response
			resp := cachedResponse{
				StatusCode: recorder.statusCode,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       recorder.body.String(),
			}

			respJSON, err := json.Marshal(resp)
			if err == nil {
				redisClient.Set(cacheKey, string(respJSON), IdempotencyTTL)
			}
		})
	}
}
