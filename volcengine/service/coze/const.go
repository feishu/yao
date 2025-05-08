package coze

const (
	ComBaseURL = "https://api.coze.com"
	CnBaseURL  = "https://api.coze.cn"
)

const (
	httpLogIDKey    = "X-Tt-Logid"
	ctxLogIDKey     = "K_LOGID"
	authorizeHeader = "Authorization"
)

// Auth types
const (
	TokenTypeBearer = "Bearer"
)

// OAuth client types
const (
	ClientTypeJWT = "jwt"
)

// Token cache
const (
	TokenExpiryBufferSeconds = 30 // Token过期前的缓冲时间（秒）
)

// API base paths
const (
	DefaultOAuthTokenPath = "/api/permission/oauth2/token"
)
