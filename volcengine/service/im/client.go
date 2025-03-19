package im

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/yaoapp/yao/volcengine/base"
)

// Im is the client for im service
type Im struct {
	*base.Client
}

// NewInstance creates a new instance of Im
func NewInstance() *Im {
	return NewInstanceWithRegion("cn-north-1")
}

// NewInstanceWithRegion creates a new instance of Im with region
func NewInstanceWithRegion(region string) *Im {
	serviceInfo, ok := ServiceInfoMap[region]
	if !ok {
		panic(fmt.Errorf("Im not support region %s", region))
	}
	instance := &Im{
		Client: base.NewClient(&serviceInfo, ApiListInfo),
	}
	return instance
}

// GetConversationMarks gets conversation marks
func (c *Im) GetConversationMarks(ctx context.Context, arg interface{}) (interface{}, error) {
	body, err := json.Marshal(arg)
	if err != nil {
		return nil, err
	}

	data, _, err := c.Client.CtxJson(ctx, "GetConversationMarks", url.Values{}, string(body))
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// MarkConversation marks conversation
func (c *Im) MarkConversation(ctx context.Context, arg interface{}) (interface{}, error) {
	body, err := json.Marshal(arg)
	if err != nil {
		return nil, err
	}

	data, _, err := c.Client.CtxJson(ctx, "MarkConversation", url.Values{}, string(body))
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// CreateConversation creates conversation
func (c *Im) CreateConversation(ctx context.Context, arg interface{}) (interface{}, error) {
	body, err := json.Marshal(arg)
	if err != nil {
		return nil, err
	}

	data, _, err := c.Client.CtxJson(ctx, "CreateConversation", url.Values{}, string(body))
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// SendMessage sends message
func (c *Im) SendMessage(ctx context.Context, arg interface{}) (interface{}, error) {
	body, err := json.Marshal(arg)
	if err != nil {
		return nil, err
	}

	data, _, err := c.Client.CtxJson(ctx, "SendMessage", url.Values{}, string(body))
	if err != nil {
		return nil, err