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
}

func TestSubscriptionModule_HTTPRoutes(t *testing.T) {
	app := kernel.NewApp(&kernel.Config{})
	mod := subscription.New()
	if err := mod.Init(app); err != nil {
		t.Fatalf("failed to init subscription module: %v", err)
	}

	// 1. POST /v1/subscription/contracts
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

	// 2. POST /v1/subscription/tenant_http/event
	eReq := map[string]string{"event": "activate"}
	eBody, _ := json.Marshal(eReq)
	reqEvt := httptest.NewRequest("POST", "/v1/subscription/tenant_http/event", bytes.NewReader(eBody))
	wEvt := httptest.NewRecorder()
	app.Mux.ServeHTTP(wEvt, reqEvt)

	if wEvt.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 OK, got %d: %s", wEvt.Code, wEvt.Body.String())
	}

	// 3. GET /v1/subscription/tenant_http
	reqGet := httptest.NewRequest("GET", "/v1/subscription/tenant_http", nil)
	wGet := httptest.NewRecorder()
	app.Mux.ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 OK, got %d: %s", wGet.Code, wGet.Body.String())
	}
	var res subscription.Contract
	if err := json.Unmarshal(wGet.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal contract response: %v", err)
	}
	if res.State != subscription.StateActive {
		t.Fatalf("expected active state in contract, got %s", res.State)
	}
}
