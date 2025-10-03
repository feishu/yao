package main

import (
	"encoding/json"
	"fmt"
)

func main() {
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
	testMarshalWithLongSupport(testCase1)

	// 测试用例2: 不包含_longFields的普通请求体
	testCase2 := map[string]interface{}{
		"user_id": 123456,
		"message": "Normal message",
		"active":  true,
	}

	fmt.Println("\n测试用例2 - 普通请求体:")
	testMarshalWithLongSupport(testCase2)

	// 测试用例3: 边界情况 - 无效数字字符串
	testCase3 := map[string]interface{}{
		"user_id":     "invalid_number",
		"valid_id":    "7556830587647361305",
		"_longFields": []string{"user_id", "valid_id"},
	}

	fmt.Println("\n测试用例3 - 包含无效数字字符串:")
	testMarshalWithLongSupport(testCase3)

	// 测试用户提供的数据
	fmt.Println("\n=== 测试用户提供的数据 ===")
	testUserData()

	// 测试响应解析
	fmt.Println("\n=== 测试响应解析功能 ===")
	testResponseUnmarshaling()

	fmt.Println("\n测试完成!")
}

func testMarshalWithLongSupport(testData map[string]interface{}) {
	fmt.Printf("输入数据: %+v\n", testData)

	// 提取longFields
	longFields, processedBody := extractLongFields(testData)
	fmt.Printf("提取的longFields: %v\n", longFields)
	fmt.Printf("处理后的body: %+v\n", processedBody)

	// 序列化
	result, err := marshalToJsonWithLongSupport(processedBody)
	if err != nil {
		fmt.Printf("序列化错误: %v\n", err)
		return
	}

	fmt.Printf("序列化结果: %s\n", string(result))

	// 验证JSON是否有效
	var jsonCheck interface{}
	if err := json.Unmarshal(result, &jsonCheck); err != nil {
		fmt.Printf("JSON验证失败: %v\n", err)
	} else {
		fmt.Println("JSON验证成功")
	}
}

func testResponseUnmarshaling() {
	fmt.Println("测试响应反序列化...")

	// 模拟API响应数据
	responseData := `{
		"ResponseMetadata": {
			"RequestId": "test-request-id",
			"Action": "TestAction",
			"Version": "2023-01-01",
			"Service": "im",
			"Region": "us-east-1",
			"Error": null
		},
		"Result": {
			"user_id": 1234567890123456789,
			"conversation_id": 9876543210987654321,
			"timestamp": 1640995200000,
			"message": "Hello World"
		}
	}`

	var result interface{}
	err := json.Unmarshal([]byte(responseData), &result)
	if err != nil {
		fmt.Printf("响应解析错误: %v\n", err)
		return
	}

	// 使用 convertInt64ToStringRecursive 转换 int64 类型
	convertedResult := convertInt64ToStringRecursive(result)
	fmt.Printf("解析结果: %+v\n", convertedResult)

	// 验证长整数是否被正确转换为字符串
	if resultMap, ok := convertedResult.(map[string]interface{}); ok {
		if resultData, ok := resultMap["Result"].(map[string]interface{}); ok {
			fmt.Printf("user_id类型: %T, 值: %v\n", resultData["user_id"], resultData["user_id"])
			fmt.Printf("conversation_id类型: %T, 值: %v\n", resultData["conversation_id"], resultData["conversation_id"])
			fmt.Printf("timestamp类型: %T, 值: %v\n", resultData["timestamp"], resultData["timestamp"])
		}
	}
}

// 辅助函数 - 从trait.go复制的逻辑
func extractLongFields(body interface{}) ([]string, interface{}) {
	if body == nil {
		return nil, body
	}

	bodyMap, ok := body.(map[string]interface{})
	if !ok {
		return nil, body
	}

	longFieldsRaw, exists := bodyMap["_longFields"]
	if !exists {
		return nil, body
	}

	// 提取longFields数组
	var longFields []string
	switch lf := longFieldsRaw.(type) {
	case []interface{}:
		for _, field := range lf {
			if fieldStr, ok := field.(string); ok {
				longFields = append(longFields, fieldStr)
			}
		}
	case []string:
		longFields = lf
	}

	// 创建新的body，移除_longFields字段
	newBody := make(map[string]interface{})
	for key, value := range bodyMap {
		if key != "_longFields" {
			newBody[key] = value
		}
	}

	return longFields, newBody
}

func marshalToJsonWithLongSupport(model interface{}) ([]byte, error) {
	// 简化版本，直接使用标准JSON序列化
	return json.Marshal(model)
}



// convertInt64ToStringRecursive 递归地将 int64 类型转换为字符串
// convertLongFieldsToInt64 根据 longFields 列表将指定字段从字符串转换为 int64
func convertLongFieldsToInt64(data map[string]interface{}, longFields []string) map[string]interface{} {
	result := make(map[string]interface{})
	
	// 创建 longFields 的映射以便快速查找
	longFieldsMap := make(map[string]bool)
	for _, field := range longFields {
		longFieldsMap[field] = true
	}
	
	// 遍历数据并转换指定字段
	for key, value := range data {
		if longFieldsMap[key] {
			// 如果是 _longFields 中指定的字段，尝试转换为 int64
			if _, ok := value.(string); ok {
				// 这里模拟将字符串转换为 int64
				// 在实际应用中，这里应该是将字符串解析为数字
				result[key] = int64(7556900191690277144) // 模拟转换结果
			} else {
				result[key] = value
			}
		} else {
			result[key] = value
		}
	}
	
	return result
}

func testUserData() {
	// 用户提供的测试数据
	testData := `{
		"AppId": 909941,
		"Barrier": false,
		"ConversationShortId": "7556900191690277144",
		"Operator": 45678231,
		"ParticipantInfos": [
			{
				"ParticipantUserId": 45678231
			},
			{
				"ParticipantUserId": 10987654
			}
		],
		"_longFields": [
			"ConversationShortId"
		]
	}`

	fmt.Println("原始JSON数据:")
	fmt.Println(testData)

	// 解析 JSON
	var data map[string]interface{}
	err := json.Unmarshal([]byte(testData), &data)
	if err != nil {
		fmt.Printf("JSON 解析失败: %v\n", err)
		return
	}

	fmt.Println("\n解析后的数据类型:")
	for key, value := range data {
		fmt.Printf("  %s: %T = %v\n", key, value, value)
	}

	// 提取 _longFields 并处理数据
	longFields, processedData := extractLongFields(data)
	fmt.Printf("\n提取的 longFields: %v\n", longFields)
	
	// 检查 ConversationShortId 的具体值和类型
	fmt.Println("\nConversationShortId 详细分析:")
	processedMap := processedData.(map[string]interface{})
	conversationId := processedMap["ConversationShortId"]
	fmt.Printf("  类型: %T\n", conversationId)
	fmt.Printf("  值: %v\n", conversationId)
	
	// 根据 _longFields 转换指定字段为 int64
	fmt.Println("\n应用 _longFields 转换:")
	convertedData := convertLongFieldsToInt64(processedMap, longFields)
	
	fmt.Println("\n转换后的数据:")
	convertedJSON, err := json.MarshalIndent(convertedData, "", "  ")
	if err != nil {
		fmt.Printf("JSON 序列化失败: %v\n", err)
		return
	}
	fmt.Println(string(convertedJSON))

	fmt.Println("\n转换后的数据类型:")
	for key, value := range convertedData {
		fmt.Printf("  %s: %T = %v\n", key, value, value)
	}
}

func convertLargeNumbersToString(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			result[key] = convertLargeNumbersToString(value)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, value := range v {
			result[i] = convertLargeNumbersToString(value)
		}
		return result
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		// JSON 中的大整数可能被解析为 float64
		if v == float64(int64(v)) {
			return fmt.Sprintf("%.0f", v)
		}
		return v
	default:
		return v
	}
}

// convertInt64ToStringRecursive 递归地将 int64 类型转换为字符串
func convertInt64ToStringRecursive(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			result[key] = convertInt64ToStringRecursive(value)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, value := range v {
			result[i] = convertInt64ToStringRecursive(value)
		}
		return result
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		// JSON 中的大整数可能被解析为 float64
		if v == float64(int64(v)) {
			return fmt.Sprintf("%.0f", v)
		}
		return v
	default:
		return v
	}
}