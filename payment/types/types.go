// Package types 提供 payment 模块的共享类型定义
// 这个包被 payment 和 providers 包共同使用，避免循环依赖
package types

import "net/http"

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

// CreateOrderParams 创建订单参数（统一）
type CreateOrderParams struct {
	// 基础字段
	OutTradeNo string    `json:"out_trade_no" validate:"required"` // 商户订单号
	Amount     int64     `json:"amount" validate:"required,min=1"` // 金额（分）
	Subject    string    `json:"subject" validate:"required"`      // 订单标题
	Body       string    `json:"body"`                             // 订单描述
	TradeType  TradeType `json:"trade_type" validate:"required"`   // 交易类型
	NotifyURL  string    `json:"notify_url" validate:"required"`   // 异步通知地址
	ReturnURL  string    `json:"return_url"`                       // 同步跳转地址
	ExpireTime string    `json:"expire_time"`                      // 过期时间
	Currency   string    `json:"currency"`                         // 货币类型（默认CNY）

	// 系统字段
	MerchantNo string         `json:"merchant_no" validate:"required"` // 商户编号
	Channel    PaymentChannel `json:"channel" validate:"required"`     // 支付渠道

	// 渠道特有参数
	AlipayParams *AlipayOrderParams `json:"alipay_params,omitempty"` // 支付宝特有参数
	WechatParams *WechatOrderParams `json:"wechat_params,omitempty"` // 微信特有参数

	// 扩展参数（两个别名，保持兼容）
	Extra        map[string]interface{} `json:"extra,omitempty"`         // 扩展参数
	ExtendParams map[string]interface{} `json:"extend_params,omitempty"` // 扩展参数（别名）
}

// AlipayOrderParams 支付宝订单参数（统一）
type AlipayOrderParams struct {
	BuyerID            string `json:"buyer_id"`             // 买家支付宝用户ID
	BuyerLogonID       string `json:"buyer_logon_id"`       // 买家支付宝账号
	ProductCode        string `json:"product_code"`         // 产品码
	QuitURL            string `json:"quit_url"`             // 退出URL
	TimeoutExpress     string `json:"timeout_express"`      // 订单超时时间
	EnablePayChannels  string `json:"enable_pay_channels"`  // 可用渠道
	DisablePayChannels string `json:"disable_pay_channels"` // 禁用渠道
	AuthToken          string `json:"auth_token"`           // 授权令牌
	QRPayMode          string `json:"qr_pay_mode"`          // 扫码方式
	QRCodeWidth        string `json:"qrcode_width"`         // 二维码宽度
	StoreID            string `json:"store_id"`             // 门店编号
	ExtendParams       string `json:"extend_params"`        // 业务扩展参数
}

// WechatOrderParams 微信订单参数（统一）
type WechatOrderParams struct {
	OpenID    string                 `json:"openid"`     // 用户OpenID（JSAPI必填）
	SceneInfo map[string]interface{} `json:"scene_info"` // 场景信息（H5必需）
	Detail    string                 `json:"detail"`     // 商品详情
	Attach    string                 `json:"attach"`     // 附加数据
	GoodsTag  string                 `json:"goods_tag"`  // 商品标记
	StoreInfo string                 `json:"store_info"` // 门店信息
}

// CreateOrderResponse 创建订单响应
type CreateOrderResponse struct {
	Success    bool                   `json:"success"`      // 是否成功
	OrderID    string                 `json:"order_id"`     // 订单ID
	OutTradeNo string                 `json:"out_trade_no"` // 商户订单号
	PayInfo    map[string]interface{} `json:"pay_info"`     // 支付信息
	QRCode     string                 `json:"qr_code"`      // 二维码
	PayURL     string                 `json:"pay_url"`      // 支付链接
	Message    string                 `json:"message"`      // 消息
	Error      string                 `json:"error"`        // 错误信息
}

// QueryOrderParams 查询订单参数（统一）
type QueryOrderParams struct {
	OutTradeNo    string         `json:"out_trade_no"`                    // 商户订单号
	TradeNo       string         `json:"trade_no"`                        // 平台订单号（兼容别名）
	TransactionID string         `json:"transaction_id"`                  // 支付平台交易号
	MerchantNo    string         `json:"merchant_no" validate:"required"` // 商户编号
	Channel       PaymentChannel `json:"channel" validate:"required"`     // 支付渠道
}

// QueryOrderResponse 查询订单响应
type QueryOrderResponse struct {
	Success       bool           `json:"success"`        // 是否成功
	OrderID       string         `json:"order_id"`       // 订单ID
	OutTradeNo    string         `json:"out_trade_no"`   // 商户订单号
	TransactionID string         `json:"transaction_id"` // 支付平台交易号
	TradeNo       string         `json:"trade_no"`       // 平台订单号（兼容性别名）
	Status        OrderStatus    `json:"status"`         // 订单状态
	Amount        int64          `json:"amount"`         // 金额（分）
	PaidAmount    int64          `json:"paid_amount"`    // 实付金额（分）
	PayTime       string         `json:"pay_time"`       // 支付时间
	Channel       PaymentChannel `json:"channel"`        // 支付渠道
	TradeType     TradeType      `json:"trade_type"`     // 交易类型
	PaidAt        string         `json:"paid_at"`        // 支付时间（兼容性别名）
	CreatedAt     string         `json:"created_at"`     // 创建时间
	Message       string         `json:"message"`        // 消息
	Error         string         `json:"error"`          // 错误信息
}

// CreateRefundParams 创建退款参数（统一）
type CreateRefundParams struct {
	OutTradeNo    string         `json:"out_trade_no"`                      // 商户订单号
	TransactionID string         `json:"transaction_id"`                    // 支付平台交易号
	OutRefundNo   string         `json:"out_refund_no" validate:"required"` // 商户退款号
	RefundAmount  int64          `json:"refund_amount" validate:"required"` // 退款金额（分）
	TotalAmount   int64          `json:"total_amount" validate:"required"`  // 订单总金额（分）
	Reason        string         `json:"reason"`                            // 退款原因
	RefundReason  string         `json:"refund_reason"`                     // 退款原因（兼容性别名）
	NotifyURL     string         `json:"notify_url"`                        // 异步通知地址
	MerchantNo    string         `json:"merchant_no" validate:"required"`   // 商户编号
	Channel       PaymentChannel `json:"channel" validate:"required"`       // 支付渠道
}

// CreateRefundResponse 创建退款响应
type CreateRefundResponse struct {
	Success      bool         `json:"success"`       // 是否成功
	OutRefundNo  string       `json:"out_refund_no"` // 商户退款号
	RefundID     string       `json:"refund_id"`     // 退款ID
	RefundAmount int64        `json:"refund_amount"` // 退款金额（分）
	Status       RefundStatus `json:"status"`        // 退款状态
	Message      string       `json:"message"`       // 消息
	Error        string       `json:"error"`         // 错误信息
}

// QueryRefundParams 查询退款参数（统一）
type QueryRefundParams struct {
	OutTradeNo  string         `json:"out_trade_no"`                    // 商户订单号
	OutRefundNo string         `json:"out_refund_no"`                   // 商户退款号
	RefundID    string         `json:"refund_id"`                       // 支付平台退款ID
	MerchantNo  string         `json:"merchant_no" validate:"required"` // 商户编号
	Channel     PaymentChannel `json:"channel" validate:"required"`     // 支付渠道
}

// QueryRefundResponse 查询退款响应（统一）
type QueryRefundResponse struct {
	Success      bool         `json:"success"`       // 是否成功
	RefundID     string       `json:"refund_id"`     // 系统退款ID
	OutRefundNo  string       `json:"out_refund_no"` // 商户退款号
	RefundNo     string       `json:"refund_no"`     // 支付平台退款号
	RefundAmount int64        `json:"refund_amount"` // 退款金额（分）
	Status       RefundStatus `json:"status"`        // 退款状态
	RefundAt     string       `json:"refund_at"`     // 退款时间
	RefundTime   string       `json:"refund_time"`   // 退款时间（兼容性别名）
	CreatedAt    string       `json:"created_at"`    // 创建时间
	Message      string       `json:"message"`       // 消息
	Error        string       `json:"error"`         // 错误信息
}

// HandleNotifyParams 处理通知参数
type HandleNotifyParams struct {
	Channel     PaymentChannel         `json:"channel"`      // 支付渠道
	Request     *http.Request          `json:"-"`            // HTTP请求对象
	RequestBody []byte                 `json:"request_body"` // 请求体数据
	NotifyData  map[string]interface{} `json:"notify_data"`  // 通知数据
	MerchantNo  string                 `json:"merchant_no"`  // 商户编号
}

// HandleNotifyResponse 处理通知响应（统一）
type HandleNotifyResponse struct {
	Success       bool                   `json:"success"`        // 是否成功
	OrderID       string                 `json:"order_id"`       // 订单ID
	OutTradeNo    string                 `json:"out_trade_no"`   // 商户订单号
	TradeNo       string                 `json:"trade_no"`       // 平台订单号（兼容性别名）
	TransactionID string                 `json:"transaction_id"` // 支付平台交易号
	Status        OrderStatus            `json:"status"`         // 订单状态
	Amount        int64                  `json:"amount"`         // 金额（分）
	PayTime       string                 `json:"pay_time"`       // 支付时间
	NotifyData    map[string]interface{} `json:"notify_data"`    // 通知数据
	Extra         map[string]interface{} `json:"extra"`          // 扩展信息
	Message       string                 `json:"message"`        // 消息
	Error         string                 `json:"error"`          // 错误信息
}

// DownloadBillParams 下载对账单参数
type DownloadBillParams struct {
	BillDate   string         `json:"bill_date"`   // 对账单日期
	BillType   string         `json:"bill_type"`   // 对账单类型
	MerchantNo string         `json:"merchant_no"` // 商户号
	Channel    PaymentChannel `json:"channel"`     // 支付渠道
}

// DownloadBillResponse 下载对账单响应
type DownloadBillResponse struct {
	Success  bool   `json:"success"`   // 是否成功
	BillData string `json:"bill_data"` // 对账单数据
	Message  string `json:"message"`   // 消息
	Error    string `json:"error"`     // 错误信息
}

// ReconcileParams 对账参数
type ReconcileParams struct {
	Channel    PaymentChannel `json:"channel" validate:"required"`     // 支付渠道
	MerchantID string         `json:"merchant_id" validate:"required"` // 商户ID
	MerchantNo string         `json:"merchant_no" validate:"required"` // 商户编号
	BillDate   string         `json:"bill_date" validate:"required"`   // 对账日期（YYYY-MM-DD）
	BillType   string         `json:"bill_type"`                       // 对账单类型
}

// ReconcileResponse 对账响应
type ReconcileResponse struct {
	Success       bool                     `json:"success"`        // 是否成功
	TotalCount    int                      `json:"total_count"`    // 总交易笔数
	SuccessCount  int                      `json:"success_count"`  // 成功交易笔数
	FailedCount   int                      `json:"failed_count"`   // 失败交易笔数
	MatchCount    int                      `json:"match_count"`    // 匹配记录数
	DiffCount     int                      `json:"diff_count"`     // 差异记录数
	TotalAmount   int64                    `json:"total_amount"`   // 总交易金额（分）
	SuccessAmount int64                    `json:"success_amount"` // 成功交易金额（分）
	DiffRecords   []map[string]interface{} `json:"diff_records"`   // 差异记录
	Message       string                   `json:"message"`        // 响应消息
	Error         string                   `json:"error"`          // 错误信息
}

// PaymentProvider 支付提供商接口
type PaymentProvider interface {
	// CreateOrder 创建支付订单
	CreateOrder(params *CreateOrderParams) (*CreateOrderResponse, error)

	// QueryOrder 查询订单状态
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
