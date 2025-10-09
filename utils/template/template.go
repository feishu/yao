package template

import (
	"bytes"
	"fmt"
	"sync"
	"text/template"

	"github.com/yaoapp/gou/process"
)

const (
	// STORE_NAME Redis存储名称
	STORE_NAME = "template"
	// TEMPLATE_PREFIX 模板键前缀
	TEMPLATE_PREFIX = "tpl:"
)

var (
	// templateCache 模板缓存，用于存储已编译的模板
	templateCache = make(map[string]*template.Template)
	// cacheMutex 缓存互斥锁，确保并发安全
	cacheMutex sync.RWMutex
)

// RegisterTemplate 注册模板
// code: 模板代码，用于标识模板
// content: 模板内容
func RegisterTemplate(code, content string) error {
	// 使用stores process存储模板内容到Redis
	key := TEMPLATE_PREFIX + code
	p := process.New("stores."+STORE_NAME+".set", key, content)
	
	_, err := p.Exec()
	if err != nil {
		return fmt.Errorf("failed to store template '%s': %v", code, err)
	}
	
	// 清除缓存中的旧模板
	cacheMutex.Lock()
	delete(templateCache, code)
	cacheMutex.Unlock()
	
	return nil
}

// GetTemplate 获取模板内容
// code: 模板代码
func GetTemplate(code string) (string, bool) {
	// 使用stores process从Redis获取模板内容
	key := TEMPLATE_PREFIX + code
	p := process.New("stores."+STORE_NAME+".get", key)
	
	result, err := p.Exec()
	if err != nil || result == nil {
		return "", false
	}
	
	content, ok := result.(string)
	if !ok {
		return "", false
	}
	
	return content, true
}

// RenderTemplate 渲染模板
// code: 模板代码
// data: 模板变量数据
func RenderTemplate(code string, data map[string]interface{}) (string, error) {
	// 获取模板内容
	content, exists := GetTemplate(code)
	if !exists {
		return "", fmt.Errorf("template with code '%s' not found", code)
	}
	
	// 尝试从缓存获取已编译的模板
	cacheMutex.RLock()
	tmpl, cached := templateCache[code]
	cacheMutex.RUnlock()
	
	// 如果缓存中没有，则编译模板
	if !cached {
		var err error
		tmpl, err = template.New(code).Parse(content)
		if err != nil {
			return "", fmt.Errorf("failed to parse template '%s': %v", code, err)
		}
		
		// 将编译后的模板存入缓存
		cacheMutex.Lock()
		templateCache[code] = tmpl
		cacheMutex.Unlock()
	}
	
	// 渲染模板
	var buf bytes.Buffer
	err := tmpl.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template '%s': %v", code, err)
	}
	
	return buf.String(), nil
}

// ListTemplates 列出所有已注册的模板
func ListTemplates() map[string]string {
	// 使用stores process获取所有模板键
	p := process.New("stores."+STORE_NAME+".keys")
	result, err := p.Exec()
	if err != nil {
		return make(map[string]string)
	}
	
	keys, ok := result.([]interface{})
	if !ok {
		return make(map[string]string)
	}
	
	templates := make(map[string]string)
	for _, keyInterface := range keys {
		key, ok := keyInterface.(string)
		if !ok {
			continue
		}
		
		// 只处理模板键（以TEMPLATE_PREFIX开头）
		if len(key) <= len(TEMPLATE_PREFIX) || key[:len(TEMPLATE_PREFIX)] != TEMPLATE_PREFIX {
			continue
		}
		
		code := key[len(TEMPLATE_PREFIX):]
		content, exists := GetTemplate(code)
		if exists {
			templates[code] = content
		}
	}
	
	return templates
}

// RemoveTemplate 移除模板
// code: 模板代码
func RemoveTemplate(code string) bool {
	// 使用stores process删除模板
	key := TEMPLATE_PREFIX + code
	p := process.New("stores."+STORE_NAME+".del", key)
	
	_, err := p.Exec()
	if err != nil {
		return false
	}
	
	// 清除缓存
	cacheMutex.Lock()
	delete(templateCache, code)
	cacheMutex.Unlock()
	
	return true
}

// ClearCache 清除模板缓存
func ClearCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	
	templateCache = make(map[string]*template.Template)
}