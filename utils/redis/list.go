package redis

import (
	goredis "github.com/go-redis/redis/v8"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

// ProcessLPush 将一个或多个值插入到列表头部
// @param connName string 连接器名称
// @param key string 键名
// @param values ...any 一个或多个值
// @return int 列表长度
func ProcessLPush(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	values := make([]interface{}, 0)
	for i := 2; i < process.NumOfArgs(); i++ {
		values = append(values, process.Args[i])
	}

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	count, err := rdb.LPush(ctx, key, values...).Result()
	if err != nil {
		exception.New("redis LPUSH error: %s", 500, err.Error()).Throw()
	}

	return count
}

// ProcessRPush 将一个或多个值插入到列表尾部
// @param connName string 连接器名称
// @param key string 键名
// @param values ...any 一个或多个值
// @return int 列表长度
func ProcessRPush(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	values := make([]interface{}, 0)
	for i := 2; i < process.NumOfArgs(); i++ {
		values = append(values, process.Args[i])
	}

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	count, err := rdb.RPush(ctx, key, values...).Result()
	if err != nil {
		exception.New("redis RPUSH error: %s", 500, err.Error()).Throw()
	}

	return count
}

// ProcessLPop 移除并返回列表头部元素
// @param connName string 连接器名称
// @param key string 键名
// @return string 头部元素
func ProcessLPop(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	val, err := rdb.LPop(ctx, key).Result()
	if err == goredis.Nil {
		return nil
	}
	if err != nil {
		exception.New("redis LPOP error: %s", 500, err.Error()).Throw()
	}

	return val
}

// ProcessRPop 移除并返回列表尾部元素
// @param connName string 连接器名称
// @param key string 键名
// @return string 尾部元素
func ProcessRPop(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	val, err := rdb.RPop(ctx, key).Result()
	if err == goredis.Nil {
		return nil
	}
	if err != nil {
		exception.New("redis RPOP error: %s", 500, err.Error()).Throw()
	}

	return val
}

// ProcessLRange 获取列表指定范围内的元素
// @param connName string 连接器名称
// @param key string 键名
// @param start int 起始索引
// @param stop int 结束索引
// @return []string 元素列表
func ProcessLRange(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	start := process.ArgsInt(2)
	stop := process.ArgsInt(3)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	vals, err := rdb.LRange(ctx, key, int64(start), int64(stop)).Result()
	if err != nil {
		exception.New("redis LRANGE error: %s", 500, err.Error()).Throw()
	}

	return vals
}

// ProcessLLen 获取列表长度
// @param connName string 连接器名称
// @param key string 键名
// @return int 列表长度
func ProcessLLen(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	count, err := rdb.LLen(ctx, key).Result()
	if err != nil {
		exception.New("redis LLEN error: %s", 500, err.Error()).Throw()
	}

	return count
}

// ProcessLIndex 通过索引获取列表中的元素
// @param connName string 连接器名称
// @param key string 键名
// @param index int 索引
// @return string 元素值
func ProcessLIndex(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	index := process.ArgsInt(2)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	val, err := rdb.LIndex(ctx, key, int64(index)).Result()
	if err == goredis.Nil {
		return nil
	}
	if err != nil {
		exception.New("redis LINDEX error: %s", 500, err.Error()).Throw()
	}

	return val
}

// ProcessLSet 通过索引设置列表中的元素值
// @param connName string 连接器名称
// @param key string 键名
// @param index int 索引
// @param value any 新值
// @return string "OK"
func ProcessLSet(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	index := process.ArgsInt(2)
	value := process.Args[3]

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	err = rdb.LSet(ctx, key, int64(index), value).Err()
	if err != nil {
		exception.New("redis LSET error: %s", 500, err.Error()).Throw()
	}

	return "OK"
}

// ProcessLRem 移除列表中指定数量的元素
// @param connName string 连接器名称
// @param key string 键名
// @param count int 移除数量（0表示全部，正数从头开始，负数从尾开始）
// @param value any 要移除的值
// @return int 移除的数量
func ProcessLRem(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	count := process.ArgsInt(2)
	value := process.Args[3]

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	removed, err := rdb.LRem(ctx, key, int64(count), value).Result()
	if err != nil {
		exception.New("redis LREM error: %s", 500, err.Error()).Throw()
	}

	return removed
}
