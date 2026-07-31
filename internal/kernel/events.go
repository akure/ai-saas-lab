package kernel

import "sync"

// EventBus is the bulletin board that lets modules cooperate without ever
// importing each other. CompletionModule publishes "usage.recorded";
// BillingModule subscribes to it. Neither package name appears in the other.
type EventBus struct {
	mu   sync.RWMutex
	subs map[string][]func(any)
}

func NewEventBus() *EventBus {
	return &EventBus{subs: make(map[string][]func(any))}
}

func (b *EventBus) Subscribe(topic string, fn func(any)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[topic] = append(b.subs[topic], fn)
}

func (b *EventBus) Publish(topic string, payload any) {
	b.mu.RLock()
	fns := make([]func(any), len(b.subs[topic]))
	copy(fns, b.subs[topic])
	b.mu.RUnlock()

	for _, fn := range fns {
		go fn(payload) // fire concurrently — a slow subscriber never blocks the publisher
	}
}
