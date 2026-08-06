# Enterprise Manual Testing & Operations Guide
## Abstract SaaS Metering & Billing Infrastructure

> **Target Audience:** DevOps, Site Reliability Engineers (SRE), QA Engineers, and Platform Developers  
> **Supported OS Platforms:** Linux (Ubuntu/RHEL/Debian), macOS (Apple Silicon/Intel), Windows 10/11 (PowerShell/WSL2)  
> **Engine Version:** Production Grade 2.0 (Timezone-Aware, Multi-Backend Chain, WAL Fallback, Circuit Breakers)

---

## Table of Contents

1. [Prerequisites & Environment Requirements](#1-prerequisites--environment-requirements)
2. [Infrastructure Setup (Docker & Local Installation)](#2-infrastructure-setup-docker--local-installation)
3. [Application Configuration & Launch](#3-application-configuration--launch)
4. [Manual Scenario Testing Suite](#4-manual-scenario-testing-suite)
   - [Scenario 1: System Bootstrapping & Storage Health Check](#scenario-1-system-bootstrapping--storage-health-check)
   - [Scenario 2: Multi-Tenant Service Subscription Registration](#scenario-2-multi-tenant-service-subscription-registration)
   - [Scenario 3: Metering Event Ingestion (REST & EventBus Paths)](#scenario-3-metering-event-ingestion-rest--eventbus-paths)
   - [Scenario 4: Timezone-Aware Cycle Window & Statement Verification](#scenario-4-timezone-aware-cycle-window--statement-verification)
   - [Scenario 5: Event Deduplication & Idempotency Validation](#scenario-5-event-deduplication--idempotency-validation)
   - [Scenario 6: Failover, Outage Simulation & Disk WAL Replay](#scenario-6-failover-outage-simulation--disk-wal-replay)
   - [Scenario 7: Query Error Isolation Verification (B1 / B2 Error Guards)](#scenario-7-query-error-isolation-verification-b1--b2-error-guards)
   - [Scenario 8: Graceful Shutdown & Drain Timeout (B3 Verification)](#scenario-8-graceful-shutdown--drain-timeout-b3-verification)
   - [Scenario 9: Subscription Lifecycle FSM State Transitions](#scenario-9-subscription-lifecycle-fsm-state-transitions)
5. [Backend Verification Commands (PostgreSQL & Redis CLI)](#5-backend-verification-commands-postgresql--redis-cli)
6. [Troubleshooting & FAQ](#6-troubleshooting--faq)
7. [Low-Level Infrastructure & Database/Redis Deep Testing Suite](#7-low-level-infrastructure--databaseredis-deep-testing-suite)
   - [Scenario 10: PostgreSQL Connection Pool & Lock Contention Test](#scenario-10-postgresql-connection-pool--lock-contention-test)
   - [Scenario 11: Redis Memory Eviction, TTL & Slowlog Inspection](#scenario-11-redis-memory-eviction-ttl--slowlog-inspection)
   - [Scenario 12: Network Partition & Partial Packet Loss Simulation](#scenario-12-network-partition--partial-packet-loss-simulation)
   - [Scenario 13: File Descriptor & System Resource Limit Audit (`ulimit` & Socket Reuse)](#scenario-13-file-descriptor--system-resource-limit-audit-ulimit--socket-reuse)
   - [Scenario 14: Direct DB & Redis Schema, Index & Memory Footprint Verification](#scenario-14-direct-db--redis-schema-index--memory-footprint-verification)

---

## 1. Prerequisites & Environment Requirements

Before beginning manual validation, ensure the host machine meets the following specifications:

| Requirement | Minimum Version | Recommended |
| :--- | :--- | :--- |
| **Go Compiler** | 1.25.0+ | 1.25.0+ |
| **Docker Engine** | 24.0+ | Docker Desktop / OrbStack / Podman |
| **PostgreSQL** | 15.0+ | 16.0 (Alpine container) |
| **Redis** | 6.2+ | 7.2 (Alpine container) |
| **CLI Utilities** | `curl`, `jq` | `curl`, `jq`, `psql`, `redis-cli` |

---

## 2. Infrastructure Setup (Docker & Local Installation)

### Option A: Recommended — Quick Infrastructure via Docker Compose

Create or use the provided `docker-compose.yml` in the project root:

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    container_name: metering-postgres
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: password
      POSTGRES_DB: metering_db
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U user -d metering_db"]
      interval: 3s
      timeout: 3s
      retries: 5

  redis:
    image: redis:7.2-alpine
    container_name: metering-redis
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 3s
      timeout: 3s
      retries: 5
```

#### Start Infrastructure Container Cluster:

* **Linux / macOS (Terminal):**
  ```bash
  docker compose up -d
  ```

* **Windows (PowerShell):**
  ```powershell
  docker compose up -d
  ```

---

### Option B: Native Infrastructure Installation

#### Linux (Ubuntu/Debian):
```bash
# PostgreSQL
sudo apt update && sudo apt install -y postgresql postgresql-contrib redis-server
sudo systemctl start postgresql redis-server

# Create database and user
sudo -u postgres psql -c "CREATE USER user WITH PASSWORD 'password';"
sudo -u postgres psql -c "CREATE DATABASE metering_db OWNER user;"
```

#### macOS (Homebrew):
```bash
brew install postgresql@16 redis
brew services start postgresql@16
brew services start redis

# Create database
createuser user --createdb
psql -d postgres -c "ALTER USER user WITH PASSWORD 'password';"
createdb metering_db -O user
```

#### Windows (Chocolatey / Scoop):
```powershell
choco install postgresql16 redis-64
# Or use Docker Desktop on Windows WSL2 (Recommended)
```

---

## 3. Application Configuration & Launch

Configure the environment variables in `config.env` (or pass via environment):

```env
# Application Core
PORT=8080
DATA_DIR=./data
DAILY_TOKEN_QUOTA=1000

# Metering Storage Chain Configuration
# Backends order: memory (L1), redis (L2), postgres (L3)
METERING_BACKENDS=memory,redis,postgres

# Database Connection DSNs
METERING_POSTGRES_DSN=postgres://user:password@localhost:5432/metering_db?sslmode=disable
METERING_REDIS_ADDR=localhost:6379

# Write-Ahead Log (WAL) Fault Tolerance
METERING_WAL_ENABLED=true
METERING_WAL_DIR=./data/wal
METERING_WAL_RETRY_MS=3000

# Circuit Breaker & Channel Tuning
METERING_CB_THRESHOLD=3
METERING_CB_COOLDOWN_MS=10000
METERING_CHANNEL_SIZE=5000
METERING_DEDUP_RETENTION_MS=3600000
```

### Launch the Service:

* **Linux / macOS:**
  ```bash
  go run ./cmd/lab
  ```

* **Windows (PowerShell):**
  ```powershell
  go run .\cmd\lab
  ```

---

## 4. Manual Scenario Testing Suite

### Scenario 1: System Bootstrapping & Storage Health Check

**Objective:** Verify application startup, database table auto-migrations, and storage backend health reporting.

#### Steps:
Query the storage health endpoint to ensure all three storage layers (Memory, Redis, Postgres) are online and healthy.

* **Linux / macOS:**
  ```bash
  curl -s http://localhost:8080/v1/billing/storage/health | jq .
  ```

* **Windows (PowerShell):**
  ```powershell
  (Invoke-RestMethod -Uri "http://localhost:8080/v1/billing/storage/health") | ConvertTo-Json -Depth 5
  ```

#### Expected Output:
```json
{
  "backends": [
    { "name": "memory",   "priority": 0,  "healthy": true, "circuit_state": "closed", "pending_writes": 0 },
    { "name": "redis",    "priority": 10, "healthy": true, "circuit_state": "closed", "pending_writes": 0 },
    { "name": "postgres", "priority": 20, "healthy": true, "circuit_state": "closed", "pending_writes": 0 }
  ],
  "wal": {
    "enabled": true,
    "depth": 0,
    "active_segment": "segment_0001.jsonl",
    "total_segments": 1
  },
  "dedup": {
    "tracked_events": 0,
    "retention": "1h0m0s"
  }
}
```

---

### Scenario 2: Multi-Tenant Service Subscription Registration

**Objective:** Register independent subscriptions for a tenant across different services, timezones, and pricing models.

#### Request 1: AI Inference Service (`Asia/Tokyo`, Monthly Recurring)

* **Linux / macOS:**
  ```bash
  curl -X POST http://localhost:8080/v1/billing/subscriptions \
    -H "Content-Type: application/json" \
    -d '{
      "tenant_key": "enterprise-corp-01",
      "service_id": "ai-completion",
      "plan_id": "llm-enterprise-v1",
      "charge_type": "recurring_monthly",
      "timezone": "Asia/Tokyo",
      "anchor_time": "2026-08-01T00:00:00Z"
    }' | jq .
  ```

* **Windows (PowerShell):**
  ```powershell
  $body = @{
    tenant_key  = "enterprise-corp-01"
    service_id  = "ai-completion"
    plan_id     = "llm-enterprise-v1"
    charge_type = "recurring_monthly"
    timezone    = "Asia/Tokyo"
    anchor_time = "2026-08-01T00:00:00Z"
  } | ConvertTo-Json

  Invoke-RestMethod -Uri "http://localhost:8080/v1/billing/subscriptions" -Method Post -Body $body -ContentType "application/json"
  ```

#### Request 2: Object Storage Service (`America/New_York`, Pay-As-You-Go Metered)

* **Linux / macOS:**
  ```bash
  curl -X POST http://localhost:8080/v1/billing/subscriptions \
    -H "Content-Type: application/json" \
    -d '{
      "tenant_key": "enterprise-corp-01",
      "service_id": "object-storage",
      "plan_id": "s3-metered-v2",
      "charge_type": "metered",
      "timezone": "America/New_York",
      "anchor_time": "2026-08-05T12:00:00Z"
    }' | jq .
  ```

* **Windows (PowerShell):**
  ```powershell
  $body = @{
    tenant_key  = "enterprise-corp-01"
    service_id  = "object-storage"
    plan_id     = "s3-metered-v2"
    charge_type = "metered"
    timezone    = "America/New_York"
    anchor_time = "2026-08-05T12:00:00Z"
  } | ConvertTo-Json

  Invoke-RestMethod -Uri "http://localhost:8080/v1/billing/subscriptions" -Method Post -Body $body -ContentType "application/json"
  ```

#### Expected Output (`201 Created`):
```json
{
  "subscription_id": "sub_ai-completion_enterprise-corp-01",
  "status": "registered"
}
```

---

### Scenario 3: Metering Event Ingestion (REST & EventBus Paths)

**Objective:** Ingest consumption events via REST API and verify event propagation to all storage backends.

#### Request 1: Ingest AI Token Consumption

* **Linux / macOS:**
  ```bash
  curl -X POST http://localhost:8080/v1/billing/events \
    -H "Content-Type: application/json" \
    -d '{
      "event_id": "evt_ai_1001",
      "tenant_key": "enterprise-corp-01",
      "service_id": "ai-completion",
      "metric_id": "tokens",
      "unit": "tokens",
      "quantity": 2500,
      "timestamp": "2026-08-06T08:30:00Z"
    }' | jq .
  ```

* **Windows (PowerShell):**
  ```powershell
  $body = @{
    event_id   = "evt_ai_1001"
    tenant_key = "enterprise-corp-01"
    service_id = "ai-completion"
    metric_id  = "tokens"
    unit       = "tokens"
    quantity   = 2500
    timestamp  = "2026-08-06T08:30:00Z"
  } | ConvertTo-Json

  Invoke-RestMethod -Uri "http://localhost:8080/v1/billing/events" -Method Post -Body $body -ContentType "application/json"
  ```

#### Request 2: Ingest Object Storage Consumption

* **Linux / macOS:**
  ```bash
  curl -X POST http://localhost:8080/v1/billing/events \
    -H "Content-Type: application/json" \
    -d '{
      "event_id": "evt_store_2001",
      "tenant_key": "enterprise-corp-01",
      "service_id": "object-storage",
      "metric_id": "gb_hours",
      "unit": "GB-hr",
      "quantity": 150,
      "timestamp": "2026-08-06T09:00:00Z"
    }' | jq .
  ```

* **Windows (PowerShell):**
  ```powershell
  $body = @{
    event_id   = "evt_store_2001"
    tenant_key = "enterprise-corp-01"
    service_id = "object-storage"
    metric_id  = "gb_hours"
    unit       = "GB-hr"
    quantity   = 150
    timestamp  = "2026-08-06T09:00:00Z"
  } | ConvertTo-Json

  Invoke-RestMethod -Uri "http://localhost:8080/v1/billing/events" -Method Post -Body $body -ContentType "application/json"
  ```

#### Expected Output (`202 Accepted`):
```json
{
  "event_id": "evt_ai_1001",
  "status": "accepted"
}
```

---

### Scenario 4: Timezone-Aware Cycle Window & Statement Verification

**Objective:** Verify per-service cycle window calculation and metric aggregation.

#### Request 1: Query Per-Service Billing Statement for AI Completion

* **Linux / macOS:**
  ```bash
  curl -s http://localhost:8080/v1/billing/enterprise-corp-01/statement/ai-completion | jq .
  ```

* **Windows (PowerShell):**
  ```powershell
  (Invoke-RestMethod -Uri "http://localhost:8080/v1/billing/enterprise-corp-01/statement/ai-completion") | ConvertTo-Json -Depth 5
  ```

#### Expected Output:
```json
{
  "subscription_id": "sub_ai-completion_enterprise-corp-01",
  "tenant_key": "enterprise-corp-01",
  "service_id": "ai-completion",
  "plan_id": "llm-enterprise-v1",
  "charge_type": "recurring_monthly",
  "timezone": "Asia/Tokyo",
  "cycle_start_utc": "2026-08-01T00:00:00Z",
  "cycle_end_utc": "2026-09-01T00:00:00Z",
  "metrics": {
    "tokens": {
      "metric_id": "tokens",
      "unit": "tokens",
      "cycle_total": 2500,
      "total_events": 1
    }
  }
}
```

#### Request 2: Query Full Tenant Billing Overview

* **Linux / macOS:**
  ```bash
  curl -s http://localhost:8080/v1/billing/enterprise-corp-01/overview | jq .
  ```

* **Windows (PowerShell):**
  ```powershell
  (Invoke-RestMethod -Uri "http://localhost:8080/v1/billing/enterprise-corp-01/overview") | ConvertTo-Json -Depth 5
  ```

---

### Scenario 5: Event Deduplication & Idempotency Validation

**Objective:** Send the exact same `event_id` multiple times and verify that duplicate billing does not occur.

#### Steps:
Re-send the exact `evt_ai_1001` payload sent in Scenario 3:

* **Linux / macOS:**
  ```bash
  curl -X POST http://localhost:8080/v1/billing/events \
    -H "Content-Type: application/json" \
    -d '{
      "event_id": "evt_ai_1001",
      "tenant_key": "enterprise-corp-01",
      "service_id": "ai-completion",
      "metric_id": "tokens",
      "unit": "tokens",
      "quantity": 2500,
      "timestamp": "2026-08-06T08:30:00Z"
    }'
  ```

#### Verification:
Re-query the statement for `ai-completion`. `cycle_total` MUST remain `2500` and `total_events` MUST remain `1`.

---

### Scenario 6: Failover, Outage Simulation & Disk WAL Replay

**Objective:** Simulate an unexpected Redis / PostgreSQL database outage during live ingestion, verify automatic Disk WAL capture, and confirm automatic recovery upon service restoration.

#### Step 1: Stop PostgreSQL Database Container
* **Terminal:**
  ```bash
  docker stop metering-postgres
  ```

#### Step 2: Send New Metering Event During Database Outage

* **Linux / macOS:**
  ```bash
  curl -X POST http://localhost:8080/v1/billing/events \
    -H "Content-Type: application/json" \
    -d '{
      "event_id": "evt_ai_outage_9001",
      "tenant_key": "enterprise-corp-01",
      "service_id": "ai-completion",
      "metric_id": "tokens",
      "unit": "tokens",
      "quantity": 5000,
      "timestamp": "2026-08-06T09:15:00Z"
    }'
  ```

* **Windows (PowerShell):**
  ```powershell
  $body = @{
    event_id   = "evt_ai_outage_9001"
    tenant_key = "enterprise-corp-01"
    service_id = "ai-completion"
    metric_id  = "tokens"
    unit       = "tokens"
    quantity   = 5000
    timestamp  = "2026-08-06T09:15:00Z"
  } | ConvertTo-Json

  Invoke-RestMethod -Uri "http://localhost:8080/v1/billing/events" -Method Post -Body $body -ContentType "application/json"
  ```

> **Result:** Event returns `202 Accepted` immediately because L1 RAM accepts the write and queues it to the WAL.

#### Step 3: Check Storage Health Endpoint
Verify that WAL depth increased and PostgreSQL circuit breaker status reports `open` or `unhealthy`.

* **Terminal:**
  ```bash
  curl -s http://localhost:8080/v1/billing/storage/health | jq .wal
  ```

#### Step 4: Restart Database Container
* **Terminal:**
  ```bash
  docker start metering-postgres
  ```

#### Step 5: Verify Automatic WAL Replay & Healing
Wait ~5 seconds for the background WAL retry loop to run, then check storage health. WAL depth should return to `0` and PostgreSQL backend status should return to `healthy: true`.

---

### Scenario 7: Query Error Isolation Verification (B1 / B2 Error Guards)

**Objective:** Verify that query-level errors (such as bad payload data or missing secondary keys) do NOT mark storage backends unhealthy.

#### Steps:
Send an event with invalid/malformed schema data or query a non-existent subscription statement.

* **Linux / macOS:**
  ```bash
  curl -s http://localhost:8080/v1/billing/non-existent-tenant/statement/ai-completion
  ```

#### Verification:
Query `GET /v1/billing/storage/health`. Both `postgres` and `redis` backends MUST remain `healthy: true`.

---

### Scenario 8: Graceful Shutdown & Drain Timeout (B3 Verification)

**Objective:** Verify application process shutdown drains worker channels within the 5-second `WaitWithTimeout` limit without hanging.

#### Steps:
1. Issue a `SIGINT` (Ctrl+C) to the running `go run ./cmd/lab` process.
2. Observe shutdown logs in the terminal.

#### Expected Terminal Log Output:
```
^C[app] Shutdown signal received...
[metering_chain] Stopping workers and draining queues...
[metering_chain] Async workers drained successfully within 5s.
[metering_wal] WAL stopped cleanly.
[app] Process terminated cleanly.
```

---

### Scenario 9: Subscription Lifecycle FSM State Transitions

**Objective:** Test the subscription finite state machine transitions (`trial` $\rightarrow$ `active` $\rightarrow$ `past_due` $\rightarrow$ `cancelled`).

#### Step 1: Transition Subscription to `active`

* **Linux / macOS:**
  ```bash
  curl -X POST http://localhost:8080/v1/subscription/enterprise-corp-01/event \
    -H "Content-Type: application/json" \
    -d '{"event": "activate"}' | jq .
  ```

#### Step 2: Transition Subscription to `past_due` (Payment Failed)

* **Linux / macOS:**
  ```bash
  curl -X POST http://localhost:8080/v1/subscription/enterprise-corp-01/event \
    -H "Content-Type: application/json" \
    -d '{"event": "payment_failed"}' | jq .
  ```

#### Expected Output:
```json
{
  "api_key": "enterprise-corp-01",
  "from": "active",
  "event": "payment_failed",
  "to": "past_due"
}
```

---

## 5. Backend Verification Commands (PostgreSQL & Redis CLI)

### PostgreSQL Direct Database Verification

Connect to PostgreSQL container to verify persistent event records:

```bash
docker exec -it metering-postgres psql -U user -d metering_db -c "SELECT event_id, tenant_key, service_id, metric_id, quantity, timestamp FROM metering_events;"
```

```bash
docker exec -it metering-postgres psql -U user -d metering_db -c "SELECT subscription_id, tenant_key, service_id, timezone, status FROM service_subscriptions;"
```

---

### Redis Cache Direct Verification

Connect to Redis container to verify sorted sets and hash keys:

```bash
# Inspect subscriptions hash
docker exec -it metering-redis redis-cli HGETALL "metering:enterprise-corp-01:subs"

# Inspect event sorted set members by score range
docker exec -it metering-redis redis-cli ZRANGE "metering:enterprise-corp-01:ai-completion:events" 0 -1 WITHSCORES
```

---

## 6. Troubleshooting & FAQ

### Q1: `METERING_POSTGRES_DSN` fails to connect on macOS / Windows
**Solution:** If running PostgreSQL inside Docker, ensure your DSN uses `localhost:5432` when testing from host, or `postgres:5432` when testing container-to-container inside Docker network.

### Q2: How do I verify WAL segment rotation on disk?
**Solution:** Inspect the `./data/wal` directory. You will see active and sealed `.jsonl` files (e.g. `segment_0001.jsonl`). As segments are replayed and confirmed written to database backends, sealed segments are deleted automatically.

### Q3: Why do duplicate event IDs return `202 Accepted`?
**Solution:** Ingestion endpoints return `202 Accepted` asynchronously. The deduplication layer (`EventDedup`) silently filters duplicate `event_id` keys in memory before applying writes to storage engines, preserving API idempotency without throwing 500 errors to client callers.

---

## 7. Low-Level Infrastructure & Database/Redis Deep Testing Suite

This section provides advanced SRE & DevOps test procedures for validating low-level health, memory limits, connection pools, and resilience of PostgreSQL, Redis, and OS kernel resources under load.

---

### Scenario 10: PostgreSQL Connection Pool & Lock Contention Test

**Objective:** Validate database connection pool configuration (`MaxConns = 10`, `MinConns = 2`) and ensure row lock contention operates properly without leaking connections.

#### Step 1: Inspect Active PostgreSQL Connection Pool State
Execute direct SQL inside the PostgreSQL container to inspect active client connections and state:

```bash
docker exec -it metering-postgres psql -U user -d metering_db -c "
SELECT pid, usename, client_addr, state, backend_type, query_start, query 
FROM pg_stat_activity 
WHERE datname = 'metering_db';"
```

#### Step 2: Simulate Database Lock Contention
In one terminal, open a transaction and lock the `service_subscriptions` row:
```bash
docker exec -it metering-postgres psql -U user -d metering_db -c "
BEGIN;
SELECT * FROM service_subscriptions WHERE tenant_key = 'enterprise-corp-01' FOR UPDATE;
SELECT pg_sleep(10);
COMMIT;"
```

In a second terminal, execute a concurrent subscription registration HTTP request:
```bash
curl -X POST http://localhost:8080/v1/billing/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_key": "enterprise-corp-01",
    "service_id": "ai-completion",
    "plan_id": "locked-test-plan",
    "charge_type": "recurring_monthly",
    "timezone": "UTC",
    "anchor_time": "2026-08-01T00:00:00Z"
  }'
```

#### Verification:
1. The HTTP request blocks safely until lock release or timeout.
2. The PostgreSQL connection pool does not leak connections (`pg_stat_activity` count returns to idle baseline).
3. The engine circuit breaker does NOT trip for row lock contention (query-level wait vs connection failure).

---

### Scenario 11: Redis Memory Eviction, TTL & Slowlog Inspection

**Objective:** Audit Redis memory allocation, inspect TTL expiration on sorted sets, and verify zero slow command execution warnings.

#### Step 1: Audit Redis Memory Policy & Usage
```bash
# Check memory allocation and maxmemory policy
docker exec -it metering-redis redis-cli INFO memory

# Verify maxmemory policy configuration
docker exec -it metering-redis redis-cli CONFIG GET maxmemory-policy
```
> **DevOps Recommendation:** Set `maxmemory-policy noeviction` or `volatile-ttl` for metering data to prevent silent data loss of un-aggregated events.

#### Step 2: Verify Sorted Set Event Key TTL Expiration
Verify that event sorted set keys (`metering:{tenant}:{service}:events`) have a valid TTL (default 90 days = 7,776,000 seconds):
```bash
docker exec -it metering-redis redis-cli TTL "metering:enterprise-corp-01:ai-completion:events"
```
*Expected Output:* Integer value near `7776000` (or `no expire` `-1` if explicitly set).

#### Step 3: Inspect Redis Slow Log
Verify zero Redis latency spikes under high load:
```bash
docker exec -it metering-redis redis-cli SLOWLOG GET 10
```

---

### Scenario 12: Network Partition & Partial Packet Loss Simulation

**Objective:** Test engine resilience under partial network degradation (e.g. 50% packet drop or network disconnection between app and Redis/DB).

#### Step 1: Disconnect App Container / Disconnect Redis Network
Using Docker network controls, disconnect Redis from the application network:
```bash
docker network disconnect bridge metering-redis
```

#### Step 2: Ingest Continuous Event Stream
Send 10 metering events via cURL:
```bash
for i in {1..10}; do
  curl -s -X POST http://localhost:8080/v1/billing/events \
    -H "Content-Type: application/json" \
    -d "{
      \"event_id\": \"evt_net_$i\",
      \"tenant_key\": \"enterprise-corp-01\",
      \"service_id\": \"ai-completion\",
      \"metric_id\": \"tokens\",
      \"unit\": \"tokens\",
      \"quantity\": 100,
      \"timestamp\": \"2026-08-06T10:00:00Z\"
    }"
done
```

#### Step 3: Verify Health Endpoint & WAL Capture
```bash
curl -s http://localhost:8080/v1/billing/storage/health | jq .
```
- Redis `healthy` status transitions to `false`, `circuit_state` shows `open`.
- RAM L1 store and Disk WAL capture all 10 events cleanly with zero event drops.

#### Step 4: Reconnect Network & Verify Auto-Recovery
```bash
docker network connect bridge metering-redis
```
Wait ~5 seconds. Verify WAL depth returns to 0 and Redis circuit state resets to `closed`.

---

### Scenario 13: File Descriptor & System Resource Limit Audit (`ulimit` & Socket Reuse)

**Objective:** Ensure host OS kernel limits support high socket turnover and file descriptors for WAL segment logging.

#### Step 1: Check Open File Descriptor Limits (Linux / macOS)
```bash
ulimit -n
```
> **Recommendation:** Ensure `ulimit -n` is set to at least `65536` in production systemd unit files or Docker daemon configs.

#### Step 2: Check Active TCP Socket Connection States
Inspect active TCP sockets connected to PostgreSQL (port 5432) and Redis (port 6379):

* **Linux (bash):**
  ```bash
  ss -tulpn | grep -E '5432|6379|8080'
  ```

* **macOS (zsh):**
  ```bash
  netstat -an | grep -E '5432|6379|8080'
  ```

* **Windows (PowerShell):**
  ```powershell
  Get-NetTCPConnection -LocalPort 5432, 6379, 8080 | Format-Table LocalAddress, LocalPort, RemoteAddress, RemotePort, State
  ```

---

### Scenario 14: Direct DB & Redis Schema, Index & Memory Footprint Verification

**Objective:** Audit database table indexes and primary keys to guarantee $O(1)$ / $O(\log N)$ lookup performance.

#### Step 1: Verify PostgreSQL Table Indexes
```bash
docker exec -it metering-postgres psql -U user -d metering_db -c "
SELECT tablename, indexname, indexdef 
FROM pg_indexes 
WHERE schemaname = 'public';"
```

*Expected Indexes:*
- `pk_service_subscriptions` ON `service_subscriptions(tenant_key, service_id)`
- `metering_events_pkey` ON `metering_events(event_id)`
- `idx_metering_events_tenant_time` ON `metering_events(tenant_key, service_id, timestamp)`

#### Step 2: Verify PostgreSQL Table Storage Sizes
```bash
docker exec -it metering-postgres psql -U user -d metering_db -c "
SELECT relname AS table_name, 
       pg_size_pretty(pg_total_relation_size(relid)) AS total_size 
FROM pg_stat_user_tables;"
```

