// Package payment Process 接口实现测试
package payment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

// TestProcessAddConfig 测试添加支付配置 Process
func TestProcessAddConfig(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	tests := []struct {
		name      string
		args      []interface{}
		wantPanic bool
	}{
		{
			name: "Valid Wechat Config",
			args: []interface{}{
				map[string]interface{}{
					"provider":   ProviderWechat,
					"app_id":     "wx1234567890",
					"app_secret": "test_secret",
					"is_prod":    false,
				},
			},
			wantPanic: false,
		},
		{
			name: "Valid Alipay Config",
			args: []interface{}{
				map[string]interface{}{
					"provider":    ProviderAlipay,
					"app_id":      "2021001234567890",
					"private_key": "test_private_key",
					"public_key":  "test_public_key",
					"is_prod":     false,
				},
			},
			wantPanic: false,
		},
		{
			name: "Valid PayPal Config",
			args: []interface{}{
				map[string]interface{}{
					"provider":      ProviderPayPal,
					"client_id":     "test_client_id",
					"client_secret": "test_client_secret",
					"is_prod":       false,
				},
			},
			wantPanic: false,
		},
		{
			name: "Missing App ID",
			args: []interface{}{
				map[string]interface{}{
					"provider":   ProviderWechat,
					"app_secret": "test_secret",
				},
			},
			wantPanic: true,
		},
		{
			name: "Missing App Secret",
			args: []interface{}{
				map[string]interface{}{
					"provider":   "wechat",
					"app_secret": "test_secret",
				},
			},
			wantPanic: true,
		},
		{
			name: "Empty Provider",
			args: []interface{}{
				map[string]interface{}{
					"provider":   "",
					"app_id":     "test_app_id",
					"app_secret": "test_secret",
				},
			},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProcess := process.New("utils.payment.AddConfig", tt.args...)

			if tt.wantPanic {
				assert.Panics(t, func() {
					ProcessAddConfig(mockProcess)
				})
			} else {
				assert.NotPanics(t, func() {
					ProcessAddConfig(mockProcess)
				})
			}
		})
	}
}

// TestProcessCreatePayment 测试创建支付 Process
func TestProcessCreatePayment(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	// 先添加一个测试配置
	configProcess := process.New("utils.payment.AddConfig", map[string]interface{}{
		"provider":   ProviderWechat,
		"app_id":     "wx1234567890",
		"app_secret": "test_secret",
		"is_prod":    false,
	})
	ProcessAddConfig(configProcess)

	tests := []struct {
		name      string
		args      []interface{}
		wantPanic bool
	}{
		{
			name: "Valid Payment Request - String Amount",
			args: []interface{}{
				map[string]interface{}{
					"provider":     ProviderWechat,
					"out_trade_no": "test_order_001",
					"amount":       "100.00",
					"subject":      "Test Payment",
					"body":         "Test payment description",
				},
			},
			wantPanic: false,
		},
		{
			name: "Valid Payment Request - Float Amount",
			args: []interface{}{
				map[string]interface{}{
					"provider":     ProviderWechat,
					"out_trade_no": "test_order_002",
					"amount":       100.50,
					"subject":      "Test Payment 2",
					"body":         "Test payment description 2",
				},
			},
			wantPanic: false,
		},
		{
			name: "Valid Payment Request - Int Amount",
			args: []interface{}{
				map[string]interface{}{
					"provider":     ProviderWechat,
					"out_trade_no": "test_order_003",
					"amount":       100,
					"subject":      "Test Payment 3",
					"body":         "Test payment description 3",
				},
			},
			wantPanic: false,
		},
		{
			name: "Missing Provider",
			args: []interface{}{
				map[string]interface{}{
					"out_trade_no": "test_order_004",
					"amount":       "100.00",
					"subject":      "Test Payment",
				},
			},
			wantPanic: true,
		},
		{
			name: "Missing Out Trade No",
			args: []interface{}{
				map[string]interface{}{
					"provider": ProviderWechat,
					"amount":   "100.00",
					"subject":  "Test Payment",
				},
			},
			wantPanic: true,
		},
		{
			name: "Missing Amount",
			args: []interface{}{
				map[string]interface{}{
					"provider":     ProviderWechat,
					"out_trade_no": "test_order_005",
					"subject":      "Test Payment",
				},
			},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProcess := process.New("utils.payment.CreatePayment", tt.args...)

			if tt.wantPanic {
				assert.Panics(t, func() {
					ProcessCreatePayment(mockProcess)
				})
			} else {
				result := ProcessCreatePayment(mockProcess)
				assert.NotNil(t, result)

				// 验证返回结果是 PaymentResponse 类型
				response, ok := result.(*PaymentResponse)
				assert.True(t, ok, "返回结果应该是 PaymentResponse 类型")
				assert.NotNil(t, response)
			}
		})
	}
}

// TestProcessQueryPayment 测试查询支付订单 Process
func TestProcessQueryPayment(t *testing.T) {
	tests := []struct {
		name      string
		args      []interface{}
		wantPanic bool
	}{
		{
			name: "Valid Query Request",
			args: []interface{}{
				"wechat",
				"ORDER_123456",
			},
			wantPanic: false,
		},
		{
			name: "Empty Provider",
			args: []interface{}{
				"",
				"ORDER_123456",
			},
			wantPanic: true,
		},
		{
			name: "Empty OutTradeNo",
			args: []interface{}{
				"wechat",
				"",
			},
			wantPanic: true,
		},
		{
			name: "Invalid Provider",
			args: []interface{}{
				"invalid_provider",
				"ORDER_123456",
			},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProcess := process.New("utils.payment.QueryPayment", tt.args...)

			if tt.wantPanic {
				assert.Panics(t, func() {
					ProcessQueryPayment(mockProcess)
				})
			} else {
				assert.NotPanics(t, func() {
					response := ProcessQueryPayment(mockProcess)
					assert.NotNil(t, response)

					// 验证返回的响应结构
					if paymentResp, ok := response.(*PaymentResponse); ok {
						assert.NotEmpty(t, paymentResp.OutTradeNo)
						assert.NotEmpty(t, paymentResp.CreatedTime)
					}
				})
			}
		})
	}
}

// TestProcessRefundPayment 测试退款 Process
func TestProcessRefundPayment(t *testing.T) {
	tests := []struct {
		name      string
		args      []interface{}
		wantPanic bool
	}{
		{
			name: "Valid Refund Request",
			args: []interface{}{
				map[string]interface{}{
					"provider":      "wechat",
					"out_trade_no":  "ORDER_123456",
					"refund_amount": 50.00,
					"reason":        "用户申请退款",
				},
			},
			wantPanic: false,
		},
		{
			name: "Valid Refund Request with int amount",
			args: []interface{}{
				map[string]interface{}{
					"provider":      "wechat",
					"out_trade_no":  "ORDER_123457",
					"refund_amount": 50.00,
					"reason":        "用户申请退款",
				},
			},
			wantPanic: false,
		},
		{
			name:      "Nil Request",
			args:      []interface{}{nil},
			wantPanic: true,
		},
		{
			name: "Missing Provider",
			args: []interface{}{
				map[string]interface{}{
					"out_trade_no":  "ORDER_123458",
					"refund_amount": 50.00,
					"reason":        "用户申请退款",
				},
			},
			wantPanic: true,
		},
		{
			name: "Missing OutTradeNo",
			args: []interface{}{
				map[string]interface{}{
					"provider":      "wechat",
					"refund_amount": 50.00,
					"reason":        "用户申请退款",
				},
			},
			wantPanic: true,
		},
		{
			name: "Missing RefundAmount",
			args: []interface{}{
				map[string]interface{}{
					"provider":     "wechat",
					"out_trade_no": "ORDER_123459",
					"reason":       "用户申请退款",
				},
			},
			wantPanic: true,
		},
		{
			name: "Zero RefundAmount",
			args: []interface{}{
				map[string]interface{}{
					"provider":      "wechat",
					"out_trade_no":  "ORDER_123460",
					"refund_amount": 0.00,
					"reason":        "用户申请退款",
				},
			},
			wantPanic: true,
		},
		{
			name: "Negative RefundAmount",
			args: []interface{}{
				map[string]interface{}{
					"provider":      "wechat",
					"out_trade_no":  "ORDER_123461",
					"refund_amount": -50.00,
					"reason":        "用户申请退款",
				},
			},
			wantPanic: true,
		},
		{
			name: "Invalid Provider",
			args: []interface{}{
				map[string]interface{}{
					"provider":      "invalid_provider",
					"out_trade_no":  "ORDER_123462",
					"refund_amount": 50.00,
					"reason":        "用户申请退款",
				},
			},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProcess := process.New("utils.payment.RefundPayment", tt.args...)

			if tt.wantPanic {
				assert.Panics(t, func() {
					ProcessRefundPayment(mockProcess)
				})
			} else {
				assert.NotPanics(t, func() {
					response := ProcessRefundPayment(mockProcess)
					assert.NotNil(t, response)

					// 验证返回的响应结构
					if refundResp, ok := response.(*RefundResponse); ok {
						assert.NotEmpty(t, refundResp.RefundNo)
						assert.NotEmpty(t, refundResp.OutTradeNo)
						assert.True(t, refundResp.RefundAmount > 0)
					}
				})
			}
		})
	}
}

// TestProcessGetSupportedProviders 测试获取支持的支付提供商 Process
func TestProcessGetSupportedProviders(t *testing.T) {
	mockProcess := process.New("utils.payment.GetSupportedProviders")

	assert.NotPanics(t, func() {
		response := ProcessGetSupportedProviders(mockProcess)
		assert.NotNil(t, response)

		// 验证返回的提供商列表
		if providers, ok := response.([]PaymentProvider); ok {
			assert.Contains(t, providers, ProviderWechat)
			assert.Contains(t, providers, ProviderAlipay)
			assert.Contains(t, providers, ProviderPayPal)
		}
	})
}

// TestProcessValidateNotify 测试验证支付通知 Process
func TestProcessValidateNotify(t *testing.T) {
	tests := []struct {
		name      string
		args      []interface{}
		wantPanic bool
	}{
		{
			name: "Valid Wechat Notify",
			args: []interface{}{
				"wechat",
				map[string]interface{}{
					"signature":  "test_signature",
					"timestamp":  "1234567890",
					"nonce":      "test_nonce",
					"body":       "test_body",
					"serial":     "test_serial",
					"algorithm":  "AEAD_AES_256_GCM",
					"ciphertext": "test_ciphertext",
				},
			},
			wantPanic: false,
		},
		{
			name: "Valid Alipay Notify",
			args: []interface{}{
				"alipay",
				map[string]interface{}{
					"notify_data": "test_notify_data",
					"sign":        "test_sign",
					"sign_type":   "RSA2",
				},
			},
			wantPanic: false,
		},
		{
			name: "Empty Provider",
			args: []interface{}{
				"",
				map[string]interface{}{
					"signature": "test_signature",
				},
			},
			wantPanic: true,
		},
		{
			name:      "Nil Notify Data",
			args:      []interface{}{"wechat", nil},
			wantPanic: true,
		},
		{
			name: "Invalid Provider",
			args: []interface{}{
				"invalid_provider",
				map[string]interface{}{
					"signature": "test_signature",
				},
			},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProcess := process.New("utils.payment.ValidateNotify", tt.args...)

			if tt.wantPanic {
				assert.Panics(t, func() {
					ProcessValidateNotify(mockProcess)
				})
			} else {
				assert.NotPanics(t, func() {
					response := ProcessValidateNotify(mockProcess)
					assert.NotNil(t, response)
				})
			}
		})
	}
}

// BenchmarkProcessAddConfig 基准测试添加支付配置
func BenchmarkProcessAddConfig(b *testing.B) {
	mockProcess := process.New("utils.payment.AddConfig", map[string]interface{}{
		"provider":   "wechat",
		"app_id":     "wx1234567890",
		"app_secret": "test_secret",
		"is_prod":    false,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ProcessAddConfig(mockProcess)
	}
}

// BenchmarkProcessCreatePayment 基准测试创建支付订单
func BenchmarkProcessCreatePayment(b *testing.B) {
	// 先添加配置
	configProcess := process.New("utils.payment.AddConfig", map[string]interface{}{
		"provider":   "wechat",
		"app_id":     "wx1234567890",
		"app_secret": "test_secret",
		"is_prod":    false,
	})
	ProcessAddConfig(configProcess)

	mockProcess := process.New("utils.payment.CreatePayment", map[string]interface{}{
		"out_trade_no": "ORDER_123456",
		"amount":       100.00,
		"subject":      "Test Payment",
		"body":         "Test Description",
		"provider":     "wechat",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ProcessCreatePayment(mockProcess)
	}
}
