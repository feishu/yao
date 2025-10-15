// Package payment 提供支付功能，支持支付宝和微信支付
// 通过 gopay SDK 实现统一的支付接口，支持多商户、多渠道、多交易类型
package payment

import (
	"time"
)

// PaymentProvider 支付提供商接口
type PaymentProvider interface {
	// CreateOrder 创建支付订单
	CreateOrder(params *CreateOrderParams) (*CreateOrderResponse, error)

	// QueryOrder 查询支付订单
	QueryOrder(params *QueryOrderParams) (*QueryOrderResponse, error)

	// CreateRefund 创建退款
	CreateRefund(params *CreateRefundParams) (*CreateRefundResponse, error)

	// QueryRefund 查询退款状态
	QueryRefund(params *QueryRefundParams) (*QueryRefundResponse, error)

	// HandleNotify 处理异步通知
	HandleNotify(params *HandleNotifyParams) (*HandleNotifyResponse, error)

	// DownloadBill 下载对账单
	DownloadBill(params *DownloadBillParams) (*DownloadBillResponse, error)
}

// PaymentChannel 支付渠道
type PaymentChannel string

const (
	ChannelAlipay PaymentChannel = "alipay" // 支付宝
	ChannelWechat PaymentChannel = "wechat" // 微信支付
)

// TradeType 交易类型
type TradeType string

const (
	TradeTypeJSAPI  TradeType = "jsapi"  // 公众号支付/小程序支付
	TradeTypeNative TradeType = "native" // 扫码支付
	TradeTypeApp    TradeType = "app"    // APP支付
	TradeTypeH5     TradeType = "h5"     // H5支付
	TradeTypeWAP    TradeType = "wap"    // WAP支付（支付宝）
)

// OrderStatus 订单状态
type OrderStatus string

const (
	OrderStatusPending OrderStatus = "pending" // 待支付
	OrderStatusPaid    OrderStatus = "paid"    // 已支付
	OrderStatusClosed  OrderStatus = "closed"  // 已关闭
	OrderStatusRefund  OrderStatus = "refund"  // 已退款
)

// RefundStatus 退款状态
type RefundStatus string

const (
	RefundStatusPending    RefundStatus = "pending"    // 退款处理中
	RefundStatusProcessing RefundStatus = "processing" // 退款处理中
	RefundStatusSuccess    RefundStatus = "success"    // 退款成功
	RefundStatusFailed     RefundStatus = "failed"     // 退款失败
	RefundStatusClosed     RefundStatus = "closed"     // 退款已关闭
	RefundStatusAbnormal   RefundStatus = "abnormal"   // 退款异常
)

// CreateOrderParams 创建订单参数
type CreateOrderParams struct {
	// 基础参数
	MerchantNo string `json:"merchant_no" validate:"required"`  // 商户编号
	Channel    string `json:"channel" validate:"required"`      // 支付渠道（alipay/wechat）
	TradeType  string `json:"trade_type" validate:"required"`   // 交易类型（jsapi/native/app/h5/wap）
	Amount     int64  `json:"amount" validate:"required,min=1"` // 支付金额（分）
	Subject    string `json:"subject" validate:"required"`      // 订单标题
	OutTradeNo string `json:"out_trade_no" validate:"required"` // 商户订单号
	NotifyURL  string `json:"notify_url" validate:"required"`   // 异步通知地址
	ReturnURL  string `json:"return_url"`                       // 支付成功跳转地址
	Body       string `json:"body"`                             // 订单描述
	Currency   string `json:"currency"`                         // 货币类型，默认CNY
	ExpireTime string `json:"expire_time"`                      // 订单过期时间

	// 微信支付特有参数
	WechatParams *WechatPayParams `json:"wechat_params,omitempty"`

	// 支付宝特有参数
	AlipayParams *AlipayPayParams `json:"alipay_params,omitempty"`

	// 扩展参数
	ExtendParams map[string]interface{} `json:"extend_params,omitempty"`
}

// WechatPayParams 微信支付特有参数
type WechatPayParams struct {
	AppID     string `json:"appid" validate:"required"` // 微信应用ID
	SceneInfo string `json:"scene_info"`                // 场景信息（H5支付必需）
	Attach    string `json:"attach"`                    // 附加数据
	GoodsTag  string `json:"goods_tag"`                 // 商品标记
	LimitPay  string `json:"limit_pay"`                 // 指定支付方式
	Detail    string `json:"detail"`                    // 商品详情
	StoreInfo string `json:"store_info"`                // 门店信息
}

// AlipayPayParams 支付宝特有参数
type AlipayPayParams struct {
	BuyerID            string `json:"buyer_id"`             // 买家支付宝用户ID
	BuyerLogonID       string `json:"buyer_logon_id"`       // 买家支付宝账号
	ProductCode        string `json:"product_code"`         // 产品码
	QuitURL            string `json:"quit_url"`             // 用户付款中途退出返回商户网站的地址
	TimeoutExpress     string `json:"timeout_express"`      // 订单超时时间
	EnablePayChannels  string `json:"enable_pay_channels"`  // 可用渠道
	DisablePayChannels string `json:"disable_pay_channels"` // 禁用渠道
	AuthToken          string `json:"auth_token"`           // 针对用户授权接口调用凭证
	QRPayMode          string `json:"qr_pay_mode"`          // 扫码支付的方式
	QRCodeWidth        string `json:"qrcode_width"`         // 商户自定义二维码宽度
	StoreID            string `json:"store_id"`             // 商户门店编号
	ExtendParams       string `json:"extend_params"`        // 业务扩展参数
}

// CreateOrderResponse 创建订单响应
type CreateOrderResponse struct {
	Success    bool                   `json:"success"`
	OrderID    string                 `json:"order_id"`     // 系统订单ID
	OutTradeNo string                 `json:"out_trade_no"` // 商户订单号
	PayInfo    map[string]interface{} `json:"pay_info"`     // 支付信息（如预支付交易会话标识等）
	QRCode     string                 `json:"qr_code"`      // 二维码内容（Native支付）
	PayURL     string                 `json:"pay_url"`      // 支付链接（H5/WAP支付）
	Message    string                 `json:"message"`      // 响应消息
	Error      string                 `json:"error"`        // 错误信息
}

// QueryOrderParams 查询订单参数
type QueryOrderParams struct {
	MerchantNo    string `json:"merchant_no" validate:"required"` // 商户编号
	OutTradeNo    string `json:"out_trade_no"`                    // 商户订单号
	TransactionID string `json:"transaction_id"`                  // 支付平台交易号
	Channel       string `json:"channel"`                         // 支付渠道（alipay/wechat）
}

// QueryOrderResponse 查询订单响应
type QueryOrderResponse struct {
	Success       bool    `json:"success"`
	OrderID       string  `json:"order_id"`       // 系统订单ID
	OutTradeNo    string  `json:"out_trade_no"`   // 商户订单号
	TransactionID string  `json:"transaction_id"` // 支付平台交易号
	Channel       string  `json:"channel"`        // 支付渠道
	TradeType     string  `json:"trade_type"`     // 交易类型
	Amount        int64   `json:"amount"`         // 支付金额（分）
	Status        string  `json:"status"`         // 订单状态
	PaidAt        *string `json:"paid_at"`        // 支付时间
	CreatedAt     string  `json:"created_at"`     // 创建时间
	Message       string  `json:"message"`        // 响应消息
	Error         string  `json:"error"`          // 错误信息
}

// CreateRefundParams 创建退款参数
type CreateRefundParams struct {
	MerchantNo    string `json:"merchant_no" validate:"required"`   // 商户编号
	OutTradeNo    string `json:"out_trade_no"`                      // 商户订单号
	TransactionID string `json:"transaction_id"`                    // 支付平台交易号
	OutRefundNo   string `json:"out_refund_no" validate:"required"` // 商户退款单号
	RefundAmount  int64  `json:"refund_amount" validate:"required"` // 退款金额（分）
	TotalAmount   int64  `json:"total_amount" validate:"required"`  // 订单总金额（分）
	Reason        string `json:"reason"`                            // 退款原因
	RefundReason  string `json:"refund_reason"`                     // 退款原因（兼容性别名）
	NotifyURL     string `json:"notify_url"`                        // 退款异步通知地址
	Channel       string `json:"channel"`                           // 支付渠道
}

// CreateRefundResponse 创建退款响应
type CreateRefundResponse struct {
	Success      bool   `json:"success"`
	RefundID     string `json:"refund_id"`     // 系统退款ID
	OutRefundNo  string `json:"out_refund_no"` // 商户退款单号
	RefundNo     string `json:"refund_no"`     // 支付平台退款单号
	RefundAmount int64  `json:"refund_amount"` // 退款金额（分）
	Status       string `json:"status"`        // 退款状态
	Message      string `json:"message"`       // 响应消息
	Error        string `json:"error"`         // 错误信息
}

// QueryRefundParams 查询退款参数
type QueryRefundParams struct {
	MerchantNo  string `json:"merchant_no" validate:"required"` // 商户编号
	OutRefundNo string `json:"out_refund_no"`                   // 商户退款单号
	RefundID    string `json:"refund_id"`                       // 支付平台退款ID
	Channel     string `json:"channel"`                         // 支付渠道
}

// QueryRefundResponse 查询退款响应
type QueryRefundResponse struct {
	Success      bool    `json:"success"`
	RefundID     string  `json:"refund_id"`     // 系统退款ID
	OutRefundNo  string  `json:"out_refund_no"` // 商户退款单号
	RefundNo     string  `json:"refund_no"`     // 支付平台退款单号
	RefundAmount int64   `json:"refund_amount"` // 退款金额（分）
	Status       string  `json:"status"`        // 退款状态
	RefundAt     *string `json:"refund_at"`     // 退款时间
	CreatedAt    string  `json:"created_at"`    // 创建时间
	Message      string  `json:"message"`       // 响应消息
	Error        string  `json:"error"`         // 错误信息
}

// HandleNotifyParams 异步通知参数
type HandleNotifyParams struct {
	MerchantID  string                 `json:"merchant_id"`  // 商户ID
	Channel     string                 `json:"channel"`      // 支付渠道
	RequestBody []byte                 `json:"request_body"` // 请求体数据
	NotifyData  map[string]interface{} `json:"notify_data"`  // 通知数据
}

// HandleNotifyResponse 异步通知响应
type HandleNotifyResponse struct {
	Success       bool                   `json:"success"`
	OutTradeNo    string                 `json:"out_trade_no"`   // 商户订单号
	TransactionID string                 `json:"transaction_id"` // 支付平台交易号
	Amount        int64                  `json:"amount"`         // 支付金额（分）
	Status        string                 `json:"status"`         // 订单状态
	PaidAt        *time.Time             `json:"paid_at"`        // 支付时间
	NotifyData    map[string]interface{} `json:"notify_data"`    // 原始通知数据
	Message       string                 `json:"message"`        // 响应消息
	Error         string                 `json:"error"`          // 错误信息
}

// NotifyResponse 异步通知响应（兼容性别名）
type NotifyResponse = HandleNotifyResponse

// DownloadBillParams 下载对账单参数
type DownloadBillParams struct {
	MerchantNo string `json:"merchant_no" validate:"required"` // 商户编号
	Channel    string `json:"channel" validate:"required"`     // 支付渠道
	BillDate   string `json:"bill_date" validate:"required"`   // 对账单日期（YYYY-MM-DD）
	BillType   string `json:"bill_type"`                       // 对账单类型
}

// DownloadBillResponse 下载对账单响应
type DownloadBillResponse struct {
	Success  bool   `json:"success"`
	BillData string `json:"bill_data"` // 对账单数据
	Message  string `json:"message"`   // 响应消息
	Error    string `json:"error"`     // 错误信息
}

// ReconcileParams 对账参数
type ReconcileParams struct {
	MerchantID string `json:"merchant_id" validate:"required"` // 商户ID
	MerchantNo string `json:"merchant_no" validate:"required"` // 商户编号
	Channel    string `json:"channel" validate:"required"`     // 支付渠道
	BillDate   string `json:"bill_date" validate:"required"`   // 对账日期（YYYY-MM-DD）
}

// ReconcileResponse 对账响应
type ReconcileResponse struct {
	Success       bool                     `json:"success"`
	TotalCount    int                      `json:"total_count"`    // 总交易笔数
	SuccessCount  int                      `json:"success_count"`  // 成功交易笔数
	FailedCount   int                      `json:"failed_count"`   // 失败交易笔数
	TotalAmount   int64                    `json:"total_amount"`   // 总交易金额（分）
	SuccessAmount int64                    `json:"success_amount"` // 成功交易金额（分）
	DiffRecords   []map[string]interface{} `json:"diff_records"`   // 差异记录
	Message       string                   `json:"message"`        // 响应消息
	Error         string                   `json:"error"`          // 错误信息
}

// MerchantConfig 商户配置
type MerchantConfig struct {
	MerchantNo string                 `json:"merchant_no"` // 商户编号
	Channel    PaymentChannel         `json:"channel"`     // 支付渠道
	Config     map[string]interface{} `json:"config"`      // 配置数据
	IsActive   bool                   `json:"is_active"`   // 是否启用
	CreatedAt  time.Time              `json:"created_at"`  // 创建时间
	UpdatedAt  time.Time              `json:"updated_at"`  // 更新时间
}

// PaymentOrder 支付订单
type PaymentOrder struct {
	ID            string                 `json:"id"`             // 系统订单ID
	MerchantNo    string                 `json:"merchant_no"`    // 商户编号
	OutTradeNo    string                 `json:"out_trade_no"`   // 商户订单号
	TransactionID string                 `json:"transaction_id"` // 支付平台交易号
	Channel       PaymentChannel         `json:"channel"`        // 支付渠道
	TradeType     TradeType              `json:"trade_type"`     // 交易类型
	Amount        int64                  `json:"amount"`         // 支付金额（分）
	Currency      string                 `json:"currency"`       // 货币类型
	Subject       string                 `json:"subject"`        // 订单标题
	Body          string                 `json:"body"`           // 订单描述
	Status        OrderStatus            `json:"status"`         // 订单状态
	NotifyURL     string                 `json:"notify_url"`     // 异步通知地址
	ReturnURL     string                 `json:"return_url"`     // 支付成功跳转地址
	WechatParams  map[string]interface{} `json:"wechat_params"`  // 微信支付参数
	AlipayParams  map[string]interface{} `json:"alipay_params"`  // 支付宝支付参数
	ExtendParams  map[string]interface{} `json:"extend_params"`  // 扩展参数
	ExpireTime    *time.Time             `json:"expire_time"`    // 订单过期时间
	PaidAt        *time.Time             `json:"paid_at"`        // 支付时间
	CreatedAt     time.Time              `json:"created_at"`     // 创建时间
	UpdatedAt     time.Time              `json:"updated_at"`     // 更新时间
}

// PaymentRefund 支付退款
type PaymentRefund struct {
	ID           string                 `json:"id"`            // 系统退款ID
	OrderID      string                 `json:"order_id"`      // 关联订单ID
	MerchantNo   string                 `json:"merchant_no"`   // 商户编号
	OutRefundNo  string                 `json:"out_refund_no"` // 商户退款单号
	RefundNo     string                 `json:"refund_no"`     // 支付平台退款单号
	Channel      PaymentChannel         `json:"channel"`       // 支付渠道
	RefundAmount int64                  `json:"refund_amount"` // 退款金额（分）
	TotalAmount  int64                  `json:"total_amount"`  // 订单总金额（分）
	RefundReason string                 `json:"refund_reason"` // 退款原因
	Status       RefundStatus           `json:"status"`        // 退款状态
	NotifyURL    string                 `json:"notify_url"`    // 退款异步通知地址
	ExtendParams map[string]interface{} `json:"extend_params"` // 扩展参数
	RefundAt     *time.Time             `json:"refund_at"`     // 退款时间
	CreatedAt    time.Time              `json:"created_at"`    // 创建时间
	UpdatedAt    time.Time              `json:"updated_at"`    // 更新时间
}

// NotifyLog 异步通知日志
type NotifyLog struct {
	ID            string                 `json:"id"`             // 日志ID
	MerchantNo    string                 `json:"merchant_no"`    // 商户编号
	OutTradeNo    string                 `json:"out_trade_no"`   // 商户订单号
	TransactionID string                 `json:"transaction_id"` // 支付平台交易号
	Channel       PaymentChannel         `json:"channel"`        // 支付渠道
	NotifyType    string                 `json:"notify_type"`    // 通知类型（payment/refund）
	NotifyData    map[string]interface{} `json:"notify_data"`    // 通知数据
	ProcessResult string                 `json:"process_result"` // 处理结果
	RetryCount    int                    `json:"retry_count"`    // 重试次数
	CreatedAt     time.Time              `json:"created_at"`     // 创建时间
	UpdatedAt     time.Time              `json:"updated_at"`     // 更新时间
}
