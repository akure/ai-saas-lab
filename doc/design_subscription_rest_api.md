# Subscription & Entitlement Module — REST API Design & Implementation Specification

**Platform**: AI SaaS Lab — Metering-as-a-Service (MaaS) Infrastructure  
**Package**: `aisaaslab/internal/modules/subscription`  
**File**: `doc/design_subscription_rest_api.md`  
**Status**: Authoritative Architectural Standard  

---

## 1. Executive Summary & Design Philosophy

The REST API for the **Subscription & Entitlement Module** provides the external HTTP interface for plan tier governance, contract lifecycle management via a Finite State Machine (FSM), and entitlement quota evaluations.

This document defines the REST standards, HTTP endpoint conventions, JSON response schemas, status code taxonomies, and Go implementation guidelines followed across the module.

---

## 2. Resource Naming & URI Hierarchy

### 2.1 Core Rules
- **Resource Nouns Only**: Endpoint URIs strictly use plural nouns identifying resources (`/v1/subscription/plans`), avoiding RPC action verbs in the path (`/v1/getPlans`).
- **Hierarchical Pathing**: Sub-resources reflect clear parent-child ownership (`/v1/subscription/plans/{id}`).
- **Hyphenated Lowercase Slugs**: Multi-word URI paths use lowercase alphanumeric characters separated by hyphens.
- **Snake-Case Field Identifiers**: All JSON payload keys use `snake_case` (`tenant_key`, `plan_id`, `subscription_id`, `anchor_time`, `is_usable`).

### 2.2 Endpoint Matrix

| Method | Endpoint Path | Description | Success Status | Error Statuses |
| :--- | :--- | :--- | :--- | :--- |
| `GET` | `/v1/subscription/plans` | List all registered plan catalog tiers | `200 OK` | `500` |
| `POST` | `/v1/subscription/plans` | Register a new plan tier | `201 Created` | `400` |
| `GET` | `/v1/subscription/plans/{id}` | Retrieve details for a specific plan tier | `200 OK` | `404` |
| `GET` | `/v1/subscription/{key}` | Retrieve tenant contract, plan, and entitlements | `200 OK` | `400` |
| `POST` | `/v1/subscription/{key}/event` | Fire FSM lifecycle transition event | `200 OK` | `400`, `409` |
| `POST` | `/v1/subscription/contracts` | Create or update a subscription contract | `201 Created` | `400`, `404` |

---

## 3. Standardized Request & Response JSON Schemas

### 3.1 Top-Level JSON Envelopes for Collections
Bare JSON arrays (`[...]`) are strictly prohibited for collection responses. Collections are wrapped in top-level JSON objects to support pagination metadata (`count`, `page`, `has_more`) without breaking contract compatibility.

**Response Schema (`GET /v1/subscription/plans`)**:
```json
{
  "plans": [
    {
      "id": "starter",
      "name": "Starter Tier",
      "entitlements": {
        "total_tokens": {
          "metric_id": "total_tokens",
          "allowed": true,
          "quota": 1000000
        }
      }
    },
    {
      "id": "pro",
      "name": "Pro Tier",
      "entitlements": {
        "total_tokens": {
          "metric_id": "total_tokens",
          "allowed": true,
          "quota": 10000000
        }
      }
    }
  ]
}
```

### 3.2 Tenant Subscription Inspect Schema (`GET /v1/subscription/{key}`)
Returns full contract state, active plan ID, usability status, and effective metric entitlements.

**Response Schema (`200 OK`)**:
```json
{
  "subscription_id": "sub_pro_tenant_102",
  "tenant_key": "tenant_102",
  "plan_id": "pro",
  "state": "active",
  "anchor_time": "2026-08-01T00:00:00Z",
  "timezone": "UTC",
  "is_usable": true,
  "entitlements": {
    "total_tokens": {
      "metric_id": "total_tokens",
      "allowed": true,
      "quota": 10000000
    }
  }
}
```

### 3.3 Standard Error Response Structure
All error responses must set `Content-Type: application/json` and return a structured JSON error body:

```json
{
  "error": "plan_id enterprise_plus does not exist"
}
```

---

## 4. HTTP Status Code Taxonomy

| Status Code | Description & Usage Scenario |
| :--- | :--- |
| **`200 OK`** | Successful resource lookup or state transition response (`GET`, `POST event`). |
| **`201 Created`** | Successful creation of a plan tier or subscription contract (`POST`). |
| **`400 Bad Request`** | Validation failure, missing required fields (`tenant_key`, `plan_id`), or malformed JSON. |
| **`401 Unauthorized`** | Missing or invalid authentication token/API key. |
| **`403 Forbidden`** | Subscription state is inactive, past due, or cancelled (`!State.IsUsable()`). |
| **`404 Not Found`** | Requested resource (`plan_id`) does not exist in the system catalog. |
| **`409 Conflict`** | Invalid FSM state transition (e.g. attempting `cancel` from state `cancelled`). |
| **`429 Too Many Requests`** | Tenant usage has exceeded plan entitlement quota. |
| **`500 Internal Error`** | Unhandled internal runtime exception. |

---

## 5. FSM State Transition Endpoint Design

Lifecycle state transitions are invoked via `POST /v1/subscription/{key}/event`.

**Request Schema**:
```json
{
  "event": "payment_failed"
}
```

**Response Schema (`200 OK`)**:
```json
{
  "tenant_key": "tenant_102",
  "from": "active",
  "event": "payment_failed",
  "to": "past_due"
}
```

---

## 6. Go Implementation Best Practices

1. **Centralized Error Helpers**: Standardized error writing via `writeJSONError(w, message, statusCode)` ensures predictable JSON content types and status codes.
2. **Sanitization**: String inputs (`tenant_key`, `plan_id`, `event`) are stripped of whitespace via `strings.TrimSpace(...)`.
3. **Timezone Fallback Safety**: Invalid IANA timezone inputs default safely to `"UTC"`.
4. **Thin HTTP Handlers**: Handlers parse HTTP request DTOs and delegate domain logic directly to `subscription.Manager`.
