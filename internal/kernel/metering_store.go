package kernel

import (
	"context"
	"time"
)


// ---------------------------------------------------------------------------
// MeteringStore — the contract every storage backend implements.
// ---------------------------------------------------------------------------

// MeteringStore defines the complete read/write contract for metering data
// storage. Any backend (memory, PostgreSQL, Redis, ClickHouse, DynamoDB)
// implements this interface. The MeteringChain orchestrator fans writes out
// to all registered backends and cascades reads by priority.
type MeteringStore interface {
	MeteringStoreReader
	MeteringEventRecorder

	// Name returns the human-readable backend identifier (e.g. "memory",
	// "postgres", "redis"). Used for logging, health reports, and WAL
	// targeting.
	Name() string

	// Priority determines read cascade order. Lower values are tried first.
	// Convention: 0 = in-memory (fastest), 10 = cache (Redis), 20 = persistent
	// (PostgreSQL), 30+ = archive (S3, ClickHouse).
	Priority() int

	// Healthy reports whether this backend is currently reachable and able
	// to serve reads/writes. Set by the background health-check goroutine.
	Healthy() bool

	// Ping verifies backend connectivity for health-check goroutines.
	Ping(ctx context.Context) error
}

// ---------------------------------------------------------------------------
// Narrow sub-interfaces for consumer/producer decoupling.
// ---------------------------------------------------------------------------

// MeteringStoreReader is the read-only view of metering data. Use this
// interface for reporting dashboards, pricing modules, and any consumer
// that should not be able to write events.
type MeteringStoreReader interface {
	GetServiceSubscriptions(tenantKey string) []ServiceSubscription
	GetServiceBillingStatement(tenantKey, serviceID string, targetTime time.Time) (ServiceBillingStatement, bool)
	GetTenantBillingOverview(tenantKey string, targetTime time.Time) TenantBillingOverview
}

// MeteringEventRecorder is the write-only view for event producers.
//
// Both write methods return error so the chain's async workers and circuit
// breakers receive real failure signals instead of polling Healthy().
type MeteringEventRecorder interface {
	// RegisterServiceSubscription persists or upserts a subscription contract.
	// Returns an error if the backend is unreachable or the write fails.
	RegisterServiceSubscription(sub ServiceSubscription) error

	// RecordMeteringEvent appends a billable event. Returns an error if the
	// backend is unreachable or the write fails. The chain will route failures
	// to the WAL and trip the circuit breaker.
	RecordMeteringEvent(event MeteringEvent) error
}
