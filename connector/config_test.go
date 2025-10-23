package connector

import (
	"testing"
)

func TestConfigConnector(t *testing.T) {
	// 创建一个测试配置
	config := &ConfigConnector{
		id:      "test.config",
		name:    "testconfig",
		label:   "Test Config",
		version: "1.0.0",
		typ:     "custom",
		options: map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
			"key3": 123,
		},
	}

	// 测试 ID 方法
	if config.ID() != "test.config" {
		t.Errorf("Expected ID 'test.config', got '%s'", config.ID())
	}

	// 测试 Name 方法
	if config.Name() != "testconfig" {
		t.Errorf("Expected Name 'testconfig', got '%s'", config.Name())
	}

	// 测试 Get 方法
	val, exists := config.Get("key1")
	if !exists {
		t.Error("Expected key1 to exist")
	}
	if val != "value1" {
		t.Errorf("Expected value1, got '%v'", val)
	}

	// 测试 GetString 方法
	str, err := config.GetString("key1")
	if err != nil {
		t.Errorf("GetString failed: %v", err)
	}
	if str != "value1" {
		t.Errorf("Expected 'value1', got '%s'", str)
	}

	// 测试不存在的 key
	_, exists = config.Get("nonexistent")
	if exists {
		t.Error("Expected nonexistent key to not exist")
	}

	// 添加到全局存储
	configMutex.Lock()
	ConfigConnectors[config.id] = config
	configMutex.Unlock()

	// 测试 SelectConfig
	selected, err := SelectConfig("test.config")
	if err != nil {
		t.Errorf("SelectConfig failed: %v", err)
	}
	if selected.ID() != config.ID() {
		t.Errorf("Expected ID '%s', got '%s'", config.ID(), selected.ID())
	}

	// 测试选择不存在的配置
	_, err = SelectConfig("nonexistent")
	if err == nil {
		t.Error("Expected error when selecting nonexistent config")
	}
}
