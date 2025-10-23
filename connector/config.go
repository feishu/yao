package connector

import (
	"fmt"
	"sync"

	"github.com/yaoapp/gou/application"
)

// ConfigConnector 配置连接器
type ConfigConnector struct {
	id      string
	name    string
	label   string
	version string
	typ     string
	options map[string]interface{}
}

// ConfigConnectors 存储所有配置类型的连接器
var ConfigConnectors = make(map[string]*ConfigConnector)
var configMutex sync.RWMutex

// ConfigDSL 配置文件的DSL结构
type ConfigDSL struct {
	Name    string                 `json:"name"`
	Label   string                 `json:"label,omitempty"`
	Version string                 `json:"version,omitempty"`
	Type    string                 `json:"type"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// LoadConfig 加载配置文件
func LoadConfig(file string, id string) (*ConfigConnector, error) {
	dsl := ConfigDSL{}
	data, err := application.App.Read(file)
	if err != nil {
		return nil, err
	}

	err = application.Parse(file, data, &dsl)
	if err != nil {
		return nil, err
	}

	config := &ConfigConnector{
		id:      id,
		name:    dsl.Name,
		label:   dsl.Label,
		version: dsl.Version,
		typ:     dsl.Type,
		options: dsl.Options,
	}

	if config.options == nil {
		config.options = make(map[string]interface{})
	}

	configMutex.Lock()
	ConfigConnectors[id] = config
	configMutex.Unlock()

	return config, nil
}

// SelectConfig 获取配置连接器
func SelectConfig(id string) (*ConfigConnector, error) {
	configMutex.RLock()
	defer configMutex.RUnlock()

	config, exists := ConfigConnectors[id]
	if !exists {
		return nil, fmt.Errorf("config connector %s not found", id)
	}
	return config, nil
}

// ID 获取配置ID
func (c *ConfigConnector) ID() string {
	return c.id
}

// Name 获取配置名称
func (c *ConfigConnector) Name() string {
	return c.name
}

// Label 获取配置标签
func (c *ConfigConnector) Label() string {
	return c.label
}

// Version 获取配置版本
func (c *ConfigConnector) Version() string {
	return c.version
}

// Type 获取配置类型
func (c *ConfigConnector) Type() string {
	return c.typ
}

// Options 获取配置选项
func (c *ConfigConnector) Options() map[string]interface{} {
	return c.options
}

// Get 获取指定key的配置值
func (c *ConfigConnector) Get(key string) (interface{}, bool) {
	val, exists := c.options[key]
	return val, exists
}

// GetString 获取字符串类型的配置值
func (c *ConfigConnector) GetString(key string) (string, error) {
	val, exists := c.options[key]
	if !exists {
		return "", fmt.Errorf("key %s not found", key)
	}
	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("key %s is not a string", key)
	}
	return str, nil
}

// GetConfigOptions 获取指定ID的配置选项
func GetConfigOptions(id string) (map[string]interface{}, error) {
	config, err := SelectConfig(id)
	if err != nil {
		return nil, err
	}
	return config.Options(), nil
}

// GetConfigOption 获取指定ID和key的配置值
func GetConfigOption(id string, key string) (interface{}, error) {
	config, err := SelectConfig(id)
	if err != nil {
		return nil, err
	}
	val, exists := config.Get(key)
	if !exists {
		return nil, fmt.Errorf("key %s not found in config %s", key, id)
	}
	return val, nil
}
