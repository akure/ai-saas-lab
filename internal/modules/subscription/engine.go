package subscription

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	ErrContractNotFound = errors.New("subscription contract not found")
	ErrPlanNotFound     = errors.New("subscription plan not found")
	ErrInvalidState     = errors.New("invalid subscription state transition")
)

// State defines the lifecycle state of a tenant's subscription.
type State string

const (
	StateTrial     State = "trial"
	StateActive    State = "active"
	StatePastDue   State = "past_due"
	StateCancelled State = "cancelled"
)

// String returns the string representation of State, satisfying fmt.Stringer.
func (s State) String() string {
	return string(s)
}

// IsZero reports whether the state is uninitialized or empty.
func (s State) IsZero() bool {
	return strings.TrimSpace(string(s)) == ""
}

// IsValid reports whether the state is a recognized valid subscription state.
func (s State) IsValid() bool {
	switch s {
	case StateTrial, StateActive, StatePastDue, StateCancelled:
		return true
	default:
		return false
	}
}

// IsUsable reports whether the subscription state permits active service usage (active or trial).
func (s State) IsUsable() bool {
	return s == StateActive || s == StateTrial
}

// IsActive reports whether the state is explicitly StateActive.
func (s State) IsActive() bool {
	return s == StateActive
}

// IsPastDue reports whether the state is StatePastDue.
func (s State) IsPastDue() bool {
	return s == StatePastDue
}

// IsCancelled reports whether the state is StateCancelled.
func (s State) IsCancelled() bool {
	return s == StateCancelled
}

// Equal performs a case-insensitive comparison with another state.
func (s State) Equal(other State) bool {
	return strings.EqualFold(strings.TrimSpace(string(s)), strings.TrimSpace(string(other)))
}

// ParseState parses and validates a raw string into a State.
func ParseState(str string) (State, error) {
	s := State(strings.ToLower(strings.TrimSpace(str)))
	if !s.IsValid() {
		return "", fmt.Errorf("%w: unrecognized subscription state %q", ErrInvalidState, str)
	}
	return s, nil
}

// Entitlement defines a feature or metric access rule for a plan.
type Entitlement struct {
	MetricID string `json:"metric_id"`
	Allowed  bool   `json:"allowed"`
	Quota    int64  `json:"quota"` // Monthly or cycle quota limit (0 = unlimited)
}

// Plan defines a product pricing tier and its associated entitlements.
type Plan struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Entitlements map[string]Entitlement `json:"entitlements"`
}

// Contract represents a tenant's active subscription contract for a plan.
type Contract struct {
	SubscriptionID string    `json:"subscription_id"`
	TenantKey      string    `json:"tenant_key"`
	PlanID         string    `json:"plan_id"`
	State          State     `json:"state"`
	AnchorTime     time.Time `json:"anchor_time"`
	Timezone       string    `json:"timezone"`
}

// Manager is a pure, thread-safe manager for subscription plans, contracts, and FSM state transitions.
// Zero kernel dependencies — uses Go primitives (string, time.Time, bool, int64).
type Manager struct {
	mu        sync.RWMutex
	plans     map[string]Plan
	contracts map[string]Contract // tenantKey -> Contract
}

// NewManager creates a new Manager instance.
func NewManager() *Manager {
	m := &Manager{
		plans:     make(map[string]Plan),
		contracts: make(map[string]Contract),
	}
	m.seedDefaultPlans()
	return m
}

func (m *Manager) seedDefaultPlans() {
	m.plans["starter"] = Plan{
		ID:   "starter",
		Name: "Starter Tier",
		Entitlements: map[string]Entitlement{
			"total_tokens": {MetricID: "total_tokens", Allowed: true, Quota: 1000000},
		},
	}
	m.plans["pro"] = Plan{
		ID:   "pro",
		Name: "Pro Tier",
		Entitlements: map[string]Entitlement{
			"total_tokens": {MetricID: "total_tokens", Allowed: true, Quota: 10000000},
		},
	}
}

// RegisterPlan adds or updates a subscription plan.
func (m *Manager) RegisterPlan(plan Plan) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if plan.Entitlements == nil {
		plan.Entitlements = make(map[string]Entitlement)
	}
	m.plans[plan.ID] = plan
}

// GetPlan retrieves a plan by ID.
func (m *Manager) GetPlan(planID string) (Plan, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.plans[planID]
	return p, ok
}

// CreateContract creates or registers a subscription contract for a tenant.
func (m *Manager) CreateContract(contract Contract) Contract {
	m.mu.Lock()
	defer m.mu.Unlock()

	if contract.Timezone == "" {
		contract.Timezone = "UTC"
	}
	if contract.AnchorTime.IsZero() {
		contract.AnchorTime = time.Now().UTC()
	}
	if contract.State == "" {
		contract.State = StateTrial
	}

	m.contracts[contract.TenantKey] = contract
	return contract
}

// GetContract retrieves the subscription contract for a tenant.
func (m *Manager) GetContract(tenantKey string) (Contract, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.contracts[tenantKey]
	return c, ok
}

// GetState retrieves the subscription state for a tenant (defaults to StateTrial if contract not found).
func (m *Manager) GetState(tenantKey string) State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if c, ok := m.contracts[tenantKey]; ok {
		return c.State
	}
	return StateTrial
}

// SetState explicitly sets the subscription state for a tenant.
func (m *Manager) SetState(tenantKey string, state State) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.contracts[tenantKey]
	if !ok {
		c = Contract{
			SubscriptionID: "sub_" + tenantKey,
			TenantKey:      tenantKey,
			PlanID:         "default",
			State:          state,
			AnchorTime:     time.Now().UTC(),
			Timezone:       "UTC",
		}
	} else {
		c.State = state
	}
	m.contracts[tenantKey] = c
}

// FireTransition executes an FSM state transition for a tenant given an event (e.g. "activate", "payment_failed", "cancel").
func (m *Manager) FireTransition(tenantKey string, event string) (State, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.contracts[tenantKey]
	currentState := StateTrial
	if ok {
		currentState = c.State
	}

	nextState, err := transitionFSM(currentState, strings.TrimSpace(event))
	if err != nil {
		return currentState, err
	}

	if !ok {
		c = Contract{
			SubscriptionID: "sub_" + tenantKey,
			TenantKey:      tenantKey,
			PlanID:         "default",
			State:          nextState,
			AnchorTime:     time.Now().UTC(),
			Timezone:       "UTC",
		}
	} else {
		c.State = nextState
	}
	m.contracts[tenantKey] = c
	return nextState, nil
}

func transitionFSM(current State, event string) (State, error) {
	switch current {
	case StateTrial:
		switch event {
		case "activate":
			return StateActive, nil
		case "cancel":
			return StateCancelled, nil
		}
	case StateActive:
		switch event {
		case "payment_failed":
			return StatePastDue, nil
		case "cancel":
			return StateCancelled, nil
		}
	case StatePastDue:
		switch event {
		case "payment_succeeded":
			return StateActive, nil
		case "cancel":
			return StateCancelled, nil
		}
	}
	return current, fmt.Errorf("%w: cannot fire event %q from state %q", ErrInvalidState, event, current)
}
