package im

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"
	"math"
	"time"
)

// Token相关常量
const (
	standardBase = 256
	targetBase   = 62
)

// 字符映射表
var alphabet = []byte{
	'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
	'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
	'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
}

// Token 表示火山引擎IM的token信息结构
type Token struct {
	ID        int32
	Type      int16
	AppID     int32
	UserID    int64
	Timestamp int64
}

// GenerateToken 生成火山引擎IM的应用令牌
// appID: 应用ID
// userID: 用户ID
// tokenExpireTime: 过期时间戳
// appKey: 应用密钥
// 返回: 生成的令牌字符串和可能的错误
func GenerateToken(appID int32, userID, tokenExpireTime int64, appKey string) (string, error) {
	token := &Token{
		ID:        appID,
		Type:      1,
		AppID:     appID,
		UserID:    userID,
		Timestamp: tokenExpireTime,
	}

	encrypted := make([]byte, 24)
	binary.BigEndian.PutUint16(encrypted[:2], uint16(1))
	binary.BigEndian.PutUint16(encrypted[2:4], uint16(token.Type))
	binary.BigEndian.PutUint32(encrypted[4:8], uint32(token.AppID))
	binary.BigEndian.PutUint64(encrypted[8:16], uint64(token.UserID))
	binary.BigEndian.PutUint64(encrypted[16:24], uint64(token.Timestamp))

	decodeKey, err := base64.StdEncoding.DecodeString(appKey)
	if err != nil {
		return "", err
	}
	c, err := aes.NewCipher(decodeKey)
	if err != nil {
		return "", err
	}
	iv := decodeKey[:16]
	encrypted = pkcs5Padding(encrypted, c.BlockSize())
	cipher.NewCBCEncrypter(c, iv).CryptBlocks(encrypted, encrypted)
	crc := crc32.ChecksumIEEE(encrypted)
	output := make([]byte, 8+len(encrypted))
	binary.BigEndian.PutUint32(output[:4], crc)
	binary.BigEndian.PutUint32(output[4:8], uint32(token.ID))
	copy(output[8:], encrypted)
	return string(encode(output)), nil
}

// GetDefaultExpireTime 获取默认的过期时间（当前时间30分钟后）
func generateExpireTime(minutes int64) int64 {
	return time.Now().UnixNano()/int64(time.Millisecond) + 1000*60*minutes // 30分钟
}

// pkcs5Padding PKCS#5填充算法实现
func pkcs5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

// encode 将字节数组编码为可显示字符
func encode(input []byte) []byte {
	output := convert(input, standardBase, targetBase)
	translate(output, alphabet)
	return output
}

// convert 进制转换实现
func convert(input []byte, sourceBase, targetBase int) []byte {
	estimatedLength := estimateOutputLength(len(input), sourceBase, targetBase)
	output := make([]byte, 0, estimatedLength)
	zeroCount := 0
	for _, b := range input {
		if b == 0 {
			zeroCount++
		} else {
			break
		}
	}

	for len(input) > 0 {
		quotient := make([]byte, 0, len(input))
		remainder := 0
		for _, b := range input {
			accumulator := int(b&0xFF) + remainder*sourceBase
			digit := (accumulator - (accumulator % targetBase)) / targetBase
			remainder = accumulator % targetBase
			if len(quotient) > 0 || digit > 0 {
				quotient = append(quotient, byte(digit))
			}
		}
		output = append(output, byte(remainder))
		input = quotient
	}

	if zeroCount != 0 {
		for i := 0; i < zeroCount; i++ {
			output = append(output, 0)
		}
	}

	for i, j := 0, len(output)-1; i < j; i, j = i+1, j-1 {
		output[i], output[j] = output[j], output[i]
	}
	return output
}

// estimateOutputLength 估算输出长度
func estimateOutputLength(inputLength, sourceBase, targetBase int) int {
	return int(math.Ceil((math.Log(float64(sourceBase)) / math.Log(float64(targetBase))) * float64(inputLength)))
}

// translate 字符转换
func translate(indices, dict []byte) {
	for i, b := range indices {
		indices[i] = dict[b]
	}
}
