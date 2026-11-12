package triagekb

import (
	"os"
	"path/filepath"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
)

// RuntimeManifestRelPath 运行态持久化清单路径（优先于代码仓 kbs/triage.kb.yao，免疫发版覆盖）
const RuntimeManifestRelPath = "data/triage/runtime/triage.runtime.kb.json"

// DefaultDSLRelPath 默认应用内知识库声明路径
const DefaultDSLRelPath = "kbs/triage.kb.yao"

// ResolveEffectiveDSLPath 按优先级定位实际生效的清单路径：运行态覆盖清单 > 包内默认 DSL
func ResolveEffectiveDSLPath(appRoot, requestedPath string) string {
	cleanReq := filepath.Clean(requestedPath)
	defaultClean := filepath.Clean(DefaultDSLRelPath)

	if requestedPath == "" || cleanReq == defaultClean {
		runtimeAbs := filepath.Join(appRoot, RuntimeManifestRelPath)
		if _, err := os.Stat(runtimeAbs); err == nil {
			return runtimeAbs
		}
		return filepath.Join(appRoot, DefaultDSLRelPath)
	}

	if filepath.IsAbs(requestedPath) {
		return requestedPath
	}

	candidate := filepath.Join(appRoot, requestedPath)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	cwd, _ := os.Getwd()
	if cwd != "" {
		alt := filepath.Join(cwd, requestedPath)
		if _, err := os.Stat(alt); err == nil {
			return alt
		}
	}
	return candidate
}

// Load 初始化导诊知识图谱引擎模块
func Load(cfg config.Config) error {
	// 1. 注册 Process 接口
	RegisterProcesses()

	// 2. 确认应用根目录并设置给全局引擎
	appRoot := application.App.Root()
	if appRoot == "" {
		cwd, _ := os.Getwd()
		appRoot = cwd
	}
	GetEngine().SetAppRoot(appRoot)

	// 3. 优先加载持久化运行态清单，若不存在则回退包内 kbs/triage.kb.yao
	dslPath := ResolveEffectiveDSLPath(appRoot, "")
	if _, err := os.Stat(dslPath); err == nil {
		log.Info("🔍 [triagekb] 检测到知识库清单: %s，开始加载内存图谱...", dslPath)
		snapshot, err := GetEngine().LoadFromDSLFile(dslPath)
		if err != nil {
			log.Warn("⚠️ [triagekb] 自动加载知识图谱失败: %v (可在业务中通过 Process 手动加载)", err)
		} else {
			log.Info("🚀 [triagekb] 临床内存知识图谱初始化成功! 版本=%s 主诉=%d 疾病=%d 规则=%d 中医证候=%d",
				snapshot.ActiveVersion, len(snapshot.Symptoms), len(snapshot.Diseases), len(snapshot.SafetyRules), len(snapshot.TCMSyndromes))
		}
	} else {
		log.Debug("[triagekb] 未在 %s 发现导诊知识库声明，跳过自动加载", dslPath)
	}

	return nil
}
