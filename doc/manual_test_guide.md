# Manual Test Guide — Tenant Catalog (RegisterService + Full CRUD)

This guide covers local testing using `curl` and PowerShell, including success paths,
failure injection, and database verification. Designed to later become automated
integration tests.

---

## Prerequisites

```powershell
# 1. Start the server (memory-only mode — no external deps required)
cd "d:\AK-FILES\Golang April\ai-saas-lab\ai-saas-lab"
go run ./cmd/lab

# Server starts on http://localhost:8080 (default HTTP_PORT)
```

```powershell
# 2. Optional: full stack with Redis + Postgres
# Create config.env with:
#   CATALOG_BACKENDS=memory,redis,postgres
#   CATALOG_REDIS_ADDR=localhost:6379
#   CATALOG_POSTGRES_DSN=postgres://postgres:postgres@localhost:5432/aisaas?sslmode=disable
#   CATALOG_WAL_ENABLED=true

# Start Redis (Docker)
docker run -d --name redis -p 6379:6379 redis:7-alpine

# Start Postgres (Docker)
docker run -d --name postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=aisaas -p 5432:5432 postgres:16-alpine
```

---

## Step 1 — Create an API Key

```powershell
# The auth module auto-creates keys via POST /v1/auth/keys
$response = Invoke-RestMethod -Method POST -Uri "http://localhost:8080/v1/auth/keys" `
    -ContentType "application/json" `
    -Body '{"plan":"growth"}'

$API_KEY = $response.key
Write-Host "API Key: $API_KEY"
```

```bash
# curl equivalent
API_KEY=$(curl -s -X POST http://localhost:8080/v1/auth/keys \
  -H "Content-Type: application/json" \
  -d '{"plan":"growth"}' | jq -r '.key')
echo "API Key: $API_KEY"
```

---

## Step 2 — Register a Service (Happy Path → 201)

```powershell
Invoke-RestMethod -Method POST -Uri "http://localhost:8080/v1/tenant/catalog/services" `
    -Headers @{ "X-API-Key" = $API_KEY; "X-Request-ID" = "req-001" } `
    -ContentType "application/json" `
    -Body '{"service_id":"ai-writer","name":"AI Writer Engine","description":"LLM-based writing assistant"}'
```

```bash
curl -s -X POST http://localhost:8080/v1/tenant/catalog/services \
  -H "X-API-Key: $API_KEY" \
  -H "X-Request-ID: req-001" \
  -H "Content-Type: application/json" \
  -d '{"service_id":"ai-writer","name":"AI Writer Engine","description":"LLM-based writing assistant"}' | jq
```

**Expected response (201 Created):**
```json
{
  "service_id": "ai-writer",
  "name": "AI Writer Engine",
  "description": "LLM-based writing assistant",
  "created_at": "2026-08-14T07:00:00Z"
}
```

---

## Step 3 — Register a Metric

```powershell
Invoke-RestMethod -Method POST -Uri "http://localhost:8080/v1/tenant/catalog/metrics" `
    -Headers @{ "X-API-Key" = $API_KEY } `
    -ContentType "application/json" `
    -Body '{"metric_id":"prompt-tokens","service_id":"ai-writer","unit":"count","name":"Prompt Tokens"}'
```

```bash
curl -s -X POST http://localhost:8080/v1/tenant/catalog/metrics \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"metric_id":"prompt-tokens","service_id":"ai-writer","unit":"count","name":"Prompt Tokens"}' | jq
```

**Expected: 201 Created**

---

## Step 4 — Register a Plan

```powershell
$planBody = @{
    plan_id    = "pro-writer-v1"
    service_id = "ai-writer"
    name       = "Pro Writer Tier"
    rates      = @{ "prompt-tokens" = 0.002 }
    included_quotas = @{ "prompt-tokens" = 100000 }
} | ConvertTo-Json -Depth 5

Invoke-RestMethod -Method POST -Uri "http://localhost:8080/v1/tenant/catalog/plans" `
    -Headers @{ "X-API-Key" = $API_KEY } `
    -ContentType "application/json" `
    -Body $planBody
```

```bash
curl -s -X POST http://localhost:8080/v1/tenant/catalog/plans \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "plan_id": "pro-writer-v1",
    "service_id": "ai-writer",
    "name": "Pro Writer Tier",
    "rates": {"prompt-tokens": 0.002},
    "included_quotas": {"prompt-tokens": 100000}
  }' | jq
```

**Expected: 201 Created**

---

## Step 5 — GET Single Entity

```bash
# Get single service by ID
curl -s "http://localhost:8080/v1/tenant/catalog/services/ai-writer" \
  -H "X-API-Key: $API_KEY" | jq

# Get single metric by ID
curl -s "http://localhost:8080/v1/tenant/catalog/metrics/prompt-tokens" \
  -H "X-API-Key: $API_KEY" | jq

# Get single plan by ID
curl -s "http://localhost:8080/v1/tenant/catalog/plans/pro-writer-v1" \
  -H "X-API-Key: $API_KEY" | jq
```

**Expected: 200 OK with descriptor JSON**

---

## Step 6 — GET Lists + Overview

```bash
# List all services
curl -s "http://localhost:8080/v1/tenant/catalog/services" \
  -H "X-API-Key: $API_KEY" | jq

# Full overview
curl -s "http://localhost:8080/v1/tenant/catalog/overview" \
  -H "X-API-Key: $API_KEY" | jq
```

**Expected overview response:**
```json
{
  "tenant_key": "tk_...",
  "services": [{"service_id": "ai-writer", ...}],
  "metrics":  [{"metric_id": "prompt-tokens", ...}],
  "plans":    [{"plan_id": "pro-writer-v1", ...}]
}
```

---

## Failure Cases

### F1 — Unauthenticated request → 401

```bash
curl -s -X POST http://localhost:8080/v1/tenant/catalog/services \
  -H "Content-Type: application/json" \
  -d '{"service_id":"test","name":"Test"}' | jq
# Expected: {"error":"authentication required: missing X-API-Key or Authorization Bearer header"}
```

### F2 — Duplicate service_id → 409 Conflict

```bash
# Register "ai-writer" again after Step 2
curl -s -X POST http://localhost:8080/v1/tenant/catalog/services \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"service_id":"ai-writer","name":"Different Name"}' | jq
# Expected: 409 {"error":"register service: service \"ai-writer\" already exists"}
```

### F3 — Invalid service_id charset → 422

```bash
curl -s -X POST http://localhost:8080/v1/tenant/catalog/services \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"service_id":"AI Writer!","name":"Test"}' | jq
# Expected: 422 {"error":"register service: validation failed on \"service_id\": ..."}
```

### F4 — service_id starting with hyphen → 422

```bash
curl -s -X POST http://localhost:8080/v1/tenant/catalog/services \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"service_id":"-bad","name":"Test"}' | jq
# Expected: 422
```

### F5 — Metric for unregistered service → 422

```bash
curl -s -X POST http://localhost:8080/v1/tenant/catalog/metrics \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"metric_id":"tokens","service_id":"nonexistent","unit":"count"}' | jq
# Expected: 422 {"error":"...referenced service_id \"nonexistent\" must be registered..."}
```

### F6 — Plan with unregistered metric → 422

```bash
curl -s -X POST http://localhost:8080/v1/tenant/catalog/plans \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"plan_id":"bad-plan","service_id":"ai-writer","rates":{"ghost-metric":0.001}}' | jq
# Expected: 422 {"error":"...metric_id \"ghost-metric\" must be registered..."}
```

### F7 — Request body too large → 413

```powershell
# Generate 65KB body
$bigBody = '{"service_id":"x","name":"' + ('a' * 65000) + '"}'
try {
    Invoke-RestMethod -Method POST -Uri "http://localhost:8080/v1/tenant/catalog/services" `
        -Headers @{ "X-API-Key" = $API_KEY } `
        -ContentType "application/json" -Body $bigBody
} catch {
    $_.Exception.Response.StatusCode  # Should be 413
}
```

### F8 — GET nonexistent entity → 404

```bash
curl -s "http://localhost:8080/v1/tenant/catalog/services/nonexistent" \
  -H "X-API-Key: $API_KEY" | jq
# Expected: 404 {"error":"service \"nonexistent\" not found"}
```

---

## Verify Databases (Full Stack Mode)

### Redis Verification

```bash
redis-cli -n 0

# Check catalog keys for your tenant
KEYS catalog:*

# Inspect services hash for tenant
HGETALL catalog:<tenant_key>:services

# Check TTL (should be ~86400 seconds = 24h)
TTL catalog:<tenant_key>:services
```

### PostgreSQL Verification

```bash
psql "postgres://postgres:postgres@localhost:5432/aisaas?sslmode=disable"
```

```sql
-- View all registered services
SELECT tenant_key, service_id, name, created_at, updated_at FROM tenant_services;

-- View metrics
SELECT tenant_key, metric_id, service_id, unit, updated_at FROM tenant_metrics;

-- View plans
SELECT tenant_key, plan_id, service_id, version, active FROM tenant_plans;
```

### WAL Verification (Simulate Backend Failure)

```powershell
# 1. Kill Redis / Postgres while server runs
# 2. Make a register call — should still 201 (L1 succeeds)
# 3. Check WAL directory for sealed segments
Get-ChildItem ".\data\wal\catalog\" | Sort-Object LastWriteTime | Format-Table Name, Length

# 4. Restart Redis / Postgres
# 5. Within 5 seconds the ReplayLoop will pick up sealed segments
# 6. Sealed files should be deleted after successful replay
Get-ChildItem ".\data\wal\catalog\" | Measure-Object | Select-Object Count
# Expected: 0 or only the active segment
```

---

## X-Request-ID Tracing

```bash
# Send with custom request ID
curl -sv "http://localhost:8080/v1/tenant/catalog/services" \
  -H "X-API-Key: $API_KEY" \
  -H "X-Request-ID: my-trace-id-123" 2>&1 | grep -i "x-request-id"
# Expected in response headers: X-Request-ID: my-trace-id-123
```

---

## EventBus Verification (Log Watch)

```powershell
# Server logs should show publish when service registered
go run ./cmd/lab 2>&1 | Select-String "tenant.service.registered"
```

The `ServiceRegisteredEvent` is published with:
```json
{
  "tenant_key": "tk_...",
  "service_id": "ai-writer",
  "name": "AI Writer Engine",
  "registered_at": "2026-08-14T07:00:00Z"
}
```

---

## Automated Integration Test Equivalent

Each manual case above maps directly to an `httptest` scenario. See:
- [`internal/modules/tenant/tenant_test.go`](../internal/modules/tenant/tenant_test.go) — HTTP-level security + CRUD + referential integrity
- [`internal/modules/tenant/service_test.go`](../internal/modules/tenant/service_test.go) — service-layer unit tests with mock catalog
- [`internal/kernel/tenant_catalog_test.go`](../internal/kernel/tenant_catalog_test.go) — CatalogChain, validation, sentinel error matching
