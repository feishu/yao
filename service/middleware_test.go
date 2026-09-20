package service

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/share"
)

func TestWithInFlightTrackOverloadProtection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(withInFlightTrack)

	// Set concurrency limit to 2
	SetMaxConcurrency(2)
	defer SetMaxConcurrency(0)

	proceed := make(chan struct{})
	router.GET("/test-slow", func(c *gin.Context) {
		<-proceed
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	var wg sync.WaitGroup
	statusCodeCh := make(chan int, 3)

	// Launch 2 requests that block on proceed
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/test-slow", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			statusCodeCh <- w.Code
		}()
	}

	// Give goroutines time to enter
	time.Sleep(30 * time.Millisecond)

	// Third request should be rejected immediately with 429
	req := httptest.NewRequest(http.MethodGet, "/test-slow", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Contains(t, w.Body.String(), "Server overloaded")

	// Release first two requests
	close(proceed)
	wg.Wait()
	close(statusCodeCh)

	for code := range statusCodeCh {
		assert.Equal(t, http.StatusOK, code)
	}
}

func TestWithRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(withInFlightTrack, withRecovery)

	router.GET("/test-panic", func(c *gin.Context) {
		panic("simulated unexpected crash in handler")
	})

	initialInFlight := share.GlobalInFlight.Count()

	req := httptest.NewRequest(http.MethodGet, "/test-panic", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 断言拦截 panic 并返回统一的 500 JSON
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Internal Server Error")
	assert.Contains(t, w.Body.String(), `"code":500`)

	// 断言在途请求计数器未泄漏，正确归零
	assert.Equal(t, initialInFlight, share.GlobalInFlight.Count())
}

