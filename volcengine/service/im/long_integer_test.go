package im

import (
	"encoding/json"
	"fmt"
	"testing"
)

// 测试大整数处理功能
func TestLongIntegerConversion(t *testing.T) {
	fmt.Println("=== 测试大整数转换功能 ===")

	// 测试用例1: 包含_longFields的请求体
	testCase1 := map[string]interface{}{
		"user_id":         "7556830587647361305", // 大整数字符串
		"conversation_id": "9876543210123456789", // 大整数字符串
		"message":         "Hello World",
		"timestamp":       "1640995200000", // 时间戳字符串
		"_longFields":     []string{"user_id", "conversation_id", "timestamp"},
	}

	fmt.Println("测试用例1 - 包含_longFields的请求体:")
	testMarshalWithLongSupport(t, testCase1)

	// 测试用例2: 不包含_longFields的普通请求体
	testCase2 := map[string]interface{}{
		"user_id": 123456,
		"message": "Normal message",
		"active":  true,
	}

	fmt.Println("\n测试用例2 - 普通请求体:")
	testMarshalWithLongSupport(t, testCase2)

	// 测试用例3: 边界情况 - 无效数字字符串
	testCase3 := map[string]interface{}{
		"user_id":     "invalid_number",
		"valid_id":    "7556830587647361305",
		"_longFields": []string{"user_id", "valid_id"},
	}

	fmt.Println("\n测试用例3 - 包含无效数字字符串:")
	testMarshalWithLongSupport(t, testCase3)

	// 测试用例4: 嵌套对象
	testCase4 := map[string]interface{}{
		"user": map[string]interface{}{
			"id":   "7556830587647361305",
			"name": "Test User",
		},
		"conversation": map[string]interface{}{
			"id":    "9876543210123456789",
			"title": "Test Conversation",
		},
		"_longFields": []string{"user.id", "conversation.id"},
	}

	fmt.Println("\n测试用例4 - 嵌套对象:")
	testMarshalWithLongSupport(t, testCase4)

	// 测试响应解析
	fmt.Println("\n=== 测试响应解析功能 ===")
	testResponseUnmarshaling(t)
}

func testMarshalWithLongSupport(t *testing.T, testData map[string]interface{}) {
	fmt.Printf("输入数据: %+v\n", testData)

	// 提取longFields
	longFields, processedBody := extractLongFields(testData)
	fmt.Printf("提取的longFields: %v\n", longFields)
	fmt.Printf("处理后的body: %+v\n", processedBody)

	// 序列化
	result, err := marshalToJsonWithLongSupport(processedBody)
	if err != nil {
		t.Errorf("序列化错误: %v", err)
		return
	}

	fmt.Printf("序列化结果: %s\n", string(result))

	// 验证JSON是否有效
	var jsonCheck interface{}
	if err := json.Unmarshal(result, &jsonCheck); err != nil {
		t.Errorf("JSON验证失败: %v", err)
	} else {
		fmt.Println("JSON验证成功")
	}
}

func testResponseUnmarshaling(t *testing.T) {
	// 模拟API响应，包含大整数字段
	responseData := `{
		"ResponseMetadata": {
			"RequestId": "test-request-id",
			"Error": null
		},
		"Result": {
			"user_id": 7556830587647361305,
			"conversation_id": 9876543210123456789,
			"message": "Test message",
			"timestamp": 1640995200000
		}
	}`

	// 定义longFields
	longFields := []string{"Result.user_id", "Result.conversation_id", "Result.timestamp"}

	// 解析响应
	var result map[string]interface{}
	err := unmarshalResultIntoWithLongSupport([]byte(responseData), &result, longFields)
	if err != nil {
		t.Errorf("响应解析失败: %v", err)
		return
	}

	fmt.Printf("解析结果: %+v\n", result)

	// 检查Result字段
	resultData, ok := result["Result"].(map[string]interface{})
	if !ok {
		t.Errorf("Result字段类型错误")
		return
	}

	// 验证大整数字段是否保持为字符串
	userID := resultData["user_id"]
	conversationID := resultData["conversation_id"]
	timestamp := resultData["timestamp"]

	fmt.Printf("user_id类型: %T, 值: %v\n", userID, userID)
	fmt.Printf("conversation_id类型: %T, 值: %v\n", conversationID, conversationID)
	fmt.Printf("timestamp类型: %T, 值: %v\n", timestamp, timestamp)

	// 验证类型是否为字符串
	if _, ok := userID.(string); !ok {
		t.Errorf("user_id应该是字符串类型，但得到: %T", userID)
	}
	if _, ok := conversationID.(string); !ok {
		t.Errorf("conversation_id应该是字符串类型，但得到: %T", conversationID)
	}
	if _, ok := timestamp.(string); !ok {
		t.Errorf("timestamp应该是字符串类型，但得到: %T", timestamp)
	}
}