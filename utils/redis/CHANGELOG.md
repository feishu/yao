# Redis Utils 更新日志

## 📦 V2.0 - 实用优先，场景化设计

### 设计理念

从 **"提供所有 Redis 命令"** 转变为 **"提供实用的场景化方法"**，让开发者更容易使用。

### ✨ 核心改进

#### 1. 高级方法 - 按业务场景设计

不再需要记忆大量 Redis 命令，直接使用业务场景方法：

```javascript
// JSON 操作 - 直接存取对象
Process("utils.redis.SetJSON", "redis", "user:1001", {
  name: "张三",
  age: 30,
  email: "zhangsan@example.com"
}, 3600);

const user = Process("utils.redis.GetJSON", "redis", "user:1001");
```

```javascript
// 计数器 - 专为统计设计
Process("utils.redis.CounterIncr", "redis", "article:100:views");
Process("utils.redis.CounterDecr", "redis", "product:200:stock");
const count = Process("utils.redis.CounterGet", "redis", "article:100:views");
```

```javascript
// 排行榜 - 完整的排行榜解决方案
Process("utils.redis.RankingAdd", "redis", "game_score", "player123", 1500);
Process("utils.redis.RankingIncrBy", "redis", "game_score", "player123", 100);
const top10 = Process("utils.redis.RankingTop", "redis", "game_score", 10);
// Returns: [{ rank: 1, member: "player999", score: 2000 }, ...]

const myRank = Process("utils.redis.RankingGetRank", "redis", "game_score", "player123");
// Returns: { rank: 2, score: 1600 }
```

```javascript
// 批量操作 - 一次操作多个键
Process("utils.redis.MSet", "redis", {
  "key1": "value1",
  "key2": "value2",
  "key3": "value3"
});

const values = Process("utils.redis.MGet", "redis", "key1", "key2", "key3");
// Returns: { key1: "value1", key2: "value2", key3: "value3" }
```

```javascript
// Pipeline - 批量执行不同类型命令
const results = Process("utils.redis.Batch", "redis", [
  { cmd: "SET", args: ["key1", "value1"] },
  { cmd: "HSET", args: ["user:1", "name", "Alice"] },
  { cmd: "ZADD", args: ["scores", 100, "player1"] },
  { cmd: "UNLINK", args: ["oldkey"] }
]);
```

```javascript
// 清理工具 - 安全批量删除
const deleted = Process("utils.redis.ClearPattern", "redis", "temp:*");
console.log(`清理了 ${deleted} 个临时键`);
```

#### 2. 调用格式统一

**当前版本**（简洁清晰）：
```javascript
Process("utils.redis.MethodName", "connectorName", ...args)
```

- 第一个参数：连接器名称（如 "redis"）
- 方法名：大写开头，符合 Golang 规范
- 易于理解和维护

#### 3. 方法精简

从 54 个基础方法精简为：
- **15 个高级方法** - 覆盖 80% 的使用场景
- **31 个基础方法** - 满足细粒度控制需求

**总计：46 个方法**，更易记忆，更实用。

### 🌟 新增高级方法

#### JSON 操作（2个）

| 方法 | 说明 |
|------|------|
| `SetJSON` | 存储 JSON 对象，自动序列化 |
| `GetJSON` | 获取 JSON 对象，自动反序列化 |

#### 批量操作（2个）

| 方法 | 说明 |
|------|------|
| `MSet` | 批量设置多个键值对 |
| `MGet` | 批量获取多个键的值 |

#### 计数器（4个）

| 方法 | 说明 |
|------|------|
| `CounterIncr` | 增加计数（默认+1） |
| `CounterDecr` | 减少计数（默认-1） |
| `CounterGet` | 获取计数值 |
| `CounterReset` | 重置计数器 |

#### 排行榜（5个）

| 方法 | 说明 |
|------|------|
| `RankingAdd` | 添加/更新排行成员 |
| `RankingIncrBy` | 增加成员分数 |
| `RankingTop` | 获取前N名 |
| `RankingGetRank` | 获取成员排名和分数 |
| `RankingRemove` | 移除排行成员 |

#### 工具方法（2个）

| 方法 | 说明 |
|------|------|
| `Batch` | Pipeline 简化版，批量执行命令 |
| `ClearPattern` | 安全批量删除（SCAN + Pipeline Unlink） |

### 🚀 性能优化

#### ClearPattern - 安全高效的批量删除

**特点**：
- 使用 SCAN 逐批扫描（每批 1000 个键）
- 使用 Pipeline + Unlink 批量删除
- 不阻塞 Redis 服务器
- 适合清理大量过期数据

**性能对比**：

```javascript
// ❌ 不推荐（会阻塞服务器）
const keys = Process("utils.redis.Keys", "redis", "temp:*");
keys.forEach(key => {
  Process("utils.redis.Del", "redis", key);
});

// ✅ 推荐（不阻塞，高性能）
const deleted = Process("utils.redis.ClearPattern", "redis", "temp:*");
```

#### Pipeline 支持 UNLINK

Batch 方法中的 Pipeline 支持 UNLINK 命令，非阻塞删除大键。

```javascript
const results = Process("utils.redis.Batch", "redis", [
  { cmd: "UNLINK", args: ["bigkey1", "bigkey2"] }
]);
```

### 📋 完整方法列表

#### 高级方法（15个）⭐

**JSON（2个）**
- SetJSON - 存储 JSON 对象
- GetJSON - 获取 JSON 对象

**批量（2个）**
- MSet - 批量设置
- MGet - 批量获取

**计数器（4个）**
- CounterIncr - 增加计数
- CounterDecr - 减少计数
- CounterGet - 获取计数
- CounterReset - 重置计数

**排行榜（5个）**
- RankingAdd - 添加成员
- RankingIncrBy - 增加分数
- RankingTop - 获取前N名
- RankingGetRank - 获取排名
- RankingRemove - 移除成员

**工具（2个）**
- Batch - 批量命令
- ClearPattern - 安全删除

#### 基础方法（31个）

**String（8个）**
- Get, Set, Del, Unlink, Exists, Expire, TTL, Keys

**Hash（5个）**
- HGet, HSet, HMSet, HGetAll, HDel

**List（6个）**
- LPush, RPush, LPop, RPop, LRange, LLen

**Set（3个）**
- SAdd, SRem, SMembers

**ZSet（9个）**
- ZAdd, ZRem, ZRange, ZRevRange, ZScore, ZCard, ZIncrBy, ZRank, ZRevRank

### 🔧 技术改进

#### 循环导入问题已解决 ✅

**问题**：
```
connector_test → test → utils → utils/conn → connector
```

**解决方案**：
1. 删除 `connector/redis.go`
2. 将 `LoadAllClients` 移到 `utils/redis/redis.go`
3. 在 `utils/process.go` 的 `Init()` 中调用
4. 简化 `connector_test.go`，移除对 `test` 包的依赖

#### 自动加载机制

系统启动时自动加载所有 Redis 连接器：
```
1. connector.Load() - 加载所有连接器
2. utils.Init() - 初始化工具包
3. redis.LoadAllClients() - 自动注册所有 Redis 客户端
```

### 🎨 使用场景示例

#### 电商系统

```javascript
// 商品缓存
Process("utils.redis.SetJSON", "redis", "product:1001", {
  id: 1001,
  name: "iPhone 15 Pro",
  price: 7999,
  stock: 100
}, 3600);

// 库存扣减
const stock = Process("utils.redis.CounterDecr", "redis", "product:1001:stock");

// 销量排行
Process("utils.redis.RankingIncrBy", "redis", "products:sales", "product:1001", 1);
const hotProducts = Process("utils.redis.RankingTop", "redis", "products:sales", 10);
```

#### 内容管理系统

```javascript
// 文章统计
Process("utils.redis.CounterIncr", "redis", "article:100:views");
Process("utils.redis.CounterIncr", "redis", "article:100:likes");

// 热门文章
Process("utils.redis.RankingIncrBy", "redis", "articles:hot", "article:100", 1);
const hotArticles = Process("utils.redis.RankingTop", "redis", "articles:hot", 20);

// 清理过期缓存
Process("utils.redis.ClearPattern", "redis", "article:temp:*");
```

#### 游戏系统

```javascript
// 记录分数
Process("utils.redis.RankingAdd", "redis", "game:level1", "player123", 8500);

// 增加分数
Process("utils.redis.RankingIncrBy", "redis", "game:level1", "player123", 1500);

// 排行榜
const top10 = Process("utils.redis.RankingTop", "redis", "game:level1", 10);
const myRank = Process("utils.redis.RankingGetRank", "redis", "game:level1", "player123");
```

### 📚 文档更新

- ✅ `README.md` - 完整使用文档
- ✅ `GUIDE.md` - 快速入门指南
- ✅ `CHANGELOG.md` - 更新日志（本文件）

### 💡 使用建议

1. **优先使用高级方法** - 覆盖 80% 的场景
2. **批量操作用 Batch** - 减少网络往返
3. **大量删除用 ClearPattern** - 安全不阻塞
4. **大键删除用 Unlink** - 非阻塞删除
5. **生产环境慎用 Keys** - 可能阻塞服务器

### 🎯 设计对比

| 维度 | 之前 | 现在 |
|------|------|------|
| 方法数量 | 54个 | 46个（精简） |
| 学习曲线 | 陡峭 | 平缓 |
| 使用方式 | 基础命令 | 场景化方法 |
| JSON 操作 | 手动序列化 | 自动处理 |
| 计数器 | 手动实现 | 专用方法 |
| 排行榜 | 手动实现 | 完整方案 |
| 批量删除 | 可能阻塞 | 安全高效 |

### ⚡ 性能提升

- **批量操作** - MSet/MGet 减少网络往返
- **Pipeline** - Batch 自动使用 Pipeline
- **非阻塞删除** - Unlink 和 ClearPattern
- **SCAN 替代 KEYS** - ClearPattern 内部使用 SCAN

### 🔗 迁移指南

如果你之前使用其他 Redis 库，迁移很简单：

**其他库**：
```javascript
redis.set("key", JSON.stringify(data));
const value = JSON.parse(redis.get("key"));
redis.incr("counter");
```

**现在**：
```javascript
Process("utils.redis.SetJSON", "redis", "key", data);
const value = Process("utils.redis.GetJSON", "redis", "key");
Process("utils.redis.CounterIncr", "redis", "counter");
```

---

## 📊 版本总结

### V2.0 核心特性
- **高级方法** - 15个场景化封装，覆盖 80% 的使用场景
- **精简实用** - 从 54 个方法优化到 46 个，更易记忆
- **性能优化** - 批量操作、Pipeline、安全删除
- **易用性** - 按业务场景设计，开箱即用

### 当前状态
- **总方法数**: 46 个（15 高级 + 31 基础）
- **覆盖场景**: 缓存、计数、排行榜、批量操作、数据清理
- **文档完善**: README + GUIDE + CHANGELOG

✅ **更简单** - 高级方法覆盖常见场景  
✅ **更实用** - 按业务场景设计  
✅ **更高效** - 批量操作和性能优化  
✅ **更安全** - ClearPattern 不阻塞服务器  
✅ **更易学** - 方法精简，文档完善  

开始使用新版本，享受更好的开发体验！
