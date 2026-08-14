package kernel

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresTenantCatalogStore is the PostgreSQL implementation of TenantCatalogStore.
type PostgresTenantCatalogStore struct {
	pool *pgxpool.Pool
	dsn  string
}

func NewPostgresTenantCatalogStore(ctx context.Context, dsn string) (*PostgresTenantCatalogStore, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres catalog: parse DSN: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres catalog: connect: %w", err)
	}

	store := &PostgresTenantCatalogStore{
		pool: pool,
		dsn:  dsn,
	}

	if err := store.autoMigrate(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres catalog: migrate: %w", err)
	}

	return store, nil
}

func (p *PostgresTenantCatalogStore) Name() string { return "postgres" }
func (p *PostgresTenantCatalogStore) Close() error {
	p.pool.Close()
	return nil
}

func (p *PostgresTenantCatalogStore) autoMigrate(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS tenant_services (
			tenant_key  TEXT NOT NULL,
			service_id  TEXT NOT NULL,
			name        TEXT NOT NULL,
			description TEXT,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (tenant_key, service_id)
		);`,
		`CREATE TABLE IF NOT EXISTS tenant_metrics (
			tenant_key  TEXT NOT NULL,
			metric_id   TEXT NOT NULL,
			service_id  TEXT NOT NULL,
			name        TEXT NOT NULL,
			unit        TEXT NOT NULL,
			description TEXT,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (tenant_key, metric_id)
		);`,
		`CREATE TABLE IF NOT EXISTS tenant_plans (
			tenant_key           TEXT    NOT NULL,
			plan_id              TEXT    NOT NULL,
			service_id           TEXT    NOT NULL,
			name                 TEXT    NOT NULL,
			rates_json           JSONB   NOT NULL DEFAULT '{}'::jsonb,
			included_quotas_json JSONB   NOT NULL DEFAULT '{}'::jsonb,
			version              INT     NOT NULL DEFAULT 1,
			active               BOOLEAN NOT NULL DEFAULT true,
			created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (tenant_key, plan_id)
		);`,
		// Idempotent: add updated_at if upgrading from older schema.
		`ALTER TABLE tenant_services ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();`,
		`ALTER TABLE tenant_metrics  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();`,
		`ALTER TABLE tenant_plans    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();`,
	}

	for _, q := range queries {
		if _, err := p.pool.Exec(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

// --- Service Catalog ---

func (p *PostgresTenantCatalogStore) RegisterService(ctx context.Context, tenant TenantKey, svc TenantServiceDescriptor) error {
	query := `
		INSERT INTO tenant_services (tenant_key, service_id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (tenant_key, service_id) DO UPDATE
		SET name = EXCLUDED.name, description = EXCLUDED.description, updated_at = NOW();
	`
	_, err := p.pool.Exec(ctx, query, tenant.String(), svc.ServiceID.String(), svc.Name, svc.Description)
	if err != nil {
		return fmt.Errorf("postgres register service: %w", err)
	}
	return nil
}

func (p *PostgresTenantCatalogStore) GetService(ctx context.Context, tenant TenantKey, id ServiceID) (TenantServiceDescriptor, bool, error) {
	query := `SELECT service_id, name, description, created_at FROM tenant_services WHERE tenant_key = $1 AND service_id = $2`
	var svc TenantServiceDescriptor
	var serviceIDStr string
	var createdAt time.Time

	err := p.pool.QueryRow(ctx, query, tenant.String(), id.String()).Scan(&serviceIDStr, &svc.Name, &svc.Description, &createdAt)
	if err != nil {
		if errorsIsNotFound(err) {
			return TenantServiceDescriptor{}, false, nil
		}
		return TenantServiceDescriptor{}, false, err
	}
	svc.ServiceID = ServiceID(serviceIDStr)
	svc.CreatedAt = createdAt.Format(time.RFC3339)
	return svc, true, nil
}

func (p *PostgresTenantCatalogStore) ListServices(ctx context.Context, tenant TenantKey) ([]TenantServiceDescriptor, error) {
	query := `SELECT service_id, name, description, created_at FROM tenant_services WHERE tenant_key = $1 ORDER BY created_at ASC`
	rows, err := p.pool.Query(ctx, query, tenant.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []TenantServiceDescriptor
	for rows.Next() {
		var svc TenantServiceDescriptor
		var serviceIDStr string
		var createdAt time.Time
		if err := rows.Scan(&serviceIDStr, &svc.Name, &svc.Description, &createdAt); err != nil {
			return nil, err
		}
		svc.ServiceID = ServiceID(serviceIDStr)
		svc.CreatedAt = createdAt.Format(time.RFC3339)
		list = append(list, svc)
	}
	return list, rows.Err()
}

// --- Metric Catalog ---

func (p *PostgresTenantCatalogStore) RegisterMetric(ctx context.Context, tenant TenantKey, metric TenantMetricDescriptor) error {
	query := `
		INSERT INTO tenant_metrics (tenant_key, metric_id, service_id, name, unit, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (tenant_key, metric_id) DO UPDATE
		SET service_id = EXCLUDED.service_id, name = EXCLUDED.name,
		    unit = EXCLUDED.unit, description = EXCLUDED.description, updated_at = NOW();
	`
	_, err := p.pool.Exec(ctx, query,
		tenant.String(), metric.MetricID.String(), metric.ServiceID.String(),
		metric.Name, metric.Unit, metric.Description)
	if err != nil {
		return fmt.Errorf("postgres register metric: %w", err)
	}
	return nil
}

func (p *PostgresTenantCatalogStore) GetMetric(ctx context.Context, tenant TenantKey, id MetricID) (TenantMetricDescriptor, bool, error) {
	query := `SELECT metric_id, service_id, name, unit, description, created_at FROM tenant_metrics WHERE tenant_key = $1 AND metric_id = $2`
	var metric TenantMetricDescriptor
	var metricIDStr, serviceIDStr string
	var createdAt time.Time

	err := p.pool.QueryRow(ctx, query, tenant.String(), id.String()).Scan(&metricIDStr, &serviceIDStr, &metric.Name, &metric.Unit, &metric.Description, &createdAt)
	if err != nil {
		if errorsIsNotFound(err) {
			return TenantMetricDescriptor{}, false, nil
		}
		return TenantMetricDescriptor{}, false, err
	}
	metric.MetricID = MetricID(metricIDStr)
	metric.ServiceID = ServiceID(serviceIDStr)
	metric.CreatedAt = createdAt.Format(time.RFC3339)
	return metric, true, nil
}

func (p *PostgresTenantCatalogStore) ListMetrics(ctx context.Context, tenant TenantKey) ([]TenantMetricDescriptor, error) {
	query := `SELECT metric_id, service_id, name, unit, description, created_at FROM tenant_metrics WHERE tenant_key = $1 ORDER BY created_at ASC`
	rows, err := p.pool.Query(ctx, query, tenant.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []TenantMetricDescriptor
	for rows.Next() {
		var metric TenantMetricDescriptor
		var metricIDStr, serviceIDStr string
		var createdAt time.Time
		if err := rows.Scan(&metricIDStr, &serviceIDStr, &metric.Name, &metric.Unit, &metric.Description, &createdAt); err != nil {
			return nil, err
		}
		metric.MetricID = MetricID(metricIDStr)
		metric.ServiceID = ServiceID(serviceIDStr)
		metric.CreatedAt = createdAt.Format(time.RFC3339)
		list = append(list, metric)
	}
	return list, rows.Err()
}

// --- Plan Catalog ---

func (p *PostgresTenantCatalogStore) RegisterPlan(ctx context.Context, tenant TenantKey, plan TenantPlanDescriptor) error {
	ratesJSON, err := json.Marshal(plan.Rates)
	if err != nil {
		return fmt.Errorf("marshal rates: %w", err)
	}
	quotasJSON, err := json.Marshal(plan.IncludedQuotas)
	if err != nil {
		return fmt.Errorf("marshal quotas: %w", err)
	}

	query := `
		INSERT INTO tenant_plans (tenant_key, plan_id, service_id, name, rates_json, included_quotas_json, version, active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (tenant_key, plan_id) DO UPDATE
		SET service_id = EXCLUDED.service_id, name = EXCLUDED.name, rates_json = EXCLUDED.rates_json,
		    included_quotas_json = EXCLUDED.included_quotas_json, version = EXCLUDED.version, active = EXCLUDED.active;
	`
	version := plan.Version
	if version <= 0 {
		version = 1
	}

	_, err = p.pool.Exec(ctx, query, tenant.String(), plan.PlanID.String(), plan.ServiceID.String(), plan.Name, ratesJSON, quotasJSON, version, plan.Active)
	return err
}

func (p *PostgresTenantCatalogStore) GetPlan(ctx context.Context, tenant TenantKey, id ApplicationPlanID) (TenantPlanDescriptor, bool, error) {
	query := `SELECT plan_id, service_id, name, rates_json, included_quotas_json, version, active, created_at FROM tenant_plans WHERE tenant_key = $1 AND plan_id = $2`
	var plan TenantPlanDescriptor
	var planIDStr, serviceIDStr string
	var ratesJSON, quotasJSON []byte
	var createdAt time.Time

	err := p.pool.QueryRow(ctx, query, tenant.String(), id.String()).Scan(&planIDStr, &serviceIDStr, &plan.Name, &ratesJSON, &quotasJSON, &plan.Version, &plan.Active, &createdAt)
	if err != nil {
		if errorsIsNotFound(err) {
			return TenantPlanDescriptor{}, false, nil
		}
		return TenantPlanDescriptor{}, false, err
	}
	plan.PlanID = ApplicationPlanID(planIDStr)
	plan.ServiceID = ServiceID(serviceIDStr)
	plan.CreatedAt = createdAt.Format(time.RFC3339)
	_ = json.Unmarshal(ratesJSON, &plan.Rates)
	_ = json.Unmarshal(quotasJSON, &plan.IncludedQuotas)
	return plan, true, nil
}

func (p *PostgresTenantCatalogStore) ListPlans(ctx context.Context, tenant TenantKey) ([]TenantPlanDescriptor, error) {
	query := `SELECT plan_id, service_id, name, rates_json, included_quotas_json, version, active, created_at FROM tenant_plans WHERE tenant_key = $1 ORDER BY created_at ASC`
	rows, err := p.pool.Query(ctx, query, tenant.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []TenantPlanDescriptor
	for rows.Next() {
		var plan TenantPlanDescriptor
		var planIDStr, serviceIDStr string
		var ratesJSON, quotasJSON []byte
		var createdAt time.Time
		if err := rows.Scan(&planIDStr, &serviceIDStr, &plan.Name, &ratesJSON, &quotasJSON, &plan.Version, &plan.Active, &createdAt); err != nil {
			return nil, err
		}
		plan.PlanID = ApplicationPlanID(planIDStr)
		plan.ServiceID = ServiceID(serviceIDStr)
		plan.CreatedAt = createdAt.Format(time.RFC3339)
		_ = json.Unmarshal(ratesJSON, &plan.Rates)
		_ = json.Unmarshal(quotasJSON, &plan.IncludedQuotas)
		list = append(list, plan)
	}
	return list, rows.Err()
}

func errorsIsNotFound(err error) bool {
	return err == pgx.ErrNoRows
}
