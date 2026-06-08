package sse

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestValidateEventNameRejectsReservedNames(t *testing.T) {
	for _, name := range []string{"connected", "heartbeat", "error"} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateEventName(name); err == nil {
				t.Fatalf("ValidateEventName(%q) returned nil, want error", name)
			}
		})
	}
}

func TestValidateEventNameRejectsInvalidNames(t *testing.T) {
	names := []string{
		"",
		"   ",
		strings.Repeat("a", 65),
		"message created",
		"message/created",
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			if err := ValidateEventName(name); err == nil {
				t.Fatalf("ValidateEventName(%q) returned nil, want error", name)
			}
		})
	}
}

func TestValidateEventNameAcceptsSupportedSeparators(t *testing.T) {
	for _, name := range []string{"message.created", "message:created", "message-created"} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateEventName(name); err != nil {
				t.Fatalf("ValidateEventName(%q) returned error: %v", name, err)
			}
		})
	}
}

func TestSerializeEventDataKeepsString(t *testing.T) {
	data, err := SerializeEventData("hello")
	if err != nil {
		t.Fatalf("SerializeEventData returned error: %v", err)
	}
	if data != "hello" {
		t.Fatalf("data = %q, want %q", data, "hello")
	}
}

func TestSerializeEventDataEncodesObject(t *testing.T) {
	data, err := SerializeEventData(map[string]interface{}{"hello": "world"})
	if err != nil {
		t.Fatalf("SerializeEventData returned error: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(data), &decoded); err != nil {
		t.Fatalf("data is not JSON object: %v", err)
	}
	if decoded["hello"] != "world" {
		t.Fatalf("decoded hello = %v, want %q", decoded["hello"], "world")
	}
}

func TestWriteEventReturnsWriteError(t *testing.T) {
	if err := writeEvent(failingWriter{}, "message", "hello"); err == nil {
		t.Fatal("writeEvent returned nil, want write error")
	}
}

func TestWriteEventWritesMultilineData(t *testing.T) {
	var buffer bytes.Buffer

	if err := writeEvent(&buffer, "message", "hello\nworld"); err != nil {
		t.Fatalf("writeEvent returned error: %v", err)
	}

	want := "event: message\ndata: hello\ndata: world\n\n"
	if got := buffer.String(); got != want {
		t.Fatalf("event = %q, want %q", got, want)
	}
}
