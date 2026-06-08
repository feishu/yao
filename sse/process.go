package sse

import (
	"context"
	"math"
	"strconv"
	"strings"

	"github.com/yaoapp/gou/process"
)

func ProcessPublish(proc *process.Process) interface{} {
	proc.ValidateArgNums(1)

	args, ok := proc.Args[0].(map[string]interface{})
	if !ok {
		return PublishResult{OK: false, Code: "INVALID_INPUT", Message: "payload must be an object"}
	}

	userID, ok := userIDArg(args, "userId")
	if !ok {
		return PublishResult{OK: false, Code: "INVALID_USER"}
	}

	eventName, ok := eventNameArg(args, "event")
	if !ok {
		return PublishResult{OK: false, Code: "INVALID_EVENT"}
	}
	if err := ValidateEventName(eventName); err != nil {
		return PublishResult{OK: false, Code: "INVALID_EVENT"}
	}

	clientID, ok := clientIDArg(args, "clientId")
	if !ok {
		return PublishResult{OK: false, Code: "INVALID_INPUT", Message: "clientId must be a string"}
	}

	targetBus := currentBus()
	if targetBus == nil {
		return PublishResult{OK: false, Code: "PUBSUB_UNAVAILABLE", Message: "Redis Pub/Sub publish failed"}
	}

	event := Event{
		UserID:   userID,
		ClientID: clientID,
		Event:    eventName,
		Data:     args["data"],
	}

	ctx := proc.Context
	if ctx == nil {
		ctx = context.Background()
	}

	if err := targetBus.Publish(ctx, event); err != nil {
		return PublishResult{OK: false, Code: "PUBSUB_UNAVAILABLE", Message: "Redis Pub/Sub publish failed"}
	}

	return PublishResult{OK: true, Published: true}
}

func userIDArg(args map[string]interface{}, key string) (string, bool) {
	value, ok := args[key]
	if !ok || value == nil {
		return "", false
	}

	switch v := value.(type) {
	case string:
		text := strings.TrimSpace(v)
		return text, text != ""
	case int:
		return strconv.FormatInt(int64(v), 10), true
	case int8:
		return strconv.FormatInt(int64(v), 10), true
	case int16:
		return strconv.FormatInt(int64(v), 10), true
	case int32:
		return strconv.FormatInt(int64(v), 10), true
	case int64:
		return strconv.FormatInt(v, 10), true
	case uint:
		return strconv.FormatUint(uint64(v), 10), true
	case uint8:
		return strconv.FormatUint(uint64(v), 10), true
	case uint16:
		return strconv.FormatUint(uint64(v), 10), true
	case uint32:
		return strconv.FormatUint(uint64(v), 10), true
	case uint64:
		return strconv.FormatUint(v, 10), true
	case float32:
		n := float64(v)
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return "", false
		}
		return strconv.FormatFloat(n, 'f', -1, 32), true
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return "", false
		}
		return strconv.FormatFloat(v, 'f', -1, 64), true
	default:
		return "", false
	}
}

func eventNameArg(args map[string]interface{}, key string) (string, bool) {
	value, ok := args[key]
	if !ok || value == nil {
		return "", false
	}

	text, ok := value.(string)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(text), true
}

func clientIDArg(args map[string]interface{}, key string) (string, bool) {
	value, ok := args[key]
	if !ok || value == nil {
		return "", true
	}

	text, ok := value.(string)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(text), true
}
