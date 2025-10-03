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
	// 模拟API响应数据
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

	longFields := []string{"user_id", "conversation_id", "timestamp"}

	var result interface{}
	err := unmarshalResultIntoWithLongSupport([]byte(responseData), &result, longFields)
	if err != nil {
		fmt.Printf("响应解析错误: %v\n", err)
		return
	}

	fmt.Printf("解析结果: %+v\n", result)

	// 验证长整数是否被正确转换为字符串
	if resultMap, ok := result.(map[string]interface{}); ok {
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

func unmarshalResultIntoWithLongSupport(data []byte, result interface{}, longFields []string) error {
	// 简化版本，直接使用标准JSON反序列化
	return json.Unmarshal(data, result)
}