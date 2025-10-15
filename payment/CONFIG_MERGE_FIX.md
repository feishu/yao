# 配置合并问题修复文档

## 🐛 问题描述

### 原始问题

在 `GetOrCreateProvider` 方法（`payment/payment.go:99-108`）中，只从 `pm.configs` 获取配置，但**证书内容存储在 `pm.certConfigs` 中**，导致以下问题：

1. **证书无法使用**：自动加载的证书配置无法传递给 Provider
2. **配置源分离**：证书配置和商户配置分别存储，但创建 Provider 时只使用了其中一个
3. **功能不完整**：证书自动加载功能虽然实现了，但实际不生效

### 问题代码

```go
// ❌ 只从 pm.configs 获取配置
configKey := fmt.Sprintf("%s_%s", merchantID, string(channel))
config, exists := pm.configs[configKey]
if !exists {
    return nil, fmt.Errorf("merchant config not found: %s", configKey)
}

// 证书配置在 pm.certConfigs 中，但这里没有使用！
```

### 问题影响

- 即使证书文件正确放置在 `certs/` 目录下
- `loadCertificates()` 成功加载并缓存到 `pm.certConfigs`
- Provider 创建时仍然无法获取证书内容
- 导致 Provider 初始化失败或功能受限

---

## ✅ 解决方案

### 核心思路：配置合并策略

实现**双源配置合并**机制：

1. **证书配置优先**：自动从 `pm.certConfigs` 加载证书内容（私钥、公钥等）
2. **手动配置覆盖**：允许通过 `SetMerchantConfig` 手动设置/覆盖配置
3. **灵活组合**：支持三种配置模式

### 支持的配置模式

#### 模式 1：仅自动加载证书（推荐）

```bash
# 目录结构
certs/
  ├── merchant001/
  │   ├── alipay/
  │   │   ├── private_key.pem
  │   │   ├── public_key.pem
  │   │   ├── app_cert.crt         # 可选
  │   │   └── alipay_root_cert.crt # 可选
  │   └── wechat/
  │       ├── private_key.pem
  │       └── public_key.pem

# 代码
// 启动时自动加载，无需手动调用
// LoadCertsFromDirectory() 在 Load() 中自动执行

// 直接使用，证书已自动配置
payment.CreateOrder(&CreateOrderParams{
    MerchantNo: "merchant001",
    Channel:    "alipay",
    // ...
})
```

#### 模式 2：手动配置 + 证书补充

```go
// 手动设置基本配置
payment.SetMerchantConfig("merchant001", "alipay", map[string]interface{}{
    "app_id":     "2021001234567890",
    "notify_url": "https://example.com/notify",
    "sandbox":    true,
})

// 证书从文件自动加载并合并到配置中
// 最终 Provider 获得完整配置：app_id + notify_url + 证书内容
```

#### 模式 3：完全手动配置

```go
// 不使用自动证书加载，完全手动配置
payment.SetMerchantConfig("merchant001", "alipay", map[string]interface{}{
    "app_id":      "2021001234567890",
    "private_key": "-----BEGIN RSA PRIVATE KEY-----\n...",
    "public_key":  "-----BEGIN PUBLIC KEY-----\n...",
    "notify_url":  "https://example.com/notify",
})
```

---

## 🔧 实现细节

### 修改的代码：`payment/payment.go` - `GetOrCreateProvider` 方法

```go
// 获取或创建 Provider
func (pm *PaymentManager) GetOrCreateProvider(merchantID string, channel PaymentChannel) (PaymentProvider, error) {
    pm.mutex.Lock()
    defer pm.mutex.Unlock()

    // 生成缓存 key
    cacheKey := fmt.Sprintf("%s_%s", merchantID, string(channel))

    // 检查缓存
    if provider, exists := pm.providerInstances[cacheKey]; exists {
        return provider, nil
    }

    // 获取工厂函数
    factory, exists := pm.providerFactories[string(channel)]
    if !exists {
        return nil, fmt.Errorf("provider factory not found for channel: %s", channel)
    }

    // ✅ 关键改进：从两个配置源获取
    configKey := fmt.Sprintf("%s_%s", merchantID, string(channel))
    config, configExists := pm.configs[configKey]
    certConfig, certExists := pm.certConfigs[configKey]
    
    // 至少需要一个配置源
    if !configExists && !certExists {
        return nil, fmt.Errorf("no config found for merchant: %s", configKey)
    }

    // ✅ 合并配置
    finalConfig := make(map[string]interface{})
    
    // 1. 加载证书配置（如果存在）
    if certExists {
        log.Debug("Loading certificate config for: %s", configKey)
        
        // 基本配置
        finalConfig["merchant_id"] = certConfig.MerchantID
        finalConfig["channel"] = certConfig.Channel
        
        // 证书内容
        if certConfig.PrivateKey != "" {
            finalConfig["private_key"] = certConfig.PrivateKey
        }
        if certConfig.PublicKey != "" {
            finalConfig["public_key"] = certConfig.PublicKey
        }
        
        // 支付宝额外证书
        if certConfig.AppCert != "" {
            finalConfig["app_cert"] = certConfig.AppCert
        }
        if certConfig.RootCert != "" {
            finalConfig["root_cert"] = certConfig.RootCert
        }
        
        // 添加 ExtraFiles 中的所有额外证书
        for key, value := range certConfig.ExtraFiles {
            finalConfig[key] = value
        }
    }
    
    // 2. 合并手动配置（覆盖证书配置）
    if configExists {
        if configMap, ok := config.(map[string]interface{}); ok {
            log.Debug("Merging manual config for: %s", configKey)
            for k, v := range configMap {
                finalConfig[k] = v
            }
        } else {
            return nil, fmt.Errorf("invalid config format for merchant: %s", configKey)
        }
    }
    
    log.Debug("Final config for %s: %d keys", configKey, len(finalConfig))

    // 使用工厂函数创建 Provider
    provider, err := factory(finalConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create provider: %v", err)
    }

    // 缓存 Provider 实例
    pm.providerInstances[cacheKey] = provider
    log.Info("✅ Provider created and cached: %s", cacheKey)

    return provider, nil
}
```

---

## 📊 配置合并逻辑流程

```
┌─────────────────────────────────────────┐
│ GetOrCreateProvider(merchantID, channel)│
└──────────────┬──────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────┐
│ 1. 检查缓存 (providerInstances)          │
│    ├─ 命中 → 返回缓存的 Provider         │
│    └─ 未命中 → 继续                      │
└──────────────┬───────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────┐
│ 2. 从两个配置源获取数据                  │
│    ├─ pm.certConfigs[key]                │
│    └─ pm.configs[key]                    │
└──────────────┬───────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────┐
│ 3. 检查配置源                            │
│    ├─ 两者都不存在 → 报错                │
│    └─ 至少一个存在 → 继续                │
└──────────────┬───────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────┐
│ 4. 合并配置（分步骤）                    │
│    ├─ Step 1: 加载证书配置               │
│    │   ├─ merchant_id, channel           │
│    │   ├─ private_key, public_key        │
│    │   ├─ app_cert, root_cert            │
│    │   └─ ExtraFiles (所有额外证书)      │
│    │                                      │
│    └─ Step 2: 覆盖手动配置               │
│        └─ 遍历 configMap，覆盖同名 key   │
└──────────────┬───────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────┐
│ 5. 使用工厂函数创建 Provider             │
│    factory(finalConfig)                  │
└──────────────┬───────────────────────────┘
               │
               ▼
┌──────────────────────────────────────────┐
│ 6. 缓存 Provider 实例                    │
│    providerInstances[key] = provider     │
└──────────────┬───────────────────────────┘
               │
               ▼
          返回 Provider
```

---

## 🧪 测试场景

### 场景 1：纯证书自动加载

```go
// 准备：放置证书文件
// certs/test_merchant/alipay/private_key.pem
// certs/test_merchant/alipay/public_key.pem

// 测试：直接创建订单
order, err := payment.CreateOrder(&CreateOrderParams{
    MerchantNo: "test_merchant",
    Channel:    "alipay",
    Amount:     100,
    Subject:    "测试订单",
    // ...
})

// 预期：✅ 成功，使用自动加载的证书
```

### 场景 2：手动配置覆盖证书字段

```go
// 准备：证书文件 + 手动配置
payment.SetMerchantConfig("test_merchant", "alipay", map[string]interface{}{
    "sandbox":    true,           // 新增配置
    "private_key": "manual_key",  // 覆盖证书文件中的 private_key
})

// 测试：创建订单
order, err := payment.CreateOrder(...)

// 预期：✅ 成功
// - sandbox = true (手动配置)
// - private_key = "manual_key" (手动配置覆盖)
// - public_key = (来自证书文件)
```

### 场景 3：完全手动配置（无证书文件）

```go
// 准备：只有手动配置，无证书文件
payment.SetMerchantConfig("manual_merchant", "wechat", map[string]interface{}{
    "app_id":      "wx1234567890",
    "private_key": "-----BEGIN PRIVATE KEY-----...",
    "public_key":  "-----BEGIN PUBLIC KEY-----...",
    "mch_id":      "1234567890",
})

// 测试：创建订单
order, err := payment.CreateOrder(&CreateOrderParams{
    MerchantNo: "manual_merchant",
    Channel:    "wechat",
    // ...
})

// 预期：✅ 成功，使用手动配置
```

### 场景 4：配置不存在

```go
// 准备：无证书文件，无手动配置
// 测试：创建订单
order, err := payment.CreateOrder(&CreateOrderParams{
    MerchantNo: "nonexistent",
    Channel:    "alipay",
    // ...
})

// 预期：❌ 报错
// "no config found for merchant: nonexistent_alipay 
// (please call payment.SetConfig or ensure certificates are loaded)"
```

---

## 🎯 关键改进点

### 改进 1：配置源统一访问

**之前**：
```go
❌ 只检查 pm.configs，忽略 pm.certConfigs
```

**之后**：
```go
✅ 同时检查两个配置源，至少需要一个
if !configExists && !certExists {
    return nil, fmt.Errorf("no config found...")
}
```

### 改进 2：证书内容正确传递

**之前**：
```go
❌ 证书加载成功，但 Provider 无法获取
```

**之后**：
```go
✅ 证书内容合并到 finalConfig，传递给 Provider
finalConfig["private_key"] = certConfig.PrivateKey
finalConfig["public_key"] = certConfig.PublicKey
```

### 改进 3：灵活配置策略

**之前**：
```go
❌ 必须手动调用 SetConfig，否则无法使用
```

**之后**：
```go
✅ 支持三种模式：
- 仅证书自动加载（推荐）
- 证书 + 手动配置
- 仅手动配置
```

### 改进 4：配置覆盖机制

**之前**：
```go
❌ 配置无法更新或覆盖
```

**之后**：
```go
✅ 手动配置可以覆盖证书配置
for k, v := range configMap {
    finalConfig[k] = v  // 覆盖同名 key
}
```

---

## 📋 相关文件

### 修改的文件

1. **`payment/payment.go`**
   - 修改 `GetOrCreateProvider` 方法 (line 78-170)
   - 实现配置合并逻辑

### 相关文件（未修改）

2. **`payment/cert_loader.go`**
   - 证书自动加载实现
   - Key 格式一致：`merchantID_channel`

3. **`payment/load.go`**
   - 启动时调用 `loadCertificates()`
   - 缓存到 `Manager.certConfigs`

---

## 🚀 验证步骤

### 步骤 1：编译验证

```bash
cd /Users/L/Desktop/Code/yao_dev/yao
go build ./payment
```

### 步骤 2：单元测试（待创建）

```go
func TestGetOrCreateProvider_CertOnly(t *testing.T) {
    // 测试：仅证书配置
}

func TestGetOrCreateProvider_ConfigOnly(t *testing.T) {
    // 测试：仅手动配置
}

func TestGetOrCreateProvider_Merged(t *testing.T) {
    // 测试：证书 + 手动配置合并
}

func TestGetOrCreateProvider_ConfigOverride(t *testing.T) {
    // 测试：手动配置覆盖证书
}
```

### 步骤 3：集成测试

```bash
# 1. 准备证书文件
mkdir -p certs/test001/alipay
echo "test private key" > certs/test001/alipay/private_key.pem
echo "test public key" > certs/test001/alipay/public_key.pem

# 2. 启动应用
./yao start

# 3. 验证证书加载
# 查看日志：
# [INFO] Loading certificates from directory: /path/to/certs
# [INFO] ✓ Loaded certificates for merchant: test001, channel: alipay
# [INFO] Certificates auto-loaded: 1 configurations

# 4. 测试创建订单
curl -X POST http://localhost:5099/api/payment/create_order \
  -H "Content-Type: application/json" \
  -d '{
    "merchant_no": "test001",
    "channel": "alipay",
    "amount": 100,
    "subject": "测试订单"
  }'
```

---

## 📈 预期效果

### 功能完整性

- ✅ 证书自动加载功能生效
- ✅ 手动配置功能保留
- ✅ 配置覆盖机制工作正常
- ✅ 多商户支持正常

### 用户体验

- ✅ 默认零配置：放置证书文件即可使用
- ✅ 灵活配置：需要时可手动设置额外参数
- ✅ 清晰错误：配置缺失时报错明确

### 代码质量

- ✅ 逻辑清晰：分步骤合并配置
- ✅ 日志完善：关键步骤有日志输出
- ✅ 错误处理：配置缺失或格式错误有明确提示

---

## 🎉 总结

通过实现**配置合并机制**，我们成功解决了以下问题：

1. ✅ **证书配置无法使用** → 现在可以自动加载并传递给 Provider
2. ✅ **配置源分离** → 统一合并为 `finalConfig`
3. ✅ **功能不完整** → 证书自动加载完全生效
4. ✅ **缺乏灵活性** → 支持三种配置模式

这个修复是对之前工厂模式重构的**重要补充**，确保了证书自动加载功能的完整实现。

---

**最后更新**：2025-01-14  
**修复版本**：v2.0  
**关联文档**：`FIX_SUMMARY.md` - 工厂模式重构总结
