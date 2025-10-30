# Redis Utils 快速开始

## 🚀 5 分钟上手

### 1. 配置 Redis 连接

创建文件 `connectors/redis.yao`:

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

### 2. 基础使用

调用格式：`Process("utils.redis.Method", "connectorName", ...args)`

```javascript
// 存取字符串
Process("utils.redis.Set", "redis", "key1", "value1", 60);  // 60秒过期
const value = Process("utils.redis.Get", "redis", "key1");

// 存取 JSON 对象（推荐）
Process("utils.redis.SetJSON", "redis", "user:1001", {
  name: "张三",
  age: 30,
  email: "zhangsan@example.com"
}, 3600);

const user = Process("utils.redis.GetJSON", "redis", "user:1001");
// Returns: { name: "张三", age: 30, email: "zhangsan@example.com" }
```

### 3. 高级功能

#### 计数器

```javascript
// 浏览量统计
const views = Process("utils.redis.CounterIncr", "redis", "article:100:views");
console.log(`浏览量: ${views}`);

// 库存扣减
const stock = Process("utils.redis.CounterDecr", "redis", "product:200:stock");
```

#### 排行榜

```javascript
// 添加玩家分数
Process("utils.redis.RankingAdd", "redis", "game_score", "player123", 1500);

// 获取前10名
const top10 = Process("utils.redis.RankingTop", "redis", "game_score", 10);
// Returns: [
//   { rank: 1, member: "player999", score: 2000 },
//   { rank: 2, member: "player123", score: 1500 },
//   ...
// ]

// 查询个人排名
const myRank = Process("utils.redis.RankingGetRank", "redis", "game_score", "player123");
// Returns: { rank: 2, score: 1500 }
```

#### 批量操作

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

### 4. 数据结构操作

#### Hash（哈希表）

```javascript
// 设置用户信息
Process("utils.redis.HMSet", "redis", "user:1001", {
  name: "John",
  age: 30,
  email: "john@example.com"
});

// 获取所有字段
const user = Process("utils.redis.HGetAll", "redis", "user:1001");

// 获取单个字段
const name = Process("utils.redis.HGet", "redis", "user:1001", "name");
```

#### List（列表）

```javascript
// 任务队列
Process("utils.redis.LPush", "redis", "tasks", "task1", "task2");
const task = Process("utils.redis.RPop", "redis", "tasks");
```

#### Set（集合）

```javascript
// 标签管理
Process("utils.redis.SAdd", "redis", "article:tags", "go", "redis", "yao");
const tags = Process("utils.redis.SMembers", "redis", "article:tags");
```

#### Sorted Set（有序集合）

```javascript
// 排行榜（基础方式）
Process("utils.redis.ZAdd", "redis", "scores", { "player1": 100, "player2": 85 });
const top = Process("utils.redis.ZRevRange", "redis", "scores", 0, 9, true);
```

### 5. 实用工具

#### 批量清理

```javascript
// 安全删除大量匹配的键（不阻塞服务器）
const deleted = Process("utils.redis.ClearPattern", "redis", "temp:*");
console.log(`清理了 ${deleted} 个临时键`);
```

#### Pipeline 批量命令

```javascript
// 批量执行多个命令
const results = Process("utils.redis.Batch", "redis", [
  { cmd: "SET", args: ["key1", "value1"] },
  { cmd: "GET", args: ["key1"] },
  { cmd: "INCR", args: ["counter"] },
  { cmd: "HSET", args: ["user:1", "name", "Alice"] }
]);
```

## 📊 完整方法列表

### 高级方法（15个）⭐ 推荐优先使用

| 方法 | 说明 |
|------|------|
| `SetJSON` / `GetJSON` | JSON 对象存取 |
| `MSet` / `MGet` | 批量键值操作 |
| `CounterIncr` / `CounterDecr` / `CounterGet` / `CounterReset` | 计数器 |
| `RankingAdd` / `RankingIncrBy` / `RankingTop` / `RankingGetRank` / `RankingRemove` | 排行榜 |
| `Batch` | 批量命令 |
| `ClearPattern` | 批量清理 |

### 基础方法（31个）

- **String**: Get, Set, Del, Unlink, Exists, Expire, TTL, Keys
- **Hash**: HGet, HSet, HMSet, HGetAll, HDel
- **List**: LPush, RPush, LPop, RPop, LRange, LLen
- **Set**: SAdd, SRem, SMembers
- **ZSet**: ZAdd, ZRem, ZRange, ZRevRange, ZScore, ZCard, ZIncrBy, ZRank, ZRevRank

## 🎯 常见场景

### 电商系统

```javascript
// 商品缓存
Process("utils.redis.SetJSON", "redis", "product:1001", {
  name: "iPhone 15 Pro",
  price: 7999,
  stock: 100
}, 3600);

// 库存扣减
const stock = Process("utils.redis.CounterDecr", "redis", "product:1001:stock");

// 销量排行
Process("utils.redis.RankingIncrBy", "redis", "products:sales", "product:1001", 1);
```

### 内容系统

```javascript
// 文章统计
Process("utils.redis.CounterIncr", "redis", "article:100:views");
Process("utils.redis.CounterIncr", "redis", "article:100:likes");

// 热门排行
Process("utils.redis.RankingIncrBy", "redis", "articles:hot", "article:100", 1);
const hotArticles = Process("utils.redis.RankingTop", "redis", "articles:hot", 20);
```

### 游戏系统

```javascript
// 玩家分数
Process("utils.redis.RankingAdd", "redis", "game:level1", "player123", 8500);

// 查询排行
const top10 = Process("utils.redis.RankingTop", "redis", "game:level1", 10);
const myRank = Process("utils.redis.RankingGetRank", "redis", "game:level1", "player123");
```

## 🔗 下一步

- 查看 [README.md](./README.md) - 完整 API 文档
- 查看 [GUIDE.md](./GUIDE.md) - 详细使用指南
- 查看 [CHANGELOG.md](./CHANGELOG.md) - 更新日志
