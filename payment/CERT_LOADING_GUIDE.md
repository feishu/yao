# Payment模块证书加载功能使用指南

## 概述

Payment模块现在提供了两个Process用于安全地加载支付证书文件：

1. **`payment.LoadCert`** - 加载证书文件并返回原始内容
2. **`payment.LoadCertBase64`** - 加载证书文件并转换为Base64编码

这两个Process提供了统一、安全的证书文件读取机制，防止路径穿越攻击，并简化了支付配置流程。

---

## API说明

### 1. payment.LoadCert

加载证书文件并返回原始内容（适用于PEM格式证书）。

**参数：**
- `certPath` (string): 证书文件路径，相对于应用根目录

**返回值：**
```json
{
  "success": true,
  "content": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----",
  "path": "certs/alipay_private_key.pem"
}
```

**支持的文件扩展名：**
- `.pem` - PEM格式证书/密钥
- `.crt` - 证书文件
- `.key` - 私钥文件
- `.p12` - PKCS#12格式
- `.pfx` - PFX格式
- `.cer` - DER编码证书

---

### 2. payment.LoadCertBase64

加载证书文件并转换为Base64编码（适用于二进制证书格式）。

**参数：**
- `certPath` (string): 证书文件路径，相对于应用根目录

**返回值：**
```json
{
  "success": true,
  "content": "MIIEvgIBADANBgkqhkiG9w0BAQEFAASC...",
  "path": "certs/wechat_cert.p12",
  "format": "base64"
}
```

---

## 安全特性

### 路径安全验证

两个Process都会进行严格的路径安全验证：

✅ **允许：**
- 相对路径：`certs/alipay_private_key.pem`
- 子目录：`config/payment/wechat_cert.pem`

❌ **禁止：**
- 路径穿越：`../../../etc/passwd`
- 绝对路径：`/etc/ssl/private.key`
- 无扩展名：`mycert`
- 不支持的扩展名：`cert.txt`, `key.dat`

### 错误处理

当路径不安全或文件不存在时，会抛出异常：

```json
{
  "code": 400,
  "message": "无效的证书路径: ../../../etc/passwd"
}
```

```json
{
  "code": 500,
  "message": "读取证书文件失败: open /path/to/app/certs/notfound.pem: no such file or directory"
}
```

---

## 使用示例

### 示例1：配置支付宝支付（推荐目录结构）

**目录结构：**
```
my-yao-app/
├── apis/
├── models/
├── scripts/
│   └── payment/
│       └── setup.js      # 支付配置脚本
└── certs/                # 证书目录
    ├── alipay/
    │   ├── app_private_key.pem
    │   └── alipay_public_key.pem
    └── wechat/
        ├── apiclient_key.pem
        ├── apiclient_cert.pem
        └── apiclient_cert.p12
```

**配置脚本（scripts/payment/setup.js）：**

```javascript
/**
 * 配置支付渠道
 * 支付宝 + 微信支付
 */
function setupPaymentChannels() {
    const merchantID = "merchant_001"
    
    // 1. 加载支付宝证书
    console.log("Loading Alipay certificates...")
    const alipayPrivateKey = Process("payment.LoadCert", "certs/alipay/app_private_key.pem")
    const alipayPublicKey = Process("payment.LoadCert", "certs/alipay/alipay_public_key.pem")
    
    if (!alipayPrivateKey.success || !alipayPublicKey.success) {
        throw new Error("Failed to load Alipay certificates")
    }
    
    // 2. 配置支付宝
    Process("payment.SetConfig", merchantID, "alipay", {
        app_id: "2021001234567890",
        private_key: alipayPrivateKey.content,
        public_key: alipayPublicKey.content,
        is_sandbox: false,
        sign_type: "RSA2"
    })
    console.log("✓ Alipay configured")
    
    // 3. 加载微信证书
    console.log("Loading WeChat certificates...")
    const wechatPrivateKey = Process("payment.LoadCert", "certs/wechat/apiclient_key.pem")
    
    if (!wechatPrivateKey.success) {
        throw new Error("Failed to load WeChat certificates")
    }
    
    // 4. 配置微信支付
    Process("payment.SetConfig", merchantID, "wechat", {
        app_id: "wx1234567890abcdef",
        mch_id: "1234567890",
        apiv3_key: "your_32_character_apiv3_key_here",
        private_key: wechatPrivateKey.content,
        serial_no: "5B3D2F1A8C9E...",  // 证书序列号
        is_sandbox: false
    })
    console.log("✓ WeChat Pay configured")
    
    return {
        success: true,
        message: "Payment channels configured successfully"
    }
}

// 导出函数
exports.setupPaymentChannels = setupPaymentChannels
```

**调用脚本：**
```javascript
// 在应用启动时或通过API调用
const setup = Process("scripts.payment.setup.setupPaymentChannels")
console.log(setup.message)
```

---

### 示例2：支持多商户配置

```javascript
/**
 * 批量配置多个商户的支付渠道
 */
function setupMultipleMerchants() {
    const merchants = [
        {
            id: "merchant_001",
            name: "主站商户",
            alipay: {
                app_id: "2021001111111111",
                cert_dir: "certs/merchant_001/alipay"
            },
            wechat: {
                app_id: "wx1111111111111111",
                mch_id: "1111111111",
                cert_dir: "certs/merchant_001/wechat",
                apiv3_key: "key_for_merchant_001_32_chars",
                serial_no: "SERIAL_001"
            }
        },
        {
            id: "merchant_002",
            name: "子站商户",
            alipay: {
                app_id: "2021002222222222",
                cert_dir: "certs/merchant_002/alipay"
            },
            wechat: {
                app_id: "wx2222222222222222",
                mch_id: "2222222222",
                cert_dir: "certs/merchant_002/wechat",
                apiv3_key: "key_for_merchant_002_32_chars",
                serial_no: "SERIAL_002"
            }
        }
    ]
    
    const results = []
    
    for (const merchant of merchants) {
        console.log(`Configuring ${merchant.name} (${merchant.id})...`)
        
        try {
            // 配置支付宝
            const alipayPrivKey = Process("payment.LoadCert", 
                `${merchant.alipay.cert_dir}/app_private_key.pem`)
            const alipayPubKey = Process("payment.LoadCert", 
                `${merchant.alipay.cert_dir}/alipay_public_key.pem`)
            
            Process("payment.SetConfig", merchant.id, "alipay", {
                app_id: merchant.alipay.app_id,
                private_key: alipayPrivKey.content,
                public_key: alipayPubKey.content,
                is_sandbox: false
            })
            
            // 配置微信
            const wechatPrivKey = Process("payment.LoadCert", 
                `${merchant.wechat.cert_dir}/apiclient_key.pem`)
            
            Process("payment.SetConfig", merchant.id, "wechat", {
                app_id: merchant.wechat.app_id,
                mch_id: merchant.wechat.mch_id,
                apiv3_key: merchant.wechat.apiv3_key,
                private_key: wechatPrivKey.content,
                serial_no: merchant.wechat.serial_no,
                is_sandbox: false
            })
            
            results.push({
                merchant_id: merchant.id,
                success: true,
                message: `${merchant.name} configured`
            })
            
            console.log(`✓ ${merchant.name} configured`)
            
        } catch (error) {
            results.push({
                merchant_id: merchant.id,
                success: false,
                error: error.message
            })
            console.error(`✗ Failed to configure ${merchant.name}: ${error.message}`)
        }
    }
    
    return {
        success: results.every(r => r.success),
        results: results,
        total: merchants.length,
        success_count: results.filter(r => r.success).length
    }
}
```

---

### 示例3：使用Base64格式（PKCS#12证书）

某些支付渠道可能需要PKCS#12格式的证书：

```javascript
/**
 * 配置使用PKCS#12证书的支付渠道
 */
function setupWithPKCS12() {
    const merchantID = "merchant_special"
    
    // 加载PKCS#12证书（二进制格式，需要Base64编码）
    const p12Cert = Process("payment.LoadCertBase64", "certs/wechat/apiclient_cert.p12")
    
    if (!p12Cert.success) {
        throw new Error("Failed to load PKCS#12 certificate")
    }
    
    // 某些SDK可能需要Base64格式的证书
    Process("payment.SetConfig", merchantID, "wechat", {
        app_id: "wx_special",
        mch_id: "1234567890",
        apiv3_key: "special_key_32_characters_long",
        cert_p12: p12Cert.content,  // Base64编码的证书
        cert_password: "cert_password_here",
        is_sandbox: false
    })
    
    console.log("✓ PKCS#12 certificate configured")
}
```

---

### 示例4：动态证书切换（生产/测试环境）

```javascript
/**
 * 根据环境自动选择证书
 */
function setupByEnvironment() {
    const merchantID = "merchant_001"
    const env = Process("utils.env.Get", "YAO_ENV") || "production"
    
    // 根据环境选择证书目录
    const certDir = env === "production" ? "certs/prod" : "certs/sandbox"
    const isSandbox = env !== "production"
    
    console.log(`Setting up payment for ${env} environment...`)
    
    // 加载对应环境的证书
    const alipayPrivKey = Process("payment.LoadCert", 
        `${certDir}/alipay/app_private_key.pem`)
    const alipayPubKey = Process("payment.LoadCert", 
        `${certDir}/alipay/alipay_public_key.pem`)
    
    // 配置支付宝
    Process("payment.SetConfig", merchantID, "alipay", {
        app_id: isSandbox ? "2021000000000001" : "2021001234567890",
        private_key: alipayPrivKey.content,
        public_key: alipayPubKey.content,
        is_sandbox: isSandbox
    })
    
    console.log(`✓ Payment configured for ${env} environment`)
    
    return {
        success: true,
        environment: env,
        is_sandbox: isSandbox
    }
}
```

---

## 最佳实践

### 1. 证书文件组织

推荐的证书目录结构：

```
certs/
├── prod/                    # 生产环境证书
│   ├── alipay/
│   │   ├── app_private_key.pem
│   │   ├── app_public_cert.crt
│   │   ├── alipay_public_key.pem
│   │   └── alipay_root_cert.crt
│   └── wechat/
│       ├── apiclient_key.pem
│       ├── apiclient_cert.pem
│       └── apiclient_cert.p12
├── sandbox/                 # 沙箱环境证书
│   ├── alipay/
│   └── wechat/
└── README.md               # 证书说明文档
```

### 2. 证书安全

⚠️ **重要提示：**

1. **不要将证书提交到版本控制系统**
   ```gitignore
   # .gitignore
   certs/
   *.pem
   *.key
   *.p12
   *.pfx
   ```

2. **使用环境变量管理敏感配置**
   ```javascript
   const apiv3Key = Process("utils.env.Get", "WECHAT_APIV3_KEY")
   const serialNo = Process("utils.env.Get", "WECHAT_SERIAL_NO")
   ```

3. **限制文件权限**
   ```bash
   chmod 600 certs/**/*.pem
   chmod 700 certs/
   ```

### 3. 错误处理

总是检查证书加载结果：

```javascript
function safeLoadCert(certPath, description) {
    try {
        const result = Process("payment.LoadCert", certPath)
        
        if (!result.success) {
            throw new Error(`Failed to load ${description}: ${result.error}`)
        }
        
        console.log(`✓ Loaded ${description} from ${certPath}`)
        return result.content
        
    } catch (error) {
        console.error(`✗ Error loading ${description}: ${error.message}`)
        throw error
    }
}

// 使用
const privateKey = safeLoadCert(
    "certs/alipay/app_private_key.pem", 
    "Alipay private key"
)
```

### 4. 配置验证

在配置后进行验证：

```javascript
function validatePaymentConfig(merchantID, channel) {
    try {
        const config = Process("payment.GetConfig", merchantID, channel)
        
        if (!config.success) {
            throw new Error(`Config not found for ${merchantID}/${channel}`)
        }
        
        console.log(`✓ Payment config validated for ${merchantID}/${channel}`)
        return true
        
    } catch (error) {
        console.error(`✗ Config validation failed: ${error.message}`)
        return false
    }
}

// 使用
setupPaymentChannels()
validatePaymentConfig("merchant_001", "alipay")
validatePaymentConfig("merchant_001", "wechat")
```

---

## 常见问题

### Q1: 支持哪些证书格式？

**A:** 支持以下文件扩展名：
- `.pem` - PEM格式（最常用）
- `.crt` - 证书文件
- `.key` - 私钥文件  
- `.p12` / `.pfx` - PKCS#12格式
- `.cer` - DER编码证书

### Q2: 证书路径是相对于哪个目录？

**A:** 相对于Yao应用的根目录（`config.Conf.Root`），通常是启动Yao时所在的目录。

### Q3: 如何处理证书文件不存在的情况？

**A:** Process会抛出异常，应该使用try-catch捕获：

```javascript
try {
    const cert = Process("payment.LoadCert", "certs/missing.pem")
} catch (error) {
    console.error("Certificate not found:", error.message)
    // 处理错误，例如使用默认配置或终止初始化
}
```

### Q4: LoadCert和LoadCertBase64应该用哪个？

**A:** 
- **LoadCert**: PEM格式的文本证书（支付宝/微信的私钥、公钥）
- **LoadCertBase64**: 二进制格式证书（.p12, .pfx）或需要Base64编码传输的场景

### Q5: 可以加载应用外部的证书吗？

**A:** 不可以。出于安全考虑，只能加载应用根目录下的证书文件，不支持绝对路径。

---

## 调试技巧

### 启用调试日志

在开发环境下，Process会输出详细的调试日志：

```
[DEBUG] ProcessLoadCert: certPath=certs/alipay/app_private_key.pem
[DEBUG] Cert loaded successfully: certs/alipay/app_private_key.pem (1675 bytes)
```

### 验证证书内容

```javascript
function debugCert(certPath) {
    const cert = Process("payment.LoadCert", certPath)
    
    console.log("=== Certificate Info ===")
    console.log("Path:", cert.path)
    console.log("Length:", cert.content.length)
    console.log("First 100 chars:", cert.content.substring(0, 100))
    console.log("========================")
    
    return cert
}
```

---

## 总结

`payment.LoadCert` 和 `payment.LoadCertBase64` 两个Process为Yao应用提供了：

✅ **安全的证书加载机制**  
✅ **统一的错误处理**  
✅ **防止路径穿越攻击**  
✅ **支持多种证书格式**  
✅ **简化配置流程**

通过这些Process，您可以安全、方便地管理支付证书，并快速配置多个支付渠道。
