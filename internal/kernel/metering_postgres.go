package kernel

import (
	"time"
)

// ---------------------------------------------------------------------------
// PostgresMeteringStore — persistent backend stub (L3 in cache hierarchy).
// ---------------------------------------------------------------------------

// PostgresMeteringStore is the PostgreSQL implementation of MeteringStore.
// This is a structured stub: all methods have correct signatures and return
// zero-values. The actual database driver integration (pgx or database/sql)
// is deferred to a future phase.
//
// When implemented, this backend provides:
//   - Durable storage across process restarts
//   - Full SQL query capability for complex billing reports
//   - Time-range indexing on MeteringEvent.Timestamp for O(log n) queries
//   - ACID guarantees for subscription registration
//
// The MeteringChain writes to this backend asynchronously via a bounded
// channel + BatchWriter, so Postgres latency never blocks the hot path.
type PostgresMeteringStore struct {
	dsn     string
	healthy bool
	// db *pgxpool.Pool  // TODO: integrate with pgx/v5
}

// NewPostgresMeteringStore creates a new PostgreSQL backend.
// In the stub phase, it marks itself as unhealthy (no real connection).
// When pgx is integrated, this will establish the connection pool.
func NewPostgresMeteringStore(dsn string) *PostgresMeteringStore {
	return &PostgresMeteringStore{
		dsn:     dsn,
		healthy: false, // stub is not connected — circuit breaker will keep it out of reads
	}
}

// --- MeteringStore identity ---

func (p *PostgresMeteringStore) Name() string  { return "postgres" }
func (p *PostgresMeteringStore) Priority() int { return 20 }
func (p *PostgresMeteringStore) Healthy() bool { return p.healthy }

// --- Write path ---

// RegisterServiceSubscription persists a subscription to PostgreSQL.
// TODO: Implement with pgx — INSERT ... ON CONFLICT UPDATE.
func (p *PostgresMeteringStore) RegisterServiceSubscription(sub ServiceSubscription) {
	// TODO: implement with pgx
	// SQL: INSERT INTO service_subscriptions (subscription_id, tenant_key, service_id, ...)
	//      VALUES ($1, $2, $3, ...)
	//      ON CONFLICT (tenant_key, service_id) DO UPDATE SET ...
}

// RecordMeteringEvent persists a metering event to PostgreSQL.
// TODO: Implement with pgx — INSERT into metering_events table.
// In production, this will be called via the BatchWriter (not directly)
// to batch multiple events into a single INSERT statement.
func (p *PostgresMeteringStore) RecordMeteringEvent(event MeteringEvent) {
	// TODO: implement with pgx
	// SQL: INSERT INTO metering_events (event_id, tenant_key, service_id, metric_id, ...)
	//      VALUES ($1, $2, $3, $4, ...)
	//      ON CONFLICT (event_id) DO NOTHING  -- idempotent via EventID
}

// --- Read path ---

// GetServiceSubscriptions queries all subscriptions for a tenant.
// TODO: Implement with pgx — SELECT * FROM service_subscriptions WHERE tenant_key = $1.
func (p *PostgresMeteringStore) GetServiceSubscriptions(tenantKey string) []ServiceSubscription {
	// TODO: implement with pgx
	return nil
}

// GetServiceBillingStatement computes a billing statement from persisted events.
// TODO: Implement with pgx — aggregate query with time-range filter.
func (p *PostgresMeteringStore) GetServiceBillingStatement(tenantKey, serviceID string, targetTime time.Time) (ServiceBillingStatement, bool) {
	// TODO: implement with pgx
	// SQL: SELECT metric_id, unit, SUM(quantity), COUNT(*)
	//      FROM metering_events
	//      WHERE tenant_key = $1 AND service_id = $2
	//        AND timestamp >= $3 AND timestamp < $4
	//      GROUP BY metric_id, unit
	return ServiceBillingStatement{}, false
}

// GetTenantBillingOverview returns all service statements for a tenant.
// TODO: Implement with pgx — join subscriptions + aggregated events.
func (p *PostgresMeteringStore) GetTenantBillingOverview(tenantKey string, targetTime time.Time) TenantBillingOverview {
	// TODO: implement with pgx
	return TenantBillingOverview{
		TenantKey:         tenantKey,
		SubscriptionState: "unknown",
		Statements:        make([]ServiceBillingStatement, 0),
		GeneratedAt:       time.Now().UTC(),
	}
}

// ---------------------------------------------------------------------------
// Schema reference (for future migration):
//
// CREATE TABLE service_subscriptions (
//     subscription_id  TEXT PRIMARY KEY,
//     tenant_key       TEXT NOT NULL,
//     service_id       TEXT NOT NULL,
//     plan_id          TEXT NOT NULL,
//     charge_type      TEXT NOT NULL,
//     timezone         TEXT NOT NULL DEFAULT 'UTC',
//     anchor_time      TIMESTAMPTZ NOT NULL,
//     status           TEXT NOT NULL DEFAULT 'active',
//     created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
//     UNIQUE (tenant_key, service_id)
// );
//
// CREATE TABLE metering_events (
//     event_id    TEXT PRIMARY KEY,
//     tenant_key  TEXT NOT NULL,
//     service_id  TEXT NOT NULL,
//     metric_id   TEXT NOT NULL,
//     unit        TEXT NOT NULL,
//     quantity    BIGINT NOT NULL,
//     timestamp   TIMESTAMPTZ NOT NULL,
//     metadata    JSONB,
//     created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
// );
//
// CREATE INDEX idx_metering_events_tenant_time
//     ON metering_events (tenant_key, service_id, timestamp);
// ---------------------------------------------------------------------------
