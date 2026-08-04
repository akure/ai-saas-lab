# Abstract SaaS Metering & Billing Engine

> A service-agnostic, timezone-aware, multi-tenant metering and billing infrastructure for any SaaS product.

---

## Table of Contents

1. [Why This Exists](#1-why-this-exists)
2. [Design Philosophy](#2-design-philosophy)
3. [Core Principles](#3-core-principles)
4. [Architecture Overview](#4-architecture-overview)
5. [Design Choices & Rationale](#5-design-choices--rationale)
6. [Data Structures & Contracts](#6-data-structures--contracts)
7. [The Cycle Window Algorithm](#7-the-cycle-window-algorithm)
8. [HTTP API Specification](#8-http-api-specification)
9. [Integration Contracts](#9-integration-contracts)
10. [Testing & Mocking](#10-testing--mocking)
11. [Limitations & Known Trade-offs](#11-limitations--known-trade-offs)
12. [Future Roadmap](#12-future-roadmap)

---

## 1. Why This Exists

Every SaaS company eventually builds a billing system. Most build it wrong — tightly coupled to their first product feature (AI tokens, API calls, seats), with hardcoded billing cycles tied to calendar months, with no regard for the user's timezone or subscription start date, and with a monolithic structure that makes adding a second pricing model feel like surgery.

The result is well-known: a **billing system that fights you** every time the product evolves.

This engine was designed from first principles to solve this permanently. It treats billing as an **infrastructure concern** — not a product concern — by providing a domain-agnostic metering and cycle window calculation layer that any service, in any SaaS product, can plug into without knowing anything about billing internals.

The engine exists to answer three simple questions correctly, for any service, for any user, anywhere in the world:

1. **What did this user consume?** → `MeteringEvent` ingestion
2. **Within which time window?** → `CurrentCycleWindow` resolver
3. **Across which services?** → Per-service `ServiceBillingStatement`

---

## 2. Design Philosophy

> *"The best billing system is one you can forget about while building your product."*

The three philosophies guiding every decision in this engine are:

### I. Metering is infrastructure; pricing is business logic.
The engine is deliberately split in two layers. This module handles **metering** — the accurate recording and windowed aggregation of billable events. **Pricing** (how much a unit costs, what currency, what discount) is a separate concern and belongs to a separate module that consumes metering data. This separation means the metering layer is stable and can be tested in complete isolation from any pricing changes.

### II. UTC storage, local reasoning.
All events are stored in UTC. All cycle window calculations are performed in the tenant's local timezone anchored to their specific subscription start date/time. This is the only correct approach to timezone-aware billing and eliminates entire categories of DST-related bugs that plague production billing systems.

### III. Each service subscription is sovereign.
A tenant subscribed to three services at different times, in different timezones, with different charge models, has three independent billing windows. The engine never merges them unless they are provably identical. This prevents cycle-window cross-contamination and enables genuinely independent per-service invoicing — a pattern required by any SaaS platform with add-on or à la carte services.

---

## 3. Core Principles

| Principle | Description |
| :--- | :--- |
| **Domain Agnosticism** | Zero compile-time knowledge of AI tokens, storage bytes, or seats. All billable activity is modeled as a generic `MeteringEvent`. |
| **Service-Anchor Sovereignty** | Each `ServiceSubscription` owns its `AnchorTime`, `Timezone`, and `ChargeType`. Cycle windows are independently calculated. |
| **UTC Ingestion / Local Resolution** | Events stored in UTC. Cycle boundaries computed in the tenant's IANA timezone. Correct across all DST transitions. |
| **Stateless Window Calculator** | `CurrentCycleWindow` is a pure function on `ServiceSubscription` — no database call, no shared state. O(1) time and space. |
| **Event-Driven Integration** | Services emit `MeteringEvent` onto the kernel EventBus. The billing module consumes asynchronously, with no compile-time coupling. |
| **Open Extension Points** | Pricing, subscription lifecycle, and invoicing are separate concerns by design. The engine provides clean interfaces for each. |
| **Kernel-Resident Types** | All metering types live in `internal/kernel/metering.go` to prevent import cycles. The billing module exposes them as aliases. |

---

## 4. Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                        SERVICE PRODUCER LAYER                        │
│                                                                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌─────────┐ │
│  │ AI Inference │  │Object Storage│  │  Service 1/2 │  │ Gateway │ │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └────┬────┘ │
└─────────┼─────────────────┼─────────────────┼────────────────┼──────┘
          │ metering.event  │ metering.event  │ metering.event │ POST
          ▼                 ▼                 ▼                ▼
┌─────────────────────────────────────────────────────────────────────┐
│           KERNEL EVENT BUS  ╱  POST /v1/billing/events              │
└───────────────────────────────────────┬─────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       BILLING MODULE FACADE                          │
│                                                                       │
│  1. ServiceSubscription Registry                                      │
│     TenantKey → ServiceID → ServiceSubscription                       │
│                                                                       │
│  2. Service Cycle Window Algorithm                                     │
│     CurrentCycleWindow(at) → [CycleStartUTC, CycleEndUTC)             │
│     Resolved per-service, per-tenant, per-timezone                    │
│                                                                       │
│  3. Event Aggregation Engine                                          │
│     Filters MeteringEvents within each service's exact cycle bounds   │
│                                                                       │
│  4. Subscription State Machine (FSM)                                  │
│     trial → active → past_due → cancelled                            │
└───────────────────────────────────────┬─────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    MULTI-TENANT KERNEL STORE                          │
│                                                                       │
│  serviceSubscriptions:  TenantKey → ServiceID → ServiceSubscription  │
│  meteringEvents:        TenantKey → []MeteringEvent (UTC log)        │
└───────────────────────────────────────┬─────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       REST QUERY LAYER                                │
│                                                                       │
│  GET  /v1/billing/{key}/overview             → TenantBillingOverview │
│  GET  /v1/billing/{key}/statement/{service}  → ServiceBillingStatement│
│  POST /v1/billing/subscriptions              → Register subscription │
│  POST /v1/billing/events                     → Ingest MeteringEvent  │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 5. Design Choices & Rationale

### 5.1 Why kernel-resident metering types?

Metering types (`MeteringEvent`, `ServiceSubscription`, etc.) live in `internal/kernel/metering.go`, not inside `internal/modules/billing`. This is deliberate.

If types lived inside `billing`, then `kernel/store.go` — which must persist these types — would need to import `billing`. And `billing` imports `kernel`. This creates a circular dependency that the Go compiler rejects.

By anchoring the types in `kernel`, the dependency graph becomes acyclic:

```
kernel/metering.go  (types defined here)
       ↑
kernel/store.go     (uses kernel types — no import needed)
       ↑
billing/module.go   (imports kernel, uses types via alias in billing/types.go)
```

The billing package re-exports these types as Go type aliases (`type MeteringEvent = kernel.MeteringEvent`), giving consumers a stable `billing.MeteringEvent` surface while avoiding duplication.

---

### 5.2 Why per-service subscription anchors instead of account-level billing?

Most billing systems anchor billing to an **account creation date** or to a **calendar month boundary** (e.g., the 1st of every month). This is wrong in practice for two reasons:

1. A user who subscribes to a second service three months after their first service should not have the second service's cycle retroactively aligned to the first service's start date.
2. Calendar-month boundaries introduce fairness issues for users who subscribed mid-month — they get a partial first cycle that complicates proration and metering.

This engine adopts **service-anchor-time billing**: the billing window for each service starts exactly when the tenant subscribed to that specific service and repeats on that same day-of-month/day-of-year offset going forward. This is the model used by Stripe, AWS, and Chargebee at scale.

---

### 5.3 Why UTC storage with IANA timezone resolution at query time?

Storing timestamps in the tenant's local timezone is a common mistake. It breaks when:
- The tenant's timezone observes DST (clocks shift forward/backward mid-cycle).
- The tenant changes their timezone preference.
- Server timezone differs from tenant timezone.

Storing in UTC and resolving cycle boundaries using `time.LoadLocation(s.Timezone)` at query time is the only approach that is both correct and auditable. The raw event log is immutable and timezone-invariant. The cycle window is always a derivation — never a stored state — so it is always recalculable.

---

### 5.4 Why a pure function for cycle window calculation?

`CurrentCycleWindow(at time.Time)` is implemented as a method on `ServiceSubscription` — a pure function with no side effects, no database calls, and no shared mutable state. Given the same `ServiceSubscription` and the same `at` timestamp, it always returns the same window. This makes it:

- **Trivially testable** — no mocks required, just construct the struct and call the function.
- **Safe to call concurrently** — no locking needed at the calculation layer.
- **Independently deployable** — the algorithm can be extracted and reused in a pricing module or reporting module without pulling in the full billing module.

---

### 5.5 Why EventBus + REST dual ingestion?

Internal service modules (AI completion, storage) emit `MeteringEvent` onto the in-process EventBus (`"metering.event"`) for zero-overhead, no-network, no-serialization event delivery. External systems or gateways that cannot publish to the in-process bus use `POST /v1/billing/events`. Both paths converge at `Store.RecordMeteringEvent`. Consumers of billing data never need to know which path was used.

---

### 5.6 Why a flat event log per tenant?

`meteringEvents` is indexed as `TenantKey → []MeteringEvent` — a flat append-only time-series log per tenant. This is intentional for the current stage:

- **Simple to reason about.** A flat log is easy to inspect, dump, and replay.
- **Window filtering is cheap.** A sequential scan over a tenant's event slice is O(n) in the number of events for that tenant, not O(N) across all tenants.
- **Future-compatible.** When a persistent store is introduced (PostgreSQL, ClickHouse, TimescaleDB), the storage contract (`RecordMeteringEvent` / `GetServiceBillingStatement`) is stable — the implementation just changes beneath it.

---

## 6. Data Structures & Contracts

### 6.1 `ChargeType`

```go
type ChargeType string

const (
    ChargeTypeMetered          ChargeType = "metered"           // Pay-as-you-go
    ChargeTypeRecurringMonthly ChargeType = "recurring_monthly" // Fixed monthly
    ChargeTypeRecurringYearly  ChargeType = "recurring_yearly"  // Fixed yearly
    ChargeTypeOneTime          ChargeType = "one_time"          // One-time purchase
)
```

---

### 6.2 `ServiceSubscription`

The foundational contract between a tenant and a specific service. Carries all information needed to independently resolve that service's billing cycle.

```go
type ServiceSubscription struct {
    SubscriptionID string     `json:"subscription_id"` // Unique subscription identifier
    TenantKey      string     `json:"tenant_key"`      // Customer/API key identifier
    ServiceID      string     `json:"service_id"`      // e.g. "ai-completion", "storage", "service1"
    PlanID         string     `json:"plan_id"`         // e.g. "pro-plan", "starter-plan"
    ChargeType     ChargeType `json:"charge_type"`     // metered | recurring_monthly | recurring_yearly | one_time
    Timezone       string     `json:"timezone"`        // IANA Timezone e.g. "America/New_York", "Asia/Tokyo"
    AnchorTime     time.Time  `json:"anchor_time"`     // Exact subscription start timestamp (UTC)
    Status         string     `json:"status"`          // "active" | "cancelled" | "past_due"
}
```

> **Design Note:** `Timezone` and `AnchorTime` together are sufficient to compute any billing window for this subscription without any additional metadata. This is the minimum required state.

---

### 6.3 `MeteringEvent`

An atomic, immutable record of a single billable unit of consumption from any service.

```go
type MeteringEvent struct {
    EventID   string                 `json:"event_id"`            // Idempotency key
    TenantKey string                 `json:"tenant_key"`          // Billing subject
    ServiceID string                 `json:"service_id"`          // Originating service
    MetricID  string                 `json:"metric_id"`           // e.g. "tokens", "requests", "gb_hours"
    Unit      string                 `json:"unit"`                // Human-readable unit string
    Quantity  int64                  `json:"quantity"`            // Consumed amount (always positive)
    Timestamp time.Time              `json:"timestamp"`           // UTC occurrence time
    Metadata  map[string]interface{} `json:"metadata,omitempty"`  // Optional context tags
}
```

> **Design Note:** `EventID` enables idempotent ingestion — a future deduplication layer can reject or merge duplicate event IDs without altering the storage schema.

---

### 6.4 `MetricSummary`

```go
type MetricSummary struct {
    MetricID    string `json:"metric_id"`
    Unit        string `json:"unit"`
    CycleTotal  int64  `json:"cycle_total"`  // Sum of Quantity within the cycle window
    TotalEvents int64  `json:"total_events"` // Event count (useful for rate/average calculations)
}
```

---

### 6.5 `ServiceBillingStatement`

The authoritative per-service usage report for a single billing cycle. Contains the exact UTC cycle bounds alongside metric aggregates.

```go
type ServiceBillingStatement struct {
    SubscriptionID string                    `json:"subscription_id"`
    TenantKey      string                    `json:"tenant_key"`
    ServiceID      string                    `json:"service_id"`
    PlanID         string                    `json:"plan_id"`
    ChargeType     ChargeType                `json:"charge_type"`
    Timezone       string                    `json:"timezone"`
    CycleStartUTC  time.Time                 `json:"cycle_start_utc"`
    CycleEndUTC    time.Time                 `json:"cycle_end_utc"`
    Metrics        map[string]*MetricSummary `json:"metrics"`
    GeneratedAt    time.Time                 `json:"generated_at"`
}
```

---

### 6.6 `TenantBillingOverview`

An enumeration of all service billing statements for a tenant at a given point in time. Does **not** merge or aggregate across services.

```go
type TenantBillingOverview struct {
    TenantKey         string                    `json:"tenant_key"`
    SubscriptionState string                    `json:"subscription_state"`
    Statements        []ServiceBillingStatement `json:"statements"`
    GeneratedAt       time.Time                 `json:"generated_at"`
}
```

---

## 7. The Cycle Window Algorithm

The cycle window resolver is the mathematical heart of the engine. Given a `ServiceSubscription` and a target timestamp `at`, it returns the half-open interval `[CycleStartUTC, CycleEndUTC)` that contains `at`.

### Invariants

- If `at < AnchorTime`: the subscription has not started yet. Returns `(AnchorTime, AnchorTime)` — an empty interval.
- Cycle boundaries are always computed in the **tenant's IANA timezone** to handle DST correctly.
- Computed boundaries are returned in **UTC** for consistent storage and comparison.

### Timeline Diagram

```
AnchorTime: Aug 15, 14:30 UTC  (e.g. user in "Asia/Kolkata", IST UTC+5:30)

Cycle 1:  [Aug 15 14:30 UTC ──────────────────────── Sep 15 14:30 UTC)
Cycle 2:                     [Sep 15 14:30 UTC ─────────────────────── Oct 15 14:30 UTC)
                                                          ▲
                                                     Query time (t)
                                             → belongs to Cycle 2
```

### Implementation

```go
func (s ServiceSubscription) CurrentCycleWindow(at time.Time) (startUTC, endUTC time.Time) {
    if at.Before(s.AnchorTime) {
        return s.AnchorTime, s.AnchorTime // not yet started
    }

    loc, err := time.LoadLocation(s.Timezone)
    if err != nil {
        loc = time.UTC // safe fallback
    }

    anchorLocal := s.AnchorTime.In(loc)
    atLocal := at.In(loc)

    switch s.ChargeType {
    case ChargeTypeRecurringYearly:
        years := atLocal.Year() - anchorLocal.Year()
        startLocal := anchorLocal.AddDate(years, 0, 0)
        if startLocal.After(atLocal) {
            startLocal = anchorLocal.AddDate(years-1, 0, 0)
        }
        return startLocal.UTC(), startLocal.AddDate(1, 0, 0).UTC()

    default: // ChargeTypeRecurringMonthly, ChargeTypeMetered, ChargeTypeOneTime
        yearDiff := atLocal.Year() - anchorLocal.Year()
        monthDiff := int(atLocal.Month()) - int(anchorLocal.Month())
        totalMonths := yearDiff*12 + monthDiff

        startLocal := anchorLocal.AddDate(0, totalMonths, 0)
        if startLocal.After(atLocal) {
            startLocal = anchorLocal.AddDate(0, totalMonths-1, 0)
        }
        return startLocal.UTC(), startLocal.AddDate(0, 1, 0).UTC()
    }
}
```

### Complexity

| Dimension | Complexity |
| :--- | :--- |
| Time | O(1) — arithmetic only, no iteration |
| Space | O(1) — no allocations beyond two `time.Time` values |
| External calls | None — pure function |

---

## 8. HTTP API Specification

### `POST /v1/billing/subscriptions`

Register a new service subscription for a tenant.

**Request:**
```json
{
  "tenant_key": "demo-key-pro",
  "service_id": "object-storage",
  "plan_id": "storage-100gb",
  "charge_type": "recurring_monthly",
  "timezone": "Asia/Tokyo",
  "anchor_time": "2026-08-15T14:30:00Z"
}
```

**Response `201 Created`:**
```json
{
  "subscription_id": "sub_object-storage_demo-key-pro",
  "status": "registered"
}
```

---

### `POST /v1/billing/events`

Ingest a billable metering event directly. Used by external systems or gateways.

**Request:**
```json
{
  "event_id": "evt_abc123",
  "tenant_key": "demo-key-pro",
  "service_id": "service1",
  "metric_id": "api_calls",
  "unit": "requests",
  "quantity": 1,
  "timestamp": "2026-08-04T12:15:00Z"
}
```

**Response `202 Accepted`:**
```json
{
  "status": "accepted",
  "event_id": "evt_abc123"
}
```

---

### `GET /v1/billing/{key}/statement/{service_id}`

Returns the statement for a single service subscription. The cycle window is resolved at the time of the request.

**Response `200 OK`:**
```json
{
  "subscription_id": "sub_ai_123",
  "tenant_key": "demo-key-pro",
  "service_id": "ai-completion",
  "plan_id": "pro-plan",
  "charge_type": "metered",
  "timezone": "America/New_York",
  "cycle_start_utc": "2026-08-01T10:00:00Z",
  "cycle_end_utc": "2026-09-01T10:00:00Z",
  "metrics": {
    "total_tokens": {
      "metric_id": "total_tokens",
      "unit": "tokens",
      "cycle_total": 4500,
      "total_events": 12
    }
  },
  "generated_at": "2026-08-04T12:00:00Z"
}
```

---

### `GET /v1/billing/{key}/overview`

Returns all per-service statements for the tenant. Each statement has its own independent cycle window.

**Response `200 OK`:**
```json
{
  "tenant_key": "demo-key-pro",
  "subscription_state": "active",
  "statements": [
    { "service_id": "ai-completion", "cycle_start_utc": "2026-08-01T10:00:00Z", "..." : "..." },
    { "service_id": "object-storage", "cycle_start_utc": "2026-08-15T14:30:00Z", "..." : "..." }
  ],
  "generated_at": "2026-08-04T12:00:00Z"
}
```

> **Note:** Statements are **not sorted** — consumers should not assume ordering. Sort by `cycle_start_utc` or `service_id` as needed.

---

## 9. Integration Contracts

### 9.1 EventBus Integration

```
Topic:    "metering.event"
Payload:  *kernel.MeteringEvent
Contract: All fields except Metadata are required. Quantity must be > 0. Timestamp must be UTC.
```

Internal service modules publish events directly:

```go
app.Events.Publish("metering.event", &kernel.MeteringEvent{
    EventID:   uuid.New().String(),
    TenantKey: apiKey,
    ServiceID: "ai-completion",
    MetricID:  "total_tokens",
    Unit:      "tokens",
    Quantity:  int64(tokens),
    Timestamp: time.Now().UTC(),
})
```

### 9.2 Store Interface

Any service or test that interacts with metering data uses these methods on `kernel.Store`:

```go
// Write path
RegisterServiceSubscription(sub ServiceSubscription)
RecordMeteringEvent(event MeteringEvent)

// Read path
GetServiceSubscriptions(tenantKey string) []ServiceSubscription
GetServiceBillingStatement(tenantKey, serviceID string, targetTime time.Time) (ServiceBillingStatement, bool)
GetTenantBillingOverview(tenantKey string, targetTime time.Time) TenantBillingOverview
```

---

## 10. Testing & Mocking

### 10.1 Interfaces for Decoupled Testing

```go
// MeteringStoreReader — for read-only query consumers
type MeteringStoreReader interface {
    GetServiceSubscriptions(tenantKey string) []ServiceSubscription
    GetServiceBillingStatement(tenantKey, serviceID string, targetTime time.Time) (ServiceBillingStatement, bool)
    GetTenantBillingOverview(tenantKey string, targetTime time.Time) TenantBillingOverview
}

// MeteringEventRecorder — for write-side integration
type MeteringEventRecorder interface {
    RegisterServiceSubscription(sub ServiceSubscription)
    RecordMeteringEvent(event MeteringEvent)
}
```

### 10.2 In-Memory Mock

```go
type MockMeteringStore struct {
    Subscriptions []ServiceSubscription
    Events        []MeteringEvent
}

func (m *MockMeteringStore) RegisterServiceSubscription(sub ServiceSubscription) {
    m.Subscriptions = append(m.Subscriptions, sub)
}

func (m *MockMeteringStore) RecordMeteringEvent(event MeteringEvent) {
    m.Events = append(m.Events, event)
}
```

### 10.3 Testing the Cycle Window in Isolation

Because `CurrentCycleWindow` is a pure method, it requires no mocks:

```go
sub := ServiceSubscription{
    AnchorTime: time.Date(2026, 8, 15, 14, 30, 0, 0, time.UTC),
    Timezone:   "America/New_York",
    ChargeType: ChargeTypeRecurringMonthly,
}

start, end := sub.CurrentCycleWindow(time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC))
// start == 2026-08-15T14:30:00Z
// end   == 2026-09-15T14:30:00Z
```

---

## 11. Limitations & Known Trade-offs

> These are deliberate trade-offs made to keep the current implementation simple, correct, and evolvable — not oversights.

| Limitation | Impact | Planned Resolution |
| :--- | :--- | :--- |
| **In-memory event store** | Events are lost on process restart. No persistence across restarts. | Swap `kernel.Store` for a Postgres/ClickHouse adapter behind the same interface — zero module-level changes required. |
| **Linear event scan** | `GetServiceBillingStatement` scans all tenant events O(n). For tenants with millions of events this becomes expensive. | Introduce time-indexed B-tree or range index over `Timestamp` per tenant. The storage interface is stable; only the implementation changes. |
| **No deduplication** | Duplicate `EventID` values are currently stored. | Add an `eventSeen map[string]bool` guard in `RecordMeteringEvent`. Interface is stable. |
| **No event compaction** | Old events from past cycles are never pruned. | Add a background compaction job that replaces raw events with pre-aggregated cycle summaries. |
| **One subscription per service per tenant** | The store uses `TenantKey → ServiceID` as the key, so a tenant can't have two overlapping plans for the same `ServiceID`. | Allow `SubscriptionID` as the primary key; `ServiceID` becomes a secondary index. |
| **No cost calculation** | This module produces usage quantities only — not prices or invoice totals. | A separate `pricing` module consumes `ServiceBillingStatement` data and applies rate cards. |

---

## 12. Future Roadmap

This engine is designed as the stable **metering foundation** for a layered billing stack. The planned modules that build on top of it are:

```
┌─────────────────────────────────────────────────────────┐
│                    SUBSCRIPTION MODULE                   │
│   Manages plan lifecycle, upgrades, downgrades, trials  │
│   Publishes ServiceSubscription into the metering engine│
└───────────────────────────┬─────────────────────────────┘
                            │ registers ServiceSubscription
                            ▼
┌─────────────────────────────────────────────────────────┐
│              METERING ENGINE  (this module)              │
│   Records MeteringEvent, resolves cycle windows,        │
│   produces ServiceBillingStatement                       │
└───────────────────────────┬─────────────────────────────┘
                            │ produces ServiceBillingStatement
                            ▼
┌─────────────────────────────────────────────────────────┐
│                     PRICING MODULE                       │
│   Applies rate cards to metric totals                   │
│   Produces line items with currency amounts             │
└───────────────────────────┬─────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                    INVOICING MODULE                      │
│   Combines line items into a tenant invoice             │
│   Supports PDF generation, payment gateway integration  │
└─────────────────────────────────────────────────────────┘
```

Each layer is independently deployable, testable, and replaceable. The metering engine's stable interface means it never needs to change as pricing models, payment methods, or invoice formats evolve.
