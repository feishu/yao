package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/go-redis/redis/v8"
	"github.com/yaoapp/gou/connector"
	redisConnector "github.com/yaoapp/gou/connector/redis"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

var ctx = context.Background()

// Clients 存储所有 Redis 客户端实例
var Clients = map[string]*Client{}

// Client Redis 客户端封装
type Client struct {
	ID   string
	Name string
	Rdb  *goredis.Client
}

// Load 加载 Redis 客户端
func Load(connName string) (*Client, error) {
	conn, err := connector.Select(connName)
	if err != nil {
		return nil, fmt.Errorf("failed to select connector %s: %w", connName, err)
	}

	redisConn, ok := conn.(*redisConnector.Connector)
	if !ok {
		return nil, fmt.Errorf("connector %s is not a redis connector", connName)
	}

	if redisConn.Rdb == nil {
		return nil, fmt.Errorf("redis client is not initialized for connector %s", connName)
	}

	client := &Client{
		ID:   redisConn.ID(),
		Name: redisConn.Name,
		Rdb:  redisConn.Rdb,
	}

	Clients[client.ID] = client
	return client, nil
}

// LoadAllClients 加载所有 Redis 连接器
// 这个函数应该在 connector.Load() 之后调用
func LoadAllClients() error {
	for id, conn := range connector.Connectors {
		// 检查是否是 Redis 连接器
		if conn.Is(connector.REDIS) {
			redisConn, ok := conn.(*redisConnector.Connector)
			if !ok {
				continue
			}

			// 创建 Redis 客户端
			client := &Client{
				ID:   id,
				Name: redisConn.Name,
				Rdb:  redisConn.Rdb,
			}

			// 注册到全局 Clients map
			Clients[id] = client
		}
	}

	return nil
}

// Select 选择已加载的 Redis 客户端
func Select(id string) *Client {
	client, has := Clients[id]
	if !has {
		exception.New("Redis client %s does not exist", 500, id).Throw()
	}
	return client
}

// getRedisClient 获取 Redis 客户端（兼容旧方法）
func getRedisClient(connName string) (*goredis.Client, error) {
	conn, err := connector.Select(connName)
	if err != nil {
		return nil, fmt.Errorf("failed to select connector %s: %w", connName, err)
	}

	redisConn, ok := conn.(*redisConnector.Connector)
	if !ok {
		return nil, fmt.Errorf("connector %s is not a redis connector", connName)
	}

	if redisConn.Rdb == nil {
		return nil, fmt.Errorf("redis client is not initialized for connector %s", connName)
	}

	return redisConn.Rdb, nil
}

// ProcessGet 获取键的值
// @param connName string Redis连接器名称
// @param key string 键名
// @return string 键值
func ProcessGet(process *process.Process) interface{} {
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

	return val
}

// ProcessSet 设置键值
// @param connName string Redis连接器名称
// @param key string 键名
// @param value any 键值
// @param expiration int64 过期时间（秒），0表示永不过期
// @return string "OK"
func ProcessSet(process *process.Process) interface{} {
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

	err = rdb.Set(ctx, key, value, expiration).Err()
	if err != nil {
		exception.New("redis SET error: %s", 500, err.Error()).Throw()
	}

	return "OK"
}

// ProcessDel 删除键
// @param connName string Redis连接器名称
// @param keys ...string 一个或多个键名
// @return int 删除的键数量
func ProcessDel(process *process.Process) interface{} {
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

	count, err := rdb.Del(ctx, keys...).Result()
	if err != nil {
		exception.New("redis DEL error: %s", 500, err.Error()).Throw()
	}

	return count
}

// ProcessExists 检查键是否存在
// @param connName string Redis连接器名称
// @param keys ...string 一个或多个键名
// @return int 存在的键数量
func ProcessExists(process *process.Process) interface{} {
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

	count, err := rdb.Exists(ctx, keys...).Result()
	if err != nil {
		exception.New("redis EXISTS error: %s", 500, err.Error()).Throw()
	}

	return count
}

// ProcessExpire 设置键的过期时间
// @param connName string Redis连接器名称
// @param key string 键名
// @param seconds int 过期时间（秒）
// @return bool 是否设置成功
func ProcessExpire(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	seconds := process.ArgsInt(2)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	result, err := rdb.Expire(ctx, key, time.Duration(seconds)*time.Second).Result()
	if err != nil {
		exception.New("redis EXPIRE error: %s", 500, err.Error()).Throw()
	}

	return result
}

// ProcessTTL 获取键的剩余过期时间
// @param connName string Redis连接器名称
// @param key string 键名
// @return int 剩余秒数，-1表示永不过期，-2表示键不存在
func ProcessTTL(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	ttl, err := rdb.TTL(ctx, key).Result()
	if err != nil {
		exception.New("redis TTL error: %s", 500, err.Error()).Throw()
	}

	return int64(ttl.Seconds())
}

// ProcessIncr 键值自增1
// @param connName string Redis连接器名称
// @param key string 键名
// @return int 自增后的值
func ProcessIncr(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	val, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		exception.New("redis INCR error: %s", 500, err.Error()).Throw()
	}

	return val
}

// ProcessIncrBy 键值增加指定值
// @param connName string Redis连接器名称
// @param key string 键名
// @param value int 增加的值
// @return int 增加后的值
func ProcessIncrBy(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	value := process.ArgsInt(2)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	val, err := rdb.IncrBy(ctx, key, int64(value)).Result()
	if err != nil {
		exception.New("redis INCRBY error: %s", 500, err.Error()).Throw()
	}

	return val
}

// ProcessDecr 键值自减1
// @param connName string Redis连接器名称
// @param key string 键名
// @return int 自减后的值
func ProcessDecr(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	val, err := rdb.Decr(ctx, key).Result()
	if err != nil {
		exception.New("redis DECR error: %s", 500, err.Error()).Throw()
	}

	return val
}

// ProcessDecrBy 键值减少指定值
// @param connName string Redis连接器名称
// @param key string 键名
// @param value int 减少的值
// @return int 减少后的值
func ProcessDecrBy(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	value := process.ArgsInt(2)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	val, err := rdb.DecrBy(ctx, key, int64(value)).Result()
	if err != nil {
		exception.New("redis DECRBY error: %s", 500, err.Error()).Throw()
	}

	return val
}

// ProcessUnlink 非阻塞删除键（比 DEL 更快）
// @param connName string Redis连接器名称
// @param keys ...string 一个或多个键名
// @return int 删除的键数量
func ProcessUnlink(process *process.Process) interface{} {
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

	count, err := rdb.Unlink(ctx, keys...).Result()
	if err != nil {
		exception.New("redis UNLINK error: %s", 500, err.Error()).Throw()
	}

	return count
}

// ProcessKeys 查找匹配模式的所有键
// @param connName string Redis连接器名称
// @param pattern string 匹配模式，如 "user:*"
// @return []string 匹配的键列表
func ProcessKeys(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	pattern := process.ArgsString(1)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	keys, err := rdb.Keys(ctx, pattern).Result()
	if err != nil {
		exception.New("redis KEYS error: %s", 500, err.Error()).Throw()
	}

	return keys
}

// ProcessClearPattern 批量删除匹配模式的所有键（使用 SCAN + Pipeline Unlink）
// @param connName string Redis连接器名称
// @param pattern string 匹配模式，如 "user:*"
// @return int 删除的键总数
func ProcessClearPattern(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	pattern := process.ArgsString(1)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	var cursor uint64
	totalDeleted := int64(0)

	for {
		// 使用 SCAN 逐批扫描匹配的 key
		keys, nextCursor, err := rdb.Scan(ctx, cursor, pattern, 1000).Result()
		if err != nil {
			exception.New("redis SCAN error: %s", 500, err.Error()).Throw()
		}

		// 如果有匹配的 key，使用 Pipeline + Unlink 批量删除
		if len(keys) > 0 {
			pipe := rdb.Pipeline()
			for _, key := range keys {
				pipe.Unlink(ctx, key) // 非阻塞删除
			}

			cmds, err := pipe.Exec(ctx)
			if err != nil && err != goredis.Nil {
				exception.New("redis Pipeline UNLINK error: %s", 500, err.Error()).Throw()
			}

			// 统计删除的数量
			for _, cmd := range cmds {
				if intCmd, ok := cmd.(*goredis.IntCmd); ok {
					count, _ := intCmd.Result()
					totalDeleted += count
				}
			}
		}

		// 如果游标为 0，表示扫描完成
		if nextCursor == 0 {
			break
		}
		cursor = nextCursor
	}

	return totalDeleted
}
