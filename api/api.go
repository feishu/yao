package api

import (
	"github.com/yaoapp/yao/asset"
	"github.com/yaoapp/yao/config"
)

// Load apis（委托至统一资产引擎）
func Load(cfg config.Config) error {
	return asset.LoadOnly(cfg, "apis")
}
