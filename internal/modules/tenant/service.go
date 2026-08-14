package tenant

import (
	"context"
	"fmt"
	"time"

	"aisaaslab/internal/kernel"
)

// Service handles tenant self-service catalog operations and event notifications.
type Service struct {
	catalog kernel.TenantCatalogStore
	events  *kernel.EventBus
}

func NewService(catalog kernel.TenantCatalogStore, events *kernel.EventBus) *Service {
	return &Service{
		catalog: catalog,
		events:  events,
	}
}

// ---------------------------------------------------------------------------
// Service Catalog
// ---------------------------------------------------------------------------

// RegisterService validates, checks for conflicts, persists, then publishes
// a TopicServiceRegistered event. Returns 409-class error on duplicate service_id.
func (s *Service) RegisterService(ctx context.Context, tenant kernel.TenantKey, descriptor kernel.TenantServiceDescriptor) (kernel.TenantServiceDescriptor, error) {
	if s.catalog == nil {
		return kernel.TenantServiceDescriptor{}, fmt.Errorf("register service: catalog store uninitialized")
	}

	// 1. Default name to service_id slug before validation runs.
	if descriptor.Name == "" {
		descriptor.Name = descriptor.ServiceID.String()
	}

	// 2. Descriptor-level validation (charset, length, required fields).
	if err := descriptor.Validate(); err != nil {
		return kernel.TenantServiceDescriptor{}, fmt.Errorf("register service: %w", err)
	}

	// 3. Conflict check — duplicate service_id is always 409.
	_, exists, err := s.catalog.GetService(ctx, tenant, descriptor.ServiceID)
	if err != nil {
		return kernel.TenantServiceDescriptor{}, fmt.Errorf("register service: check existing: %w", err)
	}
	if exists {
		return kernel.TenantServiceDescriptor{}, fmt.Errorf("register service: %w",
			kernel.NewConflictError("service", descriptor.ServiceID.String()))
	}

	// 4. Persist through CatalogChain (L1→L2→L3, WAL on failure).
	if err := s.catalog.RegisterService(ctx, tenant, descriptor); err != nil {
		return kernel.TenantServiceDescriptor{}, fmt.Errorf("register service: %w", err)
	}

	// 5. Publish typed event — fire-and-forget via EventBus goroutines.
	if s.events != nil {
		s.events.Publish(kernel.TopicServiceRegistered, kernel.ServiceRegisteredEvent{
			TenantKey:    tenant.String(),
			ServiceID:    descriptor.ServiceID.String(),
			Name:         descriptor.Name,
			RegisteredAt: time.Now().UTC(),
		})
	}

	// 6. Return the freshly-persisted descriptor.
	res, _, err := s.catalog.GetService(ctx, tenant, descriptor.ServiceID)
	if err != nil {
		return kernel.TenantServiceDescriptor{}, fmt.Errorf("register service: fetch result: %w", err)
	}
	return res, nil
}

// GetService fetches a single service descriptor by ID.
// Returns ErrServiceNotFound (404-class) when not found.
func (s *Service) GetService(ctx context.Context, tenant kernel.TenantKey, id kernel.ServiceID) (kernel.TenantServiceDescriptor, error) {
	if s.catalog == nil {
		return kernel.TenantServiceDescriptor{}, kernel.NewNotFoundError("service", id.String())
	}
	svc, found, err := s.catalog.GetService(ctx, tenant, id)
	if err != nil {
		return kernel.TenantServiceDescriptor{}, fmt.Errorf("get service: %w", err)
	}
	if !found {
		return kernel.TenantServiceDescriptor{}, kernel.NewNotFoundError("service", id.String())
	}
	return svc, nil
}

func (s *Service) ListServices(ctx context.Context, tenant kernel.TenantKey) ([]kernel.TenantServiceDescriptor, error) {
	if s.catalog == nil {
		return []kernel.TenantServiceDescriptor{}, nil
	}
	return s.catalog.ListServices(ctx, tenant)
}

// ---------------------------------------------------------------------------
// Metric Catalog
// ---------------------------------------------------------------------------

// RegisterMetric validates, checks referential integrity (service must exist),
// checks for duplicate metric_id (409), then persists and publishes an event.
func (s *Service) RegisterMetric(ctx context.Context, tenant kernel.TenantKey, descriptor kernel.TenantMetricDescriptor) (kernel.TenantMetricDescriptor, error) {
	if s.catalog == nil {
		return kernel.TenantMetricDescriptor{}, fmt.Errorf("register metric: catalog store uninitialized")
	}

	// 1. Descriptor-level validation.
	if err := descriptor.Validate(); err != nil {
		return kernel.TenantMetricDescriptor{}, fmt.Errorf("register metric: %w", err)
	}

	// 2. Referential integrity: service must exist.
	_, found, err := s.catalog.GetService(ctx, tenant, descriptor.ServiceID)
	if err != nil {
		return kernel.TenantMetricDescriptor{}, fmt.Errorf("register metric: verify service: %w", err)
	}
	if !found {
		return kernel.TenantMetricDescriptor{}, fmt.Errorf("register metric: %w",
			kernel.NewValidationError("service_id",
				fmt.Sprintf("referenced service_id %q must be registered before creating metric %q",
					descriptor.ServiceID, descriptor.MetricID)))
	}

	// 3. Conflict check.
	_, exists, err := s.catalog.GetMetric(ctx, tenant, descriptor.MetricID)
	if err != nil {
		return kernel.TenantMetricDescriptor{}, fmt.Errorf("register metric: check existing: %w", err)
	}
	if exists {
		return kernel.TenantMetricDescriptor{}, fmt.Errorf("register metric: %w",
			kernel.NewConflictError("metric", descriptor.MetricID.String()))
	}

	// 4. Default name / unit.
	if descriptor.Name == "" {
		descriptor.Name = descriptor.MetricID.String()
	}
	if descriptor.Unit == "" {
		descriptor.Unit = "count"
	}

	// 5. Persist.
	if err := s.catalog.RegisterMetric(ctx, tenant, descriptor); err != nil {
		return kernel.TenantMetricDescriptor{}, fmt.Errorf("register metric: %w", err)
	}

	// 6. Publish typed event.
	if s.events != nil {
		s.events.Publish(kernel.TopicMetricRegistered, kernel.MetricRegisteredEvent{
			TenantKey:    tenant.String(),
			MetricID:     descriptor.MetricID.String(),
			ServiceID:    descriptor.ServiceID.String(),
			Unit:         descriptor.Unit,
			RegisteredAt: time.Now().UTC(),
		})
	}

	res, _, err := s.catalog.GetMetric(ctx, tenant, descriptor.MetricID)
	if err != nil {
		return kernel.TenantMetricDescriptor{}, fmt.Errorf("register metric: fetch result: %w", err)
	}
	return res, nil
}

// GetMetric fetches a single metric descriptor by ID.
func (s *Service) GetMetric(ctx context.Context, tenant kernel.TenantKey, id kernel.MetricID) (kernel.TenantMetricDescriptor, error) {
	if s.catalog == nil {
		return kernel.TenantMetricDescriptor{}, kernel.NewNotFoundError("metric", id.String())
	}
	m, found, err := s.catalog.GetMetric(ctx, tenant, id)
	if err != nil {
		return kernel.TenantMetricDescriptor{}, fmt.Errorf("get metric: %w", err)
	}
	if !found {
		return kernel.TenantMetricDescriptor{}, kernel.NewNotFoundError("metric", id.String())
	}
	return m, nil
}

func (s *Service) ListMetrics(ctx context.Context, tenant kernel.TenantKey) ([]kernel.TenantMetricDescriptor, error) {
	if s.catalog == nil {
		return []kernel.TenantMetricDescriptor{}, nil
	}
	return s.catalog.ListMetrics(ctx, tenant)
}

// ---------------------------------------------------------------------------
// Plan Catalog
// ---------------------------------------------------------------------------

// RegisterPlan validates referential integrity (service + all rated metrics must
// exist), checks for duplicate plan_id (409), then persists and publishes an event.
func (s *Service) RegisterPlan(ctx context.Context, tenant kernel.TenantKey, descriptor kernel.TenantPlanDescriptor) (kernel.TenantPlanDescriptor, error) {
	if s.catalog == nil {
		return kernel.TenantPlanDescriptor{}, fmt.Errorf("register plan: catalog store uninitialized")
	}
	if descriptor.PlanID.IsZero() {
		return kernel.TenantPlanDescriptor{}, fmt.Errorf("register plan: %w",
			kernel.NewValidationError("plan_id", "plan_id cannot be empty"))
	}
	if err := descriptor.ServiceID.Validate(); err != nil {
		return kernel.TenantPlanDescriptor{}, fmt.Errorf("register plan: %w",
			kernel.NewValidationError("service_id", err.Error()))
	}

	// 1. Referential integrity: service must exist.
	_, found, err := s.catalog.GetService(ctx, tenant, descriptor.ServiceID)
	if err != nil {
		return kernel.TenantPlanDescriptor{}, fmt.Errorf("register plan: verify service: %w", err)
	}
	if !found {
		return kernel.TenantPlanDescriptor{}, fmt.Errorf("register plan: %w",
			kernel.NewValidationError("service_id",
				fmt.Sprintf("referenced service_id %q must be registered before creating plan %q",
					descriptor.ServiceID, descriptor.PlanID)))
	}

	// 2. Referential integrity: all rated MetricIDs must exist.
	for mID := range descriptor.Rates {
		_, foundMetric, err := s.catalog.GetMetric(ctx, tenant, mID)
		if err != nil {
			return kernel.TenantPlanDescriptor{}, fmt.Errorf("register plan: verify metric %q: %w", mID, err)
		}
		if !foundMetric {
			return kernel.TenantPlanDescriptor{}, fmt.Errorf("register plan: %w",
				kernel.NewValidationError("rates",
					fmt.Sprintf("metric_id %q must be registered before attaching a rate to plan %q",
						mID, descriptor.PlanID)))
		}
	}

	// 3. Conflict check.
	_, exists, err := s.catalog.GetPlan(ctx, tenant, descriptor.PlanID)
	if err != nil {
		return kernel.TenantPlanDescriptor{}, fmt.Errorf("register plan: check existing: %w", err)
	}
	if exists {
		return kernel.TenantPlanDescriptor{}, fmt.Errorf("register plan: %w",
			kernel.NewConflictError("plan", descriptor.PlanID.String()))
	}

	// 4. Defaults.
	if descriptor.Name == "" {
		descriptor.Name = descriptor.PlanID.String()
	}
	if descriptor.Version <= 0 {
		descriptor.Version = 1
	}
	descriptor.Active = true

	// 5. Persist.
	if err := s.catalog.RegisterPlan(ctx, tenant, descriptor); err != nil {
		return kernel.TenantPlanDescriptor{}, fmt.Errorf("register plan: %w", err)
	}

	// 6. Publish typed event.
	if s.events != nil {
		s.events.Publish(kernel.TopicPlanRegistered, kernel.PlanRegisteredEvent{
			TenantKey:    tenant.String(),
			PlanID:       descriptor.PlanID.String(),
			ServiceID:    descriptor.ServiceID.String(),
			Version:      descriptor.Version,
			RegisteredAt: time.Now().UTC(),
		})
	}

	res, _, err := s.catalog.GetPlan(ctx, tenant, descriptor.PlanID)
	if err != nil {
		return kernel.TenantPlanDescriptor{}, fmt.Errorf("register plan: fetch result: %w", err)
	}
	return res, nil
}

// GetPlan fetches a single plan descriptor by ID.
func (s *Service) GetPlan(ctx context.Context, tenant kernel.TenantKey, id kernel.ApplicationPlanID) (kernel.TenantPlanDescriptor, error) {
	if s.catalog == nil {
		return kernel.TenantPlanDescriptor{}, kernel.NewNotFoundError("plan", id.String())
	}
	p, found, err := s.catalog.GetPlan(ctx, tenant, id)
	if err != nil {
		return kernel.TenantPlanDescriptor{}, fmt.Errorf("get plan: %w", err)
	}
	if !found {
		return kernel.TenantPlanDescriptor{}, kernel.NewNotFoundError("plan", id.String())
	}
	return p, nil
}

func (s *Service) ListPlans(ctx context.Context, tenant kernel.TenantKey) ([]kernel.TenantPlanDescriptor, error) {
	if s.catalog == nil {
		return []kernel.TenantPlanDescriptor{}, nil
	}
	return s.catalog.ListPlans(ctx, tenant)
}

// ---------------------------------------------------------------------------
// Overview
// ---------------------------------------------------------------------------

// Overview returns a full catalog snapshot for the tenant.
func (s *Service) Overview(ctx context.Context, tenant kernel.TenantKey) (map[string]any, error) {
	svcs, err := s.ListServices(ctx, tenant)
	if err != nil {
		return nil, err
	}
	metrics, err := s.ListMetrics(ctx, tenant)
	if err != nil {
		return nil, err
	}
	plans, err := s.ListPlans(ctx, tenant)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"tenant_key": tenant.String(),
		"services":   svcs,
		"metrics":    metrics,
		"plans":      plans,
	}, nil
}
