# Subscription & Entitlement Module — Developer & QA Manual Testing Guide

**Platform**: AI SaaS Lab — Metering-as-a-Service (MaaS) Infrastructure  
**Package**: `aisaaslab/internal/modules/subscription`  
**File**: `doc/guide_subscription_manual_testing.md`  
**Target Audience**: Developers, QA Engineers, Manual Testers, and Frontend Client Integrators  
**Status**: Authoritative Manual Testing Runbook  

---

## 1. Executive Summary & Testing Scope

This runbook provides step-by-step instructions for **Manual Testing & Developer Verification** of the **Subscription & Entitlement Module**. It covers:
1. **cURL / API Terminal Verification** (for backend engineers & API testers).
2. **Interactive UI Dashboard Verification** (for QA manual testers & product managers).
3. **HTTP Middleware & Entitlement Enforcement Verification** (for security & tenant isolation testing).

---

## 2. Environment Setup & Prerequisites

### 2.1 Start the Go Backend Server
Open a terminal in the project root directory and run:

```powershell
cd "d:\AK-FILES\Golang April\ai-saas-lab\ai-saas-lab"
go run ./cmd/lab/main.go
```
- **Port**: `http://localhost:8080`
- **Health Check**: `curl.exe http://localhost:8080/healthz` ➔ `200 OK`

### 2.2 Start the React Frontend Dashboard
Open a second terminal window and run:

```powershell
cd "d:\AK-FILES\Golang April\ai-saas-lab\ai-saas-lab\client"
npm.cmd run dev
```
- **URL**: `http://localhost:3000`

### 2.3 Pre-Seeded Keys & Test Data

| Key Name / Slug | Purpose | Initial State |
| :--- | :--- | :--- |
| `demo-key-starter` | Starter Tier Tenant Key | `active` / `starter` plan |
| `sk_lab_pro_8f92a1c4e7b3091d` | Pro Tier Production Key | `active` / `pro` plan |
| `sk_lab_free_3c7d91e8b2a5` | Free Sandbox Key | `active` / `starter` plan |

---

## 3. Developer & API Testing Scenarios (cURL)

### Scenario 1: List All Subscription Plans Catalog
Verify that the system returns all available product plan tiers.

```powershell
curl.exe -s http://localhost:8080/v1/subscription/plans
```

**Expected Response (`200 OK`)**:
```json
{
  "plans": [
    {
      "id": "starter",
      "name": "Starter Tier",
      "entitlements": {
        "total_tokens": { "metric_id": "total_tokens", "allowed": true, "quota": 1000000 }
      }
    },
    {
      "id": "pro",
      "name": "Pro Tier",
      "entitlements": {
        "total_tokens": { "metric_id": "total_tokens", "allowed": true, "quota": 10000000 }
      }
    }
  ]
}
```

---

### Scenario 2: Register a Custom Subscription Plan
Test registering a new product plan tier with custom entitlement bounds.

```powershell
curl.exe -s -X POST http://localhost:8080/v1/subscription/plans `
  -H "Content-Type: application/json" `
  -d '{
    "id": "growth_tier",
    "name": "Growth Scale Tier",
    "entitlements": {
      "total_tokens": { "metric_id": "total_tokens", "allowed": true, "quota": 50000000 }
    }
  }'
```

**Expected Response (`201 Created`)**:
```json
{
  "id": "growth_tier",
  "name": "Growth Scale Tier",
  "entitlements": {
    "total_tokens": { "metric_id": "total_tokens", "allowed": true, "quota": 50000000 }
  }
}
```

---

### Scenario 3: Inspect Tenant Subscription Contract
Retrieve the active contract and FSM state for a given tenant key.

```powershell
curl.exe -s http://localhost:8080/v1/subscription/sk_lab_pro_8f92a1c4e7b3091d
```

**Expected Response (`200 OK`)**:
```json
{
  "subscription_id": "sub_pro_sk_lab_pro_8f92a1c4e7b3091d",
  "tenant_key": "sk_lab_pro_8f92a1c4e7b3091d",
  "plan_id": "pro",
  "state": "active",
  "anchor_time": "2026-08-23T07:25:00Z",
  "timezone": "UTC",
  "is_usable": true,
  "entitlements": {
    "total_tokens": { "metric_id": "total_tokens", "allowed": true, "quota": 10000000 }
  }
}
```

---

### Scenario 4: Create / Upgrade Tenant Contract
Bind a tenant key to a new plan tier with timezone specified.

```powershell
curl.exe -s -X POST http://localhost:8080/v1/subscription/contracts `
  -H "Content-Type: application/json" `
  -d '{
    "tenant_key": "tenant_manual_qa_01",
    "plan_id": "pro",
    "timezone": "America/New_York"
  }'
```

**Expected Response (`201 Created`)**:
```json
{
  "subscription_id": "sub_pro_tenant_manual_qa_01",
  "tenant_key": "tenant_manual_qa_01",
  "plan_id": "pro",
  "state": "active",
  "timezone": "America/New_York",
  "is_usable": true
}
```

---

### Scenario 5: Simulate FSM Lifecycle State Transitions
Test state machine transition logic and event handling.

#### 5.1 Simulate Payment Failure (`active` ➔ `past_due`)
```powershell
curl.exe -s -X POST http://localhost:8080/v1/subscription/tenant_manual_qa_01/event `
  -H "Content-Type: application/json" `
  -d '{ "event": "payment_failed" }'
```
**Expected Response (`200 OK`)**:
```json
{
  "tenant_key": "tenant_manual_qa_01",
  "from": "active",
  "event": "payment_failed",
  "to": "past_due"
}
```

#### 5.2 Reactivate Subscription (`past_due` ➔ `active`)
```powershell
curl.exe -s -X POST http://localhost:8080/v1/subscription/tenant_manual_qa_01/event `
  -H "Content-Type: application/json" `
  -d '{ "event": "reactivate" }'
```
**Expected Response (`200 OK`)**:
```json
{
  "tenant_key": "tenant_manual_qa_01",
  "from": "past_due",
  "event": "reactivate",
  "to": "active"
}
```

#### 5.3 Cancel Subscription (`active` ➔ `cancelled`)
```powershell
curl.exe -s -X POST http://localhost:8080/v1/subscription/tenant_manual_qa_01/event `
  -H "Content-Type: application/json" `
  -d '{ "event": "cancel" }'
```
**Expected Response (`200 OK`)**:
```json
{
  "tenant_key": "tenant_manual_qa_01",
  "from": "active",
  "event": "cancel",
  "to": "cancelled"
}
```

#### 5.4 Test Invalid Transition Guard (`cancelled` ➔ `payment_failed`)
Attempt an invalid state transition matrix move to verify guard protection.

```powershell
curl.exe -s -X POST http://localhost:8080/v1/subscription/tenant_manual_qa_01/event `
  -H "Content-Type: application/json" `
  -d '{ "event": "payment_failed" }'
```
**Expected Response (`409 Conflict`)**:
```json
{
  "error": "invalid state transition from cancelled via payment_failed"
}
```

---

## 4. QA Manual Tester UI Dashboard Checklist

Open `http://localhost:3000` in your web browser and click on the **Subscriptions & FSM** tab in the left sidebar.

```
+-----------------------------------------------------------------------------------+
| QA UI CHECKLIST MATRIX                                                            |
+---+----------------------------+-------------------------------------+------------+
| # | Test Action                | Expected UI Behavior                | Result     |
+---+----------------------------+-------------------------------------+------------+
| 1 | Inspect Tenant Key         | Type key in header bar & click      | Contract   |
|   |                            | Inspect. KPI stats update live.     | [ PASS ]   |
| 2 | State Badges               | Verify green Active vs amber Past   | Color/Icon |
|   |                            | Due vs red Cancelled badges.        | [ PASS ]   |
| 3 | Fire FSM Action Buttons    | Click Payment Failure button.       | Toast & Log|
|   |                            | Badge switches to Past Due (amber). | [ PASS ]   |
| 4 | Fire Reactivate Button     | Click Reactivate button.            | Badge turns|
|   |                            | State switches to Active (green).   | [ PASS ]   |
| 5 | Quota Slider Drag          | Move usage slider past quota bound. | Bar turns  |
|   |                            | Warning percentage appears.         | [ PASS ]   |
| 6 | Register Custom Plan       | Click "+ Register Plan" button,     | Plan card  |
|   |                            | fill form, submit modal.            | appears    |
| 7 | Switch Tenant Plan Tier    | Click "Switch to [Plan Name]" on    | Active Card|
|   |                            | any non-current plan card.          | updates    |
+---+----------------------------+-------------------------------------+------------+
```

---

## 5. Error Handling & Edge Case Matrix

| Error Scenario | Action / Input | Expected Status | Expected Error JSON |
| :--- | :--- | :---: | :--- |
| **Missing Tenant Key** | `GET /v1/subscription/ ` | `404 Not Found` | Route 404 |
| **Unknown Plan ID** | `POST /v1/subscription/contracts` with `plan_id: "fake_plan"` | `404 Not Found` | `{ "error": "plan not found: fake_plan" }` |
| **Invalid State Event** | `POST /v1/subscription/{key}/event` with invalid event | `409 Conflict` | `{ "error": "invalid state transition..." }` |
| **Malformed JSON Payload** | `POST /v1/subscription/plans` with malformed JSON | `400 Bad Request` | `{ "error": "invalid JSON body..." }` |
| **Invalid Timezone** | `POST /v1/subscription/contracts` with `timezone: "Invalid/Zone"` | `201 Created` | Safely falls back to `"UTC"` |
