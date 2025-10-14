package payment

import (
	"fmt"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/payment/providers"
)

// Load 加载支付模块
func Load() error {
	log.Info("Loading payment module...")

	// 初始化支付管理器
	if err := initManager(); err != nil {
		return fmt.Errorf("init payment manager failed: %v", err)
	}

	// 注册支付提供商
	if err := registerProviders(); err != nil {
		return fmt.Errorf("register payment providers failed: %v", err)
	}

	// 注册Process接口
	if err := registerProcesses(); err != nil {
		return fmt.Errorf("register payment processes failed: %v", err)
	}

	log.Info("Payment module loaded successfully")
	return nil
}

// initManager 初始化支付管理器
func initManager() error {
	// 创建全局支付管理器实例
	Manager = NewPaymentManager()

	log.Debug("Payment manager initialized")
	return nil
}

// registerProviders 注册支付提供商
func registerProviders() error {
	// 注册支付宝提供商
	alipayProvider, err := providers.NewAlipayProvider(nil)
	if err != nil {
		return fmt.Errorf("failed to create alipay provider: %v", err)
	}
	adapter := &ProviderAdapter{provider: alipayProvider}
	if err := Manager.RegisterProvider(string(ChannelAlipay), adapter); err != nil {
		return fmt.Errorf("failed to register alipay provider: %v", err)
	}

	// 注册微信支付提供商
	wechatProvider, err := providers.NewWechatProvider(nil)
	if err != nil {
		return fmt.Errorf("failed to create wechat provider: %v", err)
	}
	wechatAdapter := &ProviderAdapter{provider: wechatProvider}
	if err := Manager.RegisterProvider(string(ChannelWechat), wechatAdapter); err != nil {
		return fmt.Errorf("failed to register wechat provider: %v", err)
	}

	log.Debug("Payment providers registered successfully")
	return nil
}

// registerProcesses 注册Process接口
func registerProcesses() error {
	// 配置管理相关Process
	process.Register("payment.SetConfig", ProcessSetConfig)
	process.Register("payment.GetConfig", ProcessGetConfig)

	// 订单相关Process
	process.Register("payment.CreateOrder", ProcessCreateOrder)
	process.Register("payment.QueryOrder", ProcessQueryOrder)

	// 退款相关Process
	process.Register("payment.CreateRefund", ProcessCreateRefund)
	process.Register("payment.QueryRefund", ProcessQueryRefund)

	// 通知相关Process
	process.Register("payment.HandleNotify", ProcessHandleNotify)

	// 对账相关Process
	process.Register("payment.DownloadBill", ProcessDownloadBill)
	process.Register("payment.Reconcile", ProcessReconcile)

	log.Debug("Payment processes registered")
	return nil
}

// Unload 卸载支付模块
func Unload() error {
	log.Info("Unloading payment module...")

	// 清理支付管理器
	if Manager != nil {
		Manager = nil
	}

	log.Info("Payment module unloaded successfully")
	return nil
}

// Reload 重新加载支付模块
func Reload() error {
	log.Info("Reloading payment module...")

	// 先卸载
	if err := Unload(); err != nil {
		return fmt.Errorf("unload payment module failed: %v", err)
	}

	// 再加载
	if err := Load(); err != nil {
		return fmt.Errorf("load payment module failed: %v", err)
	}

	log.Info("Payment module reloaded successfully")
	return nil
}

// GetProcesses 获取所有注册的Process接口
func GetProcesses() []string {
	return []string{
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
}

// GetProviders 获取所有支持的支付提供商
func GetProviders() []PaymentChannel {
	return []PaymentChannel{
		ChannelAlipay,
		ChannelWechat,
	}
}

// GetTradeTypes 获取所有支持的交易类型
func GetTradeTypes() []TradeType {
	return []TradeType{
		TradeTypeJSAPI,
		TradeTypeNative,
		TradeTypeApp,
		TradeTypeH5,
		TradeTypeWAP,
	}
}

// GetOrderStatuses 获取所有订单状态
func GetOrderStatuses() []OrderStatus {
	return []OrderStatus{
		OrderStatusPending,
		OrderStatusPaid,
		OrderStatusClosed,
		OrderStatusRefund,
	}
}

// GetRefundStatuses 获取所有退款状态
func GetRefundStatuses() []RefundStatus {
	return []RefundStatus{
		RefundStatusPending,
		RefundStatusProcessing,
		RefundStatusSuccess,
		RefundStatusFailed,
	}
}

// ValidateConfig 验证支付配置
func ValidateConfig(channel PaymentChannel, config map[string]interface{}) error {
	switch channel {
	case ChannelAlipay:
		return validateAlipayConfig(config)
	case ChannelWechat:
		return validateWechatConfig(config)
	default:
		return fmt.Errorf("unsupported payment channel: %s", channel)
	}
}

// validateAlipayConfig 验证支付宝配置
func validateAlipayConfig(config map[string]interface{}) error {
	requiredFields := []string{"app_id", "private_key", "alipay_public_key"}

	for _, field := range requiredFields {
		if value, exists := config[field]; !exists || value == "" {
			return fmt.Errorf("alipay config field '%s' is required", field)
		}
	}

	return nil
}

// validateWechatConfig 验证微信配置
func validateWechatConfig(config map[string]interface{}) error {
	requiredFields := []string{"app_id", "mch_id", "apiv3_key", "private_key", "serial_no"}

	for _, field := range requiredFields {
		if value, exists := config[field]; !exists || value == "" {
			return fmt.Errorf("wechat config field '%s' is required", field)
		}
	}

	return nil
}

// GetModuleInfo 获取模块信息
func GetModuleInfo() map[string]interface{} {
	return map[string]interface{}{
		"name":        "payment",
		"version":     "1.0.0",
		"description": "Yao支付模块，支持支付宝和微信支付",
		"providers":   GetProviders(),
		"trade_types": GetTradeTypes(),
		"processes":   GetProcesses(),
		"features": []string{
			"多商户支持",
			"多支付渠道",
			"多交易类型",
			"订单管理",
			"退款管理",
			"异步通知",
			"对账功能",
		},
	}
}

// HealthCheck 健康检查
func HealthCheck() map[string]interface{} {
	checks := make(map[string]interface{})
	result := map[string]interface{}{
		"status": "healthy",
		"checks": checks,
	}

	// 检查支付管理器
	if Manager == nil {
		result["status"] = "unhealthy"
		checks["manager"] = "Payment manager not initialized"
	} else {
		checks["manager"] = "OK"
	}

	// 检查支付提供商
	providers := make(map[string]string)
	if Manager != nil {
		for _, channel := range GetProviders() {
			if Manager.HasProvider(channel) {
				providers[string(channel)] = "registered"
			} else {
				providers[string(channel)] = "not registered"
				result["status"] = "unhealthy"
			}
		}
	}
	checks["providers"] = providers

	// 检查Process接口
	processes := make(map[string]string)
	for _, processName := range GetProcesses() {
		if process.Exists(processName) {
			processes[processName] = "registered"
		} else {
			processes[processName] = "not registered"
			result["status"] = "unhealthy"
		}
	}
	checks["processes"] = processes

	return result
}
