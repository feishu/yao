/**
 * RedisHelper - Redis 操作辅助类
 * 
 * 封装所有 Redis Process 方法，提供类型安全的 TypeScript 接口
 * 支持自动添加 key 前缀，简化 Redis 操作
 * 
 * @example
 * ```typescript
 * // 创建 RedisHelper 实例
 * const redis = new RedisHelper("redis", "user:");
 * 
 * // 使用带前缀的 key
 * redis.set("1001", { name: "John", age: 30 }, 3600);
 * // 实际存储的 key 是: "user:1001"
 * 
 * const user = redis.getJSON("1001");
 * ```
 */
declare const Process: (name: string, ...args: any[]) => any;
declare class Exception {
  constructor(message: string, code?: number);
}
/**
 * Redis 排行榜项
 */
interface RankingItem {
  /** 排名（从1开始） */
  rank: number;
  /** 成员名称 */
  member: string;
  /** 分数 */
  score: number;
}

/**
 * Redis 排名信息
 */
interface RankInfo {
  /** 排名（从1开始） */
  rank: number;
  /** 分数 */
  score: number;
}

/**
 * Redis 有序集合成员
 */
interface ZSetMember {
  /** 成员名称 */
  member: string;
  /** 分数 */
  score: number;
}

/**
 * Redis Pipeline 命令
 */
interface PipelineCommand {
  /** 命令名称，如 "SET", "GET", "HGET" 等 */
  cmd: string;
  /** 命令参数数组 */
  args: any[];
}

/**
 * Redis 批量命令结果
 */
interface BatchResult {
  /** 命令执行结果，如果出错则为 error 对象 */
  result?: any;
  /** 错误信息（如果有） */
  error?: string;
}

/**
 * RedisHelper 类
 * 
 * 提供完整的 Redis 操作封装，支持：
 * - 基础 KV 操作
 * - JSON 序列化/反序列化
 * - Hash、List、Set、Sorted Set 数据结构
 * - 计数器、排行榜等高级功能
 * - Pipeline 批量操作
 * - Key 前缀自动管理
 */
export class RedisHelper {
  private connName: string;
  private keyPrefix: string;

  /**
 * 清空符合模式的键
 * @param connName redis 连接器的名称
 * @param pattern 匹配模式，支持通配符：* ? [] [^] \
 *   - * 匹配任意数量的字符
 *   - ? 匹配单个字符
 *   - [abc] 匹配 a、b 或 c
 *   - [^a] 匹配除 a 以外的字符
 * @returns 
 */
  static clearPattern(connName: string, pattern: string): Exception | number {
    if (!connName)
      return new Exception("Redis 连接器名称不能为空", 500)
    if (!pattern)
      return new Exception("pattern 不能为空", 500)

    return Process("utils.redis.ClearPattern", connName, pattern);
  }

  /**
   * 创建 RedisHelper 实例
   * 
   * @param connName Redis 连接器名称（在 connector 配置中定义）
   * @param keyPrefix 可选的 key 前缀，用于命名空间隔离（默认为空字符串）
   * 
   * @example
   * ```typescript
   * // 基本用法
   * const redis = new RedisHelper("redis");
   * 
   * // 使用 key 前缀
   * const userRedis = new RedisHelper("redis", "user:");
   * const sessionRedis = new RedisHelper("redis", "session:");
   * ```
   */
  constructor(connName: string, keyPrefix: string = "") {
    this.connName = connName;
    this.keyPrefix = keyPrefix;
  }

  /**
    * 生成完整的 Redis key（添加前缀）
    * 
    * @param key 原始 key
    * @returns 带前缀的完整 key
    */
  private getFullKey(key: string): string {
    if (!this.keyPrefix)
      return `${this.connName}:${key}`;
    return `${this.connName}:${this.keyPrefix}:${key}`;
  }

  // ============================================
  // 基础 Key-Value 操作
  // ============================================

  /**
   * 获取键的值
   * 
   * @param key 键名
   * @returns 键值，如果 key 不存在则返回 null
   * 
   * @example
   * ```typescript
   * const value = redis.get("username");
   * if (value !== null) {
   *   console.log("用户名:", value);
   * }
   * ```
   */
  get(key: string): string | null {
    return Process("utils.redis.Get", this.connName, this.getFullKey(key));
  }

  /**
   * 设置键值
   * 
   * @param key 键名
   * @param value 键值（支持字符串、数字等基本类型）
   * @param ttl 可选的过期时间（秒），0 表示永不过期
   * @returns "OK" 表示成功
   * 
   * @example
   * ```typescript
   * // 永久存储
   * redis.set("username", "john");
   * 
   * // 设置 60 秒过期
   * redis.set("captcha", "abc123", 60);
   * ```
   */
  set(key: string, value: any, ttl: number = 0): string {
    if (ttl > 0) {
      return Process("utils.redis.Set", this.connName, this.getFullKey(key), value, ttl);
    }
    return Process("utils.redis.Set", this.connName, this.getFullKey(key), value);
  }

  /**
   * 删除一个或多个键
   * 
   * @param keys 一个或多个键名
   * @returns 成功删除的键数量
   * 
   * @example
   * ```typescript
   * // 删除单个 key
   * const count = redis.del("user:1001");
   * 
   * // 删除多个 key
   * const count = redis.del("user:1001", "user:1002", "user:1003");
   * console.log(`删除了 ${count} 个键`);
   * ```
   */
  del(...keys: string[]): number {
    const fullKeys = keys.map(k => this.getFullKey(k));
    return Process("utils.redis.Del", this.connName, ...fullKeys);
  }

  /**
   * 非阻塞删除键（异步删除，适合大 key）
   * 
   * 相比 DEL 命令，UNLINK 是非阻塞的，适合删除大型数据结构（如大 hash、大 list）
   * 不会阻塞 Redis 服务器
   * 
   * @param keys 一个或多个键名
   * @returns 成功删除的键数量
   * 
   * @example
   * ```typescript
   * // 删除大型 hash 表
   * redis.unlink("large_hash_table");
   * ```
   */
  unlink(...keys: string[]): number {
    const fullKeys = keys.map(k => this.getFullKey(k));
    return Process("utils.redis.Unlink", this.connName, ...fullKeys);
  }

  /**
   * 检查键是否存在
   * 
   * @param keys 一个或多个键名
   * @returns 存在的键数量
   * 
   * @example
   * ```typescript
   * // 检查单个 key
   * if (redis.exists("user:1001") > 0) {
   *   console.log("用户存在");
   * }
   * 
   * // 检查多个 key
   * const count = redis.exists("key1", "key2", "key3");
   * console.log(`${count} 个键存在`);
   * ```
   */
  exists(...keys: string[]): number {
    const fullKeys = keys.map(k => this.getFullKey(k));
    return Process("utils.redis.Exists", this.connName, ...fullKeys);
  }

  /**
   * 设置键的过期时间
   * 
   * @param key 键名
   * @param seconds 过期时间（秒）
   * @returns true 表示成功，false 表示 key 不存在
   * 
   * @example
   * ```typescript
   * // 设置 1 小时后过期
   * redis.expire("session:abc123", 3600);
   * ```
   */
  expire(key: string, seconds: number): boolean {
    return Process("utils.redis.Expire", this.connName, this.getFullKey(key), seconds);
  }

  /**
   * 获取键的剩余过期时间
   * 
   * @param key 键名
   * @returns 剩余秒数，-1 表示永不过期，-2 表示键不存在
   * 
   * @example
   * ```typescript
   * const ttl = redis.ttl("session:abc123");
   * if (ttl > 0) {
   *   console.log(`会话还剩 ${ttl} 秒`);
   * } else if (ttl === -1) {
   *   console.log("会话永不过期");
   * } else {
   *   console.log("会话不存在");
   * }
   * ```
   */
  ttl(key: string): number {
    return Process("utils.redis.TTL", this.connName, this.getFullKey(key));
  }

  /**
   * 查找匹配模式的所有键
   * 
   * ⚠️ 注意：生产环境慎用，可能阻塞 Redis 服务器
   * 建议使用 scan 命令或 clearPattern 方法
   * 
   * @param pattern 匹配模式，支持通配符：* ? [] [^] \
   *   - * 匹配任意数量的字符
   *   - ? 匹配单个字符
   *   - [abc] 匹配 a、b 或 c
   *   - [^a] 匹配除 a 以外的字符
   * @returns 匹配的键列表
   * 
   * @example
   * ```typescript
   * // 查找所有用户相关的 key
   * const keys = redis.keys("user:*");
   * 
   * // 查找所有以 "session:" 开头的 key
   * const sessions = redis.keys("session:*");
   * ```
   */
  keys(pattern: string): string[] {
    return Process("utils.redis.Keys", this.connName, this.getFullKey(pattern));
  }

  /**
   * 键值自增 1
   * 
   * 如果 key 不存在，则初始化为 0 后再自增
   * 
   * @param key 键名
   * @returns 自增后的值
   * 
   * @example
   * ```typescript
   * const views = redis.incr("article:100:views");
   * console.log(`文章浏览量: ${views}`);
   * ```
   */
  incr(key: string): number {
    return Process("utils.redis.Incr", this.connName, this.getFullKey(key));
  }

  /**
   * 键值增加指定值
   * 
   * @param key 键名
   * @param value 增加的值（整数）
   * @returns 增加后的值
   * 
   * @example
   * ```typescript
   * // 增加 10 个点赞
   * const likes = redis.incrBy("article:100:likes", 10);
   * ```
   */
  incrBy(key: string, value: number): number {
    return Process("utils.redis.IncrBy", this.connName, this.getFullKey(key), value);
  }

  /**
   * 键值自减 1
   * 
   * @param key 键名
   * @returns 自减后的值
   * 
   * @example
   * ```typescript
   * const stock = redis.decr("product:200:stock");
   * console.log(`剩余库存: ${stock}`);
   * ```
   */
  decr(key: string): number {
    return Process("utils.redis.Decr", this.connName, this.getFullKey(key));
  }

  /**
   * 键值减少指定值
   * 
   * @param key 键名
   * @param value 减少的值（整数）
   * @returns 减少后的值
   * 
   * @example
   * ```typescript
   * // 减少 5 个库存
   * const stock = redis.decrBy("product:200:stock", 5);
   * ```
   */
  decrBy(key: string, value: number): number {
    return Process("utils.redis.DecrBy", this.connName, this.getFullKey(key), value);
  }

  // ============================================
  // JSON 操作 - 便捷存取 JSON 对象
  // ============================================

  /**
   * 存储 JSON 对象
   * 
   * 自动序列化对象为 JSON 字符串存储
   * 
   * @param key 键名
   * @param value JSON 对象（任意可序列化的 JavaScript 对象）
   * @param ttl 可选的过期时间（秒），0 表示永不过期
   * @returns "OK" 表示成功
   * 
   * @example
   * ```typescript
   * // 存储用户对象
   * redis.setJSON("user:1001", {
   *   name: "John Doe",
   *   age: 30,
   *   email: "john@example.com",
   *   tags: ["vip", "active"]
   * }, 3600); // 1小时后过期
   * ```
   */
  setJSON(key: string, value: any, ttl: number = 0): string {
    if (ttl > 0) {
      return Process("utils.redis.SetJSON", this.connName, this.getFullKey(key), value, ttl);
    }
    return Process("utils.redis.SetJSON", this.connName, this.getFullKey(key), value);
  }

  /**
   * 获取并解析 JSON 对象
   * 
   * 自动反序列化 JSON 字符串为对象
   * 
   * @param key 键名
   * @returns JSON 对象，如果 key 不存在或解析失败则返回 null
   * 
   * @example
   * ```typescript
   * interface User {
   *   name: string;
   *   age: number;
   *   email: string;
   * }
   * 
   * const user = redis.getJSON<User>("user:1001");
   * if (user) {
   *   console.log(`用户: ${user.name}, 年龄: ${user.age}`);
   * }
   * ```
   */
  getJSON<T = any>(key: string): T | null {
    return Process("utils.redis.GetJSON", this.connName, this.getFullKey(key));
  }

  // ============================================
  // 批量操作
  // ============================================

  /**
   * 批量设置多个键值对
   * 
   * 一次性设置多个 key-value，原子操作
   * 
   * @param pairs 键值对映射 { key1: value1, key2: value2, ... }
   * @returns "OK" 表示成功
   * 
   * @example
   * ```typescript
   * redis.mSet({
   *   "config:timeout": "30",
   *   "config:max_conn": "100",
   *   "config:debug": "true"
   * });
   * ```
   */
  mSet(pairs: Record<string, any>): string {
    // 添加前缀到所有 key
    const fullPairs: Record<string, any> = {};
    for (const [key, value] of Object.entries(pairs)) {
      fullPairs[this.getFullKey(key)] = value;
    }
    return Process("utils.redis.MSet", this.connName, fullPairs);
  }

  /**
   * 批量获取多个键的值
   * 
   * @param keys 多个键名
   * @returns 键值映射对象 { key1: value1, key2: value2, ... }
   *          如果某个 key 不存在，对应的 value 为 null
   * 
   * @example
   * ```typescript
   * const values = redis.mGet("key1", "key2", "key3");
   * console.log(values);
   * // { "key1": "value1", "key2": null, "key3": "value3" }
   * ```
   */
  mGet(...keys: string[]): Record<string, any> {
    const fullKeys = keys.map(k => this.getFullKey(k));
    const result = Process("utils.redis.MGet", this.connName, ...fullKeys);

    // 移除前缀，返回原始 key
    if (this.keyPrefix && result) {
      const newResult: Record<string, any> = {};
      for (const [fullKey, value] of Object.entries(result)) {
        const originalKey = fullKey.startsWith(this.keyPrefix)
          ? fullKey.substring(this.keyPrefix.length)
          : fullKey;
        newResult[originalKey] = value;
      }
      return newResult;
    }

    return result;
  }

  // ============================================
  // 计数器操作
  // ============================================

  /**
   * 计数器增加
   * 
   * 如果 key 不存在，初始化为 0 后再增加
   * 
   * @param key 计数器键名
   * @param value 增加值，默认 1
   * @returns 增加后的值
   * 
   * @example
   * ```typescript
   * // 增加 1 次浏览量
   * const views = redis.counterIncr("article:100:views");
   * 
   * // 增加 10 次点赞
   * const likes = redis.counterIncr("article:100:likes", 10);
   * ```
   */
  counterIncr(key: string, value: number = 1): number {
    if (value === 1) {
      return Process("utils.redis.CounterIncr", this.connName, this.getFullKey(key));
    }
    return Process("utils.redis.CounterIncr", this.connName, this.getFullKey(key), value);
  }

  /**
   * 计数器减少
   * 
   * @param key 计数器键名
   * @param value 减少值，默认 1
   * @returns 减少后的值
   * 
   * @example
   * ```typescript
   * // 减少 1 个库存
   * const stock = redis.counterDecr("product:200:stock");
   * 
   * // 减少 5 个库存
   * const stock = redis.counterDecr("product:200:stock", 5);
   * ```
   */
  counterDecr(key: string, value: number = 1): number {
    if (value === 1) {
      return Process("utils.redis.CounterDecr", this.connName, this.getFullKey(key));
    }
    return Process("utils.redis.CounterDecr", this.connName, this.getFullKey(key), value);
  }

  /**
   * 获取计数器值
   * 
   * @param key 计数器键名
   * @returns 计数器值，如果 key 不存在则返回 0
   * 
   * @example
   * ```typescript
   * const views = redis.counterGet("article:100:views");
   * console.log(`文章浏览量: ${views}`);
   * ```
   */
  counterGet(key: string): number {
    return Process("utils.redis.CounterGet", this.connName, this.getFullKey(key));
  }

  /**
   * 重置计数器
   * 
   * @param key 计数器键名
   * @param value 重置值，默认 0
   * @returns "OK" 表示成功
   * 
   * @example
   * ```typescript
   * // 重置为 0
   * redis.counterReset("article:100:views");
   * 
   * // 重置为指定值
   * redis.counterReset("product:200:stock", 100);
   * ```
   */
  counterReset(key: string, value: number = 0): string {
    if (value === 0) {
      return Process("utils.redis.CounterReset", this.connName, this.getFullKey(key));
    }
    return Process("utils.redis.CounterReset", this.connName, this.getFullKey(key), value);
  }

  // ============================================
  // 排行榜操作
  // ============================================

  /**
   * 添加或更新排行榜成员
   * 
   * @param key 排行榜键名
   * @param member 成员名称
   * @param score 分数
   * @returns "OK" 表示成功
   * 
   * @example
   * ```typescript
   * // 添加游戏分数
   * redis.rankingAdd("game:level1:scores", "player123", 8500);
   * ```
   */
  rankingAdd(key: string, member: string, score: number): string {
    return Process("utils.redis.RankingAdd", this.connName, this.getFullKey(key), member, score);
  }

  /**
   * 增加排行榜成员分数
   * 
   * @param key 排行榜键名
   * @param member 成员名称
   * @param increment 增加的分数（可以是负数）
   * @returns 增加后的分数
   * 
   * @example
   * ```typescript
   * // 增加 100 分
   * const newScore = redis.rankingIncrBy("game:level1:scores", "player123", 100);
   * console.log(`新分数: ${newScore}`);
   * ```
   */
  rankingIncrBy(key: string, member: string, increment: number): number {
    return Process("utils.redis.RankingIncrBy", this.connName, this.getFullKey(key), member, increment);
  }

  /**
   * 获取排行榜前 N 名
   * 
   * @param key 排行榜键名
   * @param count 获取数量
   * @returns 排行榜列表，按分数从高到低排序
   * 
   * @example
   * ```typescript
   * // 获取前 10 名
   * const top10 = redis.rankingTop("game:level1:scores", 10);
   * top10.forEach(item => {
   *   console.log(`第${item.rank}名: ${item.member}, 分数: ${item.score}`);
   * });
   * ```
   */
  rankingTop(key: string, count: number): RankingItem[] {
    return Process("utils.redis.RankingTop", this.connName, this.getFullKey(key), count);
  }

  /**
   * 获取成员排名和分数
   * 
   * @param key 排行榜键名
   * @param member 成员名称
   * @returns 包含排名和分数的对象，排名从 1 开始
   * 
   * @example
   * ```typescript
   * const rankInfo = redis.rankingGetRank("game:level1:scores", "player123");
   * console.log(`排名: ${rankInfo.rank}, 分数: ${rankInfo.score}`);
   * ```
   */
  rankingGetRank(key: string, member: string): RankInfo {
    return Process("utils.redis.RankingGetRank", this.connName, this.getFullKey(key), member);
  }

  /**
   * 移除排行榜成员
   * 
   * @param key 排行榜键名
   * @param members 一个或多个成员名称
   * @returns 移除的成员数量
   * 
   * @example
   * ```typescript
   * // 移除单个成员
   * redis.rankingRemove("game:level1:scores", "player123");
   * 
   * // 移除多个成员
   * redis.rankingRemove("game:level1:scores", "player1", "player2", "player3");
   * ```
   */
  rankingRemove(key: string, ...members: string[]): number {
    return Process("utils.redis.RankingRemove", this.connName, this.getFullKey(key), ...members);
  }

  // ============================================
  // Hash 操作
  // ============================================

  /**
   * 获取哈希表字段的值
   * 
   * @param key 键名
   * @param field 字段名
   * @returns 字段值，如果字段不存在则返回 null
   * 
   * @example
   * ```typescript
   * const name = redis.hGet("user:1001", "name");
   * console.log("用户名:", name);
   * ```
   */
  hGet(key: string, field: string): string | null {
    return Process("utils.redis.HGet", this.connName, this.getFullKey(key), field);
  }

  /**
   * 设置哈希表字段的值
   * 
   * @param key 键名
   * @param field 字段名
   * @param value 字段值
   * @returns 新增的字段数量（0 或 1）
   * 
   * @example
   * ```typescript
   * redis.hSet("user:1001", "name", "John Doe");
   * redis.hSet("user:1001", "age", 30);
   * ```
   */
  hSet(key: string, field: string, value: any): number {
    return Process("utils.redis.HSet", this.connName, this.getFullKey(key), field, value);
  }

  /**
   * 批量设置哈希表的字段
   * 
   * @param key 键名
   * @param values 字段-值映射 { field1: value1, field2: value2, ... }
   * @returns "OK" 表示成功
   * 
   * @example
   * ```typescript
   * redis.hMSet("user:1001", {
   *   name: "John Doe",
   *   age: 30,
   *   email: "john@example.com"
   * });
   * ```
   */
  hMSet(key: string, values: Record<string, any>): string {
    return Process("utils.redis.HMSet", this.connName, this.getFullKey(key), values);
  }

  /**
   * 获取哈希表所有字段和值
   * 
   * @param key 键名
   * @returns 所有字段和值的映射对象
   * 
   * @example
   * ```typescript
   * const user = redis.hGetAll("user:1001");
   * console.log(user);
   * // { name: "John Doe", age: "30", email: "john@example.com" }
   * ```
   */
  hGetAll(key: string): Record<string, string> {
    return Process("utils.redis.HGetAll", this.connName, this.getFullKey(key));
  }

  /**
   * 删除哈希表的一个或多个字段
   * 
   * @param key 键名
   * @param fields 一个或多个字段名
   * @returns 删除的字段数量
   * 
   * @example
   * ```typescript
   * // 删除单个字段
   * redis.hDel("user:1001", "age");
   * 
   * // 删除多个字段
   * redis.hDel("user:1001", "age", "email");
   * ```
   */
  hDel(key: string, ...fields: string[]): number {
    return Process("utils.redis.HDel", this.connName, this.getFullKey(key), ...fields);
  }

  /**
   * 检查哈希表字段是否存在
   * 
   * @param key 键名
   * @param field 字段名
   * @returns true 表示字段存在，false 表示不存在
   * 
   * @example
   * ```typescript
   * if (redis.hExists("user:1001", "email")) {
   *   console.log("用户有邮箱");
   * }
   * ```
   */
  hExists(key: string, field: string): boolean {
    return Process("utils.redis.HExists", this.connName, this.getFullKey(key), field);
  }

  /**
   * 获取哈希表所有字段名
   * 
   * @param key 键名
   * @returns 所有字段名的数组
   * 
   * @example
   * ```typescript
   * const fields = redis.hKeys("user:1001");
   * console.log("用户字段:", fields);
   * // ["name", "age", "email"]
   * ```
   */
  hKeys(key: string): string[] {
    return Process("utils.redis.HKeys", this.connName, this.getFullKey(key));
  }

  /**
   * 获取哈希表所有值
   * 
   * @param key 键名
   * @returns 所有值的数组
   * 
   * @example
   * ```typescript
   * const values = redis.hVals("user:1001");
   * console.log("用户值:", values);
   * // ["John Doe", "30", "john@example.com"]
   * ```
   */
  hVals(key: string): string[] {
    return Process("utils.redis.HVals", this.connName, this.getFullKey(key));
  }

  /**
   * 获取哈希表字段数量
   * 
   * @param key 键名
   * @returns 字段数量
   * 
   * @example
   * ```typescript
   * const count = redis.hLen("user:1001");
   * console.log(`用户有 ${count} 个字段`);
   * ```
   */
  hLen(key: string): number {
    return Process("utils.redis.HLen", this.connName, this.getFullKey(key));
  }

  /**
   * 哈希表字段值增加指定值
   * 
   * @param key 键名
   * @param field 字段名
   * @param value 增加的值（整数）
   * @returns 增加后的值
   * 
   * @example
   * ```typescript
   * // 增加用户积分
   * const points = redis.hIncrBy("user:1001", "points", 100);
   * console.log(`当前积分: ${points}`);
   * ```
   */
  hIncrBy(key: string, field: string, value: number): number {
    return Process("utils.redis.HIncrBy", this.connName, this.getFullKey(key), field, value);
  }

  // ============================================
  // List 操作
  // ============================================

  /**
   * 将一个或多个值插入到列表头部
   * 
   * @param key 键名
   * @param values 一个或多个值
   * @returns 列表长度
   * 
   * @example
   * ```typescript
   * // 添加单个任务
   * redis.lPush("tasks", "task1");
   * 
   * // 添加多个任务
   * redis.lPush("tasks", "task2", "task3", "task4");
   * ```
   */
  lPush(key: string, ...values: any[]): number {
    return Process("utils.redis.LPush", this.connName, this.getFullKey(key), ...values);
  }

  /**
   * 将一个或多个值插入到列表尾部
   * 
   * @param key 键名
   * @param values 一个或多个值
   * @returns 列表长度
   * 
   * @example
   * ```typescript
   * redis.rPush("queue", "item1", "item2", "item3");
   * ```
   */
  rPush(key: string, ...values: any[]): number {
    return Process("utils.redis.RPush", this.connName, this.getFullKey(key), ...values);
  }

  /**
   * 移除并返回列表头部元素
   * 
   * @param key 键名
   * @returns 头部元素，如果列表为空则返回 null
   * 
   * @example
   * ```typescript
   * const task = redis.lPop("tasks");
   * if (task) {
   *   console.log("处理任务:", task);
   * }
   * ```
   */
  lPop(key: string): string | null {
    return Process("utils.redis.LPop", this.connName, this.getFullKey(key));
  }

  /**
   * 移除并返回列表尾部元素
   * 
   * @param key 键名
   * @returns 尾部元素，如果列表为空则返回 null
   * 
   * @example
   * ```typescript
   * const item = redis.rPop("queue");
   * ```
   */
  rPop(key: string): string | null {
    return Process("utils.redis.RPop", this.connName, this.getFullKey(key));
  }

  /**
   * 获取列表指定范围内的元素
   * 
   * @param key 键名
   * @param start 起始索引（从 0 开始）
   * @param stop 结束索引（-1 表示最后一个元素）
   * @returns 元素列表
   * 
   * @example
   * ```typescript
   * // 获取前 10 个元素
   * const items = redis.lRange("queue", 0, 9);
   * 
   * // 获取所有元素
   * const all = redis.lRange("queue", 0, -1);
   * ```
   */
  lRange(key: string, start: number, stop: number): string[] {
    return Process("utils.redis.LRange", this.connName, this.getFullKey(key), start, stop);
  }

  /**
   * 获取列表长度
   * 
   * @param key 键名
   * @returns 列表长度
   * 
   * @example
   * ```typescript
   * const length = redis.lLen("queue");
   * console.log(`队列中有 ${length} 个任务`);
   * ```
   */
  lLen(key: string): number {
    return Process("utils.redis.LLen", this.connName, this.getFullKey(key));
  }

  /**
   * 通过索引获取列表中的元素
   * 
   * @param key 键名
   * @param index 索引（从 0 开始，负数表示从尾部开始）
   * @returns 元素值
   * 
   * @example
   * ```typescript
   * // 获取第一个元素
   * const first = redis.lIndex("queue", 0);
   * 
   * // 获取最后一个元素
   * const last = redis.lIndex("queue", -1);
   * ```
   */
  lIndex(key: string, index: number): string | null {
    return Process("utils.redis.LIndex", this.connName, this.getFullKey(key), index);
  }

  /**
   * 通过索引设置列表中的元素值
   * 
   * @param key 键名
   * @param index 索引
   * @param value 新值
   * @returns "OK" 表示成功
   * 
   * @example
   * ```typescript
   * redis.lSet("queue", 0, "new_value");
   * ```
   */
  lSet(key: string, index: number, value: any): string {
    return Process("utils.redis.LSet", this.connName, this.getFullKey(key), index, value);
  }

  /**
   * 移除列表中指定数量的元素
   * 
   * @param key 键名
   * @param count 移除数量
   *   - count > 0: 从头到尾移除 count 个值为 value 的元素
   *   - count < 0: 从尾到头移除 |count| 个值为 value 的元素
   *   - count = 0: 移除所有值为 value 的元素
   * @param value 要移除的值
   * @returns 移除的数量
   * 
   * @example
   * ```typescript
   * // 移除所有值为 "task1" 的元素
   * redis.lRem("tasks", 0, "task1");
   * 
   * // 从头开始移除 2 个值为 "task2" 的元素
   * redis.lRem("tasks", 2, "task2");
   * ```
   */
  lRem(key: string, count: number, value: any): number {
    return Process("utils.redis.LRem", this.connName, this.getFullKey(key), count, value);
  }

  // ============================================
  // Set 操作
  // ============================================

  /**
   * 向集合添加一个或多个成员
   * 
   * @param key 键名
   * @param members 一个或多个成员
   * @returns 新增的成员数量
   * 
   * @example
   * ```typescript
   * redis.sAdd("tags", "javascript", "typescript", "nodejs");
   * ```
   */
  sAdd(key: string, ...members: any[]): number {
    return Process("utils.redis.SAdd", this.connName, this.getFullKey(key), ...members);
  }

  /**
   * 移除集合中的一个或多个成员
   * 
   * @param key 键名
   * @param members 一个或多个成员
   * @returns 移除的成员数量
   * 
   * @example
   * ```typescript
   * redis.sRem("tags", "outdated_tag");
   * ```
   */
  sRem(key: string, ...members: any[]): number {
    return Process("utils.redis.SRem", this.connName, this.getFullKey(key), ...members);
  }

  /**
   * 获取集合的所有成员
   * 
   * @param key 键名
   * @returns 所有成员的数组
   * 
   * @example
   * ```typescript
   * const tags = redis.sMembers("article:100:tags");
   * console.log("文章标签:", tags);
   * ```
   */
  sMembers(key: string): string[] {
    return Process("utils.redis.SMembers", this.connName, this.getFullKey(key));
  }

  /**
   * 判断成员是否在集合中
   * 
   * @param key 键名
   * @param member 成员
   * @returns true 表示存在，false 表示不存在
   * 
   * @example
   * ```typescript
   * if (redis.sIsMember("tags", "javascript")) {
   *   console.log("标签存在");
   * }
   * ```
   */
  sIsMember(key: string, member: any): boolean {
    return Process("utils.redis.SIsMember", this.connName, this.getFullKey(key), member);
  }

  /**
   * 获取集合的成员数量
   * 
   * @param key 键名
   * @returns 成员数量
   * 
   * @example
   * ```typescript
   * const count = redis.sCard("tags");
   * console.log(`共有 ${count} 个标签`);
   * ```
   */
  sCard(key: string): number {
    return Process("utils.redis.SCard", this.connName, this.getFullKey(key));
  }

  /**
   * 移除并返回集合中的一个随机元素
   * 
   * @param key 键名
   * @returns 随机元素
   * 
   * @example
   * ```typescript
   * const randomTag = redis.sPop("tags");
   * ```
   */
  sPop(key: string): string | null {
    return Process("utils.redis.SPop", this.connName, this.getFullKey(key));
  }

  /**
   * 返回集合中的一个或多个随机元素（不移除）
   * 
   * @param key 键名
   * @param count 可选的返回数量，默认 1
   * @returns 如果 count 为 1，返回单个元素；否则返回数组
   * 
   * @example
   * ```typescript
   * // 获取 1 个随机元素
   * const tag = redis.sRandMember("tags");
   * 
   * // 获取 3 个随机元素
   * const tags = redis.sRandMember("tags", 3);
   * ```
   */
  sRandMember(key: string, count?: number): string | string[] {
    if (count !== undefined) {
      return Process("utils.redis.SRandMember", this.connName, this.getFullKey(key), count);
    }
    return Process("utils.redis.SRandMember", this.connName, this.getFullKey(key));
  }

  /**
   * 返回多个集合的并集
   * 
   * @param keys 一个或多个键名
   * @returns 并集成员数组
   * 
   * @example
   * ```typescript
   * const allTags = redis.sUnion("tags:article1", "tags:article2");
   * ```
   */
  sUnion(...keys: string[]): string[] {
    const fullKeys = keys.map(k => this.getFullKey(k));
    return Process("utils.redis.SUnion", this.connName, ...fullKeys);
  }

  /**
   * 返回多个集合的交集
   * 
   * @param keys 一个或多个键名
   * @returns 交集成员数组
   * 
   * @example
   * ```typescript
   * const commonTags = redis.sInter("tags:article1", "tags:article2");
   * ```
   */
  sInter(...keys: string[]): string[] {
    const fullKeys = keys.map(k => this.getFullKey(k));
    return Process("utils.redis.SInter", this.connName, ...fullKeys);
  }

  /**
   * 返回多个集合的差集
   * 
   * @param keys 一个或多个键名
   * @returns 差集成员数组（第一个集合相对于其他集合的差集）
   * 
   * @example
   * ```typescript
   * const uniqueTags = redis.sDiff("tags:article1", "tags:article2");
   * ```
   */
  sDiff(...keys: string[]): string[] {
    const fullKeys = keys.map(k => this.getFullKey(k));
    return Process("utils.redis.SDiff", this.connName, ...fullKeys);
  }

  // ============================================
  // Sorted Set (有序集合) 操作
  // ============================================

  /**
   * 向有序集合添加一个或多个成员
   * 
   * @param key 键名
   * @param members 成员-分数映射 { member1: score1, member2: score2, ... }
   * @returns 新增的成员数量
   * 
   * @example
   * ```typescript
   * redis.zAdd("leaderboard", {
   *   "player1": 1000,
   *   "player2": 850,
   *   "player3": 920
   * });
   * ```
   */
  zAdd(key: string, members: Record<string, number>): number {
    return Process("utils.redis.ZAdd", this.connName, this.getFullKey(key), members);
  }

  /**
   * 移除有序集合中的一个或多个成员
   * 
   * @param key 键名
   * @param members 一个或多个成员
   * @returns 移除的成员数量
   * 
   * @example
   * ```typescript
   * redis.zRem("leaderboard", "player1", "player2");
   * ```
   */
  zRem(key: string, ...members: any[]): number {
    return Process("utils.redis.ZRem", this.connName, this.getFullKey(key), ...members);
  }

  /**
   * 返回有序集合指定索引范围内的成员
   * 
   * @param key 键名
   * @param start 起始索引（从 0 开始）
   * @param stop 结束索引（-1 表示最后一个）
   * @param withScores 是否返回分数，默认 false
   * @returns 如果 withScores 为 true，返回包含 member 和 score 的对象数组；否则返回成员名数组
   * 
   * @example
   * ```typescript
   * // 只返回成员名
   * const members = redis.zRange("leaderboard", 0, 9);
   * 
   * // 返回成员名和分数
   * const membersWithScores = redis.zRange("leaderboard", 0, 9, true);
   * ```
   */
  zRange(key: string, start: number, stop: number, withScores: boolean = false): string[] | ZSetMember[] {
    return Process("utils.redis.ZRange", this.connName, this.getFullKey(key), start, stop, withScores);
  }

  /**
   * 返回有序集合指定索引范围内的成员（按分数从高到低）
   * 
   * @param key 键名
   * @param start 起始索引
   * @param stop 结束索引
   * @param withScores 是否返回分数，默认 false
   * @returns 成员列表（按分数降序）
   * 
   * @example
   * ```typescript
   * // 获取前 10 名（分数最高的）
   * const top10 = redis.zRevRange("leaderboard", 0, 9, true);
   * ```
   */
  zRevRange(key: string, start: number, stop: number, withScores: boolean = false): string[] | ZSetMember[] {
    return Process("utils.redis.ZRevRange", this.connName, this.getFullKey(key), start, stop, withScores);
  }

  /**
   * 返回有序集合指定分数范围内的成员
   * 
   * @param key 键名
   * @param min 最小分数（支持 "-inf" 表示负无穷，"(score" 表示不包含该分数）
   * @param max 最大分数（支持 "+inf" 表示正无穷）
   * @param withScores 是否返回分数，默认 false
   * @returns 成员列表
   * 
   * @example
   * ```typescript
   * // 获取分数在 80-100 之间的成员
   * const members = redis.zRangeByScore("scores", "80", "100");
   * 
   * // 获取分数大于 80（不包含 80）的成员
   * const members = redis.zRangeByScore("scores", "(80", "+inf");
   * ```
   */
  zRangeByScore(key: string, min: string, max: string, withScores: boolean = false): string[] | ZSetMember[] {
    return Process("utils.redis.ZRangeByScore", this.connName, this.getFullKey(key), min, max, withScores);
  }

  /**
   * 获取有序集合成员的分数
   * 
   * @param key 键名
   * @param member 成员名称
   * @returns 分数
   * 
   * @example
   * ```typescript
   * const score = redis.zScore("leaderboard", "player1");
   * console.log(`玩家分数: ${score}`);
   * ```
   */
  zScore(key: string, member: string): number {
    return Process("utils.redis.ZScore", this.connName, this.getFullKey(key), member);
  }

  /**
   * 获取有序集合的成员数量
   * 
   * @param key 键名
   * @returns 成员数量
   * 
   * @example
   * ```typescript
   * const count = redis.zCard("leaderboard");
   * console.log(`排行榜有 ${count} 名玩家`);
   * ```
   */
  zCard(key: string): number {
    return Process("utils.redis.ZCard", this.connName, this.getFullKey(key));
  }

  /**
   * 获取有序集合指定分数范围内的成员数量
   * 
   * @param key 键名
   * @param min 最小分数
   * @param max 最大分数
   * @returns 成员数量
   * 
   * @example
   * ```typescript
   * const count = redis.zCount("scores", "60", "100");
   * console.log(`及格的有 ${count} 人`);
   * ```
   */
  zCount(key: string, min: string, max: string): number {
    return Process("utils.redis.ZCount", this.connName, this.getFullKey(key), min, max);
  }

  /**
   * 有序集合成员分数增加指定值
   * 
   * @param key 键名
   * @param increment 增加的值（可以是负数）
   * @param member 成员名称
   * @returns 增加后的分数
   * 
   * @example
   * ```typescript
   * // 增加 100 分
   * const newScore = redis.zIncrBy("leaderboard", 100, "player1");
   * ```
   */
  zIncrBy(key: string, increment: number, member: string): number {
    return Process("utils.redis.ZIncrBy", this.connName, this.getFullKey(key), increment, member);
  }

  /**
   * 获取有序集合成员的排名（从低到高）
   * 
   * @param key 键名
   * @param member 成员名称
   * @returns 排名（从 0 开始）
   * 
   * @example
   * ```typescript
   * const rank = redis.zRank("scores", "student1");
   * console.log(`排名第 ${rank + 1} 名`);
   * ```
   */
  zRank(key: string, member: string): number {
    return Process("utils.redis.ZRank", this.connName, this.getFullKey(key), member);
  }

  /**
   * 获取有序集合成员的排名（从高到低）
   * 
   * @param key 键名
   * @param member 成员名称
   * @returns 排名（从 0 开始）
   * 
   * @example
   * ```typescript
   * const rank = redis.zRevRank("leaderboard", "player1");
   * console.log(`排名第 ${rank + 1} 名`);
   * ```
   */
  zRevRank(key: string, member: string): number {
    return Process("utils.redis.ZRevRank", this.connName, this.getFullKey(key), member);
  }

  // ============================================
  // 高级操作
  // ============================================

  /**
   * 批量删除匹配模式的所有键
   * 
   * 使用 SCAN + Pipeline Unlink 实现，不会阻塞 Redis 服务器
   * 
   * @param pattern 匹配模式（支持通配符 *）
   * @returns 删除的键总数
   * 
   * @example
   * ```typescript
   * // 清理所有临时缓存
   * const deleted = redis.clearPattern("temp:*");
   * console.log(`删除了 ${deleted} 个临时键`);
   * 
   * // 清理过期会话
   * redis.clearPattern("session:expired:*");
   * ```
   */
  clearPattern(pattern: string): number {
    return Process("utils.redis.ClearPattern", this.connName, this.getFullKey(pattern));
  }

  /**
   * 执行批量命令（简化的 Pipeline）
   * 
   * 一次性执行多个 Redis 命令，提高性能
   * 
   * @param commands 命令列表，每个命令包含 cmd 和 args 字段
   * @returns 每个命令的执行结果数组
   * 
   * @example
   * ```typescript
   * const results = redis.batch([
   *   { cmd: "SET", args: ["key1", "value1"] },
   *   { cmd: "SET", args: ["key2", "value2"] },
   *   { cmd: "GET", args: ["key1"] },
   *   { cmd: "INCR", args: ["counter"] }
   * ]);
   * 
   * console.log(results);
   * // ["OK", "OK", "value1", 1]
   * ```
   */
  batch(commands: PipelineCommand[]): any[] {
    // 处理命令中的 key，添加前缀
    const processedCommands = commands.map(cmd => {
      const args = [...cmd.args];

      // 大多数命令的第一个参数是 key，需要添加前缀
      if (args.length > 0 && typeof args[0] === 'string') {
        args[0] = this.getFullKey(args[0]);
      }

      return { cmd: cmd.cmd, args };
    });

    return Process("utils.redis.Batch", this.connName, processedCommands);
  }

  /**
   * 执行 Pipeline 批量命令
   * 
   * 与 batch 方法功能相同，提供更明确的命名
   * 
   * @param commands 命令列表
   * @returns 每个命令的执行结果数组
   * 
   * @example
   * ```typescript
   * const results = redis.pipeline([
   *   { cmd: "HSET", args: ["user:1", "name", "John"] },
   *   { cmd: "HSET", args: ["user:1", "age", 30] },
   *   { cmd: "HGETALL", args: ["user:1"] }
   * ]);
   * ```
   */
  pipeline(commands: PipelineCommand[]): any[] {
    return this.batch(commands);
  }
}

export default RedisHelper;
