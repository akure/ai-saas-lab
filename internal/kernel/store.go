package kernel

import (
	"time"
)

// Store is thread-safe metering storage delegating telemetry operations to MeteringChain.
type Store struct {
	// meteringChain is the multi-backend storage orchestrator.
	// Set by App.NewApp() after chain construction. If nil, metering
	// methods are no-ops (safe for tests that don't initialize the chain).
	meteringChain *MeteringChain
}

func NewStore() *Store {
	s := &Store{}
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

// ---------------------------------------------------------------------------
// Metering delegates — forward all metering operations to the MeteringChain.
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

// RecordMeteringEvent delegates cleanly to the MeteringChain.
func (s *Store) RecordMeteringEvent(event MeteringEvent) error {
	if event.TenantKey.IsZero() {
		return nil
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	if s.meteringChain != nil {
		return s.meteringChain.RecordMeteringEvent(event)
	}
	return nil
}

// GetServiceUsageStatement delegates to the MeteringChain for priority-cascaded reads.
func (s *Store) GetServiceUsageStatement(tenantKey, serviceID string, targetTime time.Time) (ServiceUsageStatement, bool) {
	if s.meteringChain != nil {
		return s.meteringChain.GetServiceUsageStatement(tenantKey, serviceID, targetTime)
	}
	return ServiceUsageStatement{}, false
}

// GetTenantUsageOverview delegates to the MeteringChain.
func (s *Store) GetTenantUsageOverview(tenantKey string, targetTime time.Time) TenantUsageOverview {
	if s.meteringChain != nil {
		return s.meteringChain.GetTenantUsageOverview(tenantKey, targetTime)
	}
	tk, _ := NewTenantKey(tenantKey)
	return TenantUsageOverview{
		TenantKey:   tk,
		Statements:  make([]ServiceUsageStatement, 0),
		GeneratedAt: time.Now().UTC(),
	}
}
