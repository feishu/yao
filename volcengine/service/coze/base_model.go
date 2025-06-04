package coze

import (
	"context"
	"net/http"
	"time"
)

// OAuthClient OAuth客户端接口
type OAuthClient interface {
	GetAccessToken(ctx context.Context) (*TokenResponse, error)
	ClientID() string
}

// OAuthConfig represents the configuration for OAuth clients
type OAuthConfig struct {
	ClientID     string `json:"client_id"`
	ClientType   string `json:"client_type"`
	ClientSecret string `json:"client_secret,omitempty"`
	PrivateKey   string `json:"private_key,omitempty"`
	PublicKeyID  string `json:"public_key_id,omitempty"`
	CozeAPIBase  string `json:"coze_api_base,omitempty"`
	CozeWWWBase  string `json:"coze_www_base,omitempty"`
}

// TokenResponse 封装OAuth token响应
type TokenResponse struct {
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in"`
	GeneratedAt  int64  `json:"generated_at,omitempty"` // 记录token生成时间
}

// IsExpired 检查token是否已过期（考虑缓冲期）
func (t *TokenResponse) IsExpired() bool {
	if t == nil || t.AccessToken == "" {
		return true
	}

	// 计算预计过期时间（生成时间 + 有效期秒数）
	expiresAt := t.GeneratedAt + t.ExpiresIn

	// 当前时间 + 缓冲期
	now := time.Now().Unix() + TokenExpiryBufferSeconds

	// 如果当前时间+缓冲期已接近或超过过期时间，则认为已过期
	return now >= expiresAt
}

// Remaining 返回token剩余有效时间（秒）
func (t *TokenResponse) Remaining() int64 {
	if t == nil || t.AccessToken == "" {
		return 0
	}

	expiresAt := t.GeneratedAt + t.ExpiresIn
	now := time.Now().Unix()

	remaining := expiresAt - now
	if remaining < 0 {
		return 0
	}

	return remaining
}

// AutoRefreshToken 是支持自动刷新的token接口
type AutoRefreshToken interface {
	AccessToken() string
	Refresh(ctx context.Context) error
}

type Responser interface {
	Response() HTTPResponse
}

type HTTPResponse interface {
	LogID() string
}

type httpResponse struct {
	Status        int
	Header        http.Header
	ContentLength int64

	logid string
}

func (r *httpResponse) LogID() string {
	if r.logid == "" {
		r.logid = r.Header.Get(httpLogIDKey)
	}
	return r.logid
}

type baseResponse struct {
	Code         int           `json:"code"`
	Msg          string        `json:"msg"`
	HTTPResponse *httpResponse `json:"http_response"`
}

func (r *baseResponse) SetHTTPResponse(httpResponse *httpResponse) {
	r.HTTPResponse = httpResponse
}

func (r *baseResponse) SetCode(code int) {
	r.Code = code
}

func (r *baseResponse) SetMsg(msg string) {
	r.Msg = msg
}

func (r *baseResponse) GetCode() int {
	return r.Code
}

func (r *baseResponse) GetMsg() string {
	return r.Msg
}

func (r *baseResponse) LogID() string {
	return r.HTTPResponse.LogID()
}

type baseRespInterface interface {
	SetHTTPResponse(httpResponse *httpResponse)
	SetCode(code int)
	SetMsg(msg string)
	GetMsg() string
	GetCode() int
}

type baseModel struct {
	httpResponse *httpResponse
}

func (r *baseModel) setHTTPResponse(httpResponse *httpResponse) {
	r.httpResponse = httpResponse
}

func (r *baseModel) Response() HTTPResponse {
	return r.httpResponse
}

func (r *baseModel) LogID() string {
	return r.httpResponse.LogID()
}

func newHTTPResponse(resp *http.Response) *httpResponse {
	return &httpResponse{
		Status:        resp.StatusCode,
		Header:        resp.Header,
		ContentLength: resp.ContentLength,
	}
}
