package connector

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/connector"
	"github.com/yaoapp/yao/asset"
	"github.com/yaoapp/yao/config"
)

func init() {
	asset.Register(asset.Definition{
		Name:     "connectors",
		Dir:      "connectors",
		Exts:     []string{"*.yao", "*.json", "*.jsonc"},
		Optional: true,
		Loader: func(file string, id string) error {
			_, err := connector.Load(file, id)
			if err != nil {
				if strings.Contains(err.Error(), "does not support") {
					_, configErr := LoadConfig(file, id)
					if configErr != nil {
						return fmt.Errorf("failed to load as config: %w", configErr)
					}
					return nil
				}
				return err
			}
			return nil
		},
	})
}

// Load load connectors（委托至统一资产引擎）
func Load(cfg config.Config) error {
	return asset.LoadOnly(cfg, "connectors")
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
