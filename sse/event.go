package sse

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
)

var eventNamePattern = regexp.MustCompile("^[A-Za-z0-9_.:-]+$")

var reservedEventNames = map[string]struct{}{
	"connected": {},
	"heartbeat": {},
	"error":     {},
}

func ValidateEventName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("sse event name is required")
	}
	if len(name) > 64 {
		return errors.New("sse event name is too long")
	}
	if _, ok := reservedEventNames[name]; ok {
		return errors.New("sse event name is reserved")
	}
	if !eventNamePattern.MatchString(name) {
		return errors.New("sse event name contains invalid characters")
	}
	return nil
}

func SerializeEventData(data interface{}) (string, error) {
	if text, ok := data.(string); ok {
		return text, nil
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}
