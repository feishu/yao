package coze

import (
	"sync"
	"time"
)

// TokenCache 接口定义了Token缓存的基本操作
type TokenCache interface {
	Get(key string) (*TokenResponse, bool)
	Set(key string, token *TokenResponse)
	Delete(key string)
}

// MemoryTokenCache 是基于内存的Token缓存实现
type MemoryTokenCache struct {
	cache sync.Map
}

// NewMemoryTokenCache 创建一个新的内存Token缓存
func NewMemoryTokenCache() *MemoryTokenCache {
	return &MemoryTokenCache{
		cache: sync.Map{},
	}
}

// Get 根据键获取Token
func (c *MemoryTokenCache) Get(key string) (*TokenResponse, bool) {
	if value, ok := c.cache.Load(key); ok {
		token := value.(*TokenResponse)
		// 如果Token已过期，则从缓存中删除
		if token.IsExpired() {
			c.Delete(key)
			return nil, false
		}
		return token, true
	}
	return nil, false
}

// Set 设置Token到缓存中
func (c *MemoryTokenCache) Set(key string, token *TokenResponse) {
	c.cache.Store(key, token)

	// 设置过期自动清理
	remaining := token.Remaining()
	if remaining > 0 {
		go func() {
			// 在过期前30秒(或Token生命周期的90%，取较小值)清除缓存
			bufferTime := int64(TokenExpiryBufferSeconds)
			if remaining < bufferTime*10 { // 如果Token生命周期很短
				bufferTime = remaining / 10 // 使用10%的生命周期作为缓冲
			}

			cleanupTime := remaining - bufferTime
			if cleanupTime <= 0 {
				cleanupTime = 1 // 至少等待1秒
			}

			time.Sleep(time.Duration(cleanupTime) * time.Second)
			c.Delete(key)
		}()
	}
}

// Delete 从缓存中删除Token
func (c *MemoryTokenCache) Delete(key string) {
	c.cache.Delete(key)
}

// 创建全局缓存实例
var (
	// DefaultTokenCache 是全局默认的Token缓存
	DefaultTokenCache TokenCache = NewMemoryTokenCache()
)
