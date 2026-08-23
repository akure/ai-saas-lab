package subscription

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"aisaaslab/internal/kernel"
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

// PlanStore abstracts plan catalog operations.
type PlanStore interface {
	RegisterPlan(plan Plan)
	GetPlan(planID string) (Plan, bool)
	ListPlans() []Plan
}

// ContractStore abstracts contract persistence.
type ContractStore interface {
	CreateContract(contract Contract) Contract
	GetContract(tenantKey string) (Contract, bool)
	GetState(tenantKey string) State
	SetState(tenantKey string, state State)
	FireTransition(tenantKey string, event string) (State, error)
}

// Manager is a pure, thread-safe manager for subscription plans, contracts, and FSM state transitions.
type Manager struct {
	mu        sync.RWMutex
	plans     map[string]Plan
	contracts map[string]Contract // tenantKey -> Contract
	catalog   kernel.PlanCatalog
	app       *kernel.App
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

func (m *Manager) SetCatalog(cat kernel.PlanCatalog) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.catalog = cat
}

func (m *Manager) SetApp(app *kernel.App) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.app = app
}

func clonePlan(p Plan) Plan {
	cp := Plan{
		ID:           p.ID,
		Name:         p.Name,
		Entitlements: make(map[string]Entitlement, len(p.Entitlements)),
	}
	for k, v := range p.Entitlements {
		cp.Entitlements[k] = v
	}
	return cp
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

// RegisterPlan adds or updates a subscription plan with deep-copy protection.
func (m *Manager) RegisterPlan(plan Plan) {
	m.mu.Lock()
	defer m.mu.Unlock()
	plan.ID = strings.TrimSpace(plan.ID)
	if plan.Entitlements == nil {
		plan.Entitlements = make(map[string]Entitlement)
	}
	m.plans[plan.ID] = clonePlan(plan)
}

// GetPlan retrieves a plan by ID with deep-copy protection against data races.
func (m *Manager) GetPlan(planID string) (Plan, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	planID = strings.TrimSpace(planID)
	p, ok := m.plans[planID]
	if !ok {
		return Plan{}, false
	}
	return clonePlan(p), true
}

// ListPlans returns all registered subscription plans.
func (m *Manager) ListPlans() []Plan {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]Plan, 0, len(m.plans))
	for _, p := range m.plans {
		list = append(list, clonePlan(p))
	}
	return list
}

// CreateContract creates or registers a subscription contract for a tenant.
func (m *Manager) CreateContract(contract Contract) Contract {
	m.mu.Lock()
	defer m.mu.Unlock()

	contract.TenantKey = strings.TrimSpace(contract.TenantKey)
	contract.PlanID = strings.TrimSpace(contract.PlanID)

	if contract.Timezone != "" {
		if _, err := time.LoadLocation(contract.Timezone); err != nil {
			contract.Timezone = "UTC"
		}
	} else {
		contract.Timezone = "UTC"
	}

	if contract.AnchorTime.IsZero() {
		contract.AnchorTime = time.Now().UTC()
	}
	if contract.State.IsZero() {
		contract.State = StateTrial
	}
	if contract.SubscriptionID == "" {
		contract.SubscriptionID = "sub_" + contract.PlanID + "_" + contract.TenantKey
	}

	m.contracts[contract.TenantKey] = contract

	if m.app != nil {
		if m.app.Store != nil {
			_ = m.app.Store.RegisterServiceSubscription(kernel.ServiceSubscription{
				SubscriptionID: kernel.MustSubscriptionID(contract.SubscriptionID),
				TenantKey:      kernel.MustTenantKey(contract.TenantKey),
				ServiceID:      kernel.ServiceIDAICompletion,
				PlanID:         kernel.PlanID(contract.PlanID),
				ChargeType:     "metered",
				Timezone:       contract.Timezone,
				AnchorTime:     contract.AnchorTime,
				Status:         toKernelStatus(contract.State),
			})
		}
		if m.app.Events != nil {
			m.app.Events.Publish(kernel.TopicSubscriptionUpdated, map[string]string{
				"tenant_key": contract.TenantKey,
				"from":       "",
				"event":      "create",
				"to":         string(contract.State),
			})
		}
	}

	return contract
}

// GetContract retrieves the subscription contract for a tenant with store hydration recovery.
func (m *Manager) GetContract(tenantKey string) (Contract, bool) {
	tenantKey = strings.TrimSpace(tenantKey)
	m.mu.RLock()
	c, ok := m.contracts[tenantKey]
	m.mu.RUnlock()
	if ok {
		return c, true
	}

	// Fallback hydration from kernel.Store if available
	if m.app != nil && m.app.Store != nil {
		subs := m.app.Store.GetServiceSubscriptions(tenantKey)
		if len(subs) > 0 {
			sub := subs[0]
			c = Contract{
				SubscriptionID: sub.SubscriptionID.String(),
				TenantKey:      sub.TenantKey.String(),
				PlanID:         sub.PlanID.String(),
				State:          fromKernelStatus(sub.Status),
				AnchorTime:     sub.AnchorTime,
				Timezone:       sub.Timezone,
			}
			m.mu.Lock()
			m.contracts[tenantKey] = c
			m.mu.Unlock()
			return c, true
		}
	}

	return Contract{}, false
}

// GetState retrieves the subscription state for a tenant (defaults to StateTrial if contract not found).
func (m *Manager) GetState(tenantKey string) State {
	tenantKey = strings.TrimSpace(tenantKey)
	if c, ok := m.GetContract(tenantKey); ok {
		return c.State
	}
	return StateTrial
}

// SetState explicitly sets the subscription state for a tenant.
func (m *Manager) SetState(tenantKey string, state State) {
	tenantKey = strings.TrimSpace(tenantKey)
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.contracts[tenantKey]
	prevState := StateTrial
	if !ok {
		c = Contract{
			SubscriptionID: "sub_" + tenantKey,
			TenantKey:      tenantKey,
			PlanID:         "starter",
			State:          state,
			AnchorTime:     time.Now().UTC(),
			Timezone:       "UTC",
		}
	} else {
		prevState = c.State
		c.State = state
	}
	m.contracts[tenantKey] = c

	if m.app != nil {
		if m.app.Store != nil {
			_ = m.app.Store.RegisterServiceSubscription(kernel.ServiceSubscription{
				SubscriptionID: kernel.MustSubscriptionID(c.SubscriptionID),
				TenantKey:      kernel.MustTenantKey(c.TenantKey),
				ServiceID:      kernel.ServiceIDAICompletion,
				PlanID:         kernel.PlanID(c.PlanID),
				ChargeType:     "metered",
				Timezone:       c.Timezone,
				AnchorTime:     c.AnchorTime,
				Status:         toKernelStatus(c.State),
			})
		}
		if m.app.Events != nil {
			m.app.Events.Publish(kernel.TopicSubscriptionUpdated, map[string]string{
				"tenant_key": tenantKey,
				"from":       string(prevState),
				"event":      "set_state",
				"to":         string(state),
			})
		}
	}
}

// FireTransition executes an FSM state transition for a tenant given an event.
func (m *Manager) FireTransition(tenantKey string, event string) (State, error) {
	tenantKey = strings.TrimSpace(tenantKey)
	m.mu.Lock()
	defer m.mu.Unlock()

	c, ok := m.contracts[tenantKey]
	currentState := StateTrial
	if ok {
		currentState = c.State
	}

	nextState, err := transitionFSM(currentState, event)
	if err != nil {
		return currentState, err
	}

	if !ok {
		c = Contract{
			SubscriptionID: "sub_" + tenantKey,
			TenantKey:      tenantKey,
			PlanID:         "starter",
			State:          nextState,
			AnchorTime:     time.Now().UTC(),
			Timezone:       "UTC",
		}
	} else {
		c.State = nextState
	}
	m.contracts[tenantKey] = c

	if m.app != nil {
		if m.app.Store != nil {
			_ = m.app.Store.RegisterServiceSubscription(kernel.ServiceSubscription{
				SubscriptionID: kernel.MustSubscriptionID(c.SubscriptionID),
				TenantKey:      kernel.MustTenantKey(c.TenantKey),
				ServiceID:      kernel.ServiceIDAICompletion,
				PlanID:         kernel.PlanID(c.PlanID),
				ChargeType:     "metered",
				Timezone:       c.Timezone,
				AnchorTime:     c.AnchorTime,
				Status:         toKernelStatus(c.State),
			})
		}
		if m.app.Events != nil {
			m.app.Events.Publish(kernel.TopicSubscriptionUpdated, map[string]string{
				"tenant_key": tenantKey,
				"from":       string(currentState),
				"event":      event,
				"to":         string(nextState),
			})
		}
	}

	return nextState, nil
}

// CheckEntitlement evaluates whether a tenant is allowed to consume quantity of a metric based on state and plan limits.
func (m *Manager) CheckEntitlement(tenantKey string, metricID string, currentUsage int64) (bool, error) {
	tenantKey = strings.TrimSpace(tenantKey)
	metricID = strings.TrimSpace(metricID)

	c, ok := m.GetContract(tenantKey)
	if !ok {
		c = Contract{TenantKey: tenantKey, PlanID: "starter", State: StateTrial}
	}

	if !c.State.IsUsable() {
		return false, fmt.Errorf("subscription state %q is not active or usable", c.State)
	}

	m.mu.RLock()
	p, ok := m.plans[c.PlanID]
	if !ok {
		p, ok = m.plans["starter"]
	}
	m.mu.RUnlock()

	if !ok {
		return true, nil
	}

	ent, ok := p.Entitlements[metricID]
	if !ok {
		return true, nil
	}

	if !ent.Allowed {
		return false, fmt.Errorf("metric %q is disallowed for plan %q", metricID, p.ID)
	}

	if ent.Quota > 0 && currentUsage >= ent.Quota {
		return false, fmt.Errorf("plan %q quota exceeded for metric %q (%d/%d)", p.ID, metricID, currentUsage, ent.Quota)
	}

	return true, nil
}

func transitionFSM(current State, rawEvent string) (State, error) {
	evt := strings.ToLower(strings.TrimSpace(rawEvent))
	switch current {
	case StateTrial:
		switch evt {
		case "activate", "payment_succeeded":
			return StateActive, nil
		case "cancel":
			return StateCancelled, nil
		case "payment_failed", "trial_expired":
			return StatePastDue, nil
		}
	case StateActive:
		switch evt {
		case "payment_failed":
			return StatePastDue, nil
		case "cancel":
			return StateCancelled, nil
		}
	case StatePastDue:
		switch evt {
		case "payment_succeeded", "activate":
			return StateActive, nil
		case "cancel":
			return StateCancelled, nil
		}
	case StateCancelled:
		switch evt {
		case "reactivate", "activate":
			return StateActive, nil
		}
	}
	return current, fmt.Errorf("%w: cannot fire event %q from state %q", ErrInvalidState, rawEvent, current)
}

func toKernelStatus(st State) kernel.SubscriptionStatus {
	switch st {
	case StateActive, StateTrial:
		return kernel.SubscriptionStatusActive
	case StatePastDue:
		return kernel.SubscriptionStatusPastDue
	case StateCancelled:
		return kernel.SubscriptionStatusCancelled
	default:
		return kernel.SubscriptionStatusActive
	}
}

func fromKernelStatus(st kernel.SubscriptionStatus) State {
	switch st {
	case kernel.SubscriptionStatusActive:
		return StateActive
	case kernel.SubscriptionStatusPastDue:
		return StatePastDue
	case kernel.SubscriptionStatusCancelled:
		return StateCancelled
	default:
		return StateTrial
	}
}
