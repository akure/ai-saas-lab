package subscription_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"aisaaslab/internal/kernel"
	"aisaaslab/internal/modules/subscription"
)

func TestMiddleware_RequireActiveSubscription(t *testing.T) {
	app := kernel.NewApp(&kernel.Config{})
	mod := subscription.New()
	_ = mod.Init(app)

	activeMW := subscription.RequireActiveSubscription(app, mod.Manager())
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// 1. Missing tenant key -> 400 Bad Request
	reqMissing := httptest.NewRequest("GET", "/protected", nil)
	wMissing := httptest.NewRecorder()
	activeMW(dummyHandler).ServeHTTP(wMissing, reqMissing)

	if wMissing.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for missing key, got %d", wMissing.Code)
	}

	// 2. Active key via X-Tenant-Key Header -> 200 OK
	mod.Manager().CreateContract(subscription.Contract{TenantKey: "t_header", PlanID: "starter", State: subscription.StateActive})
	reqHeader := httptest.NewRequest("GET", "/protected", nil)
	reqHeader.Header.Set("X-Tenant-Key", "t_header")
	wHeader := httptest.NewRecorder()
	activeMW(dummyHandler).ServeHTTP(wHeader, reqHeader)

	if wHeader.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for active tenant via header, got %d: %s", wHeader.Code, wHeader.Body.String())
	}

	// 3. Active key via Context -> 200 OK
	reqCtx := httptest.NewRequest("GET", "/protected", nil)
	ctx := context.WithValue(reqCtx.Context(), "tenant_key", "t_header")
	reqCtx = reqCtx.WithContext(ctx)
	wCtx := httptest.NewRecorder()
	activeMW(dummyHandler).ServeHTTP(wCtx, reqCtx)

	if wCtx.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for active tenant via context, got %d", wCtx.Code)
	}

	// 4. Cancelled subscription -> 403 Forbidden
	mod.Manager().SetState("t_cancelled_mw", subscription.StateCancelled)
	reqCancelled := httptest.NewRequest("GET", "/protected?tenant_key=t_cancelled_mw", nil)
	wCancelled := httptest.NewRecorder()
	activeMW(dummyHandler).ServeHTTP(wCancelled, reqCancelled)

	if wCancelled.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for cancelled tenant, got %d", wCancelled.Code)
	}
}

func TestMiddleware_RequireEntitlement(t *testing.T) {
	app := kernel.NewApp(&kernel.Config{})
	mod := subscription.New()
	_ = mod.Init(app)

	entitlementMW := subscription.RequireEntitlement(app, mod.Manager(), "total_tokens")
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// 1. Missing key -> 400 Bad Request
	reqNoKey := httptest.NewRequest("POST", "/api/chat", nil)
	wNoKey := httptest.NewRecorder()
	entitlementMW(dummyHandler).ServeHTTP(wNoKey, reqNoKey)

	if wNoKey.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for missing key, got %d", wNoKey.Code)
	}

	// 2. Under quota tenant -> 200 OK
	mod.Manager().CreateContract(subscription.Contract{TenantKey: "t_ent_mw", PlanID: "starter", State: subscription.StateActive})
	reqUnder := httptest.NewRequest("POST", "/api/chat", nil)
	reqUnder.Header.Set("X-Tenant-Key", "t_ent_mw")
	wUnder := httptest.NewRecorder()
	entitlementMW(dummyHandler).ServeHTTP(wUnder, reqUnder)

	if wUnder.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for under quota tenant, got %d: %s", wUnder.Code, wUnder.Body.String())
	}

	// 3. Fallback manager check for cancelled tenant -> 402 Payment Required
	mod.Manager().SetState("t_cancelled_ent", subscription.StateCancelled)
	entitlementManagerMW := subscription.RequireEntitlement(nil, mod.Manager(), "total_tokens")
	reqCancelled := httptest.NewRequest("POST", "/api/chat?api_key=t_cancelled_ent", nil)
	wCancelled := httptest.NewRecorder()
	entitlementManagerMW(dummyHandler).ServeHTTP(wCancelled, reqCancelled)

	if wCancelled.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402 Payment Required for cancelled tenant entitlement check, got %d", wCancelled.Code)
	}
}
