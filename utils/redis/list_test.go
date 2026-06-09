package redis

import (
	"net"
	"testing"
	"time"

	goredis "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/connector"
	redisConnector "github.com/yaoapp/gou/connector/redis"
	"github.com/yaoapp/gou/process"
)

func TestProcessLPopReturnsNilWhenListIsEmpty(t *testing.T) {
	addr := startRedisNilServer(t)
	connName := "redis-empty-lpop"
	key := "queue"
	registerRedisTestClient(t, connName, addr)

	var result interface{}
	require.NotPanics(t, func() {
		result = ProcessLPop(process.New("utils.redis.LPop", connName, key))
	})
	require.Nil(t, result)
}

func TestProcessRPopReturnsNilWhenListIsEmpty(t *testing.T) {
	addr := startRedisNilServer(t)
	connName := "redis-empty-rpop"
	key := "queue"
	registerRedisTestClient(t, connName, addr)

	var result interface{}
	require.NotPanics(t, func() {
		result = ProcessRPop(process.New("utils.redis.RPop", connName, key))
	})
	require.Nil(t, result)
}

func registerRedisTestClient(t *testing.T, connName string, addr string) {
	t.Helper()

	rdb := goredis.NewClient(&goredis.Options{
		Addr:        addr,
		DialTimeout: time.Second,
		ReadTimeout: time.Second,
	})
	connector.Connectors[connName] = &redisConnector.Connector{Name: connName, Rdb: rdb}
	t.Cleanup(func() {
		delete(connector.Connectors, connName)
		require.NoError(t, rdb.Close())
	})
}

func startRedisNilServer(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, listener.Close())
	})

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(time.Second))

		buf := make([]byte, 256)
		_, _ = conn.Read(buf)
		_, _ = conn.Write([]byte("$-1\r\n"))
	}()

	return listener.Addr().String()
}
