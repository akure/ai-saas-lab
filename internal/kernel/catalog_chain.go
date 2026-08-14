package kernel

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ---------------------------------------------------------------------------
// CatalogChain — layered TenantCatalogStore orchestrator
//
// Write strategy:  write-through all backends (L1 Memory → L2 Redis → L3 Postgres).
//                  On backend failure: append to CatalogWAL for replay.
//                  Return nil if at least L1 (memory) succeeds.
//
// Read strategy:   read-through: L1 hit → return; L1 miss → L2; L2 hit → back-fill
//                  L1 + return; L2 miss → L3; L3 hit → back-fill L1+L2 + return.
//
// ListXxx:         always authoritative from L3 (Postgres); warms L1+L2.
// ---------------------------------------------------------------------------

// CatalogChain implements TenantCatalogStore by orchestrating multiple backends.
type CatalogChain struct {
	backends []TenantCatalogStore // ordered: [0]=L1 memory, [1]=L2 redis, [2]=L3 postgres
	wal      *CatalogWAL
}

// NewCatalogChain creates a CatalogChain. backends must have at least one
// element (the L1 memory store). wal may be nil when WAL is disabled.
func NewCatalogChain(backends []TenantCatalogStore, wal *CatalogWAL) (*CatalogChain, error) {
	if len(backends) == 0 {
		return nil, fmt.Errorf("catalog chain: at least one backend required")
	}
	return &CatalogChain{backends: backends, wal: wal}, nil
}

func (c *CatalogChain) Name() string { return "catalog-chain" }

// Close shuts down all backends in reverse order and stops the WAL.
func (c *CatalogChain) Close() error {
	var lastErr error
	for i := len(c.backends) - 1; i >= 0; i-- {
		if err := c.backends[i].Close(); err != nil {
			fmt.Printf("[catalog chain] close backend %s: %v\n", c.backends[i].Name(), err)
			lastErr = err
		}
	}
	if c.wal != nil {
		if err := c.wal.Stop(); err != nil {
			fmt.Printf("[catalog chain] stop WAL: %v\n", err)
			lastErr = err
		}
	}
	return lastErr
}

// ---------------------------------------------------------------------------
// RegisterService
// ---------------------------------------------------------------------------

func (c *CatalogChain) RegisterService(ctx context.Context, tenant TenantKey, svc TenantServiceDescriptor) error {
	var firstErr error
	for _, b := range c.backends {
		if err := b.RegisterService(ctx, tenant, svc); err != nil {
			berr := fmt.Errorf("register service [%s]: %w", b.Name(), err)
			if firstErr == nil {
				firstErr = berr
			}
			// WAL only for durable backends (not L1 memory).
			if b.Name() != "memory" {
				c.walAppend(ctx, "register_service", b.Name(), tenant, svc)
			}
		}
	}
	// L1 success is enough for immediate consistency; WAL handles durability.
	if c.backends[0].Name() == "memory" {
		_, l1ok, _ := c.backends[0].GetService(ctx, tenant, svc.ServiceID)
		if l1ok {
			return nil
		}
	}
	if firstErr != nil {
		return NewBackendError("all", firstErr)
	}
	return nil
}

// GetService implements L1→L2→L3 read-through with back-fill.
func (c *CatalogChain) GetService(ctx context.Context, tenant TenantKey, id ServiceID) (TenantServiceDescriptor, bool, error) {
	for i, b := range c.backends {
		svc, found, err := b.GetService(ctx, tenant, id)
		if err != nil {
			// Backend error on read — try next layer.
			continue
		}
		if found {
			// Back-fill all faster layers that missed.
			c.backfillService(ctx, i, tenant, svc)
			return svc, true, nil
		}
	}
	return TenantServiceDescriptor{}, false, nil
}

// backfillService writes svc into backends[0..upToIndex-1] (the layers that missed).
func (c *CatalogChain) backfillService(ctx context.Context, foundAt int, tenant TenantKey, svc TenantServiceDescriptor) {
	for i := 0; i < foundAt; i++ {
		_ = c.backends[i].RegisterService(ctx, tenant, svc)
	}
}

// ListServices reads from L3 (source of truth) and warms L1+L2.
func (c *CatalogChain) ListServices(ctx context.Context, tenant TenantKey) ([]TenantServiceDescriptor, error) {
	authoritative := c.authoritativeBackend()
	list, err := authoritative.ListServices(ctx, tenant)
	if err != nil {
		// Fall back to L1 if L3 fails.
		return c.backends[0].ListServices(ctx, tenant)
	}
	// Warm faster layers.
	for _, svc := range list {
		for i := 0; i < len(c.backends)-1; i++ {
			_ = c.backends[i].RegisterService(ctx, tenant, svc)
		}
	}
	return list, nil
}

// ---------------------------------------------------------------------------
// RegisterMetric
// ---------------------------------------------------------------------------

func (c *CatalogChain) RegisterMetric(ctx context.Context, tenant TenantKey, metric TenantMetricDescriptor) error {
	var firstErr error
	for _, b := range c.backends {
		if err := b.RegisterMetric(ctx, tenant, metric); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("register metric [%s]: %w", b.Name(), err)
			}
			if b.Name() != "memory" {
				c.walAppend(ctx, "register_metric", b.Name(), tenant, metric)
			}
		}
	}
	if c.backends[0].Name() == "memory" {
		_, l1ok, _ := c.backends[0].GetMetric(ctx, tenant, metric.MetricID)
		if l1ok {
			return nil
		}
	}
	if firstErr != nil {
		return NewBackendError("all", firstErr)
	}
	return nil
}

func (c *CatalogChain) GetMetric(ctx context.Context, tenant TenantKey, id MetricID) (TenantMetricDescriptor, bool, error) {
	for i, b := range c.backends {
		m, found, err := b.GetMetric(ctx, tenant, id)
		if err != nil {
			continue
		}
		if found {
			c.backfillMetric(ctx, i, tenant, m)
			return m, true, nil
		}
	}
	return TenantMetricDescriptor{}, false, nil
}

func (c *CatalogChain) backfillMetric(ctx context.Context, foundAt int, tenant TenantKey, m TenantMetricDescriptor) {
	for i := 0; i < foundAt; i++ {
		_ = c.backends[i].RegisterMetric(ctx, tenant, m)
	}
}

func (c *CatalogChain) ListMetrics(ctx context.Context, tenant TenantKey) ([]TenantMetricDescriptor, error) {
	list, err := c.authoritativeBackend().ListMetrics(ctx, tenant)
	if err != nil {
		return c.backends[0].ListMetrics(ctx, tenant)
	}
	for _, m := range list {
		for i := 0; i < len(c.backends)-1; i++ {
			_ = c.backends[i].RegisterMetric(ctx, tenant, m)
		}
	}
	return list, nil
}

// ---------------------------------------------------------------------------
// RegisterPlan
// ---------------------------------------------------------------------------

func (c *CatalogChain) RegisterPlan(ctx context.Context, tenant TenantKey, plan TenantPlanDescriptor) error {
	var firstErr error
	for _, b := range c.backends {
		if err := b.RegisterPlan(ctx, tenant, plan); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("register plan [%s]: %w", b.Name(), err)
			}
			if b.Name() != "memory" {
				c.walAppend(ctx, "register_plan", b.Name(), tenant, plan)
			}
		}
	}
	if c.backends[0].Name() == "memory" {
		_, l1ok, _ := c.backends[0].GetPlan(ctx, tenant, plan.PlanID)
		if l1ok {
			return nil
		}
	}
	if firstErr != nil {
		return NewBackendError("all", firstErr)
	}
	return nil
}

func (c *CatalogChain) GetPlan(ctx context.Context, tenant TenantKey, id ApplicationPlanID) (TenantPlanDescriptor, bool, error) {
	for i, b := range c.backends {
		p, found, err := b.GetPlan(ctx, tenant, id)
		if err != nil {
			continue
		}
		if found {
			c.backfillPlan(ctx, i, tenant, p)
			return p, true, nil
		}
	}
	return TenantPlanDescriptor{}, false, nil
}

func (c *CatalogChain) backfillPlan(ctx context.Context, foundAt int, tenant TenantKey, p TenantPlanDescriptor) {
	for i := 0; i < foundAt; i++ {
		_ = c.backends[i].RegisterPlan(ctx, tenant, p)
	}
}

func (c *CatalogChain) ListPlans(ctx context.Context, tenant TenantKey) ([]TenantPlanDescriptor, error) {
	list, err := c.authoritativeBackend().ListPlans(ctx, tenant)
	if err != nil {
		return c.backends[0].ListPlans(ctx, tenant)
	}
	for _, p := range list {
		for i := 0; i < len(c.backends)-1; i++ {
			_ = c.backends[i].RegisterPlan(ctx, tenant, p)
		}
	}
	return list, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// authoritativeBackend returns the last backend (L3 Postgres if present, else L1).
func (c *CatalogChain) authoritativeBackend() TenantCatalogStore {
	return c.backends[len(c.backends)-1]
}

// walAppend serialises the payload and appends a CatalogWALEntry.
// Errors are logged but not propagated — WAL failure must not block the write path.
func (c *CatalogChain) walAppend(_ context.Context, op, backend string, tenant TenantKey, payload any) {
	if c.wal == nil {
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("[catalog chain] wal: marshal payload for op=%s: %v\n", op, err)
		return
	}
	entry := CatalogWALEntry{
		Timestamp:   time.Now().UTC(),
		Operation:   op,
		BackendName: backend,
		TenantKey:   tenant.String(),
		Payload:     data,
	}
	if err := c.wal.Append(entry); err != nil {
		fmt.Printf("[catalog chain] wal append failed op=%s backend=%s: %v\n", op, backend, err)
	}
}

// replayWALEntry is passed to CatalogWAL.ReplayLoop. It finds the backend that
// originally failed (by name) and replays the write operation against it.
func (c *CatalogChain) replayWALEntry(ctx context.Context, entry CatalogWALEntry) error {
	var target TenantCatalogStore
	for _, b := range c.backends {
		if b.Name() == entry.BackendName {
			target = b
			break
		}
	}
	if target == nil {
		// Backend no longer configured — discard the entry.
		return nil
	}

	tenant, err := NewTenantKey(entry.TenantKey)
	if err != nil {
		return fmt.Errorf("wal replay: invalid tenant key %q: %w", entry.TenantKey, err)
	}

	switch entry.Operation {
	case "register_service":
		var svc TenantServiceDescriptor
		if err := json.Unmarshal(entry.Payload, &svc); err != nil {
			return fmt.Errorf("wal replay: unmarshal service: %w", err)
		}
		return target.RegisterService(ctx, tenant, svc)

	case "register_metric":
		var metric TenantMetricDescriptor
		if err := json.Unmarshal(entry.Payload, &metric); err != nil {
			return fmt.Errorf("wal replay: unmarshal metric: %w", err)
		}
		return target.RegisterMetric(ctx, tenant, metric)

	case "register_plan":
		var plan TenantPlanDescriptor
		if err := json.Unmarshal(entry.Payload, &plan); err != nil {
			return fmt.Errorf("wal replay: unmarshal plan: %w", err)
		}
		return target.RegisterPlan(ctx, tenant, plan)

	default:
		fmt.Printf("[catalog chain] wal replay: unknown op %q, discarding\n", entry.Operation)
		return nil
	}
}
