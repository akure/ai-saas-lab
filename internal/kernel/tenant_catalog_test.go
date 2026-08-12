package kernel

import (
	"context"
	"testing"
)

func TestMemoryTenantCatalogStore(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryTenantCatalogStore()
	tenant := MustTenantKey("tenant-acme")

	// 1. Service CRUD
	svc := TenantServiceDescriptor{
		ServiceID:   ServiceID("ai-writer"),
		Name:        "AI Writer Engine",
		Description: "Text completion service",
	}

	if err := store.RegisterService(ctx, tenant, svc); err != nil {
		t.Fatalf("failed to register service: %v", err)
	}

	gotSvc, found, err := store.GetService(ctx, tenant, ServiceID("ai-writer"))
	if err != nil || !found {
		t.Fatalf("failed to get service: found=%v, err=%v", found, err)
	}
	if gotSvc.Name != "AI Writer Engine" {
		t.Errorf("expected name 'AI Writer Engine', got %q", gotSvc.Name)
	}

	svcs, err := store.ListServices(ctx, tenant)
	if err != nil || len(svcs) != 1 {
		t.Fatalf("expected 1 service, got %d, err=%v", len(svcs), err)
	}

	// 2. Metric CRUD
	metric := TenantMetricDescriptor{
		MetricID:  MetricID("prompt_tokens"),
		ServiceID: ServiceID("ai-writer"),
		Name:      "Prompt Tokens",
		Unit:      "count",
	}

	if err := store.RegisterMetric(ctx, tenant, metric); err != nil {
		t.Fatalf("failed to register metric: %v", err)
	}

	gotMetric, found, err := store.GetMetric(ctx, tenant, MetricID("prompt_tokens"))
	if err != nil || !found {
		t.Fatalf("failed to get metric: found=%v, err=%v", found, err)
	}
	if gotMetric.Unit != "count" {
		t.Errorf("expected unit 'count', got %q", gotMetric.Unit)
	}

	// 3. Plan CRUD
	plan := TenantPlanDescriptor{
		PlanID:    ApplicationPlanID("pro-v1"),
		ServiceID: ServiceID("ai-writer"),
		Name:      "Pro Writer Plan",
		Rates: map[MetricID]float64{
			MetricID("prompt_tokens"): 0.002,
		},
		IncludedQuotas: map[MetricID]int64{
			MetricID("prompt_tokens"): 100000,
		},
		Version: 1,
		Active:  true,
	}

	if err := store.RegisterPlan(ctx, tenant, plan); err != nil {
		t.Fatalf("failed to register plan: %v", err)
	}

	gotPlan, found, err := store.GetPlan(ctx, tenant, ApplicationPlanID("pro-v1"))
	if err != nil || !found {
		t.Fatalf("failed to get plan: found=%v, err=%v", found, err)
	}
	if gotPlan.Rates[MetricID("prompt_tokens")] != 0.002 {
		t.Errorf("expected rate 0.002, got %f", gotPlan.Rates[MetricID("prompt_tokens")])
	}
}
