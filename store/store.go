package store

import (
	"github.com/yaoapp/yao/asset"
	"github.com/yaoapp/yao/config"
)

// Load load store（委托至统一资产引擎）
func Load(cfg config.Config) error {
	return asset.LoadOnly(cfg, "stores")
}
