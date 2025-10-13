// Package payment 类型定义
package payment

import (
	"time"
)

// PaymentError 支付错误类型
type PaymentError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error 实现 error 接口
func (e *PaymentError) Error() string {
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
	return e.Message
}

// NewPaymentError 创建支付错误
func NewPaymentError(code, message, details string) *PaymentError {
	return &PaymentError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// 预定义错误代码
const (
	// ErrCodeInvalidRequest 无效请求
	ErrCodeInvalidRequest = "INVALID_REQUEST"
	// ErrCodeInvalidConfig 无效配置
	ErrCodeInvalidConfig = "INVALID_CONFIG"
	// ErrCodeProviderNotFound 支付提供商未找到
	ErrCodeProviderNotFound = "PROVIDER_NOT_FOUND"
	// ErrCodePaymentFailed 支付失败
	ErrCodePaymentFailed = "PAYMENT_FAILED"
	// ErrCodeRefundFailed 退款失败
	ErrCodeRefundFailed = "REFUND_FAILED"
	// ErrCodeQueryFailed 查询失败
	ErrCodeQueryFailed = "QUERY_FAILED"
	// ErrCodeSignatureInvalid 签名无效
	ErrCodeSignatureInvalid = "SIGNATURE_INVALID"
	// ErrCodeAmountInvalid 金额无效
	ErrCodeAmountInvalid = "AMOUNT_INVALID"
	// ErrCodeOrderNotFound 订单未找到
	ErrCodeOrderNotFound = "ORDER_NOT_FOUND"
	// ErrCodeNetworkError 网络错误
	ErrCodeNetworkError = "NETWORK_ERROR"
)

// PaymentMethod 支付方式
type PaymentMethod string

const (
	// MethodWechatNative 微信扫码支付
	MethodWechatNative PaymentMethod = "wechat_native"
	// MethodWechatJSAPI 微信公众号支付
	MethodWechatJSAPI PaymentMethod = "wechat_jsapi"
	// MethodWechatApp 微信APP支付
	MethodWechatApp PaymentMethod = "wechat_app"
	// MethodWechatH5 微信H5支付
	MethodWechatH5 PaymentMethod = "wechat_h5"
	// MethodWechatMiniProgram 微信小程序支付
	MethodWechatMiniProgram PaymentMethod = "wechat_miniprogram"

	// MethodAlipayPage 支付宝网页支付
	MethodAlipayPage PaymentMethod = "alipay_page"
	// MethodAlipayWap 支付宝手机网站支付
	MethodAlipayWap PaymentMethod = "alipay_wap"
	// MethodAlipayApp 支付宝APP支付
	MethodAlipayApp PaymentMethod = "alipay_app"
	// MethodAlipayQR 支付宝扫码支付
	MethodAlipayQR PaymentMethod = "alipay_qr"

	// MethodPayPalOrder PayPal订单支付
	MethodPayPalOrder PaymentMethod = "paypal_order"
	// MethodPayPalSubscription PayPal订阅支付
	MethodPayPalSubscription PaymentMethod = "paypal_subscription"
)

// Currency 货币类型
type Currency string

const (
	// CurrencyCNY 人民币
	CurrencyCNY Currency = "CNY"
	// CurrencyUSD 美元
	CurrencyUSD Currency = "USD"
	// CurrencyEUR 欧元
	CurrencyEUR Currency = "EUR"
	// CurrencyGBP 英镑
	CurrencyGBP Currency = "GBP"
	// CurrencyJPY 日元
	CurrencyJPY Currency = "JPY"
	// CurrencyHKD 港币
	CurrencyHKD Currency = "HKD"
)

// NotifyEvent 支付通知事件
type NotifyEvent struct {
	Provider    PaymentProvider        `json:"provider"`
	EventType   string                 `json:"event_type"`
	OrderID     string                 `json:"order_id"`
	PaymentID   string                 `json:"payment_id"`
	Status      PaymentStatus          `json:"status"`
	Amount      int64                  `json:"amount"`
	Currency    string                 `json:"currency"`
	Timestamp   time.Time              `json:"timestamp"`
	RawData     map[string]interface{} `json:"raw_data"`
	Signature   string                 `json:"signature"`
	IsValid     bool                   `json:"is_valid"`
}

// PaymentStats 支付统计
type PaymentStats struct {
	TotalOrders    int64   `json:"total_orders"`
	SuccessOrders  int64   `json:"success_orders"`
	FailedOrders   int64   `json:"failed_orders"`
	PendingOrders  int64   `json:"pending_orders"`
	TotalAmount    int64   `json:"total_amount"`
	SuccessAmount  int64   `json:"success_amount"`
	SuccessRate    float64 `json:"success_rate"`
	AverageAmount  float64 `json:"average_amount"`
}

// ProviderStats 支付提供商统计
type ProviderStats struct {
	Provider PaymentProvider `json:"provider"`
	Stats    PaymentStats    `json:"stats"`
}

// PaymentWebhook 支付回调配置
type PaymentWebhook struct {
	URL    string            `json:"url"`
	Secret string            `json:"secret"`
	Events []string          `json:"events"`
	Headers map[string]string `json:"headers,omitempty"`
}

// PaymentLimits 支付限制
type PaymentLimits struct {
	MinAmount     int64 `json:"min_amount"`      // 最小支付金额（分）
	MaxAmount     int64 `json:"max_amount"`      // 最大支付金额（分）
	DailyLimit    int64 `json:"daily_limit"`     // 日限额（分）
	MonthlyLimit  int64 `json:"monthly_limit"`   // 月限额（分）
	MaxRetries    int   `json:"max_retries"`     // 最大重试次数
	RetryInterval int   `json:"retry_interval"`  // 重试间隔（秒）
}

// PaymentMetadata 支付元数据
type PaymentMetadata struct {
	UserID       string                 `json:"user_id,omitempty"`
	ProductID    string                 `json:"product_id,omitempty"`
	ProductName  string                 `json:"product_name,omitempty"`
	Quantity     int                    `json:"quantity,omitempty"`
	Discount     int64                  `json:"discount,omitempty"`
	Tax          int64                  `json:"tax,omitempty"`
	ShippingFee  int64                  `json:"shipping_fee,omitempty"`
	Custom       map[string]interface{} `json:"custom,omitempty"`
}

// PaymentHistory 支付历史记录
type PaymentHistory struct {
	ID          string          `json:"id"`
	OrderID     string          `json:"order_id"`
	PaymentID   string          `json:"payment_id"`
	Provider    PaymentProvider `json:"provider"`
	Method      PaymentMethod   `json:"method"`
	Status      PaymentStatus   `json:"status"`
	Amount      int64           `json:"amount"`
	Currency    Currency        `json:"currency"`
	Subject     string          `json:"subject"`
	Description string          `json:"description"`
	Metadata    PaymentMetadata `json:"metadata"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
}

// RefundHistory 退款历史记录
type RefundHistory struct {
	ID          string          `json:"id"`
	RefundID    string          `json:"refund_id"`
	OrderID     string          `json:"order_id"`
	PaymentID   string          `json:"payment_id"`
	Provider    PaymentProvider `json:"provider"`
	Status      PaymentStatus   `json:"status"`
	Amount      int64           `json:"amount"`
	Reason      string          `json:"reason"`
	CreatedAt   time.Time       `json:"created_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
}

// PaymentFilter 支付查询过滤器
type PaymentFilter struct {
	Provider    PaymentProvider `json:"provider,omitempty"`
	Status      PaymentStatus   `json:"status,omitempty"`
	Method      PaymentMethod   `json:"method,omitempty"`
	Currency    Currency        `json:"currency,omitempty"`
	MinAmount   int64           `json:"min_amount,omitempty"`
	MaxAmount   int64           `json:"max_amount,omitempty"`
	StartTime   *time.Time      `json:"start_time,omitempty"`
	EndTime     *time.Time      `json:"end_time,omitempty"`
	UserID      string          `json:"user_id,omitempty"`
	ProductID   string          `json:"product_id,omitempty"`
	Limit       int             `json:"limit,omitempty"`
	Offset      int             `json:"offset,omitempty"`
	OrderBy     string          `json:"order_by,omitempty"`
	OrderDir    string          `json:"order_dir,omitempty"`
}

// PaymentListResponse 支付列表响应
type PaymentListResponse struct {
	Payments []PaymentHistory `json:"payments"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
	HasMore  bool             `json:"has_more"`
}

// RefundListResponse 退款列表响应
type RefundListResponse struct {
	Refunds  []RefundHistory `json:"refunds"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
	HasMore  bool            `json:"has_more"`
}

// HealthCheck 健康检查响应
type HealthCheck struct {
	Status    string                        `json:"status"`
	Timestamp time.Time                     `json:"timestamp"`
	Providers map[PaymentProvider]bool      `json:"providers"`
	Errors    []string                      `json:"errors,omitempty"`
	Metrics   map[string]interface{}        `json:"metrics,omitempty"`
}

// ConfigValidation 配置验证结果
type ConfigValidation struct {
	Provider PaymentProvider `json:"provider"`
	Valid    bool            `json:"valid"`
	Errors   []string        `json:"errors,omitempty"`
	Warnings []string        `json:"warnings,omitempty"`
}