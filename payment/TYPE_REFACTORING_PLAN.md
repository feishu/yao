# 支付模块类型系统重构方案

## 问题诊断

### 当前问题
1. **类型重复定义**：`payment/types.go` 和 `payment/providers/types.go` 中定义了相同的类型
2. **结构不一致**：相同用途的类型字段定义不同，造成混乱
3. **边界不清**：缺乏清晰的公共 API 类型和内部实现类型的分离

### 影响范围
- 类型转换复杂且容易出错
- 维护困难，修改一处需要同步多处
- 编译时类型检查失效
- 开发者容易混淆使用哪个类型

## 重构目标

1. **单一真理来源**：每个类型只定义一次
2. **清晰的分层**：公共 API 类型 vs 内部实现类型
3. **最小化转换**：减少不必要的类型转换
4. **向后兼容**：保持现有 Process API 不变

## 重构方案

### 方案 A：统一类型（推荐）

```
payment/
├── types.go              # 所有公共类型定义
├── providers/
│   ├── provider.go       # Provider 接口（使用 payment 包的类型）
│   ├── alipay/
│   │   └── alipay.go     # 支付宝实现
│   └── wechat/
│       └── wechat.go     # 微信支付实现
```

**核心思想**：
- `payment/types.go` 定义所有对外和对内的类型
- `providers/provider.go` 接口直接使用 `payment` 包的类型
- 各 provider 实现也直接使用 `payment` 包的类型
- **删除** `providers/types.go`

**优点**：
- 类型定义统一，无重复
- 无需类型转换
- 易于维护

**缺点**：
- providers 包依赖 payment 包（这是合理的依赖关系）

### 方案 B：严格分层（备选）

```
payment/
├── types/                # 独立的类型包
│   ├── common.go         # 公共类型（TradeType, OrderStatus 等）
│   ├── api.go            # API 层类型
│   └── provider.go       # Provider 层类型
├── payment.go
└── providers/
    └── provider.go
```

**优点**：
- 类型定义更独立
- 依赖关系更清晰

**缺点**：
- 增加了包的复杂度
- import 路径更长

## 推荐实施方案 A

### 第一步：清理 types.go

保留 `payment/types.go` 作为唯一的类型定义源：

```go
// payment/types.go

package payment

// ==================== 基础类型 ====================

// PaymentChannel 支付渠道
type PaymentChannel string

const (
    ChannelAlipay PaymentChannel = "alipay"
    ChannelWechat PaymentChannel = "wechat"
)

// TradeType 交易类型
type TradeType string

const (
    TradeTypeJSAPI  TradeType = "jsapi"  // 公众号/小程序支付
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
    RefundStatusPending    RefundStatus = "pending"    // 退款中
    RefundStatusProcessing RefundStatus = "processing" // 处理中
    RefundStatusSuccess    RefundStatus = "success"    // 退款成功
    RefundStatusFailed     RefundStatus = "failed"     // 退款失败
    RefundStatusClosed     RefundStatus = "closed"     // 退款关闭
    RefundStatusAbnormal   RefundStatus = "abnormal"   // 退款异常
)

// ==================== 订单相关 ====================

// CreateOrderParams 创建订单参数（统一）
type CreateOrderParams struct {
    // 基础字段
    OutTradeNo   string                 `json:"out_trade_no" validate:"required"`  // 商户订单号
    Amount       int64                  `json:"amount" validate:"required,min=1"`  // 金额（分）
    Subject      string                 `json:"subject" validate:"required"`       // 订单标题
    Body         string                 `json:"body"`                              // 订单描述
    TradeType    TradeType              `json:"trade_type" validate:"required"`    // 交易类型
    NotifyURL    string                 `json:"notify_url" validate:"required"`    // 异步通知地址
    ReturnURL    string                 `json:"return_url"`                        // 同步跳转地址
    ExpireTime   string                 `json:"expire_time"`                       // 过期时间
    Currency     string                 `json:"currency"`                          // 货币类型（默认CNY）
    
    // 系统字段
    MerchantNo   string                 `json:"merchant_no" validate:"required"`   // 商户编号
    Channel      PaymentChannel         `json:"channel" validate:"required"`       // 支付渠道
    
    // 渠道特有参数
    AlipayParams *AlipayOrderParams     `json:"alipay_params,omitempty"`           // 支付宝参数
    WechatParams *WechatOrderParams     `json:"wechat_params,omitempty"`           // 微信参数
    
    // 扩展参数
    ExtendParams map[string]interface{} `json:"extend_params,omitempty"`           // 扩展参数
}

// AlipayOrderParams 支付宝订单特有参数（统一）
type AlipayOrderParams struct {
    BuyerID            string `json:"buyer_id"`              // 买家支付宝用户ID
    BuyerLogonID       string `json:"buyer_logon_id"`        // 买家支付宝账号
    ProductCode        string `json:"product_code"`          // 产品码
    QuitURL            string `json:"quit_url"`              // 退出URL
    TimeoutExpress     string `json:"timeout_express"`       // 订单超时时间
    EnablePayChannels  string `json:"enable_pay_channels"`   // 可用渠道
    DisablePayChannels string `json:"disable_pay_channels"`  // 禁用渠道
    AuthToken          string `json:"auth_token"`            // 授权令牌
    QRPayMode          string `json:"qr_pay_mode"`           // 扫码方式
    QRCodeWidth        string `json:"qrcode_width"`          // 二维码宽度
    StoreID            string `json:"store_id"`              // 门店编号
    ExtendParams       string `json:"extend_params"`         // 业务扩展参数
}

// WechatOrderParams 微信订单特有参数（统一）
type WechatOrderParams struct {
    AppID     string                 `json:"appid"`      // 微信应用ID
    OpenID    string                 `json:"openid"`     // 用户OpenID（JSAPI必填）
    SceneInfo map[string]interface{} `json:"scene_info"` // 场景信息（H5必需）
    Detail    string                 `json:"detail"`     // 商品详情
    Attach    string                 `json:"attach"`     // 附加数据
    GoodsTag  string                 `json:"goods_tag"`  // 商品标记
    LimitPay  string                 `json:"limit_pay"`  // 指定支付方式
    StoreInfo string                 `json:"store_info"` // 门店信息
}

// CreateOrderResponse 创建订单响应（统一）
type CreateOrderResponse struct {
    Success    bool                   `json:"success"`      // 是否成功
    OrderID    string                 `json:"order_id"`     // 系统订单ID
    OutTradeNo string                 `json:"out_trade_no"` // 商户订单号
    PayInfo    map[string]interface{} `json:"pay_info"`     // 支付信息
    QRCode     string                 `json:"qr_code"`      // 二维码内容
    PayURL     string                 `json:"pay_url"`      // 支付链接
    Message    string                 `json:"message"`      // 响应消息
    Error      string                 `json:"error"`        // 错误信息
}

// QueryOrderParams 查询订单参数（统一）
type QueryOrderParams struct {
    OutTradeNo    string         `json:"out_trade_no"`   // 商户订单号
    TransactionID string         `json:"transaction_id"` // 支付平台交易号
    MerchantNo    string         `json:"merchant_no" validate:"required"` // 商户编号
    Channel       PaymentChannel `json:"channel" validate:"required"`     // 支付渠道
}

// QueryOrderResponse 查询订单响应（统一）
type QueryOrderResponse struct {
    Success       bool           `json:"success"`        // 是否成功
    OrderID       string         `json:"order_id"`       // 系统订单ID
    OutTradeNo    string         `json:"out_trade_no"`   // 商户订单号
    TransactionID string         `json:"transaction_id"` // 支付平台交易号
    Status        OrderStatus    `json:"status"`         // 订单状态
    Amount        int64          `json:"amount"`         // 订单金额（分）
    PaidAmount    int64          `json:"paid_amount"`    // 实付金额（分）
    Channel       PaymentChannel `json:"channel"`        // 支付渠道
    TradeType     TradeType      `json:"trade_type"`     // 交易类型
    PaidAt        string         `json:"paid_at"`        // 支付时间
    CreatedAt     string         `json:"created_at"`     // 创建时间
    Message       string         `json:"message"`        // 响应消息
    Error         string         `json:"error"`          // 错误信息
}

// ==================== 退款相关 ====================

// CreateRefundParams 创建退款参数（统一）
type CreateRefundParams struct {
    OutTradeNo    string         `json:"out_trade_no"`                      // 商户订单号
    TransactionID string         `json:"transaction_id"`                    // 支付平台交易号
    OutRefundNo   string         `json:"out_refund_no" validate:"required"` // 商户退款号
    RefundAmount  int64          `json:"refund_amount" validate:"required"` // 退款金额（分）
    TotalAmount   int64          `json:"total_amount" validate:"required"`  // 订单总金额（分）
    Reason        string         `json:"reason"`                            // 退款原因
    NotifyURL     string         `json:"notify_url"`                        // 异步通知地址
    MerchantNo    string         `json:"merchant_no" validate:"required"`   // 商户编号
    Channel       PaymentChannel `json:"channel" validate:"required"`       // 支付渠道
}

// CreateRefundResponse 创建退款响应（统一）
type CreateRefundResponse struct {
    Success      bool         `json:"success"`       // 是否成功
    RefundID     string       `json:"refund_id"`     // 系统退款ID
    OutRefundNo  string       `json:"out_refund_no"` // 商户退款号
    RefundNo     string       `json:"refund_no"`     // 支付平台退款号
    RefundAmount int64        `json:"refund_amount"` // 退款金额（分）
    Status       RefundStatus `json:"status"`        // 退款状态
    Message      string       `json:"message"`       // 响应消息
    Error        string       `json:"error"`         // 错误信息
}

// QueryRefundParams 查询退款参数（统一）
type QueryRefundParams struct {
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
    CreatedAt    string       `json:"created_at"`    // 创建时间
    Message      string       `json:"message"`       // 响应消息
    Error        string       `json:"error"`         // 错误信息
}

// ==================== 通知相关 ====================

// HandleNotifyParams 处理通知参数（统一）
type HandleNotifyParams struct {
    Channel     PaymentChannel         `json:"channel" validate:"required"` // 支付渠道
    MerchantNo  string                 `json:"merchant_no" validate:"required"` // 商户编号
    Request     *http.Request          `json:"-"`            // HTTP请求对象
    RequestBody []byte                 `json:"request_body"` // 请求体数据
    NotifyData  map[string]interface{} `json:"notify_data"`  // 通知数据
}

// HandleNotifyResponse 处理通知响应（统一）
type HandleNotifyResponse struct {
    Success       bool                   `json:"success"`        // 是否成功
    OrderID       string                 `json:"order_id"`       // 订单ID
    OutTradeNo    string                 `json:"out_trade_no"`   // 商户订单号
    TransactionID string                 `json:"transaction_id"` // 支付平台交易号
    Status        OrderStatus            `json:"status"`         // 订单状态
    Amount        int64                  `json:"amount"`         // 金额（分）
    PaidAt        *time.Time             `json:"paid_at"`        // 支付时间
    NotifyData    map[string]interface{} `json:"notify_data"`    // 原始通知数据
    Message       string                 `json:"message"`        // 响应消息
    Error         string                 `json:"error"`          // 错误信息
}

// ==================== 其他功能 ====================

// DownloadBillParams 下载对账单参数（统一）
type DownloadBillParams struct {
    Channel    PaymentChannel `json:"channel" validate:"required"`     // 支付渠道
    MerchantNo string         `json:"merchant_no" validate:"required"` // 商户编号
    BillDate   string         `json:"bill_date" validate:"required"`   // 对账单日期（YYYY-MM-DD）
    BillType   string         `json:"bill_type"`                       // 对账单类型
}

// DownloadBillResponse 下载对账单响应（统一）
type DownloadBillResponse struct {
    Success  bool   `json:"success"`   // 是否成功
    BillData string `json:"bill_data"` // 对账单数据
    Message  string `json:"message"`   // 响应消息
    Error    string `json:"error"`     // 错误信息
}
```

### 第二步：更新 Provider 接口

```go
// payment/providers/provider.go

package providers

import "github.com/yaoapp/yao/payment" // 引用 payment 包的类型

// Provider 支付渠道提供商接口
type Provider interface {
    // 基础信息
    Name() string
    Channel() payment.PaymentChannel
    
    // 订单操作
    CreateOrder(params *payment.CreateOrderParams) (*payment.CreateOrderResponse, error)
    QueryOrder(params *payment.QueryOrderParams) (*payment.QueryOrderResponse, error)
    CloseOrder(params *payment.QueryOrderParams) error
    
    // 退款操作
    CreateRefund(params *payment.CreateRefundParams) (*payment.CreateRefundResponse, error)
    QueryRefund(params *payment.QueryRefundParams) (*payment.QueryRefundResponse, error)
    
    // 通知处理
    HandleNotify(params *payment.HandleNotifyParams) (*payment.HandleNotifyResponse, error)
    
    // 对账功能
    DownloadBill(params *payment.DownloadBillParams) (*payment.DownloadBillResponse, error)
}
```

### 第三步：删除 providers/types.go

直接删除 `payment/providers/types.go` 文件，所有类型统一使用 `payment` 包中的定义。

### 第四步：更新各 Provider 实现

```go
// payment/providers/alipay/alipay.go

package alipay

import "github.com/yaoapp/yao/payment"

type AlipayProvider struct {
    // ...
}

func (p *AlipayProvider) CreateOrder(params *payment.CreateOrderParams) (*payment.CreateOrderResponse, error) {
    // 直接使用 payment 包的类型，无需转换
    // ...
}
```

## 实施步骤

1. ✅ **备份当前代码**（创建 git branch）
2. ⬜ **更新 payment/types.go**（合并和统一所有类型）
3. ⬜ **更新 providers/provider.go**（使用 payment 包类型）
4. ⬜ **删除 providers/types.go**
5. ⬜ **更新 alipay provider**
6. ⬜ **更新 wechat provider**
7. ⬜ **更新 payment.go 主文件**
8. ⬜ **运行所有测试**
9. ⬜ **修复编译错误**
10. ⬜ **验证功能正常**

## 兼容性考虑

### Process API 保持不变
所有 Process 函数的签名保持不变：
```go
func ProcessCreateOrder(process *process.Process) interface{}
func ProcessQueryOrder(process *process.Process) interface{}
// ...
```

### 配置文件保持不变
商户配置的 JSON 结构保持不变。

### 影响范围
- ✅ 对外 API：无影响
- ✅ 配置格式：无影响
- ⚠️ 内部代码：需要更新所有 provider 实现

## 预期收益

1. **代码量减少**：删除约 200 行重复类型定义
2. **维护性提升**：类型修改只需改一处
3. **类型安全**：编译期类型检查更严格
4. **开发体验**：无需记忆多套类型定义
5. **性能优化**：减少不必要的类型转换

## 风险控制

- ✅ 单元测试全覆盖
- ✅ 集成测试验证
- ✅ 向后兼容性检查
- ✅ 文档同步更新
