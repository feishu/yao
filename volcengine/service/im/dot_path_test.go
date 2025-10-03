package im

import (
	"encoding/json"
	"testing"
)

// TestDotPathAccess 测试点号路径访问功能
func TestDotPathAccess(t *testing.T) {
	tests := []struct {
		name        string
		data        interface{}
		longFields  []string
		expected    map[string]interface{}
		description string
	}{
		{
			name: "基本点号路径访问",
			data: map[string]interface{}{
				"user": map[string]interface{}{
					"user_id": int64(1234567890123456789),
					"name":    "John",
					"profile": map[string]interface{}{
						"profile_id": int64(987654321098765432),
						"age":        25,
					},
				},
				"conversation": map[string]interface{}{
					"conversation_id": int64(1111222233334444),
					"title":           "Test Chat",
					"metadata": map[string]interface{}{
						"created_timestamp": int64(1640995200000),
						"updated_at":        "2023-01-01",
					},
				},
			},
			longFields: []string{"user.user_id", "user.profile.profile_id", "conversation.metadata.created_timestamp"},
			expected: map[string]interface{}{
				"user": map[string]interface{}{
					"user_id": "1234567890123456789",
					"name":    "John",
					"profile": map[string]interface{}{
						"profile_id": "987654321098765432",
						"age":        25,
					},
				},
				"conversation": map[string]interface{}{
					"conversation_id": "1111222233334444", // 自动识别为长整数字段
					"title":           "Test Chat",
					"metadata": map[string]interface{}{
						"created_timestamp": "1640995200000",
						"updated_at":        "2023-01-01",
					},
				},
			},
			description: "测试基本的点号路径访问，指定嵌套字段转换为字符串",
		},
		{
			name: "混合自动识别和显式路径",
			data: map[string]interface{}{
				"user": map[string]interface{}{
					"user_id": int64(1234567890123456789), // 自动识别（以_id结尾）
					"name":    "John",
					"profile": map[string]interface{}{
						"profile_id":   int64(987654321098765432), // 自动识别（以_id结尾）
						"custom_field": int64(5555666677778888),   // 显式指定
					},
				},
				"timestamp": int64(1640995200000), // 自动识别（包含timestamp）
			},
			longFields: []string{"user.profile.custom_field"},
			expected: map[string]interface{}{
				"user": map[string]interface{}{
					"user_id": "1234567890123456789",
					"name":    "John",
					"profile": map[string]interface{}{
						"profile_id":   "987654321098765432",
						"custom_field": "5555666677778888",
					},
				},
				"timestamp": "1640995200000",
			},
			description: "测试自动识别和显式路径的混合使用",
		},
		{
			name: "数组中的嵌套对象",
			data: map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{
						"user_id": 1111111111111111111,
						"profile": map[string]interface{}{
							"profile_id": 2222222222222222222,
							"settings": map[string]interface{}{
								"notification_id": 3333333333333333333,
							},
						},
					},
					map[string]interface{}{
						"user_id": 4444444444444444444,
						"profile": map[string]interface{}{
							"profile_id": 5555555555555555555,
							"settings": map[string]interface{}{
								"notification_id": 6666666666666666666,
							},
						},
					},
				},
			},
			longFields: []string{"users.profile.settings.notification_id"},
			expected: map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{
						"user_id": "1111111111111111111",
						"profile": map[string]interface{}{
							"profile_id": "2222222222222222222",
							"settings": map[string]interface{}{
								"notification_id": "3333333333333333333",
							},
						},
					},
					map[string]interface{}{
						"user_id": "4444444444444444444",
						"profile": map[string]interface{}{
							"profile_id": "5555555555555555555",
							"settings": map[string]interface{}{
								"notification_id": "6666666666666666666",
							},
						},
					},
				},
			},
			description: "测试数组中嵌套对象的点号路径访问",
		},
		{
			name: "深层嵌套路径",
			data: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"level3": map[string]interface{}{
							"level4": map[string]interface{}{
								"deep_id": 9999888877776666,
							},
						},
					},
				},
			},
			longFields: []string{"level1.level2.level3.level4.deep_id"},
			expected: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"level3": map[string]interface{}{
							"level4": map[string]interface{}{
								"deep_id": "9999888877776666",
							},
						},
					},
				},
			},
			description: "测试深层嵌套的点号路径访问",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("测试用例: %s", tt.description)
			
			// 执行转换
			result := convertLongFieldsToStrings(tt.data, tt.longFields)
			
			// 验证结果
			if !deepEqual(result, tt.expected) {
				t.Errorf("转换结果不匹配")
				t.Logf("期望结果: %+v", tt.expected)
				t.Logf("实际结果: %+v", result)
				
				// 输出JSON格式便于调试
				expectedJSON, _ := json.MarshalIndent(tt.expected, "", "  ")
				resultJSON, _ := json.MarshalIndent(result, "", "  ")
				t.Logf("期望JSON:\n%s", expectedJSON)
				t.Logf("实际JSON:\n%s", resultJSON)
			}
		})
	}
}

// TestDotPathEdgeCases 测试点号路径的边界情况
func TestDotPathEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		data        interface{}
		longFields  []string
		expected    interface{}
		description string
	}{
		{
			name: "空路径",
			data: map[string]interface{}{
				"user_id": 1234567890123456789,
			},
			longFields: []string{},
			expected: map[string]interface{}{
				"user_id": "1234567890123456789", // 自动识别
			},
			description: "测试空longFields列表时的自动识别",
		},
		{
			name: "不存在的路径",
			data: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "John",
				},
			},
			longFields: []string{"user.nonexistent.field"},
			expected: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "John",
				},
			},
			description: "测试不存在的路径不会影响结果",
		},
		{
			name: "部分匹配路径",
			data: map[string]interface{}{
				"user": map[string]interface{}{
					"profile": map[string]interface{}{
						"id": 1234567890123456789,
					},
				},
			},
			longFields: []string{"user.profile.settings.id"}, // 路径不完全匹配
			expected: map[string]interface{}{
				"user": map[string]interface{}{
					"profile": map[string]interface{}{
						"id": "1234567890123456789", // 仍然通过自动识别转换
					},
				},
			},
			description: "测试部分匹配路径时的自动识别行为",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("边界测试: %s", tt.description)
			
			result := convertLongFieldsToStrings(tt.data, tt.longFields)
			
			if !deepEqual(result, tt.expected) {
				t.Errorf("边界情况处理不正确")
				t.Logf("期望结果: %+v", tt.expected)
				t.Logf("实际结果: %+v", result)
			}
		})
	}
}

// deepEqual 深度比较两个interface{}值
func deepEqual(a, b interface{}) bool {
	aJSON, err1 := json.Marshal(a)
	bJSON, err2 := json.Marshal(b)
	
	if err1 != nil || err2 != nil {
		return false
	}
	
	return string(aJSON) == string(bJSON)
}