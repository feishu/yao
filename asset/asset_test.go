package asset

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
)

func TestTopologicalSort(t *testing.T) {
	defs := []Definition{
		{Name: "apis", DependsOn: []string{"models", "flows"}},
		{Name: "flows", DependsOn: []string{"models"}},
		{Name: "models", DependsOn: []string{"connectors"}},
		{Name: "connectors"},
		{Name: "standalone"},
	}

	sorted, err := TopologicalSort(defs)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(sorted))

	indexMap := make(map[string]int)
	for i, def := range sorted {
		indexMap[def.Name] = i
	}

	// 验证依赖次序
	assert.True(t, indexMap["connectors"] < indexMap["models"], "connectors 必须排在 models 前面")
	assert.True(t, indexMap["models"] < indexMap["flows"], "models 必须排在 flows 前面")
	assert.True(t, indexMap["models"] < indexMap["apis"], "models 必须排在 apis 前面")
	assert.True(t, indexMap["flows"] < indexMap["apis"], "flows 必须排在 apis 前面")
}

func TestTopologicalSortCircularDependency(t *testing.T) {
	defs := []Definition{
		{Name: "a", DependsOn: []string{"b"}},
		{Name: "b", DependsOn: []string{"c"}},
		{Name: "c", DependsOn: []string{"a"}},
	}

	_, err := TopologicalSort(defs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular asset dependency detected")
}

func TestErrorListAggregation(t *testing.T) {
	errs := ErrorList{
		ErrorItem{Type: "models", File: "user.mod.yao", ID: "user", Err: errors.New("syntax error")},
		ErrorItem{Type: "apis", File: "login.http.yao", ID: "login", Err: errors.New("missing path")},
	}

	assert.Equal(t, 2, len(errs))
	msg := errs.Error()
	assert.Contains(t, msg, "[models:user]")
	assert.Contains(t, msg, "syntax error")
	assert.Contains(t, msg, "[apis:login]")
	assert.Contains(t, msg, "missing path")
}

func TestEngineRegisterAndGet(t *testing.T) {
	engine := NewEngine()
	engine.Register(Definition{
		Name: "custom",
		Dir:  "custom_dir",
	})

	def, ok := engine.Get("custom")
	assert.True(t, ok)
	assert.Equal(t, "custom_dir", def.Dir)

	_, ok = engine.Get("non_existent")
	assert.False(t, ok)
}

func TestBuiltinAssetsRegistration(t *testing.T) {
	expected := []string{"connectors", "models", "stores", "flows", "tasks", "schedules", "apis"}
	for _, name := range expected {
		_, ok := DefaultEngine.Get(name)
		assert.True(t, ok, "内置资产 %s 应该已注册", name)
	}
}

func TestLoadOnlyUnregisteredAsset(t *testing.T) {
	engine := NewEngine()
	err := engine.LoadOnly(config.Config{}, "unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "asset type unknown not registered")
}

func TestTopologicalSortStages(t *testing.T) {
	defs := []Definition{
		{Name: "apis", DependsOn: []string{"models", "flows"}},
		{Name: "flows", DependsOn: []string{"models"}},
		{Name: "tasks", DependsOn: []string{"models"}},
		{Name: "stores", DependsOn: []string{"connectors"}},
		{Name: "models", DependsOn: []string{"connectors"}},
		{Name: "connectors"},
		{Name: "standalone"},
	}

	stages, err := TopologicalSortStages(defs)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(stages), "应该被合理划分为 4 个并发波次")

	// 验证各 Stage 的成员
	getNames := func(stage []Definition) []string {
		res := make([]string, len(stage))
		for i, d := range stage {
			res[i] = d.Name
		}
		return res
	}

	assert.Equal(t, []string{"connectors", "standalone"}, getNames(stages[0]))
	assert.Equal(t, []string{"models", "stores"}, getNames(stages[1]))
	assert.Equal(t, []string{"flows", "tasks"}, getNames(stages[2]))
	assert.Equal(t, []string{"apis"}, getNames(stages[3]))
}

func TestConcurrentStageLoading(t *testing.T) {
	engine := NewEngine()
	loaded := make(map[string]bool)
	var mu sync.Mutex

	record := func(name string) {
		mu.Lock()
		defer mu.Unlock()
		loaded[name] = true
	}

	engine.Register(Definition{
		Name:     "c1",
		Optional: true,
		Preload: func(cfg config.Config) error {
			record("c1")
			return nil
		},
	})
	engine.Register(Definition{
		Name:     "c2",
		Optional: true,
		Preload: func(cfg config.Config) error {
			record("c2")
			return nil
		},
	})
	engine.Register(Definition{
		Name:      "c3",
		DependsOn: []string{"c1", "c2"},
		Optional:  true,
		Preload: func(cfg config.Config) error {
			record("c3")
			return nil
		},
	})

	err := engine.Load(config.Config{})
	assert.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, loaded["c1"])
	assert.True(t, loaded["c2"])
	assert.True(t, loaded["c3"])
}

func TestEngineUnloadReverseStages(t *testing.T) {
	engine := NewEngine()
	unloadedOrder := []string{}
	var mu sync.Mutex

	record := func(name string) {
		mu.Lock()
		defer mu.Unlock()
		unloadedOrder = append(unloadedOrder, name)
	}

	engine.Register(Definition{
		Name: "connectors",
		Unload: func() error {
			record("connectors")
			return nil
		},
	})
	engine.Register(Definition{
		Name:      "models",
		DependsOn: []string{"connectors"},
		Unload: func() error {
			record("models")
			return nil
		},
	})
	engine.Register(Definition{
		Name:      "apis",
		DependsOn: []string{"models"},
		Unload: func() error {
			record("apis")
			return nil
		},
	})

	err := engine.Unload()
	assert.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"apis", "models", "connectors"}, unloadedOrder, "Unload 应严格逆序释放叶子到根")
}

