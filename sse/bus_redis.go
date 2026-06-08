package sse

import (
	"context"
	"encoding/json"
	"fmt"

	redisv8 "github.com/go-redis/redis/v8"
	"github.com/yaoapp/gou/connector"
	redisConnector "github.com/yaoapp/gou/connector/redis"
)

type RedisBus struct {
	Connector string
	Channel   string
}

func NewRedisBus(connectorName, channel string) *RedisBus {
	if connectorName == "" {
		connectorName = defaultBusConnector
	}
	if channel == "" {
		channel = defaultBusChannel
	}

	return &RedisBus{Connector: connectorName, Channel: channel}
}

func (bus *RedisBus) Probe(ctx context.Context) error {
	client, err := bus.client()
	if err != nil {
		return err
	}

	return client.Ping(ctx).Err()
}

func (bus *RedisBus) Publish(ctx context.Context, event Event) error {
	client, err := bus.client()
	if err != nil {
		return err
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return client.Publish(ctx, bus.channel(), payload).Err()
}

func (bus *RedisBus) Subscribe(ctx context.Context, handler func(Event)) error {
	client, err := bus.client()
	if err != nil {
		return err
	}

	pubsub := client.Subscribe(ctx, bus.channel())
	defer pubsub.Close()

	for {
		message, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			return err
		}

		var event Event
		if err := json.Unmarshal([]byte(message.Payload), &event); err != nil {
			continue
		}
		handler(event)
	}
}

func (bus *RedisBus) client() (*redisv8.Client, error) {
	connectorName := bus.connector()
	selected, err := connector.Select(connectorName)
	if err != nil {
		return nil, fmt.Errorf("redis connector %q: %w", connectorName, err)
	}

	redis, ok := selected.(*redisConnector.Connector)
	if !ok {
		return nil, fmt.Errorf("redis connector %q has unexpected type %T", connectorName, selected)
	}
	if redis.Rdb == nil {
		return nil, fmt.Errorf("redis connector %q has nil client", connectorName)
	}

	return redis.Rdb, nil
}

func (bus *RedisBus) connector() string {
	if bus.Connector == "" {
		return defaultBusConnector
	}
	return bus.Connector
}

func (bus *RedisBus) channel() string {
	if bus.Channel == "" {
		return defaultBusChannel
	}
	return bus.Channel
}
