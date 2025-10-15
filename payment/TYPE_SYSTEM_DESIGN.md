# Payment 模块类型系统设计文档

## 设计原则

**单一数据源（Single Source of Truth）**：所有类型定义统一在 `payment/types.go`

## 架构设计

```
┌─────────────────────────────────────────┐
│      payment/types.go                   │
│  ─────────────────────────────────────  │
│  【公共类型定义 - 唯一数据源】            │
│                                         │
│  ■ 枚举类型                              │
│    - PaymentChannel (alipay/wechat)   │
│    - TradeType (jsapi/native/app...)   │
│    - OrderStatus (pending/paid...)     │
│    - RefundStatus (pending/success...) │
│                                         │
│  ■ 参数和响应类型                        │
│    - CreateOrderParams                 │
│    - CreateOrderResponse               │
│    - QueryOrderParams                  │
│    - QueryOrderResponse                │
│    - CreateRefundParams                │
│    - CreateRefundResponse              │
│    - ...                               │
│                                         │
│  ■ 业务实体                              │
│    - PaymentOrder                      │
│    - PaymentRefund                     │
│    - NotifyLog                         │
│                                         │
│  ■ 配置类型                              │
│    - MerchantConfig                    │
│    - CertConfig                        │
└─────────────────────────────────────────┘
              ▲
              │ 引用（类型别名）
              │
┌─────────────┴───────────────────────────┐
│   payment/providers/types.go            │
│  ─────────────────────────────────────  │
│  【Provider 层 - 类型别名引用】           │
│                                         │
│  // 引用公共类型                         │
│  type PaymentChannel = payment.         │
│       PaymentChannel                    │
│  type TradeType = payment.TradeType     │
│  ...                                    │
│                                         │
│  // Provider 接口使用公共类型             │
│  type PaymentProvider interface {       │
│    CreateOrder(*payment.                │
│      CreateOrderParams) (...)           │
│    QueryOrder(*payment.                 │
│      QueryOrderParams) (...)            │
│    ...                                  │
│  }                                      │
└─────────────────────────────────────────┘
              ▲
              │ 实现
              │
┌─────────────┴───────────────────────────┐
│   payment/providers/alipay.go           │
│   payment/providers/wechat.go           │
│  ─────────────────────────────────────  │
│  【具体 Provider 实现】                   │
│                                         │
│  func (a *Alipay) CreateOrder(          │
│      params *payment.CreateOrderParams) │
│      (*payment.CreateOrderResponse,     │
│      error) {                           │
│      // 实现...                         │
│  }                                      │
└─────────────────────────────────────────┘
```

## 类型统一表

### 枚举值统一（全部小写）

| 类型 | 常量值 |
|------|--------|
| PaymentChannel | `alipay`, `wechat` |
| TradeType | `jsapi`, `native`, `app`, `h5`, `wap` |
| OrderStatus | `pending`, `paid`, `closed`, `refund` |
| RefundStatus | `pending`, `processing`, `success`, `failed` |

### 类型映射

| payment/types.go | providers/types.go | 说明 |
|------------------|-------------------|------|
| `PaymentChannel` | `type PaymentChannel = payment.PaymentChannel` | 类型别名 |
| `TradeType` | `type TradeType = payment.TradeType` | 类型别名 |
| `OrderStatus` | `type OrderStatus = payment.OrderStatus` | 类型别名 |
| `RefundStatus` | `type RefundStatus = payment.RefundStatus` | 类型别名 |
| `CreateOrderParams` | `*payment.CreateOrderParams` | 直接使用 |
| `CreateOrderResponse` | `*payment.CreateOrderResponse` | 直接使用 |
| ... | ... | ... |

## 优势

### 1. 零冗余
- 没有重复定义
- 只有一个地方需要修改
- 类型自动同步

### 2. 零转换成本
- 不需要类型转换函数
- 不需要字段映射
- 没有转换层的性能开销

### 3. 类型安全
- 编译期保证类型一致
- 不会出现常量值不匹配
- IDE 自动补全和类型检查

### 4. 清晰简单
- 代码量少
- 易于理解
- 维护成本低

## 文件职责

### payment/types.go
- **唯一职责**：定义所有类型
- **包含内容**：
  - 所有枚举类型和常量
  - 所有参数和响应结构体
  - 所有业务实体
  - 所有配置类型

### payment/providers/types.go
- **唯一职责**：定义 Provider 接口
- **包含内容**：
  - 类型别名（引用 payment 包）
  - 常量别名（引用 payment 包）
  - PaymentProvider 接口定义

### payment/providers/{alipay,wechat}.go
- **唯一职责**：实现 PaymentProvider 接口
- **使用类型**：直接使用 `payment.*` 类型

## 使用示例

### Provider 实现
```go
package providers

import "github.com/yaoapp/yao/payment"

type Alipay struct {
    // ...
}

// 直接使用 payment 包的类型，无需转换
func (a *Alipay) CreateOrder(
    params *payment.CreateOrderParams,
) (*payment.CreateOrderResponse, error) {
    // 参数直接可用
    amount := params.Amount
    tradeType := params.TradeType
    
    // 返回值直接构造
    return &payment.CreateOrderResponse{
        Success: true,
        OrderID: "xxx",
        // ...
    }, nil
}
```

### 调用层使用
```go
package payment

import "github.com/yaoapp/yao/payment/providers"

func CreateOrder(params *CreateOrderParams) (*CreateOrderResponse, error) {
    // 获取 provider
    provider := getProvider(params.Channel)
    
    // 直接传参，无需转换
    return provider.CreateOrder(params)
}
```

## 注意事项

### 1. 循环依赖
- ✅ `providers` 可以导入 `payment`
- ❌ `payment` 不能导入 `providers`
- 这是单向依赖，符合分层架构

### 2. 向后兼容
- 通过类型别名保持兼容：`type PaymentChannel = payment.PaymentChannel`
- 通过常量别名保持兼容：`const ChannelAlipay = payment.ChannelAlipay`
- 现有代码无需修改即可工作

### 3. 扩展性
新增支付渠道只需：
1. 在 `payment/types.go` 添加必要的扩展字段（如有）
2. 在 `providers/` 下实现新的 Provider
3. 无需修改接口定义或类型系统

## 重构完成状态

✅ **已完成**
- [x] 移除 `providers/types.go` 中的重复定义
- [x] 使用类型别名引用 `payment` 包
- [x] 更新 `PaymentProvider` 接口使用统一类型
- [x] 删除不必要的 `converter.go`
- [x] 更新文档

⬜ **后续工作**（如需要）
- [ ] 更新 `alipay.go` 实现（如果有类型不匹配）
- [ ] 更新 `wechat.go` 实现（如果有类型不匹配）
- [ ] 运行测试确保无破坏性变更

---

**设计哲学**：简单就是美，不要过度设计

**核心思想**：
1. 一个地方定义
2. 到处引用使用
3. 不做任何转换

---

**文档版本**: v2.0.0  
**创建日期**: 2025-03-24  
**最后更新**: 2025-03-24
