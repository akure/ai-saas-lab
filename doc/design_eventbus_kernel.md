# Event Bus Design Guide

## Purpose

The event bus provides a simple, decoupled communication channel for kernel modules. It allows publishers and subscribers to interact without importing each other directly, which keeps modules loosely coupled and easier to evolve.

## Callback-Based Design in Go

This design is a classic Go callback pattern:

- A publisher emits an event on a topic.
- A subscriber registers a callback function with the bus.
- When the event is published, the bus invokes each registered callback.

In Go, a callback is simply a function value. The signature used here is:

```go
func(any)
```

That means every subscriber accepts a single interface payload, which makes the bus flexible enough to carry different event shapes while remaining simple.

### Why this works well in Go

1. No direct imports between modules
   - The completion module can publish events without importing the billing module.
2. Function values are first-class citizens
   - A callback can be passed around like any other value.
3. The bus stays small and composable
   - Registration and publication are easy to reason about.

## In-Memory Callback Model

The bus stores callbacks in memory inside a map of topic names to functions. When a publisher calls `Publish`, the bus copies the current subscriber list and then runs each callback asynchronously.

This is an in-memory callback mechanism, not a durable message queue. That means:

- callbacks live only while the process is running;
- events are lost if the process crashes before delivery;
- delivery is best-effort and asynchronous.

### Mental model

Think of the event bus as a lightweight notification board:

- subscribe = pin a callback to a topic;
- publish = notify everyone subscribed to that topic;
- callback = the receiver logic that reacts to the event.

## Production-Grade Usage Example

In a real AI SaaS deployment, the bus is useful for cross-module coordination without making modules depend on each other directly. The examples below reflect the kind of event flow we would expect once the lab evolves beyond the current proof-of-concept stage.

### 1. Completion module publishes usage for billing and cost tracking

Today the current completion module already has a simple usage shape in the form of a usage record with fields such as API key, token count, and model. In a future production extension, that same event could be expanded into a richer payload for billing, cost analytics, and auditing.

```go
type UsageRecord struct {
    APIKey string
    Tokens int
    Model  string
}

bus.Publish("usage.recorded", UsageRecord{
    APIKey: "sk-demo",
    Tokens: 120,
    Model:  "mock-mini-1",
})
```

### 2. Billing module consumes the event to meter usage or trigger charge logic

The billing module is the natural subscriber for usage events. In a more mature system, it could use the payload to update metering, calculate spend, or enqueue an invoice workflow.

```go
bus.Subscribe("usage.recorded", func(payload any) {
    rec, ok := payload.(UsageRecord)
    if !ok {
        log.Printf("billing: invalid payload type %T", payload)
        return
    }

    log.Printf("billing: metering usage for model %s with %d tokens", rec.Model, rec.Tokens)
    // future: update ledger, calculate cost, or trigger invoice logic
})
```

### 3. Safety module reacts to risky requests or policy violations

Another practical example is a safety or moderation module. Once expanded, it could publish a structured event that other modules react to, such as logging, alerting, or rate limiting.

```go
type SafetyEvent struct {
    APIKey string
    Reason string
    Severity string
}

bus.Publish("safety.flagged", SafetyEvent{
    APIKey:   "sk-demo",
    Reason:   "prompt_injection_detected",
    Severity: "high",
})
```

### 4. Admin or observability layer listens for operational events

An admin service, monitoring service, or audit pipeline could subscribe to operational events such as `quota.exceeded`, `model.failure`, or `tenant.blocked` to produce dashboards, alerts, and audit logs.

```go
bus.Subscribe("quota.exceeded", func(payload any) {
    log.Printf("alert: operational event received: %T", payload)
})
```

## Who Publishes What in a Real SaaS System

This is the most important practical distinction:

- The completion module publishes request lifecycle and usage events.
- The safety module publishes policy and moderation events.
- The auth or tenant module publishes sign-in, entitlement, or subscription events.
- The billing module typically subscribes, not publishes, unless it also emits invoice or payment events.
- The admin/observability layer subscribes to operational events for alerts and metrics.

In other words, the bus is most useful when one part of the system produces an event and several other parts need to react to it without being tightly coupled.

## Why This Is Practical

This pattern is valuable when you want to avoid hard dependencies such as:

- completion importing billing logic,
- safety importing admin logic,
- auth importing cost tracking logic.

Instead, each module emits a business event and other modules decide whether they need to react.

## Client Usage Example

A client module or service can use the event bus by importing the kernel package and then subscribing or publishing through the shared application object.

```go
package billing

import (
    "log"

    "aisaaslab/internal/kernel"
)

func Register(bus *kernel.EventBus) {
    bus.Subscribe("usage.recorded", func(payload any) {
        rec, ok := payload.(UsageRecord)
        if !ok {
            log.Printf("unexpected payload type: %T", payload)
            return
        }

        log.Printf("billing module received usage for model %s", rec.Model)
    })
}
```

### Import pattern

When another module wants to use the bus, the import usually looks like this:

```go
import "aisaaslab/internal/kernel"
```

### Who creates the bus?

The shared event bus is created once by the application bootstrap layer in the kernel package. In this codebase, that happens inside the app constructor:

```go
func NewApp(cfg *Config) *App {
    return &App{
        Config:   cfg,
        Events:   NewEventBus(),
        Store:    NewStore(),
        Mux:      http.NewServeMux(),
        encoders: make(map[string]Encoder),
        messages: make(map[string]MessageDescriptor),
        handlers: make(map[string]MessageHandler),
        policies: make(map[string]Policy),
    }
}
```

That means the ownership is centralized: the application creates one shared bus and stores it in the app as `app.Events`.

### How a module gets the bus

A module does not create a new bus itself in the normal flow. Instead, it receives the already-initialized app and uses the shared pointer from there:

```go
app.Events.Subscribe("usage.recorded", func(payload any) {
    // handle the event
})
```

The same pattern is used for publishing:

```go
app.Events.Publish("usage.recorded", UsageRecord{
    APIKey: "sk-demo",
    Tokens: 42,
    Model:  "mock-mini-1",
})
```

This is the important architectural idea: a module does not need to import another module to talk to it. It only needs the shared event bus pointer from the app and a common event topic.

## Important Go-Specific Notes

- The callback signature uses `any` because Go does not have a universal event type.
- Subscribers should type-assert payloads carefully before using them.
- Since the callbacks run in separate goroutines, handlers must be safe for concurrent execution.
- The publisher should not assume the callback has completed when `Publish` returns.

## Recommended Usage for Intermediate Developers

- Use descriptive topics such as `usage.recorded`, `billing.updated`, or `auth.failed`.
- Keep callback handlers short and focused on one responsibility.
- Avoid shared mutable state inside callbacks unless it is synchronized with a mutex or other safe primitive.
- Treat event delivery as asynchronous and non-blocking.
- Prefer small payload structs over raw strings when the event carries multiple fields.

## Testing Expectations

The event bus should be tested for:

- fan-out to every subscriber on the matching topic;
- correct payload delivery;
- no delivery for unrelated topics;
- asynchronous execution that does not block publication.

## Future Enhancements

Potential improvements include:

- retry policies for failed subscribers;
- dead-letter handling for unprocessed events;
- context-aware subscriptions;
- optional ordered delivery semantics;
- persistent storage for durable event delivery.
