package triagekb

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// GoldenRegressionPath 黄金回归病例默认相对路径
const GoldenRegressionPath = "data/triage/tests/regression_cases.json"

// SandboxRunRequest E3 右侧实时患者沙盒请求体
type SandboxRunRequest struct {
	Utterance       string            `json:"utterance,omitempty"`
	SeedConcepts    []string          `json:"seed_concepts,omitempty"`
	Answers         map[string]string `json:"answers,omitempty"`
	KnownSlots      map[string]string `json:"known_slots,omitempty"`
	ConfirmedSigns  []string          `json:"confirmed_signs,omitempty"`
	UncertainStreak int               `json:"uncertain_streak,omitempty"`
	UseActive       bool              `json:"use_active,omitempty"`
	PatientContext  struct {
		Age            int      `json:"age,omitempty"`
		Gender         string   `json:"gender,omitempty"`
		IsPregnant     *bool    `json:"is_pregnant,omitempty"`
		MedicalHistory []string `json:"medical_history,omitempty"`
	} `json:"patient_context"`
}

// GoldenCaseRecord regression_cases.json 条目映射
type GoldenCaseRecord struct {
	CaseID                 string            `json:"case_id"`
	Category               string            `json:"category"`
	Title                  string            `json:"title"`
	SeedConcepts           []string          `json:"seed_concepts"`
	ConfirmedSigns         []string          `json:"confirmed_signs,omitempty"`
	Answers                map[string]string `json:"answers,omitempty"`
	ExpectedSafetyState    string            `json:"expected_safety_state,omitempty"`
	ExpectedIsShortCircuit *bool             `json:"expected_is_short_circuit,omitempty"`
	ExpectedRuleID         string            `json:"expected_rule_id,omitempty"`
	ExpectedServiceCode    string            `json:"expected_service_code,omitempty"`
	PatientContext         struct {
		Age            int      `json:"age"`
		Gender         string   `json:"gender"`
		IsPregnant     *bool    `json:"is_pregnant,omitempty"`
		MedicalHistory []string `json:"medical_history,omitempty"`
	} `json:"patient_context"`
}

// SkippedQuestionTrace 追踪因 KnownSlots 或同槽已答而自动跳过的题目
type SkippedQuestionTrace struct {
	QuestionID string `json:"question_id"`
	SlotKey    string `json:"slot_key"`
	Title      string `json:"title"`
	KnownValue string `json:"known_value"`
	SkipReason string `json:"skip_reason"`
}

var (
	stagingCacheMu      sync.RWMutex
	cachedStagingKey    string
	cachedStagingSnap   *GraphSnapshot
)

func computeFileStamp(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "missing"
	}
	return fmt.Sprintf("%d:%d", info.ModTime().UnixNano(), info.Size())
}

func loadStagingSnapshot(appRoot string) (*GraphSnapshot, PipelineConfig, int, error) {
	adminMu.Lock()
	genDraft, tcmDraft, err := ensureDraftInitializedLocked(appRoot, false)
	if err != nil {
		adminMu.Unlock()
		return nil, DefaultPipelineConfig(), 0, err
	}
	_ = ensureDraftSchemaAndSeeds(appRoot, genDraft, tcmDraft)
	adminMu.Unlock()

	var draftRev int
	if genDB, err := openSQLiteRO(genDraft); err == nil {
		_ = genDB.QueryRow(`SELECT COALESCE(draft_revision, 1) FROM playbook_draft_meta WHERE id = 1`).Scan(&draftRev)
		_ = genDB.Close()
	}
	if draftRev <= 0 {
		draftRev = 1
	}

	dsl, _, dslErr := loadEffectiveDSL(appRoot)
	cfg := DefaultPipelineConfig()
	if dslErr == nil && len(dsl.Runtime) > 0 {
		cfg = buildPipelineConfigFromRuntime(dsl.Runtime)
	}

	cacheKey := fmt.Sprintf("%s|%s|%s|%d", appRoot, computeFileStamp(genDraft), computeFileStamp(tcmDraft), draftRev)
	stagingCacheMu.RLock()
	if cachedStagingSnap != nil && cachedStagingKey == cacheKey {
		snap := cachedStagingSnap
		stagingCacheMu.RUnlock()
		return snap, cfg, draftRev, nil
	}
	stagingCacheMu.RUnlock()

	loader := NewFederatedLoader(FederatedGraphDSL{
		Version:       dsl.Version,
		ActiveVersion: fmt.Sprintf("staging-v%d", draftRev),
		Runtime:       dsl.Runtime,
	})
	snap, err := loader.LoadFromFiles(genDraft, tcmDraft)
	if err != nil {
		return nil, cfg, draftRev, err
	}

	stagingCacheMu.Lock()
	cachedStagingKey = cacheKey
	cachedStagingSnap = snap
	stagingCacheMu.Unlock()

	return snap, cfg, draftRev, nil
}

// RunSandbox 在当前 Staging 草稿（或对比 Active 快照）上运行 Zero-LLM E1 NLU + 多跳图谱推理
func RunSandbox(appRoot string, req SandboxRunRequest) (map[string]interface{}, error) {
	startTime := time.Now()

	stagingSnap, cfg, draftRev, err := loadStagingSnapshot(appRoot)
	if err != nil {
		return nil, err
	}

	targetSnap := stagingSnap
	sourceLabel := "staging"
	if req.UseActive {
		if active := GetEngine().GetSnapshot(); active != nil {
			targetSnap = active
			sourceLabel = "active"
		}
	}

	if req.UseActive {
		cfg = GetEngine().GetConfig()
	}

	// 1. 构建基础患者上下文（未指定性别严格维持 UNKNOWN）
	baseGender := normalizeGenderInput(req.PatientContext.Gender)
	var nluRes *NLUParseResult
	if strings.TrimSpace(req.Utterance) != "" {
		parsed := AnalyzeUtterance(targetSnap, req.Utterance, NLUPatientContext{
			Age:        req.PatientContext.Age,
			Gender:     baseGender,
			IsPregnant: req.PatientContext.IsPregnant,
		})
		nluRes = &parsed
	}

	// 2. 合并种子主诉、伴随体征、已知槽位、性别/年龄与预设答案
	seedConcepts := append([]string{}, req.SeedConcepts...)
	confirmedSigns := append([]string{}, req.ConfirmedSigns...)
	knownSlots := make(map[string]string)
	for k, v := range req.KnownSlots {
		knownSlots[k] = v
	}
	mergedAnswers := make(map[string]string)
	for k, v := range req.Answers {
		mergedAnswers[k] = v
	}

	age := req.PatientContext.Age
	gender := baseGender
	isPregnant := req.PatientContext.IsPregnant

	if nluRes != nil {
		seedConcepts = uniqueNonEmptyStrings(append(seedConcepts, nluRes.SeedConcepts...))
		confirmedSigns = uniqueNonEmptyStrings(append(confirmedSigns, nluRes.ConfirmedSigns...))
		for k, v := range nluRes.KnownSlots {
			if _, exists := knownSlots[k]; !exists {
				knownSlots[k] = v
			}
		}
		for qID, ansVal := range nluRes.PreConfirmedAnswers {
			if _, exists := mergedAnswers[qID]; !exists {
				mergedAnswers[qID] = ansVal
			}
		}
		if age == 0 && nluRes.ExtractedAge != nil {
			age = *nluRes.ExtractedAge
		}
		if gender == "UNKNOWN" && nluRes.ExtractedGender != "" {
			gender = nluRes.ExtractedGender
		}
		if isPregnant == nil && nluRes.ExtractedPregnant != nil {
			isPregnant = nluRes.ExtractedPregnant
		}
	}

	clinicalCtx := ClinicalContext{
		Age:             age,
		Gender:          gender,
		IsPregnant:      isPregnant,
		MedicalHistory:  req.PatientContext.MedicalHistory,
		Symptoms:        seedConcepts,
		ConfirmedSigns:  confirmedSigns,
		Answers:         mergedAnswers,
		KnownSlots:      knownSlots,
		UncertainStreak: req.UncertainStreak,
		RawUtterance:    req.Utterance,
	}

	// 3. 收集命中的剧本卡题目中被 KnownSlots 自动跳过的轨迹，便于工作区 B 所见即所得调试
	skipped := traceSkippedQuestions(targetSnap, clinicalCtx)

	pipeline := NewInferencePipeline(targetSnap, cfg)
	decision, err := pipeline.Execute(clinicalCtx)
	if err != nil {
		return nil, err
	}

	// 生成 Top3 标准服务摘要供 Yao API 层接续工作区 A 院区映射落地检查
	topServices := make([]map[string]interface{}, 0, 3)
	for idx, svc := range decision.RankedServices {
		if idx >= 3 {
			break
		}
		topServices = append(topServices, map[string]interface{}{
			"service_code": svc.ServiceCode,
			"service_name": svc.ServiceName,
			"score":        svc.Score,
			"description":  svc.Description,
		})
	}

	return map[string]interface{}{
		"source":            sourceLabel,
		"draft_revision":    draftRev,
		"elapsed_ms":        float64(time.Since(startTime).Microseconds()) / 1000.0,
		"nlu_result":        nluRes,
		"clinical_context":  clinicalCtx,
		"skipped_questions": skipped,
		"decision":          decision,
		"top_services":      topServices,
	}, nil
}

func traceSkippedQuestions(snap *GraphSnapshot, ctx ClinicalContext) []SkippedQuestionTrace {
	if snap == nil || len(ctx.Symptoms) == 0 || len(ctx.KnownSlots) == 0 {
		return []SkippedQuestionTrace{}
	}
	symSet := make(map[string]bool, len(ctx.Symptoms))
	for _, s := range ctx.Symptoms {
		symSet[s] = true
	}

	var traces []SkippedQuestionTrace
	seenQ := make(map[string]bool)
	for _, q := range snap.Questions {
		if q == nil || seenQ[q.QuestionID] {
			continue
		}
		matchedSym := false
		for _, appSym := range q.ApplicableSymptoms {
			if symSet[appSym] {
				matchedSym = true
				break
			}
		}
		if !matchedSym {
			continue
		}
		slot := q.SlotKey
		if slot == "" {
			slot = inferLegacySlotKey(q.QuestionID)
		}
		knownVal, hasKnown := ctx.KnownSlots[slot]
		if !hasKnown {
			continue
		}
		// 保护：未直连 PRE_CONFIRMED 的 CRITICAL 题目严禁仅靠模糊槽位跳过
		if q.Category == "RED_FLAG" && slot != "has_fever" && slot != "onset_acute" {
			if _, preAnswered := ctx.Answers[q.QuestionID]; !preAnswered {
				continue
			}
		}
		seenQ[q.QuestionID] = true
		traces = append(traces, SkippedQuestionTrace{
			QuestionID: q.QuestionID,
			SlotKey:    slot,
			Title:      q.Title,
			KnownValue: knownVal,
			SkipReason: fmt.Sprintf("E1 NLU 已明确抽取槽位 %s=%s，智能跳过该常规追问", slot, knownVal),
		})
	}
	sort.Slice(traces, func(i, j int) bool {
		return traces[i].QuestionID < traces[j].QuestionID
	})
	return traces
}

// RunGoldenDiff 对比当前 Active 快照 vs Staging 草稿全量回放 regression_cases.json 黄金集，仅返回变化条目与安全告警
func RunGoldenDiff(appRoot string) (map[string]interface{}, error) {
	startTime := time.Now()
	cases, err := loadGoldenRegressionCases(appRoot)
	if err != nil {
		return nil, err
	}

	stagingSnap, cfg, draftRev, err := loadStagingSnapshot(appRoot)
	if err != nil {
		return nil, err
	}

	activeSnap := GetEngine().GetSnapshot()
	activeMissing := false
	if activeSnap == nil {
		activeMissing = true
		activeSnap = stagingSnap
	}

	diffs := make([]map[string]interface{}, 0)
	safetyRegressions := 0
	allCriticalPassed := true

	activePipe := NewInferencePipeline(activeSnap, cfg)
	stagingPipe := NewInferencePipeline(stagingSnap, cfg)

	for _, gc := range cases {
		gender := gc.PatientContext.Gender
		if gender == "" {
			gender = "UNKNOWN"
		}
		ctx := ClinicalContext{
			Age:            gc.PatientContext.Age,
			Gender:         gender,
			IsPregnant:     gc.PatientContext.IsPregnant,
			MedicalHistory: gc.PatientContext.MedicalHistory,
			Symptoms:       gc.SeedConcepts,
			ConfirmedSigns: gc.ConfirmedSigns,
			Answers:        gc.Answers,
		}

		actDec, actErr := activePipe.Execute(ctx)
		stgDec, stgErr := stagingPipe.Execute(ctx)
		if actErr != nil || stgErr != nil {
			safetyRegressions++
			allCriticalPassed = false
			diffs = append(diffs, map[string]interface{}{
				"case_id":              gc.CaseID,
				"title":                gc.Title,
				"category":             gc.Category,
				"is_safety_regression": true,
				"regression_reason":    fmt.Sprintf("引擎执行报错 active=%v staging=%v", actErr, stgErr),
			})
			continue
		}

		actRule := ""
		if actDec.EmergencyAlert != nil {
			actRule = actDec.EmergencyAlert.RuleID
		}
		stgRule := ""
		if stgDec.EmergencyAlert != nil {
			stgRule = stgDec.EmergencyAlert.RuleID
		}
		actPrimary := ""
		if len(actDec.RankedServices) > 0 {
			actPrimary = actDec.RankedServices[0].ServiceCode
		}
		stgPrimary := ""
		if len(stgDec.RankedServices) > 0 {
			stgPrimary = stgDec.RankedServices[0].ServiceCode
		}

		actShort := actDec.State == "EMERGENCY" || actDec.SafetyState == "EMERGENCY_NOW"
		stgShort := stgDec.State == "EMERGENCY" || stgDec.SafetyState == "EMERGENCY_NOW"

		regReason := ""
		isSafetyRegression := false
		if gc.ExpectedSafetyState != "" && stgDec.SafetyState != gc.ExpectedSafetyState {
			if gc.ExpectedSafetyState == "EMERGENCY_NOW" || gc.Category == "EMERGENCY" {
				isSafetyRegression = true
				allCriticalPassed = false
			}
			regReason = fmt.Sprintf("安全状态预期 %s，草稿输出 %s", gc.ExpectedSafetyState, stgDec.SafetyState)
		} else if gc.ExpectedIsShortCircuit != nil && stgShort != *gc.ExpectedIsShortCircuit {
			if *gc.ExpectedIsShortCircuit {
				isSafetyRegression = true
				allCriticalPassed = false
			}
			regReason = fmt.Sprintf("短路标志预期 %v，草稿输出 %v", *gc.ExpectedIsShortCircuit, stgShort)
		} else if gc.ExpectedRuleID != "" && stgRule != gc.ExpectedRuleID {
			isSafetyRegression = true
			allCriticalPassed = false
			regReason = fmt.Sprintf("预期命中红旗规则 %s，草稿命中 %s", gc.ExpectedRuleID, stgRule)
		} else if gc.ExpectedServiceCode != "" && stgPrimary != gc.ExpectedServiceCode && gc.Category != "EMERGENCY" {
			regReason = fmt.Sprintf("首选科室预期 %s，草稿输出 %s", gc.ExpectedServiceCode, stgPrimary)
		}

		if isSafetyRegression {
			safetyRegressions++
		}

		changed := !activeMissing && (actDec.SafetyState != stgDec.SafetyState ||
			actShort != stgShort ||
			actRule != stgRule ||
			actPrimary != stgPrimary)

		if changed || isSafetyRegression || regReason != "" {
			diffs = append(diffs, map[string]interface{}{
				"case_id":                 gc.CaseID,
				"title":                   gc.Title,
				"category":                gc.Category,
				"active_safety_state":     actDec.SafetyState,
				"staging_safety_state":    stgDec.SafetyState,
				"active_short_circuit":    actShort,
				"staging_short_circuit":   stgShort,
				"active_rule_id":          actRule,
				"staging_rule_id":         stgRule,
				"active_primary_service":  actPrimary,
				"staging_primary_service": stgPrimary,
				"is_safety_regression":    isSafetyRegression,
				"regression_reason":       regReason,
			})
		}
	}

	elapsedMs := float64(time.Since(startTime).Microseconds()) / 1000.0
	summary := map[string]interface{}{
		"total_cases":         len(cases),
		"changed_count":       len(diffs),
		"safety_regressions":  safetyRegressions,
		"all_critical_passed": allCriticalPassed,
		"draft_revision":      draftRev,
		"elapsed_ms":          elapsedMs,
	}

	// 将最新 Diff 摘要落盘到 playbook_draft_meta
	genDraft, _ := resolveDraftDBPaths(appRoot)
	if genDB, err := openSQLiteRW(genDraft); err == nil {
		if sumBytes, mErr := json.Marshal(summary); mErr == nil {
			_, _ = genDB.Exec(`UPDATE playbook_draft_meta SET last_golden_diff_summary_json = ? WHERE id = 1`, string(sumBytes))
		}
		_ = genDB.Close()
	}

	return map[string]interface{}{
		"total_cases":         len(cases),
		"changed_count":       len(diffs),
		"safety_regressions":  safetyRegressions,
		"all_critical_passed": allCriticalPassed,
		"draft_revision":      draftRev,
		"elapsed_ms":          elapsedMs,
		"diffs":               diffs,
		"summary":             summary,
	}, nil
}

func loadGoldenRegressionCases(appRoot string) ([]GoldenCaseRecord, error) {
	candidates := []string{
		filepath.Join(appRoot, GoldenRegressionPath),
		"/Users/L/Desktop/Code/yao_projects/syd/service/" + GoldenRegressionPath,
	}
	var raw []byte
	var readErr error
	for _, p := range candidates {
		if b, err := os.ReadFile(p); err == nil {
			raw = b
			readErr = nil
			break
		} else {
			readErr = err
		}
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("无法读取黄金回归用例库 %s: %v", GoldenRegressionPath, readErr)
	}
	var cases []GoldenCaseRecord
	if err := json.Unmarshal(raw, &cases); err != nil {
		return nil, fmt.Errorf("解析黄金回归用例库失败: %w", err)
	}
	return cases, nil
}
