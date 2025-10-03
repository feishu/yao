package im

import (
	"testing"
	"reflect"
)

// TestConvertInt64ToStringRecursive 测试长整型转换函数
func TestConvertInt64ToStringRecursive(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "int64 value",
			input:    int64(1234567890123456789),
			expected: "1234567890123456789",
		},
		{
			name:     "regular int",
			input:    123,
			expected: 123,
		},
		{
			name:     "string value",
			input:    "test string",
			expected: "test string",
		},
		{
			name: "map with int64",
			input: map[string]interface{}{
				"id":   int64(9223372036854775807), // 使用 int64 最大值
				"name": "test",
				"age":  25,
			},
			expected: map[string]interface{}{
				"id":   "9223372036854775807",
				"name": "test",
				"age":  25,
			},
		},
		{
			name: "slice with int64",
			input: []interface{}{
				int64(1111111111111111111),
				"string",
				123,
				int64(2222222222222222222),
			},
			expected: []interface{}{
				"1111111111111111111",
				"string",
				123,
				"2222222222222222222",
			},
		},
		{
			name: "nested structure",
			input: map[string]interface{}{
				"user": map[string]interface{}{
					"id":   int64(3333333333333333333),
					"name": "John",
				},
				"messages": []interface{}{
					map[string]interface{}{
						"id":      int64(4444444444444444444),
						"content": "Hello",
					},
				},
			},
			expected: map[string]interface{}{
				"user": map[string]interface{}{
					"id":   "3333333333333333333",
					"name": "John",
				},
				"messages": []interface{}{
					map[string]interface{}{
						"id":      "4444444444444444444",
						"content": "Hello",
					},
				},
			},
		},
		{
			name:     "nil value",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertInt64ToStringRecursive(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("convertInt64ToStringRecursive() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestConvertInt64ToStringRecursiveWithStruct 测试结构体转换
func TestConvertInt64ToStringRecursiveWithStruct(t *testing.T) {
	type TestStruct struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		UserID   int64  `json:"user_id"`
		Count    int    `json:"count"`
		Optional *int64 `json:"optional,omitempty"`
	}

	optionalValue := int64(5555555555555555555)
	input := TestStruct{
		ID:       int64(1234567890123456789),
		Name:     "Test User",
		UserID:   int64(9223372036854775806), // 使用有效的 int64 值
		Count:    42,
		Optional: &optionalValue,
	}

	result := convertInt64ToStringRecursive(input)
	
	// 由于结构体会被转换为 map，我们需要检查结果是否为 map
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result to be map[string]interface{}, got %T", result)
	}

	// 检查各个字段
	if resultMap["id"] != "1234567890123456789" {
		t.Errorf("Expected id to be '1234567890123456789', got %v", resultMap["id"])
	}
	if resultMap["name"] != "Test User" {
		t.Errorf("Expected name to be 'Test User', got %v", resultMap["name"])
	}
	if resultMap["user_id"] != "9223372036854775806" {
		t.Errorf("Expected user_id to be '9223372036854775806', got %v", resultMap["user_id"])
	}
	if resultMap["count"] != 42 {
		t.Errorf("Expected count to be 42, got %v", resultMap["count"])
	}
	if resultMap["optional"] != "5555555555555555555" {
		t.Errorf("Expected optional to be '5555555555555555555', got %v", resultMap["optional"])
	}
}

// BenchmarkConvertInt64ToStringRecursive 性能测试
func BenchmarkConvertInt64ToStringRecursive(b *testing.B) {
	testData := map[string]interface{}{
		"id":       int64(1234567890123456789),
		"user_id":  int64(9223372036854775805), // 使用有效的 int64 值
		"name":     "Test User",
		"count":    42,
		"messages": []interface{}{
			map[string]interface{}{
				"msg_id":  int64(1111111111111111111),
				"content": "Hello World",
			},
			map[string]interface{}{
				"msg_id":  int64(2222222222222222222),
				"content": "Another message",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		convertInt64ToStringRecursive(testData)
	}
}