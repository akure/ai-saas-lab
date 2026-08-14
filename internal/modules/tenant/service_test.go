package tenant

import (
	"context"
	"errors"
	"testing"

	"aisaaslab/internal/kernel"
)

// ---------------------------------------------------------------------------
// mockCatalog — minimal in-process TenantCatalogStore for service-layer tests
// ---------------------------------------------------------------------------

type mockCatalog struct {
	services map[string]kernel.TenantServiceDescriptor
	metrics  map[string]kernel.TenantMetricDescriptor
	plans    map[string]kernel.TenantPlanDescriptor
	failOn   string // op name to force failure: "register_service", "get_service", ...
}

func newMockCatalog() *mockCatalog {
	return &mockCatalog{
		services: make(map[string]kernel.TenantServiceDescriptor),
		metrics:  make(map[string]kernel.TenantMetricDescriptor),
		plans:    make(map[string]kernel.TenantPlanDescriptor),
	}
}

func (m *mockCatalog) Name() string { return "mock" }
func (m *mockCatalog) Close() error { return nil }

func (m *mockCatalog) RegisterService(_ context.Context, tenant kernel.TenantKey, svc kernel.TenantServiceDescriptor) error {
	if m.failOn == "register_service" {
		return kernel.NewBackendError("mock", errors.New("injected failure"))
	}
	m.services[tenant.String()+":"+svc.ServiceID.String()] = svc
	return nil
}

func (m *mockCatalog) GetService(_ context.Context, tenant kernel.TenantKey, id kernel.ServiceID) (kernel.TenantServiceDescriptor, bool, error) {
	if m.failOn == "get_service" {
		return kernel.TenantServiceDescriptor{}, false, errors.New("injected get failure")
	}
	svc, ok := m.services[tenant.String()+":"+id.String()]
	return svc, ok, nil
}

func (m *mockCatalog) ListServices(_ context.Context, _ kernel.TenantKey) ([]kernel.TenantServiceDescriptor, error) {
	var list []kernel.TenantServiceDescriptor
	for _, v := range m.services {
		list = append(list, v)
	}
	return list, nil
}

func (m *mockCatalog) RegisterMetric(_ context.Context, tenant kernel.TenantKey, metric kernel.TenantMetricDescriptor) error {
	if m.failOn == "register_metric" {
		return kernel.NewBackendError("mock", errors.New("injected failure"))
	}
	m.metrics[tenant.String()+":"+metric.MetricID.String()] = metric
	return nil
}

func (m *mockCatalog) GetMetric(_ context.Context, tenant kernel.TenantKey, id kernel.MetricID) (kernel.TenantMetricDescriptor, bool, error) {
	met, ok := m.metrics[tenant.String()+":"+id.String()]
	return met, ok, nil
}

func (m *mockCatalog) ListMetrics(_ context.Context, _ kernel.TenantKey) ([]kernel.TenantMetricDescriptor, error) {
	var list []kernel.TenantMetricDescriptor
	for _, v := range m.metrics {
		list = append(list, v)
	}
	return list, nil
}

func (m *mockCatalog) RegisterPlan(_ context.Context, tenant kernel.TenantKey, plan kernel.TenantPlanDescriptor) error {
	if m.failOn == "register_plan" {
		return kernel.NewBackendError("mock", errors.New("injected failure"))
	}
	m.plans[tenant.String()+":"+plan.PlanID.String()] = plan
	return nil
}

func (m *mockCatalog) GetPlan(_ context.Context, tenant kernel.TenantKey, id kernel.ApplicationPlanID) (kernel.TenantPlanDescriptor, bool, error) {
	p, ok := m.plans[tenant.String()+":"+id.String()]
	return p, ok, nil
}

func (m *mockCatalog) ListPlans(_ context.Context, _ kernel.TenantKey) ([]kernel.TenantPlanDescriptor, error) {
	var list []kernel.TenantPlanDescriptor
	for _, v := range m.plans {
		list = append(list, v)
	}
	return list, nil
}

// ---------------------------------------------------------------------------
// RegisterService tests
// ---------------------------------------------------------------------------

func TestService_RegisterService_HappyPath(t *testing.T) {
	ctx := context.Background()
	svc := NewService(newMockCatalog(), kernel.NewEventBus())
	tenant := kernel.MustTenantKey("acme")

	res, err := svc.RegisterService(ctx, tenant, kernel.TenantServiceDescriptor{
		ServiceID: "ai-writer",
		Name:      "AI Writer",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ServiceID != "ai-writer" {
		t.Errorf("expected service_id 'ai-writer', got %q", res.ServiceID)
	}
}

func TestService_RegisterService_EmptyServiceID_Returns422(t *testing.T) {
	ctx := context.Background()
	svc := NewService(newMockCatalog(), nil)
	tenant := kernel.MustTenantKey("acme")

	_, err := svc.RegisterService(ctx, tenant, kernel.TenantServiceDescriptor{
		ServiceID: "",
		Name:      "Something",
	})
	if err == nil {
		t.Fatal("expected validation error but got nil")
	}
	if !kernel.IsCatalogValidation(err) {
		t.Errorf("expected validation error, got: %v", err)
	}
}

func TestService_RegisterService_InvalidCharset_Returns422(t *testing.T) {
	ctx := context.Background()
	svc := NewService(newMockCatalog(), nil)
	tenant := kernel.MustTenantKey("acme")

	_, err := svc.RegisterService(ctx, tenant, kernel.TenantServiceDescriptor{
		ServiceID: "AI Writer!", // uppercase + special chars — invalid
		Name:      "AI Writer",
	})
	if !kernel.IsCatalogValidation(err) {
		t.Errorf("expected validation error for bad charset, got: %v", err)
	}
}

func TestService_RegisterService_Duplicate_Returns409(t *testing.T) {
	ctx := context.Background()
	catalog := newMockCatalog()
	svc := NewService(catalog, nil)
	tenant := kernel.MustTenantKey("acme")

	d := kernel.TenantServiceDescriptor{ServiceID: "dup-svc", Name: "First"}
	if _, err := svc.RegisterService(ctx, tenant, d); err != nil {
		t.Fatalf("first registration: %v", err)
	}

	_, err := svc.RegisterService(ctx, tenant, kernel.TenantServiceDescriptor{
		ServiceID: "dup-svc",
		Name:      "Second attempt",
	})
	if err == nil {
		t.Fatal("expected conflict error but got nil")
	}
	if !kernel.IsCatalogConflict(err) {
		t.Errorf("expected conflict (409) error, got: %v", err)
	}
	if !errors.Is(err, kernel.ErrServiceAlreadyExists) {
		t.Errorf("expected errors.Is to match ErrServiceAlreadyExists, got: %v", err)
	}
}

func TestService_RegisterService_BackendFailure_Returns503(t *testing.T) {
	ctx := context.Background()
	catalog := newMockCatalog()
	catalog.failOn = "register_service"
	svc := NewService(catalog, nil)
	tenant := kernel.MustTenantKey("acme")

	_, err := svc.RegisterService(ctx, tenant, kernel.TenantServiceDescriptor{
		ServiceID: "backend-fail",
		Name:      "Test",
	})
	if err == nil {
		t.Fatal("expected error on backend failure")
	}
	// Backend error wraps through service layer
	if !kernel.IsCatalogBackend(err) {
		t.Errorf("expected backend error (503-class), got: %v", err)
	}
}

func TestService_RegisterService_DefaultsName(t *testing.T) {
	ctx := context.Background()
	svc := NewService(newMockCatalog(), nil)
	tenant := kernel.MustTenantKey("acme")

	// Name omitted — should default to service_id.
	res, err := svc.RegisterService(ctx, tenant, kernel.TenantServiceDescriptor{
		ServiceID: "no-name-svc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Name != "no-name-svc" {
		t.Errorf("expected name to default to 'no-name-svc', got %q", res.Name)
	}
}

// ---------------------------------------------------------------------------
// GetService tests
// ---------------------------------------------------------------------------

func TestService_GetService_NotFound_Returns404(t *testing.T) {
	ctx := context.Background()
	svc := NewService(newMockCatalog(), nil)
	tenant := kernel.MustTenantKey("acme")

	_, err := svc.GetService(ctx, tenant, "nonexistent")
	if err == nil {
		t.Fatal("expected not-found error")
	}
	if !kernel.IsCatalogNotFound(err) {
		t.Errorf("expected not-found (404-class) error, got: %v", err)
	}
	if !errors.Is(err, kernel.ErrServiceNotFound) {
		t.Errorf("expected errors.Is to match ErrServiceNotFound")
	}
}

// ---------------------------------------------------------------------------
// RegisterMetric referential integrity tests
// ---------------------------------------------------------------------------

func TestService_RegisterMetric_ServiceMissing_Returns422(t *testing.T) {
	ctx := context.Background()
	svc := NewService(newMockCatalog(), nil)
	tenant := kernel.MustTenantKey("acme")

	_, err := svc.RegisterMetric(ctx, tenant, kernel.TenantMetricDescriptor{
		MetricID:  "tokens",
		ServiceID: "nonexistent-service",
		Unit:      "count",
	})
	if !kernel.IsCatalogValidation(err) {
		t.Errorf("expected validation error (service not found), got: %v", err)
	}
}

func TestService_RegisterMetric_DuplicateMetricID_Returns409(t *testing.T) {
	ctx := context.Background()
	catalog := newMockCatalog()
	svc := NewService(catalog, nil)
	tenant := kernel.MustTenantKey("acme")

	// Register service first.
	if _, err := svc.RegisterService(ctx, tenant, kernel.TenantServiceDescriptor{
		ServiceID: "my-svc", Name: "My Svc",
	}); err != nil {
		t.Fatalf("register service: %v", err)
	}

	metric := kernel.TenantMetricDescriptor{
		MetricID: "tokens", ServiceID: "my-svc", Unit: "count",
	}
	if _, err := svc.RegisterMetric(ctx, tenant, metric); err != nil {
		t.Fatalf("first metric: %v", err)
	}

	_, err := svc.RegisterMetric(ctx, tenant, metric)
	if !kernel.IsCatalogConflict(err) {
		t.Errorf("expected conflict error on duplicate metric, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// RegisterPlan referential integrity tests
// ---------------------------------------------------------------------------

func TestService_RegisterPlan_MetricMissing_Returns422(t *testing.T) {
	ctx := context.Background()
	catalog := newMockCatalog()
	svc := NewService(catalog, nil)
	tenant := kernel.MustTenantKey("acme")

	if _, err := svc.RegisterService(ctx, tenant, kernel.TenantServiceDescriptor{
		ServiceID: "my-svc", Name: "My Svc",
	}); err != nil {
		t.Fatalf("register service: %v", err)
	}

	// Plan references metric "tokens" which doesn't exist yet.
	_, err := svc.RegisterPlan(ctx, tenant, kernel.TenantPlanDescriptor{
		PlanID:    "pro-v1",
		ServiceID: "my-svc",
		Name:      "Pro",
		Rates:     map[kernel.MetricID]float64{"tokens": 0.002},
	})
	if !kernel.IsCatalogValidation(err) {
		t.Errorf("expected validation error (metric not found), got: %v", err)
	}
}
