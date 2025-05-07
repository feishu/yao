package coze

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
)

var Configs = map[string]OAuthConfig{}

// Load load the OAuth config
func Load(cfg config.Config) error {
	exts := []string{"*.yao", "*.json", "*.jsonc"}
	messages := []string{}
	err := application.App.Walk("conf/agents", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}

		bytes, err := application.App.Read(file)

		dsl := OAuthConfig{}

		// 解析配置文件
		err = application.Parse(file, bytes, &dsl)

		Configs[file] = dsl

		if err != nil {
			return fmt.Errorf("parse %s failed: %s", file, err.Error())
		}

		if err != nil {
			messages = append(messages, err.Error())
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
