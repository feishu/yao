package task

import (
	"github.com/yaoapp/gou/task"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/asset"
	"github.com/yaoapp/yao/config"
)

// Load load task（委托至统一资产引擎）
func Load(cfg config.Config) error {
	return asset.LoadOnly(cfg, "tasks")
}

// Start tasks
func Start() {
	for name, t := range task.Tasks {
		go t.Start()
		log.Info("[Task] %s start", name)
	}
}

// Stop tasks
func Stop() {
	for name, t := range task.Tasks {
		t.Stop()
		log.Info("[Task] %s stop", name)
	}
}
