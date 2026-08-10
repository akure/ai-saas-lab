package billing_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aisaaslab/internal/kernel"
	"aisaaslab/internal/modules/billing"
)

func TestServiceSubscription_CurrentCycleWindow(t *testing.T) {
	anchor, err := time.Parse(time.RFC3339, "2026-08-15T14:30:00Z")
	if err != nil {
		t.Fatalf("failed to parse anchor: %v", err)
	}

	sub := billing.ServiceSubscription{
		SubscriptionID: kernel.MustSubscriptionID("sub_1"),
		TenantKey:      kernel.MustTenantKey("tenant_abc"),
		ServiceID:      kernel.ServiceID("service1"),
		PlanID:         kernel.PlanIDPro,
		ChargeType:     billing.ChargeTypeRecurringMonthly,
		Timezone:       "America/New_York",
		AnchorTime:     anchor,
		Status:         kernel.SubscriptionStatusActive,
	}

	// Target time inside first cycle (e.g. Aug 20)
	target1, _ := time.Parse(time.RFC3339, "2026-08-20T10:00:00Z")
	startUTC1, endUTC1 := sub.CurrentCycleWindow(target1)

	if startUTC1.Format(time.RFC3339) != "2026-08-15T14:30:00Z" {
		t.Errorf("expected start %s, got %s", "2026-08-15T14:30:00Z", startUTC1.Format(time.RFC3339))
	}
	if endUTC1.Format(time.RFC3339) != "2026-09-15T14:30:00Z" {
		t.Errorf("expected end %s, got %s", "2026-09-15T14:30:00Z", endUTC1.Format(time.RFC3339))
	}
}

func TestStore_MultiServiceBillingAggregation(t *testing.T) {
	store := kernel.NewStore()

	anchor1, _ := time.Parse(time.RFC3339, "2026-08-01T00:00:00Z")
	anchor2, _ := time.Parse(time.RFC3339, "2026-08-05T00:00:00Z")

	// Service 1 subscription
	_ = store.RegisterServiceSubscription(billing.ServiceSubscription{
		SubscriptionID: kernel.MustSubscriptionID("sub_ai"),
		TenantKey:      kernel.MustTenantKey("tenant_1"),
		ServiceID:      kernel.ServiceIDAICompletion,
		PlanID:         kernel.PlanIDPro,
		ChargeType:     billing.ChargeTypeMetered,
		Timezone:       "UTC",
		AnchorTime:     anchor1,
		Status:         kernel.SubscriptionStatusActive,
	})

	// Service 2 subscription
	_ = store.RegisterServiceSubscription(billing.ServiceSubscription{
		SubscriptionID: kernel.MustSubscriptionID("sub_storage"),
		TenantKey:      kernel.MustTenantKey("tenant_1"),
		ServiceID:      kernel.ServiceIDStorage,
		PlanID:         kernel.PlanID("storage-100gb"),
		ChargeType:     billing.ChargeTypeRecurringMonthly,
		Timezone:       "Asia/Tokyo",
		AnchorTime:     anchor2,
		Status:         kernel.SubscriptionStatusActive,
	})

	// Record events
	evtTime1, _ := time.Parse(time.RFC3339, "2026-08-10T15:00:00Z")
	evtTime2, _ := time.Parse(time.RFC3339, "2026-08-18T09:00:00Z")

	_ = store.RecordMeteringEvent(billing.MeteringEvent{
		EventID:   "evt_1",
		TenantKey: kernel.MustTenantKey("tenant_1"),
		ServiceID: kernel.ServiceIDAICompletion,
		MetricID:  kernel.MetricIDTotalTokens,
		Unit:      "tokens",
		Quantity:  1000,
		Timestamp: evtTime1,
	})

	_ = store.RecordMeteringEvent(billing.MeteringEvent{
		EventID:   "evt_2",
		TenantKey: kernel.MustTenantKey("tenant_1"),
		ServiceID: kernel.ServiceIDStorage,
		MetricID:  kernel.MetricID("gb_stored"),
		Unit:      "gigabytes",
		Quantity:  50,
		Timestamp: evtTime2,
	})

	targetTime, _ := time.Parse(time.RFC3339, "2026-08-20T12:00:00Z")

	// Get Service Statement for AI completion
	aiStmt, ok := store.GetServiceBillingStatement("tenant_1", "ai-completion", targetTime)
	if !ok {
		t.Fatalf("expected statement for ai-completion")
	}
	if aiStmt.Metrics[kernel.MetricIDTotalTokens] == nil || aiStmt.Metrics[kernel.MetricIDTotalTokens].CycleTotal != 1000 {
		t.Errorf("expected 1000 total_tokens for ai-completion, got %+v", aiStmt.Metrics[kernel.MetricIDTotalTokens])
	}

	// Get Overview
	overview := store.GetTenantBillingOverview("tenant_1", targetTime)
	if len(overview.Statements) != 2 {
		t.Errorf("expected 2 service statements, got %d", len(overview.Statements))
	}
}

func TestBillingModule_HTTPRoutes(t *testing.T) {
	app := kernel.NewApp(&kernel.Config{DailyTokenQuota: 10000})
	bModule := billing.New()
	if err := bModule.Init(app); err != nil {
		t.Fatalf("failed to init billing module: %v", err)
	}

	// 1. POST /v1/billing/subscriptions
	subReq := billing.ServiceSubscription{
		SubscriptionID: kernel.MustSubscriptionID("sub_test_service1"),
		TenantKey:      kernel.MustTenantKey("key_test"),
		ServiceID:      kernel.ServiceID("service1"),
		PlanID:         kernel.PlanID("plan_test"),
		ChargeType:     billing.ChargeTypeMetered,
		Timezone:       "UTC",
		AnchorTime:     time.Now().UTC(),
		Status:         kernel.SubscriptionStatusActive,
	}
	body, _ := json.Marshal(subReq)
	req := httptest.NewRequest("POST", "/v1/billing/subscriptions", bytes.NewReader(body))
	w := httptest.NewRecorder()
	app.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected HTTP 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	// 2. POST /v1/billing/events
	evtReq := billing.MeteringEvent{
		EventID:   "evt_test_100",
		TenantKey: kernel.MustTenantKey("key_test"),
		ServiceID: kernel.ServiceID("service1"),
		MetricID:  kernel.MetricID("api_calls"),
		Unit:      "requests",
		Quantity:  5,
		Timestamp: time.Now().UTC(),
	}
	bodyEvt, _ := json.Marshal(evtReq)
	reqEvt := httptest.NewRequest("POST", "/v1/billing/events", bytes.NewReader(bodyEvt))
	wEvt := httptest.NewRecorder()
	app.Mux.ServeHTTP(wEvt, reqEvt)

	if wEvt.Code != http.StatusAccepted {
		t.Fatalf("expected HTTP 202 Accepted, got %d: %s", wEvt.Code, wEvt.Body.String())
	}

	// 3. GET /v1/billing/key_test/statement/service1
	reqStmt := httptest.NewRequest("GET", "/v1/billing/key_test/statement/service1", nil)
	wStmt := httptest.NewRecorder()
	app.Mux.ServeHTTP(wStmt, reqStmt)

	if wStmt.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 OK, got %d: %s", wStmt.Code, wStmt.Body.String())
	}

	var stmt billing.ServiceBillingStatement
	if err := json.Unmarshal(wStmt.Body.Bytes(), &stmt); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if stmt.Metrics[kernel.MetricID("api_calls")] == nil || stmt.Metrics[kernel.MetricID("api_calls")].CycleTotal != 5 {
		t.Errorf("expected cycle total 5 for api_calls, got %+v", stmt.Metrics[kernel.MetricID("api_calls")])
	}

	// 4. GET /v1/billing/key_test/overview
	reqOvr := httptest.NewRequest("GET", "/v1/billing/key_test/overview", nil)
	wOvr := httptest.NewRecorder()
	app.Mux.ServeHTTP(wOvr, reqOvr)

	if wOvr.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 OK, got %d: %s", wOvr.Code, wOvr.Body.String())
	}
	var ovr billing.TenantBillingOverview
	if err := json.Unmarshal(wOvr.Body.Bytes(), &ovr); err != nil {
		t.Fatalf("failed to decode overview response: %v", err)
	}
	if len(ovr.Statements) != 1 {
		t.Errorf("expected 1 statement in overview, got %d", len(ovr.Statements))
	}
}
