package asynq

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestAsynqProcess(t *testing.T) {
	// 测试参数解析与结构体
	payload := TaskPayload{
		Process: "scripts.test.Handle",
		Args:    []interface{}{"order:123"},
	}
	assert.Equal(t, "scripts.test.Handle", payload.Process)
	assert.Equal(t, 1, len(payload.Args))

	// 测试 RedisOpt 读取
	opt := GetRedisOpt()
	assert.NotEmpty(t, opt.Addr)
}

func TestProcessTimeParse(t *testing.T) {
	// 1. 测试 RFC3339 字符串解析
	isoStr := "2026-08-05T16:00:00Z"
	parsed, err := time.Parse(time.RFC3339, isoStr)
	assert.Nil(t, err)
	assert.Equal(t, 2026, parsed.Year())

	// 2. 测试标准时间格式 "YYYY-MM-DD HH:mm:ss"
	stdStr := "2026-08-05 16:00:00"
	parsedStd, errStd := time.Parse("2006-01-02 15:04:05", stdStr)
	assert.Nil(t, errStd)
	assert.Equal(t, 2026, parsedStd.Year())

	// 3. 测试 ProcessEnqueueAt 参数提取逻辑
	p := process.New("utils.asynq.EnqueueAt", "scripts.order.Cancel", []interface{}{"123"}, "2026-08-05T16:00:00Z")
	assert.Equal(t, 3, p.NumOfArgs())
	assert.Equal(t, "scripts.order.Cancel", p.ArgsString(0))
}

func TestProcessEnqueueInValidation(t *testing.T) {
	// 测试参数验证
	p := process.New("utils.asynq.EnqueueIn", "scripts.order.Cancel", []interface{}{"123"}, 60)
	assert.Equal(t, 3, p.NumOfArgs())
	assert.Equal(t, "scripts.order.Cancel", p.ArgsString(0))
	assert.Equal(t, 60, p.ArgsInt(2))
}

func TestProcessCancelValidation(t *testing.T) {
	// 测试 Cancel 参数验证
	p := process.New("utils.asynq.Cancel", "task-123456")
	assert.Equal(t, 1, p.NumOfArgs())
	assert.Equal(t, "task-123456", p.ArgsString(0))
}
