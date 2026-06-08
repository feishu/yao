package sse

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
