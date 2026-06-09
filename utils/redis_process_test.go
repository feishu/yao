package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/process"
)

func TestRedisHelperProcessNamesAreRegistered(t *testing.T) {
	Init()

	processNames := []string{
		"utils.redis.HExists",
		"utils.redis.HKeys",
		"utils.redis.HVals",
		"utils.redis.HLen",
		"utils.redis.HIncrBy",
		"utils.redis.LIndex",
		"utils.redis.LSet",
		"utils.redis.LRem",
		"utils.redis.SIsMember",
		"utils.redis.SCard",
		"utils.redis.SPop",
		"utils.redis.SRandMember",
		"utils.redis.SUnion",
		"utils.redis.SInter",
		"utils.redis.SDiff",
		"utils.redis.ZRangeByScore",
		"utils.redis.ZCount",
	}

	for _, name := range processNames {
		require.True(t, process.Exists(name), "%s should be registered", name)
	}
}
