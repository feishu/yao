package share

import (
	"sync"
	"sync/atomic"
	"time"
)

// InFlightTracker 在途请求与任务追踪器，支持平滑热重载优雅排空（Drain）
type InFlightTracker struct {
	counter int64
	cond    *sync.Cond
	mu      sync.Mutex
}

// GlobalInFlight 全局在途请求追踪单例
var GlobalInFlight = NewInFlightTracker()

// NewInFlightTracker 创建在途请求追踪器
func NewInFlightTracker() *InFlightTracker {
	t := &InFlightTracker{}
	t.cond = sync.NewCond(&t.mu)
	return t
}

// Enter 标记一个在途请求进入，返回退出时的清理回调（零锁高并发）
func (t *InFlightTracker) Enter() func() {
	leave, _ := t.TryEnter(0)
	return leave
}

// TryEnter 尝试进入在途请求。如果指定了 limit > 0 且当前在途请求数已达到或超过该限制，则返回 (nil, false)；否则通过原子 CAS 递增并返回清理回调
func (t *InFlightTracker) TryEnter(limit int64) (func(), bool) {
	if limit > 0 {
		for {
			cur := atomic.LoadInt64(&t.counter)
			if cur >= limit {
				return nil, false
			}
			if atomic.CompareAndSwapInt64(&t.counter, cur, cur+1) {
				break
			}
		}
	} else {
		atomic.AddInt64(&t.counter, 1)
	}

	var once sync.Once
	return func() {
		once.Do(func() {
			val := atomic.AddInt64(&t.counter, -1)
			if val <= 0 {
				t.mu.Lock()
				t.cond.Broadcast()
				t.mu.Unlock()
			}
		})
	}, true
}

// Count 获取当前在途请求数
func (t *InFlightTracker) Count() int64 {
	return atomic.LoadInt64(&t.counter)
}

// Drain 等待所有在途请求安全排空或超时
func (t *InFlightTracker) Drain(timeout time.Duration) bool {
	if atomic.LoadInt64(&t.counter) <= 0 {
		return true
	}

	done := make(chan struct{})
	go func() {
		t.mu.Lock()
		defer t.mu.Unlock()
		for atomic.LoadInt64(&t.counter) > 0 {
			t.cond.Wait()
		}
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// InFlightEnter 全局进入在途请求
func InFlightEnter() func() {
	return GlobalInFlight.Enter()
}

// InFlightTryEnter 全局尝试进入在途请求（支持并发上限限制）
func InFlightTryEnter(limit int64) (func(), bool) {
	return GlobalInFlight.TryEnter(limit)
}

// DrainInFlight 全局排空在途请求
func DrainInFlight(timeout time.Duration) bool {
	return GlobalInFlight.Drain(timeout)
}
