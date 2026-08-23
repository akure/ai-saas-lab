package subscription_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aisaaslab/internal/kernel"
	"aisaaslab/internal/modules/subscription"
)

func TestHandlers_ListPlans(t *testing.T) {
	app := kernel.NewApp(&kernel.Config{})
	mod := subscription.New()
	_ = mod.Init(app)

	req := httptest.NewRequest("GET", "/v1/subscription/plans", nil)
	w := httptest.NewRecorder()
	app.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var resp map[string][]subscription.Plan
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed decoding list plans response: %v", err)
	}
	if len(resp["plans"]) < 2 {
		t.Fatalf("expected at least 2 default plans (starter, pro), got %d", len(resp["plans"]))
	}
}

func TestHandlers_GetPlan(t *testing.T) {
	app := kernel.NewApp(&kernel.Config{})
	mod := subscription.New()
	_ = mod.Init(app)

	// 1. Existing plan -> 200 OK
	req := httptest.NewRequest("GET", "/v1/subscription/plans/starter", nil)
	w := httptest.NewRecorder()
	app.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for starter plan, got %d", w.Code)
	}

	// 2. Non-existent plan -> 404 Not Found
	reqNotFound := httptest.NewRequest("GET", "/v1/subscription/plans/unknown_plan", nil)
	wNotFound := httptest.NewRecorder()
	app.Mux.ServeHTTP(wNotFound, reqNotFound)

	if wNotFound.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for unknown plan, got %d", wNotFound.Code)
	}
}

func TestHandlers_RegisterPlan(t *testing.T) {
	app := kernel.NewApp(&kernel.Config{})
	mod := subscription.New()
	_ = mod.Init(app)

	// 1. Valid plan payload -> 201 Created
	planReq := subscription.Plan{
		ID:   "enterprise_plus",
		Name: "Enterprise Plus Tier",
		Entitlements: map[string]subscription.Entitlement{
			"total_tokens": {MetricID: "total_tokens", Allowed: true, Quota: 50000000},
		},
	}
	body, _ := json.Marshal(planReq)
	req := httptest.NewRequest("POST", "/v1/subscription/plans", bytes.NewReader(body))
	w := httptest.NewRecorder()
	app.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for plan registration, got %d: %s", w.Code, w.Body.String())
	}

	// 2. Missing plan ID -> 400 Bad Request
	invalidBody, _ := json.Marshal(subscription.Plan{Name: "No ID Plan"})
	reqInvalid := httptest.NewRequest("POST", "/v1/subscription/plans", bytes.NewReader(invalidBody))
	wInvalid := httptest.NewRecorder()
	app.Mux.ServeHTTP(wInvalid, reqInvalid)

	if wInvalid.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for missing plan id, got %d", wInvalid.Code)
	}
}

func TestHandlers_GetSubscription(t *testing.T) {
	app := kernel.NewApp(&kernel.Config{})
	mod := subscription.New()
	_ = mod.Init(app)

	req := httptest.NewRequest("GET", "/v1/subscription/tenant_demo", nil)
	w := httptest.NewRecorder()
	app.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var res map[string]any
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode get subscription response: %v", err)
	}
	if res["tenant_key"] != "tenant_demo" {
		t.Fatalf("expected tenant_key 'tenant_demo', got %v", res["tenant_key"])
	}
	if res["state"] != "trial" {
		t.Fatalf("expected default state 'trial', got %v", res["state"])
	}
	if res["is_usable"] != true {
		t.Fatalf("expected is_usable to be true")
	}
}

func TestHandlers_CreateContract(t *testing.T) {
	app := kernel.NewApp(&kernel.Config{})
	mod := subscription.New()
	_ = mod.Init(app)

	// 1. Valid contract creation -> 201 Created
	cReq := subscription.Contract{TenantKey: "t_created", PlanID: "pro"}
	body, _ := json.Marshal(cReq)
	req := httptest.NewRequest("POST", "/v1/subscription/contracts", bytes.NewReader(body))
	w := httptest.NewRecorder()
	app.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	// 2. Missing plan_id -> 400 Bad Request
	badReq := subscription.Contract{TenantKey: "t_no_plan"}
	badBody, _ := json.Marshal(badReq)
	reqBad := httptest.NewRequest("POST", "/v1/subscription/contracts", bytes.NewReader(badBody))
	wBad := httptest.NewRecorder()
	app.Mux.ServeHTTP(wBad, reqBad)

	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for missing plan_id, got %d", wBad.Code)
	}

	// 3. Unknown plan_id -> 404 Not Found
	missingPlanReq := subscription.Contract{TenantKey: "t_missing_p", PlanID: "non_existent"}
	mBody, _ := json.Marshal(missingPlanReq)
	reqM := httptest.NewRequest("POST", "/v1/subscription/contracts", bytes.NewReader(mBody))
	wM := httptest.NewRecorder()
	app.Mux.ServeHTTP(wM, reqM)

	if wM.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for non-existent plan, got %d", wM.Code)
	}
}

func TestHandlers_FireEvent(t *testing.T) {
	app := kernel.NewApp(&kernel.Config{})
	mod := subscription.New()
	_ = mod.Init(app)

	// 1. Valid transition: trial -> activate -> active
	eReq := map[string]string{"event": "activate"}
	body, _ := json.Marshal(eReq)
	req := httptest.NewRequest("POST", "/v1/subscription/t_event/event", bytes.NewReader(body))
	w := httptest.NewRecorder()
	app.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid event transition, got %d: %s", w.Code, w.Body.String())
	}

	// 2. Invalid state transition -> 409 Conflict
	invReq := map[string]string{"event": "payment_succeeded"}
	invBody, _ := json.Marshal(invReq)
	reqInv := httptest.NewRequest("POST", "/v1/subscription/t_event/event", bytes.NewReader(invBody))
	wInv := httptest.NewRecorder()
	app.Mux.ServeHTTP(wInv, reqInv)

	if wInv.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for invalid event transition from active state, got %d", wInv.Code)
	}

	// 3. Empty event -> 400 Bad Request
	emptyReq := map[string]string{"event": "   "}
	eBody, _ := json.Marshal(emptyReq)
	reqEmpty := httptest.NewRequest("POST", "/v1/subscription/t_event/event", bytes.NewReader(eBody))
	wEmpty := httptest.NewRecorder()
	app.Mux.ServeHTTP(wEmpty, reqEmpty)

	if wEmpty.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for empty event, got %d", wEmpty.Code)
	}
}
