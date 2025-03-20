package volcengine

import (
	"fmt"
	"path/filepath"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/volcengine/service/iam"
)

// Config 存储volcengine的配置信息
type Config struct {
	Name  string `json:"name" yaml:"name"`
	Creds struct {
		AccessKeyID     string `json:"accessKeyId" yaml:"accessKeyId"`
		AccessKeySecret string `json:"secretAccessKey" yaml:"secretAccessKey"`
	} `json:"creds" yaml:"creds"`
	IM struct {
		AppID  int    `json:"appid" yaml:"appid"`
		AppKey string `json:"appkey" yaml:"appkey"`
	} `json:"im" yaml:"im"`
	RTC struct {
		AppID  string `json:"appid" yaml:"appid"`
		AppKey string `json:"appkey" yaml:"appkey"`
	} `json:"rtc" yaml:"rtc"`
}

// VolcEngine 全局配置实例
var VolcEngine *Config

// Load 加载volcengine配置
func Load(cfg config.Config) error {

	// 初始化默认配置
	setting := Config{
		Name: "volcengine",
	}

	// 读取配置文件
	bytes, err := application.App.Read(filepath.Join("rtc", "volcengine.yml"))
	if err != nil {
		return fmt.Errorf("read volcengine.yml failed: %s", err.Error())
	}

	// 解析配置文件
	err = application.Parse("volcengine.yml", bytes, &setting)
	if err != nil {
		return fmt.Errorf("parse volcengine.yml failed: %s", err.Error())
	}

	// 设置全局配置
	VolcEngine = &setting

	iam.DefaultInstance.Client.SetAccessKey(VolcEngine.Creds.AccessKeyID)
	iam.DefaultInstance.Client.SetSecretKey(VolcEngine.Creds.AccessKeySecret)

	return nil
}
