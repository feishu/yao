package await

import (
	"fmt"
	"time"

	"github.com/yaoapp/gou/process"
	v8 "github.com/yaoapp/gou/runtime/v8"
	"github.com/yaoapp/gou/runtime/v8/bridge"
	"rogchap.com/v8go"
)

// Register injects a global function `syncAwait(promise[, timeoutMs])` into the given context.
// It synchronously waits for a Promise to settle, returning the result or throwing the reason.
func Register(ctx *v8.Context) {
	if ctx == nil || ctx.Context == nil {
		return
	}
	ctx.WithFunction("syncAwait", func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			return bridge.JsException(info.Context(), "missing parameters")
		}

		val := args[0]
		if !val.IsPromise() { // Non-promise: return as-is for convenience
			return val
		}

		prom, err := val.AsPromise()
		if err != nil {
			return bridge.JsException(info.Context(), err)
		}

		var timeout time.Duration
		if len(args) > 1 && args[1].IsNumber() {
			ms := args[1].Integer()
			if ms > 0 {
				timeout = time.Duration(ms) * time.Millisecond
			}
		}

		start := time.Now()
		v8ctx := info.Context()

		for {
			switch prom.State() {
			case v8go.Fulfilled:
				return prom.Result()
			case v8go.Rejected:
				reason := prom.Result()
				return v8ctx.Isolate().ThrowException(reason)
			default:
				v8ctx.PerformMicrotaskCheckpoint()
				if timeout > 0 && time.Since(start) > timeout {
					return bridge.JsException(info.Context(), fmt.Errorf("syncAwait timeout after %v", timeout))
				}
				time.Sleep(100 * time.Microsecond)
			}
		}
	})
}

// ProcessAwait 处理 await 表达式，支持处理 v8go.Value 类型的 Promise 和 bridge.FunctionT 类型的函数
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
			return err
		}

		// 转换剩余参数为 JS 值
		jsArgs, err := bridge.JsValues(ctx, args[1:])
		if err != nil {
			return err
		}

		var val *v8go.Value

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
			return err
		}

		defer func() {
			if val != nil {
				val.Release()
			}
		}()

		goValue, err := bridge.GoValue(val, ctx)
		if err != nil {
			return err
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

	result, err := waitPromise(ctx, promise, 10*time.Second)
	if err != nil {
		return err
	}

	return result
}

func waitPromise(ctx *v8go.Context, promise *v8go.Value, timeout time.Duration) (interface{}, error) {
	promiseObj, err := promise.AsObject()
	if err != nil {
		return nil, fmt.Errorf("promise is not an object")
	}

	then, err := promiseObj.Get("then")
	if err != nil {
		return nil, fmt.Errorf("not thenable")
	}

	done := make(chan interface{}, 1)
	fail := make(chan error, 1)

	iso := ctx.Isolate()

	onFulfilledFn := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		done <- info.Args()[0]
		return nil
	}).GetFunction(ctx)

	onRejectedFn := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		fail <- fmt.Errorf("%v", info.Args()[0])
		return nil
	}).GetFunction(ctx)

	thenFn, err := then.AsFunction()
	if err != nil {
		return nil, fmt.Errorf("then is not a function")
	}
	if _, err = thenFn.Call(promiseObj, onFulfilledFn, onRejectedFn); err != nil {
		return nil, err
	}

	// ✅ 关键：自旋 + 驱动 V8 微任务队列
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ctx.PerformMicrotaskCheckpoint() // 执行挂起的 Promise 回调

		select {
		case val := <-done:
			return bridge.GoValue(val.(*v8go.Value), ctx)
		case err := <-fail:
			return nil, err
		default:
			time.Sleep(time.Millisecond) // 避免 CPU 100%
		}
	}

	return nil, fmt.Errorf("promise timeout after %v", timeout)
}
