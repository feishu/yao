package asynq

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/hibiken/asynq"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
)

const TaskTypeYaoProcess = "yao:process:task"

var client *asynq.Client
var server *asynq.Server
var inspector *asynq.Inspector
var redisOpt asynq.RedisClientOpt

// TaskPayload 存入 asynq 队列的负载数据
type TaskPayload struct {
	Process string        `json:"process"`
	Args    []interface{} `json:"args"`
}

// GetRedisOpt 获取 Redis 连接选项
func GetRedisOpt() asynq.RedisClientOpt {
	host := config.Conf.Session.Host
	port := config.Conf.Session.Port
	password := config.Conf.Session.Password

	// 如果环境变量覆盖
	if envHost := os.Getenv("YAO_REDIS_HOST"); envHost != "" {
		host = envHost
	}
	if envPort := os.Getenv("YAO_REDIS_PORT"); envPort != "" {
		port = envPort
	}
	if envPass := os.Getenv("YAO_REDIS_PASSWORD"); envPass != "" {
		password = envPass
	}

	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "6379"
	}

	dbInt := 1
	if config.Conf.Session.DB != "" {
		if d, err := strconv.Atoi(config.Conf.Session.DB); err == nil {
			dbInt = d
		}
	}
	if envDB := os.Getenv("YAO_REDIS_DB"); envDB != "" {
		if d, err := strconv.Atoi(envDB); err == nil {
			dbInt = d
		}
	}

	return asynq.RedisClientOpt{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: password,
		DB:       dbInt,
	}
}

// Init 初始化 Asynq 模块 Client 与 Background Worker Server
func Init() {
	opt := GetRedisOpt()
	redisOpt = opt

	client = asynq.NewClient(opt)
	inspector = asynq.NewInspector(opt)

	// 后台 Worker 线程池，并发数默认为 10
	server = asynq.NewServer(opt, asynq.Config{
		Concurrency: 10,
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			log.Error("[Asynq] Task %s error: %v", task.Type(), err)
		}),
	})

	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskTypeYaoProcess, handleYaoTask)

	go func() {
		log.Info("[Asynq] Starting Asynq background worker server on %s", opt.Addr)
		if err := server.Run(mux); err != nil {
			log.Error("[Asynq] Server error: %v", err)
		}
	}()
}

// Stop 停止 Asynq 服务
func Stop() {
	if client != nil {
		client.Close()
	}
	if server != nil {
		server.Shutdown()
	}
	if inspector != nil {
		inspector.Close()
	}
}

// handleYaoTask 执行具体 Yao Process
func handleYaoTask(ctx context.Context, t *asynq.Task) error {
	var payload TaskPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal task payload: %w", err)
	}

	log.Debug("[Asynq] Processing task process=%s args=%v", payload.Process, payload.Args)
	p, err := process.Of(payload.Process, payload.Args...)
	if err != nil {
		return fmt.Errorf("failed to load process %s: %w", payload.Process, err)
	}
	defer p.Release()

	err = p.Execute()
	if err != nil {
		return fmt.Errorf("failed to execute process %s: %w", payload.Process, err)
	}

	return nil
}
