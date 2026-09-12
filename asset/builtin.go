package asset

import (
	"fmt"

	"github.com/yaoapp/gou/api"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/gou/flow"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/schedule"
	"github.com/yaoapp/gou/store"
	"github.com/yaoapp/gou/task"
	"github.com/yaoapp/yao/config"
)

func init() {
	registerBuiltinAssets(DefaultEngine)
}

// registerBuiltinAssets 注册 Yao 核心标准资产定义
func registerBuiltinAssets(e *Engine) {
	// 1. Connectors 连接器
	e.Register(Definition{
		Name:     "connectors",
		Dir:      "connectors",
		Exts:     []string{"*.yao", "*.json", "*.jsonc"},
		Optional: true,
		Loader: func(file string, id string) error {
			_, err := connector.Load(file, id)
			return err
		},
	})

	// 2. Models 数据模型
	e.Register(Definition{
		Name:      "models",
		Dir:       "models",
		Exts:      []string{"*.mod.yao", "*.mod.json", "*.mod.jsonc"},
		DependsOn: []string{"connectors"},
		Optional:  true,
		Preload: func(cfg config.Config) error {
			model.WithCrypt([]byte(fmt.Sprintf(`{"key":"%s"}`, cfg.DB.AESKey)), "AES")
			model.WithCrypt([]byte(`{}`), "PASSWORD")
			return nil
		},
		Loader: func(file string, id string) error {
			_, err := model.Load(file, id)
			return err
		},
	})

	// 3. Stores 存储
	e.Register(Definition{
		Name:      "stores",
		Dir:       "stores",
		Exts:      []string{"*.yao", "*.json", "*.jsonc"},
		DependsOn: []string{"connectors"},
		Optional:  true,
		Loader: func(file string, id string) error {
			_, err := store.Load(file, id)
			return err
		},
	})

	// 4. Flows 流程编排
	e.Register(Definition{
		Name:      "flows",
		Dir:       "flows",
		Exts:      []string{"*.flow.yao", "*.flow.json", "*.flow.jsonc"},
		DependsOn: []string{"models"},
		Optional:  true,
		Loader: func(file string, id string) error {
			_, err := flow.Load(file, id)
			return err
		},
	})

	// 5. Tasks 任务
	e.Register(Definition{
		Name:      "tasks",
		Dir:       "tasks",
		Exts:      []string{"*.yao", "*.json", "*.jsonc"},
		DependsOn: []string{"models"},
		Optional:  true,
		Loader: func(file string, id string) error {
			_, err := task.Load(file, id)
			return err
		},
	})

	// 6. Schedules 定时调度
	e.Register(Definition{
		Name:      "schedules",
		Dir:       "schedules",
		Exts:      []string{"*.sch.yao", "*.sch.json", "*.sch.jsonc"},
		DependsOn: []string{"models"},
		Optional:  true,
		Loader: func(file string, id string) error {
			_, err := schedule.Load(file, id)
			return err
		},
	})

	// 7. APIs HTTP 业务接口
	e.Register(Definition{
		Name:      "apis",
		Dir:       "apis",
		Exts:      []string{"*.http.yao", "*.http.json", "*.http.jsonc"},
		DependsOn: []string{"models", "flows"},
		Optional:  true,
		Loader: func(file string, id string) error {
			_, err := api.Load(file, id)
			return err
		},
	})
}
