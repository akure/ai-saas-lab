package kernel

import "sync"

// EventBus is the bulletin board that lets modules cooperate without ever
// importing each other. CompletionModule publishes "usage.recorded";
// BillingModule subscribes to it. Neither package name appears in the other.
// As of now, we have one EventBus for all possible topics for any pub, sub.
type EventBus struct {
	mu   sync.RWMutex
	subs map[EventTopic][]func(any) // topic table (tt) - topic -> list of callback functions
}

func NewEventBus() *EventBus {
	return &EventBus{subs: make(map[EventTopic][]func(any))}
}

// Subscribe method is called by subscriber goroutines to register
// their functions against a topic they want to subscribe to. Effectively simply
// append the callback functions in the topic row of the topic table.
func (b *EventBus) Subscribe(topic EventTopic, fn func(any)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[topic] = append(b.subs[topic], fn)
}

// Publish methond to be called by publisher goroutine with its topic and payload.
// Inside the Publish itself, it will invoke new gorotuine for each of the subscribed functions,
// to ensure that a slow subscriber never blocks the publisher.
func (b *EventBus) Publish(topic EventTopic, payload any) {
	b.mu.RLock()
	fns := make([]func(any), len(b.subs[topic]))
	copy(fns, b.subs[topic])
	b.mu.RUnlock()

	for _, fn := range fns {
		go fn(payload) // fire concurrently — a slow subscriber never blocks the publisher
	}
}
