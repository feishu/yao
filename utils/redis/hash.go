package redis

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

// ProcessHGet 获取哈希表字段的值
// @param connName string 连接器名称
// @param key string 键名
// @param field string 字段名
// @return string 字段值
func ProcessHGet(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	field := process.ArgsString(2)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	val, err := rdb.HGet(ctx, key, field).Result()
	if err != nil {
		exception.New("redis HGET error: %s", 500, err.Error()).Throw()
	}

	return val
}

// ProcessHSet 设置哈希表字段的值
// @param connName string 连接器名称
// @param key string 键名
// @param field string 字段名
// @param value any 字段值
// @return int 新增的字段数量（0或1）
func ProcessHSet(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	field := process.ArgsString(2)
	value := process.Args[3]

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	count, err := rdb.HSet(ctx, key, field, value).Result()
	if err != nil {
		exception.New("redis HSET error: %s", 500, err.Error()).Throw()
	}

	return count
}

// ProcessHMSet 批量设置哈希表的字段
// @param connName string 连接器名称
// @param key string 键名
// @param values map[string]interface{} 字段-值映射
// @return string "OK"
func ProcessHMSet(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	values := process.ArgsMap(2)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	err = rdb.HMSet(ctx, key, values).Err()
	if err != nil {
		exception.New("redis HMSET error: %s", 500, err.Error()).Throw()
	}

	return "OK"
}

// ProcessHGetAll 获取哈希表所有字段和值
// @param connName string 连接器名称
// @param key string 键名
// @return map[string]string 所有字段和值
func ProcessHGetAll(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	vals, err := rdb.HGetAll(ctx, key).Result()
	if err != nil {
		exception.New("redis HGETALL error: %s", 500, err.Error()).Throw()
	}

	return vals
}

// ProcessHDel 删除哈希表的一个或多个字段
// @param connName string 连接器名称
// @param key string 键名
// @param fields ...string 一个或多个字段名
// @return int 删除的字段数量
func ProcessHDel(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	fields := make([]string, 0)
	for i := 2; i < process.NumOfArgs(); i++ {
		fields = append(fields, process.ArgsString(i))
	}

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	count, err := rdb.HDel(ctx, key, fields...).Result()
	if err != nil {
		exception.New("redis HDEL error: %s", 500, err.Error()).Throw()
	}

	return count
}

// ProcessHExists 检查哈希表字段是否存在
// @param connName string 连接器名称
// @param key string 键名
// @param field string 字段名
// @return bool 是否存在
func ProcessHExists(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	field := process.ArgsString(2)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	exists, err := rdb.HExists(ctx, key, field).Result()
	if err != nil {
		exception.New("redis HEXISTS error: %s", 500, err.Error()).Throw()
	}

	return exists
}

// ProcessHKeys 获取哈希表所有字段名
// @param connName string 连接器名称
// @param key string 键名
// @return []string 所有字段名
func ProcessHKeys(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	keys, err := rdb.HKeys(ctx, key).Result()
	if err != nil {
		exception.New("redis HKEYS error: %s", 500, err.Error()).Throw()
	}

	return keys
}

// ProcessHVals 获取哈希表所有值
// @param connName string 连接器名称
// @param key string 键名
// @return []string 所有值
func ProcessHVals(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	vals, err := rdb.HVals(ctx, key).Result()
	if err != nil {
		exception.New("redis HVALS error: %s", 500, err.Error()).Throw()
	}

	return vals
}

// ProcessHLen 获取哈希表字段数量
// @param connName string 连接器名称
// @param key string 键名
// @return int 字段数量
func ProcessHLen(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	count, err := rdb.HLen(ctx, key).Result()
	if err != nil {
		exception.New("redis HLEN error: %s", 500, err.Error()).Throw()
	}

	return count
}

// ProcessHIncrBy 哈希表字段值增加指定值
// @param connName string 连接器名称
// @param key string 键名
// @param field string 字段名
// @param value int 增加的值
// @return int 增加后的值
func ProcessHIncrBy(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	field := process.ArgsString(2)
	value := process.ArgsInt(3)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	val, err := rdb.HIncrBy(ctx, key, field, int64(value)).Result()
	if err != nil {
		exception.New("redis HINCRBY error: %s", 500, err.Error()).Throw()
	}

	return val
}
