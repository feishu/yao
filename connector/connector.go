package connector

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Load load store
func Load(cfg config.Config) error {
	exts := []string{"*.yao", "*.json", "*.jsonc"}
	messages := []string{}
	err := application.App.Walk("connectors", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}
		id := share.ID(root, file)
		_, err := connector.Load(file, id)
		if err != nil {
			// 如果是 "does not support" 错误，尝试作为配置文件加载
			if strings.Contains(err.Error(), "does not support") {
				_, configErr := LoadConfig(file, id)
				if configErr != nil {
					messages = append(messages, fmt.Sprintf("failed to load as config: %s", configErr.Error()))
				}
			} else {
				messages = append(messages, err.Error())
			}
		}
		return nil
	}, exts...)

	if err != nil {
		return err
	}

	if len(messages) > 0 {
		return fmt.Errorf("%s", strings.Join(messages, ";\n"))
	}

	return nil
}

// Unload Connector
func Unload() error {
	messages := []string{}
	for id, conn := range connector.Connectors {
		err := conn.Close()
		if err != nil {
			messages = append(messages, err.Error())
		}
		delete(connector.Connectors, id)
	}

	// 清理配置连接器
	configMutex.Lock()
	ConfigConnectors = make(map[string]*ConfigConnector)
	configMutex.Unlock()

	if len(messages) > 0 {
		return fmt.Errorf("%s", strings.Join(messages, ";\n"))
	}
	return nil
}

// Close close connector
func Close() error {
	messages := []string{}
	for _, conn := range connector.Connectors {
		err := conn.Close()
		if err != nil {
			messages = append(messages, err.Error())
		}
	}

	if len(messages) > 0 {
		return fmt.Errorf("%s", strings.Join(messages, ";\n"))
	}
	return nil
}
