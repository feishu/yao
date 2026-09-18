package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/helper"
)

func init() {
	gin.SetMode(gin.TestMode)
	config.Conf = config.Config{
		JWTSecret: "test-jwt-secret-key-1234567890",
	}
}

func TestGuardBearerJWT_Valid(t *testing.T) {
	token := helper.JwtMake(1, map[string]interface{}{"name": "test"}, map[string]interface{}{
		"timeout": 3600,
		"sid":     "sid-123456",
	})

	router := gin.New()
	router.Use(guardBearerJWT)
	router.GET("/test", func(c *gin.Context) {
		sid, _ := c.Get("__sid")
		c.JSON(200, gin.H{"sid": sid})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token.Token)

	assert.NotPanics(t, func() {
		router.ServeHTTP(w, req)
	})

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "sid-123456")
}

func TestGuardBearerJWT_Expired(t *testing.T) {
	// Create token that expires immediately (-1 hour)
	token := helper.JwtMake(1, map[string]interface{}{"name": "expired"}, map[string]interface{}{
		"timeout": -3600,
		"sid":     "sid-expired",
	})

	router := gin.New()
	router.Use(guardBearerJWT)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token.Token)

	// MUST NOT PANIC!
	assert.NotPanics(t, func() {
		router.ServeHTTP(w, req)
	})

	assert.Equal(t, 401, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid token")
}

func TestGuardBearerJWT_InvalidFormat(t *testing.T) {
	router := gin.New()
	router.Use(guardBearerJWT)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-string")

	assert.NotPanics(t, func() {
		router.ServeHTTP(w, req)
	})

	assert.Equal(t, 401, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid token")
}

func TestGuardBearerJWT_Empty(t *testing.T) {
	router := gin.New()
	router.Use(guardBearerJWT)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)

	assert.NotPanics(t, func() {
		router.ServeHTTP(w, req)
	})

	assert.Equal(t, 403, w.Code)
}
