package payment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestLoad(t *testing.T) {
	// 执行加载
	err := Load()
	assert.NoError(t, err)

	// 验证管理器已初始化
	assert.NotNil(t, Manager)

	// 验证提供商已注册
	assert.True(t, Manager.HasProvider(ChannelAlipay))
	assert.True(t, Manager.HasProvider(ChannelWechat))

	// 验证Process已注册
	processes := GetProcesses()
	for _, processName := range processes {
		assert.True(t, process.Exists(processName), "Process %s should be registered", processName)
	}
}

func TestUnload(t *testing.T) {
	// 先加载
	err := Load()
	assert.NoError(t, err)
	assert.NotNil(t, Manager)

	// 执行卸载
	err = Unload()
	assert.NoError(t, err)

	// 验证管理器已清理
	assert.Nil(t, Manager)
}

func TestReload(t *testing.T) {
	// 先加载
	err := Load()
	assert.NoError(t, err)
	assert.NotNil(t, Manager)

	// 执行重新加载
	err = Reload()
	assert.NoError(t, err)

	// 验证管理器仍然存在
	assert.NotNil(t, Manager)

	// 验证提供商仍然注册
	assert.True(t, Manager.HasProvider(ChannelAlipay))
	assert.True(t, Manager.HasProvider(ChannelWechat))
}

func TestGetProcesses(t *testing.T) {
	processes := GetProcesses()

	expectedProcesses := []string{
		"payment.SetConfig",
		"payment.GetConfig",
		"payment.CreateOrder",
		"payment.QueryOrder",
		"payment.CreateRefund",
		"payment.QueryRefund",
		"payment.HandleNotify",
		"payment.DownloadBill",
		"payment.Reconcile",
	}

	assert.Equal(t, len(expectedProcesses), len(processes))

	for _, expected := range expectedProcesses {
		assert.Contains(t, processes, expected)
	}
}

func TestGetProviders(t *testing.T) {
	providers := GetProviders()

	expectedProviders := []PaymentChannel{
		ChannelAlipay,
		ChannelWechat,
	}

	assert.Equal(t, len(expectedProviders), len(providers))

	for _, expected := range expectedProviders {
		assert.Contains(t, providers, expected)
	}
}

func TestGetTradeTypes(t *testing.T) {
	tradeTypes := GetTradeTypes()

	expectedTradeTypes := []TradeType{
		TradeTypeJSAPI,
		TradeTypeNative,
		TradeTypeApp,
		TradeTypeH5,
		TradeTypeWAP,
	}

	assert.Equal(t, len(expectedTradeTypes), len(tradeTypes))

	for _, expected := range expectedTradeTypes {
		assert.Contains(t, tradeTypes, expected)
	}
}

func TestGetOrderStatuses(t *testing.T) {
	statuses := GetOrderStatuses()

	expectedStatuses := []OrderStatus{
		OrderStatusPending,
		OrderStatusPaid,
		OrderStatusClosed,
		OrderStatusRefund,
	}

	assert.Equal(t, len(expectedStatuses), len(statuses))

	for _, expected := range expectedStatuses {
		assert.Contains(t, statuses, expected)
	}
}

func TestGetRefundStatuses(t *testing.T) {
	statuses := GetRefundStatuses()

	expectedStatuses := []RefundStatus{
		RefundStatusPending,
		RefundStatusProcessing,
		RefundStatusSuccess,
		RefundStatusFailed,
	}

	assert.Equal(t, len(expectedStatuses), len(statuses))

	for _, expected := range expectedStatuses {
		assert.Contains(t, statuses, expected)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		channel PaymentChannel
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name:    "Valid Alipay config",
			channel: ChannelAlipay,
			config: map[string]interface{}{
				"app_id":            "test_app_id",
				"private_key":       "test_private_key",
				"alipay_public_key": "test_public_key",
			},
			wantErr: false,
		},
		{
			name:    "Invalid Alipay config - missing app_id",
			channel: ChannelAlipay,
			config: map[string]interface{}{
				"private_key":       "test_private_key",
				"alipay_public_key": "test_public_key",
			},
			wantErr: true,
		},
		{
			name:    "Valid Wechat config",
			channel: ChannelWechat,
			config: map[string]interface{}{
				"app_id":      "test_app_id",
				"mch_id":      "test_mch_id",
				"apiv3_key":   "test_apiv3_key",
				"private_key": "test_private_key",
				"serial_no":   "test_serial_no",
			},
			wantErr: false,
		},
		{
			name:    "Invalid Wechat config - missing mch_id",
			channel: ChannelWechat,
			config: map[string]interface{}{
				"app_id":      "test_app_id",
				"apiv3_key":   "test_apiv3_key",
				"private_key": "test_private_key",
				"serial_no":   "test_serial_no",
			},
			wantErr: true,
		},
		{
			name:    "Unsupported channel",
			channel: "unsupported",
			config:  map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.channel, tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAlipayConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "Valid config",
			config: map[string]interface{}{
				"app_id":            "test_app_id",
				"private_key":       "test_private_key",
				"alipay_public_key": "test_public_key",
			},
			wantErr: false,
		},
		{
			name: "Missing app_id",
			config: map[string]interface{}{
				"private_key":       "test_private_key",
				"alipay_public_key": "test_public_key",
			},
			wantErr: true,
		},
		{
			name: "Empty private_key",
			config: map[string]interface{}{
				"app_id":            "test_app_id",
				"private_key":       "",
				"alipay_public_key": "test_public_key",
			},
			wantErr: true,
		},
		{
			name: "Missing alipay_public_key",
			config: map[string]interface{}{
				"app_id":      "test_app_id",
				"private_key": "test_private_key",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAlipayConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateWechatConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "Valid config",
			config: map[string]interface{}{
				"app_id":      "test_app_id",
				"mch_id":      "test_mch_id",
				"apiv3_key":   "test_apiv3_key",
				"private_key": "test_private_key",
				"serial_no":   "test_serial_no",
			},
			wantErr: false,
		},
		{
			name: "Missing app_id",
			config: map[string]interface{}{
				"mch_id":      "test_mch_id",
				"apiv3_key":   "test_apiv3_key",
				"private_key": "test_private_key",
				"serial_no":   "test_serial_no",
			},
			wantErr: true,
		},
		{
			name: "Empty mch_id",
			config: map[string]interface{}{
				"app_id":      "test_app_id",
				"mch_id":      "",
				"apiv3_key":   "test_apiv3_key",
				"private_key": "test_private_key",
				"serial_no":   "test_serial_no",
			},
			wantErr: true,
		},
		{
			name: "Missing apiv3_key",
			config: map[string]interface{}{
				"app_id":      "test_app_id",
				"mch_id":      "test_mch_id",
				"private_key": "test_private_key",
				"serial_no":   "test_serial_no",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWechatConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetModuleInfo(t *testing.T) {
	info := GetModuleInfo()

	assert.Equal(t, "payment", info["name"])
	assert.Equal(t, "1.0.0", info["version"])
	assert.Contains(t, info["description"], "Yao支付模块")

	// 验证providers
	providers, ok := info["providers"].([]PaymentChannel)
	assert.True(t, ok)
	assert.Contains(t, providers, ChannelAlipay)
	assert.Contains(t, providers, ChannelWechat)

	// 验证trade_types
	tradeTypes, ok := info["trade_types"].([]TradeType)
	assert.True(t, ok)
	assert.Contains(t, tradeTypes, TradeTypeJSAPI)
	assert.Contains(t, tradeTypes, TradeTypeNative)

	// 验证processes
	processes, ok := info["processes"].([]string)
	assert.True(t, ok)
	assert.Contains(t, processes, "payment.CreateOrder")
	assert.Contains(t, processes, "payment.QueryOrder")

	// 验证features
	features, ok := info["features"].([]string)
	assert.True(t, ok)
	assert.Contains(t, features, "多商户支持")
	assert.Contains(t, features, "多支付渠道")
}

func TestHealthCheck(t *testing.T) {
	// 重置全局状态
	Manager = nil
	
	// 未加载时的健康检查
	result := HealthCheck()
	assert.Equal(t, "unhealthy", result["status"])

	checks, ok := result["checks"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, checks["manager"], "not initialized")

	// 加载后的健康检查
	err := Load()
	assert.NoError(t, err)

	result = HealthCheck()
	assert.Equal(t, "healthy", result["status"])

	checks, ok = result["checks"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "OK", checks["manager"])

	// 验证providers检查
	providers, ok := checks["providers"].(map[string]string)
	assert.True(t, ok)
	assert.Equal(t, "registered", providers["alipay"])
	assert.Equal(t, "registered", providers["wechat"])

	// 验证processes检查
	processes, ok := checks["processes"].(map[string]string)
	assert.True(t, ok)
	assert.Equal(t, "registered", processes["payment.CreateOrder"])
	assert.Equal(t, "registered", processes["payment.QueryOrder"])
}
