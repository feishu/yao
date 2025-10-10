package template

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

// ProcessRender 渲染模板的process方法
// 参数：
//   - args[0]: 模板代码 (string)
//   - args[1]: 模板数据 (map[string]interface{})
//
// 返回：渲染后的字符串内容
func ProcessRender(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	code := process.ArgsString(0)
	data := process.ArgsMap(1)

	result, err := RenderTemplate(code, data)
	if err != nil {
		exception.New("Template render error: %s", 400, err.Error()).Throw()
	}

	return result
}

// ProcessRenderContent 渲染模板内容的process方法
// 参数：
//   - args[0]: 模板内容 (string)
//   - args[1]: 模板数据 (map[string]interface{})
//
// 返回：渲染后的字符串内容
func ProcessRenderContent(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	content := process.ArgsString(0)
	data := process.ArgsMap(1)

	result, err := RenderTemplateContent(content, data)
	if err != nil {
		exception.New("Template render error: %s", 400, err.Error()).Throw()
	}

	return result
}

// ProcessRegister 注册模板的process方法
// 参数：
//   - args[0]: 模板代码 (string)
//   - args[1]: 模板内容 (string)
//
// 返回：成功消息
func ProcessRegister(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	code := process.ArgsString(0)
	content := process.ArgsString(1)

	err := RegisterTemplate(code, content)
	if err != nil {
		exception.New("Template register error: %s", 400, err.Error()).Throw()
	}

	return map[string]interface{}{
		"success": true,
		"message": "Template registered successfully",
		"code":    code,
	}
}

// ProcessGet 获取模板内容的process方法
// 参数：
//   - args[0]: 模板代码 (string)
//
// 返回：模板内容或错误信息
func ProcessGet(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	code := process.ArgsString(0)

	content, exists := GetTemplate(code)
	if !exists {
		exception.New("Template not found: %s", 404, code).Throw()
	}

	return map[string]interface{}{
		"code":    code,
		"content": content,
	}
}

// ProcessList 列出所有模板的process方法
// 返回：所有模板的列表
func ProcessList(process *process.Process) interface{} {
	templates := ListTemplates()

	result := make([]map[string]interface{}, 0, len(templates))
	for code, content := range templates {
		result = append(result, map[string]interface{}{
			"code":    code,
			"content": content,
		})
	}

	return map[string]interface{}{
		"templates": result,
		"count":     len(templates),
	}
}

// ProcessRemove 移除模板的process方法
// 参数：
//   - args[0]: 模板代码 (string)
//
// 返回：操作结果
func ProcessRemove(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	code := process.ArgsString(0)

	removed := RemoveTemplate(code)
	if !removed {
		exception.New("Template not found: %s", 404, code).Throw()
	}

	return map[string]interface{}{
		"success": true,
		"message": "Template removed successfully",
		"code":    code,
	}
}

// ProcessClear 清除模板缓存的process方法
// 返回：操作结果
func ProcessClear(process *process.Process) interface{} {
	ClearCache()

	return map[string]interface{}{
		"success": true,
		"message": "Template cache cleared successfully",
	}
}

// ProcessInit 初始化模板的process方法
func ProcessInit(process *process.Process) interface{} {
	Init()
	return map[string]interface{}{
		"success": true,
		"message": "Templates initialized successfully",
	}
}
