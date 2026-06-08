package sse

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeSubscriberBus struct {
	published      chan Event
	probeErr       error
	subscribeErr   error
	subscribeCalls int
	mu             sync.Mutex
}

type blockingProbeBus struct {
	fakeSubscriberBus
	probeStarted chan struct{}
	releaseProbe chan struct{}
	probeErr     error
	once         sync.Once
}

func newBlockingProbeBus(probeErr error) *blockingProbeBus {
	return &blockingProbeBus{
		fakeSubscriberBus: fakeSubscriberBus{published: make(chan Event, 1)},
		probeStarted:      make(chan struct{}),
		releaseProbe:      make(chan struct{}),
		probeErr:          probeErr,
	}
}

func (bus *blockingProbeBus) Probe(ctx context.Context) error {
	bus.once.Do(func() { close(bus.probeStarted) })
	select {
	case <-bus.releaseProbe:
		return bus.probeErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (bus *fakeSubscriberBus) Probe(ctx context.Context) error {
	return bus.probeErr
}

func (bus *fakeSubscriberBus) Publish(ctx context.Context, event Event) error {
	select {
	case bus.published <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (bus *fakeSubscriberBus) Subscribe(ctx context.Context, handler func(Event)) error {
	bus.mu.Lock()
	bus.subscribeCalls++
	bus.mu.Unlock()

	if bus.subscribeErr != nil {
		return bus.subscribeErr
	}

	for {
		select {
		case event := <-bus.published:
			handler(event)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (bus *fakeSubscriberBus) calls() int {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	return bus.subscribeCalls
}

func TestSubscriberDeliversBusEventsToLocalHub(t *testing.T) {
	hub := NewHub(2)
	connection := hub.Add(Identity{SessionID: "sid-831", UserID: "831"})

	bus := &fakeSubscriberBus{published: make(chan Event, 1)}
	registry := NewSubscriptionRegistry(hub)
	t.Cleanup(func() { registry.Stop("redis") })

	if ok := registry.Ensure(context.Background(), "redis", bus); !ok {
		t.Fatal("Ensure returned false")
	}

	want := Event{UserID: "831", Event: "message", Data: "hello"}
	if err := bus.Publish(context.Background(), want); err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	select {
	case got := <-connection.Events():
		if got.UserID != want.UserID {
			t.Fatalf("UserID = %q, want %q", got.UserID, want.UserID)
		}
		if got.Event != want.Event {
			t.Fatalf("Event = %q, want %q", got.Event, want.Event)
		}
		if got.Data != want.Data {
			t.Fatalf("Data = %#v, want %#v", got.Data, want.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("expected bus event")
	}
}

func TestSubscriptionRegistryStartsOneSubscriberPerKey(t *testing.T) {
	hub := NewHub(2)
	bus := &fakeSubscriberBus{published: make(chan Event, 1)}
	registry := NewSubscriptionRegistry(hub)
	t.Cleanup(func() { registry.Stop("redis") })

	if ok := registry.Ensure(context.Background(), "redis", bus); !ok {
		t.Fatal("first Ensure returned false")
	}

	eventually(t, time.Second, func() bool {
		return bus.calls() == 1
	})

	if ok := registry.Ensure(context.Background(), "redis", bus); !ok {
		t.Fatal("second Ensure returned false")
	}
	time.Sleep(20 * time.Millisecond)

	if got := bus.calls(); got != 1 {
		t.Fatalf("subscribeCalls = %d, want 1", got)
	}
}

func TestSubscriptionRegistryMarksUnavailableWhenProbeFails(t *testing.T) {
	hub := NewHub(2)
	bus := &fakeSubscriberBus{
		published: make(chan Event, 1),
		probeErr:  errors.New("probe failed"),
	}
	registry := NewSubscriptionRegistry(hub)

	if ok := registry.Ensure(context.Background(), "redis", bus); ok {
		t.Fatal("Ensure returned true")
	}
	if registry.IsAvailable("redis") {
		t.Fatal("registry is available")
	}
}

func TestSubscriptionRegistryMarksUnavailableWhenSubscribeFails(t *testing.T) {
	hub := NewHub(2)
	bus := &fakeSubscriberBus{
		published:    make(chan Event, 1),
		subscribeErr: errors.New("subscribe failed"),
	}
	registry := NewSubscriptionRegistry(hub)

	if ok := registry.Ensure(context.Background(), "redis", bus); !ok {
		t.Fatal("Ensure returned false")
	}

	eventually(t, time.Second, func() bool {
		return !registry.IsAvailable("redis")
	})
}

func TestSubscriptionRegistryIgnoresStaleProbeFailureAfterReplacement(t *testing.T) {
	hub := NewHub(2)
	staleBus := newBlockingProbeBus(errors.New("stale probe failed"))
	replacementBus := &fakeSubscriberBus{published: make(chan Event, 1)}
	registry := NewSubscriptionRegistry(hub)
	t.Cleanup(func() { registry.Stop("redis") })

	done := make(chan bool, 1)
	go func() {
		done <- registry.Ensure(context.Background(), "redis", staleBus)
	}()

	select {
	case <-staleBus.probeStarted:
	case <-time.After(time.Second):
		t.Fatal("stale probe did not start")
	}

	if ok := registry.Ensure(context.Background(), "redis", replacementBus); !ok {
		t.Fatal("replacement Ensure returned false")
	}
	if !registry.IsAvailable("redis") {
		t.Fatal("replacement subscription is unavailable before stale probe returns")
	}

	close(staleBus.releaseProbe)
	select {
	case ok := <-done:
		if ok {
			t.Fatal("stale Ensure returned true")
		}
	case <-time.After(time.Second):
		t.Fatal("stale Ensure did not finish")
	}

	if !registry.IsAvailable("redis") {
		t.Fatal("stale probe failure changed replacement subscription availability")
	}
}

func eventually(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("condition was not met before timeout")
}
