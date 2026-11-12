package triagekb

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

// Engine 临床导诊知识图谱内存引擎单例
type Engine struct {
	snapshot atomic.Pointer[GraphSnapshot]
	mu       sync.RWMutex
	config   PipelineConfig
	appRoot  string
}

// 全局单例
var defaultEngine = &Engine{
	config: DefaultPipelineConfig(),
}

// GetEngine 获取全局引擎实例
func GetEngine() *Engine {
	return defaultEngine
}

// SetAppRoot 记录应用根目录供草稿/发布/运行态清单定位
func (e *Engine) SetAppRoot(root string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.appRoot = strings.TrimSpace(root)
}

// GetAppRoot 获取应用根目录
func (e *Engine) GetAppRoot() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.appRoot
}

// GetConfig 获取当前生效流水线配置拷贝
func (e *Engine) GetConfig() PipelineConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.config
}

// GetSnapshot 获取当前图快照 (Zero-Lock Read 绝对无锁读取)
func (e *Engine) GetSnapshot() *GraphSnapshot {
	return e.snapshot.Load()
}

// SetSnapshot 原子替换图快照 (Hot-Swap 原子指针置换)
func (e *Engine) SetSnapshot(s *GraphSnapshot) {
	e.snapshot.Store(s)
}

func inferAppRootFromDSL(dslFilePath string) string {
	absDSL, err := filepath.Abs(dslFilePath)
	if err != nil {
		return ""
	}
	clean := filepath.Clean(absDSL)
	suffix := string(filepath.Separator) + filepath.Join("data", "triage", "runtime", "triage.runtime.kb.json")
	if strings.HasSuffix(clean, suffix) {
		return strings.TrimSuffix(clean, suffix)
	}
	kbsSuffix := string(filepath.Separator) + filepath.Join("kbs", "triage.kb.yao")
	if strings.HasSuffix(clean, kbsSuffix) {
		return strings.TrimSuffix(clean, kbsSuffix)
	}
	dir := filepath.Dir(clean)
	if filepath.Base(dir) == "kbs" {
		return filepath.Dir(dir)
	}
	return dir
}

func asFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	}
	return 0, false
}

func asInt(v interface{}) (int, bool) {
	if f, ok := asFloat64(v); ok {
		return int(f), true
	}
	return 0, false
}

func buildPipelineConfigFromRuntime(runtime map[string]interface{}) PipelineConfig {
	cfg := DefaultPipelineConfig()
	if runtime == nil {
		return cfg
	}
	if v, ok := asInt(runtime["max_steps"]); ok && v > 0 {
		cfg.MaxSteps = v
	}
	if v, ok := asFloat64(runtime["confidence_threshold"]); ok && v > 0 {
		cfg.ConfidenceThreshold = v
	}
	if v, ok := runtime["allow_early_convergence"].(bool); ok {
		cfg.AllowEarlyConvergence = v
	}
	if v, ok := asFloat64(runtime["entropy_diff_min"]); ok && v > 0 {
		cfg.EntropyDiffMin = v
	}
	if v, ok := runtime["audit_persistence"].(bool); ok {
		cfg.AuditPersistence = v
	}
	if sp, ok := runtime["spreading_activation"].(map[string]interface{}); ok {
		if decay, ok := asFloat64(sp["decay"]); ok && decay >= 0 && decay < 1 {
			cfg.Decay = decay
		}
		if dist, ok := asInt(sp["max_distance"]); ok && dist >= 0 {
			cfg.MaxDistance = dist
		}
		if thresh, ok := asFloat64(sp["threshold"]); ok && thresh >= 0 {
			cfg.SpreadingThreshold = thresh
		}
	}
	if dual, ok := runtime["dual_track"].(map[string]interface{}); ok {
		if enTCM, ok := dual["enable_tcm"].(bool); ok {
			cfg.EnableTCM = enTCM
		}
		if prefSpec, ok := dual["prefer_specialist_clinic"].(bool); ok {
			cfg.PreferSpecialistClinic = prefSpec
		}
	}
	return cfg
}

// LoadFromDSLFile 读取 DSL 配置文件并加载多源联邦 SQLite
func (e *Engine) LoadFromDSLFile(dslFilePath string) (*GraphSnapshot, error) {
	content, err := os.ReadFile(dslFilePath)
	if err != nil {
		return nil, fmt.Errorf("读取导诊图谱 DSL 失败: %w", err)
	}

	var dsl FederatedGraphDSL
	if err := json.Unmarshal(content, &dsl); err != nil {
		return nil, fmt.Errorf("解析导诊图谱 DSL 格式错误: %w", err)
	}

	nextConfig := buildPipelineConfigFromRuntime(dsl.Runtime)
	appRoot := e.GetAppRoot()
	if appRoot == "" {
		appRoot = inferAppRootFromDSL(dslFilePath)
		if appRoot != "" {
			e.SetAppRoot(appRoot)
		}
	}

	loader := NewFederatedLoader(dsl)
	snapshot, err := loader.LoadFromDSL(appRoot, filepath.Dir(dslFilePath))
	if err != nil {
		return nil, err
	}
	snapshot.DSLPath = dslFilePath

	e.mu.Lock()
	e.config = nextConfig
	e.mu.Unlock()

	// 原子替换全局只读指针
	e.SetSnapshot(snapshot)
	return snapshot, nil
}

// Evaluate 无副作用纯函数式求值接口 (微秒级无锁高并发调用)
func (e *Engine) Evaluate(clinicalCtx ClinicalContext) (*TriageDecision, error) {
	snap := e.GetSnapshot()
	if snap == nil {
		return nil, fmt.Errorf("导诊知识图谱尚未初始化或未加载")
	}

	pipeline := NewInferencePipeline(snap, e.GetConfig())
	return pipeline.Execute(clinicalCtx)
}

// SearchSymptoms 高性能倒排检索候选症状概念
func (e *Engine) SearchSymptoms(query string, limit int) []*SymptomNode {
	snap := e.GetSnapshot()
	if snap == nil {
		return nil
	}
	return snap.SearchSymptoms(query, limit)
}
