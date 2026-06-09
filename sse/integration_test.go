package sse

import (
	"bufio"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	gouapi "github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/session"
)

func TestInitRegistersPublishProcess(t *testing.T) {
	resetInitForTest()
	t.Cleanup(resetInitForTest)

	Init()

	if !process.Exists("utils.sse.Publish") {
		t.Fatal("utils.sse.Publish was not registered")
	}
}

func TestInitRegistersAuthenticatedSSEFactory(t *testing.T) {
	resetInitForTest()
	t.Cleanup(resetInitForTest)

	bus := &fakeSubscriberBus{published: make(chan Event, 1)}
	SetBusFactoryForTest(func(connector string, channel string) Bus { return bus })
	Init()

	loaded := loadAuthenticatedSSEForTest(t, "unit.authenticated.sse")
	sid := session.ID()
	if err := session.Global().ID(sid).Set("user_id", 831); err != nil {
		t.Fatalf("set session user_id: %v", err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("__sid", sid)
		c.Next()
	})
	loaded.HTTP.Routes(router, "/")

	server := httptest.NewServer(router)
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, sseURL(server.URL, "unit.authenticated.sse"), nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request sse: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusServiceUnavailable {
		t.Fatal("authenticated sse returned 503")
	}
	if got := resp.Header.Get("Content-Type"); got != "text/event-stream; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
}

func TestInitFactoryReturns503WhenBusUnavailable(t *testing.T) {
	resetInitForTest()
	t.Cleanup(resetInitForTest)

	bus := &fakeSubscriberBus{published: make(chan Event, 1), probeErr: errors.New("redis down")}
	SetBusFactoryForTest(func(connector string, channel string) Bus { return bus })
	Init()

	loaded := loadAuthenticatedSSEForTest(t, "unit.authenticated.sse.unavailable")
	sid := session.ID()
	if err := session.Global().ID(sid).Set("user_id", 831); err != nil {
		t.Fatalf("set session user_id: %v", err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("__sid", sid)
		c.Next()
	})
	loaded.HTTP.Routes(router, "/")

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/unit/authenticated/sse/unavailable/events", nil)
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusServiceUnavailable)
	}
}

func TestInitPublishProcessDeliversToAuthenticatedSSEClient(t *testing.T) {
	resetInitForTest()
	t.Cleanup(resetInitForTest)

	bus := &fakeSubscriberBus{published: make(chan Event, 1)}
	SetBusFactoryForTest(func(connector string, channel string) Bus { return bus })
	Init()

	loaded := loadAuthenticatedSSEForTest(t, "unit.authenticated.sse.publish")
	sid := session.ID()
	if err := session.Global().ID(sid).Set("user_id", 831); err != nil {
		t.Fatalf("set session user_id: %v", err)
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("__sid", sid)
		c.Next()
	})
	loaded.HTTP.Routes(router, "/")

	server := httptest.NewServer(router)
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sseURL(server.URL, "unit.authenticated.sse.publish"), nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request sse: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	reader := bufio.NewReader(resp.Body)
	connected := readSSEEventForTest(t, reader)
	if connected.event != "connected" {
		t.Fatalf("first event = %q, want connected", connected.event)
	}

	result, ok := process.New("utils.sse.Publish", map[string]interface{}{
		"userId": 831,
		"event":  "probe.message",
		"data": map[string]interface{}{
			"source": "integration-test",
		},
	}).Run().(PublishResult)
	if !ok {
		t.Fatalf("publish returned %T, want PublishResult", result)
	}
	if !result.OK || !result.Published {
		t.Fatalf("publish result = %#v, want ok and published", result)
	}

	message := readSSEEventForTest(t, reader)
	if message.event != "probe.message" {
		t.Fatalf("event = %q, want probe.message", message.event)
	}
	if !strings.Contains(message.data, "integration-test") {
		t.Fatalf("data = %q, want integration-test marker", message.data)
	}
}

func loadAuthenticatedSSEForTest(t *testing.T, id string) *gouapi.API {
	t.Helper()
	source := []byte(`{
		"name": "unit authenticated sse",
		"paths": [
			{
				"path": "/events",
				"method": "GET",
				"guard": "-",
				"out": { "type": "text/event-stream" },
				"sse": { "type": "authenticated", "adapter": "notification" }
			}
		]
	}`)

	loaded, err := gouapi.LoadSource("<"+id+">.yao", source, id)
	if err != nil {
		t.Fatalf("load api: %v", err)
	}
	t.Cleanup(func() { delete(gouapi.APIs, id) })
	return loaded
}

func sseURL(serverURL, id string) string {
	return serverURL + "/" + strings.ReplaceAll(id, ".", "/") + "/events"
}

func resetInitForTest() {
	initMu.Lock()
	initialized = false
	globalHub = NewHub(defaultQueueSize)
	globalSubscriptions = NewSubscriptionRegistry(globalHub)
	initMu.Unlock()
	SetBusForTest(nil)
	SetBusFactoryForTest(nil)
	gouapi.ResetSSEHandlerFactoryForTest()
}

type sseEventForTest struct {
	event string
	data  string
}

func readSSEEventForTest(t *testing.T, reader *bufio.Reader) sseEventForTest {
	t.Helper()

	type result struct {
		event sseEventForTest
		err   error
	}
	done := make(chan result, 1)
	go func() {
		var event sseEventForTest
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				done <- result{err: err}
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				done <- result{event: event}
				return
			}
			if strings.HasPrefix(line, "event: ") {
				event.event = strings.TrimPrefix(line, "event: ")
				continue
			}
			if strings.HasPrefix(line, "data: ") {
				if event.data != "" {
					event.data += "\n"
				}
				event.data += strings.TrimPrefix(line, "data: ")
			}
		}
	}()

	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("read sse event: %v", got.err)
		}
		return got.event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for sse event")
		return sseEventForTest{}
	}
}
