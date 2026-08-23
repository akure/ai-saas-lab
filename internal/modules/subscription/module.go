package subscription

import (
	"context"
	"time"

	"aisaaslab/internal/kernel"
)

// Module integrates the subscription & entitlement engine into the MaaS platform.
type Module struct {
	manager *Manager
}

func New() *Module {
	return &Module{
		manager: NewManager(),
	}
}

func (m *Module) Name() string { return "subscription" }

func (m *Module) Manager() *Manager {
	return m.manager
}

func (m *Module) Init(app *kernel.App) error {
	if m.manager == nil {
		m.manager = NewManager()
	}
	m.manager.SetApp(app)

	// Register kernel policy: has-active-subscription
	app.RegisterPolicy(kernel.FuncPolicy{
		PolicyName: "has-active-subscription",
		Fn: func(ctx context.Context, subject any) (bool, error) {
			key, ok := subject.(string)
			if !ok {
				return false, nil
			}
			st := m.manager.GetState(key)
			return st.IsUsable(), nil
		},
	})

	// Register kernel policy: under-subscription-quota
	app.RegisterPolicy(kernel.FuncPolicy{
		PolicyName: "under-subscription-quota",
		Fn: func(ctx context.Context, subject any) (bool, error) {
			key, ok := subject.(string)
			if !ok {
				return false, nil
			}
			var usage int64
			if app.Store != nil {
				stmt, found := app.Store.GetServiceUsageStatement(key, string(kernel.ServiceIDAICompletion), time.Now().UTC())
				if found && stmt.Metrics != nil {
					for _, met := range stmt.Metrics {
						if met != nil {
							usage += met.CycleTotal
						}
					}
				}
			}
			return m.manager.CheckEntitlement(key, "total_tokens", usage)
		},
	})

	// Register kernel policy: under-quota (alias for under-subscription-quota)
	app.RegisterPolicy(kernel.FuncPolicy{
		PolicyName: "under-quota",
		Fn: func(ctx context.Context, subject any) (bool, error) {
			key, ok := subject.(string)
			if !ok {
				return false, nil
			}
			var usage int64
			if app.Store != nil {
				stmt, found := app.Store.GetServiceUsageStatement(key, string(kernel.ServiceIDAICompletion), time.Now().UTC())
				if found && stmt.Metrics != nil {
					for _, met := range stmt.Metrics {
						if met != nil {
							usage += met.CycleTotal
						}
					}
				}
			}
			return m.manager.CheckEntitlement(key, "total_tokens", usage)
		},
	})

	// HTTP Routes - Plan, Contract & FSM
	app.Mux.HandleFunc("GET /v1/subscription/plans", m.handleListPlans)
	app.Mux.HandleFunc("POST /v1/subscription/plans", m.handleRegisterPlan)
	app.Mux.HandleFunc("GET /v1/subscription/plans/{id}", m.handleGetPlan)
	app.Mux.HandleFunc("GET /v1/subscription/{key}", m.handleGetSubscription)
	app.Mux.HandleFunc("POST /v1/subscription/{key}/event", m.handleFireEvent)
	app.Mux.HandleFunc("POST /v1/subscription/contracts", m.handleCreateContract)

	return nil
}

func (m *Module) Start(ctx context.Context) error { return nil }
func (m *Module) Stop(ctx context.Context) error  { return nil }
