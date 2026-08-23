package subscription_test

import (
	"context"
	"testing"

	"aisaaslab/internal/kernel"
	"aisaaslab/internal/modules/subscription"
)

func TestModule_LifecycleAndPolicies(t *testing.T) {
	app := kernel.NewApp(&kernel.Config{})
	mod := subscription.New()

	if mod.Name() != "subscription" {
		t.Fatalf("expected module name 'subscription', got %s", mod.Name())
	}

	if mod.Manager() == nil {
		t.Fatalf("expected non-nil manager from module")
	}

	if err := mod.Init(app); err != nil {
		t.Fatalf("failed initializing module: %v", err)
	}

	ctx := context.Background()

	if err := mod.Start(ctx); err != nil {
		t.Fatalf("failed starting module: %v", err)
	}

	// 1. Evaluate policy "has-active-subscription"
	mod.Manager().CreateContract(subscription.Contract{TenantKey: "t_mod_policy", PlanID: "pro", State: subscription.StateActive})
	if err := app.CheckPolicies(ctx, "t_mod_policy", "has-active-subscription"); err != nil {
		t.Fatalf("expected has-active-subscription policy check to pass for active tenant: %v", err)
	}

	mod.Manager().SetState("t_mod_policy", subscription.StateCancelled)
	if err := app.CheckPolicies(ctx, "t_mod_policy", "has-active-subscription"); err == nil {
		t.Fatalf("expected has-active-subscription policy check to fail for cancelled tenant")
	}

	// 2. Evaluate policy "under-subscription-quota" and alias "under-quota"
	mod.Manager().CreateContract(subscription.Contract{TenantKey: "t_mod_quota", PlanID: "starter", State: subscription.StateActive})
	if err := app.CheckPolicies(ctx, "t_mod_quota", "under-subscription-quota"); err != nil {
		t.Fatalf("expected under-subscription-quota policy check to pass: %v", err)
	}
	if err := app.CheckPolicies(ctx, "t_mod_quota", "under-quota"); err != nil {
		t.Fatalf("expected under-quota alias policy check to pass: %v", err)
	}

	if err := mod.Stop(ctx); err != nil {
		t.Fatalf("failed stopping module: %v", err)
	}
}
