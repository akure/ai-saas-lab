package subscription

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

type fireEventReq struct {
	Event string `json:"event"`
}

func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (m *Module) handleListPlans(w http.ResponseWriter, r *http.Request) {
	plans := m.manager.ListPlans()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"plans": plans})
}

func (m *Module) handleGetPlan(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeJSONError(w, "plan id parameter is required", http.StatusBadRequest)
		return
	}
	plan, ok := m.manager.GetPlan(id)
	if !ok {
		writeJSONError(w, "plan not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(plan)
}

func (m *Module) handleRegisterPlan(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		writeJSONError(w, "request body is required", http.StatusBadRequest)
		return
	}
	var plan Plan
	if err := json.NewDecoder(r.Body).Decode(&plan); err != nil {
		writeJSONError(w, "invalid plan json body", http.StatusBadRequest)
		return
	}
	plan.ID = strings.TrimSpace(plan.ID)
	if plan.ID == "" {
		writeJSONError(w, "plan id is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(plan.Name) == "" {
		plan.Name = plan.ID
	}
	m.manager.RegisterPlan(plan)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(plan)
}

func (m *Module) handleGetSubscription(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.PathValue("key"))
	if key == "" {
		writeJSONError(w, "tenant key parameter is required", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	contract, ok := m.manager.GetContract(key)
	if !ok {
		contract = Contract{
			SubscriptionID: "sub_trial_" + key,
			TenantKey:      key,
			PlanID:         "starter",
			State:          StateTrial,
			AnchorTime:     time.Now().UTC(),
			Timezone:       "UTC",
		}
	}
	plan, _ := m.manager.GetPlan(contract.PlanID)

	resp := map[string]any{
		"subscription_id": contract.SubscriptionID,
		"tenant_key":      contract.TenantKey,
		"plan_id":         contract.PlanID,
		"state":           string(contract.State),
		"anchor_time":     contract.AnchorTime,
		"timezone":       contract.Timezone,
		"is_usable":       contract.State.IsUsable(),
		"entitlements":    plan.Entitlements,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (m *Module) handleFireEvent(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.PathValue("key"))
	if key == "" {
		writeJSONError(w, "tenant key parameter is required", http.StatusBadRequest)
		return
	}

	if r.Body == nil {
		writeJSONError(w, "request body is required", http.StatusBadRequest)
		return
	}

	var body fireEventReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		if errors.Is(err, io.EOF) {
			writeJSONError(w, "event parameter is required in request body", http.StatusBadRequest)
			return
		}
		writeJSONError(w, "invalid json request body", http.StatusBadRequest)
		return
	}

	event := strings.TrimSpace(body.Event)
	if event == "" {
		writeJSONError(w, "event parameter is required", http.StatusBadRequest)
		return
	}

	current := m.manager.GetState(key)
	next, err := m.manager.FireTransition(key, event)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"tenant_key": key,
		"from":       string(current),
		"event":      event,
		"to":         string(next),
	})
}

func (m *Module) handleCreateContract(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		writeJSONError(w, "request body is required", http.StatusBadRequest)
		return
	}
	var c Contract
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeJSONError(w, "invalid contract json body", http.StatusBadRequest)
		return
	}
	c.TenantKey = strings.TrimSpace(c.TenantKey)
	c.PlanID = strings.TrimSpace(c.PlanID)

	if c.TenantKey == "" || c.PlanID == "" {
		writeJSONError(w, "tenant_key and plan_id are required", http.StatusBadRequest)
		return
	}

	if _, ok := m.manager.GetPlan(c.PlanID); !ok {
		writeJSONError(w, "plan_id "+c.PlanID+" does not exist", http.StatusNotFound)
		return
	}

	res := m.manager.CreateContract(c)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(res)
}
