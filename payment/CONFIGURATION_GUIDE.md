# Payment 模块配置管理指南

## 📋 目录

1. [配置方式概述](#配置方式概述)
2. [CertConfig 结构说明](#certconfig-结构说明)
3. [支付宝配置](#支付宝配置)
4. [微信配置](#微信配置)
5. [Process API 使用](#process-api-使用)
6. [实战场景](#实战场景)

---

## 配置方式概述

Payment 模块提供**两种配置方式**：

### 方式 1：文件自动加载（推荐用于证书）

将证书文件放置在指定目录，启动时自动加载：

```bash
certs/
  └── {merchant_id}/
      ├── alipay/
      │   ├── private_key.pem      # 必需
      │   ├── public_key.pem       # 必需
      │   ├── app_cert.crt         # 可选
      │   └── alipay_root_cert.crt # 可选
      └── wechat/
          ├── private_key.pem      # 必需
          ├── public_key.pem       # 必需
          └── apiclient_cert.pem   # 可选
```

### 方式 2：Process 动态配置（推荐用于非证书参数）

使用 `payment.SetMerchantConfig` Process 设置配置：

```javascript
// Yao DSL 中
process.Run("payment.SetMerchantConfig", "merchant001", "alipay", {
    "app_id": "2021001234567890",
    "sign_type": "RSA2",
    "is_sandbox": false
})
```

### 方式 3：混合配置（最灵活）

文件提供证书，Process 补充其他参数：

1. 证书文件自动加载
2. Process 补充 app_id、mch_id 等参数
3. 配置自动合并

---

## CertConfig 结构说明

`CertConfig` 是统一的配置结构，支持所有配置参数：

```go
type CertConfig struct {
    // 基本信息
    MerchantID string         // 商户ID
    Channel    PaymentChannel // 支付渠道 (alipay/wechat)
    
    // 证书文件（从文件或 Process 加载）
    PrivateKey string // 私钥内容
    PublicKey  string // 公钥内容
    AppCert    string // 应用公钥证书（支付宝）
    RootCert   string // 支付宝根证书
    
    // 支付宝特有配置
    AppID     string // 应用ID (必需)
    SignType  string // 签名类型 (RSA2/RSA，可选，默认RSA2)
    IsSandbox bool   // 是否沙箱环境 (可选，默认false)
    
    // 微信特有配置
    MchID    string // 商户号 (必需)
    APIv3Key string // APIv3密钥 (必需)
    SerialNo string // 证书序列号 (必需)
    
    // 扩展字段
    ExtraFields map[string]interface{} // 其他任意配置
    ExtraFiles  map[string]string      // 其他证书文件
}
```

---

## 支付宝配置

### 必需参数

| 参数 | 说明 | 来源 |
|------|------|------|
| `app_id` | 应用ID | Process 设置 |
| `private_key` | 应用私钥 | 文件或 Process |
| `public_key` | 支付宝公钥 | 文件或 Process |

### 可选参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `sign_type` | 签名类型 | RSA2 |
| `is_sandbox` | 沙箱环境 | false |
| `app_cert` | 应用公钥证书 | - |
| `root_cert` | 根证书 | - |

### 完整配置示例

#### 方式 A：纯文件配置（需补充 app_id）

```bash
# 1. 放置证书文件
certs/merchant001/alipay/private_key.pem
certs/merchant001/alipay/public_key.pem
```

```javascript
// 2. 补充 app_id
process.Run("payment.SetMerchantConfig", "merchant001", "alipay", {
    "app_id": "2021001234567890"
})
```

#### 方式 B：完全 Process 配置

```javascript
process.Run("payment.SetMerchantConfig", "merchant001", "alipay", {
    "app_id": "2021001234567890",
    "private_key": "-----BEGIN RSA PRIVATE KEY-----\nMII...\n-----END RSA PRIVATE KEY-----",
    "public_key": "-----BEGIN PUBLIC KEY-----\nMII...\n-----END PUBLIC KEY-----",
    "sign_type": "RSA2",
    "is_sandbox": false
})
```

#### 方式 C：混合配置（推荐）

```bash
# 1. 证书文件
certs/merchant001/alipay/private_key.pem
certs/merchant001/alipay/public_key.pem
certs/merchant001/alipay/app_cert.crt           # 可选
certs/merchant001/alipay/alipay_root_cert.crt   # 可选
```

```javascript
// 2. 补充业务参数
process.Run("payment.SetMerchantConfig", "merchant001", "alipay", {
    "app_id": "2021001234567890",
    "is_sandbox": false  // 生产环境
})
```

---

## 微信配置

### 必需参数

| 参数 | 说明 | 来源 |
|------|------|------|
| `app_id` | 应用ID | Process 设置 |
| `mch_id` | 商户号 | Process 设置 |
| `apiv3_key` | APIv3密钥 | Process 设置 |
| `private_key` | 商户私钥 | 文件或 Process |
| `serial_no` | 证书序列号 | Process 设置 |

### 可选参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `is_sandbox` | 沙箱环境 | false |
| `public_key` | 微信公钥 | - |

### 完整配置示例

#### 方式 A：纯文件配置（需补充参数）

```bash
# 1. 放置证书文件
certs/merchant002/wechat/private_key.pem
certs/merchant002/wechat/public_key.pem
```

```javascript
// 2. 补充必需参数
process.Run("payment.SetMerchantConfig", "merchant002", "wechat", {
    "app_id": "wx1234567890abcdef",
    "mch_id": "1234567890",
    "apiv3_key": "WECHATPAY2APIKEY32BYTESLONGSECRET",
    "serial_no": "1234567890ABCDEF1234567890ABCDEF12345678"
})
```

#### 方式 B：完全 Process 配置

```javascript
process.Run("payment.SetMerchantConfig", "merchant002", "wechat", {
    "app_id": "wx1234567890abcdef",
    "mch_id": "1234567890",
    "apiv3_key": "WECHATPAY2APIKEY32BYTESLONGSECRET",
    "private_key": "-----BEGIN PRIVATE KEY-----\nMII...\n-----END PRIVATE KEY-----",
    "serial_no": "1234567890ABCDEF1234567890ABCDEF12345678",
    "is_sandbox": false
})
```

#### 方式 C：混合配置（推荐）

```bash
# 1. 证书文件
certs/merchant002/wechat/private_key.pem
certs/merchant002/wechat/public_key.pem  
certs/merchant002/wechat/apiclient_cert.pem  # 可选
```

```javascript
// 2. 补充业务参数
process.Run("payment.SetMerchantConfig", "merchant002", "wechat", {
    "app_id": "wx1234567890abcdef",
    "mch_id": "1234567890",
    "apiv3_key": "WECHATPAY2APIKEY32BYTESLONGSECRET",
    "serial_no": "1234567890ABCDEF1234567890ABCDEF12345678"
})
```

---

## Process API 使用

### payment.SetMerchantConfig

设置或更新商户配置。

**签名：**
```
payment.SetMerchantConfig(merchantID: string, channel: string, config: map)
```

**参数：**
- `merchantID`: 商户ID
- `channel`: 支付渠道 (`alipay` 或 `wechat`)
- `config`: 配置参数 map

**返回：** `null` 或抛出异常

**示例：**
```javascript
// 设置支付宝配置
process.Run("payment.SetMerchantConfig", "merchant001", "alipay", {
    "app_id": "2021001234567890",
    "private_key": "...",
    "public_key": "...",
    "is_sandbox": true
})

// 设置微信配置
process.Run("payment.SetMerchantConfig", "merchant002", "wechat", {
    "app_id": "wx1234567890",
    "mch_id": "1234567890",
    "apiv3_key": "...",
    "private_key": "...",
    "serial_no": "..."
})

// 增量更新（只更新指定字段）
process.Run("payment.SetMerchantConfig", "merchant001", "alipay", {
    "is_sandbox": false  // 切换到生产环境，其他配置保留
})
```

---

### payment.GetMerchantConfig

获取商户配置。

**签名：**
```
payment.GetMerchantConfig(merchantID: string, channel: string): map
```

**参数：**
- `merchantID`: 商户ID
- `channel`: 支付渠道 (`alipay` 或 `wechat`)

**返回：** 配置参数 map

**示例：**
```javascript
// 获取支付宝配置
var config = process.Run("payment.GetMerchantConfig", "merchant001", "alipay")
console.log("App ID:", config.app_id)
console.log("Is Sandbox:", config.is_sandbox)
console.log("Has Private Key:", config.private_key !== "")

// 获取微信配置
var wxConfig = process.Run("payment.GetMerchantConfig", "merchant002", "wechat")
console.log("Mch ID:", wxConfig.mch_id)
console.log("Serial No:", wxConfig.serial_no)
```

---

### payment.ListMerchants

列出所有已配置的商户。

**签名：**
```
payment.ListMerchants(): []map
```

**返回：** 商户列表

**示例：**
```javascript
var merchants = process.Run("payment.ListMerchants")
console.log("已配置商户数量:", merchants.length)

merchants.forEach(function(m) {
    console.log("商户:", m.merchant_id, "渠道:", m.channel, "已配置证书:", m.has_cert)
})

// 输出示例:
// 已配置商户数量: 2
// 商户: merchant001 渠道: alipay 已配置证书: true
// 商户: merchant002 渠道: wechat 已配置证书: true
```

---

## 实战场景

### 场景 1：多商户 SaaS 平台

**需求：** 每个商户使用独立的支付配置

**方案：**
1. 商户入驻时，通过 Process 设置配置
2. 证书文件可选（可使用 base64 传递）

```javascript
// 商户入驻时
function onMerchantRegister(merchantInfo) {
    // 设置支付宝
    process.Run("payment.SetMerchantConfig", merchantInfo.id, "alipay", {
        "app_id": merchantInfo.alipay_app_id,
        "private_key": merchantInfo.alipay_private_key,
        "public_key": merchantInfo.alipay_public_key,
        "is_sandbox": false
    })
    
    // 设置微信
    process.Run("payment.SetMerchantConfig", merchantInfo.id, "wechat", {
        "app_id": merchantInfo.wechat_app_id,
        "mch_id": merchantInfo.wechat_mch_id,
        "apiv3_key": merchantInfo.wechat_apiv3_key,
        "private_key": merchantInfo.wechat_private_key,
        "serial_no": merchantInfo.wechat_serial_no
    })
}

// 创建订单时自动使用对应商户配置
process.Run("payment.CreateOrder", {
    "merchant_no": "merchant123",  // 自动查找配置
    "channel": "alipay",
    "amount": 10000,
    "subject": "商品购买"
})
```

---

### 场景 2：开发/测试/生产环境切换

**需求：** 不同环境使用不同配置

**方案：**
```javascript
// 环境配置
var env = process.Env("YAO_ENV")  // development/production

// 根据环境设置配置
if (env === "development") {
    // 开发环境：使用沙箱
    process.Run("payment.SetMerchantConfig", "default", "alipay", {
        "app_id": "2021001234567890",
        "is_sandbox": true,  // 沙箱模式
        "private_key": loadFile("dev_keys/alipay_private.pem"),
        "public_key": loadFile("dev_keys/alipay_public.pem")
    })
} else {
    // 生产环境：使用正式配置
    process.Run("payment.SetMerchantConfig", "default", "alipay", {
        "app_id": "2021009876543210",
        "is_sandbox": false,  // 生产模式
        "private_key": loadFile("prod_keys/alipay_private.pem"),
        "public_key": loadFile("prod_keys/alipay_public.pem")
    })
}
```

---

### 场景 3：动态更新配置（热更新）

**需求：** 商户修改配置后立即生效

**方案：**
```javascript
// 商户修改配置
function updateMerchantPaymentConfig(merchantID, channel, newConfig) {
    try {
        // 更新配置
        process.Run("payment.SetMerchantConfig", merchantID, channel, newConfig)
        
        // Provider 缓存自动失效，下次创建订单时使用新配置
        return {
            success: true,
            message: "配置更新成功，立即生效"
        }
    } catch (e) {
        return {
            success: false,
            error: e.message
        }
    }
}
```

---

### 场景 4：证书过期更新

**需求：** 定期更新证书文件

**方案：**

#### 方式 A：文件更新 + 重启
```bash
# 1. 替换证书文件
cp new_private_key.pem certs/merchant001/alipay/private_key.pem
cp new_public_key.pem certs/merchant001/alipay/public_key.pem

# 2. 重启应用（自动重新加载）
./yao restart
```

#### 方式 B：Process 热更新（无需重启）
```javascript
// 读取新证书内容
var newPrivateKey = loadFile("new_private_key.pem")
var newPublicKey = loadFile("new_public_key.pem")

// 更新配置
process.Run("payment.SetMerchantConfig", "merchant001", "alipay", {
    "private_key": newPrivateKey,
    "public_key": newPublicKey
})

// 立即生效，无需重启
```

---

### 场景 5：配置验证和检查

**需求：** 检查商户配置是否完整

**方案：**
```javascript
function validateMerchantConfig(merchantID, channel) {
    try {
        // 获取配置
        var config = process.Run("payment.GetMerchantConfig", merchantID, channel)
        
        var errors = []
        
        // 检查必需字段
        if (channel === "alipay") {
            if (!config.app_id) errors.push("缺少 app_id")
            if (!config.private_key) errors.push("缺少 private_key")
            if (!config.public_key) errors.push("缺少 public_key")
        } else if (channel === "wechat") {
            if (!config.app_id) errors.push("缺少 app_id")
            if (!config.mch_id) errors.push("缺少 mch_id")
            if (!config.apiv3_key) errors.push("缺少 apiv3_key")
            if (!config.private_key) errors.push("缺少 private_key")
            if (!config.serial_no) errors.push("缺少 serial_no")
        }
        
        return {
            valid: errors.length === 0,
            errors: errors,
            config: config
        }
    } catch (e) {
        return {
            valid: false,
            errors: ["配置不存在: " + e.message]
        }
    }
}

// 使用
var result = validateMerchantConfig("merchant001", "alipay")
if (!result.valid) {
    console.log("配置不完整:", result.errors)
} else {
    console.log("配置完整，可以使用")
}
```

---

## 🎯 最佳实践

### 1. 证书安全
- ✅ 生产环境证书文件放在安全目录，设置严格权限
- ✅ 不要将证书提交到版本控制系统
- ✅ 使用环境变量或密钥管理服务存储敏感配置

### 2. 配置管理
- ✅ 证书文件适合文件加载
- ✅ 业务参数（app_id, mch_id）适合 Process 设置
- ✅ 使用混合配置方式，灵活性最高

### 3. 错误处理
- ✅ 配置前先验证必需参数
- ✅ 使用 try-catch 捕获配置错误
- ✅ 提供清晰的错误提示

### 4. 性能优化
- ✅ Provider 自动缓存，重复创建订单性能好
- ✅ 配置更新后自动清除缓存
- ✅ 避免频繁更新配置

---

## 📚 相关文档

- [SIMPLIFY_ARCHITECTURE.md](./SIMPLIFY_ARCHITECTURE.md) - 架构简化说明
- [FIX_SUMMARY.md](./FIX_SUMMARY.md) - 工厂模式重构总结
- [CERTIFICATE_AUTO_LOAD.md](./CERTIFICATE_AUTO_LOAD.md) - 证书自动加载文档

---

**最后更新**：2025-01-14  
**版本**：v1.0  
**作者**：Payment Team
