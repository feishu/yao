package longfields

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ExtractLongFields 从请求体中提取_longFields数组，并返回清理后的body
func ExtractLongFields(body interface{}) ([]string, interface{}) {
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

// ConvertLongFieldsToStrings 将指定字段的长整数值转换为字符串
func ConvertLongFieldsToStrings(data interface{}, longFields []string) interface{} {
	return convertLongFieldsRecursiveWithPath(data, longFields, []string{}, true)
}

// ConvertStringFieldsToLongs 将指定字段的字符串值转换为长整数
func ConvertStringFieldsToLongs(data interface{}, longFields []string) interface{} {
	return convertLongFieldsRecursiveWithPath(data, longFields, []string{}, false)
}

// MarshalToJsonWithLongSupport 支持大整数处理的JSON序列化函数
func MarshalToJsonWithLongSupport(model interface{}) ([]byte, error) {
	if model == nil {
		return make([]byte, 0), nil
	}

	// 提取longFields并清理body
	longFields, cleanModel := ExtractLongFields(model)

	// 如果有longFields，转换相应字段为字符串
	if len(longFields) > 0 {
		cleanModel = ConvertLongFieldsToStrings(cleanModel, longFields)
	}

	result, err := json.Marshal(cleanModel)
	if err != nil {
		return []byte{}, fmt.Errorf("can not marshal model to json, %v", err)
	}
	return result, nil
}

// parseDotPath 解析点号路径，返回路径片段
func parseDotPath(path string) []string {
	if path == "" {
		return []string{}
	}
	return strings.Split(path, ".")
}

// buildCurrentPath 构建当前路径字符串
func buildCurrentPath(pathSegments []string) string {
	return strings.Join(pathSegments, ".")
}

// isLongIntegerFieldWithPath 检查字段是否为长整数字段，支持路径上下文
func isLongIntegerFieldWithPath(fieldName string, currentPath []string, longFields []string) bool {
	// 构建完整路径
	fullPath := buildCurrentPath(append(currentPath, fieldName))

	// 检查是否在显式longFields列表中（支持点号路径）
	for _, field := range longFields {
		if field == fieldName || field == fullPath {
			return true
		}
	}

	// 基于字段名称模式的自动识别
	fieldLower := strings.ToLower(fieldName)

	// 以_id或id结尾的字段
	if strings.HasSuffix(fieldLower, "_id") || strings.HasSuffix(fieldLower, "id") {
		return true
	}

	// 包含timestamp或time的字段
	if strings.Contains(fieldLower, "timestamp") || strings.Contains(fieldLower, "time") {
		return true
	}

	return false
}

// IsLongIntegerField 保持向后兼容的字段检查函数
func IsLongIntegerField(fieldName string, longFields []string) bool {
	return isLongIntegerFieldWithPath(fieldName, []string{}, longFields)
}

// IsLargeNumberString 检查字符串是否表示一个大数字（超过JavaScript安全整数范围）
func IsLargeNumberString(s string) bool {
	// 检查是否为纯数字字符串
	matched, _ := regexp.MatchString(`^-?\d+$`, s)
	if !matched {
		return false
	}

	// 检查长度是否超过15位（JavaScript安全整数范围）
	numStr := strings.TrimPrefix(s, "-")
	return len(numStr) > 15
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
		// 检查当前数组是否是需要转换的字段
		if len(currentPath) > 0 {
			currentFieldName := currentPath[len(currentPath)-1]
			parentPath := currentPath[:len(currentPath)-1]
			if isLongIntegerFieldWithPath(currentFieldName, parentPath, longFields) {
				// 如果数组本身是longField，转换数组中的每个元素
				result := make([]interface{}, len(v))
				for i, item := range v {
					if toLongString {
						result[i] = convertToLongString(item)
					} else {
						result[i] = convertToLongInt(item)
					}
				}
				return result
			}
		}

		// 否则递归处理数组中的每个元素
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = convertLongFieldsRecursiveWithPath(item, longFields, currentPath, toLongString)
		}
		return result

	default:
		return data
	}
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
