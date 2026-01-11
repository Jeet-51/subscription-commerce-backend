package main

import (
	"bytes"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"
)

type Stats struct {
	TotalRequests  int
	SuccessCount   int
	ErrorCount     int
	DuplicateCount int
	Latencies      []time.Duration
	StatusCodes    map[int]int
}

func main() {
	fmt.Println("=== Subscription Commerce Backend Load Test ===")
	fmt.Println()

	baseURL := "http://localhost:8080"

	// Test 1: Health endpoint performance
	fmt.Println("[Test 1] Health Endpoint Performance (100 requests)")
	healthStats := runHealthLoadTest(baseURL, 100, 10)
	printStats(healthStats)

	// Test 2: Idempotency validation
	fmt.Println("\n[Test 2] Idempotency Validation (20 retries with same key)")
	idempotencyStats := runIdempotencyTest(baseURL, 20)
	printIdempotencyStats(idempotencyStats)

	// Test 3: Mixed workload
	fmt.Println("\n[Test 3] Mixed Workload (50 requests)")
	mixedStats := runMixedLoadTest(baseURL, 50, 10)
	printStats(mixedStats)

	fmt.Println("\n=== Load Test Complete ===")
}

func runHealthLoadTest(baseURL string, totalRequests, concurrency int) *Stats {
	stats := &Stats{
		StatusCodes: make(map[int]int),
	}
	var mu sync.Mutex
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, concurrency)

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			start := time.Now()
			resp, err := http.Get(baseURL + "/health")
			duration := time.Since(start)

			mu.Lock()
			defer mu.Unlock()

			stats.TotalRequests++
			stats.Latencies = append(stats.Latencies, duration)

			if err != nil {
				stats.ErrorCount++
				return
			}
			defer resp.Body.Close()

			stats.StatusCodes[resp.StatusCode]++
			if resp.StatusCode == 200 {
				stats.SuccessCount++
			} else {
				stats.ErrorCount++
			}
		}()
	}

	wg.Wait()
	return stats
}

func runIdempotencyTest(baseURL string, retryCount int) *Stats {
	stats := &Stats{
		StatusCodes: make(map[int]int),
	}

	idempotencyKey := fmt.Sprintf("idem-test-%d", time.Now().UnixNano())
	body := `{"user_id":1,"plan":"monthly","duration_months":1}`

	var firstStatus int

	for i := 0; i < retryCount; i++ {
		start := time.Now()
		req, _ := http.NewRequest("POST", baseURL+"/subscribe", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		duration := time.Since(start)

		stats.TotalRequests++
		stats.Latencies = append(stats.Latencies, duration)

		if err != nil {
			stats.ErrorCount++
			continue
		}
		resp.Body.Close()

		stats.StatusCodes[resp.StatusCode]++

		if i == 0 {
			firstStatus = resp.StatusCode
			stats.SuccessCount++
		} else {
			if resp.StatusCode == firstStatus {
				stats.DuplicateCount++
				stats.SuccessCount++
			} else {
				stats.ErrorCount++
			}
		}
	}

	return stats
}

func runMixedLoadTest(baseURL string, totalRequests, concurrency int) *Stats {
	stats := &Stats{
		StatusCodes: make(map[int]int),
	}
	var mu sync.Mutex
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, concurrency)

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func(reqNum int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			start := time.Now()
			var resp *http.Response
			var err error

			// Mix: 70% health, 30% get subscriptions
			if reqNum%10 < 7 {
				resp, err = http.Get(baseURL + "/health")
			} else {
				resp, err = http.Get(baseURL + "/subscriptions/1")
			}

			duration := time.Since(start)

			mu.Lock()
			defer mu.Unlock()

			stats.TotalRequests++
			stats.Latencies = append(stats.Latencies, duration)

			if err != nil {
				stats.ErrorCount++
				return
			}
			defer resp.Body.Close()

			stats.StatusCodes[resp.StatusCode]++
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				stats.SuccessCount++
			} else {
				stats.ErrorCount++
			}
		}(i)
	}

	wg.Wait()
	return stats
}

func printStats(stats *Stats) {
	if len(stats.Latencies) == 0 {
		fmt.Println("  No data collected")
		return
	}

	sort.Slice(stats.Latencies, func(i, j int) bool {
		return stats.Latencies[i] < stats.Latencies[j]
	})

	p50 := stats.Latencies[len(stats.Latencies)*50/100]
	p95 := stats.Latencies[len(stats.Latencies)*95/100]
	p99 := stats.Latencies[len(stats.Latencies)*99/100]

	successRate := float64(stats.SuccessCount) / float64(stats.TotalRequests) * 100

	fmt.Printf("  Total Requests: %d\n", stats.TotalRequests)
	fmt.Printf("  Success: %d (%.1f%%)\n", stats.SuccessCount, successRate)
	fmt.Printf("  Errors: %d\n", stats.ErrorCount)
	fmt.Printf("  P50 Latency: %v\n", p50)
	fmt.Printf("  P95 Latency: %v\n", p95)
	fmt.Printf("  P99 Latency: %v\n", p99)

	if p50 < 150*time.Millisecond {
		fmt.Println("  ✅ P50 under 150ms target")
	} else {
		fmt.Println("  ❌ P50 exceeds 150ms target")
	}
}

func printIdempotencyStats(stats *Stats) {
	fmt.Printf("  Total Requests: %d\n", stats.TotalRequests)
	fmt.Printf("  First Request: 1\n")
	fmt.Printf("  Duplicate Responses: %d\n", stats.DuplicateCount)
	fmt.Printf("  Errors: %d\n", stats.ErrorCount)

	if stats.DuplicateCount == stats.TotalRequests-1 && stats.ErrorCount == 0 {
		fmt.Println("  ✅ Idempotency Working - No duplicate operations!")
	} else {
		fmt.Println("  ⚠️  Check idempotency behavior")
	}
}
