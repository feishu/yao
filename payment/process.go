package payment

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yaoapp/yao/payment/types"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
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
	if !IsValidChannel(types.PaymentChannel(channel)) {
		exception.New(fmt.Sprintf("不支持的支付渠道: %s", channel), 400).Throw()
	}

	// 设置配置
	err := Manager.SetMerchantConfig(merchantID, types.PaymentChannel(channel), config)
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
	if !IsValidChannel(types.PaymentChannel(channel)) {
		exception.New(fmt.Sprintf("不支持的支付渠道: %s", channel), 400).Throw()
	}

	// 获取配置
	config, err := Manager.GetMerchantConfig(merchantID, types.PaymentChannel(channel))
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
	if !IsValidChannel(types.PaymentChannel(channel)) {
		exception.New(fmt.Sprintf("不支持的支付渠道: %s", channel), 400).Throw()
	}

	// 处理通知
	result, err := Manager.HandleNotify(merchantID, types.PaymentChannel(channel), notifyData)
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
func parseCreateOrderParams(paramsMap map[string]interface{}) (*types.CreateOrderParams, error) {
	// 转换为JSON再解析，确保类型正确
	data, err := json.Marshal(paramsMap)
	if err != nil {
		return nil, fmt.Errorf("marshal params failed: %v", err)
	}

	var params types.CreateOrderParams
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
	if !IsValidChannel(types.PaymentChannel(params.Channel)) {
		return nil, fmt.Errorf("invalid channel: %s", params.Channel)
	}

	// 验证交易类型
	if !IsValidTradeType(types.TradeType(params.TradeType)) {
		return nil, fmt.Errorf("invalid trade_type: %s", params.TradeType)
	}

	return &params, nil
}

// parseQueryOrderParams 解析查询订单参数
func parseQueryOrderParams(paramsMap map[string]interface{}) (*types.QueryOrderParams, error) {
	data, err := json.Marshal(paramsMap)
	if err != nil {
		return nil, fmt.Errorf("marshal params failed: %v", err)
	}

	var params types.QueryOrderParams
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
	if !IsValidChannel(types.PaymentChannel(params.Channel)) {
		return nil, fmt.Errorf("invalid channel: %s", params.Channel)
	}

	return &params, nil
}

// parseCreateRefundParams 解析创建退款参数
func parseCreateRefundParams(paramsMap map[string]interface{}) (*types.CreateRefundParams, error) {
	data, err := json.Marshal(paramsMap)
	if err != nil {
		return nil, fmt.Errorf("marshal params failed: %v", err)
	}

	var params types.CreateRefundParams
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
	if !IsValidChannel(types.PaymentChannel(params.Channel)) {
		return nil, fmt.Errorf("invalid channel: %s", params.Channel)
	}

	return &params, nil
}

// parseQueryRefundParams 解析查询退款参数
func parseQueryRefundParams(paramsMap map[string]interface{}) (*types.QueryRefundParams, error) {
	data, err := json.Marshal(paramsMap)
	if err != nil {
		return nil, fmt.Errorf("marshal params failed: %v", err)
	}

	var params types.QueryRefundParams
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
	if !IsValidChannel(types.PaymentChannel(params.Channel)) {
		return nil, fmt.Errorf("invalid channel: %s", params.Channel)
	}

	return &params, nil
}

// parseDownloadBillParams 解析下载对账单参数
func parseDownloadBillParams(paramsMap map[string]interface{}) (*types.DownloadBillParams, error) {
	data, err := json.Marshal(paramsMap)
	if err != nil {
		return nil, fmt.Errorf("marshal params failed: %v", err)
	}

	var params types.DownloadBillParams
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
	if !IsValidChannel(types.PaymentChannel(params.Channel)) {
		return nil, fmt.Errorf("invalid channel: %s", params.Channel)
	}

	return &params, nil
}

// parseReconcileParams 解析对账参数
func parseReconcileParams(paramsMap map[string]interface{}) (*types.ReconcileParams, error) {
	data, err := json.Marshal(paramsMap)
	if err != nil {
		return nil, fmt.Errorf("marshal params failed: %v", err)
	}

	var params types.ReconcileParams
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
	if !IsValidChannel(types.PaymentChannel(params.Channel)) {
		return nil, fmt.Errorf("invalid channel: %s", params.Channel)
	}

	return &params, nil
}

// IsValidChannel 验证支付渠道是否有效
func IsValidChannel(channel types.PaymentChannel) bool {
	switch channel {
	case types.ChannelAlipay, types.ChannelWechat:
		return true
	default:
		return false
	}
}

// IsValidTradeType 验证交易类型是否有效
func IsValidTradeType(tradeType types.TradeType) bool {
	switch tradeType {
	case types.TradeTypeJSAPI, types.TradeTypeNative, types.TradeTypeApp, types.TradeTypeH5, types.TradeTypeWAP:
		return true
	default:
		return false
	}
}

// ProcessLoadCert 加载证书文件
// 参数：
//
//	args[0] (string): 证书文件路径（相对于app根目录）
//
// 返回：
//
//	map[string]interface{}: 包含证书内容的map
//	  - success (bool): 是否成功
//	  - content (string): 证书内容
//	  - path (string): 证书路径
func ProcessLoadCert(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	certPath := process.ArgsString(0)

	log.Debug("ProcessLoadCert: certPath=%s", certPath)

	// 验证路径安全性
	if !isValidCertPath(certPath) {
		exception.New(fmt.Sprintf("无效的证书路径: %s", certPath), 400).Throw()
	}

	// 获取app根目录
	if config.Conf.Root == "" {
		exception.New("应用配置未初始化，根目录为空", 500).Throw()
	}

	appRoot := config.Conf.Root
	fullPath := filepath.Join(appRoot, certPath)

	// 读取证书文件
	content, err := os.ReadFile(fullPath)
	if err != nil {
		log.Error("Failed to load cert: %v", err)
		exception.New(fmt.Sprintf("读取证书文件失败: %v", err), 500).Throw()
	}

	log.Debug("Cert loaded successfully: %s (%d bytes)", certPath, len(content))

	return map[string]interface{}{
		"success": true,
		"content": string(content),
		"path":    certPath,
	}
}

// ProcessLoadCertBase64 加载证书文件并转换为Base64编码
// 参数：
//
//	args[0] (string): 证书文件路径（相对于app根目录）
//
// 返回：
//
//	map[string]interface{}: 包含Base64编码证书内容的map
//	  - success (bool): 是否成功
//	  - content (string): Base64编码的证书内容
//	  - path (string): 证书路径
//	  - format (string): 编码格式（"base64"）
func ProcessLoadCertBase64(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	certPath := process.ArgsString(0)

	log.Debug("ProcessLoadCertBase64: certPath=%s", certPath)

	// 验证路径安全性
	if !isValidCertPath(certPath) {
		exception.New(fmt.Sprintf("无效的证书路径: %s", certPath), 400).Throw()
	}

	// 获取app根目录
	if config.Conf.Root == "" {
		exception.New("应用配置未初始化，根目录为空", 500).Throw()
	}

	appRoot := config.Conf.Root
	fullPath := filepath.Join(appRoot, certPath)

	// 读取证书文件
	content, err := os.ReadFile(fullPath)
	if err != nil {
		log.Error("Failed to load cert: %v", err)
		exception.New(fmt.Sprintf("读取证书文件失败: %v", err), 500).Throw()
	}

	// 转换为Base64
	encoded := base64.StdEncoding.EncodeToString(content)

	log.Debug("Cert loaded and encoded to base64: %s (%d bytes -> %d chars)", certPath, len(content), len(encoded))

	return map[string]interface{}{
		"success": true,
		"content": encoded,
		"path":    certPath,
		"format":  "base64",
	}
}

// ProcessGetCertificate 获取已加载的证书配置
// 参数：
//
//	args[0] (string): 商户ID
//	args[1] (string): 支付渠道 (alipay/wechat)
//
// 返回：
//
//	map[string]interface{}: 证书配置
//	  - success (bool): 是否成功
//	  - merchant_id (string): 商户ID
//	  - channel (string): 支付渠道
//	  - private_key (string): 私钥内容
//	  - public_key (string): 公钥内容
//	  - extra_files (map): 额外证书文件
func ProcessGetCertificate(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	merchantID := process.ArgsString(0)
	channel := process.ArgsString(1)

	log.Debug("ProcessGetCertificate: merchantID=%s, channel=%s", merchantID, channel)

	// 验证参数
	if merchantID == "" {
		exception.New("商户ID不能为空", 400).Throw()
	}

	if channel == "" {
		exception.New("支付渠道不能为空", 400).Throw()
	}

	// 验证支付渠道
	if !IsValidChannel(types.PaymentChannel(channel)) {
		exception.New(fmt.Sprintf("不支持的支付渠道: %s", channel), 400).Throw()
	}

	// 获取证书配置
	certConfig, err := Manager.GetCertConfig(merchantID, types.PaymentChannel(channel))
	if err != nil {
		log.Error("ProcessGetCertificate failed: %v", err)
		exception.New(fmt.Sprintf("获取证书配置失败: %v", err), 404).Throw()
	}

	return map[string]interface{}{
		"success":     true,
		"merchant_id": certConfig.MerchantID,
		"channel":     string(certConfig.Channel),
		"private_key": certConfig.PrivateKey,
		"public_key":  certConfig.PublicKey,
		"extra_files": certConfig.ExtraFiles,
	}
}

// ProcessListCertificates 列出所有已加载的证书配置
// 参数：
//
//	无
//
// 返回：
//
//	map[string]interface{}: 证书配置列表
func ProcessListCertificates(process *process.Process) interface{} {
	process.ValidateArgNums(0)

	log.Debug("ProcessListCertificates")

	Manager.mutex.RLock()
	defer Manager.mutex.RUnlock()

	certs := make([]map[string]interface{}, 0, len(Manager.certConfigs))

	for key, certConfig := range Manager.certConfigs {
		certs = append(certs, map[string]interface{}{
			"key":           key,
			"merchant_id":   certConfig.MerchantID,
			"channel":       string(certConfig.Channel),
			"has_app_cert":  certConfig.AppCert != "",
			"has_root_cert": certConfig.RootCert != "",
			"extra_files":   len(certConfig.ExtraFiles),
		})
	}

	return map[string]interface{}{
		"success": true,
		"count":   len(certs),
		"certs":   certs,
	}
}

// isValidCertPath 验证证书路径安全性
func isValidCertPath(path string) bool {
	// 防止路径穿越攻击
	if strings.Contains(path, "..") {
		log.Warn("Invalid cert path: contains '..': %s", path)
		return false
	}

	// 防止绝对路径
	if filepath.IsAbs(path) {
		log.Warn("Invalid cert path: absolute path not allowed: %s", path)
		return false
	}

	// 只允许特定扩展名
	ext := filepath.Ext(path)
	if ext == "" {
		log.Warn("Invalid cert path: no file extension: %s", path)
		return false
	}

	validExts := []string{".pem", ".crt", ".key", ".p12", ".pfx", ".cer"}
	for _, validExt := range validExts {
		if strings.EqualFold(ext, validExt) {
			return true
		}
	}

	log.Warn("Invalid cert path: unsupported extension %s: %s", ext, path)
	return false
}
