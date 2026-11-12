package triagekb

import (
	"encoding/json"
	"fmt"
	"os"
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
	process.Register("triage.engine.metadata", ProcessMetadata)
	process.Register("triage.nlu.parse", ProcessNLUParse)
	RegisterAdminProcesses()
	log.Info("✅ Triage Knowledge Graph Engine processes registered")
}

// ProcessNLUParse 注册 Process: triage.nlu.parse（Zero-LLM 6 级确定性患者口语解析）
func ProcessNLUParse(proc *process.Process) interface{} {
	if proc.NumOfArgs() < 1 {
		exception.New("缺少必填参数 utterance", 400).Throw()
	}
	utterance := ""
	var pCtx NLUPatientContext

	switch v := proc.Args[0].(type) {
	case string:
		utterance = v
		if proc.NumOfArgs() > 1 {
			if rawBytes, err := json.Marshal(proc.Args[1]); err == nil {
				_ = json.Unmarshal(rawBytes, &pCtx)
			}
		}
	case map[string]interface{}:
		if u, ok := v["utterance"].(string); ok {
			utterance = u
		} else if t, ok := v["text"].(string); ok {
			utterance = t
		}
		if rawCtx, ok := v["patient_context"]; ok {
			if rawBytes, err := json.Marshal(rawCtx); err == nil {
				_ = json.Unmarshal(rawBytes, &pCtx)
			}
		} else {
			if rawBytes, err := json.Marshal(v); err == nil {
				_ = json.Unmarshal(rawBytes, &pCtx)
			}
		}
	default:
		utterance = proc.ArgsString(0)
	}

	snap := GetEngine().GetSnapshot()
	if snap == nil {
		appRoot := GetEngine().GetAppRoot()
		if appRoot == "" && application.App != nil {
			appRoot = application.App.Root()
		}
		if appRoot != "" {
			if loaded, err := GetEngine().LoadFromDSLFile(ResolveEffectiveDSLPath(appRoot, "")); err == nil {
				snap = loaded
			}
		}
	}
	if snap == nil {
		exception.New("知识图谱尚未加载，无法执行确定性 NLU 解析", 500).Throw()
	}

	return AnalyzeUtterance(snap, utterance, pCtx)
}

// ProcessLoad 注册 Process: triage.engine.load
// 参数: [dslFilePath string (可选)]
func ProcessLoad(proc *process.Process) interface{} {
	requestedPath := ""
	if proc.NumOfArgs() > 0 && proc.ArgsString(0) != "" {
		requestedPath = proc.ArgsString(0)
	}

	appRoot := ""
	if application.App != nil {
		appRoot = application.App.Root()
	}
	if appRoot == "" {
		appRoot = GetEngine().GetAppRoot()
	}
	if appRoot == "" {
		cwd, _ := os.Getwd()
		appRoot = cwd
	}
	GetEngine().SetAppRoot(appRoot)

	realPath := ResolveEffectiveDSLPath(appRoot, requestedPath)

	log.Info("[triagekb] 正在加载联邦知识图谱: %s", realPath)
	snapshot, err := GetEngine().LoadFromDSLFile(realPath)
	if err != nil {
		exception.New(fmt.Sprintf("加载知识图谱失败: %v", err), 500).Throw()
	}

	return map[string]interface{}{
		"success":         true,
		"version":         snapshot.Version,
		"active_version":  snapshot.ActiveVersion,
		"manifest_hash":   snapshot.ManifestHash,
		"dsl_path":        snapshot.DSLPath,
		"runtime":         snapshot.RuntimeConfig,
		"layer_statuses":  snapshot.LayerStatuses,
		"layers":          snapshot.Layers,
		"symptoms_count":  len(snapshot.Symptoms),
		"diseases_count":  len(snapshot.Diseases),
		"rules_count":     len(snapshot.SafetyRules),
		"questions_count": len(snapshot.Questions),
		"services_count":  len(snapshot.Services),
		"edges_count":     len(snapshot.OutEdges),
		"tcm_syndromes":   len(snapshot.TCMSyndromes),
		"tcm_diseases":    len(snapshot.TCMDiseases),
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

	// 默认兜底：年龄若未填置 0（不强行按 30 岁偏向），性别严禁伪造为 MALE/ALL，统一置 UNKNOWN
	if ctx.Gender == "" || ctx.Gender == "ALL" {
		ctx.Gender = "UNKNOWN"
	} else {
		ctx.Gender = normalizeGenderInput(ctx.Gender)
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
		"active_version":  snap.ActiveVersion,
		"manifest_hash":   snap.ManifestHash,
		"dsl_path":        snap.DSLPath,
		"runtime":         snap.RuntimeConfig,
		"layer_statuses":  snap.LayerStatuses,
		"layers":          snap.Layers,
		"symptoms_count":  len(snap.Symptoms),
		"diseases_count":  len(snap.Diseases),
		"rules_count":     len(snap.SafetyRules),
		"questions_count": len(snap.Questions),
		"services_count":  len(snap.Services),
		"tcm_syndromes":   len(snap.TCMSyndromes),
		"tcm_diseases":    len(snap.TCMDiseases),
		"tcm_clinics":     len(snap.TCMClinics),
		"tcm_therapies":   len(snap.TCMTherapies),
	}
}

// ProcessMetadata 注册 Process: triage.engine.metadata
// 返回全量排序后的只读图谱实体与词典（供医院端 36 方向矩阵与 TS 运行时使用）
func ProcessMetadata(proc *process.Process) interface{} {
	snap := GetEngine().GetSnapshot()
	if snap == nil {
		return map[string]interface{}{
			"loaded": false,
		}
	}
	payload := snap.MetadataPayload()
	payload["loaded"] = true
	return payload
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
