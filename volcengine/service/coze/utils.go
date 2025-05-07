package coze

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/gob"
	"encoding/json"
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/log"
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

func mustToJson(obj any) string {
	jsonArray, err := json.Marshal(obj)
	if err != nil {
		return "{}"
	}
	return string(jsonArray)
}

func strToObj(ext string, data string, vPtr interface{}) error {
	bytes := make([]byte, len(data))
	copy(bytes, data)
	return byteToObj(ext, bytes, vPtr)
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

func mapToObj(ext string, data map[string]interface{}, vPtr interface{}) error {
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	err := enc.Encode(data)
	if err != nil {
		log.Fatal("Error happened in Gob encoding. Err: %s", err)
	}

	return byteToObj(ext, buffer.Bytes(), vPtr)

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
