package redis

import (
	"fmt"
	"time"

	goredis "github.com/go-redis/redis/v8"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

// ProcessPipeline 执行 Redis Pipeline 批量命令
// @param connName string 连接器名称
// @param commands []map[string]interface{} 命令列表，每个命令包含 cmd 和 args 字段
// 命令格式: [{"cmd": "SET", "args": ["key1", "value1"]}, {"cmd": "GET", "args": ["key1"]}]
// @return []interface{} 每个命令的执行结果
func ProcessPipeline(process *process.Process) interface{} {
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
		exception.New("redis PIPELINE error: %s", 500, err.Error()).Throw()
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

// executePipelineCommand 执行单个 Pipeline 命令
func executePipelineCommand(pipe goredis.Pipeliner, cmdName string, args []interface{}) goredis.Cmder {
	switch cmdName {
	case "GET":
		if len(args) < 1 {
			return nil
		}
		return pipe.Get(ctx, toString(args[0]))

	case "SET":
		if len(args) < 2 {
			return nil
		}
		return pipe.Set(ctx, toString(args[0]), args[1], 0)

	case "DEL":
		keys := make([]string, len(args))
		for i, arg := range args {
			keys[i] = toString(arg)
		}
		return pipe.Del(ctx, keys...)

	case "UNLINK":
		keys := make([]string, len(args))
		for i, arg := range args {
			keys[i] = toString(arg)
		}
		return pipe.Unlink(ctx, keys...)

	case "EXISTS":
		keys := make([]string, len(args))
		for i, arg := range args {
			keys[i] = toString(arg)
		}
		return pipe.Exists(ctx, keys...)

	case "EXPIRE":
		if len(args) < 2 {
			return nil
		}
		return pipe.Expire(ctx, toString(args[0]), time.Duration(toInt64(args[1]))*time.Second)

	case "TTL":
		if len(args) < 1 {
			return nil
		}
		return pipe.TTL(ctx, toString(args[0]))

	case "INCR":
		if len(args) < 1 {
			return nil
		}
		return pipe.Incr(ctx, toString(args[0]))

	case "DECR":
		if len(args) < 1 {
			return nil
		}
		return pipe.Decr(ctx, toString(args[0]))

	case "INCRBY":
		if len(args) < 2 {
			return nil
		}
		return pipe.IncrBy(ctx, toString(args[0]), toInt64(args[1]))

	case "DECRBY":
		if len(args) < 2 {
			return nil
		}
		return pipe.DecrBy(ctx, toString(args[0]), toInt64(args[1]))

	// Hash operations
	case "HGET":
		if len(args) < 2 {
			return nil
		}
		return pipe.HGet(ctx, toString(args[0]), toString(args[1]))

	case "HSET":
		if len(args) < 3 {
			return nil
		}
		return pipe.HSet(ctx, toString(args[0]), toString(args[1]), args[2])

	case "HGETALL":
		if len(args) < 1 {
			return nil
		}
		return pipe.HGetAll(ctx, toString(args[0]))

	case "HDEL":
		if len(args) < 2 {
			return nil
		}
		fields := make([]string, len(args)-1)
		for i := 1; i < len(args); i++ {
			fields[i-1] = toString(args[i])
		}
		return pipe.HDel(ctx, toString(args[0]), fields...)

	// List operations
	case "LPUSH":
		if len(args) < 2 {
			return nil
		}
		return pipe.LPush(ctx, toString(args[0]), args[1:]...)

	case "RPUSH":
		if len(args) < 2 {
			return nil
		}
		return pipe.RPush(ctx, toString(args[0]), args[1:]...)

	case "LPOP":
		if len(args) < 1 {
			return nil
		}
		return pipe.LPop(ctx, toString(args[0]))

	case "RPOP":
		if len(args) < 1 {
			return nil
		}
		return pipe.RPop(ctx, toString(args[0]))

	case "LRANGE":
		if len(args) < 3 {
			return nil
		}
		return pipe.LRange(ctx, toString(args[0]), toInt64(args[1]), toInt64(args[2]))

	case "LLEN":
		if len(args) < 1 {
			return nil
		}
		return pipe.LLen(ctx, toString(args[0]))

	// Set operations
	case "SADD":
		if len(args) < 2 {
			return nil
		}
		return pipe.SAdd(ctx, toString(args[0]), args[1:]...)

	case "SREM":
		if len(args) < 2 {
			return nil
		}
		return pipe.SRem(ctx, toString(args[0]), args[1:]...)

	case "SMEMBERS":
		if len(args) < 1 {
			return nil
		}
		return pipe.SMembers(ctx, toString(args[0]))

	case "SISMEMBER":
		if len(args) < 2 {
			return nil
		}
		return pipe.SIsMember(ctx, toString(args[0]), args[1])

	case "SCARD":
		if len(args) < 1 {
			return nil
		}
		return pipe.SCard(ctx, toString(args[0]))

	// Sorted Set operations
	case "ZADD":
		if len(args) < 3 {
			return nil
		}
		members := make([]*goredis.Z, 0)
		for i := 1; i < len(args); i += 2 {
			if i+1 < len(args) {
				members = append(members, &goredis.Z{
					Score:  toFloat64(args[i]),
					Member: args[i+1],
				})
			}
		}
		return pipe.ZAdd(ctx, toString(args[0]), members...)

	case "ZREM":
		if len(args) < 2 {
			return nil
		}
		return pipe.ZRem(ctx, toString(args[0]), args[1:]...)

	case "ZRANGE":
		if len(args) < 3 {
			return nil
		}
		return pipe.ZRange(ctx, toString(args[0]), toInt64(args[1]), toInt64(args[2]))

	case "ZREVRANGE":
		if len(args) < 3 {
			return nil
		}
		return pipe.ZRevRange(ctx, toString(args[0]), toInt64(args[1]), toInt64(args[2]))

	case "ZSCORE":
		if len(args) < 2 {
			return nil
		}
		return pipe.ZScore(ctx, toString(args[0]), toString(args[1]))

	case "ZCARD":
		if len(args) < 1 {
			return nil
		}
		return pipe.ZCard(ctx, toString(args[0]))

	case "ZRANK":
		if len(args) < 2 {
			return nil
		}
		return pipe.ZRank(ctx, toString(args[0]), toString(args[1]))

	case "ZREVRANK":
		if len(args) < 2 {
			return nil
		}
		return pipe.ZRevRank(ctx, toString(args[0]), toString(args[1]))

	default:
		return nil
	}
}

// parseCommandResult 解析命令结果
func parseCommandResult(cmd goredis.Cmder) (interface{}, error) {
	switch c := cmd.(type) {
	case *goredis.StringCmd:
		val, err := c.Result()
		if err == goredis.Nil {
			return nil, nil
		}
		return val, err

	case *goredis.IntCmd:
		return c.Result()

	case *goredis.BoolCmd:
		return c.Result()

	case *goredis.FloatCmd:
		return c.Result()

	case *goredis.StringSliceCmd:
		return c.Result()

	case *goredis.StringStringMapCmd:
		return c.Result()

	case *goredis.DurationCmd:
		dur, err := c.Result()
		if err != nil {
			return nil, err
		}
		return int64(dur.Seconds()), nil

	case *goredis.StatusCmd:
		return c.Result()

	default:
		return nil, fmt.Errorf("unsupported command type: %T", cmd)
	}
}

// toString 转换为字符串
func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// toInt64 转换为 int64
func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int:
		return int64(val)
	case int64:
		return val
	case float64:
		return int64(val)
	default:
		return 0
	}
}

// toFloat64 转换为 float64
func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}
