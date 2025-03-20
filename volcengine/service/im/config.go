package im

import (
	"net/http"
	"net/url"
	"time"

	"github.com/yaoapp/yao/base"
)

const (
	ServiceName    = "rtc"
	DefaultTimeout = 10 * time.Second
	ApiVersion     = "2020-12-01"
)

var (
	ServiceInfoMap = map[string]base.ServiceInfo{
		"cn-north-1": {
			Timeout: DefaultTimeout,
			Scheme:  "https",
			Host:    "rtc.volcengineapi.com",
			Header: http.Header{
				"Accept": []string{"application/json"},
			},
			Credentials: base.Credentials{
				Region:  "cn-north-1",
				Service: ServiceName,
			},
		},
		"ap-southeast-1": {
			Timeout: DefaultTimeout,
			Scheme:  "https",
			Host:    "rtc.volcengineapi.com",
			Header: http.Header{
				"Accept": []string{"application/json"},
			},
			Credentials: base.Credentials{
				Region:  "ap-southeast-1",
				Service: ServiceName,
			},
		},
	}

	ApiListInfo = map[string]*base.ApiInfo{
		"GetConversationMarks": {
			Method: http.MethodPost,
			Path:   "/",
			Query: url.Values{
				"Action":  []string{"GetConversationMarks"},
				"Version": []string{ApiVersion},
			},
		},
		"MarkConversation": {
			Method: http.MethodPost,
			Path:   "/",
			Query: url.Values{
				"Action":  []string{"MarkConversation"},
				"Version": []string{ApiVersion},
			},
		},
		"ModifyParticipantReadIndex": {
			Method: http.MethodPost,
			Path:   "/",
			Query: url.Values{
				"Action":  []string{"ModifyParticipantReadIndex"},
				"Version": []string{ApiVersion},
			},
		},
		"BatchAddBlockParticipants": {
			Method: http.MethodPost,
			Path:   "/",
			Query: url.Values{
				"Action":  []string{"BatchAddBlockParticipants"},
				"Version": []string{ApiVersion},
			},
		},
		"BatchDeleteBlockParticipants": {
			Method: http.MethodPost,
			Path:   "/",
			Query: url.Values{
				"Action":  []string{"BatchDeleteBlockParticipants"},
				"Version": []string{ApiVersion},
			},
		},
		"BatchGetBlockParticipants": {
			Method: http.MethodPost,
			Path:   "/",
			Query: url.Values{
				"Action":  []string{"BatchGetBlockParticipants"},
				"Version": []string{ApiVersion},
			},
		},
		"CreateConversation": {
			Method: http.MethodPost,
			Path:   "/",
			Query: url.Values{
				"Action":  []string{"CreateConversation"},
				"Version": []string{ApiVersion},
			},
		},
		"DeleteConversation": {
			Method: http.MethodPost,
			Path:   "/",
			Query: url.Values{
				"Action":  []string{"DeleteConversation"},
				"Version": []string{ApiVersion},
			},
		},
		"GetConversation": {
			Method: http.MethodPost,
			Path:   "/",
			Query: url.Values{
				"Action":  []string{"GetConversation"},
				"Version": []string{ApiVersion},
			},
		},
		"SendMessage": {
			Method: http.MethodPost,
			Path:   "/",
			Query: url.Values{
				"Action":  []string{"SendMessage"},
				"Version": []string{ApiVersion},
			},
		},
		"GetMessages": {
			Method: http.MethodPost,
			Path:   "/",
			Query: url.Values{
				"Action":  []string{"GetMessages"},
				"Version": []string{ApiVersion},
			},
		},
		"DeleteMessage": {
			Method: http.MethodPost,
			Path:   "/",
			Query: url.Values{
				"Action":  []string{"DeleteMessage"},
				"Version": []string{ApiVersion},
			},
		},
	}
)
