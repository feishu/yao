package await

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"github.com/yaoapp/kun/log"
	"rogchap.com/v8go"
)

// 用于保护全局状态和配置的并发安全
var (
	globalMutex sync.RWMutex
	// 全局配置，可以从环境变量或配置文件中读取
	timeoutConfig = struct {
		DefaultTimeout time.Duration
		MaxTimeout     time.Duration
		MinTimeout     time.Duration
	}{
		DefaultTimeout: 10 * time.Second,
		MaxTimeout:     60 * time.Second,
		MinTimeout:     100 * time.Millisecond,
	}
)

// ProcessAwait 处理 await 表达式，支持处理 v8go.Value 类型的 Promise 和 bridge.FunctionT 类型的函数
// 优化点：添加了并发安全、性能优化和更好的错误处理
func ProcessAwait(process *process.Process) interface{} {
	// 验证参数
	process.ValidateArgNums(1)
	args := process.Args
	value := args[0]
	var promise *v8go.Value
	var ctx *v8go.Context

	// 场景 1：直接传入 Promise
	if prom, ok := value.(bridge.PromiseT); ok {
		promise = prom.Value()
		ctx = prom.Context()
	} else if fn, ok := value.(bridge.FunctionT); ok {
		// 场景 2：传入函数，需要调用函数并获取 Promise
		fnVal := fn.Value()
		if !fnVal.IsFunction() {
			return fmt.Errorf("argument is not a function")
		}

		ctx = fn.Context()

		// 调用函数并获取 Promise
		fnFunc, err := fnVal.AsFunction()
		if err != nil {
			return fmt.Errorf("failed to convert to function: %v", err)
		}

		// 性能优化：预分配切片容量，避免动态扩容
		jsArgs := make([]*v8go.Value, 0, len(args)-1)
		for _, arg := range args[1:] {
			jsVal, err := bridge.JsValue(ctx, arg)
			if err != nil {
				return fmt.Errorf("failed to convert argument to JS value: %v", err)
			}
			jsArgs = append(jsArgs, jsVal)
		}

		var val *v8go.Value

		// 性能优化：减少反射调用，使用 switch 而不是 reflect
		switch len(jsArgs) {
		case 0:
			val, err = fnFunc.Call(ctx.Global())
		case 1:
			val, err = fnFunc.Call(ctx.Global(), jsArgs[0])
		case 2:
			val, err = fnFunc.Call(ctx.Global(), jsArgs[0], jsArgs[1])
		case 3:
			val, err = fnFunc.Call(ctx.Global(), jsArgs[0], jsArgs[1], jsArgs[2])
		default:
			val, err = fnFunc.Call(ctx.Global(), jsArgs[0], jsArgs[1], jsArgs[2], jsArgs[3])
		}

		if err != nil {
			return fmt.Errorf("function call failed: %v", err)
		}

		defer func() {
			if val != nil {
				val.Release()
			}
			// 释放临时创建的 JS 值
			for _, jsVal := range jsArgs {
				if jsVal != nil {
					jsVal.Release()
				}
			}
		}()

		goValue, err := bridge.GoValue(val, ctx)
		if err != nil {
			return fmt.Errorf("failed to convert return value: %v", err)
		}

		switch value := goValue.(type) {
		case bridge.PromiseT:
			promise = value.Value()
			ctx = value.Context()
		default:
			return value
		}
	}

	if !promise.IsPromise() {
		return value
	}

	// 性能优化：从缓存的配置中获取超时时间
	timeout := getTimeout(process)
	result, err := waitPromiseWithContext(context.Background(), ctx, promise, timeout)
	if err != nil {
		log.Warn("ProcessAwait failed: %v", err)
		return err
	}

	return result
}

// getTimeout 从 process context 中获取优化的超时时间
func getTimeout(process *process.Process) time.Duration {
	globalMutex.RLock()
	defer globalMutex.RUnlock()

	// 可以从 process 的 context 中获取自定义超时时间
	// 目前使用全局配置，但保留扩展性
	return timeoutConfig.DefaultTimeout
}

// SetTimeoutConfig 设置超时配置，线程安全
func SetTimeoutConfig(defaultTimeout, maxTimeout, minTimeout time.Duration) {
	globalMutex.Lock()
	defer globalMutex.Unlock()

	if defaultTimeout > 0 {
		timeoutConfig.DefaultTimeout = defaultTimeout
	}
	if maxTimeout > 0 {
		timeoutConfig.MaxTimeout = maxTimeout
	}
	if minTimeout > 0 {
		timeoutConfig.MinTimeout = minTimeout
	}
}

// waitPromiseWithContext 使用 context 管理超时和取消，优化了性能和并发安全性
func waitPromiseWithContext(ctx context.Context, v8ctx *v8go.Context, promise *v8go.Value, timeout time.Duration) (interface{}, error) {
	// 安全检查
	if v8ctx == nil || promise == nil {
		return nil, fmt.Errorf("invalid context or promise")
	}

	// 使用 defer 确保 V8 对象资源正确释放
	var (
		promiseObj  *v8go.Object
		then        *v8go.Value
		onFulfilled *v8go.Function
		onRejected  *v8go.Function
	)

	defer func() {
		// 确保所有 V8 对象都被正确释放
		if onFulfilled != nil {
			onFulfilled.Release()
		}
		if onRejected != nil {
			onRejected.Release()
		}
		if then != nil {
			then.Release()
		}
		if promiseObj != nil {
			promiseObj.Release()
		}
	}()

	var err error
	promiseObj, err = promise.AsObject()
	if err != nil {
		return nil, fmt.Errorf("promise is not an object: %v", err)
	}

	then, err = promiseObj.Get("then")
	if err != nil {
		return nil, fmt.Errorf("not thenable: %v", err)
	}

	// 使用 context 进行优雅的超时和取消控制
	done := make(chan interface{}, 1)
	fail := make(chan error, 1)

	iso := v8ctx.Isolate()

	// 创建回调函数模板，并确保资源管理
	onFulfilledFn := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			done <- args[0]
		}
		return nil
	}).GetFunction(v8ctx)

	onRejectedFn := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			fail <- fmt.Errorf("promise rejected: %v", args[0])
		}
		return nil
	}).GetFunction(v8ctx)

	// 检查 context 状态
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("operation cancelled: %v", ctx.Err())
	default:
	}

	thenFn, err := then.AsFunction()
	if err != nil {
		return nil, fmt.Errorf("then is not a function: %v", err)
	}

	// 调用 Promise.then()
	_, err = thenFn.Call(promiseObj, onFulfilledFn, onRejectedFn)
	if err != nil {
		return nil, fmt.Errorf("failed to call promise.then: %v", err)
	}

	// 性能优化：使用自适应的时间间隔
	// 根据剩余时间动态调整检查频率
	deadline := time.Now().Add(timeout)
	initialCheckInterval := time.Millisecond
	maxCheckInterval := 10 * time.Millisecond
	checkInterval := initialCheckInterval

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("operation cancelled: %v", ctx.Err())
		case val := <-done:
			log.Debug("Promise resolved successfully")
			return bridge.GoValue(val.(*v8go.Value), v8ctx)
		case err := <-fail:
			log.Warn("Promise rejected: %v", err)
			return nil, err
		case <-time.After(checkInterval):
			v8ctx.PerformMicrotaskCheckpoint()

			// 性能优化：根据剩余时间动态调整检查间隔
			remaining := time.Until(deadline)
			if remaining < time.Second {
				checkInterval = 500 * time.Microsecond
			} else if remaining < 5*time.Second {
				checkInterval = initialCheckInterval
			} else {
				checkInterval = maxCheckInterval
			}

			if time.Now().After(deadline) {
				return nil, fmt.Errorf("promise timeout after %v (deadline exceeded)", timeout)
			}
		}
	}
}
