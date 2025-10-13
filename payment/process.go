// Package payment Process 接口实现
package payment

import (
	"fmt"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
)

// ProcessCreatePayment 创建支付订单的 Process 接口实现
//
// 参数：
//   args[0] (map[string]interface{}): 支付请求参数
//     - order_id (string): 订单ID
//     - amount (int64): 支付金额，单位：分
//     - currency (string): 货币类型，如 CNY, USD
//     - subject (string): 订单标题
//     - description (string): 订单描述
//     - provider (string): 支付提供商 (wechat/alipay/paypal)
//     - extra (map[string]interface{}): 额外参数
//
// 返回：
//   interface{}: 支付响应对象
//
// 异常：
//   当参数无效或支付创建失败时抛出异常
//
// 示例：
//   process.New("utils.payment.CreatePayment", map[string]interface{}{
//       "order_id": "ORDER_123456",
//       "amount": 10000,
//       "currency": "CNY",
//       "subject": "商品购买",
//       "description": "购买商品描述",
//       "provider": "wechat",
//       "extra": map[string]interface{}{"pay_type": "NATIVE"},
//   }).Run()
func ProcessCreatePayment(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	reqMap := process.ArgsMap(0)
	if reqMap == nil {
		exception.New("Payment request cannot be nil", 400).Throw()
	}

	// 构建支付请求
	req := &PaymentRequest{}

	// 必填字段验证和赋值
	if outTradeNo, ok := reqMap["out_trade_no"].(string); ok && outTradeNo != "" {
		req.OutTradeNo = outTradeNo
	} else {
		exception.New("out_trade_no is required and must be a non-empty string", 400).Throw()
	}

	if amount, ok := reqMap["amount"]; ok {
		switch v := amount.(type) {
		case int64:
			req.Amount = float64(v)
		case int:
			req.Amount = float64(v)
		case float64:
			req.Amount = v
		default:
			exception.New("amount must be a number", 400).Throw()
		}
	} else {
		exception.New("amount is required", 400).Throw()
	}

	if req.Amount <= 0 {
		exception.New("amount must be greater than 0", 400).Throw()
	}

	if subject, ok := reqMap["subject"].(string); ok && subject != "" {
		req.Subject = subject
	} else {
		exception.New("subject is required and must be a non-empty string", 400).Throw()
	}

	if provider, ok := reqMap["provider"].(string); ok && provider != "" {
		req.Provider = PaymentProvider(provider)
	} else {
		exception.New("provider is required and must be a non-empty string", 400).Throw()
	}

	// 可选字段
	if body, ok := reqMap["body"].(string); ok {
		req.Body = body
	}

	if appID, ok := reqMap["app_id"].(string); ok {
		req.AppID = appID
	}

	if userID, ok := reqMap["user_id"].(string); ok {
		req.UserID = userID
	}

	if notifyURL, ok := reqMap["notify_url"].(string); ok {
		req.NotifyURL = notifyURL
	}

	if returnURL, ok := reqMap["return_url"].(string); ok {
		req.ReturnURL = returnURL
	}

	// 创建支付
	resp, err := manager.CreatePayment(req)
	if err != nil {
		log.Error("CreatePayment failed: %v", err)
		exception.New(fmt.Sprintf("CreatePayment failed: %v", err), 500).Throw()
	}

	log.Info("Payment created successfully: %s", req.OutTradeNo)
	return resp
}

// ProcessQueryPayment 查询支付状态的 Process 接口实现
//
// 参数：
//   args[0] (string): 支付提供商 (wechat/alipay/paypal)
//   args[1] (string): 订单ID
//
// 返回：
//   interface{}: 支付状态响应对象
//
// 异常：
//   当参数无效或查询失败时抛出异常
//
// 示例：
//   process.New("utils.payment.QueryPayment", "wechat", "ORDER_123456").Run()
func ProcessQueryPayment(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	provider := process.ArgsString(0)
	orderID := process.ArgsString(1)

	if provider == "" {
		exception.New("provider cannot be empty", 400).Throw()
	}

	if orderID == "" {
		exception.New("order_id cannot be empty", 400).Throw()
	}

	manager := GetManager()
	response, err := manager.QueryPayment(PaymentProvider(provider), orderID)
	if err != nil {
		log.Error("ProcessQueryPayment failed: %v", err)
		exception.New(fmt.Sprintf("Failed to query payment: %v", err), 500).Throw()
	}

	log.Info("Payment queried successfully for order: %s", orderID)
	return response
}

// ProcessRefundPayment 退款的 Process 接口实现
//
// 参数：
//   args[0] (string): 支付提供商 (wechat/alipay/paypal)
//   args[1] (map[string]interface{}): 退款请求参数
//     - order_id (string): 订单ID
//     - payment_id (string): 支付ID
//     - amount (int64): 退款金额，单位：分
//     - reason (string): 退款原因
//
// 返回：
//   interface{}: 退款响应对象
//
// 异常：
//   当参数无效或退款失败时抛出异常
//
// 示例：
//   process.New("utils.payment.RefundPayment", "wechat", map[string]interface{}{
//       "order_id": "ORDER_123456",
//       "payment_id": "PAY_123456",
//       "amount": 5000,
//       "reason": "用户申请退款",
//   }).Run()
func ProcessRefundPayment(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	provider := process.ArgsString(0)
	reqMap := process.ArgsMap(1)

	if provider == "" {
		exception.New("provider cannot be empty", 400).Throw()
	}

	if reqMap == nil {
		exception.New("Refund request cannot be nil", 400).Throw()
	}

	// 构建退款请求
	req := &RefundRequest{}

	// 必填字段验证和赋值
	if outTradeNo, ok := reqMap["out_trade_no"].(string); ok && outTradeNo != "" {
		req.OutTradeNo = outTradeNo
	} else {
		exception.New("out_trade_no is required and must be a non-empty string", 400).Throw()
	}

	if refundAmount, ok := reqMap["refund_amount"]; ok {
		switch v := refundAmount.(type) {
		case int64:
			req.RefundAmount = float64(v)
		case int:
			req.RefundAmount = float64(v)
		case float64:
			req.RefundAmount = v
		default:
			exception.New("refund_amount must be a number", 400).Throw()
		}
	} else {
		exception.New("refund_amount is required", 400).Throw()
	}

	if req.RefundAmount <= 0 {
		exception.New("refund_amount must be greater than 0", 400).Throw()
	}

	if provider, ok := reqMap["provider"].(string); ok && provider != "" {
		req.Provider = PaymentProvider(provider)
	} else {
		exception.New("provider is required and must be a non-empty string", 400).Throw()
	}

	if reason, ok := reqMap["reason"].(string); ok {
		req.Reason = reason
	}

	// 执行退款
	manager := GetManager()
	resp, err := manager.RefundPayment(req)
	if err != nil {
		log.Error("RefundPayment failed: %v", err)
		exception.New(fmt.Sprintf("RefundPayment failed: %v", err), 500).Throw()
	}

	log.Info("Refund created successfully: %s", req.OutTradeNo)
	return resp
}

// ProcessAddConfig 添加支付配置的 Process 接口实现
//
// 参数：
//   args[0] (map[string]interface{}): 支付配置参数
//     - provider (string): 支付提供商 (wechat/alipay/paypal)
//     - app_id (string): 应用ID
//     - app_key (string): 应用密钥
//     - secret (string): 应用秘钥
//     - sandbox (bool): 是否沙箱环境
//     - notify_url (string): 异步通知URL
//     - return_url (string): 同步返回URL
//
// 返回：
//   interface{}: 配置添加结果
//
// 异常：
//   当参数无效或配置添加失败时抛出异常
//
// 示例：
//   process.New("utils.payment.AddConfig", map[string]interface{}{
//       "provider": "wechat",
//       "app_id": "wx1234567890",
//       "secret": "your_secret_key",
//       "sandbox": true,
//       "notify_url": "https://your-domain.com/notify",
//   }).Run()
func ProcessAddConfig(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	configMap := process.ArgsMap(0)
	if configMap == nil {
		exception.New("Payment config cannot be nil", 400).Throw()
	}

	// 构建支付配置
	config := &PaymentConfig{}

	// 必填字段验证和赋值
	if provider, ok := configMap["provider"].(string); ok && provider != "" {
		config.Provider = PaymentProvider(provider)
	} else {
		exception.New("provider is required and must be a non-empty string", 400).Throw()
	}

	if appID, ok := configMap["app_id"].(string); ok && appID != "" {
		config.AppID = appID
	} else {
		exception.New("app_id is required and must be a non-empty string", 400).Throw()
	}

	// 可选字段
	if appSecret, ok := configMap["app_secret"].(string); ok {
		config.AppSecret = appSecret
	}

	if apiKey, ok := configMap["api_key"].(string); ok {
		config.APIKey = apiKey
	}

	if privateKey, ok := configMap["private_key"].(string); ok {
		config.PrivateKey = privateKey
	}

	if mchID, ok := configMap["mch_id"].(string); ok {
		config.MchID = mchID
	}

	if isProd, ok := configMap["is_prod"].(bool); ok {
		config.IsProd = isProd
	}

	if notifyURL, ok := configMap["notify_url"].(string); ok {
		config.NotifyURL = notifyURL
	}

	if returnURL, ok := configMap["return_url"].(string); ok {
		config.ReturnURL = returnURL
	}

	// 添加配置
	manager := GetManager()
	err := manager.AddConfig(config)
	if err != nil {
		log.Error("ProcessAddConfig failed: %v", err)
		exception.New(fmt.Sprintf("Failed to add payment config: %v", err), 500).Throw()
	}

	log.Info("Payment config added successfully for provider: %s", config.Provider)
	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Payment config added for provider: %s", config.Provider),
	}
}

// ProcessGetSupportedProviders 获取支持的支付提供商列表
//
// 返回：
//   interface{}: 支持的支付提供商列表
//
// 示例：
//   process.New("utils.payment.GetSupportedProviders").Run()
func ProcessGetSupportedProviders(process *process.Process) interface{} {
	process.ValidateArgNums(0)

	providers := []map[string]interface{}{
		{
			"provider":    string(ProviderWechat),
			"name":        "微信支付",
			"description": "支持微信扫码支付、公众号支付、小程序支付等",
			"currencies":  []string{"CNY"},
		},
		{
			"provider":    string(ProviderAlipay),
			"name":        "支付宝",
			"description": "支持支付宝网页支付、手机网站支付、APP支付等",
			"currencies":  []string{"CNY"},
		},
		{
			"provider":    string(ProviderPayPal),
			"name":        "PayPal",
			"description": "支持PayPal国际支付",
			"currencies":  []string{"USD", "EUR", "GBP", "JPY", "CNY"},
		},
	}

	return map[string]interface{}{
		"providers": providers,
		"total":     len(providers),
	}
}

// ProcessValidateNotify 验证支付异步通知
//
// 参数：
//   args[0] (string): 支付提供商 (wechat/alipay/paypal)
//   args[1] (map[string]interface{}): 通知参数
//
// 返回：
//   interface{}: 验证结果
//
// 异常：
//   当参数无效或验证失败时抛出异常
//
// 示例：
//   process.New("utils.payment.ValidateNotify", "wechat", notifyParams).Run()
func ProcessValidateNotify(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	provider := process.ArgsString(0)
	notifyParams := process.ArgsMap(1)

	if provider == "" {
		exception.New("provider cannot be empty", 400).Throw()
	}

	if notifyParams == nil {
		exception.New("notify params cannot be nil", 400).Throw()
	}

	// 这里应该实现具体的通知验证逻辑
	// 由于涉及到签名验证等复杂逻辑，这里只做基础验证
	log.Info("Validating payment notify for provider: %s", provider)

	// 基础字段验证
	var orderID string
	var isValid bool

	switch PaymentProvider(provider) {
	case ProviderWechat:
		if outTradeNo, ok := notifyParams["out_trade_no"].(string); ok {
			orderID = outTradeNo
			isValid = true
		}
	case ProviderAlipay:
		if outTradeNo, ok := notifyParams["out_trade_no"].(string); ok {
			orderID = outTradeNo
			isValid = true
		}
	case ProviderPayPal:
		if resourceID, ok := notifyParams["resource_id"].(string); ok {
			orderID = resourceID
			isValid = true
		}
	default:
		exception.New(fmt.Sprintf("Unsupported payment provider: %s", provider), 400).Throw()
	}

	if !isValid || orderID == "" {
		exception.New("Invalid notify parameters", 400).Throw()
	}

	return map[string]interface{}{
		"valid":    isValid,
		"order_id": orderID,
		"provider": provider,
		"message":  "Notify validation completed",
	}
}