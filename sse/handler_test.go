package sse

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/session"
)

func TestHandlerConnectsAuthenticatedSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hub := NewHub(2)
	sid := session.ID()
	if err := session.Global().ID(sid).Set("user_id", "831"); err != nil {
		t.Fatalf("set user_id: %v", err)
	}

	server := newHandlerTestServer(sid, NewHandler(HandlerOptions{
		Hub:              hub,
		HeartbeatSeconds: 60,
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/sse", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /sse: %v", err)
	}
	defer func() {
		cancel()
		resp.Body.Close()
		waitUntil(t, time.Second, func() bool {
			return hub.UserConnectionCount("831") == 0
		}, "connection cleanup")
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if contentType := resp.Header.Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", contentType)
	}

	reader := bufio.NewReader(resp.Body)
	eventLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read event line: %v", err)
	}
	if eventLine != "event: connected\n" {
		t.Fatalf("event line = %q, want %q", eventLine, "event: connected\n")
	}

	dataLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read data line: %v", err)
	}
	if !strings.HasPrefix(dataLine, "data: ") {
		t.Fatalf("data line = %q, want data prefix", dataLine)
	}

	var data map[string]interface{}
	payload := strings.TrimSpace(strings.TrimPrefix(dataLine, "data: "))
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		t.Fatalf("decode connected data: %v", err)
	}
	if data["clientId"] == "" {
		t.Fatalf("clientId = %v, want non-empty", data["clientId"])
	}

	waitUntil(t, time.Second, func() bool {
		return hub.UserConnectionCount("831") == 1
	}, "authenticated connection")
}

func TestHandlerReturnsUnauthorizedWithoutIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := newHandlerTestServer("", NewHandler(HandlerOptions{Hub: NewHub(2)}))
	defer server.Close()

	resp, err := server.Client().Get(server.URL + "/sse")
	if err != nil {
		t.Fatalf("GET /sse: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestHandlerReturnsUnavailableWhenDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := newHandlerTestServer("", NewHandler(HandlerOptions{
		Hub: NewHub(2),
		Available: func() bool {
			return false
		},
	}))
	defer server.Close()

	resp, err := server.Client().Get(server.URL + "/sse")
	if err != nil {
		t.Fatalf("GET /sse: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestHandlerRemovesConnectionAfterRequestContextCanceled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hub := NewHub(2)
	sid := session.ID()
	if err := session.Global().ID(sid).Set("user_id", "832"); err != nil {
		t.Fatalf("set user_id: %v", err)
	}

	server := newHandlerTestServer(sid, NewHandler(HandlerOptions{
		Hub:              hub,
		HeartbeatSeconds: 60,
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/sse", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /sse: %v", err)
	}

	waitUntil(t, time.Second, func() bool {
		return hub.UserConnectionCount("832") == 1
	}, "authenticated connection")

	cancel()
	resp.Body.Close()

	waitUntil(t, time.Second, func() bool {
		return hub.UserConnectionCount("832") == 0
	}, "connection cleanup")
}

func newHandlerTestServer(sid string, handler gin.HandlerFunc) *httptest.Server {
	router := gin.New()
	router.GET("/sse", func(c *gin.Context) {
		if sid != "" {
			c.Set("__sid", sid)
		}
		handler(c)
	})
	return httptest.NewServer(router)
}

func waitUntil(t *testing.T, timeout time.Duration, condition func() bool, label string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", label)
}

func TestFlushWriterReturnsFlushError(t *testing.T) {
	want := errors.New("flush failed")
	writer := flushErrorResponseWriter{err: want}

	if err := flushWriter(writer); !errors.Is(err, want) {
		t.Fatalf("flushWriter error = %v, want %v", err, want)
	}
}

func TestFlushWriterReturnsUnwrappedFlushError(t *testing.T) {
	want := errors.New("flush failed")
	writer := flushWrapperResponseWriter{
		ResponseWriter: flushErrorResponseWriter{err: want},
	}

	if err := flushWriter(writer); !errors.Is(err, want) {
		t.Fatalf("flushWriter error = %v, want %v", err, want)
	}
}

type flushErrorResponseWriter struct {
	err error
}

func (writer flushErrorResponseWriter) Header() http.Header {
	return http.Header{}
}

func (writer flushErrorResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (writer flushErrorResponseWriter) WriteHeader(int) {}

func (writer flushErrorResponseWriter) FlushError() error {
	return writer.err
}

type flushWrapperResponseWriter struct {
	http.ResponseWriter
}

func (writer flushWrapperResponseWriter) Flush() {}

func (writer flushWrapperResponseWriter) Unwrap() http.ResponseWriter {
	return writer.ResponseWriter
}
