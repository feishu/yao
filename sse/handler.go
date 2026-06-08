package sse

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type HandlerOptions struct {
	Hub              *Hub
	HeartbeatSeconds int
	Available        func() bool
}

type flushErrorWriter interface {
	FlushError() error
}

type unwrapResponseWriter interface {
	Unwrap() http.ResponseWriter
}

func NewHandler(options HandlerOptions) gin.HandlerFunc {
	hub := options.Hub
	if hub == nil {
		hub = NewHub(defaultQueueSize)
	}

	heartbeatSeconds := options.HeartbeatSeconds
	if heartbeatSeconds <= 0 {
		heartbeatSeconds = defaultHeartbeatSeconds
	}

	return func(c *gin.Context) {
		if options.Available != nil && !options.Available() {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"code":    http.StatusServiceUnavailable,
				"message": "authenticated sse is unavailable",
			})
			return
		}

		identity, err := ResolveIdentity(c.GetString("__sid"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    http.StatusUnauthorized,
				"message": "sse identity not found",
			})
			return
		}

		conn := hub.Add(identity)
		defer hub.Remove(conn.ID())

		c.Header("Content-Type", "text/event-stream; charset=utf-8")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")

		if err := writeJSONEvent(c.Writer, "connected", gin.H{
			"clientId":  conn.ID(),
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		}); err != nil {
			return
		}
		if err := flushWriter(c.Writer); err != nil {
			return
		}

		ticker := time.NewTicker(time.Duration(heartbeatSeconds) * time.Second)
		defer ticker.Stop()

		requestContext := c.Request.Context()
		c.Stream(func(w io.Writer) bool {
			responseWriter, ok := w.(http.ResponseWriter)
			if !ok {
				return false
			}

			select {
			case event, ok := <-conn.Events():
				if !ok {
					return false
				}
				data, err := SerializeEventData(event.Data)
				if err != nil {
					return false
				}
				if err := writeEvent(w, event.Event, data); err != nil {
					return false
				}
				return flushWriter(responseWriter) == nil

			case <-ticker.C:
				if err := writeJSONEvent(w, "heartbeat", gin.H{
					"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
				}); err != nil {
					return false
				}
				return flushWriter(responseWriter) == nil

			case <-requestContext.Done():
				return false
			}
		})
	}
}

func writeJSONEvent(w io.Writer, event string, data interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return writeEvent(w, event, string(payload))
}

func writeEvent(w io.Writer, event string, data string) error {
	if _, err := fmt.Fprintf(w, "event: %s\n", event); err != nil {
		return err
	}

	for _, line := range strings.Split(data, "\n") {
		if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
			return err
		}
	}

	_, err := fmt.Fprint(w, "\n")
	return err
}

func flushWriter(w http.ResponseWriter) error {
	for current := w; current != nil; {
		if writer, ok := current.(flushErrorWriter); ok {
			return writer.FlushError()
		}

		unwrapper, ok := current.(unwrapResponseWriter)
		if !ok {
			break
		}

		next := unwrapper.Unwrap()
		if next == nil || next == current {
			break
		}
		current = next
	}

	return http.NewResponseController(w).Flush()
}
