package triagekb

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cast"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/log"
)

// RegisterProcesses 注册 Yao Process 接口
func RegisterProcesses() {
	process.Register("triage.engine.load", ProcessLoad)
	process.Register("triage.engine.evaluate", ProcessEvaluate)
	process.Register("triage.engine.search", ProcessSearch)
	process.Register("triage.engine.status", ProcessStatus)
	process.Register("triage.engine.services", ProcessServices)
	log.Info("✅ Triage Knowledge Graph Engine processes registered")
}

// ProcessLoad 注册 Process: triage.engine.load
// 参数: [dslFilePath string (可选)]
func ProcessLoad(proc *process.Process) interface{} {
	dslPath := "kbs/triage.kb.yao"
	if proc.NumOfArgs() > 0 && proc.ArgsString(0) != "" {
		dslPath = proc.ArgsString(0)
	}

	// 尝试解析应用绝对路径
	realPath := dslPath
	if !filepath.IsAbs(realPath) {
		appRoot := application.App.Root()
		if appRoot != "" {
			realPath = filepath.Join(appRoot, dslPath)
		}
	}

	if _, err := os.Stat(realPath); os.IsNotExist(err) {
		// 备用当前工作目录
		cwd, _ := os.Getwd()
		altPath := filepath.Join(cwd, dslPath)
		if _, err := os.Stat(altPath); err == nil {
			realPath = altPath
		}
	}

	log.Info("[triagekb] 正在加载联邦知识图谱: %s", realPath)
	snapshot, err := GetEngine().LoadFromDSLFile(realPath)
	if err != nil {
		exception.New(fmt.Sprintf("加载知识图谱失败: %v", err), 500).Throw()
	}

	return map[string]interface{}{
		"success":         true,
		"version":         snapshot.Version,
		"layers":          snapshot.Layers,
		"symptoms_count":  len(snapshot.Symptoms),
		"diseases_count":  len(snapshot.Diseases),
		"rules_count":     len(snapshot.SafetyRules),
		"questions_count": len(snapshot.Questions),
		"services_count":  len(snapshot.Services),
		"edges_count":     len(snapshot.OutEdges),
		"tcm_syndromes":   len(snapshot.TCMSyndromes),
		"tcm_clinics":     len(snapshot.TCMClinics),
		"tcm_therapies":   len(snapshot.TCMTherapies),
	}
}

// ProcessEvaluate 注册 Process: triage.engine.evaluate
// 参数: [clinicalContext map[string]interface{}]
func ProcessEvaluate(proc *process.Process) interface{} {
	if proc.NumOfArgs() < 1 {
		exception.New("缺少必填参数 clinicalContext", 400).Throw()
	}

	rawArg := proc.Args[0]
	var ctx ClinicalContext

	// 序列化与反序列化至类型安全契约
	bytes, err := json.Marshal(rawArg)
	if err != nil {
		exception.New(fmt.Sprintf("解析入参格式错误: %v", err), 400).Throw()
	}
	if err := json.Unmarshal(bytes, &ctx); err != nil {
		exception.New(fmt.Sprintf("反序列化 ClinicalContext 失败: %v", err), 400).Throw()
	}

	// 默认兜底
	if ctx.Age <= 0 {
		ctx.Age = 30
	}
	if ctx.Gender == "" {
		ctx.Gender = "ALL"
	}

	decision, err := GetEngine().Evaluate(ctx)
	if err != nil {
		exception.New(fmt.Sprintf("导诊推演计算失败: %v", err), 500).Throw()
	}

	return decision
}

// ProcessSearch 注册 Process: triage.engine.search
// 参数: [query string, limit int (可选)]
func ProcessSearch(proc *process.Process) interface{} {
	if proc.NumOfArgs() < 1 {
		return []interface{}{}
	}

	query := proc.ArgsString(0)
	limit := 10
	if proc.NumOfArgs() > 1 {
		limit = cast.ToInt(proc.Args[1])
		if limit <= 0 {
			limit = 10
		}
	}

	nodes := GetEngine().SearchSymptoms(query, limit)
	return nodes
}

// ProcessStatus 注册 Process: triage.engine.status
func ProcessStatus(proc *process.Process) interface{} {
	snap := GetEngine().GetSnapshot()
	if snap == nil {
		return map[string]interface{}{
			"loaded": false,
		}
	}
	return map[string]interface{}{
		"loaded":          true,
		"version":         snap.Version,
		"layers":          snap.Layers,
		"symptoms_count":  len(snap.Symptoms),
		"diseases_count":  len(snap.Diseases),
		"rules_count":     len(snap.SafetyRules),
		"questions_count": len(snap.Questions),
		"services_count":  len(snap.Services),
		"tcm_syndromes":   len(snap.TCMSyndromes),
		"tcm_clinics":     len(snap.TCMClinics),
		"tcm_therapies":   len(snap.TCMTherapies),
	}
}

// ProcessServices 注册 Process: triage.engine.services
func ProcessServices(proc *process.Process) interface{} {
	snap := GetEngine().GetSnapshot()
	if snap == nil {
		return []interface{}{}
	}
	var list []*ServiceNode
	for _, s := range snap.Services {
		list = append(list, s)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].ServiceCode < list[j].ServiceCode
	})
	return list
}
