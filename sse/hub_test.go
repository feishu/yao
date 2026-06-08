package sse

import (
	"sync"
	"testing"
	"time"
)

func TestHubDeliverToUserConnection(t *testing.T) {
	hub := NewHub(2)
	first := hub.Add(Identity{SessionID: "sid-831-a", UserID: "831"})
	second := hub.Add(Identity{SessionID: "sid-831-b", UserID: "831"})

	delivered := hub.Deliver(Event{UserID: "831", Event: "message", Data: "hello"})
	if delivered != 2 {
		t.Fatalf("Deliver = %d, want 2", delivered)
	}

	for _, connection := range []*Connection{first, second} {
		select {
		case event := <-connection.Events():
			if event.UserID != "831" {
				t.Fatalf("UserID = %q, want %q", event.UserID, "831")
			}
			if event.Event != "message" {
				t.Fatalf("Event = %q, want %q", event.Event, "message")
			}
			if event.Data != "hello" {
				t.Fatalf("Data = %v, want %q", event.Data, "hello")
			}
		case <-time.After(time.Second):
			t.Fatal("expected event")
		}
	}
}

func TestHubDoesNotDeliverToOtherUsers(t *testing.T) {
	hub := NewHub(2)
	connection := hub.Add(Identity{SessionID: "sid-831", UserID: "831"})

	delivered := hub.Deliver(Event{UserID: "832", Event: "message", Data: "hello"})
	if delivered != 0 {
		t.Fatalf("Deliver = %d, want 0", delivered)
	}

	select {
	case event := <-connection.Events():
		t.Fatalf("unexpected event: %#v", event)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestHubRemoveClosesConnectionAndCleansCount(t *testing.T) {
	hub := NewHub(2)
	connection := hub.Add(Identity{SessionID: "sid-831", UserID: "831"})

	if count := hub.UserConnectionCount("831"); count != 1 {
		t.Fatalf("UserConnectionCount = %d, want 1", count)
	}

	hub.Remove(connection.ID())

	if count := hub.UserConnectionCount("831"); count != 0 {
		t.Fatalf("UserConnectionCount = %d, want 0", count)
	}

	_, ok := <-connection.Events()
	if ok {
		t.Fatal("connection events channel is still open")
	}
}

func TestHubClosesSlowConnectionWhenQueueFull(t *testing.T) {
	hub := NewHub(1)
	connection := hub.Add(Identity{SessionID: "sid-831", UserID: "831"})

	delivered := hub.Deliver(Event{UserID: "831", Event: "message", Data: "first"})
	if delivered != 1 {
		t.Fatalf("first Deliver = %d, want 1", delivered)
	}

	delivered = hub.Deliver(Event{UserID: "831", Event: "message", Data: "second"})
	if delivered != 0 {
		t.Fatalf("second Deliver = %d, want 0", delivered)
	}

	if count := hub.UserConnectionCount("831"); count != 0 {
		t.Fatalf("UserConnectionCount = %d, want 0", count)
	}

	event, ok := <-connection.Events()
	if !ok {
		t.Fatal("expected queued event before close")
	}
	if event.Data != "first" {
		t.Fatalf("queued Data = %v, want %q", event.Data, "first")
	}

	_, ok = <-connection.Events()
	if ok {
		t.Fatal("connection events channel is still open")
	}
}

func TestHubDeliverAndRemoveConcurrent(t *testing.T) {
	hub := NewHub(64)
	connection := hub.Add(Identity{SessionID: "sid-831", UserID: "831"})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			hub.Deliver(Event{UserID: "831", Event: "message", Data: "hello"})
		}()
		go func() {
			defer wg.Done()
			hub.Remove(connection.ID())
		}()
	}

	wg.Wait()
}
