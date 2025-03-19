package im

import (
	"context"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func init() {
	process.RegisterGroup("volcengine.im", map[string]process.Handler{
		"getConversationMarks":         ProcessGetConversationMarks,
		"markConversation":             ProcessMarkConversation,
		"createConversation":           ProcessCreateConversation,
		"sendMessage":                  ProcessSendMessage,
		"getMessages":                  ProcessGetMessages,
		"deleteMessage":                ProcessDeleteMessage,
		"batchAddBlockParticipants":    ProcessBatchAddBlockParticipants,
		"batchDeleteBlockParticipants": ProcessBatchDeleteBlockParticipants,
		"batchGetBlockParticipants":    ProcessBatchGetBlockParticipants,
		"modifyParticipantReadIndex":   ProcessModifyParticipantReadIndex,
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

// ProcessGetMessages volcengine.im.getMessages
func ProcessGetMessages(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	region := process.ArgsString(0)
	arg := process.Args[1]

	im := NewInstanceWithRegion(region)
	ctx := context.Background()
	res, err := im.GetMessages(ctx, arg)
	if err != nil {
		exception.New("GetMessages error: %s", 400, err).Throw()
	}
	return res
}

// ProcessDeleteMessage volcengine.im.deleteMessage
func ProcessDeleteMessage(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	region := process.ArgsString(0)
	arg := process.Args[1]

	im := NewInstanceWithRegion(region)
	ctx := context.Background()
	res, err := im.DeleteMessage(ctx, arg)
	if err != nil {
		exception.New("DeleteMessage error: %s", 400, err).Throw()
	}
	return res
}

// ProcessBatchAddBlockParticipants volcengine.im.batchAddBlockParticipants
func ProcessBatchAddBlockParticipants(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	region := process.ArgsString(0)
	arg := process.Args[1]

	im := NewInstanceWithRegion(region)
	ctx := context.Background()
	res, err := im.BatchAddBlockParticipants(ctx, arg)
	if err != nil {
		exception.New("BatchAddBlockParticipants error: %s", 400, err).Throw()
	}
	return res
}

// ProcessBatchDeleteBlockParticipants volcengine.im.batchDeleteBlockParticipants
func ProcessBatchDeleteBlockParticipants(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	region := process.ArgsString(0)
	arg := process.Args[1]

	im := NewInstanceWithRegion(region)
	ctx := context.Background()
	res, err := im.BatchDeleteBlockParticipants(ctx, arg)
	if err != nil {
		exception.New("BatchDeleteBlockParticipants error: %s", 400, err).Throw()
	}
	return res
}

// ProcessBatchGetBlockParticipants volcengine.im.batchGetBlockParticipants
func ProcessBatchGetBlockParticipants(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	region := process.ArgsString(0)
	arg := process.Args[1]

	im := NewInstanceWithRegion(region)
	ctx := context.Background()
	res, err := im.BatchGetBlockParticipants(ctx, arg)
	if err != nil {
		exception.New("BatchGetBlockParticipants error: %s", 400, err).Throw()
	}
	return res
}

// ProcessModifyParticipantReadIndex volcengine.im.modifyParticipantReadIndex
func ProcessModifyParticipantReadIndex(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	region := process.ArgsString(0)
	arg := process.Args[1]

	im := NewInstanceWithRegion(region)
	ctx := context.Background()
	res, err := im.ModifyParticipantReadIndex(ctx, arg)
	if err != nil {
		exception.New("ModifyParticipantReadIndex error: %s", 400, err).Throw()
	}
	return res
}
