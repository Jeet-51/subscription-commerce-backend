# Subscription & Gifting Commerce Backend

A production-grade backend service demonstrating subscription and gifting workflows with emphasis on **correctness**, **idempotency**, and **transactional safety**.

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://golang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?logo=postgresql)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D?logo=redis)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker)](https://www.docker.com/)

---

## Table of Contents

- [Overview](#overview)
- [Key Achievements](#key-achievements)
- [Motivation](#motivation)
- [Architecture](#architecture)
- [Core Design Principles](#core-design-principles)
- [Data Model](#data-model)
- [API Reference](#api-reference)
- [Workflow Details](#workflow-details)
- [Quick Start](#quick-start)
- [Testing](#testing)
- [Performance Results](#performance-results)
- [Design Tradeoffs](#design-tradeoffs)
- [Project Structure](#project-structure)
- [Future Enhancements](#future-enhancements)

---

## Overview

This project implements a backend commerce service that handles:

- **Subscription Management**: Purchase, renewal, and cancellation workflows
- **Gifting System**: Create and redeem subscription gifts
- **Idempotent Operations**: Retry-safe APIs preventing duplicate charges
- **Transactional Safety**: Atomic state transitions with rollback guarantees

---

## Why I Built This

### The Spark

While preparing for backend engineering interviews, I kept encountering questions about payment systems, idempotency, and handling distributed failures. I realized that while I understood these concepts theoretically, I had never actually implemented them from scratch.

Then I read about how Stripe processes billions in payments using idempotency keys, and how a single duplicate charge can cost companies both money and customer trust. That's when it clicked: I needed to build something that solves real problems, not just another CRUD app.

### The Challenge I Set For Myself

I gave myself a constraint: **build it in under 6 hours**, just like a take-home assignment. This forced me to:
- Focus on what matters (correctness over features)
- Make deliberate tradeoffs (documented in Design Tradeoffs section)
- Ship something working, not something perfect

### What I Learned

1. **Idempotency is harder than it looks**: Caching responses sounds simple until you handle edge cases like partial failures and race conditions.

2. **Transactions save lives**: Wrapping related operations in a single transaction prevented so many potential bugs.

3. **Redis is incredibly versatile**: Using it for both idempotency and rate limiting showed me why it's a staple in production systems.

4. **Load testing reveals truth**: My code "worked" until I threw 100 concurrent requests at it. That's when the real debugging started.

### Why This Matters

Every subscription service, from Netflix to Spotify to your local gym app, faces these exact challenges. By building this, I now understand:
- Why payment APIs require idempotency keys
- How companies prevent double-charging customers
- What happens behind the scenes when you click "Subscribe"

This isn't just a portfolio project. It's proof that I can build systems that handle real money and real consequences.

---

## Key Achievements

| Metric | Target | Achieved |
|--------|--------|----------|
| **P50 Latency** | < 150ms | **20-31ms** ✅ |
| **P95 Latency** | < 150ms | **117-123ms** ✅ |
| **P99 Latency** | < 200ms | **136-156ms** ✅ |
| **Error Rate** | < 1% | **0%** ✅ |
| **Idempotency** | 100% duplicate prevention | **100%** ✅ |

---

## Motivation

### The Problem

Subscription and payment systems face critical challenges:

1. **Duplicate Charges**: Network failures cause client retries, risking double billing
2. **Partial Failures**: Database commits succeed but response delivery fails
3. **Race Conditions**: Concurrent subscription actions create inconsistent state
4. **Retry Storms**: Traffic spikes overwhelm systems without protection

### Real-World Impact

- Stripe processes billions in payments using idempotency keys
- Payment processors lose revenue to duplicate charge disputes
- Subscription services face churn from billing inconsistencies

### This Solution

Implements **industry-standard patterns** for building reliable commerce systems:

- **Idempotency** prevents duplicate operations
- **Transactions** ensure atomic state changes
- **Explicit State Machines** avoid invalid transitions
- **Rate Limiting** protects against abuse

---

## Architecture

### System Components

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ HTTP + Idempotency-Key
       ▼
┌─────────────────────────────────────┐
│         API Layer (Go)              │
│  ┌─────────────────────────────┐   │
│  │ Rate Limiting Middleware    │   │
│  │ Idempotency Middleware      │   │
│  │ Request Validation          │   │
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

### Technology Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **API Server** | Go 1.25 | High-performance HTTP handling |
| **Database** | PostgreSQL 16 | ACID transactions, relational integrity |
| **Cache** | Redis 7 | Idempotency tracking, rate limiting |
| **Containerization** | Docker Compose | Consistent dev/test environments |

---

## Core Design Principles

### 1. Idempotency

**Problem**: Client retries cause duplicate operations (double charges)

**Solution**: Idempotency key pattern with Redis caching

```
POST /subscribe
Headers:
  Idempotency-Key: uuid-123-456

First request  → Creates subscription, caches response, returns 201
Retry (same key) → Returns cached 201 response (no DB operation)
Different key → Creates new subscription
```

**Implementation**:
- Store `{idempotency_key -> response}` in Redis (24hr TTL)
- Check before processing write operations
- Return cached response for duplicate keys

**Validated Result**: 20 requests with same key → 1 DB insert, 19 cached responses ✅

### 2. Transaction Safety

**Problem**: Partial writes corrupt state

**Solution**: Database transactions wrap all state changes

```go
tx, _ := db.BeginTx()
defer tx.Rollback()

// All operations in single transaction
subscription := db.CreateSubscriptionTx(tx, ...)
db.RecordTransactionTx(tx, ...)

tx.Commit() // All or nothing
```

**Guarantees**:
- Atomic state transitions
- Rollback on any failure
- No orphaned records

### 3. Explicit State Modeling

**Problem**: Boolean flags create ambiguous states

**Solution**: Explicit state machine

```
Subscription States:
  active → cancelled
  active → expired (auto-transition)
  pending → active (on payment)
  
Invalid transitions rejected at API layer
```

### 4. Rate Limiting

**Problem**: Request bursts overwhelm system

**Solution**: Token bucket with Redis

```
Rate: 10 requests/minute per client
After limit: Returns 429 Too Many Requests
```

---

## Data Model

### Entity-Relationship Diagram

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
                         │ recipient_id (FK)│
                         │ status           │
                         │ redeemed_at      │
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

### Table Schemas

#### subscriptions

```sql
CREATE TABLE subscriptions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    status VARCHAR(20) NOT NULL CHECK (status IN ('active', 'cancelled', 'expired', 'pending')),
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP NOT NULL,
    cancelled_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_user_status ON subscriptions(user_id, status);
CREATE INDEX idx_subscriptions_end_date ON subscriptions(end_date) WHERE status = 'active';
```

#### gifts

```sql
CREATE TABLE gifts (
    id SERIAL PRIMARY KEY,
    gifter_id INTEGER NOT NULL REFERENCES users(id),
    recipient_email VARCHAR(255) NOT NULL,
    recipient_id INTEGER REFERENCES users(id),
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'redeemed', 'expired')),
    duration_months INTEGER NOT NULL,
    redeemed_at TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_gifts_recipient_email ON gifts(recipient_email, status);
```

#### transactions

```sql
CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    idempotency_key VARCHAR(255) UNIQUE NOT NULL,
    operation_type VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id INTEGER NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_transactions_idempotency_key ON transactions(idempotency_key);
```

---

## API Reference

### Base URL

```
http://localhost:8080
```

### Endpoints Summary

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| POST | `/subscribe` | Create subscription |
| POST | `/renew` | Extend subscription |
| POST | `/cancel` | Cancel subscription |
| POST | `/gift` | Create gift |
| POST | `/gift/redeem` | Redeem gift |
| GET | `/subscriptions/{user_id}` | Get user subscriptions |

### Common Headers

```
Content-Type: application/json
Idempotency-Key: <uuid> (required for POST requests)
```

---

### 1. Health Check

**Request**
```http
GET /health
```

**Response** (200 OK)
```json
{
  "status": "healthy",
  "database": "connected",
  "redis": "connected",
  "timestamp": "2026-01-11T05:50:48Z"
}
```

---

### 2. Subscribe

**Request**
```http
POST /subscribe
Idempotency-Key: sub-001

{
  "user_id": 1,
  "plan": "monthly",
  "duration_months": 1
}
```

**Response** (201 Created)
```json
{
  "id": 1,
  "user_id": 1,
  "status": "active",
  "start_date": "2026-01-11T00:51:31.907345Z",
  "end_date": "2026-02-11T00:51:31.907345Z",
  "created_at": "2026-01-11T05:51:34.451058Z",
  "updated_at": "2026-01-11T05:51:34.451058Z"
}
```

**Error Cases**
- `400 Bad Request`: Missing Idempotency-Key or invalid payload
- `404 Not Found`: User not found
- `409 Conflict`: User already has active subscription
- `429 Too Many Requests`: Rate limit exceeded

---

### 3. Renew Subscription

**Request**
```http
POST /renew
Idempotency-Key: renew-001

{
  "subscription_id": 1,
  "duration_months": 1
}
```

**Response** (200 OK)
```json
{
  "id": 1,
  "user_id": 1,
  "status": "active",
  "start_date": "2026-01-11T00:51:31.907345Z",
  "end_date": "2026-03-11T00:51:31.907345Z",
  "created_at": "2026-01-11T05:51:34.451058Z",
  "updated_at": "2026-01-11T05:52:06.547359Z"
}
```

---

### 4. Cancel Subscription

**Request**
```http
POST /cancel
Idempotency-Key: cancel-001

{
  "subscription_id": 1
}
```

**Response** (200 OK)
```json
{
  "id": 1,
  "user_id": 1,
  "status": "cancelled",
  "cancelled_at": "2026-01-11T05:52:24.06136Z",
  "start_date": "2026-01-11T00:51:31.907345Z",
  "end_date": "2026-03-11T00:51:31.907345Z",
  "created_at": "2026-01-11T05:51:34.451058Z",
  "updated_at": "2026-01-11T05:52:24.06136Z"
}
```

---

### 5. Create Gift

**Request**
```http
POST /gift
Idempotency-Key: gift-001

{
  "gifter_id": 1,
  "recipient_email": "friend@example.com",
  "duration_months": 3
}
```

**Response** (201 Created)
```json
{
  "id": 1,
  "gifter_id": 1,
  "recipient_email": "friend@example.com",
  "status": "pending",
  "duration_months": 3,
  "expires_at": "2026-02-10T00:52:09.026476Z",
  "created_at": "2026-01-11T05:52:11.189073Z"
}
```

---

### 6. Redeem Gift

**Request**
```http
POST /gift/redeem
Idempotency-Key: redeem-001

{
  "gift_id": 1,
  "user_id": 2
}
```

**Response** (200 OK)
```json
{
  "subscription_id": 2,
  "gift_id": 1,
  "status": "redeemed",
  "start_date": "2026-01-11T00:52:14.811321Z",
  "end_date": "2026-04-11T00:52:14.811321Z"
}
```

---

### 7. Get User Subscriptions

**Request**
```http
GET /subscriptions/1
```

**Response** (200 OK)
```json
{
  "user_id": 1,
  "subscriptions": [
    {
      "id": 1,
      "user_id": 1,
      "status": "active",
      "start_date": "2026-01-11T00:51:31.907345Z",
      "end_date": "2026-02-11T00:51:31.907345Z",
      "created_at": "2026-01-11T05:51:34.451058Z",
      "updated_at": "2026-01-11T05:51:34.451058Z"
    }
  ]
}
```

---

## Workflow Details

### Subscription Purchase Flow

```
┌────────┐                ┌─────────┐                ┌──────────┐
│ Client │                │   API   │                │ Database │
└───┬────┘                └────┬────┘                └────┬─────┘
    │                          │                          │
    │ POST /subscribe          │                          │
    │ + Idempotency-Key        │                          │
    ├─────────────────────────>│                          │
    │                          │                          │
    │                          │ Check Redis for key      │
    │                          │ (idempotency check)      │
    │                          │                          │
    │                          │ BEGIN TRANSACTION        │
    │                          ├─────────────────────────>│
    │                          │                          │
    │                          │ SELECT user subscription │
    │                          │ (validate no active sub) │
    │                          │<─────────────────────────┤
    │                          │                          │
    │                          │ INSERT subscription      │
    │                          ├─────────────────────────>│
    │                          │                          │
    │                          │ INSERT transaction log   │
    │                          ├─────────────────────────>│
    │                          │                          │
    │                          │ COMMIT                   │
    │                          │<─────────────────────────┤
    │                          │                          │
    │                          │ Store response in Redis  │
    │                          │                          │
    │ 201 Created              │                          │
    │<─────────────────────────┤                          │
    │                          │                          │
```

### Retry Handling

```
Request 1 (Key: abc-123)
  → Processes normally
  → Stores response in Redis (TTL: 24h)
  → Returns 201 Created

Request 2 (Key: abc-123) - RETRY
  → Redis hit found
  → Returns cached 201 Created
  → No database transaction executed

Request 3 (Key: xyz-789) - NEW REQUEST
  → Redis miss
  → Processes normally
  → Creates new subscription
```

### Gift Redemption Flow

```
1. GIFT CREATION
   ├─> Gifter creates gift for recipient email
   ├─> Gift status: "pending"
   ├─> Expiry set to 30 days
   └─> Email notification sent (future enhancement)

2. RECIPIENT DISCOVERY
   ├─> Recipient logs in or signs up
   ├─> System matches email to pending gifts
   └─> Displays redemption option

3. REDEMPTION TRANSACTION
   BEGIN
     ├─> Validate gift not expired/redeemed
     ├─> Create/extend subscription for recipient
     ├─> Update gift status to "redeemed"
     ├─> Record transaction
   COMMIT

4. POST-REDEMPTION
   └─> Subscription activated immediately
```

---

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Go 1.21+ (for local development)

### 1. Clone Repository

```bash
git clone https://github.com/jeet-patel/subscription-commerce-backend
cd subscription-commerce-backend
```

### 2. Start Services

```bash
docker-compose up -d
```

This starts:
- PostgreSQL on `localhost:5432`
- Redis on `localhost:6379`

### 3. Run the API Server

```bash
go run cmd/api/main.go
```

Expected output:
```
2026/01/11 00:47:44 Database connected successfully
2026/01/11 00:47:44 Running migration: 001_initial_schema.sql
2026/01/11 00:47:44 Migration completed: 001_initial_schema.sql
2026/01/11 00:47:44 Redis connected successfully
2026/01/11 00:47:44 Starting server on :8080
2026/01/11 00:47:44 Available endpoints:
2026/01/11 00:47:44   GET  /health
2026/01/11 00:47:44   POST /subscribe
2026/01/11 00:47:44   POST /renew
2026/01/11 00:47:44   POST /cancel
2026/01/11 00:47:44   POST /gift
2026/01/11 00:47:44   POST /gift/redeem
2026/01/11 00:47:44   GET  /subscriptions/{user_id}
```

### 4. Create Test Users

```bash
docker exec subscription_postgres psql -U postgres -d subscription_db -c "INSERT INTO users (email) VALUES ('test@example.com'), ('friend@example.com');"
```

### 5. Test the API

#### Using PowerShell

```powershell
# Health check
Invoke-RestMethod -Uri "http://localhost:8080/health" -Method GET

# Subscribe
$headers = @{"Content-Type"="application/json"; "Idempotency-Key"="sub-001"}
Invoke-RestMethod -Uri "http://localhost:8080/subscribe" -Method POST -Headers $headers -Body '{"user_id":1,"plan":"monthly","duration_months":1}'

# Get Subscriptions
Invoke-RestMethod -Uri "http://localhost:8080/subscriptions/1" -Method GET

# Renew
$headers = @{"Content-Type"="application/json"; "Idempotency-Key"="renew-001"}
Invoke-RestMethod -Uri "http://localhost:8080/renew" -Method POST -Headers $headers -Body '{"subscription_id":1,"duration_months":1}'

# Create Gift
$headers = @{"Content-Type"="application/json"; "Idempotency-Key"="gift-001"}
Invoke-RestMethod -Uri "http://localhost:8080/gift" -Method POST -Headers $headers -Body '{"gifter_id":1,"recipient_email":"friend@example.com","duration_months":3}'

# Redeem Gift
$headers = @{"Content-Type"="application/json"; "Idempotency-Key"="redeem-001"}
Invoke-RestMethod -Uri "http://localhost:8080/gift/redeem" -Method POST -Headers $headers -Body '{"gift_id":1,"user_id":2}'

# Cancel
$headers = @{"Content-Type"="application/json"; "Idempotency-Key"="cancel-001"}
Invoke-RestMethod -Uri "http://localhost:8080/cancel" -Method POST -Headers $headers -Body '{"subscription_id":1}'
```

#### Using curl (Linux/Mac)

```bash
# Health check
curl http://localhost:8080/health

# Subscribe
curl -X POST http://localhost:8080/subscribe \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: sub-001" \
  -d '{"user_id":1,"plan":"monthly","duration_months":1}'

# Get Subscriptions
curl http://localhost:8080/subscriptions/1

# Renew
curl -X POST http://localhost:8080/renew \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: renew-001" \
  -d '{"subscription_id":1,"duration_months":1}'

# Create Gift
curl -X POST http://localhost:8080/gift \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: gift-001" \
  -d '{"gifter_id":1,"recipient_email":"friend@example.com","duration_months":3}'

# Redeem Gift
curl -X POST http://localhost:8080/gift/redeem \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: redeem-001" \
  -d '{"gift_id":1,"user_id":2}'

# Cancel
curl -X POST http://localhost:8080/cancel \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: cancel-001" \
  -d '{"subscription_id":1}'
```

### 6. Stop Services

```bash
docker-compose down
```

---

## Testing

### Load Test

```bash
go run tests/load/loadtest.go
```

**Sample Output:**
```
=== Subscription Commerce Backend Load Test ===

[Test 1] Health Endpoint Performance (100 requests)
  Total Requests: 100
  Success: 100 (100.0%)
  Errors: 0
  P50 Latency: 20.0632ms
  P95 Latency: 116.6391ms
  P99 Latency: 156.5252ms
  ✅ P50 under 150ms target

[Test 2] Idempotency Validation (20 retries with same key)
  Total Requests: 20
  First Request: 1
  Duplicate Responses: 19
  Errors: 0
  ✅ Idempotency Working - No duplicate operations!

[Test 3] Mixed Workload (50 requests)
  Total Requests: 50
  Success: 50 (100.0%)
  Errors: 0
  P50 Latency: 31.2423ms
  P95 Latency: 122.9452ms
  P99 Latency: 136.0426ms
  ✅ P50 under 150ms target

=== Load Test Complete ===
```

### Integration Tests

```bash
go test ./tests/integration/... -v
```

---

## Performance Results

### Actual Load Test Results (January 11, 2026)

| Test | Requests | Success Rate | P50 | P95 | P99 |
|------|----------|--------------|-----|-----|-----|
| Health Endpoint | 100 | 100% | 20ms | 117ms | 156ms |
| Mixed Workload | 50 | 100% | 31ms | 123ms | 136ms |

### Idempotency Validation

| Metric | Result |
|--------|--------|
| Total Requests (same key) | 20 |
| Database Inserts | 1 |
| Cached Responses | 19 |
| Duplicate Operations | 0 ✅ |

### Rate Limiting Validation

```
Requests 1-10: Success (200 OK)
Request 11+: Rate limit exceeded (429 Too Many Requests)
After 1 minute: Rate limit reset
```

---

## Design Tradeoffs

### 1. Idempotency Storage (Redis vs PostgreSQL)

**Chosen**: Redis with 24-hour TTL

**Rationale**:
- Fast lookups (O(1) vs table scan)
- Automatic expiry (no cleanup jobs)
- Reduced database load

**Tradeoff**: 
- Lost idempotency guarantees on Redis failure
- Acceptable for 24-hour window (retries typically <1min)

**Alternative**: PostgreSQL with scheduled cleanup

---

### 2. State Machine Complexity

**Chosen**: Simple 4-state model (active, cancelled, expired, pending)

**Rationale**:
- Covers 90% of real-world scenarios
- Easy to reason about and test
- Minimal invalid transition paths

**Tradeoff**:
- Doesn't model payment failures, grace periods, or paused subscriptions
- Sufficient for demonstrating correctness principles

**Production Extension**: Add states like `past_due`, `suspended`, `trial`

---

### 3. Gift Expiry Mechanism

**Chosen**: Database column + validation on redemption

**Rationale**:
- Simple implementation
- Consistent with subscription expiry
- No complex event scheduling

**Tradeoff**:
- Requires background worker for auto-expiry (future enhancement)
- Currently manual check on redemption

**Alternative**: Redis TTL for instant expiry

---

### 4. Rate Limiting Scope

**Chosen**: Per-client limits (10 req/min)

**Rationale**:
- Prevents abuse from individual clients
- Simple implementation with Redis

**Tradeoff**:
- Doesn't protect against distributed attacks
- No global rate limiting

**Production Extension**: Add IP-based and global limits

---

### 5. Authentication Exclusion

**Chosen**: No auth layer in this project

**Rationale**:
- Focuses on commerce logic correctness
- Auth is well-understood (OAuth, JWT)
- Avoids scope creep

**Tradeoff**:
- Not production-deployable as-is
- User IDs passed directly in requests

**Integration Point**: Add middleware layer before handlers

---

### 6. Payment Provider Integration

**Chosen**: Simulated payments (immediate success)

**Rationale**:
- Idempotency patterns transfer to real providers (Stripe, PayPal)
- Avoids API key management and testing complexity
- Focuses on transaction safety

**Tradeoff**:
- Doesn't handle payment webhooks or async confirmations

**Production Pattern**:
```
1. Create "pending" subscription
2. Initiate payment with provider
3. Handle webhook to mark "active"
4. Implement retry logic for webhook delivery
```

---

## Project Structure

```
subscription-commerce-backend/
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
├── internal/
│   ├── handlers/
│   │   ├── subscription.go      # Subscribe/Renew/Cancel/Get
│   │   └── gift.go              # Gift/Redeem
│   ├── middleware/
│   │   ├── idempotency.go       # Idempotency middleware
│   │   └── ratelimit.go         # Rate limiting
│   ├── models/
│   │   └── models.go            # Data structures
│   ├── database/
│   │   ├── postgres.go          # DB connection + migrations
│   │   ├── repository.go        # CRUD operations
│   │   └── migrations/
│   │       └── 001_initial_schema.sql
│   └── cache/
│       └── redis.go             # Redis client
├── tests/
│   ├── integration/
│   │   └── api_test.go          # Integration tests
│   └── load/
│       └── loadtest.go          # Load testing
├── examples/
│   └── api_requests.ps1         # PowerShell test script
├── docker-compose.yml           # PostgreSQL + Redis
├── go.mod                       # Go dependencies
└── README.md
```

---

## Dependencies

```go
require (
    github.com/lib/pq v1.10.9           // PostgreSQL driver
    github.com/go-redis/redis/v8        // Redis client
)
```

---

## Future Enhancements

### Phase 2: Advanced Features

- **Payment Webhooks**: Handle async payment confirmations
- **Email Notifications**: Gift creation/redemption alerts
- **Subscription Tiers**: Bronze/Silver/Gold pricing
- **Trial Periods**: 7-day free trial support

### Phase 3: Observability

- **Prometheus Metrics**: Request counts, latency histograms
- **Distributed Tracing**: Request flow visualization
- **Structured Logging**: JSON logs with correlation IDs

### Phase 4: Scalability

- **Database Sharding**: Partition by user_id
- **Read Replicas**: Separate read/write traffic
- **Event Sourcing**: Audit log for all state changes

---

## License

MIT License - see LICENSE file for details

---

## Author

**Jeet Patel**  
LinkedIn: [linkedin.com/in/pateljeet22](https://www.linkedin.com/in/pateljeet22/)

---

## Acknowledgments

Inspired by production commerce systems at:
- Stripe (idempotency patterns)
- Netflix (subscription management)
- Amazon (transactional workflows)

Built to demonstrate backend engineering principles for portfolio and interview discussions.
