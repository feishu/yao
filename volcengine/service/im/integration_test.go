package im

import (
	"fmt"
	"testing"
	"time"
	"github.com/yaoapp/gou/process"
)

// MockProcess 模拟 process.Process 用于测试
type MockProcess struct {
	args []interface{}
}

func (m *MockProcess) Args() []interface{} {
	return m.args
}

func (m *MockProcess) ArgsMap() map[string]interface{} {
	if len(m.args) == 0 {
		return make(map[string]interface{})
	}
	if argsMap, ok := m.args[0].(map[string]interface{}); ok {
		return argsMap
	}
	return make(map[string]interface{})
}

func (m *MockProcess) NumOfArgs() int {
	return len(m.args)
}

func (m *MockProcess) NumOfArgsIs(num int) bool {
	return len(m.args) == num
}

func (m *MockProcess) ValidateArgNums(nums ...int) error {
	return nil
}

func (m *MockProcess) ID() string {
	return "test-process"
}

func (m *MockProcess) Sid() string {
	return "test-sid"
}

func (m *MockProcess) Global() map[string]interface{} {
	return make(map[string]interface{})
}

func (m *MockProcess) WithGlobal(global map[string]interface{}) *process.Process {
	return nil
}

func (m *MockProcess) WithSid(sid string) *process.Process {
	return nil
}

// TestProcessMethodsReturnConvertedData 测试所有 process 方法是否正确转换长整型
func TestProcessMethodsReturnConvertedData(t *testing.T) {
	// 由于实际的 process 方法需要真实的 API 调用，我们这里主要测试转换逻辑
	// 创建一个包含 int64 值的模拟响应数据
	mockResponse := map[string]interface{}{
		"user_id":         int64(1234567890123456789),
		"conversation_id": int64(9223372036854775806),
		"message_id":      int64(5555555555555555555),
		"timestamp":       int64(1640995200000),
		"name":           "Test User",
		"count":          42,
		"messages": []interface{}{
			map[string]interface{}{
				"id":      int64(1111111111111111111),
				"content": "Hello",
			},
			map[string]interface{}{
				"id":      int64(2222222222222222222),
				"content": "World",
			},
		},
	}

	// 测试转换函数
	converted := convertInt64ToStringRecursive(mockResponse)
	convertedMap, ok := converted.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected converted result to be map[string]interface{}, got %T", converted)
	}

	// 验证 int64 值已被转换为字符串
	expectedStringFields := []string{"user_id", "conversation_id", "message_id", "timestamp"}
	for _, field := range expectedStringFields {
		value := convertedMap[field]
		if _, ok := value.(string); !ok {
			t.Errorf("Expected field %s to be string, got %T with value %v", field, value, value)
		}
	}

	// 验证非 int64 值保持不变
	if convertedMap["name"] != "Test User" {
		t.Errorf("Expected name to remain 'Test User', got %v", convertedMap["name"])
	}
	if convertedMap["count"] != 42 {
		t.Errorf("Expected count to remain 42, got %v", convertedMap["count"])
	}

	// 验证嵌套数组中的 int64 值也被转换
	messages, ok := convertedMap["messages"].([]interface{})
	if !ok {
		t.Fatalf("Expected messages to be []interface{}, got %T", convertedMap["messages"])
	}

	for i, msg := range messages {
		msgMap, ok := msg.(map[string]interface{})
		if !ok {
			t.Errorf("Expected message %d to be map[string]interface{}, got %T", i, msg)
			continue
		}
		
		if _, ok := msgMap["id"].(string); !ok {
			t.Errorf("Expected message %d id to be string, got %T with value %v", i, msgMap["id"], msgMap["id"])
		}
	}
}

// TestConversionPreservesDataStructure 测试转换过程是否保持数据结构完整性
func TestConversionPreservesDataStructure(t *testing.T) {
	originalData := map[string]interface{}{
		"simple_int64":    int64(1234567890123456789),
		"simple_string":   "test",
		"simple_int":      42,
		"simple_bool":     true,
		"simple_float":    3.14,
		"nil_value":       nil,
		"nested_map": map[string]interface{}{
			"inner_int64": int64(9223372036854775805),
			"inner_string": "inner",
		},
		"array_mixed": []interface{}{
			int64(1111111111111111111),
			"string_item",
			123,
			map[string]interface{}{
				"array_item_int64": int64(2222222222222222222),
			},
		},
	}

	converted := convertInt64ToStringRecursive(originalData)
	convertedMap, ok := converted.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected converted result to be map[string]interface{}, got %T", converted)
	}

	// 验证数据结构完整性
	if len(convertedMap) != len(originalData) {
		t.Errorf("Expected converted map to have %d keys, got %d", len(originalData), len(convertedMap))
	}

	// 验证各种数据类型
	if _, ok := convertedMap["simple_int64"].(string); !ok {
		t.Error("simple_int64 should be converted to string")
	}
	if convertedMap["simple_string"] != "test" {
		t.Error("simple_string should remain unchanged")
	}
	if convertedMap["simple_int"] != 42 {
		t.Error("simple_int should remain unchanged")
	}
	if convertedMap["simple_bool"] != true {
		t.Error("simple_bool should remain unchanged")
	}
	if convertedMap["simple_float"] != 3.14 {
		t.Error("simple_float should remain unchanged")
	}
	if convertedMap["nil_value"] != nil {
		t.Error("nil_value should remain nil")
	}

	// 验证嵌套结构
	nestedMap, ok := convertedMap["nested_map"].(map[string]interface{})
	if !ok {
		t.Fatal("nested_map should remain as map[string]interface{}")
	}
	if _, ok := nestedMap["inner_int64"].(string); !ok {
		t.Error("inner_int64 should be converted to string")
	}
	if nestedMap["inner_string"] != "inner" {
		t.Error("inner_string should remain unchanged")
	}

	// 验证数组结构
	arrayMixed, ok := convertedMap["array_mixed"].([]interface{})
	if !ok {
		t.Fatal("array_mixed should remain as []interface{}")
	}
	if len(arrayMixed) != 4 {
		t.Errorf("array_mixed should have 4 items, got %d", len(arrayMixed))
	}
	if _, ok := arrayMixed[0].(string); !ok {
		t.Error("First array item (int64) should be converted to string")
	}
	if arrayMixed[1] != "string_item" {
		t.Error("Second array item should remain unchanged")
	}
	if arrayMixed[2] != 123 {
		t.Error("Third array item should remain unchanged")
	}
	
	arrayItemMap, ok := arrayMixed[3].(map[string]interface{})
	if !ok {
		t.Fatal("Fourth array item should be map[string]interface{}")
	}
	if _, ok := arrayItemMap["array_item_int64"].(string); !ok {
		t.Error("array_item_int64 should be converted to string")
	}
}

// TestConversionPerformance 测试转换性能是否在可接受范围内
func TestConversionPerformance(t *testing.T) {
	// 创建一个较大的测试数据集
	largeData := make(map[string]interface{})
	
	// 添加大量字段
	for i := 0; i < 100; i++ {
		largeData[fmt.Sprintf("int64_field_%d", i)] = int64(1234567890123456789 + int64(i))
		largeData[fmt.Sprintf("string_field_%d", i)] = fmt.Sprintf("test_string_%d", i)
		largeData[fmt.Sprintf("int_field_%d", i)] = i
	}

	// 添加嵌套结构
	for i := 0; i < 10; i++ {
		nestedMap := make(map[string]interface{})
		for j := 0; j < 10; j++ {
			nestedMap[fmt.Sprintf("nested_int64_%d", j)] = int64(9223372036854775800 + int64(j))
		}
		largeData[fmt.Sprintf("nested_%d", i)] = nestedMap
	}

	// 测试转换时间（应该在合理范围内完成）
	start := time.Now()
	converted := convertInt64ToStringRecursive(largeData)
	duration := time.Since(start)

	if duration > time.Millisecond*100 { // 100ms 作为性能基准
		t.Errorf("Conversion took too long: %v (expected < 100ms)", duration)
	}

	// 验证转换结果
	convertedMap, ok := converted.(map[string]interface{})
	if !ok {
		t.Fatal("Conversion result should be map[string]interface{}")
	}

	if len(convertedMap) != len(largeData) {
		t.Errorf("Converted map should have same number of keys as original")
	}

	t.Logf("Conversion of %d fields completed in %v", len(largeData), duration)
}