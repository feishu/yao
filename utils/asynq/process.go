package asynq

import (
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

// parseOptions 从 Process 的第 4 个可选参数 (args[3]) 解析 Asynq 选项
func parseOptions(process *process.Process) []asynq.Option {
	opts := []asynq.Option{}
	if process.NumOfArgs() < 4 {
		return opts
	}

	configMap, ok := process.Args[3].(map[string]interface{})
	if !ok {
		return opts
	}

	// 1. Queue 指定
	if q, ok := configMap["queue"].(string); ok && q != "" {
		opts = append(opts, asynq.Queue(q))
	}

	// 2. MaxRetry 指定
	if r, ok := configMap["maxRetry"].(float64); ok && r >= 0 {
		opts = append(opts, asynq.MaxRetry(int(r)))
	} else if r, ok := configMap["maxRetry"].(int); ok && r >= 0 {
		opts = append(opts, asynq.MaxRetry(r))
	}

	// 3. UniqueKey 去重指定
	if uKey, ok := configMap["uniqueKey"].(string); ok && uKey != "" {
		ttlSeconds := 3600
		if ttl, ok := configMap["uniqueTTL"].(float64); ok && ttl > 0 {
			ttlSeconds = int(ttl)
		} else if ttl, ok := configMap["uniqueTTL"].(int); ok && ttl > 0 {
			ttlSeconds = ttl
		}
		opts = append(opts, asynq.Unique(time.Duration(ttlSeconds)*time.Second))
	}

	// 4. Timeout 指定
	if t, ok := configMap["timeout"].(float64); ok && t > 0 {
		opts = append(opts, asynq.Timeout(time.Duration(t)*time.Second))
	} else if t, ok := configMap["timeout"].(int); ok && t > 0 {
		opts = append(opts, asynq.Timeout(time.Duration(t)*time.Second))
	}

	// 5. 完成任务保留时长
	if r, ok := configMap["retention"].(float64); ok && r > 0 {
		opts = append(opts, asynq.Retention(time.Duration(r)*time.Second))
	} else if r, ok := configMap["retention"].(int); ok && r > 0 {
		opts = append(opts, asynq.Retention(time.Duration(r)*time.Second))
	}

	return opts
}

// ProcessEnqueueIn 相对延迟 N 秒后执行指定 Yao Process
// 参数:
// args[0] (string): 目标 Process 名称 (如 "scripts.order.Cancel")
// args[1] ([]interface{}): 传递给 Process 的参数列表
// args[2] (int/float): 延迟时间 (单位: 秒)
// args[3] (map[string]interface{}, 可选): 任务高级配置项 (queue, maxRetry, uniqueKey, timeout 等)
// 返回: (string) 注册生成的 Task ID
func ProcessEnqueueIn(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	targetProcess := process.ArgsString(0)
	args := process.ArgsArray(1)
	delaySeconds := process.ArgsInt(2)

	if client == nil {
		exception.New("[Asynq] Client is not initialized", 500).Throw()
	}

	payloadBytes, err := json.Marshal(TaskPayload{
		Process: targetProcess,
		Args:    args,
	})
	if err != nil {
		exception.New("[Asynq] Marshal Payload Error: %s", 500, err.Error()).Throw()
	}

	opts := append(parseOptions(process), asynq.ProcessIn(time.Duration(delaySeconds)*time.Second))
	task := asynq.NewTask(TaskTypeYaoProcess, payloadBytes)
	info, err := client.Enqueue(task, opts...)
	if err != nil {
		exception.New("[Asynq] EnqueueIn Error: %s", 500, err.Error()).Throw()
	}

	return info.ID
}

// ProcessEnqueueAt 在指定时间点 (ISO时间字符串或Unix秒级时间戳) 执行 Yao Process
// 参数:
// args[0] (string): 目标 Process 名称
// args[1] ([]interface{}): 传递给 Process 的参数列表
// args[2] (string|int): 指定到期时间 (ISO 8601 格式如 "2026-08-05T16:00:00Z" 或 Unix 时间戳)
// args[3] (map[string]interface{}, 可选): 任务高级配置项 (queue, maxRetry, uniqueKey, timeout 等)
// 返回: (string) 注册生成的 Task ID
func ProcessEnqueueAt(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	targetProcess := process.ArgsString(0)
	args := process.ArgsArray(1)

	var targetTime time.Time

	switch v := process.Args[2].(type) {
	case string:
		parsedTime, err := time.Parse(time.RFC3339, v)
		if err != nil {
			parsedTime, err = time.Parse("2006-01-02 15:04:05", v)
			if err != nil {
				exception.New("[Asynq] Invalid time format: %s. Expected RFC3339 or 'YYYY-MM-DD HH:mm:ss'", 400, v).Throw()
			}
		}
		targetTime = parsedTime
	case int:
		targetTime = time.Unix(int64(v), 0)
	case int64:
		targetTime = time.Unix(v, 0)
	case float64:
		targetTime = time.Unix(int64(v), 0)
	default:
		exception.New("[Asynq] Invalid executeAt parameter type", 400).Throw()
	}

	if client == nil {
		exception.New("[Asynq] Client is not initialized", 500).Throw()
	}

	payloadBytes, err := json.Marshal(TaskPayload{
		Process: targetProcess,
		Args:    args,
	})
	if err != nil {
		exception.New("[Asynq] Marshal Payload Error: %s", 500, err.Error()).Throw()
	}

	opts := append(parseOptions(process), asynq.ProcessAt(targetTime))
	task := asynq.NewTask(TaskTypeYaoProcess, payloadBytes)
	info, err := client.Enqueue(task, opts...)
	if err != nil {
		exception.New("[Asynq] EnqueueAt Error: %s", 500, err.Error()).Throw()
	}

	return info.ID
}

// ProcessCancel 撤销尚未执行的延迟 Task
// 参数:
// args[0] (string): Task ID
// args[1] (string, 可选): Queue 名称，默认为 "default"
// 返回: (bool) 是否删除成功
func ProcessCancel(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	taskID := process.ArgsString(0)
	queue := "default"
	if process.NumOfArgs() > 1 {
		queue = process.ArgsString(1)
	}

	if inspector == nil {
		exception.New("[Asynq] Inspector is not initialized", 500).Throw()
	}

	err := inspector.DeleteTask(queue, taskID)
	if err != nil {
		exception.New("[Asynq] Cancel Task Error: %s", 500, err.Error()).Throw()
	}

	return true
}
