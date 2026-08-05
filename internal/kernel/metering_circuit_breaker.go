package kernel

import (
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// CircuitBreaker — per-backend protection against cascading failures.
// ---------------------------------------------------------------------------

// CircuitState represents the three possible states of a circuit breaker.
type CircuitState string

const (
	// CircuitClosed is the normal operating state. All requests pass through.
	CircuitClosed CircuitState = "closed"

	// CircuitOpen means the backend has failed too many times. All requests
	// are short-circuited (rejected immediately) until the cooldown expires.
	CircuitOpen CircuitState = "open"

	// CircuitHalfOpen is the probe state after cooldown. One request is
	// allowed through. If it succeeds → closed. If it fails → open again.
	CircuitHalfOpen CircuitState = "half_open"
)

// CircuitBreaker implements the standard three-state circuit breaker pattern:
//
//	CLOSED   → (threshold consecutive failures) → OPEN
//	OPEN     → (cooldown expires)                → HALF-OPEN
//	HALF-OPEN → (probe succeeds)                 → CLOSED
//	HALF-OPEN → (probe fails)                    → OPEN (reset cooldown)
//
// Thread-safe for concurrent use across goroutines.
type CircuitBreaker struct {
	mu           sync.Mutex
	state        CircuitState
	failureCount int
	threshold    int           // consecutive failures before tripping open
	cooldown     time.Duration // how long to stay open before probing
	lastFailure  time.Time     // when the circuit last tripped open
}

// CircuitBreakerConfig holds the tuning parameters for a circuit breaker.
type CircuitBreakerConfig struct {
	Threshold int           // failures before tripping (default: 5)
	Cooldown  time.Duration // cooldown before half-open probe (default: 30s)
}

// DefaultCircuitBreakerConfig returns production-sensible defaults.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Threshold: 5,
		Cooldown:  30 * time.Second,
	}
}

// NewCircuitBreaker creates a circuit breaker in the closed (healthy) state.
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	if cfg.Threshold <= 0 {
		cfg.Threshold = 5
	}
	if cfg.Cooldown <= 0 {
		cfg.Cooldown = 30 * time.Second
	}
	return &CircuitBreaker{
		state:     CircuitClosed,
		threshold: cfg.Threshold,
		cooldown:  cfg.Cooldown,
	}
}

// Allow reports whether the circuit breaker permits a request to pass through.
//
// State transitions on Allow:
//   - CLOSED    → always returns true
//   - OPEN      → checks if cooldown has expired; if yes, transitions to
//     HALF-OPEN and returns true (one probe allowed); otherwise false
//   - HALF-OPEN → returns false (only one probe at a time; the in-flight
//     probe must complete before another is allowed)
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true

	case CircuitOpen:
		// Has the cooldown period elapsed?
		if time.Since(cb.lastFailure) >= cb.cooldown {
			cb.state = CircuitHalfOpen
			return true // allow one probe
		}
		return false

	case CircuitHalfOpen:
		// Only one probe at a time — reject until probe completes.
		return false

	default:
		return true
	}
}

// RecordSuccess records a successful operation. If the circuit was half-open
// (probe succeeded), it transitions back to closed. In all states, the
// failure counter is reset to zero.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0
	cb.state = CircuitClosed
}

// RecordFailure records a failed operation. If the failure count reaches
// the threshold, the circuit trips open and the cooldown timer starts.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++

	if cb.failureCount >= cb.threshold || cb.state == CircuitHalfOpen {
		cb.state = CircuitOpen
		cb.lastFailure = time.Now()
		cb.failureCount = 0 // reset for next cycle
	}
}

// State returns the current circuit state. Thread-safe.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Reset forces the circuit breaker back to the closed state. Useful for
// administrative overrides and testing.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = CircuitClosed
	cb.failureCount = 0
}
