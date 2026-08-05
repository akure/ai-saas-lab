package billing

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"aisaaslab/internal/kernel"
	"aisaaslab/internal/modules/completion"
)

type Module struct {
	app *kernel.App
	fsm *kernel.StateMachine[kernel.State, string, string]
}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "billing" }

func (m *Module) Init(app *kernel.App) error {
	m.app = app

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

	// --- 2. Legacy Event Subscription: Converts Completion Usage to MeteringEvent ---
	app.Events.Subscribe("usage.recorded", func(payload any) {
		rec, ok := payload.(*completion.UsageRecord)
		if !ok || rec == nil {
			return
		}
		app.Store.AddUsage(rec.APIKey, rec.Tokens)

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
			app.Store.RegisterServiceSubscription(ServiceSubscription{
				SubscriptionID: "sub_ai_" + rec.APIKey,
				TenantKey:      rec.APIKey,
				ServiceID:      "ai-completion",
				PlanID:         "default-ai-plan",
				ChargeType:     ChargeTypeMetered,
				Timezone:       "UTC",
				AnchorTime:     time.Now().UTC(),
				Status:         "active",
			})
		}

		app.Store.RecordMeteringEvent(MeteringEvent{
			EventID:   "evt_completion_" + time.Now().UTC().Format("20060102150405.000"),
			TenantKey: rec.APIKey,
			ServiceID: "ai-completion",
			MetricID:  "total_tokens",
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
			return app.Store.UsageFor(key) < app.Config.DailyTokenQuota, nil
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
	app.Mux.HandleFunc("POST /v1/subscription/{key}/event", m.handleFireEvent)

	return nil
}

func (m *Module) handleGetOverview(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	overview := m.app.Store.GetTenantBillingOverview(key, time.Now().UTC())
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(overview)
}

func (m *Module) handleGetServiceStatement(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	serviceID := r.PathValue("service")
	stmt, ok := m.app.Store.GetServiceBillingStatement(key, serviceID, time.Now().UTC())
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
	if sub.TenantKey == "" || sub.ServiceID == "" {
		http.Error(w, "tenant_key and service_id are required", http.StatusBadRequest)
		return
	}
	if sub.SubscriptionID == "" {
		sub.SubscriptionID = "sub_" + sub.ServiceID + "_" + sub.TenantKey
	}
	m.app.Store.RegisterServiceSubscription(sub)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"subscription_id": sub.SubscriptionID,
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
	overview := m.app.Store.GetTenantBillingOverview(key, time.Now().UTC())
	resp := map[string]any{
		"api_key":            key,
		"tokens_used":        m.app.Store.UsageFor(key),
		"daily_quota":        m.app.Config.DailyTokenQuota,
		"subscription_state": string(m.app.Store.SubscriptionState(key)),
		"overview":           overview,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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

	current := m.app.Store.SubscriptionState(key)
	if current == "unknown" {
		current = "trial"
	}

	next, err := m.fsm.Fire(r.Context(), current, strings.TrimSpace(body.Event), key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	m.app.Store.SetSubscriptionState(key, next)
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

// handleStorageHealth returns the status of all metering storage backends,
// circuit breakers, WAL state, and deduplication stats.
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
