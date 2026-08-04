# Technical Design Document: Abstract SaaS Metering & Billing Engine

## 1. Executive Summary & Design Scope

The **Abstract SaaS Metering & Billing Engine** located in [`internal/modules/billing`](file:///d:/AK-FILES/Golang%20April/ai-saas-lab/ai-saas-lab/internal/modules/billing) provides a high-throughput, multi-tenant, service-anchor-aware metering and billing infrastructure for SaaS platforms.

### Core Philosophy
1. **Domain Agnosticism:** The engine has zero compile-time coupling to specific application domains (e.g., AI tokens, storage bytes, API request counts, user seats).
2. **Service-Specific Subscription Anchors:** Each service subscription maintains its own **Anchor Time**, **Timezone**, and **Charge Type**. Payment windows and billing cycles are calculated independently per service subscription.
3. **UTC Ingestion with Localized Window Resolution:** Metering event timestamps are ingested and persisted in UTC Epoch time to prevent DST/timezone bugs. Cycle windows `[CycleStartUTC, CycleEndUTC)` are computed dynamically per service subscription.
4. **Decoupled Integration:** Services push events via EventBus (`"metering.event"`) or HTTP endpoints without direct compile-time coupling.

---

## 2. Component Architecture & Data Flow

```
+---------------------------------------------------------------------------------------------------------+
|                                    SERVICE PRODUCERS (PRODUCER LAYER)                                   |
|   +-------------------+     +------------------+     +-------------------+     +------------------+   |
|   |  AI Completion    |     |  Object Storage  |     |   Service 1 / 2   |     | External Gateway |   |
|   +-------------------+     +------------------+     +-------------------+     +------------------+   |
+---------------------------------------------------------------------------------------------------------+
          |                            |                         |                        |
          | Publish                    | Publish                 | Publish                | POST
          v                            v                         v                        v
+---------------------------------------------------------------------------------------------------------+
|                        EVENT BUS ("metering.event")  / REST API ("POST /v1/billing/events")           |
+---------------------------------------------------------------------------------------------------------+
                                                     |
                                                     v
+---------------------------------------------------------------------------------------------------------+
|                                        BILLING MODULE FACADE                                            |
|                                                                                                         |
|   1. ServiceSubscription Registry (TenantKey -> ServiceID -> ServiceSubscription)                      |
|   2. Service Cycle Window Algorithm: CurrentCycleWindow(at) -> [CycleStartUTC, CycleEndUTC)             |
|   3. Event Aggregation Engine (Filters events per service within exact cycle bounds)                     |
|   4. Subscription State Machine (FSM: trial -> active -> past_due -> cancelled)                        |
+---------------------------------------------------------------------------------------------------------+
                                                     |
                                                     v
+---------------------------------------------------------------------------------------------------------+
|                                     MULTI-TENANT KERNEL STORE                                           |
|   - Thread-safe ServiceSubscriptions index                                                              |
|   - Thread-safe MeteringEvents time-series store                                                        |
+---------------------------------------------------------------------------------------------------------+
                                                     |
                                                     v
+---------------------------------------------------------------------------------------------------------+
|                                      HTTP REST QUERY ENDPOINTS                                          |
|   - GET /v1/billing/{key}/overview             -> Overview containing array of per-service statements   |
|   - GET /v1/billing/{key}/statement/{service}  -> Single ServiceBillingStatement for target service      |
+---------------------------------------------------------------------------------------------------------+
```

---

## 3. Data Structures & Struct Contracts

### 3.1 `ChargeType`
Defines the pricing and recurring billing model for a service subscription:

```go
type ChargeType string

const (
    ChargeTypeMetered           ChargeType = "metered"           // Consumption-based (pay-as-you-go)
    ChargeTypeRecurringMonthly  ChargeType = "recurring_monthly" // Fixed monthly subscription
    ChargeTypeRecurringYearly   ChargeType = "recurring_yearly"  // Fixed yearly subscription
    ChargeTypeOneTime           ChargeType = "one_time"          // One-time addon or setup fee
)
```

---

### 3.2 `ServiceSubscription`
Represents a tenant's subscription contract for a specific service:

```go
type ServiceSubscription struct {
    SubscriptionID string     `json:"subscription_id"` // Unique subscription identifier
    TenantKey      string     `json:"tenant_key"`      // Customer/API key identifier
    ServiceID      string     `json:"service_id"`      // e.g. "ai-completion", "storage", "service1"
    PlanID         string     `json:"plan_id"`         // e.g. "pro-plan", "starter-plan"
    ChargeType     ChargeType `json:"charge_type"`     // metered, recurring_monthly, recurring_yearly
    Timezone       string     `json:"timezone"`        // IANA Timezone, e.g. "America/New_York", "UTC"
    AnchorTime     time.Time  `json:"anchor_time"`     // Timestamp when service subscription started
    Status         string     `json:"status"`          // "active", "cancelled", "past_due"
}
```

---

### 3.3 `MeteringEvent`
Represents an atomic billable event generated by any service:

```go
type MeteringEvent struct {
    EventID     string                 `json:"event_id"`   // Idempotency / event identifier
    TenantKey   string                 `json:"tenant_key"`  // Customer/API key identifier
    ServiceID   string                 `json:"service_id"`  // Service originating the event
    MetricID    string                 `json:"metric_id"`   // Metric name (e.g. "tokens", "requests")
    Unit        string                 `json:"unit"`        // Unit of measurement (e.g. "count", "bytes")
    Quantity    int64                  `json:"quantity"`    // Quantity consumed
    Timestamp   time.Time              `json:"timestamp"`   // UTC timestamp of event occurrence
    Metadata    map[string]interface{} `json:"metadata,omitempty"` // Contextual tags (e.g. model, region)
}
```

---

### 3.4 `MetricSummary`
Holds aggregated consumption totals for a single metric within a service cycle:

```go
type MetricSummary struct {
    MetricID    string `json:"metric_id"`    // Metric identifier
    Unit        string `json:"unit"`         // Unit of measurement
    CycleTotal  int64  `json:"cycle_total"`  // Total quantity consumed within cycle window
    TotalEvents int64  `json:"total_events"` // Number of metering events recorded
}
```

---

### 3.5 `ServiceBillingStatement`
Represents the statement for a single service subscription:

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

### 3.6 `TenantBillingOverview`
Consolidates all service billing statements for a tenant:

```go
type TenantBillingOverview struct {
    TenantKey         string                    `json:"tenant_key"`
    SubscriptionState string                    `json:"subscription_state"`
    Statements        []ServiceBillingStatement `json:"statements"`
    GeneratedAt       time.Time                 `json:"generated_at"`
}
```

---

## 4. Algorithmic Specification: Service Cycle Window Calculation

Given a `ServiceSubscription` $S$ and target timestamp $t$, the function `CurrentCycleWindow(at time.Time)` determines the UTC window $[T_{\text{start}}, T_{\text{end}})$:

```
                  AnchorTime (e.g. Aug 15 14:30 UTC)
                              |
    Cycle 1:  [Aug 15 14:30 UTC ------------> Sep 15 14:30 UTC)
    Cycle 2:                              [Sep 15 14:30 UTC ------------> Oct 15 14:30 UTC)
                                                         ^
                                                  Target Time (t)
```

### Algorithm Implementation Contract:

```go
func (s ServiceSubscription) CurrentCycleWindow(at time.Time) (startUTC, endUTC time.Time) {
    if at.Before(s.AnchorTime) {
        return s.AnchorTime, s.AnchorTime
    }

    loc, err := time.LoadLocation(s.Timezone)
    if err != nil {
        loc = time.UTC
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
        endLocal := startLocal.AddDate(1, 0, 0)
        return startLocal.UTC(), endLocal.UTC()

    case ChargeTypeRecurringMonthly, ChargeTypeMetered:
        fallthrough
    default:
        yearDiff := atLocal.Year() - anchorLocal.Year()
        monthDiff := int(atLocal.Month()) - int(anchorLocal.Month())
        totalMonths := yearDiff*12 + monthDiff

        startLocal := anchorLocal.AddDate(0, totalMonths, 0)
        if startLocal.After(atLocal) {
            startLocal = anchorLocal.AddDate(0, totalMonths-1, 0)
        }
        endLocal := startLocal.AddDate(0, 1, 0)
        return startLocal.UTC(), endLocal.UTC()
    }
}
```

---

## 5. Integration Contracts & HTTP API Specifications

### 5.1 EventBus Contract
- **Topic:** `"metering.event"`
- **Payload Type:** `*billing.MeteringEvent`
- **Behavior:** The billing module subscribes to `"metering.event"` on startup and asynchronously records all valid payloads in `kernel.Store`.

---

### 5.2 REST API Contracts

#### `GET /v1/billing/{key}/overview`
Returns the consolidated billing overview containing all active service statements.

**Response `200 OK`:**
```json
{
  "tenant_key": "demo-key-pro",
  "subscription_state": "active",
  "statements": [
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
  ],
  "generated_at": "2026-08-04T12:00:00Z"
}
```

---

#### `GET /v1/billing/{key}/statement/{service_id}`
Returns the statement for a specific service subscription.

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

#### `POST /v1/billing/subscriptions`
Registers a new service subscription for a tenant.

**Request Body:**
```json
{
  "tenant_key": "demo-key-pro",
  "service_id": "object-storage",
  "plan_id": "storage-pro",
  "charge_type": "recurring_monthly",
  "timezone": "Asia/Tokyo",
  "anchor_time": "2026-08-04T12:00:00Z"
}
```

**Response `201 Created`:**
```json
{
  "subscription_id": "sub_storage_999",
  "status": "active"
}
```

---

#### `POST /v1/billing/events`
Direct HTTP endpoint for external gateways to ingest metering events.

**Request Body:**
```json
{
  "tenant_key": "demo-key-pro",
  "service_id": "service1",
  "metric_id": "api_calls",
  "unit": "requests",
  "quantity": 1
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

## 6. Testing & Mocking Contracts

### 6.1 `MeteringStoreReader` Interface (For Decoupled Querying)

```go
type MeteringStoreReader interface {
    GetServiceSubscriptions(tenantKey string) []ServiceSubscription
    GetServiceBillingStatement(tenantKey, serviceID string, targetTime time.Time) (ServiceBillingStatement, bool)
    GetTenantBillingOverview(tenantKey string, targetTime time.Time) TenantBillingOverview
}
```

### 6.2 `MeteringEventRecorder` Interface (For Mock Ingestion in Tests)

```go
type MeteringEventRecorder interface {
    RegisterServiceSubscription(sub ServiceSubscription)
    RecordMeteringEvent(event MeteringEvent)
}
```

### 6.3 Mock Implementation for Unit Tests

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
