package coze

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
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

		log.Info("coze config file name %s ", file)

		dsl := OAuthConfig{}

		// 解析配置文件
		err = application.Parse(file, bytes, &dsl)
		fileId := share.ID(root, file)
		Configs[fileId] = dsl

		log.Info("ClientID,%s, \n file id %s", dsl.ClientID, fileId)

		if err != nil {
			messages = append(messages, err.Error())
			return fmt.Errorf("parse %s failed: %s", file, err.Error())
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

func Select(id string) (OAuthConfig, error) {
	conf, has := Configs[id]
	if !has {
		return conf, fmt.Errorf("connector %s not loaded", id)
	}
	return conf, nil
}
