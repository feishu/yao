package redis

import (
	"sync"
	"time"
	"unsafe"

	goredis "github.com/go-redis/redis/v8"
	json "github.com/goccy/go-json"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

var (
	// JSON 解析缓存：value 字符串 -> 解析后对象 ；TTL 5 min
	jsonCache sync.Map // map[string]cacheItem
)

type cacheItem struct {
	v      interface{}
	expire int64
}

func parseValue(val interface{}) interface{} {
	if val == nil {
		return nil
	}
	str, ok := val.(string)
	if !ok {
		return val // 非 string 直接返回
	}

	// 零拷贝转 []byte
	b := unsafe.Slice(unsafe.StringData(str), len(str))

	// 缓存命中？
	if hit, ok := jsonCache.Load(str); ok {
		item := hit.(cacheItem)
		if time.Now().Unix() < item.expire {
			return item.v
		}
		// 过期删除
		jsonCache.Delete(str)
	}

	// 解析
	var v interface{}
	if err := json.Unmarshal(b, &v); err == nil {
		// 写入缓存
		jsonCache.Store(str, cacheItem{v: v, expire: time.Now().Add(5 * time.Minute).Unix()})
		return v
	}
	// 非 JSON
	return str
}

// ============================================
// 高级封装方法：JSON 操作
// ============================================

// ProcessSetJSON 存储 JSON 对象
// @param connName string 连接器名称
// @param key string 键名
// @param value any JSON 对象
// @param ttl int 过期时间（秒），可选
func ProcessSetJSON(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	value := process.Args[2]

	var expiration time.Duration
	if process.NumOfArgs() > 3 {
		exp := process.ArgsInt(3)
		if exp > 0 {
			expiration = time.Duration(exp) * time.Second
		}
	}

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	// 序列化为 JSON
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		exception.New("JSON marshal error: %s", 500, err.Error()).Throw()
	}

	err = rdb.Set(ctx, key, jsonBytes, expiration).Err()
	if err != nil {
		exception.New("redis SET error: %s", 500, err.Error()).Throw()
	}

	return "OK"
}

// ProcessGetJSON 获取并解析 JSON 对象
// @param connName string 连接器名称
// @param key string 键名
// @return map[string]interface{} JSON 对象
func ProcessGetJSON(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	val, err := rdb.Get(ctx, key).Result()
	if err == goredis.Nil {
		return nil
	}
	if err != nil {
		exception.New("redis GET error: %s", 500, err.Error()).Throw()
	}

	// 解析 JSON
	var result interface{}
	err = json.Unmarshal([]byte(val), &result)
	if err != nil {
		exception.New("JSON unmarshal error: %s", 500, err.Error()).Throw()
	}

	return result
}

// ProcessMSet 批量设置多个键值对
// @param connName string 连接器名称
// @param pairs map[string]interface{} 键值对映射
func ProcessMSet(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	pairs := process.ArgsMap(1)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	// 转换为 Redis 需要的格式
	values := make([]interface{}, 0, len(pairs)*2)
	for key, val := range pairs {
		values = append(values, key, val)
	}

	err = rdb.MSet(ctx, values...).Err()
	if err != nil {
		exception.New("redis MSET error: %s", 500, err.Error()).Throw()
	}

	return "OK"
}

// ProcessMGet 批量获取多个键的值
// @param connName string 连接器名称
// @param keys ...string 多个键名
// @return map[string]string 键值映射
func ProcessMGet(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	keys := make([]string, 0)
	for i := 1; i < process.NumOfArgs(); i++ {
		keys = append(keys, process.ArgsString(i))
	}

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	// 1. MGet 获取所有键的值
	vals, err := rdb.MGet(ctx, keys...).Result()
	if err != nil {
		exception.New("redis MGET error: %s", 500, err.Error()).Throw()
	}

	// 2. 预分配结果
	result := make(map[string]interface{}, len(keys))

	// 3. 并行 JSON 解码
	var wg sync.WaitGroup
	wg.Add(len(keys))
	for i, key := range keys {
		go func(idx int, k string, val interface{}) {
			defer wg.Done()
			result[k] = parseValue(val)
		}(i, key, vals[i])
	}
	wg.Wait()
	return result
}

// ============================================
// 高级封装方法：计数器
// ============================================

// ProcessCounterIncr 计数器增加
// @param connName string 连接器名称
// @param key string 计数器键名
// @param value int 增加值，默认 1
// @return int 增加后的值
func ProcessCounterIncr(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)

	value := int64(1)
	if process.NumOfArgs() > 2 {
		value = int64(process.ArgsInt(2))
	}

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	val, err := rdb.IncrBy(ctx, key, value).Result()
	if err != nil {
		exception.New("redis INCRBY error: %s", 500, err.Error()).Throw()
	}

	return val
}

// ProcessCounterDecr 计数器减少
// @param connName string 连接器名称
// @param key string 计数器键名
// @param value int 减少值，默认 1
// @return int 减少后的值
func ProcessCounterDecr(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)

	value := int64(1)
	if process.NumOfArgs() > 2 {
		value = int64(process.ArgsInt(2))
	}

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	val, err := rdb.DecrBy(ctx, key, value).Result()
	if err != nil {
		exception.New("redis DECRBY error: %s", 500, err.Error()).Throw()
	}

	return val
}

// ProcessCounterGet 获取计数器值
// @param connName string 连接器名称
// @param key string 计数器键名
// @return int 计数器值
func ProcessCounterGet(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	val, err := rdb.Get(ctx, key).Int64()
	if err == goredis.Nil {
		return 0
	}
	if err != nil {
		exception.New("redis GET error: %s", 500, err.Error()).Throw()
	}

	return val
}

// ProcessCounterReset 重置计数器
// @param connName string 连接器名称
// @param key string 计数器键名
// @param value int 重置值，默认 0
func ProcessCounterReset(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)

	value := 0
	if process.NumOfArgs() > 2 {
		value = process.ArgsInt(2)
	}

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	err = rdb.Set(ctx, key, value, 0).Err()
	if err != nil {
		exception.New("redis SET error: %s", 500, err.Error()).Throw()
	}

	return "OK"
}

// ============================================
// 高级封装方法：排行榜
// ============================================

// ProcessRankingAdd 添加/更新排行榜成员
// @param connName string 连接器名称
// @param key string 排行榜键名
// @param member string 成员名
// @param score float64 分数
func ProcessRankingAdd(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	member := process.ArgsString(2)
	score := toFloat64(process.Args[3])

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	err = rdb.ZAdd(ctx, key, &goredis.Z{
		Score:  score,
		Member: member,
	}).Err()
	if err != nil {
		exception.New("redis ZADD error: %s", 500, err.Error()).Throw()
	}

	return "OK"
}

// ProcessRankingIncrBy 增加排行榜成员分数
// @param connName string 连接器名称
// @param key string 排行榜键名
// @param member string 成员名
// @param increment float64 增加的分数
// @return float64 增加后的分数
func ProcessRankingIncrBy(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	member := process.ArgsString(2)
	increment := toFloat64(process.Args[3])

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	score, err := rdb.ZIncrBy(ctx, key, increment, member).Result()
	if err != nil {
		exception.New("redis ZINCRBY error: %s", 500, err.Error()).Throw()
	}

	return score
}

// ProcessRankingTop 获取排行榜前N名
// @param connName string 连接器名称
// @param key string 排行榜键名
// @param count int 获取数量
// @return []map[string]interface{} 排行榜列表，包含 rank, member, score
func ProcessRankingTop(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	count := process.ArgsInt(2)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	members, err := rdb.ZRevRangeWithScores(ctx, key, 0, int64(count-1)).Result()
	if err != nil {
		exception.New("redis ZREVRANGE error: %s", 500, err.Error()).Throw()
	}

	result := make([]map[string]interface{}, 0)
	for i, m := range members {
		result = append(result, map[string]interface{}{
			"rank":   i + 1,
			"member": m.Member,
			"score":  m.Score,
		})
	}

	return result
}

// ProcessRankingGetRank 获取成员排名
// @param connName string 连接器名称
// @param key string 排行榜键名
// @param member string 成员名
// @return map[string]interface{} 包含 rank 和 score
func ProcessRankingGetRank(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	member := process.ArgsString(2)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	// 获取排名（从高到低）
	rank, err := rdb.ZRevRank(ctx, key, member).Result()
	if err != nil {
		exception.New("redis ZREVRANK error: %s", 500, err.Error()).Throw()
	}

	// 获取分数
	score, err := rdb.ZScore(ctx, key, member).Result()
	if err != nil {
		exception.New("redis ZSCORE error: %s", 500, err.Error()).Throw()
	}

	return map[string]interface{}{
		"rank":  rank + 1, // 排名从 1 开始
		"score": score,
	}
}

// ProcessRankingRemove 移除排行榜成员
// @param connName string 连接器名称
// @param key string 排行榜键名
// @param members ...string 一个或多个成员名
// @return int 移除的数量
func ProcessRankingRemove(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	members := make([]interface{}, 0)
	for i := 2; i < process.NumOfArgs(); i++ {
		members = append(members, process.ArgsString(i))
	}

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	count, err := rdb.ZRem(ctx, key, members...).Result()
	if err != nil {
		exception.New("redis ZREM error: %s", 500, err.Error()).Throw()
	}

	return count
}

// ============================================
// 高级封装方法：批量操作
// ============================================

// ProcessBatch 简化的批量操作
// @param connName string 连接器名称
// @param commands []map[string]interface{} 命令列表
// @return []interface{} 每个命令的执行结果
func ProcessBatch(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	commandsArg := process.Args[1]

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	commands, ok := commandsArg.([]interface{})
	if !ok {
		exception.New("commands must be an array", 400).Throw()
	}

	pipe := rdb.Pipeline()
	cmdResults := make([]goredis.Cmder, 0)

	for i, cmdArg := range commands {
		cmdMap, ok := cmdArg.(map[string]interface{})
		if !ok {
			exception.New("command at index %d must be an object", 400, i).Throw()
		}

		cmdName, ok := cmdMap["cmd"].(string)
		if !ok {
			exception.New("command at index %d must have a 'cmd' field", 400, i).Throw()
		}

		args, ok := cmdMap["args"].([]interface{})
		if !ok {
			exception.New("command at index %d must have an 'args' array field", 400, i).Throw()
		}

		cmd := executePipelineCommand(pipe, cmdName, args)
		if cmd != nil {
			cmdResults = append(cmdResults, cmd)
		}
	}

	_, err = pipe.Exec(ctx)
	if err != nil && err != goredis.Nil {
		exception.New("redis BATCH error: %s", 500, err.Error()).Throw()
	}

	results := make([]interface{}, 0)
	for _, cmd := range cmdResults {
		result, err := parseCommandResult(cmd)
		if err != nil {
			results = append(results, map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			results = append(results, result)
		}
	}

	return results
}
