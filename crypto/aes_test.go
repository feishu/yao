package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestAES256GCM(t *testing.T) {
	key := `oxxxyXVBqwqUjmbgKlwuHV2mgxxxcfOa`
	nonce := `LJEcFT6QWjkG`
	text := `{"name":"yao"}`
	additionalData := `transaction`

	crypted, err := AES256Encrypt(key, "GCM", nonce, text, additionalData)
	if err != nil {
		t.Errorf("AES256Encrypt error: %s", err)
	}

	decrypted, err := AES256Decrypt(key, "GCM", nonce, crypted, additionalData)
	if err != nil {
		t.Errorf("AES256Decrypt error: %s", err)
	}

	assert.Equal(t, text, decrypted)
}

func TestAES256GCMBase64(t *testing.T) {
	key := `oxxxyXVBqwqUjmbgKlwuHV2mgxxxcfOa`
	nonce := `LJEcFT6QWjkG`
	text := `{"name":"yao"}`
	additionalData := `transaction`

	crypted, err := AES256Encrypt(key, "GCM", nonce, text, additionalData, "base64")
	if err != nil {
		t.Errorf("AES256Encrypt error: %s", err)
	}

	decrypted, err := AES256Decrypt(key, "GCM", nonce, crypted, additionalData, "base64")
	if err != nil {
		t.Errorf("AES256Decrypt error: %s", err)
	}
	assert.Equal(t, text, decrypted)

}

func TestAES256ProcessGCM(t *testing.T) {
	key := `oxxxyXVBqwqUjmbgKlwuHV2mgxxxcfOa`
	nonce := `LJEcFT6QWjkG`
	text := `{"name":"yao"}`
	additionalData := `transaction`

	args := []interface{}{"GCM", key, nonce, text, additionalData}
	crypted, err := process.New("crypto.Aes256Encrypt", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	args = []interface{}{"GCM", key, nonce, crypted, additionalData}
	decrypted, err := process.New("crypto.Aes256Decrypt", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, text, decrypted)
}

func TestAES256ProcessGCMBase64(t *testing.T) {
	key := `oxxxyXVBqwqUjmbgKlwuHV2mgxxxcfOa`
	nonce := `LJEcFT6QWjkG`
	text := `{"name":"yao"}`
	additionalData := `transaction`

	args := []interface{}{"GCM", key, nonce, text, additionalData, "base64"}
	crypted, err := process.New("crypto.Aes256Encrypt", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	args = []interface{}{"GCM", key, nonce, crypted, additionalData, "base64"}
	decrypted, err := process.New("crypto.Aes256Decrypt", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, text, decrypted)
}

func TestWechatDecrypt(t *testing.T) {
	appID := "wx4f4bc4dec97d474b"
	sessionKey := "tiihtNczf5v6AKRyjwEUhQ=="
	iv := "r7BXXKkLb8qrSNn05n0qiA=="
	
	// Construct expected data
	expectedData := map[string]interface{}{
		"phoneNumber": "13580006666",
		"purePhoneNumber": "13580006666",
		"countryCode": "86",
		"watermark": map[string]interface{}{
			"timestamp": 1477314187.0,
			"appid":     appID,
		},
	}
	
	payload, _ := json.Marshal(expectedData)
	
	// Encrypt data to generate encryptedData
	keyBytes, _ := base64.StdEncoding.DecodeString(sessionKey)
	ivBytes, _ := base64.StdEncoding.DecodeString(iv)
	
	block, _ := aes.NewCipher(keyBytes)
	// PKCS7 Padding
	blockSize := block.BlockSize()
	padding := blockSize - len(payload)%blockSize
	padtext := make([]byte, padding)
	for i := range padtext {
		padtext[i] = byte(padding)
	}
	payload = append(payload, padtext...)
	
	ciphertext := make([]byte, len(payload))
	mode := cipher.NewCBCEncrypter(block, ivBytes)
	mode.CryptBlocks(ciphertext, payload)
	
	encryptedData := base64.StdEncoding.EncodeToString(ciphertext)
	
	// Test Decrypt
	decryptedStr, err := AES256Decrypt(sessionKey, "CBC", iv, encryptedData, "", "base64")
	if err != nil {
		t.Fatalf("AES256Decrypt error: %s", err)
	}

	var decrypted map[string]interface{}
	err = json.Unmarshal([]byte(decryptedStr), &decrypted)
	if err != nil {
		t.Fatalf("json unmarshal error: %s", err)
	}

	assert.Equal(t, expectedData["phoneNumber"], decrypted["phoneNumber"])
	assert.Equal(t, expectedData["purePhoneNumber"], decrypted["purePhoneNumber"])
	assert.Equal(t, expectedData["countryCode"], decrypted["countryCode"])
}
