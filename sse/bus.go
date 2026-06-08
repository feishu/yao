package sse

import (
	"context"
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
