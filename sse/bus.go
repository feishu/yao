package sse

import (
	"context"
	"errors"
	"sync"
)

type Bus interface {
	Probe(ctx context.Context) error
	Publish(ctx context.Context, event Event) error
	Subscribe(ctx context.Context, handler func(Event)) error
}

var (
	busMu sync.RWMutex
	bus   Bus
)

func SetBus(next Bus) {
	busMu.Lock()
	defer busMu.Unlock()

	bus = next
}

func SetBusForTest(next Bus) {
	SetBus(next)
}

func currentBus() Bus {
	busMu.RLock()
	defer busMu.RUnlock()

	return bus
}

type SubscriptionRegistry struct {
	hub           *Hub
	subscriptions map[string]*subscriptionState
	mu            sync.Mutex
}

type subscriptionState struct {
	cancel    context.CancelFunc
	available bool
}

func NewSubscriptionRegistry(hub *Hub) *SubscriptionRegistry {
	return &SubscriptionRegistry{
		hub:           hub,
		subscriptions: map[string]*subscriptionState{},
	}
}

func (registry *SubscriptionRegistry) Ensure(ctx context.Context, key string, bus Bus) bool {
	registry.mu.Lock()
	if state, ok := registry.subscriptions[key]; ok && state.available {
		registry.mu.Unlock()
		return true
	}
	if state, ok := registry.subscriptions[key]; ok {
		state.cancel()
		delete(registry.subscriptions, key)
	}

	subCtx, cancel := context.WithCancel(ctx)
	state := &subscriptionState{cancel: cancel}
	registry.subscriptions[key] = state
	registry.mu.Unlock()

	if bus == nil {
		registry.stopIfCurrent(key, state)
		return false
	}

	if err := bus.Probe(ctx); err != nil {
		registry.stopIfCurrent(key, state)
		return false
	}

	if !registry.markAvailable(key, state) {
		cancel()
		return false
	}
	go func() {
		err := bus.Subscribe(subCtx, func(event Event) {
			registry.hub.Deliver(event)
		})
		if err != nil && !errors.Is(err, context.Canceled) && subCtx.Err() == nil {
			registry.markUnavailable(key, state)
		}
	}()

	return true
}

func (registry *SubscriptionRegistry) IsAvailable(key string) bool {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	state, ok := registry.subscriptions[key]
	return ok && state.available
}

func (registry *SubscriptionRegistry) Stop(key string) {
	registry.mu.Lock()
	state, ok := registry.subscriptions[key]
	if ok {
		delete(registry.subscriptions, key)
	}
	registry.mu.Unlock()

	if ok {
		state.cancel()
	}
}

func (registry *SubscriptionRegistry) markAvailable(key string, state *subscriptionState) bool {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	if registry.subscriptions[key] != state {
		return false
	}
	state.available = true
	return true
}

func (registry *SubscriptionRegistry) markUnavailable(key string, state *subscriptionState) {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	if registry.subscriptions[key] == state {
		state.available = false
	}
}

func (registry *SubscriptionRegistry) stopIfCurrent(key string, state *subscriptionState) {
	registry.mu.Lock()
	if registry.subscriptions[key] != state {
		registry.mu.Unlock()
		return
	}
	delete(registry.subscriptions, key)
	registry.mu.Unlock()

	if state != nil {
		state.cancel()
	}
}
