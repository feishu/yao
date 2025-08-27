package im

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/volcengine"
)

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
	})
}

// ProcessGetAppToken 获取火山引擎IM的AppToken
// 用于客户端鉴权使用
// 接口文档: https://www.volcengine.com/docs/6348/435387
func ProcessGetAppToken(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	args := p.ArgsMap(0)

	// 获取用户ID
	userID, ok := args["UserId"].(int64)
	if !ok {
		exception.New("UserId is required", 400).Throw()
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
		"UserId":     int64(userID),
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

		userID, ok := userMap["UserId"].(float64)
		if !ok {
			exception.New("User.UserId is required and must be a number", 400).Throw()
		}

		userItem := RegisterUsersBodyUsersItem{
			UserID: int64(userID),
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

	return res
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

	userIDsInt := make([]int64, 0, len(userIDs))
	for _, userID := range userIDs {
		userIDsInt = append(userIDsInt, int64(userID.(float64)))
	}

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

	return res
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

		userID, ok := userMap["UserId"].(float64)
		if !ok {
			exception.New("User.UserId is required and must be a number", 400).Throw()
		}

		userItem := BatchUpdateUserBodyUsersItem{
			UserID: int64(userID),
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

	return res
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

	userIDsInt := make([]int64, 0, len(userIDs))
	for _, userID := range userIDs {
		userIDsInt = append(userIDsInt, int64(userID.(float64)))
	}

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

	return res
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

	if owner, ok := args["Owner"].(float64); ok {
		body.OwnerUserID = int64(owner)
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
	if otherUserId, ok := args["OtherUserId"].(float64); ok {
		otherUserID := int64(otherUserId)
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

	return res
}

// ProcessModifyConversation 修改会话信息
// 可修改会话名称、描述等属性
// 接口文档: https://www.volcengine.com/docs/6348/337115
func ProcessModifyConversation(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	args := p.ArgsMap(0)

	// 使用配置文件中的AppId
	appID := int32(volcengine.VolcEngine.IM.AppID)

	conversationID, ok := args["ConversationShortId"].(float64)
	if !ok {
		exception.New("ConversationShortId is required", 400).Throw()
	}

	// 构建请求体
	body := &ModifyConversationBody{
		AppID: appID,
		ConversationCoreInfo: ModifyConversationBodyConversationCoreInfo{
			ConversationShortID: int64(conversationID),
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

	return res
}

// ProcessIsUserInConversation 检查用户是否在指定会话中
// 返回用户是否是会话成员的信息
// 接口文档: https://www.volcengine.com/docs/6348/336996
func ProcessIsUserInConversation(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	args := p.ArgsMap(0)

	// 使用配置文件中的AppId
	appID := int32(volcengine.VolcEngine.IM.AppID)

	conversationID, ok := args["ConversationShortId"].(float64)
	if !ok {
		exception.New("ConversationShortId is required", 400).Throw()
	}

	// 优先使用 ParticipantUserId，如果没有则使用 UserId
	var userID int64
	if participantUserID, ok := args["ParticipantUserId"].(float64); ok {
		userID = int64(participantUserID)
	} else if userId, ok := args["UserId"].(float64); ok {
		userID = int64(userId)
	} else {
		exception.New("UserId or ParticipantUserId is required", 400).Throw()
	}

	// 构建请求体
	body := &IsUserInConversationBody{
		AppID:               appID,
		ConversationShortID: int64(conversationID),
		ParticipantUserID:   userID,
	}

	// 调用 API
	ctx := context.Background()
	res, err := GetInstance().IsUserInConversation(ctx, body)
	if err != nil {
		exception.New("Check user in conversation failed: %s", 500, err.Error()).Throw()
	}

	return res
}

// ProcessSendMessage 发送消息
// 支持发送文本、图片、视频等多种类型消息
// 接口文档: https://www.volcengine.com/docs/6348/337135
func ProcessSendMessage(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	args := p.ArgsMap(0)

	// 使用配置文件中的AppId
	appID := int32(volcengine.VolcEngine.IM.AppID)

	conversationID, ok := args["ConversationShortId"].(float64)
	if !ok {
		exception.New("ConversationShortId is required", 400).Throw()
	}

	senderID, ok := args["SenderUserId"].(float64)
	if !ok {
		exception.New("SenderUserId is required", 400).Throw()
	}

	content, ok := args["Content"].(string)
	if !ok {
		exception.New("Content is required", 400).Throw()
	}

	// 构建请求体
	body := &SendMessageBody{
		AppID:               appID,
		ConversationShortID: int64(conversationID),
		Sender:              int64(senderID),
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
		mentionedUserIDs := make([]*int64, 0, len(mentionedUsers))
		for _, user := range mentionedUsers {
			if userFloat, ok := user.(float64); ok {
				userID := int64(userFloat)
				mentionedUserIDs = append(mentionedUserIDs, &userID)
			}
		}
		body.MentionedUsers = mentionedUserIDs
	}

	// 处理 VisibleUsers 参数（可见用户列表）
	if visibleUsers, ok := args["VisibleUsers"].([]interface{}); ok {
		visibleUserIDs := make([]*int64, 0, len(visibleUsers))
		for _, user := range visibleUsers {
			if userFloat, ok := user.(float64); ok {
				userID := int64(userFloat)
				visibleUserIDs = append(visibleUserIDs, &userID)
			}
		}
		body.VisibleUsers = visibleUserIDs
	}

	// 处理 InvisibleUsers 参数（不可见用户列表）
	if invisibleUsers, ok := args["InvisibleUsers"].([]interface{}); ok {
		invisibleUserIDs := make([]*int64, 0, len(invisibleUsers))
		for _, user := range invisibleUsers {
			if userFloat, ok := user.(float64); ok {
				userID := int64(userFloat)
				invisibleUserIDs = append(invisibleUserIDs, &userID)
			}
		}
		body.InvisibleUsers = invisibleUserIDs
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
	if createTime, ok := args["CreateTime"].(float64); ok {
		createTimeInt := int64(createTime)
		body.CreateTime = &createTimeInt
	}

	// 处理 RefMsgInfo 参数（引用消息）
	if refMsgInfo, ok := args["RefMsgInfo"].(map[string]interface{}); ok {
		refInfo := &SendMessageBodyRefMsgInfo{}

		if referencedMsgId, ok := refMsgInfo["ReferencedMessageId"].(float64); ok {
			refInfo.ReferencedMessageID = int64(referencedMsgId)
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

	return res
}

// ProcessRecallMessage 撤回消息
// 允许用户撤回已发送的消息
// 接口文档: https://www.volcengine.com/docs/6348/337141
func ProcessRecallMessage(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	args := p.ArgsMap(0)

	// 使用配置文件中的AppId
	appID := int32(volcengine.VolcEngine.IM.AppID)

	conversationID, ok := args["ConversationShortId"].(float64)
	if !ok {
		exception.New("ConversationShortId is required", 400).Throw()
	}

	// 获取消息ID并转换为int64
	var messageIDInt64 int64
	if messageIDStr, ok := args["MessageId"].(string); ok {
		var err error
		messageIDInt64, err = strconv.ParseInt(messageIDStr, 10, 64)
		if err != nil {
			exception.New("MessageId must be a valid integer", 400).Throw()
		}
	} else if messageIDFloat, ok := args["MessageId"].(float64); ok {
		messageIDInt64 = int64(messageIDFloat)
	} else {
		exception.New("MessageId is required", 400).Throw()
	}

	// 获取用户ID（可选）
	var participantUserID int64
	if userID, ok := args["ParticipantUserId"].(float64); ok {
		participantUserID = int64(userID)
	}

	// 构建请求体
	body := &RecallMessageBody{
		AppID:               appID,
		ConversationShortID: int64(conversationID),
		MessageID:           messageIDInt64,
		ParticipantUserID:   participantUserID,
	}

	// 调用 API
	ctx := context.Background()
	res, err := GetInstance().RecallMessage(ctx, body)
	if err != nil {
		exception.New("Recall message failed: %s", 500, err.Error()).Throw()
	}

	return res
}

// ProcessDeleteConversationMessage 删除会话消息
// 从会话中删除指定消息
// 接口文档: https://www.volcengine.com/docs/6348/337140
func ProcessDeleteConversationMessage(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	args := p.ArgsMap(0)

	// 使用配置文件中的AppId
	appID := int32(volcengine.VolcEngine.IM.AppID)

	conversationID, ok := args["ConversationShortId"].(float64)
	if !ok {
		exception.New("ConversationShortId is required", 400).Throw()
	}

	// 获取消息ID并转换为int64
	var messageIDInt64 int64
	if messageIDStr, ok := args["MessageId"].(string); ok {
		var err error
		messageIDInt64, err = strconv.ParseInt(messageIDStr, 10, 64)
		if err != nil {
			exception.New("MessageId must be a valid integer", 400).Throw()
		}
	} else if messageIDFloat, ok := args["MessageId"].(float64); ok {
		messageIDInt64 = int64(messageIDFloat)
	} else {
		exception.New("MessageId is required", 400).Throw()
	}

	// 构建请求体
	body := &DeleteConversationMessageBody{
		AppID:               appID,
		ConversationShortID: int64(conversationID),
		MessageID:           messageIDInt64,
	}

	// 调用 API
	ctx := context.Background()
	res, err := GetInstance().DeleteConversationMessage(ctx, body)
	if err != nil {
		exception.New("Delete message failed: %s", 500, err.Error()).Throw()
	}

	return res
}

// ProcessGetConversationMessages 获取会话消息列表
// 根据会话ID获取会话中的消息列表
// 接口文档: https://www.volcengine.com/docs/6348/337138
func ProcessGetConversationMessages(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	args := p.ArgsMap(0)

	// 使用配置文件中的AppId
	appID := int32(volcengine.VolcEngine.IM.AppID)

	conversationID, ok := args["ConversationShortId"].(float64)
	if !ok {
		exception.New("ConversationShortId is required", 400).Throw()
	}

	// 构建请求体
	body := &GetConversationMessagesBody{
		AppID:               appID,
		ConversationShortID: int64(conversationID),
	}

	// 处理 Cursor 参数（查询起始位置）
	if cursor, ok := args["Cursor"].(float64); ok {
		body.Cursor = int64(cursor)
	}

	// 处理 Limit 参数（查询条数）
	if limit, ok := args["Limit"].(float64); ok {
		body.Limit = int64(limit)
	}

	// 处理 Reverse 参数（查询方向）
	if reverse, ok := args["Reverse"].(float64); ok {
		reverseInt := int32(reverse)
		body.Reverse = &reverseInt
	}

	// 构建消息ID列表
	var messageIDs []int64
	if msgIDList, ok := args["MessageIds"].([]interface{}); ok {
		for _, id := range msgIDList {
			if idFloat, ok := id.(float64); ok {
				messageIDs = append(messageIDs, int64(idFloat))
			} else if idStr, ok := id.(string); ok {
				idInt64, err := strconv.ParseInt(idStr, 10, 64)
				if err == nil {
					messageIDs = append(messageIDs, idInt64)
				}
			}
		}
	}

	// 如果提供了消息ID列表，则使用 GetMessages API
	if len(messageIDs) > 0 {
		getMsgBody := &GetMessagesBody{
			AppID:               appID,
			ConversationShortID: int64(conversationID),
			MessageIDs:          messageIDs,
		}

		ctx := context.Background()
		res, err := GetInstance().GetMessages(ctx, getMsgBody)
		if err != nil {
			exception.New("Get conversation messages failed: %s", 500, err.Error()).Throw()
		}

		return res
	}

	// 使用 GetConversationMessages API 获取会话消息
	ctx := context.Background()
	res, err := GetInstance().GetConversationMessages(ctx, body)
	if err != nil {
		exception.New("Get conversation messages failed: %s", 500, err.Error()).Throw()
	}

	return res
}

// ProcessDestroyConversation 销毁会话
// 删除指定会话，清理相关数据
// 接口文档: https://www.volcengine.com/docs/6348/337036
func ProcessDestroyConversation(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	args := p.ArgsMap(0)

	// 使用配置文件中的AppId
	appID := int32(volcengine.VolcEngine.IM.AppID)

	conversationID, ok := args["ConversationShortId"].(float64)
	if !ok {
		exception.New("ConversationShortId is required", 400).Throw()
	}

	// 构建请求体
	body := &BatchDeleteConversationParticipantBody{
		AppID:               appID,
		ConversationShortID: int64(conversationID),
	}

	// 调用 API
	ctx := context.Background()
	res, err := GetInstance().BatchDeleteConversationParticipant(ctx, body)
	if err != nil {
		exception.New("Destroy conversation failed: %s", 500, err.Error()).Throw()
	}

	return res
}
