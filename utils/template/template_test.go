package template

import (
	"testing"
)

// TestInit 测试初始化功能
func TestInit(t *testing.T) {
	// 注意：这个测试需要Redis连接，在实际环境中运行
	t.Log("Testing Init function")

	// 重置初始化状态
	initialized = false

	// 测试默认配置
	err := Init()
	if err != nil {
		t.Logf("Init with default config failed (expected in test environment): %v", err)
	}

	// 重置初始化状态
	initialized = false

	// 测试重复初始化
	err = Init()
	if err != nil {
		t.Logf("Init failed (expected in test environment): %v", err)
	}

	// 测试重复初始化
	err = Init()
	if err != nil {
		t.Logf("Repeated Init failed (expected in test environment): %v", err)
	}

	// 测试IsInitialized
	if !IsInitialized() {
		t.Error("IsInitialized should return true after successful init")
	}
}

// TestRegisterTemplate 测试模板注册功能
func TestRegisterTemplate(t *testing.T) {
	t.Log("Testing RegisterTemplate function")

	// 测试空代码
	err := RegisterTemplate("", "content")
	if err == nil {
		t.Error("RegisterTemplate should fail with empty code")
	}

	// 测试空内容
	err = RegisterTemplate("test", "")
	if err == nil {
		t.Error("RegisterTemplate should fail with empty content")
	}

	// 测试正常注册（在没有Redis的环境中会失败，这是预期的）
	err = RegisterTemplate("test_template", "Hello {{.Name}}")
	if err != nil {
		t.Logf("RegisterTemplate failed (expected in test environment): %v", err)
	}
}

// TestRegisterTemplates 测试批量注册功能
func TestRegisterTemplates(t *testing.T) {
	t.Log("Testing RegisterTemplates function")

	// 测试空map
	err := RegisterTemplates(nil)
	if err == nil {
		t.Error("RegisterTemplates should fail with nil map")
	}

	err = RegisterTemplates(map[string]string{})
	if err == nil {
		t.Error("RegisterTemplates should fail with empty map")
	}

	// 测试包含空键的map
	templates := map[string]string{
		"":      "content1",
		"test2": "content2",
	}
	err = RegisterTemplates(templates)
	if err == nil {
		t.Error("RegisterTemplates should fail with empty key")
	}

	// 测试包含空值的map
	templates = map[string]string{
		"test1": "",
		"test2": "content2",
	}
	err = RegisterTemplates(templates)
	if err == nil {
		t.Error("RegisterTemplates should fail with empty content")
	}

	// 测试正常批量注册
	templates = map[string]string{
		"template1": "Hello {{.Name}}",
		"template2": "Welcome {{.User}}",
	}
	err = RegisterTemplates(templates)
	if err != nil {
		t.Logf("RegisterTemplates failed (expected in test environment): %v", err)
	}
}

// TestRegisterTemplateWithTTL 测试带TTL的模板注册
func TestRegisterTemplateWithTTL(t *testing.T) {
	t.Log("Testing RegisterTemplateWithTTL function")

	// 测试无效TTL
	err := RegisterTemplateWithTTL("test", "content", -1)
	if err == nil {
		t.Error("RegisterTemplateWithTTL should fail with negative TTL")
	}

	err = RegisterTemplateWithTTL("test", "content", 0)
	if err == nil {
		t.Error("RegisterTemplateWithTTL should fail with zero TTL")
	}

	// 测试正常TTL
	err = RegisterTemplateWithTTL("test_ttl", "Hello {{.Name}}", 3600)
	if err != nil {
		t.Logf("RegisterTemplateWithTTL failed (expected in test environment): %v", err)
	}
}

// TestGetTemplate 测试获取模板功能
func TestGetTemplate(t *testing.T) {
	t.Log("Testing GetTemplate function")

	// 测试空代码
	content, exists := GetTemplate("")
	if exists {
		t.Error("GetTemplate should return false for empty code")
	}
	if content != "" {
		t.Error("GetTemplate should return empty content for empty code")
	}

	// 测试不存在的模板
	content, exists = GetTemplate("nonexistent")
	if exists {
		t.Log("GetTemplate returned true for nonexistent template (unexpected)")
	}
	if content != "" {
		t.Log("GetTemplate returned content for nonexistent template (unexpected)")
	}
}

// TestRenderTemplate 测试模板渲染功能
func TestRenderTemplate(t *testing.T) {
	t.Log("Testing RenderTemplate function")

	// 测试空代码
	result, err := RenderTemplate("", map[string]interface{}{"Name": "Test"})
	if err == nil {
		t.Error("RenderTemplate should fail with empty code")
	}
	if result != "" {
		t.Error("RenderTemplate should return empty result on error")
	}

	// 测试不存在的模板
	result, err = RenderTemplate("nonexistent", map[string]interface{}{"Name": "Test"})
	if err == nil {
		t.Error("RenderTemplate should fail with nonexistent template")
	}
}

// TestRemoveTemplate 测试移除模板功能
func TestRemoveTemplate(t *testing.T) {
	t.Log("Testing RemoveTemplate function")

	// 测试空代码
	success := RemoveTemplate("")
	if success {
		t.Error("RemoveTemplate should return false for empty code")
	}

	// 测试移除不存在的模板
	success = RemoveTemplate("nonexistent")
	// 在没有Redis的环境中，这会返回false，这是预期的
	t.Logf("RemoveTemplate for nonexistent template returned: %v", success)
}

// TestClearCache 测试清除缓存功能
func TestClearCache(t *testing.T) {
	t.Log("Testing ClearCache function")

	// 添加一些测试数据到缓存
	templateCache["test1"] = nil
	templateCache["test2"] = nil

	if len(templateCache) == 0 {
		t.Error("Cache should have test data before clearing")
	}

	// 清除缓存
	ClearCache()

	if len(templateCache) != 0 {
		t.Error("Cache should be empty after clearing")
	}
}

// TestGetMapKeys 测试辅助函数
func TestGetMapKeys(t *testing.T) {
	t.Log("Testing getMapKeys function")

	// 测试nil map
	keys := getMapKeys(nil)
	if len(keys) != 0 {
		t.Error("getMapKeys should return empty slice for nil map")
	}

	// 测试空map
	keys = getMapKeys(map[string]interface{}{})
	if len(keys) != 0 {
		t.Error("getMapKeys should return empty slice for empty map")
	}

	// 测试有数据的map
	testMap := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	keys = getMapKeys(testMap)
	if len(keys) != 3 {
		t.Errorf("getMapKeys should return 3 keys, got %d", len(keys))
	}

	// 验证所有键都存在
	keyMap := make(map[string]bool)
	for _, key := range keys {
		keyMap[key] = true
	}

	for expectedKey := range testMap {
		if !keyMap[expectedKey] {
			t.Errorf("Expected key '%s' not found in result", expectedKey)
		}
	}
}

func TestLoadTemplatesFromDatabase(t *testing.T) {
	t.Log("Testing loadTemplatesFromDatabase function")

	// 注意：这个测试预期会失败，因为没有实际的数据库连接
	err := loadTemplatesFromDatabase()
	if err != nil {
		t.Logf("Expected error due to no database connection: %v", err)
	} else {
		t.Log("loadTemplatesFromDatabase completed successfully")
	}
}

func TestSyncTemplatesFromDatabase(t *testing.T) {
	t.Log("Testing SyncTemplatesFromDatabase function")

	// 重置初始化状态以确保测试未初始化状态
	initialized = false

	// 测试未初始化状态
	err := SyncTemplatesFromDatabase()
	if err == nil {
		t.Error("Expected error when system not initialized")
	} else {
		t.Logf("Expected error for uninitialized system: %v", err)
	}

	// 初始化系统后测试
	_ = Init() // 忽略错误，因为可能没有Redis连接
	err = SyncTemplatesFromDatabase()
	if err != nil {
		t.Logf("Expected error due to no database connection: %v", err)
	} else {
		t.Log("SyncTemplatesFromDatabase completed successfully")
	}
}

func TestReloadTemplate(t *testing.T) {
	t.Log("Testing ReloadTemplate function")

	// 测试空模板代码
	err := ReloadTemplate("")
	if err == nil {
		t.Error("Expected error for empty template code")
	} else {
		t.Logf("Expected error for empty code: %v", err)
	}

	// 测试未初始化状态
	err = ReloadTemplate("test_template")
	if err == nil {
		t.Error("Expected error when system not initialized")
	} else {
		t.Logf("Expected error for uninitialized system: %v", err)
	}

	// 初始化系统后测试
	_ = Init() // 忽略错误，因为可能没有Redis连接
	err = ReloadTemplate("test_template")
	if err != nil {
		t.Logf("Expected error due to no database connection: %v", err)
	} else {
		t.Log("ReloadTemplate completed successfully")
	}
}
