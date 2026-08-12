package tenant

import (
	"context"
	"net/http"

	"aisaaslab/internal/kernel"
	"aisaaslab/internal/modules/auth"
)

// Module registers the tenant self-service catalog module into kernel.App.
type Module struct {
	authService *auth.Service
	service     *Service
	handlers    *Handlers
}

func New(authService *auth.Service) *Module {
	return &Module{
		authService: authService,
	}
}

func (m *Module) Name() string { return "tenant" }

func (m *Module) Service() *Service {
	return m.service
}

func (m *Module) Init(app *kernel.App) error {
	if m.service == nil {
		m.service = NewService(app.TenantCatalog, app.Events)
	}
	m.handlers = NewHandlers(m.service)

	// Auth-protected route wrappers using TenantAuthMiddleware
	authMW := func(h func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
		return TenantAuthMiddleware(app, m.authService, h)
	}

	app.Mux.HandleFunc("POST /v1/tenant/catalog/services", authMW(m.handlers.HandleServices))
	app.Mux.HandleFunc("GET /v1/tenant/catalog/services", authMW(m.handlers.HandleServices))

	app.Mux.HandleFunc("POST /v1/tenant/catalog/metrics", authMW(m.handlers.HandleMetrics))
	app.Mux.HandleFunc("GET /v1/tenant/catalog/metrics", authMW(m.handlers.HandleMetrics))

	app.Mux.HandleFunc("POST /v1/tenant/catalog/plans", authMW(m.handlers.HandlePlans))
	app.Mux.HandleFunc("GET /v1/tenant/catalog/plans", authMW(m.handlers.HandlePlans))

	app.Mux.HandleFunc("GET /v1/tenant/catalog/overview", authMW(m.handlers.HandleOverview))

	return nil
}

func (m *Module) Start(ctx context.Context) error { return nil }
func (m *Module) Stop(ctx context.Context) error  { return nil }
