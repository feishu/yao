package im

import (
	"context"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func init() {
	process.RegisterGroup("volcengine.im", map[string]process.Handler{
		"getConversationMarks": ProcessGetConversationMarks,
		"markConversation":     ProcessMarkConversation,
		"createConversation":   ProcessCreateConversation,
		"sendMessage":          ProcessSendMessage,
	})
}

// ProcessGetConversationMarks volcengine.im.getConversationMarks
func ProcessGetConversationMarks(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	region := process.ArgsString(0)
	arg := process.Args[1]

	im := NewInstanceWithRegion(region)
	ctx := context.Background()
	res, err := im.GetConversationMarks(ctx, arg)
	if err != nil {
		exception.New("GetConversationMarks error: %s", 400, err).Throw()
	}
	return res
}

// ProcessMarkConversation volcengine.im.markConversation
func ProcessMarkConversation(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	region := process.ArgsString(0)
	arg := process.Args[1]

	im := NewInstanceWithRegion(region)
	ctx := context.Background()
	res, err := im.MarkConversation(ctx, arg)
	if err != nil {
		exception.New("MarkConversation error: %s", 400, err).Throw()
	}
	return res
}

// ProcessCreateConversation volcengine.im.createConversation
func ProcessCreateConversation(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	region := process.ArgsString(0)
	arg := process.Args[1]

	im := NewInstanceWithRegion(region)
	ctx := context.Background()
	res, err := im.CreateConversation(ctx, arg)
	if err != nil {
		exception.New("CreateConversation error: %s", 400, err).Throw()
	}
	return res
}

// ProcessSendMessage volcengine.im.sendMessage
func ProcessSendMessage(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	region := process.ArgsString(0)
	arg := process.Args[1]

	im := NewInstanceWithRegion(region)
	ctx := context.Background()
	res, err := im.SendMessage(ctx, arg)
	if err != nil {
		exception.New("SendMessage error: %s", 400, err).Throw()
	}
	return res
}
