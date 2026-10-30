package share

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInFlightEnterAndLeave(t *testing.T) {
	tracker := NewInFlightTracker()
	assert.Equal(t, int64(0), tracker.Count())

	leave1 := tracker.Enter()
	assert.Equal(t, int64(1), tracker.Count())

	leave2 := tracker.Enter()
	assert.Equal(t, int64(2), tracker.Count())

	leave1()
	assert.Equal(t, int64(1), tracker.Count())

	// 幂等性，多次调用同一 leave 不重复扣减
	leave1()
	assert.Equal(t, int64(1), tracker.Count())

	leave2()
	assert.Equal(t, int64(0), tracker.Count())
}

func TestInFlightDrainSuccess(t *testing.T) {
	tracker := NewInFlightTracker()
	var wg sync.WaitGroup

	// 模拟 10 个在途请求
	for i := 0; i < 10; i++ {
		wg.Add(1)
		leave := tracker.Enter()
		go func() {
			defer wg.Done()
			time.Sleep(30 * time.Millisecond)
			leave()
		}()
	}

	start := time.Now()
	success := tracker.Drain(1 * time.Second)
	duration := time.Since(start)

	assert.True(t, success)
	assert.Equal(t, int64(0), tracker.Count())
	assert.GreaterOrEqual(t, duration, 25*time.Millisecond)
	wg.Wait()
}

func TestInFlightDrainTimeout(t *testing.T) {
	tracker := NewInFlightTracker()
	leave := tracker.Enter()
	defer leave()

	start := time.Now()
	success := tracker.Drain(50 * time.Millisecond)
	duration := time.Since(start)

	assert.False(t, success, "Drain should time out when request is still in-flight")
	assert.GreaterOrEqual(t, duration, 45*time.Millisecond)
}

func TestInFlightTryEnter(t *testing.T) {
	tracker := NewInFlightTracker()
	limit := int64(2)

	leave1, ok1 := tracker.TryEnter(limit)
	assert.True(t, ok1)
	assert.NotNil(t, leave1)
	assert.Equal(t, int64(1), tracker.Count())

	leave2, ok2 := tracker.TryEnter(limit)
	assert.True(t, ok2)
	assert.NotNil(t, leave2)
	assert.Equal(t, int64(2), tracker.Count())

	// Reached limit
	leave3, ok3 := tracker.TryEnter(limit)
	assert.False(t, ok3)
	assert.Nil(t, leave3)
	assert.Equal(t, int64(2), tracker.Count())

	// Leave one, then we can enter again
	leave1()
	assert.Equal(t, int64(1), tracker.Count())

	leave4, ok4 := tracker.TryEnter(limit)
	assert.True(t, ok4)
	assert.NotNil(t, leave4)
	assert.Equal(t, int64(2), tracker.Count())

	leave2()
	leave4()
	assert.Equal(t, int64(0), tracker.Count())
}

func TestInFlightDrainTimeoutZeroLeak(t *testing.T) {
	tracker := NewInFlightTracker()
	leave := tracker.Enter()
	defer leave()

	// 强制 GC 稳定当前 goroutine 数量
	runtime.GC()
	time.Sleep(10 * time.Millisecond)
	initialGoroutines := runtime.NumGoroutine()

	// 连续触发 10 次超时排空
	for i := 0; i < 10; i++ {
		success := tracker.Drain(10 * time.Millisecond)
		assert.False(t, success, "预期排空超时")
	}

	// 再次 GC 并检查协程数量，验证零协程泄漏
	runtime.GC()
	time.Sleep(10 * time.Millisecond)
	finalGoroutines := runtime.NumGoroutine()

	// 协程数不应增加
	assert.LessOrEqual(t, finalGoroutines, initialGoroutines+1, "超时后后台等待协程必须确定性退出，杜绝协程累积泄漏")
}

func TestInFlightDrainingState(t *testing.T) {
	tracker := NewInFlightTracker()
	assert.False(t, tracker.IsDraining())

	tracker.SetDraining(true)
	assert.True(t, tracker.IsDraining())

	tracker.SetDraining(false)
	assert.False(t, tracker.IsDraining())
}


