package asset

import (
	"fmt"
	"sync"

	"github.com/yaoapp/yao/config"
)

// DefaultEngine 默认全局资产引擎实例
var DefaultEngine = NewEngine()

// Register 注册全局资产定义
func Register(def Definition) {
	DefaultEngine.Register(def)
}

// Load 按照依赖拓扑序加载所有注册的资产
func Load(cfg config.Config) error {
	return DefaultEngine.Load(cfg)
}

// LoadOnly 仅加载指定的一组资产（自动根据依赖拓扑排序）
func LoadOnly(cfg config.Config, names ...string) error {
	return DefaultEngine.LoadOnly(cfg, names...)
}

// Unload 按照拓扑依赖反向顺序优雅卸载所有已注册资产
func Unload() error {
	return DefaultEngine.Unload()
}

// Reset 重置全局资产引擎（用于测试）
func Reset() {
	DefaultEngine.Reset()
	registerBuiltinAssets(DefaultEngine)
}

// Register 向引擎注册资产定义
func (e *Engine) Register(def Definition) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.definitions[def.Name] = def
}

// Get 获取指定资产定义
func (e *Engine) Get(name string) (Definition, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	def, has := e.definitions[name]
	return def, has
}

// Reset 清理已注册的资产定义
func (e *Engine) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.definitions = make(map[string]Definition)
}

// Load 按照拓扑分层波次并发加载资产
func (e *Engine) Load(cfg config.Config) error {
	e.mu.RLock()
	defs := make([]Definition, 0, len(e.definitions))
	for _, def := range e.definitions {
		defs = append(defs, def)
	}
	e.mu.RUnlock()

	return e.loadDefinitions(defs, cfg)
}

// LoadOnly 仅加载选定的资产及其依赖（按拓扑分层波次并发加载）
func (e *Engine) LoadOnly(cfg config.Config, names ...string) error {
	e.mu.RLock()
	targetMap := make(map[string]bool, len(names))
	for _, name := range names {
		targetMap[name] = true
	}

	selected := make([]Definition, 0, len(names))
	for name := range targetMap {
		if def, ok := e.definitions[name]; ok {
			selected = append(selected, def)
		} else {
			e.mu.RUnlock()
			return fmt.Errorf("asset type %s not registered", name)
		}
	}
	e.mu.RUnlock()

	return e.loadDefinitions(selected, cfg)
}

// Unload 按照拓扑依赖逆序优雅卸载资产
func (e *Engine) Unload() error {
	e.mu.RLock()
	defs := make([]Definition, 0, len(e.definitions))
	for _, def := range e.definitions {
		defs = append(defs, def)
	}
	e.mu.RUnlock()

	stages, err := TopologicalSortStages(defs)
	if err != nil {
		for _, def := range defs {
			if def.Unload != nil {
				_ = def.Unload()
			}
		}
		return err
	}

	var allErrors ErrorList
	// 逆序执行 stages，确保叶子节点最先释放
	for i := len(stages) - 1; i >= 0; i-- {
		stage := stages[i]
		for _, def := range stage {
			if def.Unload != nil {
				if err := def.Unload(); err != nil {
					appendAssetError(&allErrors, def, err)
				}
			}
		}
	}

	if len(allErrors) > 0 {
		return allErrors
	}
	return nil
}

// loadDefinitions 按拓扑依赖波次（Stages）分阶段并发加载资产
func (e *Engine) loadDefinitions(defs []Definition, cfg config.Config) error {
	stages, err := TopologicalSortStages(defs)
	if err != nil {
		return err
	}

	var allErrors ErrorList
	var mu sync.Mutex

	for _, stage := range stages {
		if len(stage) == 1 {
			if err := e.discoverAndLoad(stage[0], cfg); err != nil {
				mu.Lock()
				appendAssetError(&allErrors, stage[0], err)
				mu.Unlock()
			}
			continue
		}

		var wg sync.WaitGroup
		for _, def := range stage {
			wg.Add(1)
			go func(d Definition) {
				defer wg.Done()
				if err := e.discoverAndLoad(d, cfg); err != nil {
					mu.Lock()
					appendAssetError(&allErrors, d, err)
					mu.Unlock()
				}
			}(def)
		}
		wg.Wait()
	}

	if len(allErrors) > 0 {
		return allErrors
	}
	return nil
}

func appendAssetError(allErrors *ErrorList, def Definition, err error) {
	if list, ok := err.(ErrorList); ok {
		*allErrors = append(*allErrors, list...)
	} else {
		*allErrors = append(*allErrors, ErrorItem{
			Type: def.Name,
			File: def.Dir,
			Err:  err,
		})
	}
}
