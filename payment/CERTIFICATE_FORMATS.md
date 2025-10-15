# 证书文件格式支持

## 📋 概述

Payment 模块的证书自动加载功能支持多种文件格式和命名规范，以适应不同的证书管理习惯。

---

## 🔑 支持的文件扩展名

### 主要证书文件

证书加载器会按照以下优先级尝试查找文件：

| 文件类型 | 支持的扩展名 | 说明 |
|---------|------------|------|
| 私钥文件 | `.pem`, `.key`, `.pub`, `.crt`, `.cer`, `.p12`, `.pfx` | 按顺序查找 |
| 公钥文件 | `.pem`, `.key`, `.pub`, `.crt`, `.cer`, `.p12`, `.pfx` | 按顺序查找 |
| 证书文件 | `.crt`, `.cer`, `.pem` | 支付宝和微信额外证书 |
| P12/PFX | `.p12`, `.pfx` | PKCS#12 格式，可用于任意证书 |

### 查找逻辑

系统会按照扩展名列表的顺序查找文件，找到第一个存在的文件就停止。

例如，查找私钥文件时：
1. 先查找 `private_key.pem`
2. 如果不存在，查找 `private_key.key`
3. 如果不存在，查找 `private_key.pub`
4. ... 依此类推

---

## 📁 目录结构和文件命名

### 基本结构

```
certs/
  └── {merchant_id}/
      ├── alipay/
      │   ├── private_key.{ext}      # 必需：应用私钥
      │   ├── public_key.{ext}       # 必需：支付宝公钥
      │   ├── app_cert.{ext}         # 可选：应用公钥证书
      │   ├── alipay_cert.{ext}      # 可选：支付宝公钥证书
      │   ├── alipay_public_cert.{ext} # 可选：支付宝公钥证书（别名）
      │   ├── alipay_root_cert.{ext} # 可选：支付宝根证书
      │   └── root_cert.{ext}        # 可选：根证书（别名）
      └── wechat/
          ├── private_key.{ext}      # 必需：商户私钥
          ├── public_key.{ext}       # 必需：微信公钥
          ├── apiclient_cert.{ext}   # 可选：微信 API 证书
          ├── apiclient_key.{ext}    # 可选：微信 API 密钥
          └── apiclient_cert.{p12|pfx} # 可选：P12 格式证书
```

其中 `{ext}` 表示任意支持的扩展名。

---

## 🔧 支付宝证书详解

### 必需文件

| 文件名模式 | 支持的扩展名 | 说明 |
|-----------|------------|------|
| `private_key.*` | `.pem`, `.key`, `.pub`, `.crt`, `.cer`, `.p12`, `.pfx` | 应用私钥 |
| `public_key.*` | `.pem`, `.key`, `.pub`, `.crt`, `.cer`, `.p12`, `.pfx` | 支付宝公钥 |

### 可选文件（证书模式）

| 文件名模式 | 支持的扩展名 | 说明 |
|-----------|------------|------|
| `app_cert.*` | `.crt`, `.cer`, `.pem` | 应用公钥证书 |
| `alipay_cert.*` | `.crt`, `.cer`, `.pem` | 支付宝公钥证书 |
| `alipay_public_cert.*` | `.crt`, `.cer`, `.pem` | 支付宝公钥证书（别名） |
| `alipay_root_cert.*` | `.crt`, `.cer`, `.pem` | 支付宝根证书 |
| `root_cert.*` | `.crt`, `.cer`, `.pem` | 根证书（别名） |

### 示例配置 1：普通密钥模式

```bash
certs/merchant001/alipay/
  ├── private_key.pem
  └── public_key.pem
```

### 示例配置 2：证书模式

```bash
certs/merchant001/alipay/
  ├── private_key.key          # 使用 .key 扩展名
  ├── public_key.pem
  ├── app_cert.crt
  ├── alipay_public_cert.crt
  └── alipay_root_cert.crt
```

### 示例配置 3：混合扩展名

```bash
certs/merchant001/alipay/
  ├── private_key.pem
  ├── public_key.cer           # 使用 .cer 扩展名
  ├── app_cert.crt
  └── root_cert.pem            # 使用简化命名
```

---

## 🔧 微信支付证书详解

### 必需文件

| 文件名模式 | 支持的扩展名 | 说明 |
|-----------|------------|------|
| `private_key.*` | `.pem`, `.key`, `.pub`, `.crt`, `.cer`, `.p12`, `.pfx` | 商户私钥 |
| `public_key.*` | `.pem`, `.key`, `.pub`, `.crt`, `.cer`, `.p12`, `.pfx` | 微信公钥 |

### 可选文件

| 文件名模式 | 支持的扩展名 | 说明 |
|-----------|------------|------|
| `apiclient_cert.*` | `.pem`, `.crt`, `.cer` | 微信 API 证书 |
| `apiclient_key.*` | `.pem`, `.key` | 微信 API 密钥 |
| `apiclient_cert.*` | `.p12`, `.pfx` | P12 格式证书 |

### 示例配置 1：基本配置

```bash
certs/merchant002/wechat/
  ├── private_key.pem
  └── public_key.pem
```

### 示例配置 2：包含 API 证书

```bash
certs/merchant002/wechat/
  ├── private_key.key          # 使用 .key 扩展名
  ├── public_key.pem
  ├── apiclient_cert.pem       # API 证书
  └── apiclient_key.pem        # API 密钥
```

### 示例配置 3：使用 P12 证书

```bash
certs/merchant002/wechat/
  ├── private_key.pem
  ├── public_key.pem
  └── apiclient_cert.p12       # P12 格式（或 .pfx）
```

---

## 🎯 文件命名最佳实践

### 1. 使用标准命名

推荐使用标准的文件名模式，便于维护：

✅ **推荐**
```bash
private_key.pem
public_key.pem
app_cert.crt
alipay_root_cert.crt
```

❌ **不推荐**
```bash
my_private_key.pem    # 不会被识别
app123.key            # 不会被识别
```

### 2. 扩展名灵活选择

根据实际文件内容选择合适的扩展名：

- `.pem` - PEM 编码的证书或密钥（最常用）
- `.key` - 私钥文件
- `.pub` - 公钥文件
- `.crt` / `.cer` - 证书文件
- `.p12` / `.pfx` - PKCS#12 格式证书

### 3. 保持一致性

在同一商户目录下，建议保持扩展名的一致性：

✅ **推荐（统一使用 .pem）**
```bash
certs/merchant001/alipay/
  ├── private_key.pem
  ├── public_key.pem
  ├── app_cert.pem
  └── root_cert.pem
```

✅ **也可以（根据文件类型）**
```bash
certs/merchant001/alipay/
  ├── private_key.key
  ├── public_key.pub
  ├── app_cert.crt
  └── root_cert.crt
```

---

## 🔍 文件查找顺序

### 私钥文件查找顺序

```
1. private_key.pem
2. private_key.key
3. private_key.pub
4. private_key.crt
5. private_key.cer
```

### 公钥文件查找顺序

```
1. public_key.pem
2. public_key.key
3. public_key.pub
4. public_key.crt
5. public_key.cer
```

### 支付宝证书文件查找顺序

**应用证书：**
```
1. app_cert.crt
2. app_cert.cer
3. app_cert.pem
```

**支付宝公钥证书：**
```
1. alipay_cert.crt (优先)
2. alipay_cert.cer
3. alipay_cert.pem
4. alipay_public_cert.crt (备用)
5. alipay_public_cert.cer
6. alipay_public_cert.pem
```

**根证书：**
```
1. alipay_root_cert.crt (优先)
2. alipay_root_cert.cer
3. alipay_root_cert.pem
4. root_cert.crt (备用)
5. root_cert.cer
6. root_cert.pem
```

### 微信证书文件查找顺序

**API 证书：**
```
1. apiclient_cert.pem
2. apiclient_cert.crt
3. apiclient_cert.cer
```

**API 密钥：**
```
1. apiclient_key.pem
2. apiclient_key.key
```

**P12 证书：**
```
1. apiclient_cert.p12
2. apiclient_cert.pfx
```

---

## 💡 常见问题

### Q1: 我的证书文件扩展名是 `.txt`，能用吗？

❌ 不能。系统只支持标准的证书文件扩展名（`.pem`, `.key`, `.crt` 等）。

**解决方案：** 重命名文件为支持的扩展名：
```bash
mv private_key.txt private_key.pem
```

### Q2: 我有多个版本的证书，如何选择？

系统会使用**第一个找到的文件**。建议：

1. 删除或移走旧版本证书
2. 或使用版本管理目录：

```bash
certs/
  ├── merchant001/
  │   └── alipay/
  │       ├── private_key.pem      # 当前使用
  │       └── backup/
  │           └── private_key_old.pem  # 备份
```

### Q3: 文件名大小写敏感吗？

在 Linux/macOS 上**敏感**，在 Windows 上**不敏感**。

为了跨平台兼容，建议统一使用**小写**文件名：

✅ `private_key.pem`  
❌ `Private_Key.PEM`

### Q4: 可以使用软链接吗？

✅ 可以。系统会自动解析软链接：

```bash
cd certs/merchant001/alipay/
ln -s /secure/certs/prod_private_key.pem private_key.pem
```

### Q5: 文件必须有读取权限吗？

✅ 是的。确保 Yao 进程有读取权限：

```bash
# 设置正确的权限
chmod 600 certs/merchant001/alipay/private_key.pem
chown yao:yao certs/merchant001/alipay/private_key.pem
```

---

## 🚀 验证配置

启动应用后，检查日志确认证书加载：

```
[INFO] Loading certificates from directory: /path/to/certs
[DEBUG] Scanning merchant directory: merchant001
[DEBUG] Loading certificates for merchant001/alipay
[DEBUG] Loaded private key from: /path/to/certs/merchant001/alipay/private_key.pem
[DEBUG] Loaded public key from: /path/to/certs/merchant001/alipay/public_key.pem
[DEBUG] Loaded app cert from: /path/to/certs/merchant001/alipay/app_cert.crt
[INFO] ✓ Loaded certificates for merchant: merchant001, channel: alipay
[INFO] Certificate auto-loading completed: 1 configurations loaded
```

使用 Process 检查：

```javascript
var merchants = process.Run("payment.ListMerchants")
console.log(merchants)
// [
//   {
//     "key": "merchant001_alipay",
//     "merchant_id": "merchant001",
//     "channel": "alipay",
//     "has_cert": true,
//     "has_app_id": false
//   }
// ]
```

---

## 📚 相关文档

- [CONFIGURATION_GUIDE.md](./CONFIGURATION_GUIDE.md) - 完整配置指南
- [CERTIFICATE_AUTO_LOAD.md](./CERTIFICATE_AUTO_LOAD.md) - 证书自动加载说明
- [SIMPLIFY_ARCHITECTURE.md](./SIMPLIFY_ARCHITECTURE.md) - 架构设计

---

**最后更新**：2025-01-14  
**版本**：v1.1 - 支持多种文件扩展名  
**作者**：Payment Team
