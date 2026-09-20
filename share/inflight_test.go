package share

import (
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

