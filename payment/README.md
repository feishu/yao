# Payment Module

## 概述

Payment 模块为 Yao 项目提供统一的支付功能，集成了 `go-pay/gopay` 库，支持多种主流支付方式，包括微信支付、支付宝和 PayPal。该模块遵循 Clean Architecture 设计原则，提供完善的错误处理机制和安全支付标准。

## 功能特性

- **多支付方式支持**：微信支付、支付宝、PayPal
- **统一 API 接口**：提供一致的支付、查询、退款接口
- **Process 接口集成**：完全兼容 Yao 项目的 Process 系统
- **完善的错误处理**：详细的错误码和错误信息
- **安全支付标准**：支持沙箱环境、签名验证、通知验证
- **配置管理**：支持多个支付提供商的配置管理
- **并发安全**：线程安全的单例模式设计

## 支持的支付方式

### 微信支付 (WeChat Pay)
- 支付方式：JSAPI、NATIVE、APP、H5、小程序
- 货币：CNY（人民币）
- 特性：实时到账、支持退款、支持分账

### 支付宝 (Alipay)
- 支付方式：网页支付、手机网站支付、APP支付、当面付
- 货币：CNY（人民币）
- 特性：实时到账、支持退款、支持花呗分期

### PayPal
- 支付方式：网页支付、移动支付、快速结账
- 货币：USD、EUR、GBP、JPY 等多种国际货币
- 特性：国际支付、买家保护、支持退款

## 安装和配置

### 1. 依赖安装

该模块已集成到 Yao 项目中，依赖 `github.com/go-pay/gopay v1.5.104`。

### 2. 模块加载

在 Yao 项目启动时，Payment 模块会自动加载并注册相关的 Process 接口：

```go
import "github.com/yaoapp/yao/payment"

// 在应用启动时调用
payment.Load()
```

### 3. 配置支付提供商

使用 `utils.payment.AddConfig` Process 接口添加支付配置：

#### 微信支付配置
```javascript
Process("utils.payment.AddConfig", {
  "provider": "wechat",
  "app_id": "wx1234567890abcdef",
  "secret": "your_wechat_secret_key",
  "mch_id": "1234567890",
  "api_key": "your_api_key",
  "sandbox": false,
  "notify_url": "https://your-domain.com/payment/notify/wechat"
})
```

#### 支付宝配置
```javascript
Process("utils.payment.AddConfig", {
  "provider": "alipay",
  "app_id": "2021001234567890",
  "app_key": "your_alipay_app_key",
  "sandbox": false,
  "notify_url": "https://your-domain.com/payment/notify/alipay",
  "return_url": "https://your-domain.com/payment/return"
})
```

#### PayPal 配置
```javascript
Process("utils.payment.AddConfig", {
  "provider": "paypal",
  "app_id": "your_paypal_client_id",
  "secret": "your_paypal_secret",
  "sandbox": true,
  "return_url": "https://your-domain.com/payment/return",
  "cancel_url": "https://your-domain.com/payment/cancel"
})
```

## API 参考

### Process 接口列表

| Process 名称 | 功能描述 | 参数 | 返回值 |
|-------------|----------|------|--------|
| `utils.payment.CreatePayment` | 创建支付订单 | PaymentRequest | PaymentResponse |
| `utils.payment.QueryPayment` | 查询支付状态 | provider, order_id | PaymentResponse |
| `utils.payment.RefundPayment` | 申请退款 | provider, RefundRequest | RefundResponse |
| `utils.payment.AddConfig` | 添加支付配置 | PaymentConfig | 配置结果 |
| `utils.payment.GetSupportedProviders` | 获取支持的支付提供商 | 无 | 提供商列表 |
| `utils.payment.ValidateNotify` | 验证支付通知 | provider, notify_params | 验证结果 |

### 1. 创建支付订单

```javascript
// 创建微信支付订单
const paymentResponse = Process("utils.payment.CreatePayment", {
  "order_id": "ORDER_20240324_001",
  "amount": 10000,  // 金额，单位：分
  "currency": "CNY",
  "subject": "商品购买",
  "description": "购买商品描述",
  "provider": "wechat",
  "extra": {
    "pay_type": "NATIVE",  // 微信支付类型
    "openid": "user_openid"  // JSAPI 支付需要
  }
})

// 返回结果
{
  "order_id": "ORDER_20240324_001",
  "payment_id": "PAY_1234567890",
  "status": "pending",
  "amount": 10000,
  "currency": "CNY",
  "pay_url": "weixin://wxpay/bizpayurl?pr=xxx",
  "qr_code": "data:image/png;base64,xxx",
  "created_at": "2024-03-24T10:30:00Z",
  "expires_at": "2024-03-24T12:30:00Z"
}
```

### 2. 查询支付状态

```javascript
// 查询支付状态
const paymentStatus = Process("utils.payment.QueryPayment", "wechat", "ORDER_20240324_001")

// 返回结果
{
  "order_id": "ORDER_20240324_001",
  "payment_id": "PAY_1234567890",
  "status": "paid",  // pending, paid, failed, cancelled, refunded
  "amount": 10000,
  "currency": "CNY",
  "paid_at": "2024-03-24T10:35:00Z",
  "created_at": "2024-03-24T10:30:00Z"
}
```

### 3. 申请退款

```javascript
// 申请退款
const refundResponse = Process("utils.payment.RefundPayment", "wechat", {
  "order_id": "ORDER_20240324_001",
  "payment_id": "PAY_1234567890",
  "amount": 5000,  // 退款金额，单位：分
  "reason": "用户申请退款"
})

// 返回结果
{
  "refund_id": "REFUND_1234567890",
  "order_id": "ORDER_20240324_001",
  "payment_id": "PAY_1234567890",
  "status": "refunded",
  "amount": 5000,
  "reason": "用户申请退款",
  "created_at": "2024-03-24T11:00:00Z"
}
```

### 4. 获取支持的支付提供商

```javascript
// 获取支持的支付提供商
const providers = Process("utils.payment.GetSupportedProviders")

// 返回结果
{
  "providers": [
    {
      "provider": "wechat",
      "name": "微信支付",
      "description": "腾讯微信支付服务",
      "currencies": ["CNY"],
      "methods": ["JSAPI", "NATIVE", "APP", "H5", "MWEB"]
    },
    {
      "provider": "alipay",
      "name": "支付宝",
      "description": "蚂蚁集团支付宝服务",
      "currencies": ["CNY"],
      "methods": ["WEB", "WAP", "APP", "QR"]
    },
    {
      "provider": "paypal",
      "name": "PayPal",
      "description": "PayPal 国际支付服务",
      "currencies": ["USD", "EUR", "GBP", "JPY", "CNY"],
      "methods": ["WEB", "MOBILE", "EXPRESS"]
    }
  ],
  "total": 3
}
```

### 5. 验证支付通知

```javascript
// 验证微信支付通知
const validation = Process("utils.payment.ValidateNotify", "wechat", {
  "out_trade_no": "ORDER_20240324_001",
  "trade_state": "SUCCESS",
  "transaction_id": "4200001234567890",
  // ... 其他微信通知参数
})

// 返回结果
{
  "valid": true,
  "order_id": "ORDER_20240324_001",
  "provider": "wechat",
  "message": "通知验证成功"
}
```

## 数据结构

### PaymentRequest（支付请求）

```go
type PaymentRequest struct {
    OrderID     string                 `json:"order_id"`     // 订单号（必填）
    Amount      int64                  `json:"amount"`       // 金额，单位：分（必填）
    Currency    string                 `json:"currency"`     // 货币代码（必填）
    Subject     string                 `json:"subject"`      // 支付主题（必填）
    Description string                 `json:"description"`  // 支付描述（可选）
    Provider    string                 `json:"provider"`     // 支付提供商（必填）
    NotifyURL   string                 `json:"notify_url"`   // 异步通知地址（可选）
    ReturnURL   string                 `json:"return_url"`   // 同步返回地址（可选）
    Extra       map[string]interface{} `json:"extra"`        // 额外参数（可选）
}
```

### PaymentResponse（支付响应）

```go
type PaymentResponse struct {
    OrderID     string                 `json:"order_id"`     // 订单号
    PaymentID   string                 `json:"payment_id"`   // 支付ID
    Status      PaymentStatus          `json:"status"`       // 支付状态
    Amount      int64                  `json:"amount"`       // 金额
    Currency    string                 `json:"currency"`     // 货币代码
    PayURL      string                 `json:"pay_url"`      // 支付链接
    QRCode      string                 `json:"qr_code"`      // 二维码（Base64）
    PaidAt      *time.Time             `json:"paid_at"`      // 支付时间
    CreatedAt   time.Time              `json:"created_at"`   // 创建时间
    ExpiresAt   *time.Time             `json:"expires_at"`   // 过期时间
    Extra       map[string]interface{} `json:"extra"`        // 额外信息
}
```

### RefundRequest（退款请求）

```go
type RefundRequest struct {
    OrderID   string `json:"order_id"`   // 原订单号（必填）
    PaymentID string `json:"payment_id"` // 支付ID（必填）
    Amount    int64  `json:"amount"`     // 退款金额，单位：分（必填）
    Reason    string `json:"reason"`     // 退款原因（可选）
}
```

### RefundResponse（退款响应）

```go
type RefundResponse struct {
    RefundID  string        `json:"refund_id"`  // 退款ID
    OrderID   string        `json:"order_id"`   // 原订单号
    PaymentID string        `json:"payment_id"` // 支付ID
    Status    PaymentStatus `json:"status"`     // 退款状态
    Amount    int64         `json:"amount"`     // 退款金额
    Reason    string        `json:"reason"`     // 退款原因
    CreatedAt time.Time     `json:"created_at"` // 退款时间
}
```

### PaymentStatus（支付状态）

```go
type PaymentStatus string

const (
    StatusPending   PaymentStatus = "pending"   // 待支付
    StatusPaid      PaymentStatus = "paid"      // 已支付
    StatusFailed    PaymentStatus = "failed"    // 支付失败
    StatusCancelled PaymentStatus = "cancelled" // 已取消
    StatusRefunded  PaymentStatus = "refunded"  // 已退款
)
```

## 错误处理

### 错误码定义

```go
const (
    ErrCodeInvalidConfig    = "INVALID_CONFIG"     // 配置无效
    ErrCodeProviderNotFound = "PROVIDER_NOT_FOUND" // 支付提供商未找到
    ErrCodeInvalidRequest   = "INVALID_REQUEST"    // 请求参数无效
    ErrCodePaymentFailed    = "PAYMENT_FAILED"     // 支付失败
    ErrCodeRefundFailed     = "REFUND_FAILED"      // 退款失败
    ErrCodeNotifyInvalid    = "NOTIFY_INVALID"     // 通知验证失败
    ErrCodeNetworkError     = "NETWORK_ERROR"      // 网络错误
    ErrCodeSystemError      = "SYSTEM_ERROR"       // 系统错误
)
```

### 错误处理示例

```javascript
try {
  const result = Process("utils.payment.CreatePayment", paymentRequest)
  console.log("支付创建成功:", result)
} catch (error) {
  switch (error.code) {
    case "INVALID_CONFIG":
      console.error("支付配置无效:", error.message)
      break
    case "PROVIDER_NOT_FOUND":
      console.error("支付提供商未找到:", error.message)
      break
    case "INVALID_REQUEST":
      console.error("请求参数无效:", error.message)
      break
    case "PAYMENT_FAILED":
      console.error("支付失败:", error.message)
      break
    default:
      console.error("未知错误:", error.message)
  }
}
```

## 安全注意事项

### 1. 配置安全
- 生产环境必须使用 HTTPS
- API 密钥和证书文件需要安全存储
- 定期更换 API 密钥
- 不要在客户端暴露敏感配置

### 2. 通知验证
- 必须验证支付通知的签名
- 验证通知来源的 IP 地址
- 防止重复处理同一通知
- 记录所有通知日志

### 3. 金额处理
- 使用整数表示金额（单位：分）
- 避免浮点数计算精度问题
- 验证金额范围和格式
- 记录所有金额变更日志

### 4. 订单安全
- 订单号必须唯一且不可预测
- 实现幂等性处理
- 设置合理的订单过期时间
- 防止订单状态异常变更

## 测试

### 运行测试

```bash
# 运行所有测试
go test ./payment/...

# 运行测试并显示覆盖率
go test -cover ./payment/...

# 运行性能测试
go test -bench=. ./payment/...

# 运行特定测试
go test -run TestPaymentManager ./payment/...
```

### 测试覆盖率

当前测试覆盖率目标：**80%** 以上

主要测试场景：
- 配置管理测试
- 支付创建测试
- 支付查询测试
- 退款处理测试
- 通知验证测试
- 错误处理测试
- 并发安全测试
- 性能基准测试

### 沙箱测试

所有支付提供商都支持沙箱环境测试：

```javascript
// 启用沙箱模式
Process("utils.payment.AddConfig", {
  "provider": "wechat",
  "app_id": "wx_sandbox_app_id",
  "secret": "sandbox_secret",
  "sandbox": true,  // 启用沙箱模式
  "notify_url": "https://your-test-domain.com/payment/notify/wechat"
})
```

## 性能优化

### 1. 连接池管理
- HTTP 客户端使用连接池
- 合理设置连接超时时间
- 复用 HTTP 连接

### 2. 缓存策略
- 缓存支付配置信息
- 缓存支付状态查询结果
- 使用本地缓存减少网络请求

### 3. 异步处理
- 支付通知异步处理
- 退款申请异步处理
- 状态同步异步更新

### 4. 监控指标
- 支付成功率监控
- 接口响应时间监控
- 错误率统计
- 并发量监控

## 故障排查

### 常见问题

1. **配置错误**
   - 检查 API 密钥是否正确
   - 确认沙箱/生产环境配置
   - 验证回调地址可访问性

2. **支付失败**
   - 检查网络连接
   - 验证请求参数格式
   - 查看支付提供商错误信息

3. **通知验证失败**
   - 检查签名算法实现
   - 验证证书配置
   - 确认通知参数完整性

4. **退款异常**
   - 确认原支付订单状态
   - 检查退款金额限制
   - 验证退款权限配置

### 日志分析

```go
// 启用详细日志
log.With(log.F{
    "module": "payment",
    "action": "create_payment",
    "order_id": orderID,
    "provider": provider,
}).Info("创建支付订单")

log.With(log.F{
    "module": "payment",
    "action": "payment_notify",
    "provider": provider,
    "order_id": orderID,
    "status": status,
}).Info("收到支付通知")
```

## 版本历史

- **v1.0.0** (2024-03-24)
  - 初始版本发布
  - 支持微信支付、支付宝、PayPal
  - 实现完整的 Process 接口
  - 提供完善的错误处理和安全机制

## 贡献指南

1. Fork 项目仓库
2. 创建功能分支
3. 编写测试用例
4. 确保测试通过
5. 提交 Pull Request

## 许可证

本模块遵循 Yao 项目的开源许可证。

---

**最后更新：2024-03-24**  
**版本：v1.0.0**  
**维护者：Yao 开发团队**