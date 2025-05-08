package coze

import (
	"context"
	"time"

	"github.com/yaoapp/kun/log"
)

type Auth interface {
	Token(ctx context.Context) (string, error)
}

var (
	_ Auth = &tokenAuthImpl{}
	_ Auth = &jwtOAuthImpl{}
)

// tokenAuthImpl implements the Auth interface with fixed access token.
type tokenAuthImpl struct {
	accessToken string
}

// NewTokenAuth creates a new token authentication instance.
func NewTokenAuth(accessToken string) Auth {
	return &tokenAuthImpl{
		accessToken: accessToken,
	}
}

func getRefreshBefore(ttl int) int64 {
	if ttl >= 600 {
		return 30
	} else if ttl >= 60 {
		return 10
	} else if ttl >= 30 {
		return 5
	}
	return 0
}

func NewJWTAuth(client *JWTOAuthClient, opt *GetJWTAccessTokenReq) Auth {
	ttl := 900
	if opt == nil {
		return &jwtOAuthImpl{
			TTL:           ttl,
			client:        client,
			refreshBefore: getRefreshBefore(ttl),
		}
	}
	if opt.TTL > 0 {
		ttl = opt.TTL
	}

	return &jwtOAuthImpl{
		TTL:           ttl,
		Scope:         opt.Scope,
		SessionName:   opt.SessionName,
		refreshBefore: getRefreshBefore(ttl),
		client:        client,
		accountID:     opt.AccountID,
	}
}

// Token returns the access token.
func (r *tokenAuthImpl) Token(ctx context.Context) (string, error) {
	return r.accessToken, nil
}

type jwtOAuthImpl struct {
	TTL           int
	SessionName   *string
	Scope         *Scope
	client        *JWTOAuthClient
	accessToken   *string
	expireIn      int64
	refreshBefore int64 // refresh moment before expireIn, unit second
	refreshAt     int64
	accountID     *int64
}

func (r *jwtOAuthImpl) needRefresh() bool {
	return r.accessToken == nil || time.Now().Unix() > r.refreshAt
}

func (r *jwtOAuthImpl) Token(ctx context.Context) (string, error) {
	if !r.needRefresh() {
		log.Debug("jwtOAuthImpl: Token for account %v, session %s is still valid, using cached token", r.accountID, ptrValue(r.SessionName))
		return ptrValue(r.accessToken), nil
	}
	log.Info("jwtOAuthImpl: Refreshing token for account %v, session %s", r.accountID, ptrValue(r.SessionName))
	resp, err := r.client.GetJWTAccessToken(ctx, &GetJWTAccessTokenReq{
		TTL:         r.TTL,
		SessionName: r.SessionName,
		Scope:       r.Scope,
		AccountID:   r.accountID,
	})
	if err != nil {
		log.Error("jwtOAuthImpl: Failed to refresh token for account %v, session %s: %v", r.accountID, ptrValue(r.SessionName), err)
		return "", err
	}
	r.accessToken = ptr(resp.AccessToken)
	r.expireIn = resp.ExpiresIn
	r.refreshAt = resp.ExpiresIn - r.refreshBefore
	log.Info("jwtOAuthImpl: Token refreshed successfully for account %v, session %s. New expiry: %d, refresh_at: %d", r.accountID, ptrValue(r.SessionName), r.expireIn, r.refreshAt)
	return resp.AccessToken, nil
}
