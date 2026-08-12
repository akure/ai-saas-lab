package tenant

import (
	"context"
	"fmt"

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

// --- Service Registration ---

func (s *Service) RegisterService(ctx context.Context, tenant kernel.TenantKey, descriptor kernel.TenantServiceDescriptor) (kernel.TenantServiceDescriptor, error) {
	if s.catalog == nil {
		return kernel.TenantServiceDescriptor{}, fmt.Errorf("catalog store uninitialized")
	}
	if err := descriptor.ServiceID.Validate(); err != nil {
		return kernel.TenantServiceDescriptor{}, err
	}
	if descriptor.Name == "" {
		descriptor.Name = descriptor.ServiceID.String()
	}

	if err := s.catalog.RegisterService(ctx, tenant, descriptor); err != nil {
		return kernel.TenantServiceDescriptor{}, err
	}

	if s.events != nil {
		s.events.Publish("tenant.service.registered", map[string]any{
			"tenant_key": tenant.String(),
			"service_id": descriptor.ServiceID.String(),
			"name":       descriptor.Name,
		})
	}

	res, _, err := s.catalog.GetService(ctx, tenant, descriptor.ServiceID)
	return res, err
}

func (s *Service) ListServices(ctx context.Context, tenant kernel.TenantKey) ([]kernel.TenantServiceDescriptor, error) {
	if s.catalog == nil {
		return []kernel.TenantServiceDescriptor{}, nil
	}
	return s.catalog.ListServices(ctx, tenant)
}

// --- Metric Registration ---

func (s *Service) RegisterMetric(ctx context.Context, tenant kernel.TenantKey, descriptor kernel.TenantMetricDescriptor) (kernel.TenantMetricDescriptor, error) {
	if s.catalog == nil {
		return kernel.TenantMetricDescriptor{}, fmt.Errorf("catalog store uninitialized")
	}
	if err := descriptor.MetricID.Validate(); err != nil {
		return kernel.TenantMetricDescriptor{}, err
	}
	if err := descriptor.ServiceID.Validate(); err != nil {
		return kernel.TenantMetricDescriptor{}, err
	}

	// Referential Integrity: Ensure the target service is registered first
	_, found, err := s.catalog.GetService(ctx, tenant, descriptor.ServiceID)
	if err != nil {
		return kernel.TenantMetricDescriptor{}, fmt.Errorf("verify service: %w", err)
	}
	if !found {
		return kernel.TenantMetricDescriptor{}, fmt.Errorf("referenced service_id %q must be registered before creating metric %q", descriptor.ServiceID, descriptor.MetricID)
	}

	if descriptor.Name == "" {
		descriptor.Name = descriptor.MetricID.String()
	}
	if descriptor.Unit == "" {
		descriptor.Unit = "count"
	}

	if err := s.catalog.RegisterMetric(ctx, tenant, descriptor); err != nil {
		return kernel.TenantMetricDescriptor{}, err
	}

	if s.events != nil {
		s.events.Publish("tenant.metric.registered", map[string]any{
			"tenant_key": tenant.String(),
			"metric_id":  descriptor.MetricID.String(),
			"service_id": descriptor.ServiceID.String(),
		})
	}

	res, _, err := s.catalog.GetMetric(ctx, tenant, descriptor.MetricID)
	return res, err
}

func (s *Service) ListMetrics(ctx context.Context, tenant kernel.TenantKey) ([]kernel.TenantMetricDescriptor, error) {
	if s.catalog == nil {
		return []kernel.TenantMetricDescriptor{}, nil
	}
	return s.catalog.ListMetrics(ctx, tenant)
}

// --- Plan Registration ---

func (s *Service) RegisterPlan(ctx context.Context, tenant kernel.TenantKey, descriptor kernel.TenantPlanDescriptor) (kernel.TenantPlanDescriptor, error) {
	if s.catalog == nil {
		return kernel.TenantPlanDescriptor{}, fmt.Errorf("catalog store uninitialized")
	}
	if descriptor.PlanID.IsZero() {
		return kernel.TenantPlanDescriptor{}, fmt.Errorf("plan_id cannot be empty")
	}
	if err := descriptor.ServiceID.Validate(); err != nil {
		return kernel.TenantPlanDescriptor{}, err
	}

	// Referential Integrity Check 1: Ensure Service exists
	_, found, err := s.catalog.GetService(ctx, tenant, descriptor.ServiceID)
	if err != nil {
		return kernel.TenantPlanDescriptor{}, fmt.Errorf("verify service: %w", err)
	}
	if !found {
		return kernel.TenantPlanDescriptor{}, fmt.Errorf("referenced service_id %q must be registered before creating plan %q", descriptor.ServiceID, descriptor.PlanID)
	}

	// Referential Integrity Check 2: Ensure all rated MetricIDs exist
	for mID := range descriptor.Rates {
		_, foundMetric, err := s.catalog.GetMetric(ctx, tenant, mID)
		if err != nil {
			return kernel.TenantPlanDescriptor{}, fmt.Errorf("verify metric %q: %w", mID, err)
		}
		if !foundMetric {
			return kernel.TenantPlanDescriptor{}, fmt.Errorf("referenced metric_id %q must be registered before attaching rate to plan %q", mID, descriptor.PlanID)
		}
	}

	if descriptor.Name == "" {
		descriptor.Name = descriptor.PlanID.String()
	}
	if descriptor.Version <= 0 {
		descriptor.Version = 1
	}
	descriptor.Active = true

	if err := s.catalog.RegisterPlan(ctx, tenant, descriptor); err != nil {
		return kernel.TenantPlanDescriptor{}, err
	}

	if s.events != nil {
		s.events.Publish("tenant.plan.created", map[string]any{
			"tenant_key": tenant.String(),
			"plan_id":    descriptor.PlanID.String(),
			"service_id": descriptor.ServiceID.String(),
			"version":    descriptor.Version,
		})
	}

	res, _, err := s.catalog.GetPlan(ctx, tenant, descriptor.PlanID)
	return res, err
}

func (s *Service) ListPlans(ctx context.Context, tenant kernel.TenantKey) ([]kernel.TenantPlanDescriptor, error) {
	if s.catalog == nil {
		return []kernel.TenantPlanDescriptor{}, nil
	}
	return s.catalog.ListPlans(ctx, tenant)
}

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
