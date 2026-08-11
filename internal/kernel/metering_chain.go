package kernel

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// MeteringChain — multi-backend orchestrator with production-grade resilience.
// ---------------------------------------------------------------------------

// writeOp represents a deferred write operation sent to an async backend
// via its bounded channel.
type writeOp struct {
	opType string          // "record_event" | "register_sub"
	event  *MeteringEvent
	sub    *ServiceSubscription
}

// MeteringChainConfig holds all tuning parameters for the chain.
type MeteringChainConfig struct {
	WAL             WALConfig
	CircuitBreaker  CircuitBreakerConfig
	ChannelSize     int           // bounded channel capacity per async backend (default: 10000)
	DedupRetention  time.Duration // how long to remember EventIDs (default: 1h)
	ShutdownTimeout time.Duration // max duration to wait for async workers to drain during Stop() (default: 5s)
}

// DefaultMeteringChainConfig returns production-sensible defaults.
func DefaultMeteringChainConfig() MeteringChainConfig {
	return MeteringChainConfig{
		WAL:             DefaultWALConfig(),
		CircuitBreaker:  DefaultCircuitBreakerConfig(),
		ChannelSize:     10000,
		DedupRetention:  time.Hour,
		ShutdownTimeout: 5 * time.Second,
	}
}

// MeteringChain orchestrates multi-backend metering storage.
//
// Write path:
//  1. Dedup check (EventID seen before? → reject silently)
//  2. Sync write to L1 (memory, priority 0) — if this fails, return
//  3. Async fan-out to L2..Ln via bounded channels
//  4. If channel full → overflow to WAL (never drop, never block caller)
//  5. If async backend write fails → circuit breaker + WAL capture
//
// Read path:
//  1. Try backends in priority order (lowest first)
//  2. Skip backends with open circuit breakers or unhealthy status
//  3. On miss/error → try next backend
//  4. On success → return immediately
type MeteringChain struct {
	mu         sync.RWMutex
	backends   []MeteringStore             // sorted by Priority ascending
	breakers   map[string]*CircuitBreaker  // backend name → circuit breaker
	asyncChans map[string]chan writeOp     // backend name → bounded write channel
	dedup      *EventDedup
	wal        *MeteringWAL
	cfg        MeteringChainConfig
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// BackendHealth reports the status of a single backend for observability.
type BackendHealth struct {
	Name          string `json:"name"`
	Priority      int    `json:"priority"`
	Healthy       bool   `json:"healthy"`
	CircuitState  string `json:"circuit_state"`  // "closed", "open", "half_open"
	PendingWrites int    `json:"pending_writes"` // async channel depth
}

// NewMeteringChain creates the chain orchestrator. Call AddBackend() to
// register backends, then Start() to begin async workers and WAL replay.
func NewMeteringChain(cfg MeteringChainConfig) (*MeteringChain, error) {
	ctx, cancel := context.WithCancel(context.Background())

	var wal *MeteringWAL
	var err error
	if cfg.WAL.Enabled {
		wal, err = NewMeteringWAL(cfg.WAL)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("metering chain: failed to create WAL: %w", err)
		}
	}

	return &MeteringChain{
		backends:   make([]MeteringStore, 0),
		breakers:   make(map[string]*CircuitBreaker),
		asyncChans: make(map[string]chan writeOp),
		dedup:      NewEventDedup(cfg.DedupRetention),
		wal:        wal,
		cfg:        cfg,
		ctx:        ctx,
		cancel:     cancel,
	}, nil
}

// AddBackend registers a new storage backend. The chain is re-sorted by
// priority after each addition. If the backend has priority > 0 (not the
// sync L1), an async write worker goroutine is started.
func (c *MeteringChain) AddBackend(backend MeteringStore) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.backends = append(c.backends, backend)
	sort.Slice(c.backends, func(i, j int) bool {
		return c.backends[i].Priority() < c.backends[j].Priority()
	})

	// Create circuit breaker for this backend.
	c.breakers[backend.Name()] = NewCircuitBreaker(c.cfg.CircuitBreaker)

	// For non-L1 backends (priority > 0), set up async write channel + worker.
	if backend.Priority() > 0 {
		ch := make(chan writeOp, c.cfg.ChannelSize)
		c.asyncChans[backend.Name()] = ch
		c.wg.Add(1)
		go c.asyncWriteWorker(backend, ch)
	}
}

// RemoveBackend removes a backend by name. Its async channel is drained
// and closed, and the worker goroutine exits.
func (c *MeteringChain) RemoveBackend(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Close and remove async channel (worker will exit on channel close).
	if ch, ok := c.asyncChans[name]; ok {
		close(ch)
		delete(c.asyncChans, name)
	}

	// Remove circuit breaker.
	delete(c.breakers, name)

	// Remove from backends slice.
	filtered := make([]MeteringStore, 0, len(c.backends))
	for _, b := range c.backends {
		if b.Name() != name {
			filtered = append(filtered, b)
		}
	}
	c.backends = filtered
}

// ---------------------------------------------------------------------------
// Write path
// ---------------------------------------------------------------------------

// RecordMeteringEvent implements MeteringStore. Sync to L1, async fan-out
// to all other backends with dedup, circuit breaker, and WAL protection.
func (c *MeteringChain) RecordMeteringEvent(event MeteringEvent) error {
	// Step 1: Dedup check.
	if c.dedup.IsDuplicate(event.EventID) {
		return nil // silently reject duplicate
	}

	c.mu.RLock()
	backends := make([]MeteringStore, len(c.backends))
	copy(backends, c.backends)
	c.mu.RUnlock()

	// Step 2: Sync write to L1 (first backend, priority 0 = memory).
	for _, b := range backends {
		if b.Priority() == 0 {
			if err := b.RecordMeteringEvent(event); err != nil {
				return fmt.Errorf("L1 write failed: %w", err)
			}
			break
		}
	}

	// Step 3: Async fan-out to all other backends.
	for _, b := range backends {
		if b.Priority() == 0 {
			continue // already written synchronously
		}

		c.mu.RLock()
		cb, hasCB := c.breakers[b.Name()]
		ch, hasCh := c.asyncChans[b.Name()]
		c.mu.RUnlock()

		// Skip if circuit is open.
		if hasCB && !cb.Allow() {
			c.walAppendEvent(b.Name(), event)
			continue
		}

		if hasCh {
			op := writeOp{opType: "record_event", event: &event}
			select {
			case ch <- op:
				// Successfully queued for async processing.
			default:
				// Channel full — backpressure! Overflow to WAL.
				c.walAppendEvent(b.Name(), event)
			}
		}
	}
	return nil
}

// RegisterServiceSubscription implements MeteringStore. Subscriptions are
// rare operations where correctness trumps speed, so we write synchronously
// to all healthy backends (not async), collecting any errors.
func (c *MeteringChain) RegisterServiceSubscription(sub ServiceSubscription) error {
	c.mu.RLock()
	backends := make([]MeteringStore, len(c.backends))
	copy(backends, c.backends)
	c.mu.RUnlock()

	var firstErr error
	for _, b := range backends {
		c.mu.RLock()
		cb, hasCB := c.breakers[b.Name()]
		c.mu.RUnlock()

		if hasCB && !cb.Allow() {
			c.walAppendSub(b.Name(), sub)
			continue
		}

		if err := b.RegisterServiceSubscription(sub); err != nil {
			if hasCB {
				cb.RecordFailure()
			}
			c.walAppendSub(b.Name(), sub)
			if firstErr == nil {
				firstErr = fmt.Errorf("backend %q: %w", b.Name(), err)
			}
			continue
		}

		if hasCB {
			cb.RecordSuccess()
		}
	}
	return firstErr
}

// ---------------------------------------------------------------------------
// Read path — cascade by priority
// ---------------------------------------------------------------------------

// GetServiceSubscriptions implements MeteringStore. Cascades through backends
// by priority until one returns a non-nil result.
func (c *MeteringChain) GetServiceSubscriptions(tenantKey string) []ServiceSubscription {
	c.mu.RLock()
	backends := make([]MeteringStore, len(c.backends))
	copy(backends, c.backends)
	c.mu.RUnlock()

	for _, b := range backends {
		c.mu.RLock()
		cb, hasCB := c.breakers[b.Name()]
		c.mu.RUnlock()

		if !b.Healthy() || (hasCB && !cb.Allow()) {
			continue
		}

		result := b.GetServiceSubscriptions(tenantKey)
		if result != nil {
			if hasCB {
				cb.RecordSuccess()
			}
			return result
		}
	}
	return nil
}

// GetServiceUsageStatement implements MeteringStore. Cascades through
// backends by priority until one returns a valid statement.
func (c *MeteringChain) GetServiceUsageStatement(tenantKey, serviceID string, targetTime time.Time) (ServiceUsageStatement, bool) {
	c.mu.RLock()
	backends := make([]MeteringStore, len(c.backends))
	copy(backends, c.backends)
	c.mu.RUnlock()

	for _, b := range backends {
		c.mu.RLock()
		cb, hasCB := c.breakers[b.Name()]
		c.mu.RUnlock()

		if !b.Healthy() || (hasCB && !cb.Allow()) {
			continue
		}

		stmt, ok := b.GetServiceUsageStatement(tenantKey, serviceID, targetTime)
		if ok {
			if hasCB {
				cb.RecordSuccess()
			}
			return stmt, true
		}
	}
	return ServiceUsageStatement{}, false
}

// GetTenantUsageOverview implements MeteringStore. Cascades through
// backends by priority until one returns an overview with statements.
func (c *MeteringChain) GetTenantUsageOverview(tenantKey string, targetTime time.Time) TenantUsageOverview {
	c.mu.RLock()
	backends := make([]MeteringStore, len(c.backends))
	copy(backends, c.backends)
	c.mu.RUnlock()

	for _, b := range backends {
		c.mu.RLock()
		cb, hasCB := c.breakers[b.Name()]
		c.mu.RUnlock()

		if !b.Healthy() || (hasCB && !cb.Allow()) {
			continue
		}

		overview := b.GetTenantUsageOverview(tenantKey, targetTime)
		// Return if this backend has data (statements present or
		// it's the L1 memory store which always has the authoritative view).
		if len(overview.Statements) > 0 || b.Priority() == 0 {
			if hasCB {
				cb.RecordSuccess()
			}
			return overview
		}
	}

	tk, _ := NewTenantKey(tenantKey)
	return TenantUsageOverview{
		TenantKey:   tk,
		Statements:  make([]ServiceUsageStatement, 0),
		GeneratedAt: time.Now().UTC(),
	}
}

// ---------------------------------------------------------------------------
// MeteringStore identity (the chain itself satisfies the interface)
// ---------------------------------------------------------------------------

func (c *MeteringChain) Name() string                 { return "chain" }
func (c *MeteringChain) Priority() int                 { return -1 }
func (c *MeteringChain) Ping(_ context.Context) error  { return nil } // chain-of-chains always reachable
func (c *MeteringChain) Healthy() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, b := range c.backends {
		if b.Healthy() {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Health & Observability
// ---------------------------------------------------------------------------

// HealthReport returns the status of all registered backends including
// circuit breaker state and async channel depth.
func (c *MeteringChain) HealthReport() []BackendHealth {
	c.mu.RLock()
	defer c.mu.RUnlock()

	report := make([]BackendHealth, 0, len(c.backends))
	for _, b := range c.backends {
		h := BackendHealth{
			Name:     b.Name(),
			Priority: b.Priority(),
			Healthy:  b.Healthy(),
		}

		if cb, ok := c.breakers[b.Name()]; ok {
			h.CircuitState = string(cb.State())
		} else {
			h.CircuitState = string(CircuitClosed)
		}

		if ch, ok := c.asyncChans[b.Name()]; ok {
			h.PendingWrites = len(ch)
		}

		report = append(report, h)
	}
	return report
}

// WALDepth returns the total pending WAL entries. Returns 0 if WAL is disabled.
func (c *MeteringChain) WALDepth() int {
	if c.wal == nil {
		return 0
	}
	return c.wal.Depth()
}

// WALEnabled reports whether the WAL is active.
func (c *MeteringChain) WALEnabled() bool {
	return c.wal != nil && c.cfg.WAL.Enabled
}

// WALActiveSegment returns the name of the currently active WAL segment.
func (c *MeteringChain) WALActiveSegment() string {
	if c.wal == nil {
		return ""
	}
	return c.wal.ActiveSegmentName()
}

// WALTotalSegments returns the total number of WAL segment files.
func (c *MeteringChain) WALTotalSegments() int {
	if c.wal == nil {
		return 0
	}
	return c.wal.TotalSegments()
}

// DedupTrackedEvents returns the number of EventIDs currently being tracked
// by the deduplication filter.
func (c *MeteringChain) DedupTrackedEvents() int {
	if c.dedup == nil {
		return 0
	}
	return c.dedup.Len()
}

// DedupRetention returns the configured deduplication retention window.
func (c *MeteringChain) DedupRetention() time.Duration {
	return c.cfg.DedupRetention
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

// Start begins async write workers and the WAL retry loop. Call after all
// backends have been added via AddBackend().
func (c *MeteringChain) Start() {
	if c.wal != nil {
		c.wal.StartRetryLoop(c.replayWALEntry)
	}
}

// Stop gracefully shuts down the chain: drains async channels, flushes WAL,
// stops workers and background goroutines. It waits up to ShutdownTimeout (default 5s)
// for async workers to finish draining before forcing shutdown to prevent hanging indefinitely
// (e.g. when a slow backend write is blocked by a network partition).
func (c *MeteringChain) Stop() {
	// Signal all goroutines to stop.
	c.cancel()

	// Close all async channels (workers exit on channel close).
	c.mu.Lock()
	for name, ch := range c.asyncChans {
		close(ch)
		delete(c.asyncChans, name)
	}
	c.mu.Unlock()

	// Wait for all async workers to finish draining with a timeout.
	timeout := c.cfg.ShutdownTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All async workers drained successfully.
	case <-time.After(timeout):
		// Timeout expired waiting for workers (e.g. worker hung on slow/unresponsive backend).
		// Force proceeding with shutdown so process/pod termination does not hang indefinitely.
	}

	// Stop the WAL and dedup background goroutines.
	if c.wal != nil {
		c.wal.Stop()
	}
	if c.dedup != nil {
		c.dedup.Stop()
	}
}

// ---------------------------------------------------------------------------
// Internal: async write worker
// ---------------------------------------------------------------------------

// asyncWriteWorker drains the bounded channel for a single backend,
// writing each operation and updating the circuit breaker on success/failure
// using the returned error — not Healthy() polling.
func (c *MeteringChain) asyncWriteWorker(backend MeteringStore, ch chan writeOp) {
	defer c.wg.Done()

	for op := range ch {
		c.mu.RLock()
		cb, hasCB := c.breakers[backend.Name()]
		c.mu.RUnlock()

		// Check circuit breaker before attempting.
		if hasCB && !cb.Allow() {
			switch op.opType {
			case "record_event":
				c.walAppendEvent(backend.Name(), *op.event)
			case "register_sub":
				c.walAppendSub(backend.Name(), *op.sub)
			}
			continue
		}

		switch op.opType {
		case "record_event":
			if err := backend.RecordMeteringEvent(*op.event); err != nil {
				if hasCB {
					cb.RecordFailure()
				}
				c.walAppendEvent(backend.Name(), *op.event)
			} else if hasCB {
				cb.RecordSuccess()
			}

		case "register_sub":
			if err := backend.RegisterServiceSubscription(*op.sub); err != nil {
				if hasCB {
					cb.RecordFailure()
				}
				c.walAppendSub(backend.Name(), *op.sub)
			} else if hasCB {
				cb.RecordSuccess()
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Internal: WAL helpers
// ---------------------------------------------------------------------------

func (c *MeteringChain) walAppendEvent(backendName string, event MeteringEvent) {
	if c.wal == nil {
		return
	}
	payload, _ := json.Marshal(event)
	_ = c.wal.Append(WALEntry{
		Timestamp:   time.Now(),
		Operation:   "record_event",
		BackendName: backendName,
		Payload:     payload,
	})
}

func (c *MeteringChain) walAppendSub(backendName string, sub ServiceSubscription) {
	if c.wal == nil {
		return
	}
	payload, _ := json.Marshal(sub)
	_ = c.wal.Append(WALEntry{
		Timestamp:   time.Now(),
		Operation:   "register_sub",
		BackendName: backendName,
		Payload:     payload,
	})
}

// replayWALEntry is the callback passed to the WAL retry loop. It routes
// each replayed entry to the correct backend.
func (c *MeteringChain) replayWALEntry(entry WALEntry) error {
	c.mu.RLock()
	var target MeteringStore
	for _, b := range c.backends {
		if b.Name() == entry.BackendName {
			target = b
			break
		}
	}
	cb, hasCB := c.breakers[entry.BackendName]
	c.mu.RUnlock()

	if target == nil {
		return fmt.Errorf("backend %q not found", entry.BackendName)
	}

	// Check circuit breaker.
	if hasCB && !cb.Allow() {
		return fmt.Errorf("backend %q circuit is open", entry.BackendName)
	}

	switch entry.Operation {
	case "record_event":
		var event MeteringEvent
		if err := json.Unmarshal(entry.Payload, &event); err != nil {
			return fmt.Errorf("unmarshal event: %w", err)
		}
		if err := target.RecordMeteringEvent(event); err != nil {
			if hasCB {
				cb.RecordFailure()
			}
			return fmt.Errorf("backend %q replay failed: %w", entry.BackendName, err)
		}

	case "register_sub":
		var sub ServiceSubscription
		if err := json.Unmarshal(entry.Payload, &sub); err != nil {
			return fmt.Errorf("unmarshal subscription: %w", err)
		}
		if err := target.RegisterServiceSubscription(sub); err != nil {
			if hasCB {
				cb.RecordFailure()
			}
			return fmt.Errorf("backend %q replay failed: %w", entry.BackendName, err)
		}

	default:
		return fmt.Errorf("unknown WAL operation: %q", entry.Operation)
	}

	if hasCB {
		cb.RecordSuccess()
	}
	return nil
}
