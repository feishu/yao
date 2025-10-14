package payment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/yaoapp/gou/process"
)

func TestProcessSetConfig(t *testing.T) {
	// 注册模拟提供商
	mockProvider := &MockPaymentProvider{}
	Manager.RegisterProvider(string(ChannelAlipay), mockProvider)

	// 创建process
	p := process.New("payment.SetConfig", "test_merchant", string(ChannelAlipay), map[string]interface{}{
		"app_id":            "test_app_id",
		"private_key":       "test_private_key",
		"alipay_public_key": "test_public_key",
	})

	// 执行process
	result := ProcessSetConfig(p)

	// 验证结果
	assert.NotNil(t, result)
	response := result.(map[string]interface{})
	assert.True(t, response["success"].(bool))
}

func TestProcessGetConfig(t *testing.T) {
	// 先设置配置
	config := map[string]interface{}{
		"app_id":            "test_app_id",
		"private_key":       "test_private_key",
		"alipay_public_key": "test_public_key",
	}
	Manager.SetMerchantConfig("test_merchant", ChannelAlipay, config)

	// 创建process
	p := process.New("payment.GetConfig", "test_merchant", string(ChannelAlipay))

	// 执行process
	result := ProcessGetConfig(p)

	// 验证结果
	assert.NotNil(t, result)
	response := result.(map[string]interface{})
	assert.True(t, response["success"].(bool))
	assert.Equal(t, config, response["config"])
}

func TestProcessCreateOrder(t *testing.T) {
	// 注册模拟提供商
	mockProvider := &MockPaymentProvider{}
	Manager.RegisterProvider(string(ChannelAlipay), mockProvider)

	// 设置配置
	config := map[string]interface{}{
		"app_id":            "test_app_id",
		"private_key":       "test_private_key",
		"alipay_public_key": "test_public_key",
	}
	Manager.SetMerchantConfig("test_merchant", ChannelAlipay, config)

	// 设置模拟期望
	expectedResponse := &CreateOrderResponse{
		Success:    true,
		OrderID:    "test_order_id",
		OutTradeNo: "test_order_001",
		PayURL:     "https://qr.alipay.com/test",
		Message:    "Order created successfully",
	}
	mockProvider.On("CreateOrder", mock.AnythingOfType("*payment.CreateOrderParams")).Return(expectedResponse, nil)

	// 创建process
	p := process.New("payment.CreateOrder", map[string]interface{}{
		"merchant_no": "test_merchant",
		"channel":     string(ChannelAlipay),
		"trade_type":  string(TradeTypeNative),
		"amount":      100,
		"subject":     "Test Order",
		"out_trade_no": "test_order_001",
		"notify_url":  "https://example.com/notify",
	})

	// 执行process
	result := ProcessCreateOrder(p)

	// 验证结果
	assert.NotNil(t, result)
	mockProvider.AssertExpectations(t)
}

func TestProcessQueryOrder(t *testing.T) {
	// 注册模拟提供商
	mockProvider := &MockPaymentProvider{}
	Manager.RegisterProvider(string(ChannelAlipay), mockProvider)

	// 设置配置
	config := map[string]interface{}{
		"app_id":            "test_app_id",
		"private_key":       "test_private_key",
		"alipay_public_key": "test_public_key",
	}
	Manager.SetMerchantConfig("test_merchant", ChannelAlipay, config)

	// 设置模拟期望
	expectedResponse := &QueryOrderResponse{
		Success:    true,
		OrderID:    "test_order_id",
		OutTradeNo: "test_order_001",
		Status:     "paid",
		Amount:     100,
		Message:    "Order query successful",
	}
	mockProvider.On("QueryOrder", mock.AnythingOfType("*payment.QueryOrderParams")).Return(expectedResponse, nil)

	// 创建process
	p := process.New("payment.QueryOrder", map[string]interface{}{
		"merchant_no": "test_merchant",
		"channel":     string(ChannelAlipay),
		"out_trade_no": "test_order_001",
	})

	// 执行process
	result := ProcessQueryOrder(p)

	// 验证结果
	assert.NotNil(t, result)
	mockProvider.AssertExpectations(t)
}

func TestProcessCreateRefund(t *testing.T) {
	// 注册模拟提供商
	mockProvider := &MockPaymentProvider{}
	Manager.RegisterProvider(string(ChannelAlipay), mockProvider)

	// 设置配置
	config := map[string]interface{}{
		"app_id":            "test_app_id",
		"private_key":       "test_private_key",
		"alipay_public_key": "test_public_key",
	}
	Manager.SetMerchantConfig("test_merchant", ChannelAlipay, config)

	// 设置模拟期望
	expectedResponse := &CreateRefundResponse{
		Success:      true,
		RefundID:     "test_refund_id",
		OutRefundNo:  "test_refund_001",
		RefundAmount: 50,
		Status:       "success",
		Message:      "Refund created successfully",
	}
	mockProvider.On("CreateRefund", mock.AnythingOfType("*payment.CreateRefundParams")).Return(expectedResponse, nil)

	// 创建process
	p := process.New("payment.CreateRefund", map[string]interface{}{
		"merchant_no":    "test_merchant",
		"channel":        string(ChannelAlipay),
		"out_trade_no":   "test_order_001",
		"out_refund_no":  "test_refund_001",
		"refund_amount":  50,
		"total_amount":   100,
		"reason":         "User request",
	})

	// 执行process
	result := ProcessCreateRefund(p)

	// 验证结果
	assert.NotNil(t, result)
	mockProvider.AssertExpectations(t)
}

func TestProcessHandleNotify(t *testing.T) {
	// 注册模拟提供商
	mockProvider := &MockPaymentProvider{}
	Manager.RegisterProvider(string(ChannelAlipay), mockProvider)

	// 设置配置
	config := map[string]interface{}{
		"app_id":            "test_app_id",
		"private_key":       "test_private_key",
		"alipay_public_key": "test_public_key",
	}
	Manager.SetMerchantConfig("test_merchant", ChannelAlipay, config)

	// 设置模拟期望
	expectedResponse := &HandleNotifyResponse{
		Success:    true,
		OutTradeNo: "test_order_001",
		Status:     "paid",
		Amount:     100,
		Message:    "Notify handled successfully",
	}
	mockProvider.On("HandleNotify", mock.AnythingOfType("*payment.HandleNotifyParams")).Return(expectedResponse, nil)

	// 创建process
	p := process.New("payment.HandleNotify", "test_merchant", string(ChannelAlipay), map[string]interface{}{
		"out_trade_no": "test_order_001",
		"trade_status": "TRADE_SUCCESS",
	})

	// 执行process
	result := ProcessHandleNotify(p)

	// 验证结果
	assert.NotNil(t, result)
	mockProvider.AssertExpectations(t)
}

func TestProcessReconcile(t *testing.T) {
	// 初始化全局管理器
	if Manager == nil {
		Manager = NewPaymentManager()
	}
	
	// 注册模拟提供商
	mockProvider := &MockPaymentProvider{}
	
	// 设置 mock 期望
	mockProvider.On("DownloadBill", mock.AnythingOfType("*payment.DownloadBillParams")).Return(&DownloadBillResponse{
		Success:  true,
		BillData: "test_bill_data",
		Message:  "success",
	}, nil)
	
	Manager.RegisterProvider(string(ChannelAlipay), mockProvider)

	// 设置配置
	config := map[string]interface{}{
		"app_id":            "test_app_id",
		"private_key":       "test_private_key",
		"alipay_public_key": "test_public_key",
	}
	Manager.SetMerchantConfig("test_merchant", ChannelAlipay, config)

	// 创建process
	p := process.New("payment.Reconcile", map[string]interface{}{
		"merchant_id": "test_merchant",
		"merchant_no": "test_merchant",
		"channel":     string(ChannelAlipay),
		"bill_date":   "2023-01-01",
	})

	// 执行process
	result := ProcessReconcile(p)

	// 验证结果
	assert.NotNil(t, result)
}

func TestValidatePaymentChannel(t *testing.T) {
	// 测试有效渠道
	assert.True(t, IsValidChannel(ChannelAlipay))
	assert.True(t, IsValidChannel(ChannelWechat))

	// 测试无效渠道
	assert.False(t, IsValidChannel(PaymentChannel("invalid_channel")))
	assert.False(t, IsValidChannel(PaymentChannel("")))
}
