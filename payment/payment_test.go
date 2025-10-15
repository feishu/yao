package payment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPaymentProvider 模拟支付提供商
type MockPaymentProvider struct {
	mock.Mock
}

func (m *MockPaymentProvider) CreateOrder(params *CreateOrderParams) (*CreateOrderResponse, error) {
	args := m.Called(params)
	return args.Get(0).(*CreateOrderResponse), args.Error(1)
}

func (m *MockPaymentProvider) QueryOrder(params *QueryOrderParams) (*QueryOrderResponse, error) {
	args := m.Called(params)
	return args.Get(0).(*QueryOrderResponse), args.Error(1)
}

func (m *MockPaymentProvider) CreateRefund(params *CreateRefundParams) (*CreateRefundResponse, error) {
	args := m.Called(params)
	return args.Get(0).(*CreateRefundResponse), args.Error(1)
}

func (m *MockPaymentProvider) QueryRefund(params *QueryRefundParams) (*QueryRefundResponse, error) {
	args := m.Called(params)
	return args.Get(0).(*QueryRefundResponse), args.Error(1)
}

func (m *MockPaymentProvider) HandleNotify(params *HandleNotifyParams) (*HandleNotifyResponse, error) {
	args := m.Called(params)
	return args.Get(0).(*HandleNotifyResponse), args.Error(1)
}

func (m *MockPaymentProvider) DownloadBill(params *DownloadBillParams) (*DownloadBillResponse, error) {
	args := m.Called(params)
	return args.Get(0).(*DownloadBillResponse), args.Error(1)
}

func TestNewPaymentManager(t *testing.T) {
	manager := NewPaymentManager()

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.providers)
	assert.NotNil(t, manager.certConfigs)
}

func TestPaymentManager_RegisterProvider(t *testing.T) {
	manager := NewPaymentManager()

	// 注册提供商
	mockProvider := &MockPaymentProvider{}
	err := manager.RegisterProvider(string(ChannelAlipay), mockProvider)
	assert.NoError(t, err)

	// 验证提供商已注册
	assert.True(t, manager.HasProvider(ChannelAlipay))
	assert.False(t, manager.HasProvider(ChannelWechat))
}

func TestPaymentManager_ValidateCreateOrderParams(t *testing.T) {
	manager := NewPaymentManager()

	// 测试有效参数
	validParams := &CreateOrderParams{
		MerchantNo: "test_merchant",
		Channel:    string(ChannelAlipay),
		TradeType:  string(TradeTypeNative),
		Amount:     100,
		Subject:    "Test Order",
		OutTradeNo: "test_order_123",
		NotifyURL:  "https://example.com/notify",
	}

	err := manager.ValidateCreateOrderParams(validParams)
	assert.NoError(t, err)

	// 测试无效参数 - 缺少商户号
	invalidParams := &CreateOrderParams{
		Channel:    string(ChannelAlipay),
		TradeType:  string(TradeTypeNative),
		Amount:     100,
		Subject:    "Test Order",
		OutTradeNo: "test_order_123",
		NotifyURL:  "https://example.com/notify",
	}

	err = manager.ValidateCreateOrderParams(invalidParams)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "merchant_no")
}

func TestPaymentManager_ValidateQueryOrderParams(t *testing.T) {
	manager := NewPaymentManager()

	// 测试有效参数
	validParams := &QueryOrderParams{
		MerchantNo: "test_merchant",
		OutTradeNo: "test_order_123",
		Channel:    string(ChannelAlipay),
	}

	err := manager.ValidateQueryOrderParams(validParams)
	assert.NoError(t, err)

	// 测试无效参数 - 缺少商户号
	invalidParams := &QueryOrderParams{
		OutTradeNo: "test_order_123",
		Channel:    string(ChannelAlipay),
	}

	err = manager.ValidateQueryOrderParams(invalidParams)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "merchant_no")
}

func TestPaymentManager_GetProvider(t *testing.T) {
	manager := NewPaymentManager()

	// 测试获取不存在的提供商
	_, err := manager.GetProvider(string(ChannelAlipay))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// 注册提供商后再测试
	mockProvider := &MockPaymentProvider{}
	manager.RegisterProvider(string(ChannelAlipay), mockProvider)

	provider, err := manager.GetProvider(string(ChannelAlipay))
	assert.NoError(t, err)
	assert.NotNil(t, provider)
}
