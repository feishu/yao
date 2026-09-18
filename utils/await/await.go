package await

import (
	"context"
	"errors"
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

	// 防御性前置检查：若上游 Context 已取消或超时，直接返回，避免在失效上下文中继续操作 V8 资源
	if process.Context != nil {
		if err := process.Context.Err(); err != nil {
			return fmt.Errorf("ProcessAwait cancelled: %w", err)
		}
	}

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

	if promise == nil || !promise.IsPromise() {
		return value
	}

	// 性能优化：从缓存的配置中获取超时时间
	timeout := getTimeout(process)

	// 提取上游调用方 Context，级联构建超时上下文
	ctxCaller := process.Context
	if ctxCaller == nil {
		ctxCaller = context.Background()
	}
	waitCtx, cancel := context.WithTimeout(ctxCaller, timeout)
	defer cancel()

	result, err := waitPromiseWithContext(waitCtx, ctx, promise, timeout)
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

// waitPromiseWithContext 使用独立 Watchdog 协程管理超时与取消，结合底层 Isolate 中断机制彻底杜绝挂死
func waitPromiseWithContext(ctx context.Context, v8ctx *v8go.Context, promise *v8go.Value, timeout time.Duration) (interface{}, error) {
	// 安全检查
	if v8ctx == nil || promise == nil {
		return nil, fmt.Errorf("invalid context or promise")
	}

	// 检查 context 初始状态
	if err := ctx.Err(); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("Promise timeout after %v", timeout)
		}
		return nil, fmt.Errorf("Promise cancelled: %v", err)
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

	done := make(chan interface{}, 1)
	fail := make(chan error, 1)

	iso := v8ctx.Isolate()

	// 启动独立伴生看门狗协程 (Companion Watchdog)
	// 一旦超时或上游 Context 取消，通过线程安全的 TerminateExecution 强行打断卡住的微任务执行
	stopWatchdog := make(chan struct{})
	defer close(stopWatchdog)

	go func() {
		select {
		case <-ctx.Done():
			if iso != nil {
				iso.TerminateExecution()
			}
		case <-stopWatchdog:
			return
		}
	}()

	// 创建回调函数模板，并确保资源管理
	onFulfilled = v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			done <- args[0]
		}
		return nil
	}).GetFunction(v8ctx)

	onRejected = v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) > 0 {
			fail <- fmt.Errorf("promise rejected: %v", args[0])
		}
		return nil
	}).GetFunction(v8ctx)

	thenFn, err := then.AsFunction()
	if err != nil {
		return nil, fmt.Errorf("then is not a function: %v", err)
	}
	defer thenFn.Release()

	// 调用 Promise.then()
	_, err = thenFn.Call(promiseObj, onFulfilled, onRejected)
	if err != nil {
		return nil, fmt.Errorf("failed to call promise.then: %v", err)
	}

	deadline := time.Now().Add(timeout)
	initialCheckInterval := 500 * time.Microsecond
	maxCheckInterval := 10 * time.Millisecond
	checkInterval := initialCheckInterval

	for {
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, fmt.Errorf("Promise timeout after %v (execution terminated by watchdog)", timeout)
			}
			return nil, fmt.Errorf("Promise cancelled: %v", ctx.Err())

		case val := <-done:
			log.Debug("Promise resolved successfully")
			return bridge.GoValue(val.(*v8go.Value), v8ctx)

		case err := <-fail:
			log.Warn("Promise rejected: %v", err)
			return nil, err

		case <-time.After(checkInterval):
			if err := ctx.Err(); err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					return nil, fmt.Errorf("Promise timeout after %v (execution terminated by watchdog)", timeout)
				}
				return nil, fmt.Errorf("Promise cancelled: %v", err)
			}

			v8ctx.PerformMicrotaskCheckpoint()

			remaining := time.Until(deadline)
			if remaining < time.Second {
				checkInterval = 500 * time.Microsecond
			} else if remaining < 5*time.Second {
				checkInterval = 2 * time.Millisecond
			} else {
				checkInterval = maxCheckInterval
			}

			if time.Now().After(deadline) {
				if iso != nil {
					iso.TerminateExecution()
				}
				return nil, fmt.Errorf("Promise timeout after %v (deadline exceeded)", timeout)
			}
		}
	}
}
