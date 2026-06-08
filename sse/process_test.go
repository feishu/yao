package sse

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/yaoapp/gou/process"
)

type fakeBus struct {
	event      Event
	publishErr error
	published  bool
}

func (bus *fakeBus) Probe(ctx context.Context) error {
	return nil
}

func (bus *fakeBus) Publish(ctx context.Context, event Event) error {
	bus.event = event
	bus.published = true
	return bus.publishErr
}

func (bus *fakeBus) Subscribe(ctx context.Context, handler func(Event)) error {
	return nil
}

func TestProcessPublishSubmitsEventToBus(t *testing.T) {
	bus := &fakeBus{}
	SetBusForTest(bus)
	t.Cleanup(func() { SetBusForTest(nil) })

	data := map[string]interface{}{"hello": "world"}
	proc := process.New("utils.sse.Publish", map[string]interface{}{
		"userId": 831,
		"event":  "message",
		"data":   data,
	})

	result, ok := ProcessPublish(proc).(PublishResult)
	if !ok {
		t.Fatalf("ProcessPublish returned %T, want PublishResult", result)
	}
	if !result.OK || !result.Published {
		t.Fatalf("ProcessPublish returned %#v, want ok and published", result)
	}
	if !bus.published {
		t.Fatal("bus.Publish was not called")
	}
	if bus.event.UserID != "831" {
		t.Fatalf("UserID = %q, want %q", bus.event.UserID, "831")
	}
	if bus.event.Event != "message" {
		t.Fatalf("Event = %q, want %q", bus.event.Event, "message")
	}
	if bus.event.ClientID != "" {
		t.Fatalf("ClientID = %q, want empty string", bus.event.ClientID)
	}
	if !reflect.DeepEqual(bus.event.Data, data) {
		t.Fatalf("Data = %#v, want %#v", bus.event.Data, data)
	}
}

func TestProcessPublishKeepsOptionalClientIDEmpty(t *testing.T) {
	bus := &fakeBus{}
	SetBusForTest(bus)
	t.Cleanup(func() { SetBusForTest(nil) })

	proc := process.New("utils.sse.Publish", map[string]interface{}{
		"userId": "831",
		"event":  "message",
	})

	result := ProcessPublish(proc).(PublishResult)
	if !result.OK || !result.Published {
		t.Fatalf("ProcessPublish returned %#v, want ok and published", result)
	}
	if bus.event.ClientID != "" {
		t.Fatalf("ClientID = %q, want empty string", bus.event.ClientID)
	}
}

func TestProcessPublishAcceptsCommonNumericUserIDTypes(t *testing.T) {
	tests := []struct {
		name   string
		userID interface{}
		want   string
	}{
		{name: "int32", userID: int32(831), want: "831"},
		{name: "uint", userID: uint(831), want: "831"},
		{name: "float32", userID: float32(831.5), want: "831.5"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bus := &fakeBus{}
			SetBusForTest(bus)
			t.Cleanup(func() { SetBusForTest(nil) })

			proc := process.New("utils.sse.Publish", map[string]interface{}{
				"userId": test.userID,
				"event":  "message",
			})

			result := ProcessPublish(proc).(PublishResult)
			if !result.OK || !result.Published {
				t.Fatalf("ProcessPublish returned %#v, want ok and published", result)
			}
			if bus.event.UserID != test.want {
				t.Fatalf("UserID = %q, want %q", bus.event.UserID, test.want)
			}
		})
	}
}

func TestProcessPublishUsesClientIDWhenProvided(t *testing.T) {
	bus := &fakeBus{}
	SetBusForTest(bus)
	t.Cleanup(func() { SetBusForTest(nil) })

	proc := process.New("utils.sse.Publish", map[string]interface{}{
		"userId":   "831",
		"event":    "message",
		"clientId": "client-1",
	})

	result := ProcessPublish(proc).(PublishResult)
	if !result.OK || !result.Published {
		t.Fatalf("ProcessPublish returned %#v, want ok and published", result)
	}
	if bus.event.ClientID != "client-1" {
		t.Fatalf("ClientID = %q, want %q", bus.event.ClientID, "client-1")
	}
}

func TestProcessPublishRejectsReservedEvent(t *testing.T) {
	proc := process.New("utils.sse.Publish", map[string]interface{}{
		"userId": "831",
		"event":  "heartbeat",
	})

	result := ProcessPublish(proc).(PublishResult)
	if result.OK {
		t.Fatalf("ProcessPublish returned %#v, want not ok", result)
	}
	if result.Code != "INVALID_EVENT" {
		t.Fatalf("Code = %q, want %q", result.Code, "INVALID_EVENT")
	}
}

func TestProcessPublishRejectsNonStringEvent(t *testing.T) {
	bus := &fakeBus{}
	SetBusForTest(bus)
	t.Cleanup(func() { SetBusForTest(nil) })

	proc := process.New("utils.sse.Publish", map[string]interface{}{
		"userId": "831",
		"event":  true,
	})

	result := ProcessPublish(proc).(PublishResult)
	if result.OK {
		t.Fatalf("ProcessPublish returned %#v, want not ok", result)
	}
	if result.Code != "INVALID_EVENT" {
		t.Fatalf("Code = %q, want %q", result.Code, "INVALID_EVENT")
	}
	if bus.published {
		t.Fatal("bus.Publish was called")
	}
}

func TestProcessPublishRejectsInvalidUserIDType(t *testing.T) {
	bus := &fakeBus{}
	SetBusForTest(bus)
	t.Cleanup(func() { SetBusForTest(nil) })

	proc := process.New("utils.sse.Publish", map[string]interface{}{
		"userId": true,
		"event":  "message",
	})

	result := ProcessPublish(proc).(PublishResult)
	if result.OK {
		t.Fatalf("ProcessPublish returned %#v, want not ok", result)
	}
	if result.Code != "INVALID_USER" {
		t.Fatalf("Code = %q, want %q", result.Code, "INVALID_USER")
	}
	if bus.published {
		t.Fatal("bus.Publish was called")
	}
}

func TestProcessPublishRejectsNaNUserID(t *testing.T) {
	bus := &fakeBus{}
	SetBusForTest(bus)
	t.Cleanup(func() { SetBusForTest(nil) })

	proc := process.New("utils.sse.Publish", map[string]interface{}{
		"userId": math.NaN(),
		"event":  "message",
	})

	result := ProcessPublish(proc).(PublishResult)
	if result.OK {
		t.Fatalf("ProcessPublish returned %#v, want not ok", result)
	}
	if result.Code != "INVALID_USER" {
		t.Fatalf("Code = %q, want %q", result.Code, "INVALID_USER")
	}
	if bus.published {
		t.Fatal("bus.Publish was called")
	}
}

func TestProcessPublishRejectsInvalidClientIDType(t *testing.T) {
	bus := &fakeBus{}
	SetBusForTest(bus)
	t.Cleanup(func() { SetBusForTest(nil) })

	proc := process.New("utils.sse.Publish", map[string]interface{}{
		"userId":   "831",
		"event":    "message",
		"clientId": true,
	})

	result := ProcessPublish(proc).(PublishResult)
	if result.OK {
		t.Fatalf("ProcessPublish returned %#v, want not ok", result)
	}
	if result.Code != "INVALID_INPUT" {
		t.Fatalf("Code = %q, want %q", result.Code, "INVALID_INPUT")
	}
	if result.Message != "clientId must be a string" {
		t.Fatalf("Message = %q, want %q", result.Message, "clientId must be a string")
	}
	if bus.published {
		t.Fatal("bus.Publish was called")
	}
}

func TestProcessPublishReturnsPubSubUnavailable(t *testing.T) {
	bus := &fakeBus{publishErr: errors.New("publish failed")}
	SetBusForTest(bus)
	t.Cleanup(func() { SetBusForTest(nil) })

	proc := process.New("utils.sse.Publish", map[string]interface{}{
		"userId": "831",
		"event":  "message",
	})

	result := ProcessPublish(proc).(PublishResult)
	if result.OK {
		t.Fatalf("ProcessPublish returned %#v, want not ok", result)
	}
	if result.Code != "PUBSUB_UNAVAILABLE" {
		t.Fatalf("Code = %q, want %q", result.Code, "PUBSUB_UNAVAILABLE")
	}
	if result.Message != "Redis Pub/Sub publish failed" {
		t.Fatalf("Message = %q, want %q", result.Message, "Redis Pub/Sub publish failed")
	}
}

func TestProcessPublishReturnsPubSubUnavailableWhenBusIsNil(t *testing.T) {
	SetBusForTest(nil)
	t.Cleanup(func() { SetBusForTest(nil) })

	proc := process.New("utils.sse.Publish", map[string]interface{}{
		"userId": "831",
		"event":  "message",
	})

	result := ProcessPublish(proc).(PublishResult)
	if result.OK {
		t.Fatalf("ProcessPublish returned %#v, want not ok", result)
	}
	if result.Code != "PUBSUB_UNAVAILABLE" {
		t.Fatalf("Code = %q, want %q", result.Code, "PUBSUB_UNAVAILABLE")
	}
	if result.Message != "Redis Pub/Sub publish failed" {
		t.Fatalf("Message = %q, want %q", result.Message, "Redis Pub/Sub publish failed")
	}
}
