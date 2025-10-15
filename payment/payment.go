// Package payment 提供统一的支付接口，支持多种支付渠道
package payment

import (
	"github.com/yaoapp/yao/payment/types"
	"fmt"
	"sync"

	"github.com/yaoapp/kun/log"
)

// PaymentManager 支付管理器
type PaymentManager struct {
	providerFactories map[string]ProviderFactory   // Provider 工厂函数
	providerInstances map[string]types.PaymentProvider   // 缓存的 Provider 实例
	providers         map[string]types.PaymentProvider   // 旧的 providers（保留兼容）
	certConfigs       map[string]*CertConfig       // 证书配置缓存（统一配置入口）
	mutex             sync.RWMutex
}

// 全局支付管理器实例
var manager *PaymentManager
var once sync.Once

// Manager 全局支付管理器实例（供外部使用）
var Manager *PaymentManager

// ProviderFactory Provider 工厂函数类型
type ProviderFactory func(config map[string]interface{}) (types.PaymentProvider, error)

// NewPaymentManager 创建新的支付管理器
func NewPaymentManager() *PaymentManager {
	return &PaymentManager{
		providerFactories: make(map[string]ProviderFactory),
		providerInstances: make(map[string]types.PaymentProvider),
		providers:         make(map[string]types.PaymentProvider),
		certConfigs:       make(map[string]*CertConfig),
	}
}

// GetManager 获取全局支付管理器实例
func GetManager() *PaymentManager {
	once.Do(func() {
		manager = NewPaymentManager()
	})
	return manager
}

// RegisterProvider 注册支付提供商（兼容旧接口）
func (pm *PaymentManager) RegisterProvider(channel string, provider types.PaymentProvider) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	pm.providers[channel] = provider
	log.Info("Payment provider registered: %s", channel)
	return nil
}

// RegisterProviderFactory 注册 Provider 工厂函数
func (pm *PaymentManager) RegisterProviderFactory(channel string, factory ProviderFactory) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if factory == nil {
		return fmt.Errorf("factory cannot be nil")
	}

	pm.providerFactories[channel] = factory
	log.Info("Provider factory registered: %s", channel)
	return nil
}

// GetOrCreateProvider 获取或创建 Provider
func (pm *PaymentManager) GetOrCreateProvider(merchantID string, channel types.PaymentChannel) (types.PaymentProvider, error) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// 生成缓存 key
	cacheKey := fmt.Sprintf("%s_%s", merchantID, string(channel))

	// 检查缓存
	if provider, exists := pm.providerInstances[cacheKey]; exists {
		log.Debug("Provider found in cache: %s", cacheKey)
		return provider, nil
	}

	// 获取工厂函数
	factory, exists := pm.providerFactories[string(channel)]
	if !exists {
		return nil, fmt.Errorf("provider factory not found for channel: %s", channel)
	}

	// 从 certConfigs 获取配置
	configKey := fmt.Sprintf("%s_%s", merchantID, string(channel))
	certConfig, exists := pm.certConfigs[configKey]
	
	if !exists {
		return nil, fmt.Errorf("config not found for merchant: %s (please call payment.SetMerchantConfig or ensure certificates are loaded)", configKey)
	}

	// 将 CertConfig 转换为 Provider 配置
	finalConfig := certConfig.ToProviderConfig()
	
	log.Debug("Using config for %s: %d keys", configKey, len(finalConfig))

	// 使用工厂函数创建 Provider
	provider, err := factory(finalConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %v", err)
	}

	// 缓存 Provider 实例
	pm.providerInstances[cacheKey] = provider
	log.Info("✅ Provider created and cached: %s", cacheKey)

	return provider, nil
}

// InvalidateProviderCache 使 Provider 缓存失效（配置更新时调用）
func (pm *PaymentManager) InvalidateProviderCache(merchantID string, channel types.PaymentChannel) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	cacheKey := fmt.Sprintf("%s_%s", merchantID, string(channel))
	delete(pm.providerInstances, cacheKey)
	log.Info("Provider cache invalidated: %s", cacheKey)
}

// GetProvider 获取支付提供商
func (pm *PaymentManager) GetProvider(channel string) (types.PaymentProvider, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	provider, exists := pm.providers[channel]
	if !exists {
		return nil, fmt.Errorf("payment provider not found for channel: %s", channel)
	}

	return provider, nil
}

// HasProvider 检查是否存在指定的支付提供商
func (pm *PaymentManager) HasProvider(channel types.PaymentChannel) bool {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	_, exists := pm.providers[string(channel)]
	return exists
}

// SetConfig 设置支付配置（弃用，请使用 SetMerchantConfig）
func (pm *PaymentManager) SetConfig(merchantID string, config interface{}) error {
	return fmt.Errorf("SetConfig is deprecated, please use SetMerchantConfig(merchantID, channel, config)")
}

// GetConfig 获取支付配置（弃用，请使用 GetMerchantConfig）
func (pm *PaymentManager) GetConfig(merchantID string) (interface{}, error) {
	return nil, fmt.Errorf("GetConfig is deprecated, please use GetMerchantConfig(merchantID, channel)")
}

// CreateOrder 创建支付订单
func (pm *PaymentManager) CreateOrder(params *types.CreateOrderParams) (*types.CreateOrderResponse, error) {
	if params == nil {
		return &types.CreateOrderResponse{Success: false}, fmt.Errorf("params cannot be nil")
	}

	// 验证参数
	if err := pm.ValidateCreateOrderParams(params); err != nil {
		return &types.CreateOrderResponse{Success: false}, err
	}

	// 获取或创建支付提供商（根据商户ID和渠道）
	provider, err := pm.GetOrCreateProvider(params.MerchantNo, types.PaymentChannel(params.Channel))
	if err != nil {
		return &types.CreateOrderResponse{Success: false}, err
	}

	// 创建订单
	response, err := provider.CreateOrder(params)
	if err != nil {
		log.Error("Failed to create order: %v", err)
		return &types.CreateOrderResponse{Success: false}, err
	}

	log.Info("Order created successfully: %s", params.OutTradeNo)
	return response, nil
}

// QueryOrder 查询支付订单
func (pm *PaymentManager) QueryOrder(params *types.QueryOrderParams) (*types.QueryOrderResponse, error) {
	if params == nil {
		return &types.QueryOrderResponse{Success: false}, fmt.Errorf("params cannot be nil")
	}

	// 验证参数
	if err := pm.ValidateQueryOrderParams(params); err != nil {
		return &types.QueryOrderResponse{Success: false}, err
	}

	// 获取或创建支付提供商
	provider, err := pm.GetOrCreateProvider(params.MerchantNo, types.PaymentChannel(params.Channel))
	if err != nil {
		return &types.QueryOrderResponse{Success: false}, err
	}

	// 查询订单
	response, err := provider.QueryOrder(params)
	if err != nil {
		log.Error("Failed to query order: %v", err)
		return &types.QueryOrderResponse{Success: false}, err
	}

	return response, nil
}

// CreateRefund 创建退款
func (pm *PaymentManager) CreateRefund(params *types.CreateRefundParams) (*types.CreateRefundResponse, error) {
	if params == nil {
		return &types.CreateRefundResponse{Success: false}, fmt.Errorf("params cannot be nil")
	}

	// 验证参数
	if err := pm.ValidateCreateRefundParams(params); err != nil {
		return &types.CreateRefundResponse{Success: false, Error: err.Error()}, err
	}

	// 获取或创建支付提供商
	provider, err := pm.GetOrCreateProvider(params.MerchantNo, types.PaymentChannel(params.Channel))
	if err != nil {
		return &types.CreateRefundResponse{Success: false, Error: err.Error()}, err
	}

	// 创建退款
	response, err := provider.CreateRefund(params)
	if err != nil {
		log.Error("Failed to create refund: %v", err)
		return &types.CreateRefundResponse{Success: false, Error: err.Error()}, err
	}

	log.Info("Refund created successfully: %s", params.OutRefundNo)
	return response, nil
}

// QueryRefund 查询退款状态
func (pm *PaymentManager) QueryRefund(params *types.QueryRefundParams) (*types.QueryRefundResponse, error) {
	if params == nil {
		return &types.QueryRefundResponse{Success: false}, fmt.Errorf("params cannot be nil")
	}

	// 验证参数
	if err := pm.ValidateQueryRefundParams(params); err != nil {
		return &types.QueryRefundResponse{Success: false}, err
	}

	// 获取或创建支付提供商
	provider, err := pm.GetOrCreateProvider(params.MerchantNo, types.PaymentChannel(params.Channel))
	if err != nil {
		return &types.QueryRefundResponse{Success: false}, err
	}

	// 查询退款
	response, err := provider.QueryRefund(params)
	if err != nil {
		log.Error("Failed to query refund: %v", err)
		return &types.QueryRefundResponse{Success: false}, err
	}

	return response, nil
}

// HandleNotify 处理异步通知
func (pm *PaymentManager) HandleNotify(merchantID string, channel types.PaymentChannel, notifyData []byte) (*types.HandleNotifyResponse, error) {
	if merchantID == "" {
		return &types.HandleNotifyResponse{Success: false}, fmt.Errorf("merchant ID cannot be empty")
	}

	if len(notifyData) == 0 {
		return &types.HandleNotifyResponse{Success: false}, fmt.Errorf("request body is required")
	}

	// 获取或创建支付提供商
	provider, err := pm.GetOrCreateProvider(merchantID, channel)
	if err != nil {
		return &types.HandleNotifyResponse{Success: false}, err
	}

	// 构建通知参数
	params := &types.HandleNotifyParams{
		MerchantNo:  merchantID,
		Channel:     channel,
		RequestBody: notifyData,
		NotifyData:  make(map[string]interface{}),
	}

	// 处理通知
	response, err := provider.HandleNotify(params)
	if err != nil {
		log.Error("Failed to handle notify: %v", err)
		return &types.HandleNotifyResponse{Success: false}, err
	}

	log.Info("Notify handled successfully for merchant: %s", merchantID)
	return response, nil
}

// DownloadBill 下载对账单
func (pm *PaymentManager) DownloadBill(params *types.DownloadBillParams) (*types.DownloadBillResponse, error) {
	if params == nil {
		return &types.DownloadBillResponse{Success: false}, fmt.Errorf("params cannot be nil")
	}

	// 验证参数
	if err := pm.ValidateDownloadBillParams(params); err != nil {
		return &types.DownloadBillResponse{Success: false, Error: err.Error()}, err
	}

	// 获取或创建支付提供商
	provider, err := pm.GetOrCreateProvider(params.MerchantNo, types.PaymentChannel(params.Channel))
	if err != nil {
		return &types.DownloadBillResponse{Success: false, Error: err.Error()}, err
	}

	// 下载对账单
	response, err := provider.DownloadBill(params)
	if err != nil {
		log.Error("Failed to download bill: %v", err)
		return &types.DownloadBillResponse{Success: false, Error: err.Error()}, err
	}

	log.Info("Bill downloaded successfully: %s", params.BillDate)
	return response, nil
}

// SetMerchantConfig 设置商户配置
func (pm *PaymentManager) SetMerchantConfig(merchantID string, channel types.PaymentChannel, config map[string]interface{}) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if merchantID == "" {
		return fmt.Errorf("merchant ID cannot be empty")
	}

	if len(config) == 0 {
		return fmt.Errorf("config cannot be empty")
	}

	key := fmt.Sprintf("%s_%s", merchantID, string(channel))
	
	// 检查是否已经有证书配置（从文件加载）
	existingCert, hasExisting := pm.certConfigs[key]
	
	var certConfig *CertConfig
	if hasExisting {
	// 如果已有证书配置，复制并更新
		certConfig = &CertConfig{
			MerchantID: existingCert.MerchantID,
			Channel:    existingCert.Channel,
			PrivateKey: existingCert.PrivateKey,
			PublicKey:  existingCert.PublicKey,
			AppCert:    existingCert.AppCert,
			RootCert:   existingCert.RootCert,
			AppID:      existingCert.AppID,
			SignType:   existingCert.SignType,
			IsSandbox:  existingCert.IsSandbox,
			MchID:      existingCert.MchID,
			APIv3Key:   existingCert.APIv3Key,
			SerialNo:   existingCert.SerialNo,
			ExtraFiles:  make(map[string]string),
			ExtraFields: make(map[string]interface{}),
		}
		// 复制 ExtraFiles
		for k, v := range existingCert.ExtraFiles {
			certConfig.ExtraFiles[k] = v
		}
		// 复制 ExtraFields
		for k, v := range existingCert.ExtraFields {
			certConfig.ExtraFields[k] = v
		}
	} else {
		// 创建新的配置
		certConfig = &CertConfig{
			MerchantID:  merchantID,
			Channel:     channel,
			ExtraFiles:  make(map[string]string),
			ExtraFields: make(map[string]interface{}),
		}
	}
	
	// 从 config 中更新字段
	// 证书文件
	if privateKey, ok := config["private_key"].(string); ok {
		certConfig.PrivateKey = privateKey
	}
	if publicKey, ok := config["public_key"].(string); ok {
		certConfig.PublicKey = publicKey
	}
	if appCert, ok := config["app_cert"].(string); ok {
		certConfig.AppCert = appCert
	}
	if rootCert, ok := config["root_cert"].(string); ok {
		certConfig.RootCert = rootCert
	}
	
	// 支付宝配置
	if appID, ok := config["app_id"].(string); ok {
		certConfig.AppID = appID
	}
	if signType, ok := config["sign_type"].(string); ok {
		certConfig.SignType = signType
	}
	if isSandbox, ok := config["is_sandbox"].(bool); ok {
		certConfig.IsSandbox = isSandbox
	}
	
	// 微信配置
	if mchID, ok := config["mch_id"].(string); ok {
		certConfig.MchID = mchID
	}
	if apiv3Key, ok := config["apiv3_key"].(string); ok {
		certConfig.APIv3Key = apiv3Key
	}
	if serialNo, ok := config["serial_no"].(string); ok {
		certConfig.SerialNo = serialNo
	}
	
	// 其他字段存入 ExtraFields
	knownFields := map[string]bool{
		"private_key": true, "public_key": true, "app_cert": true, "root_cert": true,
		"app_id": true, "sign_type": true, "is_sandbox": true,
		"mch_id": true, "apiv3_key": true, "serial_no": true,
	}
	for k, v := range config {
		if !knownFields[k] {
			certConfig.ExtraFields[k] = v
		}
	}
	
	// 保存到 certConfigs
	pm.certConfigs[key] = certConfig
	log.Info("Merchant config set: %s", key)

	// 使对应的 Provider 缓存失效
	delete(pm.providerInstances, key)
	log.Debug("Provider cache invalidated due to config update: %s", key)

	return nil
}

// GetMerchantConfig 获取商户配置
func (pm *PaymentManager) GetMerchantConfig(merchantID string, channel types.PaymentChannel) (map[string]interface{}, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	key := fmt.Sprintf("%s_%s", merchantID, string(channel))
	certConfig, exists := pm.certConfigs[key]
	if !exists {
		return nil, fmt.Errorf("merchant config not found: %s", key)
	}

	// 转换为 map 返回
	return certConfig.ToProviderConfig(), nil
}

// Reconcile 对账
func (pm *PaymentManager) Reconcile(params *types.ReconcileParams) (*types.ReconcileResponse, error) {
	if params == nil {
		return &types.ReconcileResponse{Success: false}, fmt.Errorf("params cannot be nil")
	}

	// 验证参数
	if err := pm.ValidateReconcileParams(params); err != nil {
		return &types.ReconcileResponse{Success: false, Error: err.Error()}, err
	}

	// 获取或创建支付提供商
	provider, err := pm.GetOrCreateProvider(params.MerchantNo, types.PaymentChannel(params.Channel))
	if err != nil {
		return &types.ReconcileResponse{Success: false, Error: err.Error()}, err
	}

	// 下载对账单
	billParams := &types.DownloadBillParams{
		MerchantNo: params.MerchantNo,
		Channel:    params.Channel,
		BillDate:   params.BillDate,
		BillType:   "ALL",
	}

	billResponse, err := provider.DownloadBill(billParams)
	if err != nil {
		log.Error("Failed to download bill for reconcile: %v", err)
		return &types.ReconcileResponse{Success: false, Error: err.Error()}, err
	}

	if !billResponse.Success {
		return &types.ReconcileResponse{
			Success: false,
			Error:   billResponse.Error,
		}, fmt.Errorf("download bill failed: %s", billResponse.Error)
	}

	// 这里可以添加具体的对账逻辑
	// 比如解析对账单数据，与本地订单数据进行比对等

	log.Info("Reconcile completed successfully: %s", params.BillDate)
	return &types.ReconcileResponse{
		Success: true,
		Message: "对账完成",
	}, nil
}

// ValidateCreateOrderParams 验证创建订单参数
func (pm *PaymentManager) ValidateCreateOrderParams(params *types.CreateOrderParams) error {
	if params.MerchantNo == "" {
		return fmt.Errorf("merchant_no is required")
	}

	if params.OutTradeNo == "" {
		return fmt.Errorf("out_trade_no is required")
	}

	if params.Channel == "" {
		return fmt.Errorf("channel is required")
	}

	if params.TradeType == "" {
		return fmt.Errorf("trade_type is required")
	}

	if params.Amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}

	if params.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if params.NotifyURL == "" {
		return fmt.Errorf("notify_url is required")
	}

	// 验证支付渠道
	if !isValidPaymentChannel(string(params.Channel)) {
		return fmt.Errorf("invalid payment channel: %s", params.Channel)
	}

	// 验证交易类型
	if !isValidTradeType(string(params.TradeType)) {
		return fmt.Errorf("invalid trade type: %s", params.TradeType)
	}

	return nil
}

// ValidateQueryOrderParams 验证查询订单参数
func (pm *PaymentManager) ValidateQueryOrderParams(params *types.QueryOrderParams) error {
	if params.MerchantNo == "" {
		return fmt.Errorf("merchant_no is required")
	}

	if params.Channel == "" {
		return fmt.Errorf("channel is required")
	}

	if params.OutTradeNo == "" && params.TransactionID == "" {
		return fmt.Errorf("either out_trade_no or transaction_id is required")
	}

	// 验证支付渠道
	if !isValidPaymentChannel(string(params.Channel)) {
		return fmt.Errorf("invalid payment channel: %s", params.Channel)
	}

	return nil
}

// ValidateQueryRefundParams 验证查询退款参数
func (pm *PaymentManager) ValidateQueryRefundParams(params *types.QueryRefundParams) error {
	if params.Channel == "" {
		return fmt.Errorf("channel is required")
	}

	if params.OutRefundNo == "" && params.RefundID == "" {
		return fmt.Errorf("either out_refund_no or refund_id is required")
	}

	// 验证支付渠道
	if !isValidPaymentChannel(string(params.Channel)) {
		return fmt.Errorf("invalid payment channel: %s", params.Channel)
	}

	return nil
}

// ValidateDownloadBillParams 验证下载对账单参数
func (pm *PaymentManager) ValidateDownloadBillParams(params *types.DownloadBillParams) error {
	if params.Channel == "" {
		return fmt.Errorf("channel is required")
	}

	if params.BillDate == "" {
		return fmt.Errorf("bill_date is required")
	}

	// 验证支付渠道
	if !isValidPaymentChannel(string(params.Channel)) {
		return fmt.Errorf("invalid payment channel: %s", params.Channel)
	}

	// 验证对账单类型（可选）
	if params.BillType != "" && !isValidBillType(params.BillType) {
		return fmt.Errorf("invalid bill type: %s", params.BillType)
	}

	return nil
}

// ValidateCreateRefundParams 验证创建退款参数
func (pm *PaymentManager) ValidateCreateRefundParams(params *types.CreateRefundParams) error {
	if params.Channel == "" {
		return fmt.Errorf("channel is required")
	}

	if params.OutRefundNo == "" {
		return fmt.Errorf("out_refund_no is required")
	}

	if params.RefundAmount <= 0 {
		return fmt.Errorf("refund_amount must be greater than 0")
	}

	if params.TotalAmount <= 0 {
		return fmt.Errorf("total_amount must be greater than 0")
	}

	if params.OutTradeNo == "" && params.TransactionID == "" {
		return fmt.Errorf("either out_trade_no or transaction_id is required")
	}

	// 验证支付渠道
	if !isValidPaymentChannel(string(params.Channel)) {
		return fmt.Errorf("invalid payment channel: %s", params.Channel)
	}

	return nil
}

// ValidateReconcileParams 验证对账参数
func (pm *PaymentManager) ValidateReconcileParams(params *types.ReconcileParams) error {
	if params.Channel == "" {
		return fmt.Errorf("channel is required")
	}

	if params.BillDate == "" {
		return fmt.Errorf("bill_date is required")
	}

	// 验证支付渠道
	if !isValidPaymentChannel(string(params.Channel)) {
		return fmt.Errorf("invalid payment channel: %s", params.Channel)
	}

	return nil
}

// isValidPaymentChannel 验证支付渠道是否有效
func isValidPaymentChannel(channel string) bool {
	validChannels := []string{
		string(types.ChannelAlipay),
		string(types.ChannelWechat),
	}

	for _, validChannel := range validChannels {
		if channel == validChannel {
			return true
		}
	}

	return false
}

// isValidBillType 验证账单类型是否有效
func isValidBillType(billType string) bool {
	validTypes := []string{"ALL", "SUCCESS", "REFUND"}
	for _, validType := range validTypes {
		if billType == validType {
			return true
		}
	}
	return false
}

// GetSupportedProviders 获取所有支持的支付渠道
func GetSupportedProviders() []types.PaymentChannel {
	return []types.PaymentChannel{
		types.PaymentChannel(types.ChannelAlipay),
		types.PaymentChannel(types.ChannelWechat),
	}
}

// isValidTradeType 验证交易类型是否有效
func isValidTradeType(tradeType string) bool {
	validTypes := []string{
		string(types.TradeTypeNative),
		string(types.TradeTypeJSAPI),
		string(types.TradeTypeApp),
		string(types.TradeTypeH5),
	}
	for _, validType := range validTypes {
		if tradeType == validType {
			return true
		}
	}
	return false
}
