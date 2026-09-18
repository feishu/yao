package await

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

// setupV8Context 创建隔离的测试用 V8 环境
func setupV8Context(t *testing.T) (*v8go.Isolate, *v8go.Context) {
	iso := v8go.NewIsolate()
	ctx := v8go.NewContext(iso)
	return iso, ctx
}

func closeV8Context(iso *v8go.Isolate, ctx *v8go.Context) {
	if ctx != nil {
		ctx.Close()
	}
	if iso != nil {
		iso.Dispose()
	}
}

// TestProcessAwait_ResolveDirect 测试立即决议的 Promise
func TestProcessAwait_ResolveDirect(t *testing.T) {
	iso, ctx := setupV8Context(t)
	defer closeV8Context(iso, ctx)

	val, err := ctx.RunScript(`Promise.resolve("hello syncAwait")`, "test_resolve.js")
	assert.NoError(t, err)
	defer val.Release()

	goVal, err := bridge.GoValue(val, ctx)
	assert.NoError(t, err)

	proc := &process.Process{
		Args: []interface{}{goVal},
	}

	result := ProcessAwait(proc)
	assert.Equal(t, "hello syncAwait", result)
}

// TestProcessAwait_ResolveAsync 测试微任务异步决议的 Promise
func TestProcessAwait_ResolveAsync(t *testing.T) {
	iso, ctx := setupV8Context(t)
	defer closeV8Context(iso, ctx)

	val, err := ctx.RunScript(`
		new Promise((resolve) => {
			Promise.resolve().then(() => {
				resolve(98765);
			});
		})
	`, "test_async.js")
	assert.NoError(t, err)
	defer val.Release()

	goVal, err := bridge.GoValue(val, ctx)
	assert.NoError(t, err)

	proc := &process.Process{
		Args: []interface{}{goVal},
	}

	result := ProcessAwait(proc)
	assert.Equal(t, 98765, result)
}

// TestProcessAwait_Reject 测试被拒绝 (Reject) 的 Promise 错误捕获
func TestProcessAwait_Reject(t *testing.T) {
	iso, ctx := setupV8Context(t)
	defer closeV8Context(iso, ctx)

	val, err := ctx.RunScript(`Promise.reject("operation failed with reason")`, "test_reject.js")
	assert.NoError(t, err)
	defer val.Release()

	goVal, err := bridge.GoValue(val, ctx)
	assert.NoError(t, err)

	proc := &process.Process{
		Args: []interface{}{goVal},
	}

	result := ProcessAwait(proc)
	errResult, ok := result.(error)
	assert.True(t, ok, "result should be an error")
	assert.Contains(t, errResult.Error(), "promise rejected")
	assert.Contains(t, errResult.Error(), "operation failed with reason")
}

// TestProcessAwait_NonPromise 测试非 Promise 普通数值原样透传且防空指针解引用
func TestProcessAwait_NonPromise(t *testing.T) {
	// 测试普通字符串
	procStr := &process.Process{
		Args: []interface{}{"regular string"},
	}
	resStr := ProcessAwait(procStr)
	assert.Equal(t, "regular string", resStr)

	// 测试普通数字
	procInt := &process.Process{
		Args: []interface{}{12345},
	}
	resInt := ProcessAwait(procInt)
	assert.Equal(t, 12345, resInt)

	// 测试普通 nil
	procNil := &process.Process{
		Args: []interface{}{nil},
	}
	resNil := ProcessAwait(procNil)
	assert.Nil(t, resNil)
}

// TestProcessAwait_Function 测试传入返回 Promise 的函数
func TestProcessAwait_Function(t *testing.T) {
	iso, ctx := setupV8Context(t)
	defer closeV8Context(iso, ctx)

	val, err := ctx.RunScript(`(name) => Promise.resolve("Hello " + name)`, "test_func.js")
	assert.NoError(t, err)
	defer val.Release()

	goVal, err := bridge.GoValue(val, ctx)
	assert.NoError(t, err)

	proc := &process.Process{
		Args: []interface{}{goVal, "Yao"},
	}

	result := ProcessAwait(proc)
	assert.Equal(t, "Hello Yao", result)
}

// TestProcessAwait_WatchdogTimeout 测试伴生看门狗在 Promise 永久挂起时的强行超时打断
func TestProcessAwait_WatchdogTimeout(t *testing.T) {
	iso, ctx := setupV8Context(t)
	defer closeV8Context(iso, ctx)

	// 临时设置短超时：50毫秒
	originalDefault := timeoutConfig.DefaultTimeout
	SetTimeoutConfig(50*time.Millisecond, 500*time.Millisecond, 10*time.Millisecond)
	defer func() {
		SetTimeoutConfig(originalDefault, 60*time.Second, 100*time.Millisecond)
	}()

	val, err := ctx.RunScript(`new Promise(() => { /* 永久挂起不 resolve */ })`, "test_timeout.js")
	assert.NoError(t, err)
	defer val.Release()

	goVal, err := bridge.GoValue(val, ctx)
	assert.NoError(t, err)

	proc := &process.Process{
		Args: []interface{}{goVal},
	}

	start := time.Now()
	result := ProcessAwait(proc)
	duration := time.Since(start)

	errResult, ok := result.(error)
	assert.True(t, ok, "should return timeout error")
	assert.True(t, strings.Contains(errResult.Error(), "timeout") || strings.Contains(errResult.Error(), "watchdog"),
		"error should mention timeout or watchdog: %v", errResult)

	// 验证确实在短时间内被看门狗打断，绝不永久阻塞（在 50ms 超时后，应在 500ms 内安全返回）
	assert.True(t, duration < 800*time.Millisecond, "should return within 800ms, actual: %v", duration)
}

// TestProcessAwait_ContextCancelled 测试上游 Context 取消时的级联立即中断
func TestProcessAwait_ContextCancelled(t *testing.T) {
	iso, ctx := setupV8Context(t)
	defer closeV8Context(iso, ctx)

	val, err := ctx.RunScript(`new Promise(() => { /* 挂起 */ })`, "test_cancel.js")
	assert.NoError(t, err)
	defer val.Release()

	goVal, err := bridge.GoValue(val, ctx)
	assert.NoError(t, err)

	cancelCtx, cancel := context.WithCancel(context.Background())
	proc := &process.Process{
		Context: cancelCtx,
		Args:    []interface{}{goVal},
	}

	// 20毫秒后主动取消上游 Context
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	result := ProcessAwait(proc)
	duration := time.Since(start)

	errResult, ok := result.(error)
	assert.True(t, ok, "should return cancellation error")
	assert.Contains(t, errResult.Error(), "cancelled")
	// 验证在 300ms 内即刻响应取消
	assert.True(t, duration < 500*time.Millisecond, "should cancel within 500ms, actual: %v", duration)
}

// TestTimeoutConfig 测试超时配置的读写
func TestTimeoutConfig(t *testing.T) {
	SetTimeoutConfig(5*time.Second, 30*time.Second, 50*time.Millisecond)

	proc := &process.Process{}
	timeout := getTimeout(proc)
	assert.Equal(t, 5*time.Second, timeout)

	// 恢复默认
	SetTimeoutConfig(10*time.Second, 60*time.Second, 100*time.Millisecond)
}

// TestProcessAwait_Concurrent 测试多协程并发调用安全性
func TestProcessAwait_Concurrent(t *testing.T) {
	const count = 10
	errChan := make(chan error, count)

	for i := 0; i < count; i++ {
		go func(idx int) {
			iso, ctx := setupV8Context(t)
			defer closeV8Context(iso, ctx)

			val, err := ctx.RunScript(`Promise.resolve("res")`, "test_concurrent.js")
			if err != nil {
				errChan <- err
				return
			}
			defer val.Release()

			goVal, err := bridge.GoValue(val, ctx)
			if err != nil {
				errChan <- err
				return
			}

			proc := &process.Process{
				Args: []interface{}{goVal},
			}

			res := ProcessAwait(proc)
			if res != "res" {
				errChan <- assert.AnError
				return
			}
			errChan <- nil
		}(i)
	}

	for i := 0; i < count; i++ {
		err := <-errChan
		assert.NoError(t, err)
	}
}

// TestProcessAwait_PreCancelledContext 测试进入前即已取消的 Context 防御拦截
func TestProcessAwait_PreCancelledContext(t *testing.T) {
	iso, ctx := setupV8Context(t)
	defer closeV8Context(iso, ctx)

	val, err := ctx.RunScript(`Promise.resolve("should_not_run")`, "test_precancel.js")
	assert.NoError(t, err)
	defer val.Release()

	goVal, err := bridge.GoValue(val, ctx)
	assert.NoError(t, err)

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel() // 预先取消

	proc := &process.Process{
		Context: cancelCtx,
		Args:    []interface{}{goVal},
	}

	result := ProcessAwait(proc)
	errResult, ok := result.(error)
	assert.True(t, ok, "should return error when pre-cancelled")
	assert.Contains(t, errResult.Error(), "ProcessAwait cancelled")
}

