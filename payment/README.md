# Payment Module

## 概述

Payment 模块是 Yao 应用引擎的支付处理模块，提供统一的支付接口，支持多种支付渠道（支付宝、微信支付等）。该模块采用插件化架构，易于扩展和维护。

## 功能特性

- **多支付渠道支持**：支付宝、微信支付
- **统一接口设计**：提供一致的 API 接口
- **灵活配置管理**：支持多商户配置
- **完整的支付流程**：订单创建、查询、退款、对账
- **异步通知处理**：支持支付结果异步通知
- **数据持久化**：完整的订单和退款记录
- **Process 接口集成**：与 Yao 引擎无缝集成
- **安全可靠**：支持签名验证和数据加密
- **高性能**：支持并发处理和缓存优化

## 支持的支付方式

### 支付宝
- **Native**：扫码支付
- **WAP**：手机网站支付
- **APP**：移动应用支付

### 微信支付
- **JSAPI**：公众号支付/小程序支付
- **Native**：扫码支付
- **APP**：移动应用支付
- **H5**：H5支付

## 目录结构

```
payment/
├── payment.go              # 主要实现文件
├── process.go              # Process 接口实现
├── types.go                # 类型定义
├── load.go                 # 加载和初始化
├── providers/              # 支付提供商实现
│   ├── alipay.go          # 支付宝提供商
│   ├── alipay_test.go     # 支付宝测试
│   ├── wechat.go          # 微信支付提供商
│   └── wechat_test.go     # 微信支付测试
├── payment_test.go         # 主模块测试
├── process_test.go         # Process 接口测试
├── load_test.go           # 加载模块测试
└── README.md              # 模块文档
```

## 快速开始

### 1. 环境准备

在使用 Payment 模块之前，请确保：

1. **Yao 应用引擎**：已正确安装和配置 Yao 应用引擎
2. **支付平台账户**：已在支付宝/微信支付平台注册商户账户
3. **证书和密钥**：已获取相应的应用证书、私钥和公钥
4. **网络环境**：确保服务器能够访问支付平台的 API 接口

### 2. 模块加载

Payment 模块会在 Yao 应用启动时自动加载，无需手动初始化。

### 3. 配置商户信息

#### 3.1 支付宝配置

```go
// 支付宝配置参数
alipayConfig := map[string]interface{}{
    "app_id":            "2021000000000000",           // 应用ID（必填）
    "private_key":       "MIIEvQIBADANBgkqhkiG9w0...", // 应用私钥（必填）
    "alipay_public_key": "MIIBIjANBgkqhkiG9w0BAQE...", // 支付宝公钥（必填）
    "is_production":     false,                        // 是否生产环境（可选，默认false）
    "sign_type":         "RSA2",                       // 签名类型（可选，默认RSA2）
    "charset":           "utf-8",                      // 字符集（可选，默认utf-8）
}

// 设置商户配置
process.New("payment.SetConfig", "merchant_001", "alipay", alipayConfig).Run()
```

#### 3.2 微信支付配置

```go
// 微信支付配置参数
wechatConfig := map[string]interface{}{
    "app_id":        "wx1234567890abcdef",           // 应用ID（必填）
    "mch_id":        "1234567890",                   // 商户号（必填）
    "apiv3_key":     "your_apiv3_key_32_characters", // APIv3密钥（必填）
    "private_key":   "-----BEGIN PRIVATE KEY-----...", // 商户私钥（必填）
    "serial_no":     "1234567890ABCDEF1234567890ABCDEF12345678", // 证书序列号（必填）
    "is_production": false,                          // 是否生产环境（可选，默认false）
}

// 设置商户配置
process.New("payment.SetConfig", "merchant_001", "wechat", wechatConfig).Run()
```

### 4. 创建支付订单

#### 4.1 支付宝扫码支付

```go
orderParams := map[string]interface{}{
    "merchant_no":  "merchant_001",                    // 商户编号
    "channel":      "alipay",                          // 支付渠道
    "trade_type":   "native",                          // 交易类型
    "out_trade_no": "order_20231201_001",              // 商户订单号
    "amount":       100,                               // 支付金额（分）
    "subject":      "测试商品",                         // 订单标题
    "body":         "这是一个测试商品的详细描述",        // 订单描述
    "notify_url":   "https://your-domain.com/notify",  // 异步通知地址
    "return_url":   "https://your-domain.com/return",  // 同步跳转地址（可选）
    "expire_time":  "2023-12-01T18:00:00+08:00",      // 订单过期时间（可选）
}

result := process.New("payment.CreateOrder", orderParams).Run()
```

#### 4.2 微信公众号支付

```go
orderParams := map[string]interface{}{
    "merchant_no":  "merchant_001",
    "channel":      "wechat",
    "trade_type":   "jsapi",
    "out_trade_no": "order_20231201_002",
    "amount":       200,
    "subject":      "VIP会员服务",
    "body":         "购买一年期VIP会员服务",
    "notify_url":   "https://your-domain.com/notify",
    "wechat_params": map[string]interface{}{
        "appid":      "wx1234567890abcdef",
        "scene_info": `{"h5_info": {"type":"Wap","wap_url": "https://your-domain.com","wap_name": "商户名称"}}`,
        "attach":     "member_service",
    },
}

result := process.New("payment.CreateOrder", orderParams).Run()
```

### 5. 查询订单状态

```go
queryParams := map[string]interface{}{
    "merchant_no":  "merchant_001",
    "channel":      "alipay",
    "out_trade_no": "order_20231201_001",  // 使用商户订单号查询
    // 或者使用支付平台交易号查询
    // "transaction_id": "2023120122001234567890123456",
}

result := process.New("payment.QueryOrder", queryParams).Run()
```

### 6. 创建退款

```go
refundParams := map[string]interface{}{
    "merchant_no":    "merchant_001",
    "channel":        "alipay",
    "out_trade_no":   "order_20231201_001",           // 原订单号
    "out_refund_no":  "refund_20231201_001",          // 退款单号
    "refund_amount":  50,                             // 退款金额（分）
    "total_amount":   100,                            // 原订单总金额（分）
    "reason":         "用户申请退款",                  // 退款原因
    "notify_url":     "https://your-domain.com/refund_notify", // 退款异步通知地址（可选）
}

result := process.New("payment.CreateRefund", refundParams).Run()
```

### 7. 处理异步通知

```go
// 在您的 HTTP 处理器中
func handlePaymentNotify(c *gin.Context) {
    // 获取请求体数据
    body, _ := ioutil.ReadAll(c.Request.Body)
    
    // 处理通知
    result := process.New("payment.HandleNotify", "merchant_001", "alipay", string(body)).Run()
    
    // 根据处理结果返回响应
    if result.(map[string]interface{})["success"].(bool) {
        c.String(200, "success")
    } else {
        c.String(400, "fail")
    }
}
```

## API 参考

### Process 接口列表

Payment 模块提供以下 Process 接口：

#### 1. payment.SetConfig - 设置商户配置

**功能**：为指定商户和支付渠道设置配置信息

**参数**：
- `merchant_no` (string): 商户编号，用于标识不同的商户
- `channel` (string): 支付渠道，支持 "alipay"、"wechat"
- `config` (map[string]interface{}): 配置参数对象

**返回值**：
```go
{
    "success": true,
    "message": "配置设置成功"
}
```

**错误情况**：
- 参数不足或类型错误
- 不支持的支付渠道
- 配置参数验证失败

#### 2. payment.GetConfig - 获取商户配置

**功能**：获取指定商户和支付渠道的配置信息

**参数**：
- `merchant_no` (string): 商户编号
- `channel` (string): 支付渠道

**返回值**：
```go
{
    "success": true,
    "data": {
        "app_id": "2021000000000000",
        "is_production": false,
        "sign_type": "RSA2",
        // ... 其他配置信息（敏感信息已脱敏）
    }
}
```

#### 3. payment.CreateOrder - 创建支付订单

**功能**：创建支付订单并获取支付参数

**参数**：
- `merchant_no` (string): 商户编号
- `channel` (string): 支付渠道
- `trade_type` (string): 交易类型
  - 支付宝：native（扫码）、app（APP）、page（网页）、wap（手机网页）
  - 微信：native（扫码）、jsapi（公众号/小程序）、app（APP）、h5（H5）
- `out_trade_no` (string): 商户订单号，需保证唯一性
- `amount` (int): 支付金额，单位为分
- `subject` (string): 订单标题
- `body` (string): 订单描述（可选）
- `notify_url` (string): 异步通知地址
- `return_url` (string): 同步跳转地址（可选）
- `expire_time` (string): 订单过期时间（可选，ISO 8601 格式）
- `alipay_params` (map[string]interface{}): 支付宝特有参数（可选）
- `wechat_params` (map[string]interface{}): 微信支付特有参数（可选）

**返回值**：
```go
{
    "success": true,
    "data": {
        "out_trade_no": "order_20231201_001",
        "transaction_id": "2023120122001234567890123456",
        "pay_url": "https://qr.alipay.com/bax08431...", // 支付链接或二维码内容
        "qr_code": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA...", // 二维码图片（base64）
        "pay_params": {
            // 客户端支付所需的参数
        }
    }
}
```

#### 4. payment.QueryOrder - 查询订单状态

**功能**：查询支付订单的当前状态

**参数**：
- `merchant_no` (string): 商户编号
- `channel` (string): 支付渠道
- `out_trade_no` (string): 商户订单号（与 transaction_id 二选一）
- `transaction_id` (string): 支付平台交易号（与 out_trade_no 二选一）

**返回值**：
```go
{
    "success": true,
    "data": {
        "out_trade_no": "order_20231201_001",
        "transaction_id": "2023120122001234567890123456",
        "status": "PAID", // PENDING, PAID, FAILED, CLOSED, REFUNDED
        "amount": 100,
        "paid_amount": 100,
        "paid_time": "2023-12-01T15:30:00+08:00",
        "buyer_info": {
            "buyer_id": "2088123456789012",
            "buyer_name": "张***"
        }
    }
}
```

#### 5. payment.CreateRefund - 创建退款

**功能**：创建退款申请

**参数**：
- `merchant_no` (string): 商户编号
- `channel` (string): 支付渠道
- `out_trade_no` (string): 原订单的商户订单号
- `out_refund_no` (string): 退款单号，需保证唯一性
- `refund_amount` (int): 退款金额，单位为分
- `total_amount` (int): 原订单总金额，单位为分
- `reason` (string): 退款原因
- `notify_url` (string): 退款异步通知地址（可选）

**返回值**：
```go
{
    "success": true,
    "data": {
        "out_refund_no": "refund_20231201_001",
        "refund_id": "2023120122001234567890123456",
        "status": "PROCESSING", // PROCESSING, SUCCESS, FAILED
        "refund_amount": 50,
        "refund_time": "2023-12-01T16:00:00+08:00"
    }
}
```

#### 6. payment.QueryRefund - 查询退款状态

**功能**：查询退款申请的当前状态

**参数**：
- `merchant_no` (string): 商户编号
- `channel` (string): 支付渠道
- `out_refund_no` (string): 退款单号

**返回值**：
```go
{
    "success": true,
    "data": {
        "out_refund_no": "refund_20231201_001",
        "refund_id": "2023120122001234567890123456",
        "status": "SUCCESS", // PROCESSING, SUCCESS, FAILED
        "refund_amount": 50,
        "refund_time": "2023-12-01T16:00:00+08:00",
        "arrival_time": "2023-12-01T16:05:00+08:00"
    }
}
```

#### 7. payment.HandleNotify - 处理异步通知

**功能**：处理来自支付平台的异步通知

**参数**：
- `merchant_no` (string): 商户编号
- `channel` (string): 支付渠道
- `notify_data` (string): 通知数据（HTTP 请求体）

**返回值**：
```go
{
    "success": true,
    "data": {
        "out_trade_no": "order_20231201_001",
        "transaction_id": "2023120122001234567890123456",
        "status": "PAID",
        "amount": 100,
        "paid_time": "2023-12-01T15:30:00+08:00",
        "notify_type": "trade_status_sync" // 通知类型
    }
}
```

#### 8. payment.DownloadBill - 下载对账单

**功能**：下载指定日期的交易对账单

**参数**：
- `merchant_no` (string): 商户编号
- `channel` (string): 支付渠道
- `bill_date` (string): 对账单日期，格式：YYYY-MM-DD
- `bill_type` (string): 对账单类型（可选）
  - 支付宝：trade（交易）、signcustomer（签约）
  - 微信：ALL（全部）、SUCCESS（成功）、REFUND（退款）

**返回值**：
```go
{
    "success": true,
    "data": {
        "bill_date": "2023-12-01",
        "download_url": "https://api.alipay.com/...", // 对账单下载链接
        "file_content": "交易时间,商户订单号,交易号...", // 对账单内容（CSV格式）
        "total_count": 150,
        "total_amount": 15000
    }
}
```

#### 9. payment.Reconcile - 对账处理

**功能**：执行对账处理，比较本地订单与支付平台账单

**参数**：
- `merchant_no` (string): 商户编号
- `channel` (string): 支付渠道
- `bill_date` (string): 对账日期，格式：YYYY-MM-DD
- `local_orders` ([]map[string]interface{}): 本地订单数据（可选）

**返回值**：
```go
{
    "success": true,
    "data": {
        "bill_date": "2023-12-01",
        "total_orders": 150,
        "matched_orders": 148,
        "unmatched_orders": 2,
        "platform_only": 1,  // 平台有但本地没有的订单数
        "local_only": 1,     // 本地有但平台没有的订单数
        "differences": [
            {
                "out_trade_no": "order_20231201_100",
                "type": "amount_mismatch",
                "local_amount": 100,
                "platform_amount": 99,
                "description": "金额不匹配"
            }
        ]
    }
}
```

## 配置说明

### 支付宝配置参数

| 参数名 | 类型 | 必填 | 说明 | 示例值 |
|--------|------|------|------|--------|
| `app_id` | string | 是 | 支付宝应用ID | "2021000000000000" |
| `private_key` | string | 是 | 应用私钥（PKCS8格式） | "MIIEvQIBADANBgkqhkiG9w0..." |
| `alipay_public_key` | string | 是 | 支付宝公钥 | "MIIBIjANBgkqhkiG9w0BAQE..." |
| `is_production` | bool | 否 | 是否生产环境 | false |
| `sign_type` | string | 否 | 签名类型 | "RSA2" |
| `charset` | string | 否 | 字符集 | "utf-8" |
| `format` | string | 否 | 数据格式 | "JSON" |
| `version` | string | 否 | API版本 | "1.0" |

**获取配置信息的步骤：**

1. **注册支付宝开放平台账户**
   - 访问 [支付宝开放平台](https://open.alipay.com/)
   - 注册开发者账户并完成实名认证

2. **创建应用**
   - 在控制台创建应用（网页&移动应用或小程序应用）
   - 获取应用ID（app_id）

3. **配置应用信息**
   - 上传应用图标和描述
   - 配置应用网关和授权回调地址

4. **生成密钥**
   - 使用支付宝提供的密钥生成工具生成RSA2密钥对
   - 上传应用公钥到支付宝平台
   - 获取支付宝公钥

5. **添加功能**
   - 在应用中添加需要的支付功能（如：手机网站支付、电脑网站支付等）
   - 签约相应的产品协议

### 微信支付配置参数

| 参数名 | 类型 | 必填 | 说明 | 示例值 |
|--------|------|------|------|--------|
| `app_id` | string | 是 | 微信应用ID | "wx1234567890abcdef" |
| `mch_id` | string | 是 | 微信支付商户号 | "1234567890" |
| `apiv3_key` | string | 是 | APIv3密钥 | "your_apiv3_key_32_characters" |
| `private_key` | string | 是 | 商户私钥（PKCS8格式） | "-----BEGIN PRIVATE KEY-----..." |
| `serial_no` | string | 是 | 商户证书序列号 | "1234567890ABCDEF1234567890ABCDEF12345678" |
| `is_production` | bool | 否 | 是否生产环境 | false |
| `notify_url` | string | 否 | 默认异步通知地址 | "https://your-domain.com/notify" |

**获取配置信息的步骤：**

1. **注册微信支付商户**
   - 访问 [微信支付商户平台](https://pay.weixin.qq.com/)
   - 注册商户账户并完成资质审核

2. **获取商户信息**
   - 登录商户平台获取商户号（mch_id）
   - 在账户中心查看商户基本信息

3. **配置API证书**
   - 在商户平台下载API证书
   - 获取证书序列号（serial_no）
   - 提取商户私钥（private_key）

4. **设置APIv3密钥**
   - 在商户平台设置APIv3密钥（32位字符）
   - 记录密钥用于配置（apiv3_key）

5. **关联微信应用**
   - 在微信公众平台或开放平台创建应用
   - 获取应用ID（app_id）
   - 在商户平台关联应用

### 环境配置

#### 开发环境配置

```go
// 开发环境 - 支付宝
alipayDevConfig := map[string]interface{}{
    "app_id":            "2021000000000000",  // 沙箱应用ID
    "private_key":       devPrivateKey,       // 开发环境私钥
    "alipay_public_key": devAlipayPublicKey,  // 沙箱公钥
    "is_production":     false,               // 开发环境
    "gateway_url":       "https://openapi.alipaydev.com/gateway.do", // 沙箱网关
}

// 开发环境 - 微信支付
wechatDevConfig := map[string]interface{}{
    "app_id":        "wx1234567890abcdef",
    "mch_id":        "1234567890",
    "apiv3_key":     "test_apiv3_key_32_characters_dev",
    "private_key":   devPrivateKey,
    "serial_no":     devSerialNo,
    "is_production": false,
    "base_url":      "https://api.mch.weixin.qq.com", // 测试环境URL
}
```

#### 生产环境配置

```go
// 生产环境 - 支付宝
alipayProdConfig := map[string]interface{}{
    "app_id":            "2021000000000001",  // 正式应用ID
    "private_key":       prodPrivateKey,      // 生产环境私钥
    "alipay_public_key": prodAlipayPublicKey, // 正式公钥
    "is_production":     true,                // 生产环境
    "gateway_url":       "https://openapi.alipay.com/gateway.do", // 正式网关
}

// 生产环境 - 微信支付
wechatProdConfig := map[string]interface{}{
    "app_id":        "wx1234567890abcdef",
    "mch_id":        "1234567890",
    "apiv3_key":     "prod_apiv3_key_32_characters_prod",
    "private_key":   prodPrivateKey,
    "serial_no":     prodSerialNo,
    "is_production": true,
    "base_url":      "https://api.mch.weixin.qq.com", // 生产环境URL
}
```

### 安全配置建议

1. **密钥管理**
   - 私钥文件权限设置为600（仅所有者可读写）
   - 使用环境变量或配置文件存储敏感信息
   - 定期轮换APIv3密钥

2. **网络安全**
   - 使用HTTPS协议进行所有API调用
   - 配置防火墙仅允许必要的出站连接
   - 设置IP白名单（如支持）

3. **日志安全**
   - 不在日志中记录完整的私钥和密钥
   - 对敏感参数进行脱敏处理
   - 定期清理过期日志文件

4. **配置验证**
   ```go
   // 配置验证示例
   func validateConfig(config map[string]interface{}) error {
       required := []string{"app_id", "private_key", "alipay_public_key"}
       for _, key := range required {
           if _, exists := config[key]; !exists {
               return fmt.Errorf("缺少必需的配置参数: %s", key)
           }
       }
       return nil
   }
   ```

## 数据模型

### MerchantConfig
商户配置模型，用于存储商户的支付配置信息。

### PaymentOrder
支付订单模型，记录所有支付订单的详细信息。

### PaymentRefund
退款记录模型，记录所有退款操作的详细信息。

### NotifyLog
通知日志模型，记录所有异步通知的处理日志。

## 错误处理

### 常见错误类型

Payment 模块的错误处理遵循统一的错误格式，所有错误都会返回包含错误信息的结构：

```go
{
    "success": false,
    "error": {
        "code": "INVALID_PARAMETER",
        "message": "参数验证失败：merchant_no 不能为空",
        "details": {
            "field": "merchant_no",
            "value": "",
            "expected": "非空字符串"
        }
    }
}
```

### 错误代码说明

#### 1. 参数错误 (4xx)

| 错误代码 | 说明 | 解决方案 |
|----------|------|----------|
| `INVALID_PARAMETER` | 参数验证失败 | 检查参数类型和必填项 |
| `MISSING_PARAMETER` | 缺少必需参数 | 补充缺失的参数 |
| `INVALID_MERCHANT` | 商户不存在或未配置 | 检查商户编号和配置 |
| `INVALID_CHANNEL` | 不支持的支付渠道 | 使用支持的渠道：alipay、wechat |
| `INVALID_TRADE_TYPE` | 不支持的交易类型 | 检查交易类型是否与渠道匹配 |
| `DUPLICATE_ORDER` | 订单号重复 | 使用唯一的商户订单号 |
| `ORDER_NOT_FOUND` | 订单不存在 | 检查订单号是否正确 |
| `INVALID_AMOUNT` | 金额格式错误 | 确保金额为正整数（分） |

#### 2. 配置错误 (5xx)

| 错误代码 | 说明 | 解决方案 |
|----------|------|----------|
| `CONFIG_NOT_FOUND` | 商户配置不存在 | 先调用 SetConfig 设置配置 |
| `INVALID_CONFIG` | 配置参数无效 | 检查配置参数格式和完整性 |
| `CERT_INVALID` | 证书或密钥无效 | 检查证书格式和有效期 |
| `SIGN_VERIFY_FAILED` | 签名验证失败 | 检查密钥配置和签名算法 |

#### 3. 业务错误 (6xx)

| 错误代码 | 说明 | 解决方案 |
|----------|------|----------|
| `ORDER_EXPIRED` | 订单已过期 | 创建新订单 |
| `ORDER_PAID` | 订单已支付 | 不能重复支付 |
| `ORDER_CLOSED` | 订单已关闭 | 创建新订单 |
| `REFUND_FAILED` | 退款失败 | 检查退款条件和金额 |
| `INSUFFICIENT_BALANCE` | 余额不足 | 用户充值后重试 |

#### 4. 网络错误 (7xx)

| 错误代码 | 说明 | 解决方案 |
|----------|------|----------|
| `NETWORK_ERROR` | 网络连接失败 | 检查网络连接和防火墙设置 |
| `TIMEOUT_ERROR` | 请求超时 | 重试或增加超时时间 |
| `API_UNAVAILABLE` | 支付平台API不可用 | 稍后重试或联系平台客服 |

### 错误处理最佳实践

#### 1. 统一错误处理

```go
func handlePaymentResult(result interface{}) error {
    resultMap, ok := result.(map[string]interface{})
    if !ok {
        return fmt.Errorf("无效的返回结果格式")
    }
    
    success, exists := resultMap["success"]
    if !exists || !success.(bool) {
        errorInfo := resultMap["error"].(map[string]interface{})
        return fmt.Errorf("支付操作失败: %s", errorInfo["message"])
    }
    
    return nil
}
```

#### 2. 重试机制

```go
func createOrderWithRetry(params map[string]interface{}, maxRetries int) (interface{}, error) {
    var lastErr error
    
    for i := 0; i < maxRetries; i++ {
        result := process.New("payment.CreateOrder", params).Run()
        
        if err := handlePaymentResult(result); err == nil {
            return result, nil
        } else {
            lastErr = err
            
            // 判断是否需要重试
            if shouldRetry(err) {
                time.Sleep(time.Duration(i+1) * time.Second)
                continue
            }
            break
        }
    }
    
    return nil, lastErr
}

func shouldRetry(err error) bool {
    // 网络错误和超时错误可以重试
    return strings.Contains(err.Error(), "NETWORK_ERROR") ||
           strings.Contains(err.Error(), "TIMEOUT_ERROR")
}
```

#### 3. 日志记录

```go
import "github.com/yaoapp/kun/log"

func logPaymentError(operation string, params map[string]interface{}, err error) {
    log.Error("Payment operation failed: %s", operation)
    log.With(log.F{
        "operation": operation,
        "merchant_no": params["merchant_no"],
        "channel": params["channel"],
        "error": err.Error(),
    }).Error("Payment error details")
}
```

### 故障排查指南

#### 1. 配置问题排查

**问题**：签名验证失败
```bash
# 检查步骤
1. 验证私钥格式是否正确（PKCS8格式）
2. 确认公钥是否已正确上传到支付平台
3. 检查应用ID和商户号是否匹配
4. 验证证书序列号是否正确
```

**问题**：网络连接失败
```bash
# 检查步骤
1. 测试网络连通性：ping api.alipay.com
2. 检查防火墙设置和出站规则
3. 验证DNS解析：nslookup api.mch.weixin.qq.com
4. 检查代理设置（如有）
```

#### 2. 订单问题排查

**问题**：订单创建失败
```go
// 调试代码示例
func debugCreateOrder(params map[string]interface{}) {
    log.Debug("Creating order with params: %+v", params)
    
    // 验证必需参数
    required := []string{"merchant_no", "channel", "trade_type", "out_trade_no", "amount", "subject"}
    for _, key := range required {
        if value, exists := params[key]; !exists || value == "" {
            log.Error("Missing required parameter: %s", key)
            return
        }
    }
    
    // 验证金额格式
    if amount, ok := params["amount"].(int); !ok || amount <= 0 {
        log.Error("Invalid amount: %v", params["amount"])
        return
    }
    
    result := process.New("payment.CreateOrder", params).Run()
    log.Debug("Create order result: %+v", result)
}
```

#### 3. 通知处理问题排查

**问题**：异步通知处理失败
```go
func debugNotifyHandler(merchantNo, channel, notifyData string) {
    log.Debug("Processing notify: merchant=%s, channel=%s", merchantNo, channel)
    log.Debug("Notify data length: %d", len(notifyData))
    
    // 记录原始通知数据（注意脱敏）
    maskedData := maskSensitiveData(notifyData)
    log.Debug("Masked notify data: %s", maskedData)
    
    result := process.New("payment.HandleNotify", merchantNo, channel, notifyData).Run()
    
    if resultMap, ok := result.(map[string]interface{}); ok {
        if success, exists := resultMap["success"]; exists && success.(bool) {
            log.Info("Notify processed successfully")
        } else {
            log.Error("Notify processing failed: %+v", resultMap["error"])
        }
    }
}

func maskSensitiveData(data string) string {
    // 脱敏处理，隐藏敏感信息
    re := regexp.MustCompile(`("sign"|"signature"):"[^"]*"`)
    return re.ReplaceAllString(data, `$1:"***"`)
}
```

#### 4. 性能问题排查

**问题**：支付接口响应慢
```go
func monitorPaymentPerformance(operation string, params map[string]interface{}) interface{} {
    start := time.Now()
    
    result := process.New(operation, params).Run()
    
    duration := time.Since(start)
    log.With(log.F{
        "operation": operation,
        "duration_ms": duration.Milliseconds(),
        "merchant_no": params["merchant_no"],
        "channel": params["channel"],
    }).Info("Payment operation performance")
    
    // 如果响应时间超过阈值，记录警告
    if duration > 5*time.Second {
        log.Warn("Slow payment operation: %s took %v", operation, duration)
    }
    
    return result
}
```

### 监控和告警

#### 1. 关键指标监控

```go
// 支付成功率监控
func trackPaymentMetrics(operation string, success bool, duration time.Duration) {
    // 记录指标到监控系统
    metrics := map[string]interface{}{
        "operation": operation,
        "success": success,
        "duration_ms": duration.Milliseconds(),
        "timestamp": time.Now().Unix(),
    }
    
    // 发送到监控系统（如 Prometheus、InfluxDB 等）
    sendMetrics(metrics)
}
```

#### 2. 告警规则

- **支付成功率低于95%**：立即告警
- **平均响应时间超过3秒**：警告告警
- **连续5次API调用失败**：紧急告警
- **异步通知处理失败率超过1%**：警告告警

#### 3. 健康检查

```go
func healthCheck() map[string]interface{} {
    result := map[string]interface{}{
        "status": "healthy",
        "timestamp": time.Now().Format(time.RFC3339),
        "checks": map[string]interface{}{},
    }
    
    // 检查支付宝连通性
    if err := checkAlipayConnectivity(); err != nil {
        result["checks"].(map[string]interface{})["alipay"] = map[string]interface{}{
            "status": "unhealthy",
            "error": err.Error(),
        }
        result["status"] = "degraded"
    } else {
        result["checks"].(map[string]interface{})["alipay"] = map[string]interface{}{
            "status": "healthy",
        }
    }
    
    // 检查微信支付连通性
    if err := checkWechatConnectivity(); err != nil {
        result["checks"].(map[string]interface{})["wechat"] = map[string]interface{}{
            "status": "unhealthy",
            "error": err.Error(),
        }
        result["status"] = "degraded"
    } else {
        result["checks"].(map[string]interface{})["wechat"] = map[string]interface{}{
            "status": "healthy",
        }
    }
    
    return result
}
```

## 测试

运行单元测试：

```bash
cd payment
go test -v ./...
```

运行覆盖率测试：

```bash
go test -v -cover ./...
```

## 最佳实践

### 1. 安全实践

#### 密钥和证书管理
```go
// ✅ 推荐：使用环境变量存储敏感信息
config := map[string]interface{}{
    "app_id":          os.Getenv("ALIPAY_APP_ID"),
    "private_key":     os.Getenv("ALIPAY_PRIVATE_KEY"),
    "alipay_cert":     os.Getenv("ALIPAY_CERT_PATH"),
    "app_cert":        os.Getenv("ALIPAY_APP_CERT_PATH"),
    "root_cert":       os.Getenv("ALIPAY_ROOT_CERT_PATH"),
}

// ❌ 避免：硬编码敏感信息
config := map[string]interface{}{
    "app_id":      "2021001234567890",  // 不要这样做
    "private_key": "MIIEvQIBADANBgkq...", // 不要这样做
}
```

#### 签名验证
```go
// 始终验证异步通知的签名
func handleNotify(merchantNo, channel, notifyData string) {
    result := process.New("payment.HandleNotify", merchantNo, channel, notifyData).Run()
    
    resultMap := result.(map[string]interface{})
    if !resultMap["success"].(bool) {
        // 签名验证失败，记录日志并拒绝处理
        log.Error("Notify signature verification failed")
        return
    }
    
    // 处理业务逻辑
    processOrderUpdate(resultMap["data"])
}
```

#### 网络安全
```go
// 使用HTTPS和证书验证
config := map[string]interface{}{
    "sandbox":     false,  // 生产环境必须设为false
    "timeout":     30,     // 设置合理的超时时间
    "retry_count": 3,      // 设置重试次数
}
```

### 2. 性能优化

#### 连接池管理
```go
// 复用HTTP连接，避免频繁建立连接
var httpClient = &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
}
```

#### 缓存配置
```go
// 缓存支付配置，避免重复设置
var configCache = sync.Map{}

func getOrSetConfig(merchantNo string, config map[string]interface{}) error {
    if _, exists := configCache.Load(merchantNo); !exists {
        result := process.New("payment.SetConfig", merchantNo, config).Run()
        if handlePaymentResult(result) == nil {
            configCache.Store(merchantNo, config)
        }
        return handlePaymentResult(result)
    }
    return nil
}
```

#### 异步处理
```go
// 对于非关键路径，使用异步处理
func asyncProcessNotify(merchantNo, channel, notifyData string) {
    go func() {
        defer func() {
            if r := recover(); r != nil {
                log.Error("Async notify processing panic: %v", r)
            }
        }()
        
        result := process.New("payment.HandleNotify", merchantNo, channel, notifyData).Run()
        // 处理结果...
    }()
}
```

### 3. 业务流程最佳实践

#### 订单号生成
```go
// 生成唯一的商户订单号
func generateOrderNo(merchantNo string) string {
    timestamp := time.Now().Format("20060102150405")
    random := fmt.Sprintf("%06d", rand.Intn(1000000))
    return fmt.Sprintf("%s%s%s", merchantNo, timestamp, random)
}

// 验证订单号格式
func validateOrderNo(orderNo string) bool {
    // 订单号长度应在6-64位之间
    if len(orderNo) < 6 || len(orderNo) > 64 {
        return false
    }
    
    // 只允许字母、数字、下划线、横线
    matched, _ := regexp.MatchString("^[a-zA-Z0-9_-]+$", orderNo)
    return matched
}
```

#### 金额处理
```go
// 金额统一使用分为单位
func yuanToFen(yuan float64) int {
    return int(math.Round(yuan * 100))
}

func fenToYuan(fen int) float64 {
    return float64(fen) / 100
}

// 验证金额范围
func validateAmount(amount int) error {
    if amount <= 0 {
        return fmt.Errorf("金额必须大于0")
    }
    if amount > 100000000 { // 100万元
        return fmt.Errorf("金额不能超过100万元")
    }
    return nil
}
```

#### 幂等性处理
```go
// 使用Redis实现接口幂等性
func ensureIdempotent(key string, ttl time.Duration, fn func() interface{}) interface{} {
    // 检查是否已处理
    if result := getFromCache(key); result != nil {
        return result
    }
    
    // 设置处理标记
    if !setNXCache(key, "processing", ttl) {
        // 正在处理中，等待结果
        return waitForResult(key, ttl)
    }
    
    // 执行业务逻辑
    result := fn()
    
    // 缓存结果
    setCacheWithTTL(key, result, ttl)
    
    return result
}
```

### 4. 监控和日志

#### 结构化日志
```go
import "github.com/yaoapp/kun/log"

func logPaymentOperation(operation string, params map[string]interface{}, result interface{}, duration time.Duration) {
    log.With(log.F{
        "module":      "payment",
        "operation":   operation,
        "merchant_no": params["merchant_no"],
        "channel":     params["channel"],
        "duration_ms": duration.Milliseconds(),
        "success":     isSuccess(result),
    }).Info("Payment operation completed")
}
```

#### 关键指标收集
```go
// 收集业务指标
type PaymentMetrics struct {
    TotalOrders    int64
    SuccessOrders  int64
    FailedOrders   int64
    TotalAmount    int64
    AvgDuration    time.Duration
}

func collectMetrics(operation string, success bool, amount int, duration time.Duration) {
    // 发送到监控系统
    metrics := map[string]interface{}{
        "operation":   operation,
        "success":     success,
        "amount":      amount,
        "duration_ms": duration.Milliseconds(),
        "timestamp":   time.Now().Unix(),
    }
    
    // 异步发送，避免影响主流程
    go sendToMonitoring(metrics)
}
```

### 5. 测试策略

#### 单元测试
```go
func TestCreateOrder(t *testing.T) {
    // 准备测试数据
    params := map[string]interface{}{
        "merchant_no":   "test_merchant",
        "channel":       "alipay",
        "trade_type":    "qr_code",
        "out_trade_no":  "test_order_001",
        "amount":        1000,
        "subject":       "测试商品",
    }
    
    // 模拟配置
    mockConfig(t, "test_merchant")
    
    // 执行测试
    result := process.New("payment.CreateOrder", params).Run()
    
    // 验证结果
    assert.NotNil(t, result)
    resultMap := result.(map[string]interface{})
    assert.True(t, resultMap["success"].(bool))
}
```

#### 集成测试
```go
func TestPaymentFlow(t *testing.T) {
    if testing.Short() {
        t.Skip("跳过集成测试")
    }
    
    // 1. 设置配置
    setTestConfig(t)
    
    // 2. 创建订单
    orderResult := createTestOrder(t)
    
    // 3. 查询订单
    queryResult := queryTestOrder(t, orderResult["out_trade_no"].(string))
    
    // 4. 验证状态
    assert.Equal(t, "pending", queryResult["status"])
}
```

## 注意事项

### 1. 环境配置

#### 开发环境
- 使用沙箱环境进行开发测试
- 配置测试用的应用ID和密钥
- 确保网络可以访问支付平台的沙箱API

#### 生产环境
- 必须使用正式环境配置
- 定期更新证书和密钥
- 配置生产环境的回调地址
- 启用HTTPS和安全传输

### 2. 数据安全

#### 敏感信息保护
```go
// ✅ 正确的做法
func logOrderInfo(orderNo string, amount int) {
    log.Info("Order created: %s, amount: %d", orderNo, amount)
}

// ❌ 错误的做法 - 不要记录敏感信息
func logPaymentInfo(params map[string]interface{}) {
    log.Info("Payment params: %+v", params) // 可能包含密钥等敏感信息
}
```

#### 数据脱敏
```go
func maskSensitiveFields(data map[string]interface{}) map[string]interface{} {
    masked := make(map[string]interface{})
    for k, v := range data {
        switch k {
        case "private_key", "app_secret", "sign":
            masked[k] = "***"
        case "phone", "id_card":
            if str, ok := v.(string); ok && len(str) > 4 {
                masked[k] = str[:2] + "***" + str[len(str)-2:]
            }
        default:
            masked[k] = v
        }
    }
    return masked
}
```

### 3. 异常处理

#### 网络异常
```go
func handleNetworkError(err error) {
    if strings.Contains(err.Error(), "timeout") {
        // 超时错误，可以重试
        log.Warn("Network timeout, will retry: %v", err)
    } else if strings.Contains(err.Error(), "connection refused") {
        // 连接被拒绝，检查网络配置
        log.Error("Connection refused, check network: %v", err)
    } else {
        // 其他网络错误
        log.Error("Network error: %v", err)
    }
}
```

#### 业务异常
```go
func handleBusinessError(errorCode string) {
    switch errorCode {
    case "ORDER_NOT_EXIST":
        // 订单不存在，可能是订单号错误
        log.Warn("Order not found, check order number")
    case "ORDER_CLOSED":
        // 订单已关闭，需要创建新订单
        log.Info("Order closed, need create new order")
    case "INSUFFICIENT_BALANCE":
        // 余额不足，提示用户充值
        log.Info("Insufficient balance, user need recharge")
    default:
        log.Error("Unknown business error: %s", errorCode)
    }
}
```

### 4. 性能考虑

#### 并发控制
```go
// 使用信号量控制并发数
var semaphore = make(chan struct{}, 10) // 最多10个并发

func processPaymentWithLimit(params map[string]interface{}) interface{} {
    semaphore <- struct{}{} // 获取信号量
    defer func() { <-semaphore }() // 释放信号量
    
    return process.New("payment.CreateOrder", params).Run()
}
```

#### 超时控制
```go
func processWithTimeout(operation string, params map[string]interface{}, timeout time.Duration) (interface{}, error) {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    
    resultChan := make(chan interface{}, 1)
    errorChan := make(chan error, 1)
    
    go func() {
        defer func() {
            if r := recover(); r != nil {
                errorChan <- fmt.Errorf("panic: %v", r)
            }
        }()
        
        result := process.New(operation, params).Run()
        resultChan <- result
    }()
    
    select {
    case result := <-resultChan:
        return result, nil
    case err := <-errorChan:
        return nil, err
    case <-ctx.Done():
        return nil, fmt.Errorf("operation timeout after %v", timeout)
    }
}
```

### 5. 版本兼容性

#### API版本管理
```go
// 支持多个API版本
func getAPIVersion(channel string) string {
    switch channel {
    case "alipay":
        return "1.0" // 支付宝API版本
    case "wechat":
        return "3.0" // 微信支付API版本
    default:
        return "1.0"
    }
}
```

#### 向后兼容
```go
// 保持向后兼容性
func normalizeParams(params map[string]interface{}) map[string]interface{} {
    normalized := make(map[string]interface{})
    
    for k, v := range params {
        // 处理旧版本参数名
        switch k {
        case "order_no": // 旧版本
            normalized["out_trade_no"] = v // 新版本
        case "money": // 旧版本
            normalized["amount"] = v // 新版本
        default:
            normalized[k] = v
        }
    }
    
    return normalized
}
```

### 6. 部署和运维

#### 健康检查
```go
// 实现健康检查接口
func HealthCheck() map[string]interface{} {
    status := "healthy"
    checks := make(map[string]interface{})
    
    // 检查支付宝连通性
    if err := pingAlipay(); err != nil {
        status = "unhealthy"
        checks["alipay"] = map[string]interface{}{
            "status": "down",
            "error":  err.Error(),
        }
    } else {
        checks["alipay"] = map[string]interface{}{
            "status": "up",
        }
    }
    
    // 检查微信支付连通性
    if err := pingWechat(); err != nil {
        status = "unhealthy"
        checks["wechat"] = map[string]interface{}{
            "status": "down",
            "error":  err.Error(),
        }
    } else {
        checks["wechat"] = map[string]interface{}{
            "status": "up",
        }
    }
    
    return map[string]interface{}{
        "status":    status,
        "timestamp": time.Now().Format(time.RFC3339),
        "checks":    checks,
    }
}
```

#### 配置热更新
```go
// 支持配置热更新
func reloadConfig(merchantNo string) error {
    // 从配置中心获取最新配置
    newConfig, err := fetchConfigFromCenter(merchantNo)
    if err != nil {
        return err
    }
    
    // 验证配置有效性
    if err := validateConfig(newConfig); err != nil {
        return err
    }
    
    // 更新配置
    result := process.New("payment.SetConfig", merchantNo, newConfig).Run()
    return handlePaymentResult(result)
}
```

### 7. 合规要求

#### 数据保护
- 遵循GDPR、个人信息保护法等法规要求
- 实施数据最小化原则，只收集必要信息
- 提供数据删除和导出功能
- 定期进行安全审计

#### 支付合规
- 遵循PCI DSS标准
- 实施反洗钱(AML)检查
- 保留交易记录用于审计
- 实施风险控制措施

#### 日志合规
```go
// 合规的日志记录
func logComplianceEvent(eventType string, details map[string]interface{}) {
    // 脱敏处理
    maskedDetails := maskSensitiveFields(details)
    
    log.With(log.F{
        "event_type":  eventType,
        "timestamp":   time.Now().Format(time.RFC3339),
        "details":     maskedDetails,
        "compliance":  true,
    }).Info("Compliance event logged")
}
```

## 扩展开发

### 1. 添加新的支付渠道

要添加新的支付渠道（如银联、PayPal等），需要实现 `PaymentProvider` 接口：

```go
// 1. 在 types.go 中添加新的渠道常量
const (
    PaymentChannelUnionPay = "unionpay"  // 银联支付
    PaymentChannelPayPal   = "paypal"    // PayPal支付
)

// 2. 创建新的提供商实现
type UnionPayProvider struct {
    config map[string]interface{}
}

func (u *UnionPayProvider) CreateOrder(params *CreateOrderParams) (*OrderResult, error) {
    // 实现银联支付订单创建逻辑
    return &OrderResult{
        OutTradeNo:   params.OutTradeNo,
        TradeNo:      generateUnionPayTradeNo(),
        PaymentURL:   buildUnionPayURL(params),
        QRCode:       generateUnionPayQRCode(params),
        Status:       OrderStatusPending,
    }, nil
}

func (u *UnionPayProvider) QueryOrder(outTradeNo string) (*OrderQueryResult, error) {
    // 实现银联支付订单查询逻辑
    return &OrderQueryResult{
        OutTradeNo: outTradeNo,
        Status:     OrderStatusPaid,
        Amount:     1000,
        PaidAt:     time.Now(),
    }, nil
}

// 实现其他必需的接口方法...

// 3. 在 load.go 中注册新的提供商
func registerProviders() {
    manager.RegisterProvider(PaymentChannelAlipay, &AlipayProvider{})
    manager.RegisterProvider(PaymentChannelWechat, &WechatProvider{})
    manager.RegisterProvider(PaymentChannelUnionPay, &UnionPayProvider{})  // 新增
}
```

### 2. 自定义业务逻辑

可以通过钩子函数扩展业务逻辑：

```go
// 定义钩子函数类型
type OrderHook func(params *CreateOrderParams) error
type NotifyHook func(data *NotifyData) error

// 注册钩子函数
var (
    beforeCreateOrderHooks []OrderHook
    afterCreateOrderHooks  []OrderHook
    beforeNotifyHooks      []NotifyHook
    afterNotifyHooks       []NotifyHook
)

// 注册钩子
func RegisterBeforeCreateOrderHook(hook OrderHook) {
    beforeCreateOrderHooks = append(beforeCreateOrderHooks, hook)
}

// 在订单创建时执行钩子
func executeBeforeCreateOrderHooks(params *CreateOrderParams) error {
    for _, hook := range beforeCreateOrderHooks {
        if err := hook(params); err != nil {
            return err
        }
    }
    return nil
}

// 使用示例
func init() {
    // 注册风控检查钩子
    RegisterBeforeCreateOrderHook(func(params *CreateOrderParams) error {
        return checkRiskControl(params)
    })
    
    // 注册订单记录钩子
    RegisterAfterCreateOrderHook(func(params *CreateOrderParams) error {
        return saveOrderToDatabase(params)
    })
}
```

### 3. 插件系统

支持通过插件扩展功能：

```go
// 定义插件接口
type PaymentPlugin interface {
    Name() string
    Version() string
    Init(config map[string]interface{}) error
    BeforeCreateOrder(params *CreateOrderParams) error
    AfterCreateOrder(result *OrderResult) error
    BeforeNotify(data *NotifyData) error
    AfterNotify(data *NotifyData) error
}

// 插件管理器
type PluginManager struct {
    plugins map[string]PaymentPlugin
}

func (pm *PluginManager) RegisterPlugin(plugin PaymentPlugin) error {
    if err := plugin.Init(nil); err != nil {
        return err
    }
    pm.plugins[plugin.Name()] = plugin
    return nil
}

// 风控插件示例
type RiskControlPlugin struct {
    config map[string]interface{}
}

func (r *RiskControlPlugin) Name() string { return "risk_control" }
func (r *RiskControlPlugin) Version() string { return "1.0.0" }

func (r *RiskControlPlugin) BeforeCreateOrder(params *CreateOrderParams) error {
    // 实施风控检查
    if params.Amount > 100000 { // 金额超过1000元需要额外验证
        return fmt.Errorf("金额过大，需要额外验证")
    }
    return nil
}
```

### 4. 数据库集成

提供数据库操作的扩展接口：

```go
// 定义数据存储接口
type PaymentStorage interface {
    SaveOrder(order *Order) error
    GetOrder(outTradeNo string) (*Order, error)
    UpdateOrderStatus(outTradeNo string, status OrderStatus) error
    SaveRefund(refund *Refund) error
    GetRefund(refundNo string) (*Refund, error)
}

// MySQL存储实现
type MySQLStorage struct {
    db *sql.DB
}

func (m *MySQLStorage) SaveOrder(order *Order) error {
    query := `INSERT INTO payment_orders 
              (out_trade_no, trade_no, merchant_no, channel, amount, subject, status, created_at) 
              VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
    
    _, err := m.db.Exec(query, 
        order.OutTradeNo, order.TradeNo, order.MerchantNo, 
        order.Channel, order.Amount, order.Subject, 
        order.Status, order.CreatedAt)
    
    return err
}

// 在配置中指定存储实现
func SetStorage(storage PaymentStorage) {
    manager.storage = storage
}
```

### 5. 事件系统

实现事件驱动的架构：

```go
// 定义事件类型
type EventType string

const (
    EventOrderCreated   EventType = "order.created"
    EventOrderPaid      EventType = "order.paid"
    EventOrderCancelled EventType = "order.cancelled"
    EventRefundCreated  EventType = "refund.created"
    EventRefundSuccess  EventType = "refund.success"
)

// 事件数据结构
type Event struct {
    Type      EventType              `json:"type"`
    Timestamp time.Time              `json:"timestamp"`
    Data      map[string]interface{} `json:"data"`
}

// 事件监听器
type EventListener func(event *Event) error

// 事件管理器
type EventManager struct {
    listeners map[EventType][]EventListener
    mutex     sync.RWMutex
}

func (em *EventManager) Subscribe(eventType EventType, listener EventListener) {
    em.mutex.Lock()
    defer em.mutex.Unlock()
    
    if em.listeners == nil {
        em.listeners = make(map[EventType][]EventListener)
    }
    
    em.listeners[eventType] = append(em.listeners[eventType], listener)
}

func (em *EventManager) Publish(event *Event) {
    em.mutex.RLock()
    listeners := em.listeners[event.Type]
    em.mutex.RUnlock()
    
    for _, listener := range listeners {
        go func(l EventListener) {
            if err := l(event); err != nil {
                log.Error("Event listener error: %v", err)
            }
        }(listener)
    }
}

// 使用示例
func init() {
    eventManager.Subscribe(EventOrderPaid, func(event *Event) error {
        // 发送支付成功通知
        return sendPaymentNotification(event.Data)
    })
    
    eventManager.Subscribe(EventOrderPaid, func(event *Event) error {
        // 更新用户积分
        return updateUserPoints(event.Data)
    })
}
```

### 6. 配置中心集成

支持从配置中心动态获取配置：

```go
// 配置中心接口
type ConfigCenter interface {
    GetConfig(key string) (map[string]interface{}, error)
    WatchConfig(key string, callback func(config map[string]interface{})) error
}

// Consul配置中心实现
type ConsulConfigCenter struct {
    client *consul.Client
}

func (c *ConsulConfigCenter) GetConfig(key string) (map[string]interface{}, error) {
    kv := c.client.KV()
    pair, _, err := kv.Get(key, nil)
    if err != nil {
        return nil, err
    }
    
    var config map[string]interface{}
    if err := json.Unmarshal(pair.Value, &config); err != nil {
        return nil, err
    }
    
    return config, nil
}

func (c *ConsulConfigCenter) WatchConfig(key string, callback func(config map[string]interface{})) error {
    // 实现配置监听逻辑
    go func() {
        for {
            config, err := c.GetConfig(key)
            if err == nil {
                callback(config)
            }
            time.Sleep(30 * time.Second) // 每30秒检查一次
        }
    }()
    return nil
}

// 集成配置中心
func InitWithConfigCenter(configCenter ConfigCenter) error {
    // 监听配置变化
    return configCenter.WatchConfig("payment/config", func(config map[string]interface{}) {
        // 更新支付配置
        for merchantNo, merchantConfig := range config {
            if cfg, ok := merchantConfig.(map[string]interface{}); ok {
                process.New("payment.SetConfig", merchantNo, cfg).Run()
            }
        }
    })
}
```

### 7. 监控指标扩展

提供自定义监控指标：

```go
// 指标收集器接口
type MetricsCollector interface {
    Counter(name string, tags map[string]string) Counter
    Gauge(name string, tags map[string]string) Gauge
    Histogram(name string, tags map[string]string) Histogram
}

// Prometheus指标收集器
type PrometheusCollector struct {
    registry *prometheus.Registry
}

func (p *PrometheusCollector) Counter(name string, tags map[string]string) Counter {
    counter := prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: name},
        getTagKeys(tags),
    )
    p.registry.MustRegister(counter)
    return &PrometheusCounter{counter: counter, tags: tags}
}

// 使用示例
func collectPaymentMetrics(operation string, channel string, success bool) {
    tags := map[string]string{
        "operation": operation,
        "channel":   channel,
        "status":    getStatusString(success),
    }
    
    // 计数器
    metricsCollector.Counter("payment_operations_total", tags).Inc()
    
    // 成功率
    if success {
        metricsCollector.Counter("payment_operations_success_total", tags).Inc()
    }
}
```

## 贡献指南

### 1. 开发环境设置

```bash
cd yao/payment

# 2. 安装依赖
go mod tidy

# 3. 运行测试
go test -v ./...

# 4. 代码格式化
go fmt ./...

# 5. 静态检查
go vet ./...
```

---

**最后更新时间**：2025-03-24  
**文档版本**：v1.0.0