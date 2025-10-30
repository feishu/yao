package redis

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

// ProcessSAdd 向集合添加一个或多个成员
// @param connName string 连接器名称
// @param key string 键名
// @param members ...any 一个或多个成员
// @return int 新增的成员数量
func ProcessSAdd(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	members := make([]interface{}, 0)
	for i := 2; i < process.NumOfArgs(); i++ {
		members = append(members, process.Args[i])
	}

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	count, err := rdb.SAdd(ctx, key, members...).Result()
	if err != nil {
		exception.New("redis SADD error: %s", 500, err.Error()).Throw()
	}

	return count
}

// ProcessSRem 移除集合中的一个或多个成员
// @param connName string 连接器名称
// @param key string 键名
// @param members ...any 一个或多个成员
// @return int 移除的成员数量
func ProcessSRem(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	members := make([]interface{}, 0)
	for i := 2; i < process.NumOfArgs(); i++ {
		members = append(members, process.Args[i])
	}

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	count, err := rdb.SRem(ctx, key, members...).Result()
	if err != nil {
		exception.New("redis SREM error: %s", 500, err.Error()).Throw()
	}

	return count
}

// ProcessSMembers 获取集合的所有成员
// @param connName string 连接器名称
// @param key string 键名
// @return []string 所有成员
func ProcessSMembers(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	members, err := rdb.SMembers(ctx, key).Result()
	if err != nil {
		exception.New("redis SMEMBERS error: %s", 500, err.Error()).Throw()
	}

	return members
}

// ProcessSIsMember 判断成员是否在集合中
// @param connName string 连接器名称
// @param key string 键名
// @param member any 成员
// @return bool 是否存在
func ProcessSIsMember(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	member := process.Args[2]

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	exists, err := rdb.SIsMember(ctx, key, member).Result()
	if err != nil {
		exception.New("redis SISMEMBER error: %s", 500, err.Error()).Throw()
	}

	return exists
}

// ProcessSCard 获取集合的成员数量
// @param connName string 连接器名称
// @param key string 键名
// @return int 成员数量
func ProcessSCard(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	count, err := rdb.SCard(ctx, key).Result()
	if err != nil {
		exception.New("redis SCARD error: %s", 500, err.Error()).Throw()
	}

	return count
}

// ProcessSPop 移除并返回集合中的一个随机元素
// @param connName string 连接器名称
// @param key string 键名
// @return string 随机元素
func ProcessSPop(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	member, err := rdb.SPop(ctx, key).Result()
	if err != nil {
		exception.New("redis SPOP error: %s", 500, err.Error()).Throw()
	}

	return member
}

// ProcessSRandMember 返回集合中的一个或多个随机元素
// @param connName string 连接器名称
// @param key string 键名
// @param count int 返回数量（可选，默认1）
// @return []string 随机元素
func ProcessSRandMember(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	if process.NumOfArgs() > 2 {
		count := process.ArgsInt(2)
		members, err := rdb.SRandMemberN(ctx, key, int64(count)).Result()
		if err != nil {
			exception.New("redis SRANDMEMBER error: %s", 500, err.Error()).Throw()
		}
		return members
	}

	member, err := rdb.SRandMember(ctx, key).Result()
	if err != nil {
		exception.New("redis SRANDMEMBER error: %s", 500, err.Error()).Throw()
	}

	return member
}

// ProcessSUnion 返回多个集合的并集
// @param connName string 连接器名称
// @param keys ...string 一个或多个键名
// @return []string 并集成员
func ProcessSUnion(process *process.Process) interface{} {
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

	members, err := rdb.SUnion(ctx, keys...).Result()
	if err != nil {
		exception.New("redis SUNION error: %s", 500, err.Error()).Throw()
	}

	return members
}

// ProcessSInter 返回多个集合的交集
// @param connName string 连接器名称
// @param keys ...string 一个或多个键名
// @return []string 交集成员
func ProcessSInter(process *process.Process) interface{} {
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

	members, err := rdb.SInter(ctx, keys...).Result()
	if err != nil {
		exception.New("redis SINTER error: %s", 500, err.Error()).Throw()
	}

	return members
}

// ProcessSDiff 返回多个集合的差集
// @param connName string 连接器名称
// @param keys ...string 一个或多个键名
// @return []string 差集成员
func ProcessSDiff(process *process.Process) interface{} {
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

	members, err := rdb.SDiff(ctx, keys...).Result()
	if err != nil {
		exception.New("redis SDIFF error: %s", 500, err.Error()).Throw()
	}

	return members
}
