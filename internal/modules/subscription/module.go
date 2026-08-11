package subscription

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

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

	// HTTP Routes
	app.Mux.HandleFunc("GET /v1/subscription/{key}", m.handleGetSubscription)
	app.Mux.HandleFunc("POST /v1/subscription/{key}/event", m.handleFireEvent)
	app.Mux.HandleFunc("POST /v1/subscription/contracts", m.handleCreateContract)

	return nil
}

func (m *Module) Start(ctx context.Context) error { return nil }
func (m *Module) Stop(ctx context.Context) error  { return nil }

func (m *Module) handleGetSubscription(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	contract, ok := m.manager.GetContract(key)
	w.Header().Set("Content-Type", "application/json")
	if !ok {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tenant_key": key,
			"state":      string(StateTrial),
			"status":     "trial (default)",
		})
		return
	}
	_ = json.NewEncoder(w).Encode(contract)
}

type fireEventReq struct {
	Event string `json:"event"`
}

func (m *Module) handleFireEvent(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	var body fireEventReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json request body", http.StatusBadRequest)
		return
	}

	current := m.manager.GetState(key)
	next, err := m.manager.FireTransition(key, strings.TrimSpace(body.Event))
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"tenant_key": key,
		"from":       string(current),
		"event":      body.Event,
		"to":         string(next),
	})
}

func (m *Module) handleCreateContract(w http.ResponseWriter, r *http.Request) {
	var c Contract
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "invalid contract request body", http.StatusBadRequest)
		return
	}
	if c.TenantKey == "" || c.PlanID == "" {
		http.Error(w, "tenant_key and plan_id are required", http.StatusBadRequest)
		return
	}
	if c.SubscriptionID == "" {
		c.SubscriptionID = "sub_" + c.PlanID + "_" + c.TenantKey
	}

	res := m.manager.CreateContract(c)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(res)
}
