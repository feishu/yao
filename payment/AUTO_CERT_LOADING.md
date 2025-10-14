# Payment 模块自动证书加载功能

## 概述

Payment模块现在支持在启动时自动扫描并加载证书文件。你只需要按照规定的目录结构放置证书文件，系统会自动加载并缓存，无需手动编写加载代码。

**核心特性：**
- ✅ 启动时自动扫描 `certs/` 目录
- ✅ 支持多商户、多渠道
- ✅ 自动识别支付宝和微信支付
- ✅ 缓存证书配置，高性能访问
- ✅ 支持额外的证书文件（可选）

---

## 目录结构规范

### 标准结构

```
your-yao-app/
├── certs/                              # 证书根目录
│   ├── merchant_001/                   # 商户ID目录
│   │   ├── alipay/                     # 支付宝渠道
│   │   │   ├── private_key.pem         # ✅ 必需：应用私钥
│   │   │   ├── public_key.pem          # ✅ 必需：支付宝公钥
│   │   │   ├── app_cert.crt            # 可选：应用公钥证书（证书模式）
│   │   │   ├── alipay_cert.crt         # 可选：支付宝公钥证书
│   │   │   └── alipay_root_cert.crt    # 可选：支付宝根证书
│   │   └── wechat/                     # 微信支付渠道
│   │       ├── private_key.pem         # ✅ 必需：商户API私钥
│   │       ├── public_key.pem          # ✅ 必需：微信支付公钥
│   │       ├── apiclient_cert.pem      # 可选：商户证书
│   │       └── apiclient_cert.p12      # 可选：PKCS#12格式证书
│   ├── merchant_002/                   # 第二个商户
│   │   ├── alipay/
│   │   │   ├── private_key.pem
│   │   │   └── public_key.pem
│   │   └── wechat/
│   │       ├── private_key.pem
│   │       └── public_key.pem
│   └── .gitignore                      # ⚠️ 重要：排除证书文件
├── apis/
├── models/
└── scripts/
```

### 目录命名规则

**商户ID目录：** 任意合法的目录名
- 示例：`merchant_001`, `shop_main`, `company_A`

**渠道目录：** 必须是以下之一（不区分大小写）

| 渠道名称 | 支持的目录名 |
|---------|------------|
| 支付宝   | `alipay`, `支付宝` |
| 微信支付 | `wechat`, `wechatpay`, `wxpay`, `微信`, `微信支付` |

### 必需文件

每个渠道目录必须包含：
1. **`private_key.pem`** - 应用/商户私钥
2. **`public_key.pem`** - 平台公钥

---

## 自动加载流程

### 启动时加载

当Yao应用启动时，Payment模块会自动：

1. **扫描** `certs/` 目录下的所有商户和渠道
2. **读取** 每个渠道下的证书文件
3. **验证** 文件格式和完整性
4. **缓存** 证书配置到内存

**日志输出示例：**

```
[INFO] Loading payment module...
[INFO] Loading certificates from directory: /path/to/app/certs
[DEBUG] Scanning merchant directory: merchant_001
[DEBUG] Loading certificates for merchant_001/alipay
[DEBUG] Loaded app_cert.crt for merchant_001/alipay
[INFO] ✓ Loaded certificates for merchant: merchant_001, channel: alipay
[DEBUG] Loading certificates for merchant_001/wechat
[INFO] ✓ Loaded certificates for merchant: merchant_001, channel: wechat
[INFO] Certificate auto-loading completed: 2 configurations loaded
[INFO] Payment module loaded successfully
```

---

## 使用已加载的证书

### 方式1：通过Process获取证书（推荐）

```javascript
/**
 * 使用自动加载的证书配置支付渠道
 */
function setupPaymentWithAutoCerts() {
    const merchantID = "merchant_001"
    
    try {
        // 获取支付宝证书
        const alipayCert = Process("payment.GetCertificate", merchantID, "alipay")
        
        if (!alipayCert.success) {
            throw new Error("Alipay certificate not loaded")
        }
        
        // 使用自动加载的证书配置支付宝
        Process("payment.SetConfig", merchantID, "alipay", {
            app_id: "2021001234567890",
            private_key: alipayCert.private_key,  // 来自自动加载
            public_key: alipayCert.public_key,    // 来自自动加载
            is_sandbox: false,
            sign_type: "RSA2"
        })
        
        console.log("✓ Alipay configured with auto-loaded certs")
        
        // 获取微信支付证书
        const wechatCert = Process("payment.GetCertificate", merchantID, "wechat")
        
        if (!wechatCert.success) {
            throw new Error("WeChat certificate not loaded")
        }
        
        // 使用自动加载的证书配置微信支付
        Process("payment.SetConfig", merchantID, "wechat", {
            app_id: "wx1234567890abcdef",
            mch_id: "1234567890",
            apiv3_key: Process("utils.env.Get", "WECHAT_APIV3_KEY"),
            private_key: wechatCert.private_key,  // 来自自动加载
            serial_no: Process("utils.env.Get", "WECHAT_SERIAL_NO"),
            is_sandbox: false
        })
        
        console.log("✓ WeChat Pay configured with auto-loaded certs")
        
        return {
            success: true,
            message: "Payment channels configured successfully"
        }
        
    } catch (error) {
        console.error("Failed to setup payment:", error.message)
        return {
            success: false,
            error: error.message
        }
    }
}
```

### 方式2：列出所有已加载的证书

```javascript
/**
 * 查看所有自动加载的证书配置
 */
function listLoadedCertificates() {
    const result = Process("payment.ListCertificates")
    
    console.log("=== Loaded Certificates ===")
    console.log(`Total: ${result.count} configurations`)
    
    for (const cert of result.certs) {
        console.log(`\n- Merchant: ${cert.merchant_id}`)
        console.log(`  Channel: ${cert.channel}`)
        console.log(`  Has App Cert: ${cert.has_app_cert}`)
        console.log(`  Has Root Cert: ${cert.has_root_cert}`)
        console.log(`  Extra Files: ${cert.extra_files}`)
    }
    
    return result
}
```

### 方式3：批量配置所有商户

```javascript
/**
 * 自动配置所有已加载证书的商户
 */
function autoConfigureAllMerchants() {
    const certsResult = Process("payment.ListCertificates")
    
    if (!certsResult.success || certsResult.count === 0) {
        console.log("No certificates loaded, skipping auto-configuration")
        return
    }
    
    console.log(`Auto-configuring ${certsResult.count} merchant-channel combinations...`)
    
    const results = []
    
    for (const certInfo of certsResult.certs) {
        try {
            // 获取证书详细信息
            const cert = Process("payment.GetCertificate", 
                certInfo.merchant_id, 
                certInfo.channel
            )
            
            // 根据渠道配置
            if (certInfo.channel === "alipay") {
                // 从环境变量或配置文件读取AppID
                const appID = Process("utils.env.Get", 
                    `ALIPAY_APP_ID_${certInfo.merchant_id.toUpperCase()}`
                ) || "默认AppID"
                
                Process("payment.SetConfig", certInfo.merchant_id, "alipay", {
                    app_id: appID,
                    private_key: cert.private_key,
                    public_key: cert.public_key,
                    is_sandbox: false
                })
                
            } else if (certInfo.channel === "wechat") {
                // 从环境变量读取微信配置
                const envPrefix = `WECHAT_${certInfo.merchant_id.toUpperCase()}`
                
                Process("payment.SetConfig", certInfo.merchant_id, "wechat", {
                    app_id: Process("utils.env.Get", `${envPrefix}_APP_ID`),
                    mch_id: Process("utils.env.Get", `${envPrefix}_MCH_ID`),
                    apiv3_key: Process("utils.env.Get", `${envPrefix}_APIV3_KEY`),
                    private_key: cert.private_key,
                    serial_no: Process("utils.env.Get", `${envPrefix}_SERIAL_NO`),
                    is_sandbox: false
                })
            }
            
            results.push({
                merchant_id: certInfo.merchant_id,
                channel: certInfo.channel,
                success: true
            })
            
            console.log(`✓ Configured ${certInfo.merchant_id}/${certInfo.channel}`)
            
        } catch (error) {
            results.push({
                merchant_id: certInfo.merchant_id,
                channel: certInfo.channel,
                success: false,
                error: error.message
            })
            
            console.error(`✗ Failed to configure ${certInfo.merchant_id}/${certInfo.channel}: ${error.message}`)
        }
    }
    
    return {
        success: results.every(r => r.success),
        total: results.length,
        results: results
    }
}
```

---

## Process API 参考

### payment.GetCertificate

获取自动加载的证书配置。

**参数：**
- `merchantID` (string): 商户ID
- `channel` (string): 支付渠道（alipay/wechat）

**返回值：**
```javascript
{
    success: true,
    merchant_id: "merchant_001",
    channel: "alipay",
    private_key: "-----BEGIN PRIVATE KEY-----\n...",
    public_key: "-----BEGIN PUBLIC KEY-----\n...",
    extra_files: {
        app_cert: "...",           // 可选
        alipay_cert: "...",         // 可选
        alipay_root_cert: "..."     // 可选
    }
}
```

**异常：**
- `400` - 参数错误
- `404` - 证书配置未找到

---

### payment.ListCertificates

列出所有自动加载的证书配置。

**参数：** 无

**返回值：**
```javascript
{
    success: true,
    count: 2,
    certs: [
        {
            key: "merchant_001_alipay",
            merchant_id: "merchant_001",
            channel: "alipay",
            has_app_cert: true,
            has_root_cert: true,
            extra_files: 3
        },
        {
            key: "merchant_001_wechat",
            merchant_id: "merchant_001",
            channel: "wechat",
            has_app_cert: false,
            has_root_cert: false,
            extra_files: 1
        }
    ]
}
```

---

## 与手动加载的对比

### 之前：手动加载（仍然支持）

```javascript
// 需要手动调用LoadCert加载每个文件
const alipayPrivKey = Process("payment.LoadCert", "certs/merchant_001/alipay/private_key.pem")
const alipayPubKey = Process("payment.LoadCert", "certs/merchant_001/alipay/public_key.pem")

Process("payment.SetConfig", "merchant_001", "alipay", {
    app_id: "2021001234567890",
    private_key: alipayPrivKey.content,
    public_key: alipayPubKey.content,
    is_sandbox: false
})
```

### 现在：自动加载（推荐）

```javascript
// 证书已在启动时自动加载，直接获取即可
const cert = Process("payment.GetCertificate", "merchant_001", "alipay")

Process("payment.SetConfig", "merchant_001", "alipay", {
    app_id: "2021001234567890",
    private_key: cert.private_key,  // 直接使用
    public_key: cert.public_key,    // 直接使用
    is_sandbox: false
})
```

**优势：**
- ✅ 代码更简洁
- ✅ 启动时验证证书完整性
- ✅ 避免重复读取文件
- ✅ 集中管理证书路径
- ✅ 支持热重载（可扩展）

---

## 环境变量配置建议

为了保护敏感配置信息，建议使用环境变量：

### .env 文件示例

```bash
# Merchant 001 - Alipay
ALIPAY_APP_ID_MERCHANT_001=2021001234567890

# Merchant 001 - WeChat Pay
WECHAT_MERCHANT_001_APP_ID=wx1234567890abcdef
WECHAT_MERCHANT_001_MCH_ID=1234567890
WECHAT_MERCHANT_001_APIV3_KEY=your_32_character_apiv3_key_here
WECHAT_MERCHANT_001_SERIAL_NO=ABC123456789DEF

# Merchant 002 - Alipay
ALIPAY_APP_ID_MERCHANT_002=2021009876543210

# Merchant 002 - WeChat Pay
WECHAT_MERCHANT_002_APP_ID=wx9876543210fedcba
WECHAT_MERCHANT_002_MCH_ID=9876543210
WECHAT_MERCHANT_002_APIV3_KEY=another_32_character_key_goes_here
WECHAT_MERCHANT_002_SERIAL_NO=XYZ987654321ABC
```

### 在脚本中读取环境变量

```javascript
function getEnvConfig(merchantID, channel) {
    if (channel === "alipay") {
        return {
            app_id: Process("utils.env.Get", 
                `ALIPAY_APP_ID_${merchantID.toUpperCase()}`
            )
        }
    }
    
    if (channel === "wechat") {
        const prefix = `WECHAT_${merchantID.toUpperCase()}`
        return {
            app_id: Process("utils.env.Get", `${prefix}_APP_ID`),
            mch_id: Process("utils.env.Get", `${prefix}_MCH_ID`),
            apiv3_key: Process("utils.env.Get", `${prefix}_APIV3_KEY`),
            serial_no: Process("utils.env.Get", `${prefix}_SERIAL_NO`)
        }
    }
}
```

---

## 完整配置示例

### 目录设置

```bash
# 创建证书目录结构
mkdir -p certs/merchant_001/{alipay,wechat}
mkdir -p certs/merchant_002/{alipay,wechat}

# 复制证书文件
cp /path/to/merchant_001_alipay_private.pem certs/merchant_001/alipay/private_key.pem
cp /path/to/merchant_001_alipay_public.pem certs/merchant_001/alipay/public_key.pem
cp /path/to/merchant_001_wechat_private.pem certs/merchant_001/wechat/private_key.pem
cp /path/to/merchant_001_wechat_public.pem certs/merchant_001/wechat/public_key.pem

# 设置权限
chmod 600 certs/**/*.pem
chmod 700 certs/*
```

### 初始化脚本 (scripts/init/payment.js)

```javascript
/**
 * Payment 模块初始化脚本
 * 在应用启动后自动配置所有支付渠道
 */

// 查看已加载的证书
const loaded = Process("payment.ListCertificates")
console.log(`Payment module initialized with ${loaded.count} certificate configurations`)

// 配置每个商户的支付渠道
function configurePaymentChannels() {
    const configs = [
        {
            merchant_id: "merchant_001",
            alipay: { app_id: "2021001234567890", sandbox: false },
            wechat: {
                app_id: "wx1234567890",
                mch_id: "1234567890",
                apiv3_key: Process("utils.env.Get", "WECHAT_001_APIV3_KEY"),
                serial_no: Process("utils.env.Get", "WECHAT_001_SERIAL_NO"),
                sandbox: false
            }
        },
        {
            merchant_id: "merchant_002",
            alipay: { app_id: "2021009876543210", sandbox: false },
            wechat: {
                app_id: "wx9876543210",
                mch_id: "9876543210",
                apiv3_key: Process("utils.env.Get", "WECHAT_002_APIV3_KEY"),
                serial_no: Process("utils.env.Get", "WECHAT_002_SERIAL_NO"),
                sandbox: false
            }
        }
    ]
    
    for (const merchantConfig of configs) {
        const merchantID = merchantConfig.merchant_id
        
        // 配置支付宝
        try {
            const alipayCert = Process("payment.GetCertificate", merchantID, "alipay")
            
            Process("payment.SetConfig", merchantID, "alipay", {
                app_id: merchantConfig.alipay.app_id,
                private_key: alipayCert.private_key,
                public_key: alipayCert.public_key,
                is_sandbox: merchantConfig.alipay.sandbox
            })
            
            console.log(`✓ ${merchantID}/alipay configured`)
        } catch (error) {
            console.error(`✗ ${merchantID}/alipay failed:`, error.message)
        }
        
        // 配置微信
        try {
            const wechatCert = Process("payment.GetCertificate", merchantID, "wechat")
            
            Process("payment.SetConfig", merchantID, "wechat", {
                app_id: merchantConfig.wechat.app_id,
                mch_id: merchantConfig.wechat.mch_id,
                apiv3_key: merchantConfig.wechat.apiv3_key,
                private_key: wechatCert.private_key,
                serial_no: merchantConfig.wechat.serial_no,
                is_sandbox: merchantConfig.wechat.sandbox
            })
            
            console.log(`✓ ${merchantID}/wechat configured`)
        } catch (error) {
            console.error(`✗ ${merchantID}/wechat failed:`, error.message)
        }
    }
    
    console.log("Payment channels configuration completed")
}

// 执行配置
configurePaymentChannels()

// 导出
exports.configurePaymentChannels = configurePaymentChannels
```

---

## 故障排查

### 问题1：证书未自动加载

**可能原因：**
- `certs/` 目录不存在
- 目录结构不正确
- 文件名不正确（必须是 `private_key.pem` 和 `public_key.pem`）
- 文件权限问题

**解决方法：**
```bash
# 检查目录结构
ls -la certs/*/

# 检查日志
# 启动时应该看到类似的日志：
# [INFO] Loading certificates from directory: /path/to/app/certs
# [INFO] Certificate auto-loading completed: N configurations loaded
```

### 问题2：获取证书时返回404

**可能原因：**
- 商户ID或渠道名称错误
- 证书未成功加载

**解决方法：**
```javascript
// 列出所有已加载的证书
const certs = Process("payment.ListCertificates")
console.log(certs)

// 检查特定商户的证书
try {
    const cert = Process("payment.GetCertificate", "merchant_001", "alipay")
    console.log("Certificate found!")
} catch (error) {
    console.error("Certificate not found:", error.message)
}
```

### 问题3：渠道目录未识别

**症状：** 日志显示 "Unknown channel directory: xxx, skipping"

**解决方法：** 确保渠道目录名是以下之一：
- 支付宝：`alipay` 或 `支付宝`
- 微信：`wechat`, `wechatpay`, `wxpay`, `微信`, `微信支付`

---

## 安全建议

### 1. 保护证书文件

```bash
# .gitignore
certs/
*.pem
*.crt
*.key
*.p12
*.pfx
!certs/.gitkeep
!certs/README.md
```

### 2. 限制文件权限

```bash
# 只有所有者可以读写
chmod 600 certs/**/*.{pem,crt,key,p12}

# 限制目录访问
chmod 700 certs/*/
```

### 3. 使用环境变量

不要在代码中硬编码 AppID、APIv3Key、SerialNo 等敏感信息。

---

## 总结

自动证书加载功能让支付配置变得更加简单：

✅ **自动扫描** - 无需手动编写加载代码  
✅ **统一管理** - 所有证书集中在 `certs/` 目录  
✅ **高性能** - 启动时加载，运行时从缓存读取  
✅ **灵活配置** - 支持多商户、多渠道  
✅ **向后兼容** - 仍然支持手动加载方式  

通过这个功能，你可以更专注于业务逻辑，而不是证书管理！
