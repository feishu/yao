package coze

import (
	"context"
	"fmt"
	"time"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
)

type TokenResponse struct {
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    string `json:"expires_in"`
}

func init() {
	process.RegisterGroup("agent.coze", map[string]process.Handler{
		"getAppToken": ProcessGetAppToken,
	})
}

func timestampToDateTime(timestamp int64) string {
	t := time.Unix(timestamp, 0)
	return t.Format("2006-01-02 15:04:05")
}

// ProcessGetAppToken 获取COZE JWT TOKEN
// 用于客户端鉴权使用
// 接口文档: https://api.coze.cn/api/permission/oauth2/token
func ProcessGetAppToken(p *process.Process) interface{} {
	p.ValidateArgNums(1)
	config := p.ArgsMap(0)
	ext := p.ArgsString(1)

	ctx := context.Background()

	conf := OAuthConfig{}

	log.Info("config %s", config)

	mapToObj(ext, config, &conf)

	log.Info("conf %s", conf)

	oauth, err := LoadOAuthAppFromConfig(&conf)

	if err != nil {
		exception.New("failed to load OAuth config: %v", 500, err.Error()).Throw()
		// return nil, fmt.Errorf("failed to load OAuth config: %v", err)
	}

	jwtClient, ok := oauth.(*JWTOAuthClient)
	if !ok {
		exception.New("invalid OAuth client type: expected JWT client", 500).Throw()
	}

	resp, err := jwtClient.GetAccessToken(ctx, nil)
	if err != nil {
		exception.New("GetAppToken failed", 500).Throw()
	}

	expiresStr := fmt.Sprintf("%d (%s)", resp.ExpiresIn, timestampToDateTime(resp.ExpiresIn))
	tokenResp := TokenResponse{
		TokenType:    "Bearer",
		AccessToken:  resp.AccessToken,
		RefreshToken: "",
		ExpiresIn:    expiresStr,
	}

	return tokenResp
}
