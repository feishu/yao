package asset

import (
	"fmt"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// discoverAndLoad 扫描并加载单类资产
func (e *Engine) discoverAndLoad(def Definition, cfg config.Config) error {
	var errs ErrorList

	// 1. 执行批次 Preload 钩子（如加密 key 设置或依赖初始化）
	if def.Preload != nil {
		if err := def.Preload(cfg); err != nil {
			errs = append(errs, ErrorItem{
				Type: def.Name,
				File: def.Dir,
				Err:  fmt.Errorf("preload hook failed: %w", err),
			})
			return errs
		}
	}

	// 2. 检查 application.App 运行环境
	if application.App == nil {
		if def.Optional {
			return nil
		}
		errs = append(errs, ErrorItem{
			Type: def.Name,
			File: def.Dir,
			Err:  fmt.Errorf("application.App is not initialized"),
		})
		return errs
	}

	// 3. 安全遍历资产目录并逐项装载
	walkErr := application.App.Walk(def.Dir, func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		if def.Loader == nil {
			return nil
		}

		id := share.ID(root, file)
		if err := def.Loader(file, id); err != nil {
			errs = append(errs, ErrorItem{
				Type: def.Name,
				File: file,
				ID:   id,
				Err:  err,
			})
		}
		return nil
	}, def.Exts...)

	if walkErr != nil && !def.Optional {
		errs = append(errs, ErrorItem{
			Type: def.Name,
			File: def.Dir,
			Err:  fmt.Errorf("walk directory failed: %w", walkErr),
		})
	}

	// 4. 执行批次 Postload 钩子
	if def.Postload != nil {
		if err := def.Postload(cfg); err != nil {
			errs = append(errs, ErrorItem{
				Type: def.Name,
				File: def.Dir,
				Err:  fmt.Errorf("postload hook failed: %w", err),
			})
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}
