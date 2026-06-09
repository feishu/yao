package redis

import (
	goredis "github.com/go-redis/redis/v8"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

// ProcessZAdd 向有序集合添加一个或多个成员
// @param connName string 连接器名称
// @param key string 键名
// @param members map[string]interface{} 成员-分数映射，格式: {"member1": score1, "member2": score2}
// @return int 新增的成员数量
func ProcessZAdd(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	membersMap := process.ArgsMap(2)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	members := make([]*goredis.Z, 0)
	for member, scoreVal := range membersMap {
		var score float64
		switch v := scoreVal.(type) {
		case float64:
			score = v
		case int:
			score = float64(v)
		case int64:
			score = float64(v)
		default:
			exception.New("invalid score type for member %s", 400, member).Throw()
		}
		members = append(members, &goredis.Z{
			Score:  score,
			Member: member,
		})
	}

	count, err := rdb.ZAdd(ctx, key, members...).Result()
	if err != nil {
		exception.New("redis ZADD error: %s", 500, err.Error()).Throw()
	}

	return count
}

// ProcessZRem 移除有序集合中的一个或多个成员
// @param connName string 连接器名称
// @param key string 键名
// @param members ...any 一个或多个成员
// @return int 移除的成员数量
func ProcessZRem(process *process.Process) interface{} {
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

	count, err := rdb.ZRem(ctx, key, members...).Result()
	if err != nil {
		exception.New("redis ZREM error: %s", 500, err.Error()).Throw()
	}

	return count
}

// ProcessZRange 返回有序集合指定索引范围内的成员
// @param connName string 连接器名称
// @param key string 键名
// @param start int 起始索引
// @param stop int 结束索引
// @param withScores bool 是否返回分数（可选，默认false）
// @return []string 或 []map[string]interface{} 成员列表
func ProcessZRange(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	start := process.ArgsInt(2)
	stop := process.ArgsInt(3)

	withScores := false
	if process.NumOfArgs() > 4 {
		withScores = process.ArgsBool(4)
	}

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	if withScores {
		members, err := rdb.ZRangeWithScores(ctx, key, int64(start), int64(stop)).Result()
		if err != nil {
			exception.New("redis ZRANGE error: %s", 500, err.Error()).Throw()
		}

		result := make([]map[string]interface{}, 0)
		for _, m := range members {
			result = append(result, map[string]interface{}{
				"member": m.Member,
				"score":  m.Score,
			})
		}
		return result
	}

	members, err := rdb.ZRange(ctx, key, int64(start), int64(stop)).Result()
	if err != nil {
		exception.New("redis ZRANGE error: %s", 500, err.Error()).Throw()
	}

	return members
}

// ProcessZRevRange 返回有序集合指定索引范围内的成员（按分数从高到低）
// @param connName string 连接器名称
// @param key string 键名
// @param start int 起始索引
// @param stop int 结束索引
// @param withScores bool 是否返回分数（可选，默认false）
// @return []string 或 []map[string]interface{} 成员列表
func ProcessZRevRange(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	start := process.ArgsInt(2)
	stop := process.ArgsInt(3)

	withScores := false
	if process.NumOfArgs() > 4 {
		withScores = process.ArgsBool(4)
	}

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	if withScores {
		members, err := rdb.ZRevRangeWithScores(ctx, key, int64(start), int64(stop)).Result()
		if err != nil {
			exception.New("redis ZREVRANGE error: %s", 500, err.Error()).Throw()
		}

		result := make([]map[string]interface{}, 0)
		for _, m := range members {
			result = append(result, map[string]interface{}{
				"member": m.Member,
				"score":  m.Score,
			})
		}
		return result
	}

	members, err := rdb.ZRevRange(ctx, key, int64(start), int64(stop)).Result()
	if err != nil {
		exception.New("redis ZREVRANGE error: %s", 500, err.Error()).Throw()
	}

	return members
}

// ProcessZRangeByScore 返回有序集合指定分数范围内的成员
// @param connName string 连接器名称
// @param key string 键名
// @param min string 最小分数
// @param max string 最大分数
// @param withScores bool 是否返回分数（可选，默认false）
// @return []string 或 []map[string]interface{} 成员列表
func ProcessZRangeByScore(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	min := process.ArgsString(2)
	max := process.ArgsString(3)

	withScores := false
	if process.NumOfArgs() > 4 {
		withScores = process.ArgsBool(4)
	}

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	if withScores {
		members, err := rdb.ZRangeByScoreWithScores(ctx, key, &goredis.ZRangeBy{
			Min: min,
			Max: max,
		}).Result()
		if err != nil {
			exception.New("redis ZRANGEBYSCORE error: %s", 500, err.Error()).Throw()
		}

		result := make([]map[string]interface{}, 0)
		for _, m := range members {
			result = append(result, map[string]interface{}{
				"member": m.Member,
				"score":  m.Score,
			})
		}
		return result
	}

	members, err := rdb.ZRangeByScore(ctx, key, &goredis.ZRangeBy{
		Min: min,
		Max: max,
	}).Result()
	if err != nil {
		exception.New("redis ZRANGEBYSCORE error: %s", 500, err.Error()).Throw()
	}

	return members
}

// ProcessZScore 获取有序集合成员的分数
// @param connName string 连接器名称
// @param key string 键名
// @param member string 成员
// @return float64 分数
func ProcessZScore(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	member := process.ArgsString(2)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	score, err := rdb.ZScore(ctx, key, member).Result()
	if err == goredis.Nil {
		return nil
	}
	if err != nil {
		exception.New("redis ZSCORE error: %s", 500, err.Error()).Throw()
	}

	return score
}

// ProcessZCard 获取有序集合的成员数量
// @param connName string 连接器名称
// @param key string 键名
// @return int 成员数量
func ProcessZCard(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	count, err := rdb.ZCard(ctx, key).Result()
	if err != nil {
		exception.New("redis ZCARD error: %s", 500, err.Error()).Throw()
	}

	return count
}

// ProcessZCount 获取有序集合指定分数范围内的成员数量
// @param connName string 连接器名称
// @param key string 键名
// @param min string 最小分数
// @param max string 最大分数
// @return int 成员数量
func ProcessZCount(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	min := process.ArgsString(2)
	max := process.ArgsString(3)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	count, err := rdb.ZCount(ctx, key, min, max).Result()
	if err != nil {
		exception.New("redis ZCOUNT error: %s", 500, err.Error()).Throw()
	}

	return count
}

// ProcessZIncrBy 有序集合成员分数增加指定值
// @param connName string 连接器名称
// @param key string 键名
// @param increment float64 增加的值
// @param member string 成员
// @return float64 增加后的分数
func ProcessZIncrBy(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	increment := toFloat64(process.Args[2])
	member := process.ArgsString(3)

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

// ProcessZRank 获取有序集合成员的排名（从低到高）
// @param connName string 连接器名称
// @param key string 键名
// @param member string 成员
// @return int 排名（从0开始）
func ProcessZRank(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	member := process.ArgsString(2)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	rank, err := rdb.ZRank(ctx, key, member).Result()
	if err == goredis.Nil {
		return nil
	}
	if err != nil {
		exception.New("redis ZRANK error: %s", 500, err.Error()).Throw()
	}

	return rank
}

// ProcessZRevRank 获取有序集合成员的排名（从高到低）
// @param connName string 连接器名称
// @param key string 键名
// @param member string 成员
// @return int 排名（从0开始）
func ProcessZRevRank(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	connName := process.ArgsString(0)
	key := process.ArgsString(1)
	member := process.ArgsString(2)

	rdb, err := getRedisClient(connName)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	rank, err := rdb.ZRevRank(ctx, key, member).Result()
	if err == goredis.Nil {
		return nil
	}
	if err != nil {
		exception.New("redis ZREVRANK error: %s", 500, err.Error()).Throw()
	}

	return rank
}
