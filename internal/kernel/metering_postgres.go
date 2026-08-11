package kernel

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// PostgresMeteringStore — durable L3 backend backed by PostgreSQL (pgx/v5).
// ---------------------------------------------------------------------------

// postgresHealthCheckInterval is how often the background probe pings Postgres.
const postgresHealthCheckInterval = 15 * time.Second

// PostgresMeteringStore is the PostgreSQL implementation of MeteringStore.
//
// Write path:
//   - RegisterServiceSubscription: INSERT … ON CONFLICT DO UPDATE (upsert)
//   - RecordMeteringEvent: INSERT … ON CONFLICT (event_id) DO NOTHING (idempotent)
//
// Read path:
//   - GetServiceSubscriptions: SELECT * FROM service_subscriptions WHERE tenant_key = $1
//   - GetServiceBillingStatement: aggregate query with time-range filter
//   - GetTenantBillingOverview: join across all service subscriptions
//
// Health:
//   - A background goroutine pings every 15s. Healthy() reflects last probe result.
//   - Circuit breaker in MeteringChain provides additional isolation.
type PostgresMeteringStore struct {
	pool    *pgxpool.Pool
	dsn     string
	healthy atomic.Bool
	stopCh  chan struct{}
	once    sync.Once // ensures Stop() is idempotent
}

// NewPostgresMeteringStore creates and connects a PostgreSQL backend.
// Returns an error if the initial connection pool cannot be established.
// The health-check goroutine is started automatically; call Stop() on shutdown.
func NewPostgresMeteringStore(ctx context.Context, dsn string) (*PostgresMeteringStore, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse DSN: %w", err)
	}
	// Sensible pool defaults for metering workloads.
	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: create pool: %w", err)
	}

	// Verify connectivity immediately.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: initial ping failed: %w", err)
	}

	s := &PostgresMeteringStore{
		pool:   pool,
		dsn:    dsn,
		stopCh: make(chan struct{}),
	}
	s.healthy.Store(true)
	go s.healthProbe()
	return s, nil
}

// --- MeteringStore identity ---

func (p *PostgresMeteringStore) Name() string  { return "postgres" }
func (p *PostgresMeteringStore) Priority() int { return 20 }
func (p *PostgresMeteringStore) Healthy() bool { return p.healthy.Load() }

// Ping verifies connectivity. Called by the health-check goroutine and
// by the MeteringChain during WAL replay probing.
func (p *PostgresMeteringStore) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// Stop terminates the health-check goroutine and closes the connection pool.
// Safe to call multiple times.
func (p *PostgresMeteringStore) Stop() {
	p.once.Do(func() {
		close(p.stopCh)
		p.pool.Close()
	})
}

// --- Write path ---

// RegisterServiceSubscription upserts a service subscription.
// Uses ON CONFLICT (tenant_key, service_id) DO UPDATE to be idempotent.
func (p *PostgresMeteringStore) RegisterServiceSubscription(sub ServiceSubscription) error {
	if !p.healthy.Load() {
		return fmt.Errorf("postgres: backend unhealthy")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := p.pool.Exec(ctx, `
		INSERT INTO service_subscriptions
			(subscription_id, tenant_key, service_id, plan_id, charge_type,
			 timezone, anchor_time, status, created_at)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (tenant_key, service_id)
		DO UPDATE SET
			subscription_id = EXCLUDED.subscription_id,
			plan_id         = EXCLUDED.plan_id,
			charge_type     = EXCLUDED.charge_type,
			timezone        = EXCLUDED.timezone,
			anchor_time     = EXCLUDED.anchor_time,
			status          = EXCLUDED.status`,
		sub.SubscriptionID, sub.TenantKey, sub.ServiceID,
		sub.PlanID, string(sub.ChargeType),
		sub.Timezone, sub.AnchorTime, sub.Status,
	)
	if err != nil {
		if isConnectionError(err) {
			p.healthy.Store(false)
		}
		return fmt.Errorf("postgres: upsert subscription: %w", err)
	}
	return nil
}

// RecordMeteringEvent inserts a metering event, ignoring duplicates by event_id.
// Uses ON CONFLICT (event_id) DO NOTHING for idempotent at-least-once delivery.
func (p *PostgresMeteringStore) RecordMeteringEvent(event MeteringEvent) error {
	if !p.healthy.Load() {
		return fmt.Errorf("postgres: backend unhealthy")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := p.pool.Exec(ctx, `
		INSERT INTO metering_events
			(event_id, tenant_key, service_id, metric_id, unit,
			 quantity, timestamp, created_at)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (event_id) DO NOTHING`,
		event.EventID, event.TenantKey, event.ServiceID,
		event.MetricID, event.Unit, event.Quantity, event.Timestamp,
	)
	if err != nil {
		if isConnectionError(err) {
			p.healthy.Store(false)
		}
		return fmt.Errorf("postgres: insert event: %w", err)
	}
	return nil
}

// isConnectionError reports whether err is a connection-level failure rather than a
// PostgreSQL query-level error (such as a constraint violation or invalid parameter).
// Query-level errors returned by Postgres implement *pgconn.PgError.
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	return !errors.As(err, &pgErr)
}

// --- Read path ---

// GetServiceSubscriptions returns all subscriptions for a tenant.
func (p *PostgresMeteringStore) GetServiceSubscriptions(tenantKey string) []ServiceSubscription {
	if !p.healthy.Load() {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := p.pool.Query(ctx, `
		SELECT subscription_id, tenant_key, service_id, plan_id,
		       charge_type, timezone, anchor_time, status
		FROM service_subscriptions
		WHERE tenant_key = $1`, tenantKey)
	if err != nil {
		return nil
	}
	defer rows.Close()

	subs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (ServiceSubscription, error) {
		var s ServiceSubscription
		var chargeType string
		err := row.Scan(&s.SubscriptionID, &s.TenantKey, &s.ServiceID,
			&s.PlanID, &chargeType, &s.Timezone, &s.AnchorTime, &s.Status)
		s.ChargeType = ChargeType(chargeType)
		return s, err
	})
	if err != nil {
		return nil
	}
	return subs
}

// GetServiceUsageStatement computes a usage statement for a specific service
// by aggregating metering events within the subscription's cycle window.
func (p *PostgresMeteringStore) GetServiceUsageStatement(tenantKey, serviceID string, targetTime time.Time) (ServiceUsageStatement, bool) {
	if !p.healthy.Load() {
		return ServiceUsageStatement{}, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Step 1: Load the subscription to determine cycle window.
	var sub ServiceSubscription
	var chargeType string
	err := p.pool.QueryRow(ctx, `
		SELECT subscription_id, tenant_key, service_id, plan_id,
		       charge_type, timezone, anchor_time, status
		FROM service_subscriptions
		WHERE tenant_key = $1 AND service_id = $2`,
		tenantKey, serviceID,
	).Scan(&sub.SubscriptionID, &sub.TenantKey, &sub.ServiceID,
		&sub.PlanID, &chargeType, &sub.Timezone, &sub.AnchorTime, &sub.Status)
	if err != nil {
		return ServiceUsageStatement{}, false
	}
	sub.ChargeType = ChargeType(chargeType)

	startUTC, endUTC := sub.CurrentCycleWindow(targetTime)
	stmt := ServiceUsageStatement{
		SubscriptionID: sub.SubscriptionID,
		TenantKey:      sub.TenantKey,
		ServiceID:      sub.ServiceID,
		PlanID:         sub.PlanID,
		ChargeType:     sub.ChargeType,
		Timezone:       sub.Timezone,
		CycleStartUTC:  startUTC,
		CycleEndUTC:    endUTC,
		Metrics:        make(map[MetricID]*MetricSummary),
		GeneratedAt:    time.Now().UTC(),
	}

	// Step 2: Aggregate events within the cycle window.
	rows, err := p.pool.Query(ctx, `
		SELECT metric_id, unit, SUM(quantity) AS total, COUNT(*) AS cnt
		FROM metering_events
		WHERE tenant_key = $1
		  AND service_id = $2
		  AND timestamp >= $3
		  AND timestamp <  $4
		GROUP BY metric_id, unit`,
		tenantKey, serviceID, startUTC, endUTC,
	)
	if err != nil {
		return stmt, true // return partial statement
	}
	defer rows.Close()

	for rows.Next() {
		ms := &MetricSummary{}
		if err := rows.Scan(&ms.MetricID, &ms.Unit, &ms.CycleTotal, &ms.TotalEvents); err != nil {
			continue
		}
		stmt.Metrics[ms.MetricID] = ms
	}
	return stmt, true
}

// GetTenantUsageOverview returns all service usage statements for a tenant.
func (p *PostgresMeteringStore) GetTenantUsageOverview(tenantKey string, targetTime time.Time) TenantUsageOverview {
	tk, _ := NewTenantKey(tenantKey)
	overview := TenantUsageOverview{
		TenantKey:   tk,
		Statements:  make([]ServiceUsageStatement, 0),
		GeneratedAt: time.Now().UTC(),
	}

	subs := p.GetServiceSubscriptions(tenantKey)
	for _, sub := range subs {
		stmt, ok := p.GetServiceUsageStatement(tenantKey, sub.ServiceID.String(), targetTime)
		if ok {
			overview.Statements = append(overview.Statements, stmt)
		}
	}
	return overview
}

// --- Background health-check goroutine ---

// healthProbe pings the database every postgresHealthCheckInterval and updates
// the healthy flag. This is what Healthy() reads — no polling on hot paths.
func (p *PostgresMeteringStore) healthProbe() {
	ticker := time.NewTicker(postgresHealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			err := p.pool.Ping(ctx)
			cancel()
			if err != nil {
				p.healthy.Store(false)
			} else {
				p.healthy.Store(true)
			}
		case <-p.stopCh:
			return
		}
	}
}

// ---------------------------------------------------------------------------
// Schema migration reference:
//
// CREATE TABLE IF NOT EXISTS service_subscriptions (
//     subscription_id  TEXT        NOT NULL,
//     tenant_key       TEXT        NOT NULL,
//     service_id       TEXT        NOT NULL,
//     plan_id          TEXT        NOT NULL,
//     charge_type      TEXT        NOT NULL,
//     timezone         TEXT        NOT NULL DEFAULT 'UTC',
//     anchor_time      TIMESTAMPTZ NOT NULL,
//     status           TEXT        NOT NULL DEFAULT 'active',
//     created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
//     CONSTRAINT pk_service_subscriptions PRIMARY KEY (tenant_key, service_id)
// );
//
// CREATE TABLE IF NOT EXISTS metering_events (
//     event_id    TEXT        NOT NULL PRIMARY KEY,
//     tenant_key  TEXT        NOT NULL,
//     service_id  TEXT        NOT NULL,
//     metric_id   TEXT        NOT NULL,
//     unit        TEXT        NOT NULL,
//     quantity    BIGINT      NOT NULL,
//     timestamp   TIMESTAMPTZ NOT NULL,
//     created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
// );
//
// CREATE INDEX IF NOT EXISTS idx_metering_events_tenant_time
//     ON metering_events (tenant_key, service_id, timestamp);
//
// ---------------------------------------------------------------------------
