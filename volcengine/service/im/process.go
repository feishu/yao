package im

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/volcengine"
)

// convertInt64ToStringRecursive 递归地将数据结构中的 int64 类型转换为字符串
// 这是为了解决 JavaScript 处理大整数时的精度问题
func convertInt64ToStringRecursive(data interface{}) interface{} {
	if data == nil {
		return nil
	}

	switch v := data.(type) {
	case int64:
		return strconv.FormatInt(v, 10)
	case *int64:
		if v == nil {
			return nil
		}
		return strconv.FormatInt(*v, 10)
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			result[key] = convertInt64ToStringRecursive(value)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = convertInt64ToStringRecursive(item)
		}
		return result
	default:
		// 使用反射处理结构体类型
		val := reflect.ValueOf(data)
		if val.Kind() == reflect.Ptr {
			if val.IsNil() {
				return nil
			}
			val = val.Elem()
		}

		switch val.Kind() {
		case reflect.Struct:
			// 创建一个新的 map 来存储结构体字段
			result := make(map[string]interface{})
			typ := val.Type()
			for i := 0; i < val.NumField(); i++ {
				field := val.Field(i)
				fieldType := typ.Field(i)
				
				// 跳过未导出的字段
				if !field.CanInterface() {
					continue
				}
				
				// 获取字段名（优先使用 json tag）
				fieldName := fieldType.Name
				if jsonTag := fieldType.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
					if commaIdx := strings.Index(jsonTag, ","); commaIdx != -1 {
						fieldName = jsonTag[:commaIdx]
					} else {
						fieldName = jsonTag
					}
				}
				
				result[fieldName] = convertInt64ToStringRecursive(field.Interface())
			}
			return result
		case reflect.Slice, reflect.Array:
			result := make([]interface{}, val.Len())
			for i := 0; i < val.Len(); i++ {
				result[i] = convertInt64ToStringRecursive(val.Index(i).Interface())
			}
			return result
		case reflect.Map:
			result := make(map[string]interface{})
			for _, key := range val.MapKeys() {
				keyStr := fmt.Sprintf("%v", key.Interface())
				result[keyStr] = convertInt64ToStringRecursive(val.MapIndex(key).Interface())
			}
			return result
		}
	}

	return data
}

// parseInt64FromArgs 从参数中解析int64值，支持字符串和数值类型以保持兼容性
func parseInt64FromArgs(args map[string]interface{}, key string) (int64, error) {
	val, exists := args[key]
	if !exists {
		return 0, fmt.Errorf("%s is required", key)
	}

	switch v := val.(type) {
	case string:
		return strconv.ParseInt(v, 10, 64)
	case float64:
		return int64(v), nil
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	case json.Number: // 处理 JSON 数字类型
		return v.Int64()
	default:
		// 尝试通过格式化字符串转换
		strVal := fmt.Sprintf("%v", v)
		return strconv.ParseInt(strVal, 10, 64)
	}
}

// parseInt64ArrayFromArgs 从参数中解析int64数组，支持字符串和数值类型
func parseInt64ArrayFromArgs(args []interface{}) []int64 {
	result := make([]int64, 0, len(args))
	for _, arg := range args {
		if strVal, ok := arg.(string); ok {
			if val, err := strconv.ParseInt(strVal, 10, 64); err == nil {
				result = append(result, val)
			}
		} else if floatVal, ok := arg.(float64); ok {
			result = append(result, int64(floatVal))
		}
	}
	return result
}

// parseInt64PointerArrayFromArgs 从参数中解析int64指针数组，支持字符串和数值类型
func parseInt64PointerArrayFromArgs(args []interface{}) []*int64 {
	result := make([]*int64, 0, len(args))
	for _, arg := range args {
		if strVal, ok := arg.(string); ok {
			if val, err := strconv.ParseInt(strVal, 10, 64); err == nil {
				result = append(result, &val)
			}
		} else if floatVal, ok := arg.(float64); ok {
			val := int64(floatVal)
			result = append(result, &val)
		}
	}
	return result
}

// 单例模式实现
var (
	instance *Im
	once     sync.Once
)

// GetInstance 获取 Im 单例实例
func GetInstance() *Im {
	once.Do(func() {
		instance = NewInstance()
	})
	return instance
}

func init() {
	process.RegisterGroup("volc.im", map[string]process.Handler{
		"registerUsers":             ProcessRegisterUsers,
		"batchGetUser":              ProcessBatchGetUser,
		"unRegisterUsers":           ProcessUnRegisterUsers,
		"batchUpdateUser":           ProcessBatchUpdateUser,
		"createConversation":        ProcessCreateConversation,
		"modifyConversation":        ProcessModifyConversation,
		"isUserInConversation":      ProcessIsUserInConversation,
		"sendMessage":               ProcessSendMessage,
		"recallMessage":             ProcessRecallMessage,
		"deleteConversationMessage": ProcessDeleteConversationMessage,
		"getConversationMessages":   ProcessGetConversationMessages,
		"destroyConversation":       ProcessDestroyConversation,
		"getAppToken":               ProcessGetAppToken,
		"Client":                    ProcessClient,
	})
}

func ProcessClient(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	args := p.ArgsMap(0)

	// 获取方法名
	method, ok := args["method"].(string)
	if !ok {
		exception.New("method is required", 400).Throw()
	}

	// 获取请求体参数
	body, ok := args["body"]
	if !ok {
		exception.New("body is required", 400).Throw()
	}

	// 提取longFields并获取处理后的body
	longFields, processedBody := extractLongFields(body)

	// 将processedBody中的longFields字段从字符串转换为int64
	// 这是必要的，因为GetInstance().Client.CtxJson期望longFields中的字段为int64类型
	if len(longFields) > 0 {
		processedBody = convertStringFieldsToLongs(processedBody, longFields)
	}

	// 将body序列化为JSON，支持大整数处理
	bodyBytes, err := marshalToJsonWithLongSupport(processedBody)
	if err != nil {
		exception.New("Failed to marshal body to JSON: %s", 500, err.Error()).Throw()
	}

	// 调用Client.CtxJson方法
	ctx := context.Background()
	data, _, err := GetInstance().Client.CtxJson(ctx, method, url.Values{}, string(bodyBytes))
	if err != nil {
		exception.New("Client request failed: %s", 500, err.Error()).Throw()
	}

	// 解析响应
	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		exception.New("Failed to unmarshal response: %s", 500, err.Error()).Throw()
	}

	// 将 int64 类型转换为字符串，避免 JavaScript 精度问题
	convertedResult := convertInt64ToStringRecursive(result)
	return convertedResult
}

// ProcessGetAppToken 获取火山引擎IM的AppToken
// 用于客户端鉴权使用
// 接口文档: https://www.volcengine.com/docs/6348/435387
func ProcessGetAppToken(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	args := p.ArgsMap(0)

	// 获取用户ID
	userID, err := parseInt64FromArgs(args, "UserId")
	if err != nil {
		exception.New("UserId is required", 400, err.Error()).Throw()
	}

	// 获取过期时间，默认30分钟
	var expireTime int64
	if expire, ok := args["ExpireTime"].(int64); ok {
		expireTime = generateExpireTime(int64(expire))
	} else {
		// 默认3600分钟后过期
		expireTime = generateExpireTime(3600)
	}

	// 使用配置文件中的AppID和AppKey
	appID := int32(volcengine.VolcEngine.IM.AppID)
	appKey := volcengine.VolcEngine.IM.AppKey

	// 生成Token，调用token.go中的GenerateToken函数
	token, err := GenerateToken(appID, userID, expireTime, appKey)
	if err != nil {
		exception.New("Generate token failed: %s", 500, err.Error()).Throw()
	}

	return map[string]interface{}{
		"Token":      token,
		"UserId":     userID,
		"AppId":      appID,
		"ExpireTime": expireTime,
	}
}

// ProcessRegisterUsers 注册用户到IM系统
// 支持批量注册多个用户，通过Users数组传入用户信息
// 接口文档: https://www.volcengine.com/docs/6348/1125993
func ProcessRegisterUsers(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	args := p.ArgsMap(0)

	// 使用配置文件中的AppId
	appID := int32(volcengine.VolcEngine.IM.AppID)

	users, ok := args["Users"].([]interface{})
	if !ok {
		exception.New("Users is required", 400).Throw()
	}

	// 构建请求体
	body := &RegisterUsersBody{
		AppID: appID,
		Users: []RegisterUsersBodyUsersItem{},
	}

	// 转换用户信息并构建用户项
	for _, user := range users {
		userMap, ok := user.(map[string]interface{})
		if !ok {
			exception.New("User must be an object", 400).Throw()
		}

		userID, err := parseInt64FromArgs(userMap, "UserId")
		if err != nil {
			exception.New("User.UserId is required and must be a valid integer", 400).Throw()
		}

		userItem := RegisterUsersBodyUsersItem{
			UserID: userID,
		}

		// 设置可选字段
		if nickName, ok := userMap["NickName"].(string); ok {
			userItem.NickName = &nickName
		}

		if portrait, ok := userMap["Portrait"].(string); ok {
			userItem.Portrait = &portrait
		}

		// 处理标签
		if tags, ok := userMap["Tags"].([]interface{}); ok {
			tagStrings := make([]string, 0, len(tags))
			for _, tag := range tags {
				if tagStr, ok := tag.(string); ok {
					tagStrings = append(tagStrings, tagStr)
				}
			}
			userItem.Tags = tagStrings
		}

		// 处理扩展字段
		if ext, ok := userMap["Ext"].(map[string]interface{}); ok {
			extMap := make(map[string]string)
			for k, v := range ext {
				if vStr, ok := v.(string); ok {
					extMap[k] = vStr
				}
			}
			userItem.Ext = extMap
		}

		body.Users = append(body.Users, userItem)
	}

	// 调用 API
	ctx := context.Background()
	res, err := GetInstance().RegisterUsers(ctx, body)
	if err != nil {
		exception.New("Register users failed: %s", 500, err.Error()).Throw()
	}

	// 将返回数据中的 int64 类型转换为字符串，避免 JavaScript 精度问题
	convertedRes := convertInt64ToStringRecursive(res)
	return convertedRes
}

// ProcessBatchGetUser 批量获取用户信息
// 支持批量获取多个用户信息，通过UserIds数组传入用户ID
// 接口文档: https://www.volcengine.com/docs/6348/1125995
func ProcessBatchGetUser(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	args := p.ArgsMap(0)

	// 使用配置文件中的AppId
	appID := int32(volcengine.VolcEngine.IM.AppID)

	userIDs, ok := args["UserIds"].([]interface{})
	if !ok {
		exception.New("UserIds is required", 400).Throw()
	}

	userIDsInt := parseInt64ArrayFromArgs(userIDs)

	body := &BatchGetUserBody{
		AppID:   appID,
		UserIDs: userIDsInt,
	}

	// 调用 API
	ctx := context.Background()
	res, err := GetInstance().BatchGetUser(ctx, body)

	if err != nil {
		exception.New("Batch get user failed: %s", 500, err.Error()).Throw()
	}

	fmt.Printf("res: %+v\n", res)

	// 将返回数据中的 int64 类型转换为字符串，避免 JavaScript 精度问题
	convertedRes := convertInt64ToStringRecursive(res)
	return convertedRes
}

// ProcessBatchUpdateUser 批量更新用户信息
// 支持批量更新多个用户信息，通过Users数组传入用户信息
// 接口文档: https://www.volcengine.com/docs/6348/1125996
func ProcessBatchUpdateUser(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	args := p.ArgsMap(0)

	// 使用配置文件中的AppId
	appID := int32(volcengine.VolcEngine.IM.AppID)

	users, ok := args["Users"].([]interface{})
	if !ok {
		exception.New("Users is required", 400).Throw()
	}

	userItems := make([]BatchUpdateUserBodyUsersItem, 0, len(users))

	body := &BatchUpdateUserBody{
		AppID: appID,
		Users: []BatchUpdateUserBodyUsersItem{},
	}

	// 转换用户信息并构建用户项
	for _, user := range users {
		userMap, ok := user.(map[string]interface{})
		if !ok {
			exception.New("User must be an object", 400).Throw()
		}

		userID, err := parseInt64FromArgs(userMap, "UserId")
		if err != nil {
			exception.New("User.UserId is required and must be a valid integer", 400).Throw()
		}

		userItem := BatchUpdateUserBodyUsersItem{
			UserID: userID,
		}

		if nickName, ok := userMap["NickName"].(string); ok {
			userItem.NickName = nickName
		}

		if portrait, ok := userMap["Portrait"].(string); ok {
			userItem.Portrait = portrait
		}

		if tags, ok := userMap["Tags"].([]interface{}); ok {
			tagStrings := make([]string, 0, len(tags))
			for _, tag := range tags {
				if tagStr, ok := tag.(string); ok {
					tagStrings = append(tagStrings, tagStr)
				}
			}
			userItem.Tags = tagStrings
		}

		if ext, ok := userMap["Ext"].(map[string]interface{}); ok {
			extMap := make(map[string]string)
			for k, v := range ext {
				if vStr, ok := v.(string); ok {
					extMap[k] = vStr
				}
			}
			userItem.Ext = extMap
		}

		userItems = append(userItems, userItem)

	}

	body.Users = userItems

	// 调用 API
	ctx := context.Background()
	res, err := GetInstance().BatchUpdateUser(ctx, body)
	if err != nil {
		exception.New("Batch update user failed: %s", 500, err.Error()).Throw()
	}

	// 将返回数据中的 int64 类型转换为字符串，避免 JavaScript 精度问题
	convertedRes := convertInt64ToStringRecursive(res)
	return convertedRes
}

// ProcessUnRegisterUsers 注销用户
// 支持批量注销多个用户，通过UserIds数组传入用户ID
// 接口文档: https://www.volcengine.com/docs/6348/1125994
func ProcessUnRegisterUsers(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	args := p.ArgsMap(0)

	// 使用配置文件中的AppId
	appID := int32(volcengine.VolcEngine.IM.AppID)

	userIDs, ok := args["UserIds"].([]interface{})
	if !ok {
		exception.New("UserIds is required", 400).Throw()
	}

	userIDsInt := parseInt64ArrayFromArgs(userIDs)

	body := &BatchGetUserBody{
		AppID:   appID,
		UserIDs: userIDsInt,
	}

	// 调用 API
	ctx := context.Background()
	res, err := GetInstance().BatchGetUser(ctx, body)

	if err != nil {
		exception.New("Batch get user failed: %s", 500, err.Error()).Throw()
	}

	// 将返回数据中的 int64 类型转换为字符串，避免 JavaScript 精度问题
	convertedRes := convertInt64ToStringRecursive(res)
	return convertedRes
}

// ProcessCreateConversation 创建会话（单聊或群聊）
// 可设置会话名称、类型、管理员等属性
// 接口文档: https://www.volcengine.com/docs/6348/337013
func ProcessCreateConversation(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	args := p.ArgsMap(0)

	// 使用配置文件中的AppId
	appID := int32(volcengine.VolcEngine.IM.AppID)

	// 构建请求体
	body := &CreateConversationBody{
		AppID:                appID,
		ConversationCoreInfo: CreateConversationBodyConversationCoreInfo{},
	}

	// 设置可选参数
	if name, ok := args["Name"].(string); ok {
		body.ConversationCoreInfo.Name = &name
	}

	if conversationType, ok := args["ConversationType"].(float64); ok {
		convType := int32(conversationType)
		body.ConversationCoreInfo.ConversationType = convType
	}

	if ownerID, err := parseInt64FromArgs(args, "Owner"); err == nil {
		body.OwnerUserID = ownerID
	}

	// 处理 Description 参数
	if description, ok := args["Description"].(string); ok {
		body.ConversationCoreInfo.Description = &description
	}

	// 处理 AvatarUrl 参数
	if avatarUrl, ok := args["AvatarUrl"].(string); ok {
		body.ConversationCoreInfo.AvatarURL = &avatarUrl
	}

	// 处理 Notice 参数
	if notice, ok := args["Notice"].(string); ok {
		body.ConversationCoreInfo.Notice = &notice
	}

	// 处理 Ext 参数（扩展字段）
	if ext, ok := args["Ext"].(map[string]interface{}); ok {
		extMap := make(map[string]string)
		for k, v := range ext {
			if vStr, ok := v.(string); ok {
				extMap[k] = vStr
			}
		}
		body.ConversationCoreInfo.Ext = extMap
	}

	// 处理 OtherUserId 参数（单聊时另一个用户的ID）
	if otherUserID, err := parseInt64FromArgs(args, "OtherUserId"); err == nil {
		body.OtherUserID = &otherUserID
	}

	// 处理 IdempotentId 参数（幂等ID）
	if idempotentId, ok := args["IdempotentId"].(string); ok {
		body.IdempotentID = &idempotentId
	}

	// 处理 InboxType 参数（信箱类型）
	if inboxType, ok := args["InboxType"].(float64); ok {
		inboxTypeInt := int32(inboxType)
		body.InboxType = &inboxTypeInt
	}

	// 调用 API
	ctx := context.Background()
	res, err := GetInstance().CreateConversation(ctx, body)
	if err != nil {
		exception.New("Create conversation failed: %s", 500, err.Error()).Throw()
	}

	// 将返回数据中的 int64 类型转换为字符串，避免 JavaScript 精度问题
	convertedRes := convertInt64ToStringRecursive(res)
	return convertedRes
}

// ProcessModifyConversation 修改会话信息
// 可修改会话名称、描述等属性
// 接口文档: https://www.volcengine.com/docs/6348/337115
func ProcessModifyConversation(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	args := p.ArgsMap(0)

	// 使用配置文件中的AppId
	appID := int32(volcengine.VolcEngine.IM.AppID)

	conversationID, err := parseInt64FromArgs(args, "ConversationShortId")
	if err != nil {
		exception.New("ConversationShortId is required and must be a valid integer", 400).Throw()
	}

	// 构建请求体
	body := &ModifyConversationBody{
		AppID: appID,
		ConversationCoreInfo: ModifyConversationBodyConversationCoreInfo{
			ConversationShortID: conversationID,
		},
	}

	// 设置可选参数
	if name, ok := args["Name"].(string); ok {
		body.ConversationCoreInfo.Name = &name
	}

	if description, ok := args["Description"].(string); ok {
		body.ConversationCoreInfo.Description = &description
	}

	// 处理 Notice 参数
	if notice, ok := args["Notice"].(string); ok {
		body.ConversationCoreInfo.Notice = &notice
	}

	// 处理 AvatarUrl 参数
	if avatarUrl, ok := args["AvatarUrl"].(string); ok {
		body.ConversationCoreInfo.AvatarURL = &avatarUrl
	}

	// 处理 Ext 参数（扩展字段）
	if ext, ok := args["Ext"].(map[string]interface{}); ok {
		extMap := make(map[string]string)
		for k, v := range ext {
			if vStr, ok := v.(string); ok {
				extMap[k] = vStr
			}
		}
		body.ConversationCoreInfo.Ext = extMap
	}

	// 调用 API
	ctx := context.Background()
	res, err := GetInstance().ModifyConversation(ctx, body)
	if err != nil {
		exception.New("Modify conversation failed: %s", 500, err.Error()).Throw()
	}

	// 将返回数据中的 int64 类型转换为字符串，避免 JavaScript 精度问题
	convertedRes := convertInt64ToStringRecursive(res)
	return convertedRes
}

// ProcessIsUserInConversation 检查用户是否在指定会话中
// 返回用户是否是会话成员的信息
// 接口文档: https://www.volcengine.com/docs/6348/336996
func ProcessIsUserInConversation(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	args := p.ArgsMap(0)

	// 使用配置文件中的AppId
	appID := int32(volcengine.VolcEngine.IM.AppID)

	conversationID, err := parseInt64FromArgs(args, "ConversationShortId")
	if err != nil {
		exception.New("ConversationShortId is required and must be a valid integer", 400).Throw()
	}

	// 优先使用 ParticipantUserId，如果没有则使用 UserId
	var userID int64

	if participantUserID, err := parseInt64FromArgs(args, "ParticipantUserId"); err == nil {
		userID = participantUserID
	} else if userIdFromArgs, err := parseInt64FromArgs(args, "UserId"); err == nil {
		userID = userIdFromArgs
	} else {
		return exception.New("UserId or ParticipantUserId is required", 400)
	}

	// 构建请求体
	body := &IsUserInConversationBody{
		AppID:               appID,
		ConversationShortID: conversationID,
		ParticipantUserID:   userID,
	}

	// 调用 API
	ctx := context.Background()
	res, err := GetInstance().IsUserInConversation(ctx, body)
	if err != nil {
		exception.New("Check user in conversation failed: %s", 500, err.Error()).Throw()
	}

	// 将返回数据中的 int64 类型转换为字符串，避免 JavaScript 精度问题
	convertedRes := convertInt64ToStringRecursive(res)
	return convertedRes
}

// ProcessSendMessage 发送消息
// 支持发送文本、图片、视频等多种类型消息
// 接口文档: https://www.volcengine.com/docs/6348/337135
func ProcessSendMessage(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	args := p.ArgsMap(0)

	// 使用配置文件中的AppId
	appID := int32(volcengine.VolcEngine.IM.AppID)

	conversationID, err := parseInt64FromArgs(args, "ConversationShortId")
	if err != nil {
		exception.New("ConversationShortId is required and must be a valid integer", 400).Throw()
	}

	senderID, err := parseInt64FromArgs(args, "SenderUserId")
	if err != nil {
		exception.New("SenderUserId is required and must be a valid integer", 400).Throw()
	}

	content, ok := args["Content"].(string)
	if !ok {
		exception.New("Content is required", 400).Throw()
	}

	// 构建请求体
	body := &SendMessageBody{
		AppID:               appID,
		ConversationShortID: conversationID,
		Sender:              senderID,
		Content:             content,
	}

	// 设置可选参数
	if messageType, ok := args["MessageType"].(float64); ok {
		msgType := int32(messageType)
		body.MsgType = msgType
	}

	// 处理 Ext 参数（扩展字段）
	if ext, ok := args["Ext"].(map[string]interface{}); ok {
		extMap := make(map[string]string)
		for k, v := range ext {
			if vStr, ok := v.(string); ok {
				extMap[k] = vStr
			}
		}
		body.Ext = extMap
	}

	// 处理 MentionedUsers 参数（@的用户列表）
	if mentionedUsers, ok := args["MentionedUsers"].([]interface{}); ok {
		body.MentionedUsers = parseInt64PointerArrayFromArgs(mentionedUsers)
	}

	// 处理 VisibleUsers 参数（可见用户列表）
	if visibleUsers, ok := args["VisibleUsers"].([]interface{}); ok {
		body.VisibleUsers = parseInt64PointerArrayFromArgs(visibleUsers)
	}

	// 处理 InvisibleUsers 参数（不可见用户列表）
	if invisibleUsers, ok := args["InvisibleUsers"].([]interface{}); ok {
		body.InvisibleUsers = parseInt64PointerArrayFromArgs(invisibleUsers)
	}

	// 处理 Priority 参数（消息优先级）
	if priority, ok := args["Priority"].(float64); ok {
		priorityInt := int32(priority)
		body.Priority = &priorityInt
	}

	// 处理 ClientMsgId 参数（客户端消息ID）
	if clientMsgId, ok := args["ClientMsgId"].(string); ok {
		body.ClientMsgID = &clientMsgId
	}

	// 处理 CreateTime 参数（消息创建时间）
	if createTime, err := parseInt64FromArgs(args, "CreateTime"); err == nil {
		body.CreateTime = &createTime
	}

	// 处理 RefMsgInfo 参数（引用消息）
	if refMsgInfo, ok := args["RefMsgInfo"].(map[string]interface{}); ok {
		refInfo := &SendMessageBodyRefMsgInfo{}

		if referencedMsgId, err := parseInt64FromArgs(refMsgInfo, "ReferencedMessageId"); err == nil {
			refInfo.ReferencedMessageID = referencedMsgId
		}

		if hint, ok := refMsgInfo["Hint"].(string); ok {
			refInfo.Hint = hint
		}

		body.RefMsgInfo = refInfo
	}

	// 调用 API
	ctx := context.Background()
	res, err := GetInstance().SendMessage(ctx, body)
	if err != nil {
		exception.New("Send message failed: %s", 500, err.Error()).Throw()
	}

	// 将返回数据中的 int64 类型转换为字符串，避免 JavaScript 精度问题
	convertedRes := convertInt64ToStringRecursive(res)
	return convertedRes
}

// ProcessRecallMessage 撤回消息
// 允许用户撤回已发送的消息
// 接口文档: https://www.volcengine.com/docs/6348/337141
func ProcessRecallMessage(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	args := p.ArgsMap(0)

	// 使用配置文件中的AppId
	appID := int32(volcengine.VolcEngine.IM.AppID)

	conversationID, err := parseInt64FromArgs(args, "ConversationShortId")
	if err != nil {
		exception.New("ConversationShortId is required and must be a valid integer", 400).Throw()
	}

	// 获取消息ID并转换为int64
	messageIDInt64, err := parseInt64FromArgs(args, "MessageId")
	if err != nil {
		exception.New("MessageId is required and must be a valid integer", 400).Throw()
	}

	// 获取用户ID（可选）
	var participantUserID int64
	if userID, err := parseInt64FromArgs(args, "ParticipantUserId"); err == nil {
		participantUserID = userID
	}

	// 构建请求体
	body := &RecallMessageBody{
		AppID:               appID,
		ConversationShortID: conversationID,
		MessageID:           messageIDInt64,
		ParticipantUserID:   participantUserID,
	}

	// 调用 API
	ctx := context.Background()
	res, err := GetInstance().RecallMessage(ctx, body)
	if err != nil {
		exception.New("Recall message failed: %s", 500, err.Error()).Throw()
	}

	// 将返回数据中的 int64 类型转换为字符串，避免 JavaScript 精度问题
	convertedRes := convertInt64ToStringRecursive(res)
	return convertedRes
}

// ProcessDeleteConversationMessage 删除会话消息
// 从会话中删除指定消息
// 接口文档: https://www.volcengine.com/docs/6348/337140
func ProcessDeleteConversationMessage(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	args := p.ArgsMap(0)

	// 使用配置文件中的AppId
	appID := int32(volcengine.VolcEngine.IM.AppID)

	conversationID, err := parseInt64FromArgs(args, "ConversationShortId")
	if err != nil {
		exception.New("ConversationShortId is required and must be a valid integer", 400).Throw()
	}

	// 获取消息ID并转换为int64
	messageIDInt64, err := parseInt64FromArgs(args, "MessageId")
	if err != nil {
		exception.New("MessageId is required and must be a valid integer", 400).Throw()
	}

	// 构建请求体
	body := &DeleteConversationMessageBody{
		AppID:               appID,
		ConversationShortID: conversationID,
		MessageID:           messageIDInt64,
	}

	// 调用 API
	ctx := context.Background()
	res, err := GetInstance().DeleteConversationMessage(ctx, body)
	if err != nil {
		exception.New("Delete message failed: %s", 500, err.Error()).Throw()
	}

	// 将返回数据中的 int64 类型转换为字符串，避免 JavaScript 精度问题
	convertedRes := convertInt64ToStringRecursive(res)
	return convertedRes
}

// ProcessGetConversationMessages 获取会话消息列表
// 根据会话ID获取会话中的消息列表
// 接口文档: https://www.volcengine.com/docs/6348/337138
func ProcessGetConversationMessages(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	args := p.ArgsMap(0)

	// 使用配置文件中的AppId
	appID := int32(volcengine.VolcEngine.IM.AppID)

	conversationID, err := parseInt64FromArgs(args, "ConversationShortId")
	if err != nil {
		exception.New("ConversationShortId is required and must be a valid integer", 400).Throw()
	}

	// 构建请求体
	body := &GetConversationMessagesBody{
		AppID:               appID,
		ConversationShortID: conversationID,
	}

	// 处理 Cursor 参数（查询起始位置）
	if cursor, err := parseInt64FromArgs(args, "Cursor"); err == nil {
		body.Cursor = cursor
	}

	// 处理 Limit 参数（查询条数）
	if limit, err := parseInt64FromArgs(args, "Limit"); err == nil {
		body.Limit = limit
	}

	// 处理 Reverse 参数（查询方向）
	if reverse, ok := args["Reverse"].(float64); ok {
		reverseInt := int32(reverse)
		body.Reverse = &reverseInt
	}

	// 构建消息ID列表
	var messageIDs []int64
	if msgIDList, ok := args["MessageIds"].([]interface{}); ok {
		messageIDs = parseInt64ArrayFromArgs(msgIDList)
	}

	// 如果提供了消息ID列表，则使用 GetMessages API
	if len(messageIDs) > 0 {
		getMsgBody := &GetMessagesBody{
			AppID:               appID,
			ConversationShortID: conversationID,
			MessageIDs:          messageIDs,
		}

		ctx := context.Background()
		res, err := GetInstance().GetMessages(ctx, getMsgBody)
		if err != nil {
			exception.New("Get conversation messages failed: %s", 500, err.Error()).Throw()
		}

		// 将返回数据中的 int64 类型转换为字符串，避免 JavaScript 精度问题
		convertedRes := convertInt64ToStringRecursive(res)
		return convertedRes
	}

	// 使用 GetConversationMessages API 获取会话消息
	ctx := context.Background()
	res, err := GetInstance().GetConversationMessages(ctx, body)
	if err != nil {
		exception.New("Get conversation messages failed: %s", 500, err.Error()).Throw()
	}

	// 将返回数据中的 int64 类型转换为字符串，避免 JavaScript 精度问题
	convertedRes := convertInt64ToStringRecursive(res)
	return convertedRes
}

// ProcessDestroyConversation 销毁会话
// 删除指定会话，清理相关数据
// 接口文档: https://www.volcengine.com/docs/6348/337036
func ProcessDestroyConversation(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	args := p.ArgsMap(0)

	// 使用配置文件中的AppId
	appID := int32(volcengine.VolcEngine.IM.AppID)

	conversationID, err := parseInt64FromArgs(args, "ConversationShortId")
	if err != nil {
		exception.New("ConversationShortId is required and must be a valid integer", 400).Throw()
	}

	// 构建请求体
	body := &BatchDeleteConversationParticipantBody{
		AppID:               appID,
		ConversationShortID: conversationID,
	}

	// 调用 API
	ctx := context.Background()
	res, err := GetInstance().BatchDeleteConversationParticipant(ctx, body)
	if err != nil {
		exception.New("Destroy conversation failed: %s", 500, err.Error()).Throw()
	}

	// 将返回数据中的 int64 类型转换为字符串，避免 JavaScript 精度问题
	convertedRes := convertInt64ToStringRecursive(res)
	return convertedRes
}
