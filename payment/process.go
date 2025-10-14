package payment

import (
	"encoding/json"
	"fmt"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
)

// ProcessSetConfig 设置商户配置
// 参数：
//
//	args[0] (string): 商户ID
//	args[1] (string): 支付渠道 (alipay/wechat)
//	args[2] (map[string]interface{}): 配置信息
//
// 返回：
//
//	interface{}: 设置结果
func ProcessSetConfig(process *process.Process) interface{} {
	process.ValidateArgNums(3)

	merchantID := process.ArgsString(0)
	channel := process.ArgsString(1)
	config := process.ArgsMap(2)

	log.Debug("ProcessSetConfig: merchantID=%s, channel=%s", merchantID, channel)

	// 验证参数
	if merchantID == "" {
		exception.New("商户ID不能为空", 400).Throw()
	}

	if channel == "" {
		exception.New("支付渠道不能为空", 400).Throw()
	}

	if len(config) == 0 {
		exception.New("配置信息不能为空", 400).Throw()
	}

	// 验证支付渠道
	if !IsValidChannel(PaymentChannel(channel)) {
		exception.New(fmt.Sprintf("不支持的支付渠道: %s", channel), 400).Throw()
	}

	// 设置配置
	err := Manager.SetMerchantConfig(merchantID, PaymentChannel(channel), config)
	if err != nil {
		log.Error("ProcessSetConfig failed: %v", err)
		exception.New(fmt.Sprintf("设置商户配置失败: %v", err), 500).Throw()
	}

	return map[string]interface{}{
		"success": true,
		"message": "配置设置成功",
	}
}

// ProcessGetConfig 获取商户配置
// 参数：
//
//	args[0] (string): 商户ID
//	args[1] (string): 支付渠道 (alipay/wechat)
//
// 返回：
//
//	interface{}: 配置信息
func ProcessGetConfig(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	merchantID := process.ArgsString(0)
	channel := process.ArgsString(1)

	log.Debug("ProcessGetConfig: merchantID=%s, channel=%s", merchantID, channel)

	// 验证参数
	if merchantID == "" {
		exception.New("商户ID不能为空", 400).Throw()
	}

	if channel == "" {
		exception.New("支付渠道不能为空", 400).Throw()
	}

	// 验证支付渠道
	if !IsValidChannel(PaymentChannel(channel)) {
		exception.New(fmt.Sprintf("不支持的支付渠道: %s", channel), 400).Throw()
	}

	// 获取配置
	config, err := Manager.GetMerchantConfig(merchantID, PaymentChannel(channel))
	if err != nil {
		log.Error("ProcessGetConfig failed: %v", err)
		exception.New(fmt.Sprintf("获取商户配置失败: %v", err), 500).Throw()
	}

	return map[string]interface{}{
		"success": true,
		"config":  config,
		"message": "获取配置成功",
	}
}

// ProcessCreateOrder 创建支付订单
// 参数：
//
//	args[0] (map[string]interface{}): 创建订单参数
//
// 返回：
//
//	interface{}: 创建结果
func ProcessCreateOrder(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	paramsMap := process.ArgsMap(0)

	log.Debug("ProcessCreateOrder: params=%+v", paramsMap)

	// 解析参数
	params, err := parseCreateOrderParams(paramsMap)
	if err != nil {
		log.Error("ProcessCreateOrder parse params failed: %v", err)
		exception.New(fmt.Sprintf("参数解析失败: %v", err), 400).Throw()
	}

	// 创建订单
	result, err := Manager.CreateOrder(params)
	if err != nil {
		log.Error("ProcessCreateOrder failed: %v", err)
		exception.New(fmt.Sprintf("创建订单失败: %v", err), 500).Throw()
	}

	return result
}

// ProcessQueryOrder 查询支付订单
// 参数：
//
//	args[0] (map[string]interface{}): 查询订单参数
//
// 返回：
//
//	interface{}: 查询结果
func ProcessQueryOrder(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	paramsMap := process.ArgsMap(0)

	log.Debug("ProcessQueryOrder: params=%+v", paramsMap)

	// 解析参数
	params, err := parseQueryOrderParams(paramsMap)
	if err != nil {
		log.Error("ProcessQueryOrder parse params failed: %v", err)
		exception.New(fmt.Sprintf("参数解析失败: %v", err), 400).Throw()
	}

	// 查询订单
	result, err := Manager.QueryOrder(params)
	if err != nil {
		log.Error("ProcessQueryOrder failed: %v", err)
		exception.New(fmt.Sprintf("查询订单失败: %v", err), 500).Throw()
	}

	return result
}

// ProcessCreateRefund 创建退款
// 参数：
//
//	args[0] (map[string]interface{}): 创建退款参数
//
// 返回：
//
//	interface{}: 创建结果
func ProcessCreateRefund(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	paramsMap := process.ArgsMap(0)

	log.Debug("ProcessCreateRefund: params=%+v", paramsMap)

	// 解析参数
	params, err := parseCreateRefundParams(paramsMap)
	if err != nil {
		log.Error("ProcessCreateRefund parse params failed: %v", err)
		exception.New(fmt.Sprintf("参数解析失败: %v", err), 400).Throw()
	}

	// 创建退款
	result, err := Manager.CreateRefund(params)
	if err != nil {
		log.Error("ProcessCreateRefund failed: %v", err)
		exception.New(fmt.Sprintf("创建退款失败: %v", err), 500).Throw()
	}

	return result
}

// ProcessQueryRefund 查询退款状态
// 参数：
//
//	args[0] (map[string]interface{}): 查询退款参数
//
// 返回：
//
//	interface{}: 查询结果
func ProcessQueryRefund(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	paramsMap := process.ArgsMap(0)

	log.Debug("ProcessQueryRefund: params=%+v", paramsMap)

	// 解析参数
	params, err := parseQueryRefundParams(paramsMap)
	if err != nil {
		log.Error("ProcessQueryRefund parse params failed: %v", err)
		exception.New(fmt.Sprintf("参数解析失败: %v", err), 400).Throw()
	}

	// 查询退款
	result, err := Manager.QueryRefund(params)
	if err != nil {
		log.Error("ProcessQueryRefund failed: %v", err)
		exception.New(fmt.Sprintf("查询退款失败: %v", err), 500).Throw()
	}

	return result
}

// ProcessHandleNotify 处理异步通知
// 参数：
//
//	args[0] (string): 商户ID
//	args[1] (string): 支付渠道 (alipay/wechat)
//	args[2] ([]byte): 通知数据
//
// 返回：
//
//	interface{}: 处理结果
func ProcessHandleNotify(process *process.Process) interface{} {
	process.ValidateArgNums(3)

	merchantID := process.ArgsString(0)
	channel := process.ArgsString(1)
	notifyDataStr := process.ArgsString(2)
	notifyData := []byte(notifyDataStr)

	log.Debug("ProcessHandleNotify: merchantID=%s, channel=%s", merchantID, channel)

	// 验证参数
	if merchantID == "" {
		exception.New("商户ID不能为空", 400).Throw()
	}

	if channel == "" {
		exception.New("支付渠道不能为空", 400).Throw()
	}

	if len(notifyData) == 0 {
		exception.New("通知数据不能为空", 400).Throw()
	}

	// 验证支付渠道
	if !IsValidChannel(PaymentChannel(channel)) {
		exception.New(fmt.Sprintf("不支持的支付渠道: %s", channel), 400).Throw()
	}

	// 处理通知
	result, err := Manager.HandleNotify(merchantID, PaymentChannel(channel), notifyData)
	if err != nil {
		log.Error("ProcessHandleNotify failed: %v", err)
		exception.New(fmt.Sprintf("处理通知失败: %v", err), 500).Throw()
	}

	return result
}

// ProcessDownloadBill 下载对账单
// 参数：
//
//	args[0] (map[string]interface{}): 下载对账单参数
//
// 返回：
//
//	interface{}: 下载结果
func ProcessDownloadBill(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	paramsMap := process.ArgsMap(0)

	log.Debug("ProcessDownloadBill: params=%+v", paramsMap)

	// 解析参数
	params, err := parseDownloadBillParams(paramsMap)
	if err != nil {
		log.Error("ProcessDownloadBill parse params failed: %v", err)
		exception.New(fmt.Sprintf("参数解析失败: %v", err), 400).Throw()
	}

	// 下载对账单
	result, err := Manager.DownloadBill(params)
	if err != nil {
		log.Error("ProcessDownloadBill failed: %v", err)
		exception.New(fmt.Sprintf("下载对账单失败: %v", err), 500).Throw()
	}

	return result
}

// ProcessReconcile 对账
// 参数：
//
//	args[0] (map[string]interface{}): 对账参数
//
// 返回：
//
//	interface{}: 对账结果
func ProcessReconcile(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	paramsMap := process.ArgsMap(0)

	log.Debug("ProcessReconcile: params=%+v", paramsMap)

	// 解析参数
	params, err := parseReconcileParams(paramsMap)
	if err != nil {
		log.Error("ProcessReconcile parse params failed: %v", err)
		exception.New(fmt.Sprintf("参数解析失败: %v", err), 400).Throw()
	}

	// 对账
	result, err := Manager.Reconcile(params)
	if err != nil {
		log.Error("ProcessReconcile failed: %v", err)
		exception.New(fmt.Sprintf("对账失败: %v", err), 500).Throw()
	}

	return result
}

// parseCreateOrderParams 解析创建订单参数
func parseCreateOrderParams(paramsMap map[string]interface{}) (*CreateOrderParams, error) {
	// 转换为JSON再解析，确保类型正确
	data, err := json.Marshal(paramsMap)
	if err != nil {
		return nil, fmt.Errorf("marshal params failed: %v", err)
	}

	var params CreateOrderParams
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, fmt.Errorf("unmarshal params failed: %v", err)
	}

	// 验证必要参数
	if params.MerchantNo == "" {
		return nil, fmt.Errorf("merchant_no is required")
	}

	if params.Channel == "" {
		return nil, fmt.Errorf("channel is required")
	}

	if params.OutTradeNo == "" {
		return nil, fmt.Errorf("out_trade_no is required")
	}

	if params.Amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than 0")
	}

	if params.Subject == "" {
		return nil, fmt.Errorf("subject is required")
	}

	if params.TradeType == "" {
		return nil, fmt.Errorf("trade_type is required")
	}

	// 验证支付渠道
	if !IsValidChannel(PaymentChannel(params.Channel)) {
		return nil, fmt.Errorf("invalid channel: %s", params.Channel)
	}

	// 验证交易类型
	if !IsValidTradeType(TradeType(params.TradeType)) {
		return nil, fmt.Errorf("invalid trade_type: %s", params.TradeType)
	}

	return &params, nil
}

// parseQueryOrderParams 解析查询订单参数
func parseQueryOrderParams(paramsMap map[string]interface{}) (*QueryOrderParams, error) {
	data, err := json.Marshal(paramsMap)
	if err != nil {
		return nil, fmt.Errorf("marshal params failed: %v", err)
	}

	var params QueryOrderParams
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, fmt.Errorf("unmarshal params failed: %v", err)
	}

	// 验证必要参数
	if params.MerchantNo == "" {
		return nil, fmt.Errorf("merchant_no is required")
	}

	if params.Channel == "" {
		return nil, fmt.Errorf("channel is required")
	}

	if params.OutTradeNo == "" && params.TransactionID == "" {
		return nil, fmt.Errorf("out_trade_no or transaction_id is required")
	}

	// 验证支付渠道
	if !IsValidChannel(PaymentChannel(params.Channel)) {
		return nil, fmt.Errorf("invalid channel: %s", params.Channel)
	}

	return &params, nil
}

// parseCreateRefundParams 解析创建退款参数
func parseCreateRefundParams(paramsMap map[string]interface{}) (*CreateRefundParams, error) {
	data, err := json.Marshal(paramsMap)
	if err != nil {
		return nil, fmt.Errorf("marshal params failed: %v", err)
	}

	var params CreateRefundParams
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, fmt.Errorf("unmarshal params failed: %v", err)
	}

	// 验证必要参数
	if params.MerchantNo == "" {
		return nil, fmt.Errorf("merchant_no is required")
	}

	if params.Channel == "" {
		return nil, fmt.Errorf("channel is required")
	}

	if params.OutRefundNo == "" {
		return nil, fmt.Errorf("out_refund_no is required")
	}

	if params.RefundAmount <= 0 {
		return nil, fmt.Errorf("refund_amount must be greater than 0")
	}

	if params.TotalAmount <= 0 {
		return nil, fmt.Errorf("total_amount must be greater than 0")
	}

	if params.OutTradeNo == "" && params.TransactionID == "" {
		return nil, fmt.Errorf("out_trade_no or transaction_id is required")
	}

	// 验证支付渠道
	if !IsValidChannel(PaymentChannel(params.Channel)) {
		return nil, fmt.Errorf("invalid channel: %s", params.Channel)
	}

	return &params, nil
}

// parseQueryRefundParams 解析查询退款参数
func parseQueryRefundParams(paramsMap map[string]interface{}) (*QueryRefundParams, error) {
	data, err := json.Marshal(paramsMap)
	if err != nil {
		return nil, fmt.Errorf("marshal params failed: %v", err)
	}

	var params QueryRefundParams
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, fmt.Errorf("unmarshal params failed: %v", err)
	}

	// 验证必要参数
	if params.MerchantNo == "" {
		return nil, fmt.Errorf("merchant_no is required")
	}

	if params.Channel == "" {
		return nil, fmt.Errorf("channel is required")
	}

	if params.OutRefundNo == "" {
		return nil, fmt.Errorf("out_refund_no is required")
	}

	// 验证支付渠道
	if !IsValidChannel(PaymentChannel(params.Channel)) {
		return nil, fmt.Errorf("invalid channel: %s", params.Channel)
	}

	return &params, nil
}

// parseDownloadBillParams 解析下载对账单参数
func parseDownloadBillParams(paramsMap map[string]interface{}) (*DownloadBillParams, error) {
	data, err := json.Marshal(paramsMap)
	if err != nil {
		return nil, fmt.Errorf("marshal params failed: %v", err)
	}

	var params DownloadBillParams
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, fmt.Errorf("unmarshal params failed: %v", err)
	}

	// 验证必要参数
	if params.MerchantNo == "" {
		return nil, fmt.Errorf("merchant_no is required")
	}

	if params.Channel == "" {
		return nil, fmt.Errorf("channel is required")
	}

	if params.BillDate == "" {
		return nil, fmt.Errorf("bill_date is required")
	}

	// 验证支付渠道
	if !IsValidChannel(PaymentChannel(params.Channel)) {
		return nil, fmt.Errorf("invalid channel: %s", params.Channel)
	}

	return &params, nil
}

// parseReconcileParams 解析对账参数
func parseReconcileParams(paramsMap map[string]interface{}) (*ReconcileParams, error) {
	data, err := json.Marshal(paramsMap)
	if err != nil {
		return nil, fmt.Errorf("marshal params failed: %v", err)
	}

	var params ReconcileParams
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, fmt.Errorf("unmarshal params failed: %v", err)
	}

	// 验证必要参数
	if params.MerchantNo == "" {
		return nil, fmt.Errorf("merchant_no is required")
	}

	if params.Channel == "" {
		return nil, fmt.Errorf("channel is required")
	}

	if params.BillDate == "" {
		return nil, fmt.Errorf("bill_date is required")
	}

	// 验证支付渠道
	if !IsValidChannel(PaymentChannel(params.Channel)) {
		return nil, fmt.Errorf("invalid channel: %s", params.Channel)
	}

	return &params, nil
}

// IsValidChannel 验证支付渠道是否有效
func IsValidChannel(channel PaymentChannel) bool {
	switch channel {
	case ChannelAlipay, ChannelWechat:
		return true
	default:
		return false
	}
}

// IsValidTradeType 验证交易类型是否有效
func IsValidTradeType(tradeType TradeType) bool {
	switch tradeType {
	case TradeTypeJSAPI, TradeTypeNative, TradeTypeApp, TradeTypeH5, TradeTypeWAP:
		return true
	default:
		return false
	}
}
