package rtc

import (
	"fmt"
	"strconv"
	"time"

	json "github.com/goccy/go-json"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/volcengine"
)

const defaultTokenExpireSeconds = 24 * 60 * 60

func init() {
	process.RegisterGroup("volc.rtc", map[string]process.Handler{
		"getToken": ProcessGetToken,
		"gettoken": ProcessGetToken,
	})
}

func ProcessGetToken(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	args := p.ArgsMap(0)

	roomID := stringArg(args, "RoomId", "RoomID", "roomId", "room_id")
	if roomID == "" {
		exception.New("RoomId is required", 400).Throw()
	}

	userID := stringArg(args, "UserId", "UserID", "userId", "user_id")
	if userID == "" {
		exception.New("UserId is required", 400).Throw()
	}

	if volcengine.VolcEngine == nil {
		exception.New("volcengine config is not loaded", 500).Throw()
	}

	appID := volcengine.VolcEngine.RTC.AppID
	appKey := volcengine.VolcEngine.RTC.AppKey
	if appID == "" || appKey == "" {
		exception.New("rtc appid or appkey is not configured", 500).Throw()
	}

	expireSeconds, err := int64Arg(args, defaultTokenExpireSeconds, "ExpireSeconds", "expireSeconds", "ExpireTime", "expireTime")
	if err != nil {
		exception.New("invalid expire seconds: %s", 400, err.Error()).Throw()
	}
	if expireSeconds <= 0 {
		exception.New("expire seconds must be greater than 0", 400).Throw()
	}

	expiredAt := time.Now().Add(time.Duration(expireSeconds) * time.Second)
	token, err := generateToken(appID, appKey, roomID, userID, expiredAt)
	if err != nil {
		exception.New("generate rtc token failed: %s", 500, err.Error()).Throw()
	}

	return map[string]interface{}{
		"AppId":     appID,
		"RoomId":    roomID,
		"UserId":    userID,
		"Token":     token,
		"ExpiredAt": expiredAt.Unix(),
	}
}

func generateToken(appID, appKey, roomID, userID string, expiredAt time.Time) (string, error) {
	token, err := newAccessToken(appID, appKey, roomID, userID)
	if err != nil {
		return "", err
	}

	token.expireTime(expiredAt)
	token.addPrivilege(privSubscribeStream, expiredAt)
	token.addPrivilege(privPublishStream, expiredAt)
	return token.serialize()
}

func stringArg(args map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := args[key]; ok && value != nil {
			return fmt.Sprintf("%v", value)
		}
	}
	return ""
}

func int64Arg(args map[string]interface{}, defaultValue int64, keys ...string) (int64, error) {
	for _, key := range keys {
		value, ok := args[key]
		if !ok || value == nil {
			continue
		}

		switch typed := value.(type) {
		case int:
			return int64(typed), nil
		case int64:
			return typed, nil
		case float64:
			return int64(typed), nil
		case string:
			return strconv.ParseInt(typed, 10, 64)
		case json.Number:
			return typed.Int64()
		default:
			return strconv.ParseInt(fmt.Sprintf("%v", typed), 10, 64)
		}
	}
	return defaultValue, nil
}
