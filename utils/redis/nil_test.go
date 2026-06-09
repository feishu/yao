package redis

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/process"
)

func TestProcessHGetReturnsNilWhenFieldIsMissing(t *testing.T) {
	connName := "redis-empty-hget"
	registerRedisTestClient(t, connName, startRedisNilServer(t))

	var result interface{}
	require.NotPanics(t, func() {
		result = ProcessHGet(process.New("utils.redis.HGet", connName, "hash", "missing"))
	})
	require.Nil(t, result)
}

func TestProcessLIndexReturnsNilWhenIndexIsMissing(t *testing.T) {
	connName := "redis-empty-lindex"
	registerRedisTestClient(t, connName, startRedisNilServer(t))

	var result interface{}
	require.NotPanics(t, func() {
		result = ProcessLIndex(process.New("utils.redis.LIndex", connName, "list", 99))
	})
	require.Nil(t, result)
}

func TestProcessSPopReturnsNilWhenSetIsEmpty(t *testing.T) {
	connName := "redis-empty-spop"
	registerRedisTestClient(t, connName, startRedisNilServer(t))

	var result interface{}
	require.NotPanics(t, func() {
		result = ProcessSPop(process.New("utils.redis.SPop", connName, "set"))
	})
	require.Nil(t, result)
}

func TestProcessSRandMemberReturnsNilWhenSetIsEmpty(t *testing.T) {
	connName := "redis-empty-srandmember"
	registerRedisTestClient(t, connName, startRedisNilServer(t))

	var result interface{}
	require.NotPanics(t, func() {
		result = ProcessSRandMember(process.New("utils.redis.SRandMember", connName, "set"))
	})
	require.Nil(t, result)
}

func TestProcessZScoreReturnsNilWhenMemberIsMissing(t *testing.T) {
	connName := "redis-empty-zscore"
	registerRedisTestClient(t, connName, startRedisNilServer(t))

	var result interface{}
	require.NotPanics(t, func() {
		result = ProcessZScore(process.New("utils.redis.ZScore", connName, "zset", "missing"))
	})
	require.Nil(t, result)
}

func TestProcessZRankReturnsNilWhenMemberIsMissing(t *testing.T) {
	connName := "redis-empty-zrank"
	registerRedisTestClient(t, connName, startRedisNilServer(t))

	var result interface{}
	require.NotPanics(t, func() {
		result = ProcessZRank(process.New("utils.redis.ZRank", connName, "zset", "missing"))
	})
	require.Nil(t, result)
}

func TestProcessZRevRankReturnsNilWhenMemberIsMissing(t *testing.T) {
	connName := "redis-empty-zrevrank"
	registerRedisTestClient(t, connName, startRedisNilServer(t))

	var result interface{}
	require.NotPanics(t, func() {
		result = ProcessZRevRank(process.New("utils.redis.ZRevRank", connName, "zset", "missing"))
	})
	require.Nil(t, result)
}

func TestProcessRankingGetRankReturnsNilWhenMemberIsMissing(t *testing.T) {
	connName := "redis-empty-ranking-rank"
	registerRedisTestClient(t, connName, startRedisNilServer(t))

	var result interface{}
	require.NotPanics(t, func() {
		result = ProcessRankingGetRank(process.New("utils.redis.RankingGetRank", connName, "ranking", "missing"))
	})
	require.Nil(t, result)
}

func TestProcessBatchReturnsNilForNilScalarCommands(t *testing.T) {
	tests := []string{"HGET", "LPOP", "RPOP", "ZSCORE", "ZRANK", "ZREVRANK"}

	for _, cmd := range tests {
		t.Run(cmd, func(t *testing.T) {
			connName := "redis-empty-batch-" + cmd
			registerRedisTestClient(t, connName, startRedisNilServer(t))

			result := ProcessBatch(process.New("utils.redis.Batch", connName, []interface{}{
				map[string]interface{}{"cmd": cmd, "args": batchNilArgs(cmd)},
			}))

			require.Equal(t, []interface{}{nil}, result)
		})
	}
}

func batchNilArgs(cmd string) []interface{} {
	switch cmd {
	case "HGET":
		return []interface{}{"hash", "missing"}
	case "ZSCORE", "ZRANK", "ZREVRANK":
		return []interface{}{"zset", "missing"}
	default:
		return []interface{}{"key"}
	}
}
