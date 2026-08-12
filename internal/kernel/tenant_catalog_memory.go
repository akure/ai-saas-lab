package kernel

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryTenantCatalogStore is an in-memory L1 implementation of TenantCatalogStore.
type MemoryTenantCatalogStore struct {
	mu       sync.RWMutex
	services map[string]map[ServiceID]TenantServiceDescriptor         // tenantKey -> serviceID -> descriptor
	metrics  map[string]map[MetricID]TenantMetricDescriptor           // tenantKey -> metricID -> descriptor
	plans    map[string]map[ApplicationPlanID]TenantPlanDescriptor    // tenantKey -> planID -> descriptor
}

func NewMemoryTenantCatalogStore() *MemoryTenantCatalogStore {
	return &MemoryTenantCatalogStore{
		services: make(map[string]map[ServiceID]TenantServiceDescriptor),
		metrics:  make(map[string]map[MetricID]TenantMetricDescriptor),
		plans:    make(map[string]map[ApplicationPlanID]TenantPlanDescriptor),
	}
}

func (m *MemoryTenantCatalogStore) Name() string { return "memory" }
func (m *MemoryTenantCatalogStore) Close() error { return nil }

// --- Service Catalog ---

func (m *MemoryTenantCatalogStore) RegisterService(ctx context.Context, tenant TenantKey, svc TenantServiceDescriptor) error {
	if tenant.IsZero() {
		return fmt.Errorf("register service: tenant key required")
	}
	if err := svc.ServiceID.Validate(); err != nil {
		return fmt.Errorf("register service: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	tk := tenant.String()
	if m.services[tk] == nil {
		m.services[tk] = make(map[ServiceID]TenantServiceDescriptor)
	}
	if svc.CreatedAt == "" {
		svc.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	m.services[tk][svc.ServiceID] = svc
	return nil
}

func (m *MemoryTenantCatalogStore) GetService(ctx context.Context, tenant TenantKey, id ServiceID) (TenantServiceDescriptor, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tk := tenant.String()
	tenantServices, ok := m.services[tk]
	if !ok {
		return TenantServiceDescriptor{}, false, nil
	}
	svc, found := tenantServices[id]
	return svc, found, nil
}

func (m *MemoryTenantCatalogStore) ListServices(ctx context.Context, tenant TenantKey) ([]TenantServiceDescriptor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tk := tenant.String()
	tenantServices, ok := m.services[tk]
	if !ok {
		return []TenantServiceDescriptor{}, nil
	}

	list := make([]TenantServiceDescriptor, 0, len(tenantServices))
	for _, svc := range tenantServices {
		list = append(list, svc)
	}
	return list, nil
}

// --- Metric Catalog ---

func (m *MemoryTenantCatalogStore) RegisterMetric(ctx context.Context, tenant TenantKey, metric TenantMetricDescriptor) error {
	if tenant.IsZero() {
		return fmt.Errorf("register metric: tenant key required")
	}
	if err := metric.MetricID.Validate(); err != nil {
		return fmt.Errorf("register metric: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	tk := tenant.String()
	if m.metrics[tk] == nil {
		m.metrics[tk] = make(map[MetricID]TenantMetricDescriptor)
	}
	if metric.CreatedAt == "" {
		metric.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	m.metrics[tk][metric.MetricID] = metric
	return nil
}

func (m *MemoryTenantCatalogStore) GetMetric(ctx context.Context, tenant TenantKey, id MetricID) (TenantMetricDescriptor, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tk := tenant.String()
	tenantMetrics, ok := m.metrics[tk]
	if !ok {
		return TenantMetricDescriptor{}, false, nil
	}
	metric, found := tenantMetrics[id]
	return metric, found, nil
}

func (m *MemoryTenantCatalogStore) ListMetrics(ctx context.Context, tenant TenantKey) ([]TenantMetricDescriptor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tk := tenant.String()
	tenantMetrics, ok := m.metrics[tk]
	if !ok {
		return []TenantMetricDescriptor{}, nil
	}

	list := make([]TenantMetricDescriptor, 0, len(tenantMetrics))
	for _, metric := range tenantMetrics {
		list = append(list, metric)
	}
	return list, nil
}

// --- Plan Catalog ---

func (m *MemoryTenantCatalogStore) RegisterPlan(ctx context.Context, tenant TenantKey, plan TenantPlanDescriptor) error {
	if tenant.IsZero() {
		return fmt.Errorf("register plan: tenant key required")
	}
	if plan.PlanID.IsZero() {
		return fmt.Errorf("register plan: plan_id required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	tk := tenant.String()
	if m.plans[tk] == nil {
		m.plans[tk] = make(map[ApplicationPlanID]TenantPlanDescriptor)
	}
	if plan.CreatedAt == "" {
		plan.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if plan.Version <= 0 {
		plan.Version = 1
	}
	m.plans[tk][plan.PlanID] = plan
	return nil
}

func (m *MemoryTenantCatalogStore) GetPlan(ctx context.Context, tenant TenantKey, id ApplicationPlanID) (TenantPlanDescriptor, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tk := tenant.String()
	tenantPlans, ok := m.plans[tk]
	if !ok {
		return TenantPlanDescriptor{}, false, nil
	}
	plan, found := tenantPlans[id]
	return plan, found, nil
}

func (m *MemoryTenantCatalogStore) ListPlans(ctx context.Context, tenant TenantKey) ([]TenantPlanDescriptor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tk := tenant.String()
	tenantPlans, ok := m.plans[tk]
	if !ok {
		return []TenantPlanDescriptor{}, nil
	}

	list := make([]TenantPlanDescriptor, 0, len(tenantPlans))
	for _, plan := range tenantPlans {
		list = append(list, plan)
	}
	return list, nil
}
