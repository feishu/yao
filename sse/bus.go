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
	subscriptions map[string]context.CancelFunc
	available     map[string]bool
	mu            sync.Mutex
}

func NewSubscriptionRegistry(hub *Hub) *SubscriptionRegistry {
	return &SubscriptionRegistry{
		hub:           hub,
		subscriptions: map[string]context.CancelFunc{},
		available:     map[string]bool{},
	}
}

func (registry *SubscriptionRegistry) Ensure(ctx context.Context, key string, bus Bus) bool {
	registry.mu.Lock()
	if registry.available[key] {
		registry.mu.Unlock()
		return true
	}
	if cancel, ok := registry.subscriptions[key]; ok {
		cancel()
		delete(registry.subscriptions, key)
		delete(registry.available, key)
	}

	subCtx, cancel := context.WithCancel(ctx)
	registry.subscriptions[key] = cancel
	registry.available[key] = false
	registry.mu.Unlock()

	if bus == nil {
		registry.Stop(key)
		return false
	}

	if err := bus.Probe(ctx); err != nil {
		registry.markUnavailable(key)
		registry.Stop(key)
		return false
	}

	registry.markAvailable(key)
	go func() {
		err := bus.Subscribe(subCtx, func(event Event) {
			registry.hub.Deliver(event)
		})
		if err != nil && !errors.Is(err, context.Canceled) && subCtx.Err() == nil {
			registry.markUnavailable(key)
		}
	}()

	return true
}

func (registry *SubscriptionRegistry) IsAvailable(key string) bool {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	return registry.available[key]
}

func (registry *SubscriptionRegistry) Stop(key string) {
	registry.mu.Lock()
	cancel, ok := registry.subscriptions[key]
	if ok {
		delete(registry.subscriptions, key)
	}
	delete(registry.available, key)
	registry.mu.Unlock()

	if ok {
		cancel()
	}
}

func (registry *SubscriptionRegistry) markAvailable(key string) {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	if _, ok := registry.subscriptions[key]; ok {
		registry.available[key] = true
	}
}

func (registry *SubscriptionRegistry) markUnavailable(key string) {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	if _, ok := registry.subscriptions[key]; ok {
		registry.available[key] = false
	}
}
