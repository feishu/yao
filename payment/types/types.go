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

// CreateOrderParams 创建订单参数
type CreateOrderParams struct {
	OutTradeNo   string                 `json:"out_trade_no"`  // 商户订单号
	Amount       int64                  `json:"amount"`        // 金额（分）
	Subject      string                 `json:"subject"`       // 订单标题
	Body         string                 `json:"body"`          // 订单描述
	TradeType    TradeType              `json:"trade_type"`    // 交易类型
	NotifyURL    string                 `json:"notify_url"`    // 异步通知地址
	ReturnURL    string                 `json:"return_url"`    // 同步跳转地址
	ExpireTime   string                 `json:"expire_time"`   // 过期时间
	MerchantNo   string                 `json:"merchant_no"`   // 商户编号
	Channel      PaymentChannel         `json:"channel"`       // 支付渠道
	Extra        map[string]interface{} `json:"extra"`         // 扩展参数
	AlipayParams *AlipayOrderParams     `json:"alipay_params"` // 支付宝特有参数
	WechatParams *WechatOrderParams     `json:"wechat_params"` // 微信特有参数
}

// AlipayOrderParams 支付宝订单参数
type AlipayOrderParams struct {
	TimeoutExpress    string `json:"timeout_express"`     // 超时时间
	ExtendParams      string `json:"extend_params"`       // 扩展参数
	BuyerID           string `json:"buyer_id"`            // 买家ID
	BuyerLogonID      string `json:"buyer_logon_id"`      // 买家登录ID
	ProductCode       string `json:"product_code"`        // 产品码
	QuitURL           string `json:"quit_url"`            // 退出URL
	EnablePayChannels string `json:"enable_pay_channels"` // 可用支付渠道
}

// WechatOrderParams 微信订单参数
type WechatOrderParams struct {
	OpenID    string                 `json:"openid"`     // 用户OpenID（JSAPI支付必填）
	SceneInfo map[string]interface{} `json:"scene_info"` // 场景信息
	Detail    string                 `json:"detail"`     // 商品详情
	AppID     string                 `json:"app_id"`     // 应用ID
	Attach    string                 `json:"attach"`     // 附加数据
	GoodsTag  string                 `json:"goods_tag"`  // 商品标记
	LimitPay  string                 `json:"limit_pay"`  // 指定支付方式
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

// QueryOrderParams 查询订单参数
type QueryOrderParams struct {
	OutTradeNo    string         `json:"out_trade_no"`   // 商户订单号
	TradeNo       string         `json:"trade_no"`       // 平台订单号
	TransactionID string         `json:"transaction_id"` // 交易ID
	MerchantNo    string         `json:"merchant_no"`    // 商户编号
	Channel       PaymentChannel `json:"channel"`        // 支付渠道
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

// CreateRefundParams 创建退款参数
type CreateRefundParams struct {
	OutTradeNo    string         `json:"out_trade_no"`   // 商户订单号
	TransactionID string         `json:"transaction_id"` // 交易ID
	OutRefundNo   string         `json:"out_refund_no"`  // 商户退款号
	RefundAmount  int64          `json:"refund_amount"`  // 退款金额（分）
	TotalAmount   int64          `json:"total_amount"`   // 订单总金额（分）
	Reason        string         `json:"reason"`         // 退款原因
	RefundReason  string         `json:"refund_reason"`  // 退款原因（兼容性别名）
	NotifyURL     string         `json:"notify_url"`     // 异步通知地址
	MerchantNo    string         `json:"merchant_no"`    // 商户编号
	Channel       PaymentChannel `json:"channel"`        // 支付渠道
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

// QueryRefundParams 查询退款参数
type QueryRefundParams struct {
	OutTradeNo  string         `json:"out_trade_no"`  // 商户订单号
	OutRefundNo string         `json:"out_refund_no"` // 商户退款号
	RefundID    string         `json:"refund_id"`     // 退款ID
	MerchantNo  string         `json:"merchant_no"`   // 商户编号
	Channel     PaymentChannel `json:"channel"`       // 支付渠道
}

// QueryRefundResponse 查询退款响应
type QueryRefundResponse struct {
	Success      bool         `json:"success"`       // 是否成功
	OutRefundNo  string       `json:"out_refund_no"` // 商户退款号
	RefundID     string       `json:"refund_id"`     // 退款ID
	Status       RefundStatus `json:"status"`        // 退款状态
	RefundAmount int64        `json:"refund_amount"` // 退款金额（分）
	RefundTime   string       `json:"refund_time"`   // 退款时间
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

// HandleNotifyResponse 处理通知响应
type HandleNotifyResponse struct {
	Success    bool                   `json:"success"`      // 是否成功
	OrderID    string                 `json:"order_id"`     // 订单ID
	OutTradeNo string                 `json:"out_trade_no"` // 商户订单号
	TradeNo    string                 `json:"trade_no"`     // 平台订单号
	Status     OrderStatus            `json:"status"`       // 订单状态
	Amount     int64                  `json:"amount"`       // 金额（分）
	PayTime    string                 `json:"pay_time"`     // 支付时间
	NotifyData map[string]interface{} `json:"notify_data"`  // 通知数据
	Extra      map[string]interface{} `json:"extra"`        // 扩展信息
	Message    string                 `json:"message"`      // 消息
	Error      string                 `json:"error"`        // 错误信息
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
