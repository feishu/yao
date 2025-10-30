# RedisHelper TypeScript 使用指南

## 概述

`RedisHelper` 是一个 TypeScript 类，封装了所有 Yao Redis Process 方法，提供类型安全和便捷的 Redis 操作接口。

## 特性

✅ **完整功能覆盖** - 包含所有 60+ Redis 操作方法
✅ **类型安全** - 完整的 TypeScript 类型定义
✅ **自动 Key 前缀** - 支持命名空间隔离
✅ **详细文档** - 每个方法都有完整的 JSDoc 注释和使用示例
✅ **易于使用** - 简洁的 API 设计

## 快速开始

### 1. 基本用法

```typescript
import { RedisHelper } from './redishelper';

// 创建 Redis 客户端实例
const redis = new RedisHelper("redis");

// 基础操作
redis.set("username", "john", 3600); // 设置值，3600秒过期
const username = redis.get("username"); // 获取值

// JSON 操作
redis.setJSON("user:1001", {
  name: "John Doe",
  age: 30,
  email: "john@example.com"
});

const user = redis.getJSON("user:1001");
```

### 2. 使用 Key 前缀

```typescript
// 为不同的业务模块创建独立的 Redis 实例
const userRedis = new RedisHelper("redis", "user:");
const sessionRedis = new RedisHelper("redis", "session:");
const cacheRedis = new RedisHelper("redis", "cache:");

// 自动添加前缀
userRedis.set("1001", { name: "John" }); // 实际key: "user:1001"
sessionRedis.set("abc123", "data");      // 实际key: "session:abc123"
cacheRedis.set("page1", "html");         // 实际key: "cache:page1"
```

## 功能分类

### 基础 Key-Value 操作

```typescript
// 获取和设置
redis.get("key");
redis.set("key", "value", 60); // 60秒过期

// 删除
redis.del("key1", "key2");
redis.unlink("large_key"); // 非阻塞删除

// 存在检查
if (redis.exists("key") > 0) {
  console.log("键存在");
}

// 过期时间
redis.expire("key", 3600);
const ttl = redis.ttl("key");

// 自增自减
redis.incr("counter");
redis.incrBy("counter", 10);
redis.decr("counter");
redis.decrBy("counter", 5);

// 查找键
const keys = redis.keys("user:*");
```

### JSON 操作

```typescript
interface User {
  name: string;
  age: number;
  email: string;
}

// 存储对象
redis.setJSON("user:1001", {
  name: "John",
  age: 30,
  email: "john@example.com"
}, 3600);

// 获取对象（带类型推断）
const user = redis.getJSON<User>("user:1001");
console.log(user.name, user.age);
```

### 批量操作

```typescript
// 批量设置
redis.mSet({
  "key1": "value1",
  "key2": "value2",
  "key3": "value3"
});

// 批量获取
const values = redis.mGet("key1", "key2", "key3");
console.log(values); // { key1: "value1", key2: "value2", key3: "value3" }
```

### 计数器

```typescript
// 页面浏览量
const views = redis.counterIncr("article:100:views");
console.log(`浏览量: ${views}`);

// 增加指定数量
redis.counterIncr("article:100:likes", 10);

// 获取计数
const count = redis.counterGet("article:100:views");

// 减少计数（库存等）
const stock = redis.counterDecr("product:200:stock");

// 重置计数器
redis.counterReset("daily:views", 0);
```

### 排行榜

```typescript
// 添加玩家分数
redis.rankingAdd("game:level1", "player123", 8500);

// 增加分数
redis.rankingIncrBy("game:level1", "player123", 100);

// 获取前10名
const top10 = redis.rankingTop("game:level1", 10);
top10.forEach(item => {
  console.log(`第${item.rank}名: ${item.member}, 分数: ${item.score}`);
});

// 获取玩家排名
const rankInfo = redis.rankingGetRank("game:level1", "player123");
console.log(`排名: ${rankInfo.rank}, 分数: ${rankInfo.score}`);

// 移除玩家
redis.rankingRemove("game:level1", "player456");
```

### Hash 操作

```typescript
// 设置字段
redis.hSet("user:1001", "name", "John");
redis.hSet("user:1001", "age", 30);

// 批量设置
redis.hMSet("user:1001", {
  name: "John",
  age: 30,
  email: "john@example.com"
});

// 获取字段
const name = redis.hGet("user:1001", "name");

// 获取所有字段
const user = redis.hGetAll("user:1001");

// 删除字段
redis.hDel("user:1001", "email");

// 检查字段存在
if (redis.hExists("user:1001", "name")) {
  console.log("字段存在");
}

// 获取所有字段名和值
const fields = redis.hKeys("user:1001");
const values = redis.hVals("user:1001");

// 字段数量
const count = redis.hLen("user:1001");

// 字段值自增
redis.hIncrBy("user:1001", "points", 100);
```

### List 操作

```typescript
// 添加到列表
redis.lPush("tasks", "task1", "task2"); // 头部插入
redis.rPush("queue", "item1", "item2"); // 尾部插入

// 从列表取出
const task = redis.lPop("tasks"); // 从头部取出
const item = redis.rPop("queue"); // 从尾部取出

// 获取范围
const tasks = redis.lRange("tasks", 0, 9); // 前10个
const all = redis.lRange("tasks", 0, -1);  // 所有

// 列表长度
const length = redis.lLen("tasks");

// 通过索引操作
const first = redis.lIndex("tasks", 0);
redis.lSet("tasks", 0, "new_value");

// 移除元素
redis.lRem("tasks", 0, "task1"); // 移除所有值为"task1"的元素
```

### Set 操作

```typescript
// 添加成员
redis.sAdd("tags", "javascript", "typescript", "nodejs");

// 移除成员
redis.sRem("tags", "outdated");

// 获取所有成员
const tags = redis.sMembers("tags");

// 检查成员
if (redis.sIsMember("tags", "javascript")) {
  console.log("标签存在");
}

// 成员数量
const count = redis.sCard("tags");

// 随机操作
const randomTag = redis.sPop("tags"); // 移除并返回
const randomTags = redis.sRandMember("tags", 3); // 不移除

// 集合运算
const union = redis.sUnion("set1", "set2");      // 并集
const inter = redis.sInter("set1", "set2");      // 交集
const diff = redis.sDiff("set1", "set2");        // 差集
```

### Sorted Set (有序集合) 操作

```typescript
// 添加成员
redis.zAdd("leaderboard", {
  "player1": 1000,
  "player2": 850,
  "player3": 920
});

// 移除成员
redis.zRem("leaderboard", "player2");

// 获取范围（按分数升序）
const members = redis.zRange("leaderboard", 0, 9);
const membersWithScores = redis.zRange("leaderboard", 0, 9, true);

// 获取范围（按分数降序）
const top10 = redis.zRevRange("leaderboard", 0, 9, true);

// 按分数范围获取
const highScorers = redis.zRangeByScore("scores", "90", "100");
const above80 = redis.zRangeByScore("scores", "(80", "+inf"); // 大于80

// 获取分数
const score = redis.zScore("leaderboard", "player1");

// 成员数量
const count = redis.zCard("leaderboard");

// 分数范围内的数量
const count = redis.zCount("scores", "60", "100");

// 增加分数
redis.zIncrBy("leaderboard", 100, "player1");

// 获取排名
const rank = redis.zRank("scores", "student1");     // 升序排名
const rank = redis.zRevRank("leaderboard", "player1"); // 降序排名
```

### 高级操作

```typescript
// 批量删除匹配的键
const deleted = redis.clearPattern("temp:*");
console.log(`删除了 ${deleted} 个临时键`);

redis.clearPattern("session:expired:*");
redis.clearPattern("cache:2024-01-*");

// Pipeline 批量命令
const results = redis.batch([
  { cmd: "SET", args: ["key1", "value1"] },
  { cmd: "SET", args: ["key2", "value2"] },
  { cmd: "GET", args: ["key1"] },
  { cmd: "INCR", args: ["counter"] }
]);

// 或使用 pipeline 方法（功能相同）
const results = redis.pipeline([
  { cmd: "HSET", args: ["user:1", "name", "John"] },
  { cmd: "HSET", args: ["user:1", "age", 30] },
  { cmd: "HGETALL", args: ["user:1"] }
]);
```

## 实际应用场景

### 1. 用户会话管理

```typescript
const sessionRedis = new RedisHelper("redis", "session:");

// 创建会话
sessionRedis.setJSON("abc123", {
  userId: 1001,
  username: "john",
  loginTime: Date.now()
}, 3600); // 1小时过期

// 检查会话
if (sessionRedis.exists("abc123") > 0) {
  const session = sessionRedis.getJSON("abc123");
  console.log("用户:", session.username);
}

// 延长会话
sessionRedis.expire("abc123", 7200);

// 销毁会话
sessionRedis.del("abc123");
```

### 2. 缓存管理

```typescript
const cacheRedis = new RedisHelper("redis", "cache:");

// 缓存文章列表
cacheRedis.setJSON("articles:latest", articles, 300); // 5分钟缓存

// 获取缓存
const cached = cacheRedis.getJSON("articles:latest");
if (cached) {
  return cached; // 使用缓存
} else {
  // 查询数据库
  const articles = queryDatabase();
  cacheRedis.setJSON("articles:latest", articles, 300);
  return articles;
}

// 清理过期缓存
cacheRedis.clearPattern("cache:2024-01-*");
```

### 3. 实时统计

```typescript
const statsRedis = new RedisHelper("redis", "stats:");

// 文章浏览
statsRedis.counterIncr("article:100:views");
statsRedis.counterIncr("article:100:unique_ips");

// 点赞
statsRedis.counterIncr("article:100:likes");

// 获取统计
const views = statsRedis.counterGet("article:100:views");
const likes = statsRedis.counterGet("article:100:likes");

// 每日统计重置
statsRedis.counterReset("daily:total_views");
```

### 4. 游戏排行榜

```typescript
const gameRedis = new RedisHelper("redis", "game:");

// 更新玩家分数
gameRedis.rankingAdd("level1:scores", "player123", 8500);

// 玩家完成任务，增加分数
gameRedis.rankingIncrBy("level1:scores", "player123", 1500);

// 显示排行榜
const top10 = gameRedis.rankingTop("level1:scores", 10);
console.log("== 排行榜 ==");
top10.forEach(item => {
  console.log(`${item.rank}. ${item.member} - ${item.score}分`);
});

// 显示玩家自己的排名
const myRank = gameRedis.rankingGetRank("level1:scores", "player123");
console.log(`你的排名: ${myRank.rank}, 分数: ${myRank.score}`);
```

### 5. 库存管理

```typescript
const inventoryRedis = new RedisHelper("redis", "inventory:");

// 初始化库存
inventoryRedis.counterReset("product:200", 100);

// 下单减库存
const stock = inventoryRedis.counterDecr("product:200", 1);
if (stock < 0) {
  // 库存不足，回滚
  inventoryRedis.counterIncr("product:200", 1);
  throw new Error("库存不足");
}

// 补货
inventoryRedis.counterIncr("product:200", 50);

// 查看库存
const currentStock = inventoryRedis.counterGet("product:200");
```

### 6. 消息队列

```typescript
const queueRedis = new RedisHelper("redis", "queue:");

// 生产者：添加任务
queueRedis.rPush("email_queue", JSON.stringify({
  to: "user@example.com",
  subject: "Welcome",
  body: "Thank you for joining!"
}));

// 消费者：处理任务
const task = queueRedis.lPop("email_queue");
if (task) {
  const email = JSON.parse(task);
  sendEmail(email);
}

// 查看队列长度
const queueLength = queueRedis.lLen("email_queue");
console.log(`待处理任务: ${queueLength}`);
```

## 最佳实践

### 1. 使用合适的数据结构

- **String**: 简单的键值对、计数器、缓存
- **Hash**: 对象存储、用户信息
- **List**: 队列、栈、最近访问记录
- **Set**: 标签、去重、交集/并集运算
- **Sorted Set**: 排行榜、带权重的队列

### 2. Key 命名规范

```typescript
// 使用前缀分组
const userRedis = new RedisHelper("redis", "user:");
const orderRedis = new RedisHelper("redis", "order:");

// 使用分隔符
userRedis.set("1001:profile", data);    // user:1001:profile
userRedis.set("1001:settings", data);   // user:1001:settings
```

### 3. 设置合理的过期时间

```typescript
// 短期数据
redis.set("captcha:abc", "123456", 300);    // 5分钟

// 中期数据
redis.setJSON("cache:articles", data, 3600); // 1小时

// 长期数据
redis.setJSON("user:profile", data, 86400);  // 24小时
```

### 4. 批量操作优化性能

```typescript
// ❌ 不好：多次调用
redis.set("key1", "value1");
redis.set("key2", "value2");
redis.set("key3", "value3");

// ✅ 好：批量操作
redis.mSet({
  "key1": "value1",
  "key2": "value2",
  "key3": "value3"
});

// ✅ 好：Pipeline
redis.batch([
  { cmd: "SET", args: ["key1", "value1"] },
  { cmd: "SET", args: ["key2", "value2"] },
  { cmd: "INCR", args: ["counter"] }
]);
```

### 5. 避免大 Key

```typescript
// ❌ 不好：单个大 Hash
redis.hMSet("all_users", { ... }); // 可能很大

// ✅ 好：分散存储
redis.setJSON("user:1001", user1);
redis.setJSON("user:1002", user2);
```

## 类型定义

RedisHelper 提供了完整的 TypeScript 类型定义：

```typescript
interface RankingItem {
  rank: number;
  member: string;
  score: number;
}

interface RankInfo {
  rank: number;
  score: number;
}

interface ZSetMember {
  member: string;
  score: number;
}

interface PipelineCommand {
  cmd: string;
  args: any[];
}
```

## 注意事项

1. **Keys 命令慎用**: `keys()` 方法在生产环境可能阻塞 Redis，建议使用 `clearPattern()` 替代
2. **大 Key 删除**: 对于大型数据结构，使用 `unlink()` 而不是 `del()`
3. **过期时间**: 为缓存数据设置合理的过期时间，避免内存溢出
4. **Pipeline**: 批量操作时使用 `batch()` 或 `pipeline()` 提高性能
5. **Key 前缀**: 使用 key 前缀进行命名空间隔离，避免 key 冲突

## 完整 API 列表

### 基础操作
- `get()`, `set()`, `del()`, `unlink()`, `exists()`, `expire()`, `ttl()`, `keys()`
- `incr()`, `incrBy()`, `decr()`, `decrBy()`

### JSON 操作
- `setJSON()`, `getJSON()`

### 批量操作
- `mSet()`, `mGet()`

### 计数器
- `counterIncr()`, `counterDecr()`, `counterGet()`, `counterReset()`

### 排行榜
- `rankingAdd()`, `rankingIncrBy()`, `rankingTop()`, `rankingGetRank()`, `rankingRemove()`

### Hash
- `hGet()`, `hSet()`, `hMSet()`, `hGetAll()`, `hDel()`
- `hExists()`, `hKeys()`, `hVals()`, `hLen()`, `hIncrBy()`

### List
- `lPush()`, `rPush()`, `lPop()`, `rPop()`, `lRange()`, `lLen()`
- `lIndex()`, `lSet()`, `lRem()`

### Set
- `sAdd()`, `sRem()`, `sMembers()`, `sIsMember()`, `sCard()`
- `sPop()`, `sRandMember()`, `sUnion()`, `sInter()`, `sDiff()`

### Sorted Set
- `zAdd()`, `zRem()`, `zRange()`, `zRevRange()`, `zRangeByScore()`
- `zScore()`, `zCard()`, `zCount()`, `zIncrBy()`, `zRank()`, `zRevRank()`

### 高级操作
- `clearPattern()`, `batch()`, `pipeline()`

## 总结

RedisHelper 类提供了完整、类型安全的 Redis 操作接口，通过简洁的 API 设计和详细的文档，让你能够轻松地在 TypeScript 项目中使用 Redis。

如有问题或建议，欢迎反馈！
