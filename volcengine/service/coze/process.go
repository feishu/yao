package coze

import (
	"context"
	"encoding/json"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
)

func init() {
	process.RegisterGroup("agent.coze", map[string]process.Handler{
		"getAppToken": ProcessGetAppToken,
	})
}

// ProcessGetAppToken 获取COZE JWT TOKEN
// 用于客户端鉴权使用
// 接口文档: https://api.coze.cn/api/permission/oauth2/token
func ProcessGetAppToken(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	config := p.ArgsMap(0)

	// 生成缓存键（使用clientID）
	var clientID string
	if id, ok := config["client_id"].(string); ok {
		clientID = id
	} else {
		log.Error("client_id is missing or not a string in config")
		exception.New("invalid config: client_id is required", 400).Throw()
	}

	// 生成缓存键
	cacheKey := GenerateCacheKey(clientID) // Assuming GenerateCacheKey is accessible

	// 从全局缓存中查找
	if cachedToken, found := DefaultTokenCache.Get(cacheKey); found {
		// DefaultTokenCache.Get already handles expiry internally by deleting expired tokens
		// or returning not found. An explicit IsExpired check here is redundant if Get guarantees freshness.
		// However, to be absolutely safe or if the cache implementation might change:
		if !cachedToken.IsExpired() {
			log.Info("Using cached token for client ID: %s from DefaultTokenCache", clientID)
			return cachedToken // Return *coze.TokenResponse directly
		}
		log.Info("Cached token for client ID: %s found in DefaultTokenCache but was expired or invalid", clientID)
		DefaultTokenCache.Delete(cacheKey) // Explicitly delete if found but expired
	}

	ctx := context.Background()
	var conf *OAuthConfig // This OAuthConfig is from the coze package (defined in base_model.go)

	// 将config转换为OAuthConfig
	jsonData, err := json.Marshal(config)
	if err != nil {
		log.Error("Failed to marshal config map to JSON: %v", err)
		exception.New("failed to marshal config map: %v", 500, err.Error()).Throw()
	}

	conf = &OAuthConfig{} // 初始化conf
	err = json.Unmarshal(jsonData, conf)
	if err != nil {
		log.Error("Failed to unmarshal JSON to OAuthConfig: %v", err)
		exception.New("failed to unmarshal config to OAuthConfig: %v", 500, err.Error()).Throw()
	}

	// 从配置加载OAuth客户端 (LoadOAuthAppFromConfig is from the coze package)
	// It returns the OAuthClient interface from base_model.go
	oauth, err := LoadOAuthAppFromConfig(conf)
	if err != nil {
		log.Error("Failed to load OAuth config: %v", err)
		exception.New("failed to load OAuth config: %v", 500, err.Error()).Throw()
	}

	// 获取访问令牌, resp is of type *coze.TokenResponse (from base_model.go)
	resp, err := oauth.GetAccessToken(ctx)
	if err != nil {
		log.Error("GetAppToken failed %s", err)
		exception.New("GetAppToken failed %s", 500, err.Error()).Throw()
	}

	// 缓存新的TokenResponse到全局缓存
	DefaultTokenCache.Set(cacheKey, resp) // resp is *coze.TokenResponse
	log.Info("New token for client ID: %s fetched and stored in DefaultTokenCache", clientID)

	// 返回 *coze.TokenResponse 结构
	return resp
}
