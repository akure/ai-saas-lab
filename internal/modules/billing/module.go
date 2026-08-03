package billing

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

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

	// --- Event subscription: reacts to CompletionModule's usage events
	// without ever importing anything beyond its message type. ---
	app.Events.Subscribe("usage.recorded", func(payload any) {
		rec, ok := payload.(*completion.UsageRecord)
		if !ok {
			return
		}
		app.Store.AddUsage(rec.APIKey, rec.Tokens)
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
		// Note: no transition exists from "cancelled" to anything, and none
		// from "trial" straight to "past_due" — both are structurally blocked.

	app.Mux.HandleFunc("GET /v1/usage/{key}", m.handleGetUsage)
	app.Mux.HandleFunc("POST /v1/subscription/{key}/event", m.handleFireEvent)

	return nil
}

func (m *Module) handleGetUsage(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	resp := map[string]any{
		"api_key":            key,
		"tokens_used":        m.app.Store.UsageFor(key),
		"daily_quota":        m.app.Config.DailyTokenQuota,
		"subscription_state": string(m.app.Store.SubscriptionState(key)),
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
		current = "trial" // first time we see this key, it starts on trial
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
