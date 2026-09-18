package asynq

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/hibiken/asynq"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
)

const TaskTypeYaoProcess = "yao:process:task"

var (
	client    *asynq.Client
	server    *asynq.Server
	inspector *asynq.Inspector
	redisOpt  asynq.RedisClientOpt
	enabled   bool
	mu        sync.RWMutex
)

// asynqLogAdapter 桥接 asynq.Logger 到 kun/log
type asynqLogAdapter struct{}

func (l *asynqLogAdapter) Debug(args ...interface{}) {
	log.Debug("[Asynq] " + fmt.Sprint(args...))
}

func (l *asynqLogAdapter) Info(args ...interface{}) {
	log.Info("[Asynq] " + fmt.Sprint(args...))
}

func (l *asynqLogAdapter) Warn(args ...interface{}) {
	log.Warn("[Asynq] " + fmt.Sprint(args...))
}

func (l *asynqLogAdapter) Error(args ...interface{}) {
	log.Error("[Asynq] " + fmt.Sprint(args...))
}

func (l *asynqLogAdapter) Fatal(args ...interface{}) {
	log.Fatal("[Asynq] " + fmt.Sprint(args...))
}

// TaskPayload 存入 asynq 队列的负载数据
type TaskPayload struct {
	Process string        `json:"process"`
	Args    []interface{} `json:"args"`
}

// IsConfigured 检查是否显式配置了 Redis 连接信息
func IsConfigured() bool {
	// 1. 显式指定 session store 为 redis（通过配置文件或环境变量 YAO_SESSION_STORE=redis）
	if config.Conf.Session.Store == "redis" || os.Getenv("YAO_SESSION_STORE") == "redis" {
		return true
	}
	// 2. 显式设置了 YAO_REDIS_* 环境变量
	if os.Getenv("YAO_REDIS_HOST") != "" || os.Getenv("YAO_REDIS_PORT") != "" {
		return true
	}
	// 3. 显式设置了 YAO_SESSION_HOST 环境变量（区别于结构体 tag 的 envDefault 默认值）
	if os.Getenv("YAO_SESSION_HOST") != "" {
		return true
	}
	return false
}

// CheckConnection 快速检测目标 Redis 地址是否连通，返回是否连通与具体错误
func CheckConnection(addr string, timeout time.Duration) (bool, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false, err
	}
	_ = conn.Close()
	return true, nil
}

// IsEnabled 获取当前 Asynq 是否已成功启用
func IsEnabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return enabled
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
		dbStr := strings.TrimSpace(strings.Split(config.Conf.Session.DB, "#")[0])
		if d, err := strconv.Atoi(dbStr); err == nil {
			dbInt = d
		}
	}
	if envDB := os.Getenv("YAO_REDIS_DB"); envDB != "" {
		dbStr := strings.TrimSpace(strings.Split(envDB, "#")[0])
		if d, err := strconv.Atoi(dbStr); err == nil {
			dbInt = d
		}
	}

	return asynq.RedisClientOpt{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: password,
		DB:       dbInt,
	}
}

// initClientLocked 在持有锁的情况下初始化或重置 client
func initClientLocked(opt asynq.RedisClientOpt) {
	if client != nil && (redisOpt.Addr != opt.Addr || redisOpt.Password != opt.Password || redisOpt.DB != opt.DB) {
		client.Close()
		client = nil
		if inspector != nil {
			inspector.Close()
			inspector = nil
		}
	}
	if client == nil {
		client = asynq.NewClient(opt)
	}
	if inspector == nil {
		inspector = asynq.NewInspector(opt)
	}
	redisOpt = opt
	enabled = true
}

// Init 初始化 Asynq 模块客户端 Client 与 Inspector（若 Redis 可用）
func Init() {
	mu.Lock()
	defer mu.Unlock()

	opt := GetRedisOpt()
	configured := IsConfigured()
	timeout := 300 * time.Millisecond
	if configured {
		timeout = 2 * time.Second
	}
	reachable, _ := CheckConnection(opt.Addr, timeout)

	if !configured && !reachable {
		log.Debug("[Asynq] Redis is not configured and %s is unreachable, Asynq client disabled", opt.Addr)
		enabled = false
		return
	}

	initClientLocked(opt)
}

// GetClient 安全获取 Asynq Client，若尚未初始化且 Redis 可用则按需初始化
func GetClient() (*asynq.Client, error) {
	mu.Lock()
	defer mu.Unlock()

	opt := GetRedisOpt()
	if client == nil || redisOpt.Addr != opt.Addr || redisOpt.Password != opt.Password || redisOpt.DB != opt.DB {
		configured := IsConfigured()
		timeout := 300 * time.Millisecond
		if configured {
			timeout = 2 * time.Second
		}
		reachable, err := CheckConnection(opt.Addr, timeout)

		if !configured && !reachable {
			enabled = false
			return nil, fmt.Errorf("Redis is not configured and %s is unreachable (%v)", opt.Addr, err)
		}

		if configured && !reachable {
			enabled = false
			return nil, fmt.Errorf("Configured Redis server at %s is unreachable (%v)", opt.Addr, err)
		}

		initClientLocked(opt)
	}

	return client, nil
}

// GetInspector 安全获取 Asynq Inspector，若尚未初始化且 Redis 可用则按需初始化
func GetInspector() (*asynq.Inspector, error) {
	mu.Lock()
	defer mu.Unlock()

	opt := GetRedisOpt()
	if inspector == nil || redisOpt.Addr != opt.Addr || redisOpt.Password != opt.Password || redisOpt.DB != opt.DB {
		configured := IsConfigured()
		timeout := 300 * time.Millisecond
		if configured {
			timeout = 2 * time.Second
		}
		reachable, err := CheckConnection(opt.Addr, timeout)

		if !configured && !reachable {
			enabled = false
			return nil, fmt.Errorf("Redis is not configured and %s is unreachable (%v)", opt.Addr, err)
		}

		if configured && !reachable {
			enabled = false
			return nil, fmt.Errorf("Configured Redis server at %s is unreachable (%v)", opt.Addr, err)
		}

		initClientLocked(opt)
	}

	return inspector, nil
}

// StartServer 启动 Asynq 后台消费 Worker 线程池（由服务常驻命令 yao start 统一生命周期管理）
func StartServer() {
	mu.Lock()
	defer mu.Unlock()

	if server != nil {
		return
	}

	opt := GetRedisOpt()
	configured := IsConfigured()

	timeout := 300 * time.Millisecond
	if configured {
		timeout = 2 * time.Second
	}
	reachable, checkErr := CheckConnection(opt.Addr, timeout)

	if !configured {
		if !reachable {
			// 未显式配置且 127.0.0.1:6379 不可达，优雅跳过并在终端打出明确提示
			msg := fmt.Sprintf("[Asynq] Redis not configured (%s unreachable: %v), task queue skipped.", opt.Addr, checkErr)
			log.Info("%s", msg)
			fmt.Println(color.WhiteString("%s", msg))
			enabled = false
			return
		}
		msg := fmt.Sprintf("[Asynq] Local Redis detected on %s, worker server started.", opt.Addr)
		log.Info("%s", msg)
		fmt.Println(color.GreenString("%s", msg))
	} else {
		if !reachable {
			// 显式配置但连不上，在控制台打印高亮警告并附带具体错误原因
			msg := fmt.Sprintf("[Asynq] Configured Redis (%s) unreachable: %v. Worker server disabled.", opt.Addr, checkErr)
			log.Warn("%s", msg)
			fmt.Println(color.YellowString("%s", msg))
			enabled = false
			return
		}
		msg := fmt.Sprintf("[Asynq] Worker server started on %s (Redis connected)", opt.Addr)
		log.Info("%s", msg)
		fmt.Println(color.GreenString("%s", msg))
	}

	server = asynq.NewServer(opt, asynq.Config{
		Concurrency: 10,
		Logger:      &asynqLogAdapter{},
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			log.Error("[Asynq] Task %s error: %v", task.Type(), err)
		}),
	})

	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskTypeYaoProcess, handleYaoTask)

	if err := server.Start(mux); err != nil {
		log.Error("[Asynq] Server start error: %v", err)
		fmt.Println(color.RedString("[Asynq] Server start error: %v", err))
		return
	}

	initClientLocked(opt)
}

// Stop 停止 Asynq 服务
func Stop() {
	mu.Lock()
	defer mu.Unlock()

	if client != nil {
		client.Close()
		client = nil
	}
	if server != nil {
		server.Shutdown()
		server = nil
	}
	if inspector != nil {
		inspector.Close()
		inspector = nil
	}
	enabled = false
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
