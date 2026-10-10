package triagekb

import (
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
)

// Engine 临床导诊知识图谱内存引擎单例
type Engine struct {
	snapshot atomic.Pointer[GraphSnapshot]
	config   PipelineConfig
}

// 全局单例
var defaultEngine = &Engine{
	config: DefaultPipelineConfig(),
}

// GetEngine 获取全局引擎实例
func GetEngine() *Engine {
	return defaultEngine
}

// GetSnapshot 获取当前图快照 (Zero-Lock Read 绝对无锁读取)
func (e *Engine) GetSnapshot() *GraphSnapshot {
	return e.snapshot.Load()
}

// SetSnapshot 原子替换图快照 (Hot-Swap 原子指针置换)
func (e *Engine) SetSnapshot(s *GraphSnapshot) {
	e.snapshot.Store(s)
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

	// 提取连接库路径
	generalDB := "data/triage/kb/kb_triage_general.db"
	tcmDB := "data/triage/kb/kb_triage_tcm.db"

	for _, l := range dsl.Layers {
		if l.Type == "BASE" && l.DBFile != "" {
			generalDB = l.DBFile
		}
		if l.Type == "OVERLAY" && l.DBFile != "" {
			tcmDB = l.DBFile
		}
	}

	// 提取 runtime 配置
	if dsl.Runtime != nil {
		if sp, ok := dsl.Runtime["spreading_activation"].(map[string]interface{}); ok {
			if decay, ok := sp["decay"].(float64); ok {
				e.config.Decay = decay
			}
			if dist, ok := sp["max_distance"].(float64); ok {
				e.config.MaxDistance = int(dist)
			}
		}
		if dual, ok := dsl.Runtime["dual_track"].(map[string]interface{}); ok {
			if enTCM, ok := dual["enable_tcm"].(bool); ok {
				e.config.EnableTCM = enTCM
			}
		}
	}

	loader := NewFederatedLoader(dsl)
	snapshot, err := loader.LoadFromFiles(generalDB, tcmDB)
	if err != nil {
		return nil, err
	}

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

	pipeline := NewInferencePipeline(snap, e.config)
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
