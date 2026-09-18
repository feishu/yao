package asynq

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
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

func TestIsConfigured(t *testing.T) {
	// 清理环境变量
	os.Unsetenv("YAO_SESSION_STORE")
	os.Unsetenv("YAO_REDIS_HOST")
	os.Unsetenv("YAO_REDIS_PORT")
	os.Unsetenv("YAO_SESSION_HOST")
	config.Conf.Session.Store = "file"

	assert.False(t, IsConfigured())

	// 1. Session.Store = redis
	config.Conf.Session.Store = "redis"
	assert.True(t, IsConfigured())
	config.Conf.Session.Store = "file"

	// 2. YAO_SESSION_STORE = redis
	os.Setenv("YAO_SESSION_STORE", "redis")
	assert.True(t, IsConfigured())
	os.Unsetenv("YAO_SESSION_STORE")

	// 3. YAO_REDIS_HOST
	os.Setenv("YAO_REDIS_HOST", "192.168.1.100")
	assert.True(t, IsConfigured())
	os.Unsetenv("YAO_REDIS_HOST")

	// 4. YAO_SESSION_HOST
	os.Setenv("YAO_SESSION_HOST", "39.101.71.171")
	assert.True(t, IsConfigured())
	os.Unsetenv("YAO_SESSION_HOST")
}

func TestCheckConnection(t *testing.T) {
	// 检测一个几乎不可能开放的端口
	reachable, err := CheckConnection("127.0.0.1:59999", 50*time.Millisecond)
	assert.False(t, reachable)
	assert.Error(t, err)
}

func TestAsynqUnconfiguredSkip(t *testing.T) {
	Stop()

	// 模拟未配置 Redis，且指向一个不可达地址
	os.Unsetenv("YAO_SESSION_STORE")
	os.Unsetenv("YAO_REDIS_HOST")
	os.Unsetenv("YAO_REDIS_PORT")
	os.Unsetenv("YAO_SESSION_HOST")
	config.Conf.Session.Store = "file"
	config.Conf.Session.Host = "127.0.0.1"
	config.Conf.Session.Port = "59998"

	// 启动 Server 应该直接跳过，不会 panic，也不会启动 server
	StartServer()
	assert.False(t, IsEnabled())
	assert.Nil(t, server)

	// 获取 Client 应该返回明确的未配置/不可达错误，而不是抛出未捕获异常
	cli, err := GetClient()
	assert.Nil(t, cli)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unreachable")
}

func TestAsynqConfiguredUnreachable(t *testing.T) {
	Stop()

	// 显式配置了 Redis，但地址不可达
	os.Setenv("YAO_REDIS_HOST", "127.0.0.1")
	os.Setenv("YAO_REDIS_PORT", "59997")
	defer func() {
		os.Unsetenv("YAO_REDIS_HOST")
		os.Unsetenv("YAO_REDIS_PORT")
	}()

	StartServer()
	assert.False(t, IsEnabled())
	assert.Nil(t, server)

	cli, err := GetClient()
	assert.Nil(t, cli)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Configured Redis server")
}

func TestAsynqLogger(t *testing.T) {
	adapter := &asynqLogAdapter{}
	// 验证适配器日志输出不抛 panic
	assert.NotPanics(t, func() {
		adapter.Debug("test debug")
		adapter.Info("test info")
		adapter.Warn("test warn")
		adapter.Error("test error")
	})
}
