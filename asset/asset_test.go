package asset

import (
	"errors"
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
