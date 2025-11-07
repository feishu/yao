package await

import (
    "fmt"
    "time"

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
