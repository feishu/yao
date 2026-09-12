package asset

import (
	"fmt"
	"strings"
	"sync"

	"github.com/yaoapp/yao/config"
)

// Definition 资产定义规范
type Definition struct {
	Name      string                             // 资产类型名称，如 "models", "apis", "flows"
	Dir       string                             // 资产存放相对目录
	Exts      []string                           // 文件匹配后缀
	Loader    func(file string, id string) error // 单个资产加载器
	Preload   func(cfg config.Config) error      // 资产批次加载前钩子（可选）
	Postload  func(cfg config.Config) error      // 资产批次加载后钩子（可选）
	DependsOn []string                           // 依赖的其他资产类型名称（DAG 排序用）
	Optional  bool                               // 若目录不存在或为空是否忽略（默认 true）
}

// ErrorItem 结构化资产加载错误
type ErrorItem struct {
	Type string // 资产类型，如 "models"
	File string // 资产相对/绝对路径
	ID   string // 资产 ID
	Err  error  // 错误本体
}

// Error 实现 error 接口
func (e ErrorItem) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("[%s:%s] %s: %v", e.Type, e.ID, e.File, e.Err)
	}
	return fmt.Sprintf("[%s] %s: %v", e.Type, e.File, e.Err)
}

// ErrorList 聚合错误列表
type ErrorList []ErrorItem

// Error 实现 error 接口
func (list ErrorList) Error() string {
	if len(list) == 0 {
		return ""
	}
	msgs := make([]string, 0, len(list))
	for _, item := range list {
		msgs = append(msgs, item.Error())
	}
	return strings.Join(msgs, ";\n")
}

// Engine 资产引擎
type Engine struct {
	mu          sync.RWMutex
	definitions map[string]Definition
}

// NewEngine 创建新资产引擎实例
func NewEngine() *Engine {
	return &Engine{
		definitions: make(map[string]Definition),
	}
}
