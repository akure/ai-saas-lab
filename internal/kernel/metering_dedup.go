package kernel

import (
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// EventDedup — prevents duplicate metering events from being recorded.
// ---------------------------------------------------------------------------

// EventDedup tracks seen EventIDs to prevent duplicate event processing.
// This is critical for billing correctness: WAL replays and at-least-once
// delivery semantics can produce duplicate events, and duplicate events
// mean duplicate charges.
//
// Implementation uses a time-bucketed map: each EventID is stored with its
// first-seen timestamp. A background prune cycle removes entries older than
// the retention window, bounding memory usage.
//
// Memory budget at default retention (1 hour):
//   - 10k events/sec × 3600s = 36M entries
//   - ~64 bytes per entry (string key + time.Time) ≈ 2.3 GB
//   - For higher throughput, swap to a counting bloom filter.
type EventDedup struct {
	mu        sync.RWMutex
	seen      map[string]time.Time // EventID → first-seen timestamp
	retention time.Duration        // how long to remember an EventID
	stopCh    chan struct{}         // signals the prune goroutine to stop
}

// NewEventDedup creates a deduplication tracker with the given retention
// window. A background goroutine prunes expired entries every retention/4.
func NewEventDedup(retention time.Duration) *EventDedup {
	if retention <= 0 {
		retention = time.Hour
	}
	d := &EventDedup{
		seen:      make(map[string]time.Time),
		retention: retention,
		stopCh:    make(chan struct{}),
	}
	go d.pruneLoop()
	return d
}

// IsDuplicate atomically checks whether eventID has been seen within the
// retention window. If not seen, it records the eventID and returns false.
// If already seen (duplicate), it returns true without modifying state.
//
// This is the hot-path guard — called on every RecordMeteringEvent.
func (d *EventDedup) IsDuplicate(eventID string) bool {
	if eventID == "" {
		return false // events without IDs are never considered duplicates
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.seen[eventID]; exists {
		return true
	}
	d.seen[eventID] = time.Now()
	return false
}

// Prune removes all entries older than the retention window. Called
// automatically by the background prune goroutine, but safe to call
// manually (e.g., in tests).
func (d *EventDedup) Prune() {
	d.mu.Lock()
	defer d.mu.Unlock()

	cutoff := time.Now().Add(-d.retention)
	for id, ts := range d.seen {
		if ts.Before(cutoff) {
			delete(d.seen, id)
		}
	}
}

// Len returns the number of currently tracked EventIDs. Useful for
// health reporting and observability.
func (d *EventDedup) Len() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.seen)
}

// Stop terminates the background prune goroutine. Call during shutdown.
func (d *EventDedup) Stop() {
	select {
	case <-d.stopCh:
		// already stopped
	default:
		close(d.stopCh)
	}
}

// pruneLoop runs in a background goroutine, pruning expired entries
// at an interval of retention/4.
func (d *EventDedup) pruneLoop() {
	interval := d.retention / 4
	if interval < time.Second {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.Prune()
		case <-d.stopCh:
			return
		}
	}
}
