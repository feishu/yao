package sse

import (
	"context"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	gouapi "github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/process"
)

var (
	initMu              sync.Mutex
	initialized         bool
	globalHub           = NewHub(defaultQueueSize)
	globalSubscriptions = NewSubscriptionRegistry(globalHub)
	busFactory          = func(connector string, channel string) Bus {
		return NewRedisBus(connector, channel)
	}
)

func SetBusFactoryForTest(factory func(connector string, channel string) Bus) {
	initMu.Lock()
	defer initMu.Unlock()

	if factory == nil {
		busFactory = func(connector string, channel string) Bus {
			return NewRedisBus(connector, channel)
		}
		return
	}
	busFactory = factory
}

func Init() {
	initMu.Lock()
	defer initMu.Unlock()

	if initialized {
		return
	}

	SetBus(NewRedisBus(defaultBusConnector, defaultBusChannel))
	process.Register("utils.sse.Publish", ProcessPublish)
	gouapi.SetSSEHandlerFactory(func(path gouapi.Path) gin.HandlerFunc {
		if path.SSE == nil || strings.ToLower(path.SSE.Type) != "authenticated" {
			return nil
		}

		connector := path.SSE.Bus.Connector
		if connector == "" {
			connector = defaultBusConnector
		}

		channel := path.SSE.Bus.Channel
		if channel == "" {
			channel = defaultBusChannel
		}

		key := connector + "/" + channel
		bus := busFactory(connector, channel)
		SetBus(bus)
		globalSubscriptions.Ensure(context.Background(), key, bus)

		return NewHandler(HandlerOptions{
			Hub:              globalHub,
			HeartbeatSeconds: path.SSE.Heartbeat,
			Available: func() bool {
				return globalSubscriptions.Ensure(context.Background(), key, bus) && globalSubscriptions.IsAvailable(key)
			},
		})
	})

	initialized = true
}
