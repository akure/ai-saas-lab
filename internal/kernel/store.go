package kernel

import (
	"sync"
	"time"
)

// APIKeyRecord is the in-memory representation of a key used by the auth layer.
type APIKeyRecord struct {
	Key    string
	Plan   string
	Active bool
}

// Store is thread-safe, multi-tenant storage supporting API keys, subscriptions,
// and service-specific metering events.
//
// Metering operations (subscriptions, events, billing statements) are delegated
// to the MeteringChain for multi-backend fan-out/cascade. The Store retains
// ownership of non-metering concerns (API keys, legacy usage counters,
// subscription FSM state).
type Store struct {
	mu            sync.RWMutex
	apiKeys       map[string]APIKeyRecord
	usage         map[string]int   // key -> tokens used today (legacy quota counter)
	subscriptions map[string]State // key -> overall subscription state (FSM)

	// meteringChain is the multi-backend storage orchestrator.
	// Set by App.NewApp() after chain construction. If nil, metering
	// methods are no-ops (safe for tests that don't initialize the chain).
	meteringChain *MeteringChain
}

func NewStore() *Store {
	s := &Store{
		apiKeys:       make(map[string]APIKeyRecord),
		usage:         make(map[string]int),
		subscriptions: make(map[string]State),
	}
	// Auto-wire a WAL-disabled, memory-only chain so the Store is
	// immediately usable without an App (e.g. in tests and migrations).
	// App.NewApp() replaces this with the config-driven chain.
	chain, err := NewMeteringChain(MeteringChainConfig{
		WAL:            WALConfig{Enabled: false},
		CircuitBreaker: DefaultCircuitBreakerConfig(),
		ChannelSize:    1000,
		DedupRetention: time.Hour,
	})
	if err == nil {
		chain.AddBackend(NewMemoryMeteringStore())
		chain.Start()
		s.meteringChain = chain
	}
	return s
}

// SetMeteringChain wires the multi-backend metering chain into the store.
// Called by App during initialization.
func (s *Store) SetMeteringChain(chain *MeteringChain) {
	s.meteringChain = chain
}

// MeteringChainRef returns the underlying MeteringChain for direct access
// (e.g., health reporting). Returns nil if chain is not initialized.
func (s *Store) MeteringChainRef() *MeteringChain {
	return s.meteringChain
}

func (s *Store) SeedAPIKey(key, plan string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if key == "" {
		return
	}
	s.apiKeys[key] = APIKeyRecord{Key: key, Plan: plan, Active: true}
}

func (s *Store) IsValidAPIKey(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.apiKeys[key]
	return ok && rec.Active
}

func (s *Store) ActivateAPIKey(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec, ok := s.apiKeys[key]; ok {
		rec.Active = true
		s.apiKeys[key] = rec
	}
}

func (s *Store) RevokeAPIKey(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec, ok := s.apiKeys[key]; ok {
		rec.Active = false
		s.apiKeys[key] = rec
	}
}

func (s *Store) APIKeyInfo(key string) (APIKeyRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.apiKeys[key]
	return rec, ok
}

func (s *Store) AddUsage(key string, tokens int) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.usage[key] += tokens
	return s.usage[key]
}

func (s *Store) UsageFor(key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.usage[key]
}

func (s *Store) SetSubscriptionState(key string, st State) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscriptions[key] = st
}

func (s *Store) SubscriptionState(key string) State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if st, ok := s.subscriptions[key]; ok {
		return st
	}
	return "unknown"
}

// ---------------------------------------------------------------------------
// Metering delegates — forward all metering operations to the MeteringChain.
// These methods maintain full backward compatibility: any code calling
// app.Store.RecordMeteringEvent(...) continues to work unchanged.
// ---------------------------------------------------------------------------

// RegisterServiceSubscription delegates to the MeteringChain for multi-backend
// fan-out. Returns any error from non-L1 backends (L1 memory never fails).
func (s *Store) RegisterServiceSubscription(sub ServiceSubscription) error {
	if s.meteringChain != nil {
		return s.meteringChain.RegisterServiceSubscription(sub)
	}
	return nil
}

// GetServiceSubscriptions delegates to the MeteringChain for priority-cascaded reads.
func (s *Store) GetServiceSubscriptions(tenantKey string) []ServiceSubscription {
	if s.meteringChain != nil {
		return s.meteringChain.GetServiceSubscriptions(tenantKey)
	}
	return nil
}

// RecordMeteringEvent delegates to the MeteringChain and maintains the legacy
// usage counter for the quota policy.
func (s *Store) RecordMeteringEvent(event MeteringEvent) error {
	if event.TenantKey == "" {
		return nil
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	var chainErr error
	if s.meteringChain != nil {
		chainErr = s.meteringChain.RecordMeteringEvent(event)
	}

	// Maintain legacy usage counter regardless of chain result.
	if event.MetricID == "total_tokens" || event.MetricID == "tokens" {
		s.mu.Lock()
		s.usage[event.TenantKey] += int(event.Quantity)
		s.mu.Unlock()
	}
	return chainErr
}

// GetServiceBillingStatement delegates to the MeteringChain for priority-cascaded reads.
func (s *Store) GetServiceBillingStatement(tenantKey, serviceID string, targetTime time.Time) (ServiceBillingStatement, bool) {
	if s.meteringChain != nil {
		return s.meteringChain.GetServiceBillingStatement(tenantKey, serviceID, targetTime)
	}
	return ServiceBillingStatement{}, false
}

// GetTenantBillingOverview delegates to the MeteringChain and enriches the
// result with the Store's subscription FSM state.
func (s *Store) GetTenantBillingOverview(tenantKey string, targetTime time.Time) TenantBillingOverview {
	if s.meteringChain != nil {
		overview := s.meteringChain.GetTenantBillingOverview(tenantKey, targetTime)
		// Enrich with the subscription FSM state from Store.
		overview.SubscriptionState = string(s.SubscriptionState(tenantKey))
		return overview
	}
	return TenantBillingOverview{
		TenantKey:         tenantKey,
		SubscriptionState: string(s.SubscriptionState(tenantKey)),
		Statements:        make([]ServiceBillingStatement, 0),
		GeneratedAt:       time.Now().UTC(),
	}
}
