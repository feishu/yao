package coze

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	jsoniter "github.com/json-iterator/go"
	"gopkg.in/yaml.v3"
)

func ptrValue[T any](s *T) T {
	if s != nil {
		return *s
	}
	var empty T
	return empty
}

func ptr[T any](s T) *T {
	return &s
}

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return bytesToHex(bytes), nil
}

func bytesToHex(bytes []byte) string {
	hex := make([]byte, len(bytes)*2)
	for i, b := range bytes {
		hex[i*2] = hexChar(b >> 4)
		hex[i*2+1] = hexChar(b & 0xF)
	}
	return string(hex)
}

func hexChar(b byte) byte {
	if b < 10 {
		return '0' + b
	}
	return 'a' + (b - 10)
}

func byteToObj(ext string, data []byte, vPtr interface{}) error {
	switch ext {
	case ".yao", ".jsonc":
		content := trim(data, nil)
		err := jsoniter.Unmarshal(content, vPtr)
		if err != nil {
			return fmt.Errorf("[Parse] %s Error %s", ext, err.Error())
		}
		return nil

	case ".json":
		err := jsoniter.Unmarshal(data, vPtr)
		if err != nil {
			return fmt.Errorf("[Parse] %s Error %s", ext, err.Error())
		}
		return nil

	case ".yml", ".yaml":
		err := yaml.Unmarshal(data, vPtr)
		if err != nil {
			return fmt.Errorf("[Parse] %s Error %s", ext, err.Error())
		}
		return nil
	}

	return fmt.Errorf("[Parse] %s Error %s does not support", ext, ext)
}

type contextKey string

const (
	authContextKey   = contextKey("auth_context")
	authContextValue = "1"
)

func genAuthContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, authContextKey, authContextValue)
}

func isAuthContext(ctx context.Context) bool {
	v := ctx.Value(authContextKey)
	if v == nil {
		return false
	}
	strV, ok := v.(string)
	if !ok {
		return false
	}
	return strV == authContextValue
}

func trim(src, dst []byte) []byte {
	dst = dst[:0]
	for i := 0; i < len(src); i++ {
		if src[i] == '/' {
			if i < len(src)-1 {
				if src[i+1] == '/' {
					dst = append(dst, ' ', ' ')
					i += 2
					for ; i < len(src); i++ {
						if src[i] == '\n' {
							dst = append(dst, '\n')
							break
						} else if src[i] == '\t' || src[i] == '\r' {
							dst = append(dst, src[i])
						} else {
							dst = append(dst, ' ')
						}
					}
					continue
				}
				if src[i+1] == '*' {
					dst = append(dst, ' ', ' ')
					i += 2
					for ; i < len(src)-1; i++ {
						if src[i] == '*' && src[i+1] == '/' {
							dst = append(dst, ' ', ' ')
							i++
							break
						} else if src[i] == '\n' || src[i] == '\t' ||
							src[i] == '\r' {
							dst = append(dst, src[i])
						} else {
							dst = append(dst, ' ')
						}
					}
					continue
				}
			}
		}
		dst = append(dst, src[i])
		if src[i] == '"' {
			for i = i + 1; i < len(src); i++ {
				dst = append(dst, src[i])
				if src[i] == '"' {
					j := i - 1
					for ; ; j-- {
						if src[j] != '\\' {
							break
						}
					}
					if (j-i)%2 != 0 {
						break
					}
				}
			}
		} else if src[i] == '}' || src[i] == ']' {
			for j := len(dst) - 2; j >= 0; j-- {
				if dst[j] <= ' ' {
					continue
				}
				if dst[j] == ',' {
					dst[j] = ' '
				}
				break
			}
		}
	}
	return dst
}

// ConvertOAuthTokenToTokenResponse 将OAuthToken转换为TokenResponse
func ConvertOAuthTokenToTokenResponse(token *OAuthToken) *TokenResponse {
	if token == nil {
		return nil
	}

	return &TokenResponse{
		TokenType:    TokenTypeBearer,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresIn:    token.ExpiresIn,
		GeneratedAt:  time.Now().Unix(), // 记录生成时间
	}
}

// GenerateCacheKey 根据ClientID和可选参数生成缓存键
func GenerateCacheKey(clientID string, params ...string) string {
	key := clientID
	for _, param := range params {
		if param != "" {
			key += "|" + param
		}
	}
	return key
}

// mustToJson 将任意值转为JSON字符串，若失败则返回空字符串
func mustToJson(v interface{}) string {
	if v == nil {
		return ""
	}

	data, err := jsoniter.Marshal(v)
	if err != nil {
		return ""
	}

	return string(data)
}
