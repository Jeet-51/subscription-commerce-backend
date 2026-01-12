# Subscription Commerce Backend

A production-grade backend service demonstrating subscription and gifting workflows with emphasis on **correctness**, **idempotency**, and **transactional safety**.

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://golang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?logo=postgresql)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D?logo=redis)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker)](https://www.docker.com/)

📝 **[Read the detailed blog post on Medium](https://medium.com/@jeetp5118/i-built-a-subscription-backend-like-stripe-in-6-hours-heres-what-i-learned-bc080d2b39e7)** - Covers the learning journey, concepts explained, and why these patterns matter.

---

## Overview

This project implements a backend commerce service that handles:

- **Subscription Management**: Purchase, renewal, and cancellation workflows
- **Gifting System**: Create and redeem subscription gifts
- **Idempotent Operations**: Retry-safe APIs preventing duplicate charges
- **Transactional Safety**: Atomic state transitions with rollback guarantees

---

## Key Results

| Metric | Target | Achieved |
|--------|--------|----------|
| P50 Latency | < 150ms | **20ms** ✅ |
| P95 Latency | < 150ms | **117ms** ✅ |
| Error Rate | < 1% | **0%** ✅ |
| Idempotency | 100% | **100%** ✅ |

---

## Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ HTTP + Idempotency-Key
       ▼
┌─────────────────────────────────────┐
│            API Layer (Go)           │
│  ┌─────────────────────────────┐   │
│  │ Rate Limiter                │   │
│  │ Idempotency Middleware      │   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │ Handlers                    │   │
│  │ - Subscribe/Renew/Cancel    │   │
│  │ - Gift/Redeem               │   │
│  └─────────────────────────────┘   │
│  ┌─────────────────────────────┐   │
│  │ Repository Layer            │   │
│  │ - Transaction Support       │   │
│  │ - CRUD Operations           │   │
│  └─────────────────────────────┘   │
└───────┬─────────────────┬───────────┘
        │                 │
        ▼                 ▼
┌───────────────┐  ┌─────────────┐
│  PostgreSQL   │  │    Redis    │
│               │  │             │
│ - Users       │  │ Idempotency │
│ - Subs        │  │ Rate Limits │
│ - Gifts       │  │             │
│ - Transactions│  │             │
└───────────────┘  └─────────────┘
```

---

## Core Patterns Implemented

### 1. Idempotency

Every write request includes an `Idempotency-Key` header. The system checks Redis before processing:

- **Key not seen**: Process request, cache response (24hr TTL)
- **Key exists**: Return cached response, skip database

**Validated**: 20 requests with same key → 1 DB insert, 19 cached responses.

### 2. Database Transactions

All multi-step operations are wrapped in transactions:

```go
tx, _ := db.BeginTx()
defer tx.Rollback()

// All operations atomic
db.CreateSubscriptionTx(tx, ...)
db.RecordTransactionTx(tx, ...)

tx.Commit() // All or nothing
```

### 3. Rate Limiting

Token bucket algorithm with Redis:
- 10 requests/minute per client
- Returns 429 when exceeded
- Auto-resets after window

---

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Go 1.21+

### 1. Start Services

```bash
docker-compose up -d
```

### 2. Run the Server

```bash
go run cmd/api/main.go
```

Output:
```
2026/01/11 00:47:44 Database connected successfully
2026/01/11 00:47:44 Running migration: 001_initial_schema.sql
2026/01/11 00:47:44 Redis connected successfully
2026/01/11 00:47:44 Starting server on :8080
```

### 3. Create Test Users

```bash
docker exec subscription_postgres psql -U postgres -d subscription_db \
  -c "INSERT INTO users (email) VALUES ('test@example.com'), ('friend@example.com');"
```

### 4. Test the API

```bash
# Subscribe
curl -X POST http://localhost:8080/subscribe \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: sub-001" \
  -d '{"user_id": 1, "plan": "monthly", "duration_months": 1}'

# Test idempotency (same key = same response)
curl -X POST http://localhost:8080/subscribe \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: sub-001" \
  -d '{"user_id": 1, "plan": "monthly", "duration_months": 1}'
```

---

## API Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| POST | `/subscribe` | Create subscription |
| POST | `/renew` | Extend subscription |
| POST | `/cancel` | Cancel subscription |
| POST | `/gift` | Create gift |
| POST | `/gift/redeem` | Redeem gift |
| GET | `/subscriptions/{user_id}` | Get user subscriptions |

**Note**: All POST requests require `Idempotency-Key` header.

### Request/Response Examples

#### Subscribe

```bash
POST /subscribe
Headers:
  Content-Type: application/json
  Idempotency-Key: sub-001

Body:
{
  "user_id": 1,
  "plan": "monthly",
  "duration_months": 1
}

Response (201):
{
  "id": 1,
  "user_id": 1,
  "status": "active",
  "start_date": "2026-01-11T00:51:31Z",
  "end_date": "2026-02-11T00:51:31Z"
}
```

#### Create Gift

```bash
POST /gift
Headers:
  Idempotency-Key: gift-001

Body:
{
  "gifter_id": 1,
  "recipient_email": "friend@example.com",
  "duration_months": 3
}

Response (201):
{
  "id": 1,
  "gifter_id": 1,
  "recipient_email": "friend@example.com",
  "status": "pending",
  "duration_months": 3,
  "expires_at": "2026-02-10T00:52:09Z"
}
```

#### Redeem Gift

```bash
POST /gift/redeem
Headers:
  Idempotency-Key: redeem-001

Body:
{
  "gift_id": 1,
  "user_id": 2
}

Response (200):
{
  "subscription_id": 2,
  "gift_id": 1,
  "status": "redeemed",
  "start_date": "2026-01-11T00:52:14Z",
  "end_date": "2026-04-11T00:52:14Z"
}
```

---

## Data Model

```
┌──────────────┐         ┌──────────────────┐
│    users     │         │  subscriptions   │
├──────────────┤         ├──────────────────┤
│ id (PK)      │───1:N───│ id (PK)          │
│ email        │         │ user_id (FK)     │
│ created_at   │         │ status           │
└──────────────┘         │ start_date       │
                         │ end_date         │
      │                  │ cancelled_at     │
      │                  └──────────────────┘
      │
      │                  ┌──────────────────┐
      │                  │      gifts       │
      │                  ├──────────────────┤
      └───────1:N────────│ id (PK)          │
                         │ gifter_id (FK)   │
                         │ recipient_id     │
                         │ status           │
                         │ expires_at       │
                         └──────────────────┘

┌─────────────────────┐
│    transactions     │
├─────────────────────┤
│ id (PK)             │
│ idempotency_key (UK)│
│ operation_type      │
│ entity_id           │
│ created_at          │
└─────────────────────┘
```

**Subscription States**: `active` | `cancelled` | `expired` | `pending`

**Gift States**: `pending` | `redeemed` | `expired`

---

## Testing

### Run Load Tests

```bash
go run tests/load/loadtest.go
```

**Output:**
```
=== Subscription Commerce Backend Load Test ===

[Test 1] Health Endpoint Performance (100 requests)
  Success: 100 (100.0%)
  P50 Latency: 20.0632ms
  P95 Latency: 116.6391ms
  ✅ P50 under 150ms target

[Test 2] Idempotency Validation (20 retries with same key)
  First Request: 1
  Duplicate Responses: 19
  ✅ Idempotency Working - No duplicate operations!

[Test 3] Mixed Workload (50 requests)
  Success: 50 (100.0%)
  P50 Latency: 31.2423ms
  ✅ P50 under 150ms target

=== Load Test Complete ===
```

---

## Project Structure

```
subscription-commerce-backend/
├── cmd/api/
│   └── main.go                 # Entry point
├── internal/
│   ├── handlers/
│   │   ├── subscription.go     # Subscribe/Renew/Cancel
│   │   └── gift.go             # Gift/Redeem
│   ├── middleware/
│   │   ├── idempotency.go      # Idempotency middleware
│   │   └── ratelimit.go        # Rate limiting
│   ├── models/
│   │   └── models.go           # Data structures
│   ├── database/
│   │   ├── postgres.go         # DB connection
│   │   ├── repository.go       # CRUD operations
│   │   └── migrations/
│   │       └── 001_initial_schema.sql
│   └── cache/
│       └── redis.go            # Redis client
├── tests/
│   ├── integration/
│   │   └── api_test.go
│   └── load/
│       └── loadtest.go
├── docker-compose.yml
├── go.mod
└── README.md
```

---

## Tech Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| API Server | Go 1.25 | HTTP handling |
| Database | PostgreSQL 16 | ACID transactions |
| Cache | Redis 7 | Idempotency + rate limiting |
| Containers | Docker Compose | Local development |

---

## Design Tradeoffs

| Decision | Why | Tradeoff |
|----------|-----|----------|
| Redis for idempotency | Fast O(1) lookups, auto-expiry | Lost guarantees if Redis crashes |
| Simple 4-state model | Easy to test and reason about | No grace periods or trials |
| No authentication | Focus on commerce patterns | Not production-ready as-is |
| Simulated payments | Avoid Stripe complexity | No real payment webhooks |

---

## Future Enhancements

- Payment provider integration (Stripe webhooks)
- Email notifications for gifts
- Subscription tiers (Bronze/Silver/Gold)
- Prometheus metrics and distributed tracing

---

## Learn More

For a detailed walkthrough of the concepts, implementation journey, and lessons learned:

📝 **[Read the full blog post on Medium](https://medium.com/@jeetp5118/i-built-a-subscription-backend-like-stripe-in-6-hours-heres-what-i-learned-bc080d2b39e7)**

---

## Author

**Jeet Patel**  
[LinkedIn](https://www.linkedin.com/in/pateljeet22/) · [GitHub](https://github.com/Jeet-51)

---

## License

MIT License
