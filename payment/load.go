// Package payment 加载和初始化
package payment

import (
	"fmt"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
)

func init() {
	// 自动注册 Process 接口
	if err := registerProcesses(); err != nil {
		log.Error("Failed to register payment processes: %v", err)
	}
}

// Load 加载支付模块
func Load() error {
	log.Info("Loading payment module...")

	// 注册 Process 接口
	if err := registerProcesses(); err != nil {
		return fmt.Errorf("failed to register payment processes: %v", err)
	}

	log.Info("Payment module loaded successfully")
	return nil
}

// registerProcesses 注册 Process 接口
func registerProcesses() error {
	processes := map[string]process.Handler{
		"utils.payment.CreatePayment":         ProcessCreatePayment,
		"utils.payment.QueryPayment":          ProcessQueryPayment,
		"utils.payment.RefundPayment":         ProcessRefundPayment,
		"utils.payment.AddConfig":             ProcessAddConfig,
		"utils.payment.GetSupportedProviders": ProcessGetSupportedProviders,
		"utils.payment.ValidateNotify":        ProcessValidateNotify,
	}

	for name, handler := range processes {
		process.Register(name, handler)
		log.Debug("Registered payment process: %s", name)
	}

	return nil
}

// Init 初始化支付模块（可选）
func Init() error {
	// 这里可以添加初始化逻辑，比如：
	// - 加载默认配置
	// - 预热连接池
	// - 验证配置有效性
	// - 设置默认限制等

	log.Info("Payment module initialized")
	return nil
}

// Cleanup 清理资源（可选）
func Cleanup() error {
	// 这里可以添加清理逻辑，比如：
	// - 关闭连接
	// - 清理缓存
	// - 保存状态等

	log.Info("Payment module cleanup completed")
	return nil
}