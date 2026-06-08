package sse

import "errors"

const (
	defaultHeartbeatSeconds = 15
	defaultQueueSize        = 64
	defaultBusConnector     = "redis"
	defaultBusChannel       = "yao:sse:notification"
)

var ErrIdentityNotFound = errors.New("sse identity not found")

type Identity struct {
	SessionID string
	UserID    string
}

type Event struct {
	UserID    string      `json:"userId"`
	ClientID  string      `json:"clientId,omitempty"`
	Event     string      `json:"event"`
	Data      interface{} `json:"data"`
	MessageID string      `json:"messageId,omitempty"`
	Timestamp string      `json:"timestamp,omitempty"`
}

type PublishResult struct {
	OK        bool   `json:"ok"`
	Published bool   `json:"published"`
	Code      string `json:"code,omitempty"`
	Message   string `json:"message,omitempty"`
}
