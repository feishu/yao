# Redis Utils 使用文档

## 🎯 特点

- ✅ **实用优先** - 提供高级封装方法，按业务场景设计
- ✅ **简单易用** - 无需记忆大量 Redis 命令
- ✅ **自动加载** - 系统启动时自动注册 Redis 连接器
- ✅ **多连接支持** - 支持同时使用多个 Redis 实例

## 📦 连接配置

在 `connectors` 目录下创建 Redis 连接配置，例如 `connectors/redis.yao`:

```json
{
  "name": "Redis Cache",
  "options": {
    "host": "127.0.0.1",
    "port": "6379",
    "pass": "",
    "db": "0",
    "timeout": 5
  }
}
```

系统启动时会自动加载所有 Redis 连接器。

## 🚀 快速开始

### 调用格式

```javascript
Process("utils.redis.MethodName", "connectorName", ...args)
```

- **第一个参数**：连接器名称（如 "redis"）
- **后续参数**：具体操作参数

### 基础示例

```javascript
// 存取字符串
Process("utils.redis.Set", "redis", "key1", "value1", 60);
const value = Process("utils.redis.Get", "redis", "key1");

// 存取 JSON 对象（推荐）
Process("utils.redis.SetJSON", "redis", "user:1001", {
  name: "张三",
  age: 30
}, 3600);
const user = Process("utils.redis.GetJSON", "redis", "user:1001");

// 计数器
const views = Process("utils.redis.CounterIncr", "redis", "article:100:views");

// 排行榜
Process("utils.redis.RankingAdd", "redis", "game_score", "player123", 1500);
const top10 = Process("utils.redis.RankingTop", "redis", "game_score", 10);
```

## 🌟 核心方法（按场景分类）

### 1. JSON 操作

直接存取 JSON 对象，自动序列化/反序列化。

```javascript
// 存储 JSON
Process("utils.redis.SetJSON", "redis", "user:1001", {
  id: 1001,
  name: "张三",
  email: "zhangsan@example.com",
  role: "admin"
}, 3600);  // 可选：过期时间（秒）

// 获取 JSON
const user = Process("utils.redis.GetJSON", "redis", "user:1001");
// Returns: { id: 1001, name: "张三", ... }
```

### 2. 批量操作

一次操作多个键，提升性能。

```javascript
// 批量设置
Process("utils.redis.MSet", "redis", {
  "key1": "value1",
  "key2": "value2",
  "key3": "value3"
});

// 批量获取
const values = Process("utils.redis.MGet", "redis", "key1", "key2", "key3");
// Returns: { key1: "value1", key2: "value2", key3: "value3" }
```

### 3. 计数器

页面访问、点赞统计、库存管理等。

```javascript
// 增加计数（默认 +1）
const views = Process("utils.redis.CounterIncr", "redis", "article:100:views");

// 增加指定值
const likes = Process("utils.redis.CounterIncr", "redis", "article:100:likes", 10);

// 减少计数
const stock = Process("utils.redis.CounterDecr", "redis", "product:200:stock");

// 获取当前值
const count = Process("utils.redis.CounterGet", "redis", "article:100:views");

// 重置计数器
Process("utils.redis.CounterReset", "redis", "article:100:views", 0);
```

### 4. 排行榜

游戏分数、热度排名、销量排行等。

```javascript
// 添加/更新成员分数
Process("utils.redis.RankingAdd", "redis", "game_score", "player123", 1500);

// 增加分数
const newScore = Process("utils.redis.RankingIncrBy", "redis", "game_score", "player123", 100);

// 获取前 N 名
const top10 = Process("utils.redis.RankingTop", "redis", "game_score", 10);
// Returns: [
//   { rank: 1, member: "player999", score: 2000 },
//   { rank: 2, member: "player123", score: 1600 },
//   ...
// ]

// 获取成员排名和分数
const myRank = Process("utils.redis.RankingGetRank", "redis", "game_score", "player123");
// Returns: { rank: 2, score: 1600 }

// 移除成员
Process("utils.redis.RankingRemove", "redis", "game_score", "player456");
```

### 5. Pipeline 批量命令

批量执行多个不同类型的命令。

```javascript
const results = Process("utils.redis.Batch", "redis", [
  { cmd: "SET", args: ["key1", "value1"] },
  { cmd: "GET", args: ["key1"] },
  { cmd: "HSET", args: ["user:1", "name", "Alice"] },
  { cmd: "HGETALL", args: ["user:1"] },
  { cmd: "INCR", args: ["counter"] },
  { cmd: "ZADD", args: ["scores", 100, "player1"] },
  { cmd: "UNLINK", args: ["oldkey1", "oldkey2"] }
]);
// Returns: ["OK", "value1", 1, {...}, 1, 1, 2]
```

**支持的命令**：GET, SET, DEL, UNLINK, EXISTS, EXPIRE, TTL, INCR, DECR, INCRBY, DECRBY, HGET, HSET, HGETALL, HDEL, LPUSH, RPUSH, LPOP, RPOP, LRANGE, LLEN, SADD, SREM, SMEMBERS, SISMEMBER, SCARD, ZADD, ZREM, ZRANGE, ZREVRANGE, ZSCORE, ZCARD, ZRANK, ZREVRANK

### 6. 清理工具

安全高效地清理大量数据。

```javascript
// 批量删除匹配的键（使用 SCAN + Pipeline Unlink，不阻塞）
const deleted = Process("utils.redis.ClearPattern", "redis", "temp:*");
console.log(`清理了 ${deleted} 个临时键`);

// 常见场景
Process("utils.redis.ClearPattern", "redis", "session:expired:*");
Process("utils.redis.ClearPattern", "redis", "cache:2023-10-29:*");
```

## 📦 基础操作方法

### String 操作

```javascript
// 基本操作
Process("utils.redis.Get", "redis", "key");
Process("utils.redis.Set", "redis", "key", "value", 60);  // 60秒过期
Process("utils.redis.Del", "redis", "key1", "key2");
Process("utils.redis.Unlink", "redis", "bigkey");  // 非阻塞删除大键
Process("utils.redis.Exists", "redis", "key");
Process("utils.redis.Expire", "redis", "key", 3600);
Process("utils.redis.TTL", "redis", "key");
Process("utils.redis.Keys", "redis", "user:*");    // 查找匹配的键（生产慎用）
```

### Hash 操作

```javascript
Process("utils.redis.HSet", "redis", "user:1", "name", "John");
Process("utils.redis.HGet", "redis", "user:1", "name");
Process("utils.redis.HMSet", "redis", "user:1", { name: "John", age: 30 });
Process("utils.redis.HGetAll", "redis", "user:1");
Process("utils.redis.HDel", "redis", "user:1", "age");
```

### List 操作

```javascript
Process("utils.redis.LPush", "redis", "queue", "task1", "task2");
Process("utils.redis.RPush", "redis", "queue", "task3");
Process("utils.redis.LPop", "redis", "queue");
Process("utils.redis.RPop", "redis", "queue");
Process("utils.redis.LRange", "redis", "queue", 0, -1);
Process("utils.redis.LLen", "redis", "queue");
```

### Set 操作

```javascript
Process("utils.redis.SAdd", "redis", "tags", "go", "redis", "yao");
Process("utils.redis.SRem", "redis", "tags", "redis");
Process("utils.redis.SMembers", "redis", "tags");
```

### Sorted Set 操作

```javascript
Process("utils.redis.ZAdd", "redis", "scores", { "player1": 100, "player2": 85 });
Process("utils.redis.ZRem", "redis", "scores", "player2");
Process("utils.redis.ZRange", "redis", "scores", 0, -1);
Process("utils.redis.ZRevRange", "redis", "scores", 0, 9, true);  // 前10名，带分数
Process("utils.redis.ZScore", "redis", "scores", "player1");
Process("utils.redis.ZCard", "redis", "scores");
Process("utils.redis.ZIncrBy", "redis", "scores", 10, "player1");
Process("utils.redis.ZRank", "redis", "scores", "player1");
Process("utils.redis.ZRevRank", "redis", "scores", "player1");
```

## 🎨 完整使用场景

### 场景1：电商系统

```javascript
// 商品信息缓存
Process("utils.redis.SetJSON", "redis", "product:1001", {
  id: 1001,
  name: "iPhone 15 Pro",
  price: 7999,
  stock: 100
}, 3600);

// 库存扣减
const stock = Process("utils.redis.CounterDecr", "redis", "product:1001:stock");
if (stock < 0) {
  throw new Error("库存不足");
}

// 销量排行
Process("utils.redis.RankingIncrBy", "redis", "products:sales", "product:1001", 1);
const hotProducts = Process("utils.redis.RankingTop", "redis", "products:sales", 10);

// 用户浏览历史
Process("utils.redis.LPush", "redis", "user:1001:history", "product:1001");
const history = Process("utils.redis.LRange", "redis", "user:1001:history", 0, 9);

// 批量创建订单
const results = Process("utils.redis.Batch", "redis", [
  { cmd: "HSET", args: ["order:2001", "user_id", "1001"] },
  { cmd: "HSET", args: ["order:2001", "product_id", "1001"] },
  { cmd: "HSET", args: ["order:2001", "amount", "7999"] },
  { cmd: "HINCRBY", args: ["stats:today", "orders", 1] },
  { cmd: "ZINCRBY", args: ["products:sales", 1, "product:1001"] }
]);
```

### 场景2：内容管理系统

```javascript
// 文章缓存
Process("utils.redis.SetJSON", "redis", "article:100", {
  id: 100,
  title: "Redis 使用指南",
  author: "张三",
  content: "...",
  tags: ["redis", "golang"]
}, 7200);

// 浏览和点赞统计
Process("utils.redis.CounterIncr", "redis", "article:100:views");
Process("utils.redis.CounterIncr", "redis", "article:100:likes");

// 热门文章排行
Process("utils.redis.RankingIncrBy", "redis", "articles:hot", "article:100", 1);
const hotArticles = Process("utils.redis.RankingTop", "redis", "articles:hot", 20);

// 标签管理
Process("utils.redis.SAdd", "redis", "tag:redis:articles", "100", "101", "102");
const articles = Process("utils.redis.SMembers", "redis", "tag:redis:articles");

// 定时清理过期缓存
Process("utils.redis.ClearPattern", "redis", "article:temp:*");
```

### 场景3：游戏系统

```javascript
// 玩家完成游戏，记录分数
Process("utils.redis.RankingAdd", "redis", "game:level1:scores", "player123", 8500);

// 玩家再次挑战，增加分数
Process("utils.redis.RankingIncrBy", "redis", "game:level1:scores", "player123", 1500);

// 获取全服前 10 名
const top10 = Process("utils.redis.RankingTop", "redis", "game:level1:scores", 10);
top10.forEach(item => {
  console.log(`第 ${item.rank} 名：${item.member} - ${item.score}分`);
});

// 查询玩家自己的排名
const myRank = Process("utils.redis.RankingGetRank", "redis", "game:level1:scores", "player123");
console.log(`你的排名：第 ${myRank.rank} 名，分数：${myRank.score}`);
```

## 📊 完整方法列表

### 高级方法（15个）⭐ 推荐优先使用

| 方法 | 说明 | 场景 |
|-----|------|-----|
| `SetJSON` | 存储 JSON 对象 | 用户信息、商品详情 |
| `GetJSON` | 获取 JSON 对象 | 缓存读取 |
| `MSet` | 批量设置键值 | 批量初始化 |
| `MGet` | 批量获取键值 | 批量查询 |
| `CounterIncr` | 增加计数 | 浏览量、点赞数 |
| `CounterDecr` | 减少计数 | 库存扣减 |
| `CounterGet` | 获取计数值 | 统计查询 |
| `CounterReset` | 重置计数器 | 重置统计 |
| `RankingAdd` | 添加排行成员 | 初始化排行 |
| `RankingIncrBy` | 增加排行分数 | 积分增加 |
| `RankingTop` | 获取前N名 | 排行榜展示 |
| `RankingGetRank` | 获取成员排名 | 个人排名查询 |
| `RankingRemove` | 移除排行成员 | 删除记录 |
| `Batch` | 批量命令 | 复杂批量操作 |
| `ClearPattern` | 安全批量删除 | 清理过期数据 |

### 基础方法（31个）

**String**: Get, Set, Del, Unlink, Exists, Expire, TTL, Keys  
**Hash**: HGet, HSet, HMSet, HGetAll, HDel  
**List**: LPush, RPush, LPop, RPop, LRange, LLen  
**Set**: SAdd, SRem, SMembers  
**ZSet**: ZAdd, ZRem, ZRange, ZRevRange, ZScore, ZCard, ZIncrBy, ZRank, ZRevRank

**总计：46 个方法**（15 高级 + 31 基础）

## 💡 最佳实践

### 1. 优先使用高级方法

```javascript
// ❌ 繁琐
const user = { name: "John", age: 30 };
Process("utils.redis.Set", "redis", "user:1", JSON.stringify(user));
const data = Process("utils.redis.Get", "redis", "user:1");
const user2 = JSON.parse(data);

// ✅ 简洁
Process("utils.redis.SetJSON", "redis", "user:1", { name: "John", age: 30 });
const user2 = Process("utils.redis.GetJSON", "redis", "user:1");
```

### 2. 使用计数器而非手动操作

```javascript
// ❌ 繁琐
const current = parseInt(Process("utils.redis.Get", "redis", "views") || "0");
Process("utils.redis.Set", "redis", "views", (current + 1).toString());

// ✅ 简洁
Process("utils.redis.CounterIncr", "redis", "views");
```

### 3. 批量操作提升性能

```javascript
// ❌ 多次网络请求
Process("utils.redis.Set", "redis", "key1", "value1");
Process("utils.redis.Set", "redis", "key2", "value2");
Process("utils.redis.Set", "redis", "key3", "value3");

// ✅ 一次请求
Process("utils.redis.MSet", "redis", {
  "key1": "value1",
  "key2": "value2",
  "key3": "value3"
});
```

### 4. 安全删除大量数据

```javascript
// ❌ 可能阻塞服务器
const keys = Process("utils.redis.Keys", "redis", "temp:*");
keys.forEach(key => Process("utils.redis.Del", "redis", key));

// ✅ 不阻塞
Process("utils.redis.ClearPattern", "redis", "temp:*");
```

## 🔗 相关文档

- [GUIDE.md](./GUIDE.md) - 快速入门指南
- [CHANGELOG.md](./CHANGELOG.md) - 更新日志

## 💡 其他功能

如需使用 **Pub/Sub** 或 **Stream** 等消息功能，建议直接使用成熟的第三方库：
- [go-redis/redis](https://github.com/go-redis/redis) - 功能完整的 Redis 客户端
- 或通过 `connector.Select()` 获取原生 Redis 客户端进行操作
