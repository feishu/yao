package schedule

import (
	"github.com/yaoapp/gou/schedule"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/asset"
	"github.com/yaoapp/yao/config"
)

// Load load schedule（委托至统一资产引擎）
func Load(cfg config.Config) error {
	return asset.LoadOnly(cfg, "schedules")
}

// Start schedules
func Start() {
	for name, sch := range schedule.Schedules {
		sch.Start()
		log.Info("[Schedule] %s start", name)
	}
}

// Stop schedules
func Stop() {
	for name, sch := range schedule.Schedules {
		sch.Stop()
		log.Info("[Schedule] %s stop", name)
	}
}
