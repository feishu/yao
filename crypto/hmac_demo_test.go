package crypto

import (
	"fmt"
	"testing"
)

func TestHmacSHA256Demo(t *testing.T) {
	partnerId := "50002003"
	partnerSecret := "fb5333587e6bf3409fb2481413c023cd"
	timestamp := "1767967411220"

	value := partnerId + timestamp
	key := partnerSecret

	result, err := Hmac(HashTypes["SHA256"], value, key)
	if err != nil {
		t.Fatalf("Hmac error: %v", err)
	}

	fmt.Println("=== HMAC-SHA256 签名测试 ===")
	fmt.Printf("PartnerId:     %s\n", partnerId)
	fmt.Printf("PartnerSecret: %s\n", partnerSecret)
	fmt.Printf("Timestamp:     %s\n", timestamp)
	fmt.Printf("待签名内容:    %s\n", value)
	fmt.Printf("生成的签名:    %s\n", result)
}
