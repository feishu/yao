package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"regexp"
)

// 测试数组字符串转长整型的问题
func testArrayConversion() {
	fmt.Println("=== 测试数组字符串转长整型问题 ===")
	
	// 用户提供的测试数据
	testData := map[string]interface{}{
		"ConversationShortId": []interface{}{"7556830587647361305"},
		"_longFields": []string{"ConversationShortId"},
	}

	fmt.Printf("原始数据: %+v\n", testData)

	// 提取longFields
	longFields, processedBody := extractLongFields(testData)
	fmt.Printf("提取的longFields: %v\n", longFields)
	fmt.Printf("处理后的body: %+v\n", processedBody)

	// 动态处理longFields中指定的字段
	if len(longFields) > 0 {
		fmt.Println("\n开始处理longFields中的字段:")
		for _, field := range longFields {
			fmt.Printf("处理字段: %s\n", field)
			
			if value, exists := processedBody.(map[string]interface{})[field]; exists {
				fmt.Printf("  原始值: %v (类型: %T)\n", value, value)
				
				switch v := value.(type) {
				case []interface{}:
					fmt.Printf("  检测到数组，包含 %d 个元素\n", len(v))
					convertedArray := make([]interface{}, len(v))
					for i, item := range v {
						if str, ok := item.(string); ok {
							if num, err := strconv.ParseInt(str, 10, 64); err == nil {
								convertedArray[i] = num
								fmt.Printf("    [%d]: \"%s\" -> %d (string -> int64) ✅\n", i, str, num)
							} else {
								convertedArray[i] = item
								fmt.Printf("    [%d]: %v (转换失败，保持原值) ❌\n", i, item)
							}
						} else {
							convertedArray[i] = item
							fmt.Printf("    [%d]: %v (非字符串，保持原值)\n", i, item)
						}
					}
					processedBody.(map[string]interface{})[field] = convertedArray
					fmt.Printf("  转换后的数组: %v\n", convertedArray)
				case string:
					if num, err := strconv.ParseInt(v, 10, 64); err == nil {
						processedBody.(map[string]interface{})[field] = num
						fmt.Printf("  \"%s\" -> %d (string -> int64) ✅\n", v, num)
					} else {
						fmt.Printf("  \"%s\" (转换失败，保持原值) ❌\n", v)
					}
				default:
					fmt.Printf("  不支持的类型: %T，保持原值\n", v)
				}
			} else {
				fmt.Printf("  字段 %s 不存在\n", field)
			}
		}
	}

	fmt.Printf("\n最终处理结果: %+v\n", processedBody)
}

// 测试convertStringFieldsToLongs方法
func testConvertStringFieldsToLongs() {
	fmt.Println("\n=== 测试convertStringFieldsToLongs方法 ===")
	
	// 测试用例1：基本字符串转换
	fmt.Println("\n--- 测试用例1：基本字符串转换 ---")
	testCase1 := map[string]interface{}{
		"AppId": "7556830587647361305",
		"Operator": "7556830587647361305",
		"ParticipantUserId": "7556830587647361305",
	}
	longFields1 := []string{"AppId", "Operator", "ParticipantUserId"}
	
	fmt.Printf("输入数据: %+v\n", testCase1)
	fmt.Printf("longFields: %v\n", longFields1)
	
	result1 := convertStringFieldsToLongs(testCase1, longFields1)
	fmt.Printf("转换结果: %+v\n", result1)
	
	// 测试用例2：数组字符串转换
	fmt.Println("\n--- 测试用例2：数组字符串转换 ---")
	testCase2 := map[string]interface{}{
		"ConversationShortId": []interface{}{"7556830587647361305", "9876543210123456789"},
		"AppId": "7556830587647361305",
	}
	longFields2 := []string{"ConversationShortId", "AppId"}
	
	fmt.Printf("输入数据: %+v\n", testCase2)
	fmt.Printf("longFields: %v\n", longFields2)
	
	result2 := convertStringFieldsToLongs(testCase2, longFields2)
	fmt.Printf("转换结果: %+v\n", result2)
	
	// 测试用例3：嵌套对象转换
	fmt.Println("\n--- 测试用例3：嵌套对象转换 ---")
	testCase3 := map[string]interface{}{
		"user": map[string]interface{}{
			"id": "7556830587647361305",
			"name": "test_user",
		},
		"metadata": map[string]interface{}{
			"timestamp": "1640995200000",
		},
	}
	longFields3 := []string{"user.id", "metadata.timestamp"}
	
	fmt.Printf("输入数据: %+v\n", testCase3)
	fmt.Printf("longFields: %v\n", longFields3)
	
	result3 := convertStringFieldsToLongs(testCase3, longFields3)
	fmt.Printf("转换结果: %+v\n", result3)
}

// convertStringFieldsToLongs 将指定字段从字符串转换为int64
func convertStringFieldsToLongs(data interface{}, longFields []string) interface{} {
	return convertLongFieldsRecursiveWithPath(data, longFields, []string{}, false)
}

// convertLongFieldsRecursiveWithPath 递归处理数据结构，支持路径上下文
func convertLongFieldsRecursiveWithPath(data interface{}, longFields []string, currentPath []string, toLongString bool) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			newPath := append(currentPath, key)
			
			// 检查当前字段是否需要转换
			if isLongIntegerFieldWithPath(key, currentPath, longFields) {
				if toLongString {
					result[key] = convertToLongString(value)
				} else {
					result[key] = convertToLongInt(value)
				}
			} else {
				// 递归处理嵌套结构
				result[key] = convertLongFieldsRecursiveWithPath(value, longFields, newPath, toLongString)
			}
		}
		return result
		
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = convertLongFieldsRecursiveWithPath(item, longFields, currentPath, toLongString)
		}
		return result
		
	default:
		return data
	}
}

// isLongIntegerFieldWithPath 检查字段是否为长整数字段（支持路径）
func isLongIntegerFieldWithPath(fieldName string, currentPath []string, longFields []string) bool {
	// 构建当前完整路径
	fullPath := buildCurrentPath(append(currentPath, fieldName))
	
	for _, longField := range longFields {
		// 直接匹配字段名
		if longField == fieldName {
			return true
		}
		// 匹配完整路径
		if longField == fullPath {
			return true
		}
	}
	return false
}

// buildCurrentPath 构建当前路径字符串
func buildCurrentPath(pathSegments []string) string {
	return strings.Join(pathSegments, ".")
}

// convertToLongInt 将值转换为int64
func convertToLongInt(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		if num, err := strconv.ParseInt(v, 10, 64); err == nil {
			return num
		}
		return value
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = convertToLongInt(item)
		}
		return result
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	default:
		return value
	}
}

// convertToLongString 将值转换为字符串
func convertToLongString(value interface{}) interface{} {
	switch v := value.(type) {
	case int64:
		return strconv.FormatInt(v, 10)
	case int:
		return strconv.Itoa(v)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = convertToLongString(item)
		}
		return result
	default:
		return value
	}
}

// isLargeNumberString 检查字符串是否为大数字
func isLargeNumberString(s string) bool {
	matched, _ := regexp.MatchString(`^\d{15,}$`, s)
	return matched
}

func main() {
	// 测试数组转换
	testArrayConversion()
	
	// 测试convertStringFieldsToLongs方法
	testConvertStringFieldsToLongs()
	
	// 测试序列化
	testData := map[string]interface{}{
		"AppId":               "7556830587647361305",
		"ConversationShortId": []interface{}{"7556830587647361305"},
		"Operator":            "7556830587647361305",
		"ParticipantUserId":   "7556830587647361305",
		"_longFields":         []string{"AppId", "Operator", "ParticipantUserId", "ConversationShortId"},
	}
	
	testMarshalWithLongSupport(testData)
	
	// 测试响应解析
	testResponseUnmarshaling()
	
	// 测试用户数据
	testUserData()
}

func testMarshalWithLongSupport(testData map[string]interface{}) {
	fmt.Println("\n=== 测试序列化支持 ===")
	fmt.Printf("测试数据: %+v\n", testData)
	
	jsonData, err := marshalToJsonWithLongSupport(testData)
	if err != nil {
		fmt.Printf("序列化失败: %v\n", err)
		return
	}
	
	fmt.Printf("序列化结果: %s\n", string(jsonData))
}

func testResponseUnmarshaling() {
	fmt.Println("\n=== 测试响应解析 ===")
	
	responseJSON := `{
                "AppId": "7556830587647361305",
                "Operator": "7556830587647361305", 
                "ParticipantUserId": "7556830587647361305",
                "ConversationShortId": ["7556830587647361305"],
                "_longFields": ["AppId", "Operator", "ParticipantUserId", "ConversationShortId"]
        }`
	
	fmt.Printf("响应JSON: %s\n", responseJSON)
	
	var responseData map[string]interface{}
	if err := json.Unmarshal([]byte(responseJSON), &responseData); err != nil {
		fmt.Printf("解析失败: %v\n", err)
		return
	}
	
	fmt.Printf("解析后的数据: %+v\n", responseData)
	
	longFields, processedBody := extractLongFields(responseData)
	fmt.Printf("提取的longFields: %v\n", longFields)
	
	convertedData := convertLongFieldsToInt64(processedBody.(map[string]interface{}), longFields)
	fmt.Printf("转换后的数据: %+v\n", convertedData)
}

// extractLongFields 提取_longFields并返回处理后的数据
func extractLongFields(body interface{}) ([]string, interface{}) {
	if bodyMap, ok := body.(map[string]interface{}); ok {
		if longFieldsInterface, exists := bodyMap["_longFields"]; exists {
			var longFields []string
			if longFieldsArray, ok := longFieldsInterface.([]interface{}); ok {
				for _, field := range longFieldsArray {
					if fieldStr, ok := field.(string); ok {
						longFields = append(longFields, fieldStr)
					}
				}
			}
			
			// 创建新的map，排除_longFields
			newBody := make(map[string]interface{})
			for key, value := range bodyMap {
				if key != "_longFields" {
					newBody[key] = value
				}
			}
			
			return longFields, newBody
		}
	}
	return nil, body
}

func marshalToJsonWithLongSupport(model interface{}) ([]byte, error) {
	convertedModel := convertLargeNumbersToString(model)
	return json.Marshal(convertedModel)
}

// convertLongFieldsToInt64 将指定字段转换为int64类型
func convertLongFieldsToInt64(data map[string]interface{}, longFields []string) map[string]interface{} {
	result := make(map[string]interface{})
	
	for key, value := range data {
		// 检查是否为长整数字段
		isLongField := false
		for _, field := range longFields {
			if field == key {
				isLongField = true
				break
			}
		}
		
		if isLongField {
			// 转换为int64
			if strValue, ok := value.(string); ok {
				if intValue, err := strconv.ParseInt(strValue, 10, 64); err == nil {
					result[key] = intValue
				} else {
					result[key] = value // 转换失败，保持原值
				}
			} else {
				result[key] = value // 非字符串，保持原值
			}
		} else {
			result[key] = value // 非长整数字段，保持原值
		}
	}
	
	return result
}

func testUserData() {
	fmt.Println("\n=== 测试用户数据 ===")
	userData := map[string]interface{}{
		"user_id":      "7556830587647361305",
		"username":     "test_user",
		"created_time": "1640995200000",
		"profile": map[string]interface{}{
			"avatar_id": "9876543210123456789",
			"bio":       "Hello World",
		},
		"friends": []interface{}{
			map[string]interface{}{
				"friend_id": "1234567890123456789",
				"name":      "Friend 1",
			},
			map[string]interface{}{
				"friend_id": "9876543210987654321",
				"name":      "Friend 2",
			},
		},
	}

	fmt.Printf("用户数据: %+v\n", userData)

	// 转换为JSON字符串
	jsonData, err := marshalToJsonWithLongSupport(userData)
	if err != nil {
		fmt.Printf("序列化失败: %v\n", err)
		return
	}

	fmt.Printf("JSON数据: %s\n", string(jsonData))

	// 解析回来
	var parsedData map[string]interface{}
	if err := json.Unmarshal(jsonData, &parsedData); err != nil {
		fmt.Printf("解析失败: %v\n", err)
		return
	}

	fmt.Printf("解析后的数据: %+v\n", parsedData)
}

func convertLargeNumbersToString(data interface{}) interface{} {
	return convertInt64ToStringRecursive(data)
}

// convertInt64ToStringRecursive 递归地将int64转换为字符串
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
		for i, item := range v {
			result[i] = convertInt64ToStringRecursive(item)
		}
		return result
	case int64:
		// 检查是否为大数字（超过JavaScript安全整数范围）
		if v > 9007199254740991 || v < -9007199254740991 {
			return strconv.FormatInt(v, 10)
		}
		return v
	case int:
		int64Val := int64(v)
		if int64Val > 9007199254740991 || int64Val < -9007199254740991 {
			return strconv.FormatInt(int64Val, 10)
		}
		return v
	default:
		return data
	}
}