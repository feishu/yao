package triagekb

import (
	"os"
	"path/filepath"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
)

// Load 初始化导诊知识图谱引擎模块
func Load(cfg config.Config) error {
	// 1. 注册 Process 接口
	RegisterProcesses()

	// 2. 检查当前应用根目录下是否存在 kbs/triage.kb.yao
	appRoot := application.App.Root()
	if appRoot == "" {
		cwd, _ := os.Getwd()
		appRoot = cwd
	}

	dslPath := filepath.Join(appRoot, "kbs", "triage.kb.yao")
	if _, err := os.Stat(dslPath); err == nil {
		log.Info("🔍 [triagekb] 检测到应用知识库声明: %s，开始加载内存图谱...", dslPath)
		snapshot, err := GetEngine().LoadFromDSLFile(dslPath)
		if err != nil {
			log.Warn("⚠️ [triagekb] 自动加载知识图谱失败: %v (可在业务中通过 Process 手动加载)", err)
		} else {
			log.Info("🚀 [triagekb] 临床内存知识图谱初始化成功! 包含 %d 主诉, %d 疾病, %d 规则, %d 中医证候",
				len(snapshot.Symptoms), len(snapshot.Diseases), len(snapshot.SafetyRules), len(snapshot.TCMSyndromes))
		}
	} else {
		log.Debug("[triagekb] 未在 %s 发现导诊知识库声明，跳过自动加载", dslPath)
	}

	return nil
}
