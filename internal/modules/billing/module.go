package billing

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"aisaaslab/internal/kernel"
	"aisaaslab/internal/modules/completion"
)

// ---------------------------------------------------------------------------
// Billing Module (App Integration Adapter)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Billing Module
// ---------------------------------------------------------------------------

type Module struct {
	app           *kernel.App
	fsm           *kernel.StateMachine[kernel.State, string, string]
	mu            sync.RWMutex
	subscriptions map[string]kernel.State // tenant key -> FSM subscription state
}

func New() *Module {
	return &Module{
		subscriptions: make(map[string]kernel.State),
	}
}

func (m *Module) Name() string { return "billing" }

func (m *Module) SetSubscriptionState(key string, st kernel.State) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscriptions[key] = st
}

func (m *Module) SubscriptionState(key string) kernel.State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if st, ok := m.subscriptions[key]; ok {
		return st
	}
	return "unknown"
}

func (m *Module) Init(app *kernel.App) error {
	m.app = app
	if m.subscriptions == nil {
		m.subscriptions = make(map[string]kernel.State)
	}

	// --- 1. Event Subscription: Generic Metering Events ---
	app.Events.Subscribe("metering.event", func(payload any) {
		evt, ok := payload.(*MeteringEvent)
		if !ok || evt == nil {
			return
		}
		if err := evt.Validate(); err != nil {
			return
		}
		app.Store.RecordMeteringEvent(*evt)
	})

	// --- 2. Event Subscription: Converts Completion Usage to MeteringEvent ---
	app.Events.Subscribe("usage.recorded", func(payload any) {
		rec, ok := payload.(*completion.UsageRecord)
		if !ok || rec == nil {
			return
		}

		// Auto-register default subscription if none exists for ai-completion
		subs := app.Store.GetServiceSubscriptions(rec.APIKey)
		hasAISub := false
		for _, s := range subs {
			if s.ServiceID == "ai-completion" {
				hasAISub = true
				break
			}
		}
		if !hasAISub {
			_ = app.Store.RegisterServiceSubscription(ServiceSubscription{
				SubscriptionID: kernel.MustSubscriptionID("sub_ai_" + rec.APIKey),
				TenantKey:      kernel.MustTenantKey(rec.APIKey),
				ServiceID:      kernel.ServiceIDAICompletion,
				PlanID:         kernel.PlanID("default-ai-plan"),
				ChargeType:     ChargeTypeMetered,
				Timezone:       "UTC",
				AnchorTime:     time.Now().UTC(),
				Status:         kernel.SubscriptionStatusActive,
			})
		}

		_ = app.Store.RecordMeteringEvent(MeteringEvent{
			EventID:   "evt_completion_" + time.Now().UTC().Format("20060102150405.000"),
			TenantKey: kernel.MustTenantKey(rec.APIKey),
			ServiceID: kernel.ServiceIDAICompletion,
			MetricID:  kernel.MetricIDTotalTokens,
			Unit:      "tokens",
			Quantity:  int64(rec.Tokens),
			Timestamp: time.Now().UTC(),
		})
	})

	// --- Policy: quota check other modules require by name. ---
	app.RegisterPolicy(kernel.FuncPolicy{
		PolicyName: "under-quota",
		Fn: func(ctx context.Context, subject any) (bool, error) {
			key, ok := subject.(string)
			if !ok {
				return false, nil
			}
			tokensUsed := getTenantTokenUsage(app.Store, key)
			return tokensUsed < app.Config.DailyTokenQuota, nil
		},
	})

	// --- Subscription state machine: trial -> active -> past_due -> cancelled ---
	m.fsm = kernel.NewStateMachine[kernel.State, string, string]().
		AddTransition(kernel.NewTransition[kernel.State, string, string]("trial", "activate", "active")).
		AddTransition(kernel.NewTransition[kernel.State, string, string]("active", "payment_failed", "past_due")).
		AddTransition(kernel.NewTransition[kernel.State, string, string]("past_due", "payment_succeeded", "active")).
		AddTransition(kernel.NewTransition[kernel.State, string, string]("past_due", "cancel", "cancelled")).
		AddTransition(kernel.NewTransition[kernel.State, string, string]("active", "cancel", "cancelled")).
		AddTransition(kernel.NewTransition[kernel.State, string, string]("trial", "cancel", "cancelled"))

	// HTTP Routes
	app.Mux.HandleFunc("GET /v1/billing/{key}/overview", m.handleGetOverview)
	app.Mux.HandleFunc("GET /v1/billing/{key}/statement/{service}", m.handleGetServiceStatement)
	app.Mux.HandleFunc("POST /v1/billing/subscriptions", m.handleRegisterSubscription)
	app.Mux.HandleFunc("POST /v1/billing/events", m.handleIngestEvent)
	app.Mux.HandleFunc("GET /v1/billing/storage/health", m.handleStorageHealth)

	// Backward compatible & FSM routes
	app.Mux.HandleFunc("GET /v1/usage/{key}", m.handleGetUsage)

	return nil
}

func (m *Module) handleGetOverview(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	overview := m.app.Store.GetTenantUsageOverview(key, time.Now().UTC())
	st := m.SubscriptionState(key)
	resp := map[string]any{
		"tenant_key":         overview.TenantKey.String(),
		"subscription_state": string(st),
		"statements":        overview.Statements,
		"generated_at":       overview.GeneratedAt,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (m *Module) handleGetServiceStatement(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	serviceID := r.PathValue("service")
	stmt, ok := m.app.Store.GetServiceUsageStatement(key, serviceID, time.Now().UTC())
	if !ok {
		http.Error(w, "service subscription not found for tenant", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stmt)
}

func (m *Module) handleRegisterSubscription(w http.ResponseWriter, r *http.Request) {
	var sub ServiceSubscription
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		http.Error(w, "invalid subscription request body", http.StatusBadRequest)
		return
	}
	if sub.TenantKey.IsZero() || sub.ServiceID.IsZero() {
		http.Error(w, "tenant_key and service_id are required", http.StatusBadRequest)
		return
	}
	if sub.SubscriptionID.IsZero() {
		sub.SubscriptionID = kernel.MustSubscriptionID("sub_" + sub.ServiceID.String() + "_" + sub.TenantKey.String())
	}
	if err := m.app.Store.RegisterServiceSubscription(sub); err != nil {
		http.Error(w, "failed to register subscription: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"subscription_id": sub.SubscriptionID.String(),
		"status":          "registered",
	})
}

func (m *Module) handleIngestEvent(w http.ResponseWriter, r *http.Request) {
	var evt MeteringEvent
	if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
		http.Error(w, "invalid metering event request body", http.StatusBadRequest)
		return
	}
	if err := evt.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	m.app.Store.RecordMeteringEvent(evt)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "accepted",
		"event_id": evt.EventID,
	})
}

func (m *Module) handleGetUsage(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	overview := m.app.Store.GetTenantUsageOverview(key, time.Now().UTC())
	tokensUsed := getTenantTokenUsage(m.app.Store, key)
	resp := map[string]any{
		"api_key":            key,
		"tokens_used":        tokensUsed,
		"daily_quota":        m.app.Config.DailyTokenQuota,
		"subscription_state": string(m.SubscriptionState(key)),
		"overview":           overview,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func getTenantTokenUsage(store *kernel.Store, tenantKey string) int {
	stmt, ok := store.GetServiceUsageStatement(tenantKey, string(kernel.ServiceIDAICompletion), time.Now().UTC())
	if !ok || stmt.Metrics == nil {
		return 0
	}
	var total int
	for _, m := range stmt.Metrics {
		if m != nil {
			total += int(m.CycleTotal)
		}
	}
	return total
}

type fireEventReq struct {
	Event string `json:"event"`
}

func (m *Module) handleFireEvent(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	var body fireEventReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request body", http.StatusBadRequest)
		return
	}

	current := m.SubscriptionState(key)
	if current == "unknown" {
		current = "trial"
	}

	next, err := m.fsm.Fire(r.Context(), current, strings.TrimSpace(body.Event), key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	m.SetSubscriptionState(key, next)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"api_key": key,
		"from":    string(current),
		"event":   body.Event,
		"to":      string(next),
	})
}

func (m *Module) Start(ctx context.Context) error { return nil }
func (m *Module) Stop(ctx context.Context) error  { return nil }

func (m *Module) handleStorageHealth(w http.ResponseWriter, r *http.Request) {
	chain := m.app.Store.MeteringChainRef()
	if chain == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "no metering chain configured"})
		return
	}

	type walInfo struct {
		Enabled       bool   `json:"enabled"`
		Depth         int    `json:"depth"`
		ActiveSegment string `json:"active_segment"`
		TotalSegments int    `json:"total_segments"`
	}

	type dedupInfo struct {
		TrackedEvents int    `json:"tracked_events"`
		Retention     string `json:"retention"`
	}

	resp := map[string]any{
		"backends": chain.HealthReport(),
		"wal": walInfo{
			Enabled:       chain.WALEnabled(),
			Depth:         chain.WALDepth(),
			ActiveSegment: chain.WALActiveSegment(),
			TotalSegments: chain.WALTotalSegments(),
		},
		"dedup": dedupInfo{
			TrackedEvents: chain.DedupTrackedEvents(),
			Retention:     chain.DedupRetention().String(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
