package template

import (
	"bytes"
	"fmt"
	"sync"
	"text/template"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
)

const (
	// STORE_NAME Redis存储名称
	STORE_NAME = "cache"
	// TEMPLATE_PREFIX 模板键前缀
	TEMPLATE_PREFIX = "tpl:"
)

var (
	// templateCache 模板缓存，用于存储已编译的模板
	templateCache = make(map[string]*template.Template)
	// cacheMutex 缓存互斥锁，确保并发安全
	cacheMutex sync.RWMutex
	// initialized 标记模板系统是否已初始化
	initialized bool
	// initMutex 初始化互斥锁
	initMutex sync.Mutex
)

// Init 初始化模板系统
// 检查Redis连接并确保stores可用
func Init() error {
	initMutex.Lock()
	defer initMutex.Unlock()

	if initialized {
		log.Info("Template system already initialized")
		return nil
	}

	log.Info("Initializing template system...")

	// 从数据库加载模板数据
	err := loadTemplatesFromDatabase()
	if err != nil {
		log.Error("Template system initialization failed: database loading error - %v", err)
		return fmt.Errorf("template system initialization failed: %v", err)
	}

	initialized = true
	log.Info("Template system initialized successfully")
	return nil
}

// loadTemplatesFromDatabase 从数据库加载模板数据到Redis
func loadTemplatesFromDatabase() error {
	log.Info("Loading templates from database...")

	// 查询数据库中的模板数据
	// 使用 models.smstemplate.get 处理器查询所有启用的模板
	p := process.New("models.sys.smstemplate.Get", map[string]interface{}{
		"wheres": []map[string]interface{}{
			{
				"column": "status",
				"value":  1, // 只加载启用状态的模板
			},
		},
		"select": []string{"code", "content"},
	})

	result, err := p.Exec()
	if err != nil {
		log.Warn("Failed to query templates from database (database may not be available): %v", err)
		log.Info("Template system will continue without database templates")
		return nil // 不返回错误，允许系统在没有数据库的情况下继续运行
	}

	// 处理查询结果 - 实际返回的是 []maps.MapStrAny 格式
	templates, ok := result.([]maps.MapStrAny)
	if !ok {
		log.Warn("Invalid database query result format (expected []maps.MapStrAny), skipping database templates")
		return nil // 不返回错误，允许系统继续运行
	}

	loadedCount := 0
	errorCount := 0

	// 遍历模板数据并存储到Redis
	for _, template := range templates {

		// 获取模板字段
		code, codeOk := template["code"].(string)
		content, contentOk := template["content"].(string)

		if !codeOk || !contentOk || code == "" {
			log.Warn("Template record missing required fields (code/content), skipping")
			errorCount++
			continue
		}

		// 如果content为空，跳过该模板（符合用户要求：不存在的模板保持为空）
		if content == "" {
			log.Debug("Template '%s' has empty content, skipping", code)
			continue
		}

		// 存储到Redis（直接使用content，不需要额外的templateData结构）
		key := TEMPLATE_PREFIX + code
		p := process.New("stores."+STORE_NAME+".Set", key, content)
		_, err := p.Exec()
		if err != nil {
			log.Error("Failed to store template '%s' to Redis: %v", code, err)
			errorCount++
			continue
		}

		loadedCount++
		log.Debug("Template '%s' loaded successfully", code)
	}

	log.Info("Template loading completed: %d loaded, %d errors", loadedCount, errorCount)

	// 不再因为没有加载到模板而返回错误，这是正常情况
	// 数据库中可能确实没有启用的模板，或者模板内容为空
	if loadedCount == 0 {
		log.Info("No templates loaded from database (this is normal if no templates exist or are enabled)")
	}

	return nil
}

// SyncTemplatesFromDatabase 同步数据库中的模板数据到Redis
// 这个方法可以在模板数据发生变化时调用，重新加载所有模板
func SyncTemplatesFromDatabase() error {
	if !initialized {
		return fmt.Errorf("template system not initialized")
	}

	log.Info("Synchronizing templates from database...")

	// 清除现有的模板缓存
	ClearCache()

	// 重新从数据库加载模板
	err := loadTemplatesFromDatabase()
	if err != nil {
		log.Error("Failed to sync templates from database: %v", err)
		return fmt.Errorf("failed to sync templates from database: %v", err)
	}

	log.Info("Templates synchronized successfully")
	return nil
}

// ReloadTemplate 重新加载指定的模板
func ReloadTemplate(code string) error {
	if !initialized {
		return fmt.Errorf("template system not initialized")
	}

	if code == "" {
		return fmt.Errorf("template code cannot be empty")
	}

	log.Info("Reloading template: %s", code)

	// 查询指定模板的数据
	p := process.New("models.smstemplate.get", map[string]interface{}{
		"wheres": []map[string]interface{}{
			{
				"column": "code",
				"value":  code,
			},
			{
				"column": "status",
				"value":  1, // 只加载启用状态的模板
			},
		},
		"select": []string{"code", "content", "name", "type", "params"},
	})

	result, err := p.Exec()
	if err != nil {
		log.Error("Failed to query template '%s' from database: %v", code, err)
		return fmt.Errorf("failed to query template from database: %v", err)
	}

	templates, ok := result.([]interface{})
	if !ok {
		log.Error("Invalid database query result format for template '%s'", code)
		return fmt.Errorf("invalid database query result format")
	}

	if len(templates) == 0 {
		// 模板不存在或已禁用，从Redis中删除
		key := TEMPLATE_PREFIX + code
		p := process.New("stores."+STORE_NAME+".del", key)
		_, err := p.Exec()
		if err != nil {
			log.Warn("Failed to remove template '%s' from Redis: %v", code, err)
		}

		// 从内存缓存中删除
		cacheMutex.Lock()
		delete(templateCache, code)
		cacheMutex.Unlock()
		log.Info("Template '%s' removed (not found or disabled)", code)
		return nil
	}

	// 更新模板
	template, ok := templates[0].(map[string]interface{})
	if !ok {
		log.Error("Invalid template record format for '%s'", code)
		return fmt.Errorf("invalid template record format")
	}

	content, contentOk := template["content"].(string)
	if !contentOk || content == "" {
		log.Error("Template '%s' missing content field", code)
		return fmt.Errorf("template missing content field")
	}

	// 存储到Redis
	key := TEMPLATE_PREFIX + code
	p = process.New("stores."+STORE_NAME+".set", key, content)
	_, err = p.Exec()
	if err != nil {
		log.Error("Failed to update template '%s' in Redis: %v", code, err)
		return fmt.Errorf("failed to update template in Redis: %v", err)
	}

	// 从内存缓存中删除，强制重新编译
	cacheMutex.Lock()
	delete(templateCache, code)
	cacheMutex.Unlock()

	log.Info("Template '%s' reloaded successfully", code)
	return nil
}

// IsInitialized 检查模板系统是否已初始化
func IsInitialized() bool {
	initMutex.Lock()
	defer initMutex.Unlock()
	return initialized
}

// RegisterTemplate 注册模板
// code: 模板代码，用于标识模板
// content: 模板内容
func RegisterTemplate(code, content string) error {
	return RegisterTemplates(map[string]string{code: content})
}

// RegisterTemplates 批量注册模板
// templates: 模板映射，key为模板代码，value为模板内容
func RegisterTemplates(templates map[string]string) error {
	// 确保系统已初始化
	if !IsInitialized() {
		if err := Init(); err != nil {
			return fmt.Errorf("failed to initialize template system: %v", err)
		}
	}

	// 参数验证
	if len(templates) == 0 {
		return fmt.Errorf("no templates provided for registration")
	}

	// 验证模板代码和内容
	for code, content := range templates {
		if code == "" {
			return fmt.Errorf("template code cannot be empty")
		}
		if content == "" {
			log.Warn("Template '%s' has empty content", code)
		}
	}

	// 批量存储到Redis
	successCount := 0
	failedTemplates := make(map[string]error)

	for code, content := range templates {
		key := TEMPLATE_PREFIX + code
		p := process.New("stores."+STORE_NAME+".set", key, content)

		_, err := p.Exec()
		if err != nil {
			log.Error("Failed to register template '%s': %v", code, err)
			failedTemplates[code] = err
			continue
		}

		// 清除缓存中的旧模板
		cacheMutex.Lock()
		delete(templateCache, code)
		cacheMutex.Unlock()

		successCount++
		log.Debug("Template '%s' registered successfully", code)
	}

	// 报告结果
	if len(failedTemplates) > 0 {
		log.Error("Failed to register %d out of %d templates", len(failedTemplates), len(templates))
		if successCount == 0 {
			return fmt.Errorf("failed to register all templates: %v", failedTemplates)
		}
		return fmt.Errorf("partially failed to register templates: %d succeeded, %d failed", successCount, len(failedTemplates))
	}

	log.Info("Successfully registered %d templates", successCount)
	return nil
}

// RegisterTemplateWithTTL 注册带过期时间的模板
// code: 模板代码，用于标识模板
// content: 模板内容
// ttl: 过期时间（秒）
func RegisterTemplateWithTTL(code, content string, ttl int) error {
	// 确保系统已初始化
	if !IsInitialized() {
		if err := Init(); err != nil {
			return fmt.Errorf("failed to initialize template system: %v", err)
		}
	}

	// 参数验证
	if code == "" {
		return fmt.Errorf("template code cannot be empty")
	}
	if ttl <= 0 {
		return fmt.Errorf("TTL must be positive, got: %d", ttl)
	}

	// 使用stores process存储模板内容到Redis，带TTL
	key := TEMPLATE_PREFIX + code
	p := process.New("stores."+STORE_NAME+".set", key, content, ttl)

	_, err := p.Exec()
	if err != nil {
		log.Error("Failed to register template '%s' with TTL %d: %v", code, ttl, err)
		return fmt.Errorf("failed to store template '%s' with TTL: %v", code, err)
	}

	// 清除缓存中的旧模板
	cacheMutex.Lock()
	delete(templateCache, code)
	cacheMutex.Unlock()

	log.Info("Template '%s' registered successfully with TTL %d seconds", code, ttl)
	return nil
}

// GetTemplate 获取模板内容
// code: 模板代码
func GetTemplate(code string) (string, bool) {
	if code == "" {
		log.Warn("GetTemplate called with empty code")
		return "", false
	}

	// 使用stores process从Redis获取模板内容
	key := TEMPLATE_PREFIX + code
	p := process.New("stores."+STORE_NAME+".get", key)

	result, err := p.Exec()
	if err != nil {
		log.Error("Failed to get template '%s' from Redis: %v", code, err)
		return "", false
	}

	if result == nil {
		log.Debug("Template '%s' not found in Redis", code)
		return "", false
	}

	content, ok := result.(string)
	if !ok {
		log.Error("Template '%s' content is not a string, got type: %T", code, result)
		return "", false
	}

	log.Debug("Template '%s' retrieved successfully, content length: %d", code, len(content))
	return content, true
}

// RenderTemplate 渲染模板
// code: 模板代码
// data: 模板变量数据
func RenderTemplate(code string, data map[string]interface{}) (string, error) {
	if code == "" {
		err := fmt.Errorf("template code cannot be empty")
		log.Error("RenderTemplate failed: %v", err)
		return "", err
	}

	log.Debug("Rendering template '%s' with data keys: %v", code, getMapKeys(data))

	// 获取模板内容
	content, exists := GetTemplate(code)
	if !exists {
		err := fmt.Errorf("template with code '%s' not found", code)
		log.Error("RenderTemplate failed: %v", err)
		return "", err
	}

	// 尝试从缓存获取已编译的模板
	cacheMutex.RLock()
	tmpl, cached := templateCache[code]
	cacheMutex.RUnlock()

	// 如果缓存中没有，则编译模板
	if !cached {
		log.Debug("Template '%s' not in cache, compiling...", code)
		var err error
		tmpl, err = template.New(code).Parse(content)
		if err != nil {
			err = fmt.Errorf("failed to parse template '%s': %v", code, err)
			log.Error("RenderTemplate failed: %v", err)
			return "", err
		}

		// 将编译后的模板存入缓存
		cacheMutex.Lock()
		templateCache[code] = tmpl
		cacheMutex.Unlock()
		log.Debug("Template '%s' compiled and cached successfully", code)
	} else {
		log.Debug("Template '%s' found in cache", code)
	}

	// 渲染模板
	var buf bytes.Buffer
	err := tmpl.Execute(&buf, data)
	if err != nil {
		err = fmt.Errorf("failed to execute template '%s': %v", code, err)
		log.Error("RenderTemplate failed: %v", err)
		return "", err
	}

	result := buf.String()
	log.Debug("Template '%s' rendered successfully, output length: %d", code, len(result))
	return result, nil
}

// ListTemplates 列出所有已注册的模板
func ListTemplates() map[string]string {
	log.Debug("Listing all registered templates")

	// 使用stores process获取所有模板键
	p := process.New("stores." + STORE_NAME + ".keys")
	result, err := p.Exec()
	if err != nil {
		log.Error("Failed to get template keys from Redis: %v", err)
		return make(map[string]string)
	}

	keys, ok := result.([]interface{})
	if !ok {
		log.Error("Template keys result is not a slice, got type: %T", result)
		return make(map[string]string)
	}

	templates := make(map[string]string)
	templateCount := 0

	for _, keyInterface := range keys {
		key, ok := keyInterface.(string)
		if !ok {
			log.Warn("Template key is not a string, got type: %T", keyInterface)
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
			templateCount++
		} else {
			log.Warn("Template key '%s' exists but content could not be retrieved", key)
		}
	}

	log.Info("Listed %d templates successfully", templateCount)
	return templates
}

// RemoveTemplate 移除模板
// code: 模板代码
func RemoveTemplate(code string) bool {
	if code == "" {
		log.Warn("RemoveTemplate called with empty code")
		return false
	}

	log.Debug("Removing template '%s'", code)

	// 使用stores process删除模板
	key := TEMPLATE_PREFIX + code
	p := process.New("stores."+STORE_NAME+".del", key)

	_, err := p.Exec()
	if err != nil {
		log.Error("Failed to remove template '%s' from Redis: %v", code, err)
		return false
	}

	// 清除缓存
	cacheMutex.Lock()
	delete(templateCache, code)
	cacheMutex.Unlock()

	log.Info("Template '%s' removed successfully", code)
	return true
}

// ClearCache 清除模板缓存
func ClearCache() {
	log.Debug("Clearing template cache")

	cacheMutex.Lock()
	cacheSize := len(templateCache)
	templateCache = make(map[string]*template.Template)
	cacheMutex.Unlock()

	log.Info("Template cache cleared, removed %d cached templates", cacheSize)
}

// getMapKeys 获取map的所有键（用于日志记录）
func getMapKeys(m map[string]interface{}) []string {
	if m == nil {
		return []string{}
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
