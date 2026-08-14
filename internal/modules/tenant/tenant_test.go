package tenant

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aisaaslab/internal/kernel"
	"aisaaslab/internal/modules/auth"
)

func TestTenantCatalogModule_SecurityAndCRUD(t *testing.T) {
	cfg := &kernel.Config{}
	app := kernel.NewApp(cfg)

	authMod := auth.New()
	tenantMod := New(authMod.Service())

	app.RegisterModule(authMod)
	app.RegisterModule(tenantMod)

	if err := app.InitAll(); err != nil {
		t.Fatalf("failed to init app: %v", err)
	}

	// Create valid tenant key
	validKey, err := authMod.Service().CreateAPIKey(kernel.MaaSPlanGrowth)
	if err != nil {
		t.Fatalf("failed to create api key: %v", err)
	}

	// 1. Unauthenticated request -> expect 401 Unauthorized
	reqUnauth := httptest.NewRequest(http.MethodPost, "/v1/tenant/catalog/services", bytes.NewBufferString(`{"service_id":"ai-writer"}`))
	reqUnauth.Header.Set("Content-Type", "application/json")
	wUnauth := httptest.NewRecorder()
	app.Mux.ServeHTTP(wUnauth, reqUnauth)

	if wUnauth.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for unauthenticated request, got %d", wUnauth.Code)
	}

	// 2. Authenticated Service Registration -> expect 201 Created
	reqSvc := httptest.NewRequest(http.MethodPost, "/v1/tenant/catalog/services", bytes.NewBufferString(`{"service_id":"ai-writer","name":"AI Writer Engine"}`))
	reqSvc.Header.Set("Content-Type", "application/json")
	reqSvc.Header.Set("X-API-Key", validKey)
	wSvc := httptest.NewRecorder()
	app.Mux.ServeHTTP(wSvc, reqSvc)

	if wSvc.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d: %s", wSvc.Code, wSvc.Body.String())
	}

	// 3. Authenticated Metric Registration -> expect 201 Created
	reqMetric := httptest.NewRequest(http.MethodPost, "/v1/tenant/catalog/metrics", bytes.NewBufferString(`{"metric_id":"prompt_tokens","service_id":"ai-writer","unit":"count"}`))
	reqMetric.Header.Set("Content-Type", "application/json")
	reqMetric.Header.Set("X-API-Key", validKey)
	wMetric := httptest.NewRecorder()
	app.Mux.ServeHTTP(wMetric, reqMetric)

	if wMetric.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d: %s", wMetric.Code, wMetric.Body.String())
	}

	// 4. Authenticated Plan Registration -> expect 201 Created
	reqPlan := httptest.NewRequest(http.MethodPost, "/v1/tenant/catalog/plans", bytes.NewBufferString(`{
		"plan_id": "pro-writer-v1",
		"service_id": "ai-writer",
		"name": "Pro Writer Tier",
		"rates": {"prompt_tokens": 0.002},
		"included_quotas": {"prompt_tokens": 100000}
	}`))
	reqPlan.Header.Set("Content-Type", "application/json")
	reqPlan.Header.Set("X-API-Key", validKey)
	wPlan := httptest.NewRecorder()
	app.Mux.ServeHTTP(wPlan, reqPlan)

	if wPlan.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d: %s", wPlan.Code, wPlan.Body.String())
	}

	// 5. GET Overview -> expect 200 OK with all objects
	reqOverview := httptest.NewRequest(http.MethodGet, "/v1/tenant/catalog/overview", nil)
	reqOverview.Header.Set("X-API-Key", validKey)
	wOverview := httptest.NewRecorder()
	app.Mux.ServeHTTP(wOverview, reqOverview)

	if wOverview.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", wOverview.Code)
	}

	var overview map[string]any
	if err := json.Unmarshal(wOverview.Body.Bytes(), &overview); err != nil {
		t.Fatalf("failed to parse overview json: %v", err)
	}

	svcs, ok := overview["services"].([]any)
	if !ok || len(svcs) != 1 {
		t.Errorf("expected 1 service in overview, got %v", overview["services"])
	}
}

func TestTenantCatalogModule_ReferentialIntegrity(t *testing.T) {
	cfg := &kernel.Config{}
	app := kernel.NewApp(cfg)

	authMod := auth.New()
	tenantMod := New(authMod.Service())

	app.RegisterModule(authMod)
	app.RegisterModule(tenantMod)

	_ = app.InitAll()

	validKey, _ := authMod.Service().CreateAPIKey(kernel.MaaSPlanGrowth)

	// Try registering metric for unregistered service -> expect 400 Bad Request
	reqMetric := httptest.NewRequest(http.MethodPost, "/v1/tenant/catalog/metrics", bytes.NewBufferString(`{"metric_id":"prompt_tokens","service_id":"unregistered-service","unit":"count"}`))
	reqMetric.Header.Set("Content-Type", "application/json")
	reqMetric.Header.Set("X-API-Key", validKey)
	wMetric := httptest.NewRecorder()
	app.Mux.ServeHTTP(wMetric, reqMetric)

	// Referential integrity: metric for unregistered service → 422 Unprocessable Entity
	// (structured error: validation failure on service_id, not a generic 400).
	if wMetric.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422 Unprocessable Entity when service missing, got %d: %s", wMetric.Code, wMetric.Body.String())
	}
}
