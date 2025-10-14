// Package payment 提供统一的支付接口，支持多种支付渠道
package payment

import (
	"fmt"
	"sync"

	"github.com/yaoapp/kun/log"
)

// PaymentManager 支付管理器
type PaymentManager struct {
	providers map[string]PaymentProvider
	configs   map[string]interface{}
	mutex     sync.RWMutex
}

// 全局支付管理器实例
var manager *PaymentManager
var once sync.Once

// Manager 全局支付管理器实例（供外部使用）
var Manager *PaymentManager

// NewPaymentManager 创建新的支付管理器
func NewPaymentManager() *PaymentManager {
	return &PaymentManager{
		providers: make(map[string]PaymentProvider),
		configs:   make(map[string]interface{}),
	}
}

// GetManager 获取全局支付管理器实例
func GetManager() *PaymentManager {
	once.Do(func() {
		manager = NewPaymentManager()
	})
	return manager
}

// RegisterProvider 注册支付提供商
func (pm *PaymentManager) RegisterProvider(channel string, provider PaymentProvider) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	pm.providers[channel] = provider
	log.Info("Payment provider registered: %s", channel)
	return nil
}

// GetProvider 获取支付提供商
func (pm *PaymentManager) GetProvider(channel string) (PaymentProvider, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	provider, exists := pm.providers[channel]
	if !exists {
		return nil, fmt.Errorf("payment provider not found for channel: %s", channel)
	}

	return provider, nil
}

// HasProvider 检查是否存在指定的支付提供商
func (pm *PaymentManager) HasProvider(channel PaymentChannel) bool {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	_, exists := pm.providers[string(channel)]
	return exists
}

// SetConfig 设置支付配置
func (pm *PaymentManager) SetConfig(merchantID string, config interface{}) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if merchantID == "" {
		return fmt.Errorf("merchant ID cannot be empty")
	}

	pm.configs[merchantID] = config
	log.Info("Payment config set for merchant: %s", merchantID)
	return nil
}

// GetConfig 获取支付配置
func (pm *PaymentManager) GetConfig(merchantID string) (interface{}, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	config, exists := pm.configs[merchantID]
	if !exists {
		return nil, fmt.Errorf("payment config not found for merchant: %s", merchantID)
	}

	return config, nil
}

// CreateOrder 创建支付订单
func (pm *PaymentManager) CreateOrder(params *CreateOrderParams) (*CreateOrderResponse, error) {
	if params == nil {
		return &CreateOrderResponse{Success: false}, fmt.Errorf("params cannot be nil")
	}

	// 验证参数
	if err := pm.ValidateCreateOrderParams(params); err != nil {
		return &CreateOrderResponse{Success: false}, err
	}

	// 获取支付提供商
	provider, err := pm.GetProvider(params.Channel)
	if err != nil {
		return &CreateOrderResponse{Success: false}, err
	}

	// 创建订单
	response, err := provider.CreateOrder(params)
	if err != nil {
		log.Error("Failed to create order: %v", err)
		return &CreateOrderResponse{Success: false}, err
	}

	log.Info("Order created successfully: %s", params.OutTradeNo)
	return response, nil
}

// QueryOrder 查询支付订单
func (pm *PaymentManager) QueryOrder(params *QueryOrderParams) (*QueryOrderResponse, error) {
	if params == nil {
		return &QueryOrderResponse{Success: false}, fmt.Errorf("params cannot be nil")
	}

	// 验证参数
	if err := pm.ValidateQueryOrderParams(params); err != nil {
		return &QueryOrderResponse{Success: false}, err
	}

	// 获取支付提供商
	provider, err := pm.GetProvider(params.Channel)
	if err != nil {
		return &QueryOrderResponse{Success: false}, err
	}

	// 查询订单
	response, err := provider.QueryOrder(params)
	if err != nil {
		log.Error("Failed to query order: %v", err)
		return &QueryOrderResponse{Success: false}, err
	}

	return response, nil
}

// CreateRefund 创建退款
func (pm *PaymentManager) CreateRefund(params *CreateRefundParams) (*CreateRefundResponse, error) {
	if params == nil {
		return &CreateRefundResponse{Success: false}, fmt.Errorf("params cannot be nil")
	}

	// 验证参数
	if err := pm.ValidateCreateRefundParams(params); err != nil {
		return &CreateRefundResponse{Success: false, Error: err.Error()}, err
	}

	// 获取支付提供商
	provider, err := pm.GetProvider(params.Channel)
	if err != nil {
		return &CreateRefundResponse{Success: false, Error: err.Error()}, err
	}

	// 创建退款
	response, err := provider.CreateRefund(params)
	if err != nil {
		log.Error("Failed to create refund: %v", err)
		return &CreateRefundResponse{Success: false, Error: err.Error()}, err
	}

	log.Info("Refund created successfully: %s", params.OutRefundNo)
	return response, nil
}

// QueryRefund 查询退款状态
func (pm *PaymentManager) QueryRefund(params *QueryRefundParams) (*QueryRefundResponse, error) {
	if params == nil {
		return &QueryRefundResponse{Success: false}, fmt.Errorf("params cannot be nil")
	}

	// 验证参数
	if err := pm.ValidateQueryRefundParams(params); err != nil {
		return &QueryRefundResponse{Success: false}, err
	}

	// 获取支付提供商
	provider, err := pm.GetProvider(params.Channel)
	if err != nil {
		return &QueryRefundResponse{Success: false}, err
	}

	// 查询退款
	response, err := provider.QueryRefund(params)
	if err != nil {
		log.Error("Failed to query refund: %v", err)
		return &QueryRefundResponse{Success: false}, err
	}

	return response, nil
}

// HandleNotify 处理异步通知
func (pm *PaymentManager) HandleNotify(merchantID string, channel PaymentChannel, notifyData []byte) (*HandleNotifyResponse, error) {
	if merchantID == "" {
		return &HandleNotifyResponse{Success: false}, fmt.Errorf("merchant ID cannot be empty")
	}

	if len(notifyData) == 0 {
		return &HandleNotifyResponse{Success: false}, fmt.Errorf("request body is required")
	}

	// 获取支付提供商
	provider, err := pm.GetProvider(string(channel))
	if err != nil {
		return &HandleNotifyResponse{Success: false}, err
	}

	// 构建通知参数
	params := &HandleNotifyParams{
		MerchantID:  merchantID,
		Channel:     string(channel),
		RequestBody: notifyData,
		NotifyData:  make(map[string]interface{}),
	}

	// 处理通知
	response, err := provider.HandleNotify(params)
	if err != nil {
		log.Error("Failed to handle notify: %v", err)
		return &HandleNotifyResponse{Success: false}, err
	}

	log.Info("Notify handled successfully for merchant: %s", merchantID)
	return response, nil
}

// DownloadBill 下载对账单
func (pm *PaymentManager) DownloadBill(params *DownloadBillParams) (*DownloadBillResponse, error) {
	if params == nil {
		return &DownloadBillResponse{Success: false}, fmt.Errorf("params cannot be nil")
	}

	// 验证参数
	if err := pm.ValidateDownloadBillParams(params); err != nil {
		return &DownloadBillResponse{Success: false, Error: err.Error()}, err
	}

	// 获取支付提供商
	provider, err := pm.GetProvider(params.Channel)
	if err != nil {
		return &DownloadBillResponse{Success: false, Error: err.Error()}, err
	}

	// 下载对账单
	response, err := provider.DownloadBill(params)
	if err != nil {
		log.Error("Failed to download bill: %v", err)
		return &DownloadBillResponse{Success: false, Error: err.Error()}, err
	}

	log.Info("Bill downloaded successfully: %s", params.BillDate)
	return response, nil
}

// SetMerchantConfig 设置商户配置
func (pm *PaymentManager) SetMerchantConfig(merchantID string, channel PaymentChannel, config map[string]interface{}) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if merchantID == "" {
		return fmt.Errorf("merchant ID cannot be empty")
	}

	if len(config) == 0 {
		return fmt.Errorf("config cannot be empty")
	}

	key := fmt.Sprintf("%s_%s", merchantID, string(channel))
	pm.configs[key] = config
	log.Info("Merchant config set: %s", key)
	return nil
}

// GetMerchantConfig 获取商户配置
func (pm *PaymentManager) GetMerchantConfig(merchantID string, channel PaymentChannel) (map[string]interface{}, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	key := fmt.Sprintf("%s_%s", merchantID, string(channel))
	config, exists := pm.configs[key]
	if !exists {
		return nil, fmt.Errorf("merchant config not found: %s", key)
	}

	configMap, ok := config.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid config format for merchant: %s", key)
	}

	return configMap, nil
}

// Reconcile 对账
func (pm *PaymentManager) Reconcile(params *ReconcileParams) (*ReconcileResponse, error) {
	if params == nil {
		return &ReconcileResponse{Success: false}, fmt.Errorf("params cannot be nil")
	}

	// 验证参数
	if err := pm.ValidateReconcileParams(params); err != nil {
		return &ReconcileResponse{Success: false, Error: err.Error()}, err
	}

	// 获取支付提供商
	provider, err := pm.GetProvider(params.Channel)
	if err != nil {
		return &ReconcileResponse{Success: false, Error: err.Error()}, err
	}

	// 下载对账单
	billParams := &DownloadBillParams{
		MerchantNo: params.MerchantNo,
		Channel:    params.Channel,
		BillDate:   params.BillDate,
		BillType:   "ALL",
	}

	billResponse, err := provider.DownloadBill(billParams)
	if err != nil {
		log.Error("Failed to download bill for reconcile: %v", err)
		return &ReconcileResponse{Success: false, Error: err.Error()}, err
	}

	if !billResponse.Success {
		return &ReconcileResponse{
			Success: false,
			Error:   billResponse.Error,
		}, fmt.Errorf("download bill failed: %s", billResponse.Error)
	}

	// 这里可以添加具体的对账逻辑
	// 比如解析对账单数据，与本地订单数据进行比对等

	log.Info("Reconcile completed successfully: %s", params.BillDate)
	return &ReconcileResponse{
		Success: true,
		Message: "对账完成",
	}, nil
}

// ValidateCreateOrderParams 验证创建订单参数
func (pm *PaymentManager) ValidateCreateOrderParams(params *CreateOrderParams) error {
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
	if !isValidPaymentChannel(params.Channel) {
		return fmt.Errorf("invalid payment channel: %s", params.Channel)
	}

	// 验证交易类型
	if !isValidTradeType(params.TradeType) {
		return fmt.Errorf("invalid trade type: %s", params.TradeType)
	}

	return nil
}

// ValidateQueryOrderParams 验证查询订单参数
func (pm *PaymentManager) ValidateQueryOrderParams(params *QueryOrderParams) error {
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
	if !isValidPaymentChannel(params.Channel) {
		return fmt.Errorf("invalid payment channel: %s", params.Channel)
	}

	return nil
}

// ValidateQueryRefundParams 验证查询退款参数
func (pm *PaymentManager) ValidateQueryRefundParams(params *QueryRefundParams) error {
	if params.Channel == "" {
		return fmt.Errorf("channel is required")
	}

	if params.OutRefundNo == "" && params.RefundID == "" {
		return fmt.Errorf("either out_refund_no or refund_id is required")
	}

	// 验证支付渠道
	if !isValidPaymentChannel(params.Channel) {
		return fmt.Errorf("invalid payment channel: %s", params.Channel)
	}

	return nil
}

// ValidateDownloadBillParams 验证下载对账单参数
func (pm *PaymentManager) ValidateDownloadBillParams(params *DownloadBillParams) error {
	if params.Channel == "" {
		return fmt.Errorf("channel is required")
	}

	if params.BillDate == "" {
		return fmt.Errorf("bill_date is required")
	}

	// 验证支付渠道
	if !isValidPaymentChannel(params.Channel) {
		return fmt.Errorf("invalid payment channel: %s", params.Channel)
	}

	// 验证对账单类型（可选）
	if params.BillType != "" && !isValidBillType(params.BillType) {
		return fmt.Errorf("invalid bill type: %s", params.BillType)
	}

	return nil
}

// ValidateCreateRefundParams 验证创建退款参数
func (pm *PaymentManager) ValidateCreateRefundParams(params *CreateRefundParams) error {
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
	if !isValidPaymentChannel(params.Channel) {
		return fmt.Errorf("invalid payment channel: %s", params.Channel)
	}

	return nil
}

// ValidateReconcileParams 验证对账参数
func (pm *PaymentManager) ValidateReconcileParams(params *ReconcileParams) error {
	if params.Channel == "" {
		return fmt.Errorf("channel is required")
	}

	if params.BillDate == "" {
		return fmt.Errorf("bill_date is required")
	}

	// 验证支付渠道
	if !isValidPaymentChannel(params.Channel) {
		return fmt.Errorf("invalid payment channel: %s", params.Channel)
	}

	return nil
}

// isValidPaymentChannel 验证支付渠道是否有效
func isValidPaymentChannel(channel string) bool {
	validChannels := []string{
		string(ChannelAlipay),
		string(ChannelWechat),
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
func GetSupportedProviders() []PaymentChannel {
	return []PaymentChannel{
		PaymentChannel(ChannelAlipay),
		PaymentChannel(ChannelWechat),
	}
}

// isValidTradeType 验证交易类型是否有效
func isValidTradeType(tradeType string) bool {
	validTypes := []string{
		string(TradeTypeNative),
		string(TradeTypeJSAPI),
		string(TradeTypeApp),
		string(TradeTypeH5),
	}
	for _, validType := range validTypes {
		if tradeType == validType {
			return true
		}
	}
	return false
}
