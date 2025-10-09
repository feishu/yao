package template

// TemplateData 模板数据结构
type TemplateData struct {
	Code string                 `json:"code"` // 模板代码，用于标识模板
	Data map[string]interface{} `json:"data"` // 模板变量数据
}

// TemplateResult 模板渲染结果
type TemplateResult struct {
	Content string `json:"content"` // 渲染后的内容
	Error   string `json:"error"`   // 错误信息（如果有）
}

// TemplateConfig 模板配置
type TemplateConfig struct {
	Templates map[string]string `json:"templates"` // 模板代码到模板内容的映射
}