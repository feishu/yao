package coze

import (
	"context"
	"fmt"
	"time"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
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
	oauthConfPath := p.ArgsString(0)

	ctx := context.Background()

	var config = OAuthConfig{}
	config = Configs["conf/agents/oauth.json"]
	if oauthConfPath != "" {
		config = Configs[oauthConfPath]
	}

	oauth, err := LoadOAuthAppFromConfig(&config)
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
