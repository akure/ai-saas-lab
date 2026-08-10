package kernel

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestEventBus_PublishFanOutAndPayloadDelivery(t *testing.T) {
	bus := NewEventBus()

	const topic = TopicUsageRecorded
	const payload = "token-used"

	subscribers := []string{"completion", "billing"}
	results := make(chan string, len(subscribers))
	var wg sync.WaitGroup

	for _, name := range subscribers {
		wg.Add(1)
		subscriberName := name

		bus.Subscribe(topic, func(data any) {
			defer wg.Done()

			actual, ok := data.(string)
			if !ok {
				t.Errorf("subscriber %s received unexpected payload type %T", subscriberName, data)
				return
			}

			results <- fmt.Sprintf("%s:%s", subscriberName, actual)
		})
	}

	bus.Publish(topic, payload)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for subscribers to handle the published event")
	}

	close(results)

	var received []string
	for item := range results {
		received = append(received, item)
	}

	if len(received) != len(subscribers) {
		t.Fatalf("expected %d deliveries, got %d: %v", len(subscribers), len(received), received)
	}

	for _, name := range subscribers {
		expected := fmt.Sprintf("%s:%s", name, payload)
		found := false
		for _, item := range received {
			if item == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("subscriber %s did not receive the expected payload; received %v", name, received)
		}
	}
}

func TestEventBus_DoesNotDeliverToOtherTopics(t *testing.T) {
	bus := NewEventBus()

	const topic = TopicUsageRecorded
	const payload = "token-used"

	received := make(chan string, 1)
	bus.Subscribe(topic, func(data any) {
		received <- data.(string)
	})

	bus.Publish(TopicSubscriptionUpdated, payload)

	select {
	case item := <-received:
		t.Fatalf("expected no delivery for a different topic, got %q", item)
	case <-time.After(200 * time.Millisecond):
	}
}
