package subscription_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aisaaslab/internal/kernel"
	"aisaaslab/internal/modules/subscription"
)

func TestSubscriptionManager_FSMTransitions(t *testing.T) {
	mgr := subscription.NewManager()

	// Initial state defaults to StateTrial
	if st := mgr.GetState("tenant_1"); st != subscription.StateTrial {
		t.Fatalf("expected initial state trial, got %s", st)
	}

	// trial -> activate -> active
	next, err := mgr.FireTransition("tenant_1", "activate")
	if err != nil {
		t.Fatalf("expected activate transition to succeed, got %v", err)
	}
	if next != subscription.StateActive {
		t.Fatalf("expected active state, got %s", next)
	}

	// active -> payment_failed -> past_due
	next, err = mgr.FireTransition("tenant_1", "payment_failed")
	if err != nil {
		t.Fatalf("expected payment_failed transition to succeed, got %v", err)
	}
	if next != subscription.StatePastDue {
		t.Fatalf("expected past_due state, got %s", next)
	}

	// past_due -> payment_succeeded -> active
	next, err = mgr.FireTransition("tenant_1", "payment_succeeded")
	if err != nil {
		t.Fatalf("expected payment_succeeded transition to succeed, got %v", err)
	}
	if next != subscription.StateActive {
		t.Fatalf("expected active state, got %s", next)
	}

	// active -> cancel -> cancelled
	next, err = mgr.FireTransition("tenant_1", "cancel")
	if err != nil {
		t.Fatalf("expected cancel transition to succeed, got %v", err)
	}
	if next != subscription.StateCancelled {
		t.Fatalf("expected cancelled state, got %s", next)
	}

	// cancelled -> reactivate -> active
	next, err = mgr.FireTransition("tenant_1", "reactivate")
	if err != nil {
		t.Fatalf("expected reactivate transition to succeed, got %v", err)
	}
	if next != subscription.StateActive {
		t.Fatalf("expected active state, got %s", next)
	}

	// trial -> trial_expired -> past_due
	next, err = mgr.FireTransition("tenant_trial", "trial_expired")
	if err != nil {
		t.Fatalf("expected trial_expired transition to succeed, got %v", err)
	}
	if next != subscription.StatePastDue {
		t.Fatalf("expected past_due state, got %s", next)
	}
}

func TestSubscriptionManager_PlanCRUDAndRaceSafety(t *testing.T) {
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

	// Fetch plan and verify deep copy
	fetched, ok := mgr.GetPlan("enterprise")
	if !ok {
		t.Fatalf("expected enterprise plan to be found")
	}
	fetched.Entitlements["total_tokens"] = subscription.Entitlement{MetricID: "total_tokens", Allowed: false}

	// Internal stored plan should remain unmodified
	original, _ := mgr.GetPlan("enterprise")
	if !original.Entitlements["total_tokens"].Allowed {
		t.Fatalf("data race protection failed: original plan map was mutated")
	}

	// Concurrent read/write safety check
	done := make(chan bool)
	go func() {
		for i := 0; i < 100; i++ {
			mgr.RegisterPlan(subscription.Plan{
				ID:   "enterprise",
				Name: "Enterprise Tier Modified",
				Entitlements: map[string]subscription.Entitlement{
					"total_tokens": {MetricID: "total_tokens", Allowed: true, Quota: int64(100000000 + i)},
				},
			})
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_, _ = mgr.GetPlan("enterprise")
			_ = mgr.ListPlans()
		}
		done <- true
	}()

	<-done
	<-done
}

func TestSubscriptionManager_CheckEntitlement(t *testing.T) {
	mgr := subscription.NewManager()

	// Tenant 1 on default Starter plan (1M token quota)
	ok, err := mgr.CheckEntitlement("tenant_starter", "total_tokens", 500000)
	if !ok || err != nil {
		t.Fatalf("expected under quota check to pass, got ok=%v, err=%v", ok, err)
	}

	ok, err = mgr.CheckEntitlement("tenant_starter", "total_tokens", 1000000)
	if ok || err == nil {
		t.Fatalf("expected quota exceeded check to fail")
	}

	// Cancelled tenant should be denied entitlement
	_, _ = mgr.FireTransition("tenant_cancelled", "cancel")
	ok, err = mgr.CheckEntitlement("tenant_cancelled", "total_tokens", 0)
	if ok || err == nil {
		t.Fatalf("expected cancelled tenant entitlement to be denied")
	}

	// Plan with disallowed metric
	mgr.RegisterPlan(subscription.Plan{
		ID: "restricted",
		Entitlements: map[string]subscription.Entitlement{
			"gpt4": {MetricID: "gpt4", Allowed: false},
		},
	})
	mgr.CreateContract(subscription.Contract{TenantKey: "tenant_restricted", PlanID: "restricted", State: subscription.StateActive})
	ok, err = mgr.CheckEntitlement("tenant_restricted", "gpt4", 0)
	if ok || err == nil {
		t.Fatalf("expected disallowed metric check to fail")
	}
}

func TestSubscriptionManager_StoreHydrationAndValidation(t *testing.T) {
	app := kernel.NewApp(&kernel.Config{})
	mgr := subscription.NewManager()
	mgr.SetApp(app)

	// Register subscription in store directly
	_ = app.Store.RegisterServiceSubscription(kernel.ServiceSubscription{
		SubscriptionID: kernel.MustSubscriptionID("sub_store_1"),
		TenantKey:      kernel.MustTenantKey("tenant_store"),
		ServiceID:      kernel.ServiceIDAICompletion,
		PlanID:         kernel.PlanID("pro"),
		ChargeType:     "metered",
		Timezone:       "America/New_York",
		AnchorTime:     time.Now().UTC(),
		Status:         kernel.SubscriptionStatusActive,
	})

	// GetContract should hydrate from store
	contract, ok := mgr.GetContract("tenant_store")
	if !ok {
		t.Fatalf("expected contract to be hydrated from store")
	}
	if contract.PlanID != "pro" {
		t.Fatalf("expected plan_id 'pro', got %s", contract.PlanID)
	}
	if contract.State != subscription.StateActive {
		t.Fatalf("expected state 'active', got %s", contract.State)
	}

	// Timezone validation test
	invalidTZContract := mgr.CreateContract(subscription.Contract{
		TenantKey: "tenant_tz",
		PlanID:    "starter",
		Timezone:  "Invalid/Timezone_Name",
	})
	if invalidTZContract.Timezone != "UTC" {
		t.Fatalf("expected invalid timezone to fallback to UTC, got %s", invalidTZContract.Timezone)
	}
}

func TestSubscriptionModule_Middleware(t *testing.T) {
	app := kernel.NewApp(&kernel.Config{})
	mod := subscription.New()
	if err := mod.Init(app); err != nil {
		t.Fatalf("failed to init module: %v", err)
	}

	activeMW := subscription.RequireActiveSubscription(app, mod.Manager())
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// 1. Missing tenant key -> expect 400
	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	activeMW(dummyHandler).ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for missing key, got %d", w.Code)
	}

	// 2. Active subscription -> expect 200
	mod.Manager().CreateContract(subscription.Contract{TenantKey: "t_active", PlanID: "starter", State: subscription.StateActive})
	reqActive := httptest.NewRequest("GET", "/protected", nil)
	reqActive.Header.Set("X-Tenant-Key", "t_active")
	wActive := httptest.NewRecorder()
	activeMW(dummyHandler).ServeHTTP(wActive, reqActive)
	if wActive.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for active tenant, got %d: %s", wActive.Code, wActive.Body.String())
	}
}

func TestSubscriptionModule_HTTPRoutes(t *testing.T) {
	app := kernel.NewApp(&kernel.Config{})
	mod := subscription.New()
	if err := mod.Init(app); err != nil {
		t.Fatalf("failed to init subscription module: %v", err)
	}

	// 1. POST /v1/subscription/plans
	pReq := subscription.Plan{
		ID:   "custom_plan",
		Name: "Custom Tier",
		Entitlements: map[string]subscription.Entitlement{
			"total_tokens": {MetricID: "total_tokens", Allowed: true, Quota: 5000000},
		},
	}
	pBody, _ := json.Marshal(pReq)
	reqPlan := httptest.NewRequest("POST", "/v1/subscription/plans", bytes.NewReader(pBody))
	wPlan := httptest.NewRecorder()
	app.Mux.ServeHTTP(wPlan, reqPlan)
	if wPlan.Code != http.StatusCreated {
		t.Fatalf("expected HTTP 201 Created for plan registration, got %d: %s", wPlan.Code, wPlan.Body.String())
	}

	// 2. GET /v1/subscription/plans
	reqList := httptest.NewRequest("GET", "/v1/subscription/plans", nil)
	wList := httptest.NewRecorder()
	app.Mux.ServeHTTP(wList, reqList)
	if wList.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 OK for list plans, got %d", wList.Code)
	}

	// 3. POST /v1/subscription/contracts
	cReq := subscription.Contract{
		TenantKey: "tenant_http",
		PlanID:    "pro",
	}
	body, _ := json.Marshal(cReq)
	req := httptest.NewRequest("POST", "/v1/subscription/contracts", bytes.NewReader(body))
	w := httptest.NewRecorder()
	app.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected HTTP 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	// 4. POST /v1/subscription/tenant_http/event
	eReq := map[string]string{"event": "activate"}
	eBody, _ := json.Marshal(eReq)
	reqEvt := httptest.NewRequest("POST", "/v1/subscription/tenant_http/event", bytes.NewReader(eBody))
	wEvt := httptest.NewRecorder()
	app.Mux.ServeHTTP(wEvt, reqEvt)

	if wEvt.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 OK, got %d: %s", wEvt.Code, wEvt.Body.String())
	}

	// 5. GET /v1/subscription/tenant_http
	reqGet := httptest.NewRequest("GET", "/v1/subscription/tenant_http", nil)
	wGet := httptest.NewRecorder()
	app.Mux.ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 OK, got %d: %s", wGet.Code, wGet.Body.String())
	}
	var res map[string]any
	if err := json.Unmarshal(wGet.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal contract response: %v", err)
	}
	if res["state"] != string(subscription.StateActive) {
		t.Fatalf("expected active state in contract, got %v", res["state"])
	}
	if res["is_usable"] != true {
		t.Fatalf("expected is_usable to be true")
	}

	// 6. Test invalid plan creation -> expect 404
	invalidReq := subscription.Contract{TenantKey: "t_invalid", PlanID: "non_existent_plan"}
	invBody, _ := json.Marshal(invalidReq)
	reqInv := httptest.NewRequest("POST", "/v1/subscription/contracts", bytes.NewReader(invBody))
	wInv := httptest.NewRecorder()
	app.Mux.ServeHTTP(wInv, reqInv)
	if wInv.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for invalid plan contract, got %d", wInv.Code)
	}
}
