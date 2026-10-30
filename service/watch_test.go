package service

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/engine"
)

func TestWatch(t *testing.T) {
	err := engine.Load(config.Conf, engine.LoadOption{})
	if err != nil {
		t.Fatal(err)
	}

	srv, err := Start(config.Conf)
	if err != nil {
		t.Fatal(err)
	}
	defer Stop(srv)

	done := make(chan uint8, 1)
	go Watch(srv, done)

	select {
	case <-time.After(200 * time.Millisecond):
		done <- 1
		return
	}
}

func TestDebounceAggregator_Coalescing(t *testing.T) {
	var count int64
	var lastBatch []string
	var mu sync.Mutex

	da := NewDebounceAggregator(40*time.Millisecond, func(files []string) {
		atomic.AddInt64(&count, 1)
		mu.Lock()
		lastBatch = files
		mu.Unlock()
	})
	defer da.Close()

	// 快速突发连续推送 6 个事件（包含重复文件，模拟编辑器保存行为）
	filesToPush := []string{
		"/scripts/service.js",
		"/scripts/service.js",
		"/apis/user.http.yao",
		"/models/user.mod.yao",
		"/scripts/service.js",
		"/apis/user.http.yao",
	}

	for _, f := range filesToPush {
		da.Push("WRITE", f)
		time.Sleep(2 * time.Millisecond)
	}

	// 等待防抖窗口过期
	time.Sleep(80 * time.Millisecond)

	// 验证：尽管推送了 6 次事件，回调只触发了 1 次
	assert.Equal(t, int64(1), atomic.LoadInt64(&count), "多重瞬时事件必须合并为单次触发")

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 3, len(lastBatch), "重复文件必须被自动去重")

	fileMap := make(map[string]bool)
	for _, f := range lastBatch {
		fileMap[f] = true
	}
	assert.True(t, fileMap["/scripts/service.js"])
	assert.True(t, fileMap["/apis/user.http.yao"])
	assert.True(t, fileMap["/models/user.mod.yao"])
}

func TestDebounceAggregator_SequentialBursts(t *testing.T) {
	var count int64

	da := NewDebounceAggregator(30*time.Millisecond, func(files []string) {
		atomic.AddInt64(&count, 1)
	})
	defer da.Close()

	// 第一波突发
	da.Push("WRITE", "/scripts/a.js")
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int64(1), atomic.LoadInt64(&count))

	// 第二波突发（距离上一波超过防抖窗口）
	da.Push("WRITE", "/scripts/b.js")
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int64(2), atomic.LoadInt64(&count))
}

func TestDebounceAggregator_Close(t *testing.T) {
	da := NewDebounceAggregator(50*time.Millisecond, func(files []string) {})
	da.Push("WRITE", "/scripts/c.js")
	da.Close()

	// Close 后再次 Push 不会 panic 且通道不泄露
	assert.NotPanics(t, func() {
		da.Push("WRITE", "/scripts/d.js")
		da.Close() // 幂等调用
	})
}

