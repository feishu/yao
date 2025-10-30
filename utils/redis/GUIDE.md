# Redis Utils 使用指南

## 🎯 设计理念

**简洁实用，按场景分组**，提供高级封装方法，让开发者无需记忆大量 Redis 命令。

## 📚 方法分类

### 🌟 高级方法（推荐使用）

#### 1. JSON 操作 - 直接存取 JSON 对象

```javascript
// 存储 JSON 对象（自动序列化）
Process("utils.redis.SetJSON", "redis", "user:1001", {
  name: "张三",
  age: 30,
  email: "zhangsan@example.com"
}, 3600);  // 可选：过期时间

// 获取 JSON 对象（自动反序列化）
const user = Process("utils.redis.GetJSON", "redis", "user:1001");
// Returns: { name: "张三", age: 30, email: "zhangsan@example.com" }
```

#### 2. 批量操作 - 一次操作多个键

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

#### 3. 计数器 - 页面访问、点赞统计等

```javascript
// 增加计数（默认 +1）
const views = Process("utils.redis.CounterIncr", "redis", "article:100:views");
// Returns: 101

// 增加指定值
const likes = Process("utils.redis.CounterIncr", "redis", "article:100:likes", 10);
// Returns: 10

// 减少计数
const stock = Process("utils.redis.CounterDecr", "redis", "product:200:stock");
// Returns: 99

// 获取计数值
const count = Process("utils.redis.CounterGet", "redis", "article:100:views");
// Returns: 101

// 重置计数
Process("utils.redis.CounterReset", "redis", "article:100:views", 0);
```

#### 4. 排行榜 - 游戏分数、热度排名等

```javascript
// 添加/更新排行榜成员
Process("utils.redis.RankingAdd", "redis", "game_score", "player123", 1500);

// 增加分数
const newScore = Process("utils.redis.RankingIncrBy", "redis", "game_score", "player123", 100);
// Returns: 1600

// 获取前 10 名
const top10 = Process("utils.redis.RankingTop", "redis", "game_score", 10);
// Returns: [
//   { rank: 1, member: "player999", score: 2000 },
//   { rank: 2, member: "player123", score: 1600 },
//   ...
// ]

// 获取某个成员的排名和分数
const myRank = Process("utils.redis.RankingGetRank", "redis", "game_score", "player123");
// Returns: { rank: 2, score: 1600 }

// 移除排行榜成员
Process("utils.redis.RankingRemove", "redis", "game_score", "player456");
```

#### 5. 批量操作 - Pipeline 简化版

```javascript
// 批量执行多个命令（自动使用 Pipeline）
const results = Process("utils.redis.Batch", "redis", [
  { cmd: "SET", args: ["key1", "value1"] },
  { cmd: "GET", args: ["key1"] },
  { cmd: "HSET", args: ["user:1", "name", "Alice"] },
  { cmd: "HGETALL", args: ["user:1"] },
  { cmd: "ZADD", args: ["scores", 100, "player1"] }
]);
// Returns: ["OK", "value1", 1, { name: "Alice" }, 1]
```

#### 6. 清理工具 - 批量删除过期数据

```javascript
// 批量删除匹配的键（使用 SCAN + Pipeline，安全不阻塞）
const deleted = Process("utils.redis.ClearPattern", "redis", "temp:*");
// Returns: 1500 (删除的键总数)

// 常见场景
Process("utils.redis.ClearPattern", "redis", "session:expired:*");
Process("utils.redis.ClearPattern", "redis", "cache:2023-10-29:*");
```

### 📦 基础操作

```javascript
// 字符串操作
Process("utils.redis.Get", "redis", "key");
Process("utils.redis.Set", "redis", "key", "value", 60);  // 60秒过期
Process("utils.redis.Del", "redis", "key1", "key2");
Process("utils.redis.Exists", "redis", "key");
Process("utils.redis.Expire", "redis", "key", 3600);
Process("utils.redis.TTL", "redis", "key");
Process("utils.redis.Unlink", "redis", "bigkey");  // 非阻塞删除大键
Process("utils.redis.Keys", "redis", "user:*");     // 查找匹配的键
```

### 🗂️ Hash 操作

```javascript
// Hash 操作
Process("utils.redis.HSet", "redis", "user:1", "name", "John");
Process("utils.redis.HGet", "redis", "user:1", "name");
Process("utils.redis.HMSet", "redis", "user:1", { name: "John", age: 30 });
Process("utils.redis.HGetAll", "redis", "user:1");
Process("utils.redis.HDel", "redis", "user:1", "age");
```

### 📋 List 操作

```javascript
// List 操作
Process("utils.redis.LPush", "redis", "queue", "task1", "task2");
Process("utils.redis.RPush", "redis", "queue", "task3");
Process("utils.redis.LPop", "redis", "queue");
Process("utils.redis.RPop", "redis", "queue");
Process("utils.redis.LRange", "redis", "queue", 0, -1);
Process("utils.redis.LLen", "redis", "queue");
```

### 🎲 Set 操作

```javascript
// Set 操作
Process("utils.redis.SAdd", "redis", "tags", "go", "redis", "yao");
Process("utils.redis.SRem", "redis", "tags", "redis");
Process("utils.redis.SMembers", "redis", "tags");
```

### 🏆 Sorted Set 操作

```javascript
// Sorted Set 操作
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

## 🎨 实用场景示例

### 场景1：用户信息缓存

```javascript
// 使用 SetJSON 存储用户信息
Process("utils.redis.SetJSON", "redis", "user:1001", {
  id: 1001,
  name: "张三",
  email: "zhangsan@example.com",
  role: "admin",
  permissions: ["read", "write", "delete"]
}, 3600);

// 使用 GetJSON 获取用户信息
const user = Process("utils.redis.GetJSON", "redis", "user:1001");
if (user) {
  console.log(`欢迎回来，${user.name}`);
}
```

### 场景2：文章统计

```javascript
// 增加浏览量
Process("utils.redis.CounterIncr", "redis", "article:100:views");

// 增加点赞
Process("utils.redis.CounterIncr", "redis", "article:100:likes");

// 获取统计数据
const views = Process("utils.redis.CounterGet", "redis", "article:100:views");
const likes = Process("utils.redis.CounterGet", "redis", "article:100:likes");

console.log(`浏览: ${views}, 点赞: ${likes}`);
```

### 场景3：游戏排行榜

```javascript
// 玩家完成游戏，记录分数
Process("utils.redis.RankingAdd", "redis", "game:level1:scores", "player123", 8500);

// 玩家再次挑战，增加分数
Process("utils.redis.RankingIncrBy", "redis", "game:level1:scores", "player123", 1500);

// 获取全服前 10 名
const top10 = Process("utils.redis.RankingTop", "redis", "game:level1:scores", 10);
console.log("排行榜前10名：", top10);

// 查询玩家自己的排名
const myRank = Process("utils.redis.RankingGetRank", "redis", "game:level1:scores", "player123");
console.log(`你的排名：第 ${myRank.rank} 名，分数：${myRank.score}`);
```

### 场景4：热点商品缓存

```javascript
// 批量缓存商品信息
const products = [
  { id: 1, name: "商品A", price: 99 },
  { id: 2, name: "商品B", price: 199 },
  { id: 3, name: "商品C", price: 299 }
];

products.forEach(p => {
  Process("utils.redis.SetJSON", "redis", `product:${p.id}`, p, 7200);
});

// 批量获取多个商品
const productIds = ["product:1", "product:2", "product:3"];
const cachedProducts = Process("utils.redis.MGet", "redis", ...productIds);
```

### 场景5：定时清理过期数据

```javascript
// 每天凌晨清理昨天的临时数据
const yesterday = new Date();
yesterday.setDate(yesterday.getDate() - 1);
const dateStr = yesterday.toISOString().split('T')[0];

const deleted = Process("utils.redis.ClearPattern", "redis", `temp:${dateStr}:*`);
console.log(`清理了 ${deleted} 个过期临时键`);
```

### 场景6：批量初始化数据

```javascript
// 使用 Batch 批量初始化
const results = Process("utils.redis.Batch", "redis", [
  // 设置系统配置
  { cmd: "SET", args: ["config:maintenance", "false"] },
  { cmd: "SET", args: ["config:version", "1.0.0"] },
  
  // 初始化计数器
  { cmd: "SET", args: ["counter:total_users", "0"] },
  { cmd: "SET", args: ["counter:total_orders", "0"] },
  
  // 初始化排行榜
  { cmd: "ZADD", args: ["ranking:daily", 0, "system"] }
]);

console.log("初始化完成：", results);
```

## 📊 方法对比

### JSON vs 基础方法

```javascript
// ❌ 繁琐的方式
const user = { name: "John", age: 30 };
Process("utils.redis.Set", "redis", "user:1", JSON.stringify(user));
const data = Process("utils.redis.Get", "redis", "user:1");
const user2 = JSON.parse(data);

// ✅ 简洁的方式
Process("utils.redis.SetJSON", "redis", "user:1", { name: "John", age: 30 });
const user2 = Process("utils.redis.GetJSON", "redis", "user:1");
```

### 计数器 vs 基础方法

```javascript
// ❌ 繁琐的方式
Process("utils.redis.Set", "redis", "views", "0");
const current = parseInt(Process("utils.redis.Get", "redis", "views") || "0");
Process("utils.redis.Set", "redis", "views", (current + 1).toString());

// ✅ 简洁的方式
Process("utils.redis.CounterIncr", "redis", "views");
```

### 排行榜 vs 基础方法

```javascript
// ❌ 繁琐的方式
Process("utils.redis.ZAdd", "redis", "scores", { "player1": 100 });
const members = Process("utils.redis.ZRevRange", "redis", "scores", 0, 9, true);
const top10 = members.map((m, i) => ({
  rank: i + 1,
  member: m.member,
  score: m.score
}));

// ✅ 简洁的方式
Process("utils.redis.RankingAdd", "redis", "scores", "player1", 100);
const top10 = Process("utils.redis.RankingTop", "redis", "scores", 10);
```

## 🎯 方法选择指南

| 场景 | 推荐方法 | 说明 |
|------|---------|------|
| 存储 JSON 对象 | `SetJSON` / `GetJSON` | 自动序列化，更方便 |
| 批量键值操作 | `MSet` / `MGet` | 一次操作多个键 |
| 计数统计 | `CounterIncr` / `CounterDecr` | 专为计数设计 |
| 排行榜系统 | `Ranking*` 系列 | 完整的排行榜解决方案 |
| 批量命令执行 | `Batch` | 自动使用 Pipeline |
| 清理大量键 | `ClearPattern` | 安全不阻塞 |
| 简单字符串 | `Get` / `Set` | 最基础的操作 |
| 删除大键 | `Unlink` | 比 `Del` 更快 |

## 总结

### 优先使用高级方法 🌟
- `SetJSON` / `GetJSON` - 存取 JSON
- `MSet` / `MGet` - 批量操作
- `Counter*` - 计数器
- `Ranking*` - 排行榜
- `Batch` - 批量命令
- `ClearPattern` - 安全清理

### 基础方法作为补充 📦
- 当高级方法不满足需求时使用
- 用于更细粒度的控制
