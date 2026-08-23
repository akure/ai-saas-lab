package subscription_test

import (
	"errors"
	"testing"
	"time"

	"aisaaslab/internal/kernel"
	"aisaaslab/internal/modules/subscription"
)

func TestState_Methods(t *testing.T) {
	stTrial := subscription.StateTrial
	stActive := subscription.StateActive
	stPastDue := subscription.StatePastDue
	stCancelled := subscription.StateCancelled

	if stTrial.String() != "trial" {
		t.Errorf("expected string 'trial', got %s", stTrial.String())
	}
	if subscription.State("").IsZero() != true {
		t.Errorf("expected IsZero to be true for empty state")
	}
	if stTrial.IsZero() != false {
		t.Errorf("expected IsZero to be false for trial state")
	}
	if !stTrial.IsValid() || !stActive.IsValid() || !stPastDue.IsValid() || !stCancelled.IsValid() {
		t.Errorf("expected all standard states to be valid")
	}
	if subscription.State("unknown").IsValid() {
		t.Errorf("expected 'unknown' state to be invalid")
	}

	if !stTrial.IsUsable() || !stActive.IsUsable() {
		t.Errorf("expected trial and active states to be usable")
	}
	if stPastDue.IsUsable() || stCancelled.IsUsable() {
		t.Errorf("expected past_due and cancelled states to be not usable")
	}

	if !stActive.IsActive() || stTrial.IsActive() {
		t.Errorf("IsActive check failed")
	}
	if !stPastDue.IsPastDue() || stActive.IsPastDue() {
		t.Errorf("IsPastDue check failed")
	}
	if !stCancelled.IsCancelled() || stActive.IsCancelled() {
		t.Errorf("IsCancelled check failed")
	}

	if !stActive.Equal(subscription.State(" ACTIVE ")) {
		t.Errorf("Equal case-insensitive check failed")
	}
}

func TestParseState(t *testing.T) {
	st, err := subscription.ParseState(" ACTIVE ")
	if err != nil || st != subscription.StateActive {
		t.Fatalf("expected active state, got %v, err=%v", st, err)
	}

	_, err = subscription.ParseState("invalid_state_name")
	if err == nil || !errors.Is(err, subscription.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState for invalid state string")
	}
}

func TestManager_FireTransition_FSM(t *testing.T) {
	mgr := subscription.NewManager()

	// Initial default state is StateTrial
	if st := mgr.GetState("tenant_fsm"); st != subscription.StateTrial {
		t.Fatalf("expected trial state initially, got %s", st)
	}

	// 1. trial -> activate -> active
	next, err := mgr.FireTransition("tenant_fsm", "activate")
	if err != nil || next != subscription.StateActive {
		t.Fatalf("expected activate transition to succeed, got %s, err=%v", next, err)
	}

	// 2. active -> payment_failed -> past_due
	next, err = mgr.FireTransition("tenant_fsm", "payment_failed")
	if err != nil || next != subscription.StatePastDue {
		t.Fatalf("expected payment_failed transition to succeed, got %s, err=%v", next, err)
	}

	// 3. past_due -> payment_succeeded -> active
	next, err = mgr.FireTransition("tenant_fsm", "payment_succeeded")
	if err != nil || next != subscription.StateActive {
		t.Fatalf("expected payment_succeeded transition to succeed, got %s, err=%v", next, err)
	}

	// 4. active -> cancel -> cancelled
	next, err = mgr.FireTransition("tenant_fsm", "cancel")
	if err != nil || next != subscription.StateCancelled {
		t.Fatalf("expected cancel transition to succeed, got %s, err=%v", next, err)
	}

	// 5. cancelled -> reactivate -> active
	next, err = mgr.FireTransition("tenant_fsm", "reactivate")
	if err != nil || next != subscription.StateActive {
		t.Fatalf("expected reactivate transition to succeed, got %s, err=%v", next, err)
	}

	// 6. trial -> trial_expired -> past_due
	next, err = mgr.FireTransition("tenant_trial_exp", "trial_expired")
	if err != nil || next != subscription.StatePastDue {
		t.Fatalf("expected trial_expired transition to succeed, got %s, err=%v", next, err)
	}
}

func TestManager_FireTransition_InvalidScenarios(t *testing.T) {
	mgr := subscription.NewManager()

	// Move to cancelled
	_, _ = mgr.FireTransition("tenant_invalid_t", "cancel")

	// Firing payment_failed from cancelled is invalid
	_, err := mgr.FireTransition("tenant_invalid_t", "payment_failed")
	if err == nil || !errors.Is(err, subscription.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState when firing invalid event from cancelled state")
	}
}

func TestManager_PlanCRUDAndDeepCopy(t *testing.T) {
	mgr := subscription.NewManager()

	// Register custom plan
	plan := subscription.Plan{
		ID:   "enterprise",
		Name: "Enterprise Tier",
		Entitlements: map[string]subscription.Entitlement{
			"total_tokens": {MetricID: "total_tokens", Allowed: true, Quota: 100000000},
		},
	}
	mgr.RegisterPlan(plan)

	// Retrieve plan
	fetched, ok := mgr.GetPlan("enterprise")
	if !ok {
		t.Fatalf("expected enterprise plan to be retrieved")
	}

	// Mutate retrieved map
	fetched.Entitlements["total_tokens"] = subscription.Entitlement{MetricID: "total_tokens", Allowed: false}

	// Stored plan should remain unmodified
	original, _ := mgr.GetPlan("enterprise")
	if !original.Entitlements["total_tokens"].Allowed {
		t.Fatalf("data race protection failed: original plan map was mutated")
	}

	plans := mgr.ListPlans()
	if len(plans) < 3 {
		t.Fatalf("expected at least 3 plans (starter, pro, enterprise), got %d", len(plans))
	}
}

func TestManager_CheckEntitlement_Scenarios(t *testing.T) {
	mgr := subscription.NewManager()

	// 1. Under quota check on default starter plan
	ok, err := mgr.CheckEntitlement("t_ent_1", "total_tokens", 500000)
	if !ok || err != nil {
		t.Fatalf("expected under quota check to pass, got ok=%v, err=%v", ok, err)
	}

	// 2. Over quota check on default starter plan (1M quota)
	ok, err = mgr.CheckEntitlement("t_ent_1", "total_tokens", 1000000)
	if ok || err == nil {
		t.Fatalf("expected quota limit check to fail for 1M tokens")
	}

	// 3. Cancelled tenant entitlement check
	mgr.SetState("t_cancelled", subscription.StateCancelled)
	ok, err = mgr.CheckEntitlement("t_cancelled", "total_tokens", 0)
	if ok || err == nil {
		t.Fatalf("expected entitlement check for cancelled tenant to be denied")
	}

	// 4. Metric with Allowed: false
	mgr.RegisterPlan(subscription.Plan{
		ID: "restricted_plan",
		Entitlements: map[string]subscription.Entitlement{
			"premium_feature": {MetricID: "premium_feature", Allowed: false},
		},
	})
	mgr.CreateContract(subscription.Contract{TenantKey: "t_restricted", PlanID: "restricted_plan", State: subscription.StateActive})
	ok, err = mgr.CheckEntitlement("t_restricted", "premium_feature", 0)
	if ok || err == nil {
		t.Fatalf("expected disallowed metric check to fail")
	}

	// 5. Non-existent metric check (defaults to permitted)
	ok, err = mgr.CheckEntitlement("t_ent_1", "unknown_metric", 999999)
	if !ok || err != nil {
		t.Fatalf("expected unknown metric check to be permitted")
	}
}

func TestManager_StoreHydration(t *testing.T) {
	app := kernel.NewApp(&kernel.Config{})
	mgr := subscription.NewManager()
	mgr.SetApp(app)

	// Direct store registration
	_ = app.Store.RegisterServiceSubscription(kernel.ServiceSubscription{
		SubscriptionID: kernel.MustSubscriptionID("sub_hydrate_1"),
		TenantKey:      kernel.MustTenantKey("t_hydrate"),
		ServiceID:      kernel.ServiceIDAICompletion,
		PlanID:         kernel.PlanID("pro"),
		ChargeType:     "metered",
		Timezone:       "America/Chicago",
		AnchorTime:     time.Now().UTC(),
		Status:         kernel.SubscriptionStatusActive,
	})

	// GetContract should hydrate contract from store
	c, ok := mgr.GetContract("t_hydrate")
	if !ok {
		t.Fatalf("expected contract to be hydrated from store")
	}
	if c.PlanID != "pro" || c.State != subscription.StateActive {
		t.Fatalf("expected hydrated contract plan='pro', state='active', got plan=%s, state=%s", c.PlanID, c.State)
	}
}

func TestManager_TimezoneValidation(t *testing.T) {
	mgr := subscription.NewManager()

	c1 := mgr.CreateContract(subscription.Contract{
		TenantKey: "t_tz_valid",
		PlanID:    "starter",
		Timezone:  "America/New_York",
	})
	if c1.Timezone != "America/New_York" {
		t.Fatalf("expected valid timezone to be kept, got %s", c1.Timezone)
	}

	c2 := mgr.CreateContract(subscription.Contract{
		TenantKey: "t_tz_invalid",
		PlanID:    "starter",
		Timezone:  "Mars/Phobos_Base",
	})
	if c2.Timezone != "UTC" {
		t.Fatalf("expected invalid timezone to fallback to UTC, got %s", c2.Timezone)
	}
}

func TestManager_ConcurrencyDataRaceSafety(t *testing.T) {
	mgr := subscription.NewManager()
	done := make(chan bool)

	for g := 0; g < 10; g++ {
		go func(id int) {
			for i := 0; i < 50; i++ {
				tenant := "tenant_race"
				mgr.RegisterPlan(subscription.Plan{
					ID:   "pro",
					Name: "Pro Tier",
					Entitlements: map[string]subscription.Entitlement{
						"total_tokens": {MetricID: "total_tokens", Allowed: true, Quota: int64(1000 + i)},
					},
				})
				mgr.CreateContract(subscription.Contract{TenantKey: tenant, PlanID: "pro", State: subscription.StateActive})
				_, _ = mgr.GetContract(tenant)
				_ = mgr.GetState(tenant)
				_, _ = mgr.CheckEntitlement(tenant, "total_tokens", int64(i))
				_, _ = mgr.FireTransition(tenant, "payment_failed")
				_, _ = mgr.FireTransition(tenant, "payment_succeeded")
			}
			done <- true
		}(g)
	}

	for g := 0; g < 10; g++ {
		<-done
	}
}
