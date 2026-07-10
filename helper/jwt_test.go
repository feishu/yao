package helper

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func TestJwt(t *testing.T) {
	data := map[string]interface{}{"hello": "world", "id": 1}
	option := map[string]interface{}{"subject": "Unit Test", "audience": "Test", "issuer": "UnitTest", "timeout": 1, "sid": ""}
	token := JwtMake(1, data, option)
	tokenString := token.Token
	res := JwtValidate(tokenString)
	assert.NotNil(t, res)
	assert.Equal(t, float64(1), res.Data["id"])
	assert.Equal(t, "world", res.Data["hello"])
	assert.Equal(t, "UnitTest", res.Issuer)
	time.Sleep(2 * time.Second)
	assert.Panics(t, func() { JwtValidate(tokenString) })
}

func TestProcessJwt(t *testing.T) {
	data := map[string]interface{}{"hello": "world", "id": 1}
	option := map[string]interface{}{"subject": "Unit Test", "audience": "Test", "issuer": "UnitTest", "timeout": 1, "sid": ""}
	args := []interface{}{1, data, option}
	p := process.New("xiang.helper.JwtMake", args...)
	token := p.Run().(JwtToken)
	tokenString := token.Token
	res := process.New("xiang.helper.JwtValidate", tokenString).Run().(*JwtClaims)
	assert.Equal(t, float64(1), res.Data["id"])
	assert.Equal(t, "world", res.Data["hello"])
	assert.Equal(t, "UnitTest", res.Issuer)
	time.Sleep(2 * time.Second)
	assert.Panics(t, func() { process.New("xiang.helper.JwtValidate", tokenString).Run() })
}

func TestJwtValidateRejectsNonHS256Algorithm(t *testing.T) {
	secret := []byte("jwt-test-secret")
	claims := &JwtClaims{
		ID:   1,
		SID:  "unit-test",
		Data: map[string]interface{}{"hello": "world"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)
	tokenString, err := token.SignedString(secret)
	assert.NoError(t, err)

	assertJwtPanicMessage(t, "Invalid token", func() {
		JwtValidate(tokenString, secret)
	})
}

func TestJwtValidateRejectsNoneAlgorithmVariants(t *testing.T) {
	for _, alg := range []string{"none", "None", "NoNe", "NONE"} {
		t.Run(alg, func(t *testing.T) {
			tokenString := makeUnsignedToken(t, alg, "signature")

			assertJwtPanicMessage(t, "Invalid token", func() {
				JwtValidate(tokenString, []byte("jwt-test-secret"))
			})
		})
	}
}

func TestJwtValidateRejectsEmptySignatureAsInvalidFormat(t *testing.T) {
	secret := []byte("jwt-test-secret")
	token := JwtMake(1, map[string]interface{}{}, map[string]interface{}{"timeout": 60}, secret)
	parts := strings.Split(token.Token, ".")
	assert.Len(t, parts, MaxTokenParts)

	tokenWithoutSignature := strings.Join(parts[:2], ".") + "."
	assertJwtPanicMessage(t, "Invalid token format", func() {
		JwtValidate(tokenWithoutSignature, secret)
	})
}

func makeUnsignedToken(t *testing.T, alg string, signature string) string {
	t.Helper()

	header := jwt.EncodeSegment([]byte(fmt.Sprintf(`{"alg":%q,"typ":"JWT"}`, alg)))
	claims := jwt.EncodeSegment([]byte(`{"id":1,"sid":"unit-test","data":{}}`))
	return strings.Join([]string{header, claims, signature}, ".")
}

func assertJwtPanicMessage(t *testing.T, want string, fn func()) {
	t.Helper()

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("expected JWT validation to panic")
		}

		switch ex := recovered.(type) {
		case exception.Exception:
			assert.Equal(t, want, ex.Message)
		case *exception.Exception:
			assert.Equal(t, want, ex.Message)
		default:
			t.Fatalf("expected exception panic, got %T", recovered)
		}
	}()

	fn()
}
