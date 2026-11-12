package triagekb

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// PipelineConfig 推演流水线超参数
type PipelineConfig struct {
	Decay                  float64 // 扩散衰减系数 (默认 0.2)
	MaxDistance            int     // 最大扩散步数 (默认 2)
	SpreadingThreshold     float64 // 扩散活化最小得分阈值 (默认 0.35)
	MaxSteps               int     // 最大临床追问步数上限 (默认 5)
	ConfidenceThreshold    float64 // 收敛置信度阈值 (默认 0.85)
	AllowEarlyConvergence  bool    // 是否允许红旗清零后提前收敛 (默认 true)
	EntropyDiffMin         float64 // 区分度阈值 (默认 0.25)
	EnableTCM              bool    // 是否启用中医双轨 (默认 true)
	PreferSpecialistClinic bool    // 是否优先推荐中医专病门诊 (默认 true)
	AuditPersistence       bool    // 是否开启审计持久化 (默认 true)
}

// DefaultPipelineConfig 默认配置
func DefaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		Decay:                  0.2,
		MaxDistance:            2,
		SpreadingThreshold:     0.35,
		MaxSteps:               5,
		ConfidenceThreshold:    0.85,
		AllowEarlyConvergence:  true,
		EntropyDiffMin:         0.25,
		EnableTCM:              true,
		PreferSpecialistClinic: true,
		AuditPersistence:       true,
	}
}

// InferencePipeline 纯函数式临床推演流水线
type InferencePipeline struct {
	snapshot *GraphSnapshot
	config   PipelineConfig
}

// NewInferencePipeline 创建推演流水线
func NewInferencePipeline(snapshot *GraphSnapshot, cfg PipelineConfig) *InferencePipeline {
	return &InferencePipeline{
		snapshot: snapshot,
		config:   cfg,
	}
}

// Execute 执行临床问诊求值
func (p *InferencePipeline) Execute(clinicalCtx ClinicalContext) (*TriageDecision, error) {
	answeredLabels := p.resolveAnsweredLabels(clinicalCtx)
	kbVer := p.snapshot.ActiveVersion
	if kbVer == "" {
		kbVer = p.snapshot.Version
	}
	decision := &TriageDecision{
		State:                  "NORMAL",
		SafetyState:            "ROUTINE_PATHWAY",
		CanEarlyTerminate:      false,
		RankedServices:         make([]RankedService, 0),
		SuspectedDiseases:      make([]ScoredDisease, 0),
		EvidenceChains:         make([]EvidenceChain, 0),
		KBVersion:              kbVer,
		KBHash:                 p.snapshot.ManifestHash,
		PreferSpecialistClinic: p.config.PreferSpecialistClinic,
		AuditPersistence:       p.config.AuditPersistence,
	}
	if len(answeredLabels) > 0 {
		decision.AnsweredLabels = answeredLabels
	}

	// 1. 实体对齐算子 (Entity Linking)
	activeSymptoms := p.linkEntities(clinicalCtx.Symptoms)
	if len(activeSymptoms) == 0 {
		return decision, nil
	}

	// 2. 红旗急危重症安全评估 (CRITICAL 短路 + URGENT 升级 + 选项/不确定兜底)
	safetyEval := p.evaluateSafety(activeSymptoms, clinicalCtx)
	decision.SafetyState = safetyEval.SafetyState
	decision.EmergencyAlert = safetyEval.Alert
	if safetyEval.SafetyState == "EMERGENCY_NOW" && safetyEval.Alert != nil {
		decision.State = "EMERGENCY"
		decision.ConvergenceReason = "EMERGENCY_SHORT_CIRCUIT"
		decision.CanEarlyTerminate = false
		// 装配急诊科默认服务
		decision.RankedServices = append(decision.RankedServices, RankedService{
			ServiceCode:          "EMERGENCY_TRIAGE_FAST",
			ServiceName:          safetyEval.Alert.RecommendedDeptName,
			Score:                1.0,
			IsPrimary:            true,
			TrackType:            "WESTERN_GENERAL",
			Description:          safetyEval.Alert.WarningText,
			PrimaryDeptKeywords:  []string{safetyEval.Alert.RecommendedDeptName, "急诊科", "急诊医学科"},
			FallbackDeptKeywords: []string{"全科医疗科"},
		})
		// 组装循证证据
		p.assembleEvidence(decision, activeSymptoms, nil)
		return decision, nil
	}

	// 3. 受激扩散加权算子 (Spreading Activation)
	scoredDiseases := p.spreadingActivation(activeSymptoms, clinicalCtx)
	decision.SuspectedDiseases = scoredDiseases

	// 4. 路由与推荐现代医学科室服务 (Western Medical Service Routing，融合已答选项倾向)
	rankedServices := p.routeWesternServices(scoredDiseases, activeSymptoms, clinicalCtx)
	decision.RankedServices = rankedServices

	// 5. 中医双轨辨证与特色专病门诊决策 (TCM Dual-Track Decision，接收四诊选项与 2 跳证候聚合)
	if p.config.EnableTCM {
		tcmRec := p.inferTCMTrack(activeSymptoms, scoredDiseases, clinicalCtx)
		decision.TCM = tcmRec

		// 仅当开启 prefer_specialist_clinic 时将中医特色专病门诊推入临床服务候选排序
		if p.config.PreferSpecialistClinic {
			for _, clinic := range tcmRec.Clinics {
				priKeywords := clinic.TargetDeptKeywords
				if len(priKeywords) == 0 && clinic.DeptName != "" {
					priKeywords = []string{clinic.DeptName}
				}
				decision.RankedServices = append(decision.RankedServices, RankedService{
					ServiceCode:          clinic.ClinicCode,
					ServiceName:          clinic.DeptName,
					Score:                clinic.Score,
					IsPrimary:            false,
					TrackType:            "TCM_SPECIALIST",
					Description:          clinic.ClinicName + " (特色专病门诊)",
					PrimaryDeptKeywords:  priKeywords,
					FallbackDeptKeywords: []string{"中医科"},
					AssociatedClinicCode: clinic.ClinicCode,
					AssociatedClinicName: clinic.ClinicName,
					BookingURL:           clinic.BookingURL,
				})
			}
		}
	}

	// 6. 信息熵歧义与红旗分层追问算子 (Tiered Safety + Service Convergence + slot_key 去重)
	nextInquiry, isConverged, redFlagsCleared, convReason := p.selectNextInquiry(scoredDiseases, rankedServices, activeSymptoms, clinicalCtx, safetyEval)
	if isConverged {
		decision.State = "CONVERGED"
		if convReason != "" {
			decision.ConvergenceReason = convReason
		} else {
			decision.ConvergenceReason = "CONVERGED"
		}
		// SP-2 非红旗连续不确定安全兜底：服务分差 < EntropyDiffMin 时收敛至全科/综合内科评估
		if isSP2UncertainConvergence(rankedServices, clinicalCtx, safetyEval, p.config.EntropyDiffMin) {
			decision.ConvergenceReason = "UNCERTAIN_STREAK"
			promoteGeneralInternalFallback(decision, p.snapshot)
		}
	} else if nextInquiry != nil {
		decision.NextInquiry = nextInquiry
	}
	decision.CanEarlyTerminate = p.config.AllowEarlyConvergence && redFlagsCleared && safetyEval.SafetyState == "ROUTINE_PATHWAY" && len(rankedServices) > 0

	// 7. 人卫统编教材逐字循证证据链装配 (Evidence Assembly)
	p.assembleEvidence(decision, activeSymptoms, scoredDiseases)

	return decision, nil
}

// linkEntities 实体链接，支持 concept_code 及名称映射
func (p *InferencePipeline) linkEntities(inputs []string) []*SymptomNode {
	var nodes []*SymptomNode
	seen := make(map[string]bool)

	for _, input := range inputs {
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		// 1. 直接精确命中 concept_code
		if node, ok := p.snapshot.Symptoms[input]; ok {
			if !seen[node.ConceptCode] {
				seen[node.ConceptCode] = true
				nodes = append(nodes, node)
			}
			continue
		}
		// 2. 倒排索引检索
		matched := p.snapshot.SearchSymptoms(input, 1)
		if len(matched) > 0 {
			node := matched[0]
			if !seen[node.ConceptCode] {
				seen[node.ConceptCode] = true
				nodes = append(nodes, node)
			}
		}
	}
	return nodes
}

type safetyEvaluation struct {
	Alert             *EmergencyAlert
	SafetyState       string // ROUTINE_PATHWAY / URGENT_ASSESSMENT / EMERGENCY_NOW
	PositiveUrgentHit bool   // 命中显式 URGENT 规则或选了 is_red_flag 选项
	UncertainHit      bool   // 在红旗题上选择了 UNCERTAIN 安全兜底
}

func (p *InferencePipeline) resolveAnsweredLabels(ctx ClinicalContext) map[string]string {
	if len(ctx.Answers) == 0 {
		return nil
	}
	labels := make(map[string]string, len(ctx.Answers))
	for _, qID := range sortedAnswerKeys(ctx.Answers) {
		ansVal := strings.TrimSpace(ctx.Answers[qID])
		if ansVal == "" {
			continue
		}
		q, ok := p.snapshot.Questions[qID]
		if !ok || q == nil {
			continue
		}
		if label := resolveOptionLabel(q, ansVal); label != "" {
			labels[qID] = label
		}
	}
	if len(labels) == 0 {
		return nil
	}
	return labels
}

func resolveOptionLabel(q *QuestionNode, ansVal string) string {
	var matched []string
	for _, opt := range q.Options {
		if optionMatchesValue(ansVal, opt.Value) && opt.Label != "" {
			matched = append(matched, opt.Label)
		}
	}
	if len(matched) == 0 {
		return ""
	}
	return strings.Join(matched, "、")
}

func sortedAnswerKeys(answers map[string]string) []string {
	keys := make([]string, 0, len(answers))
	for k := range answers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func optionMatchesValue(selected, expected string) bool {
	selected = strings.TrimSpace(selected)
	expected = strings.TrimSpace(expected)
	if selected == "" || expected == "" {
		return false
	}
	if selected == expected {
		return true
	}
	if strings.Contains(selected, ",") {
		for _, part := range strings.Split(selected, ",") {
			if strings.TrimSpace(part) == expected {
				return true
			}
		}
	}
	return false
}

func normalizeRuleSafetyState(rule *SafetyRuleNode) string {
	if rule == nil {
		return "ROUTINE_PATHWAY"
	}
	if rule.Level == "CRITICAL" || rule.SafetyState == "EMERGENCY_NOW" {
		return "EMERGENCY_NOW"
	}
	if rule.Level == "URGENT" || rule.SafetyState == "URGENT_ASSESSMENT" {
		return "URGENT_ASSESSMENT"
	}
	return "EMERGENCY_NOW"
}

func safetySeverityRank(state string) int {
	switch state {
	case "EMERGENCY_NOW":
		return 0
	case "URGENT_ASSESSMENT":
		return 1
	default:
		return 2
	}
}

func sortSafetyRules(rules map[string]*SafetyRuleNode) []*SafetyRuleNode {
	sorted := make([]*SafetyRuleNode, 0, len(rules))
	for _, r := range rules {
		if r != nil {
			sorted = append(sorted, r)
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		ri := safetySeverityRank(normalizeRuleSafetyState(sorted[i]))
		rj := safetySeverityRank(normalizeRuleSafetyState(sorted[j]))
		if ri != rj {
			return ri < rj
		}
		return sorted[i].RuleID < sorted[j].RuleID
	})
	return sorted
}

func ruleMatchesSymptoms(rule *SafetyRuleNode, symCodeSet map[string]bool) bool {
	for _, prim := range rule.PrimarySymptoms {
		if symCodeSet[prim] {
			return true
		}
	}
	return false
}

func matchesAnswerCondition(qID, expectedVal string, ctx ClinicalContext) bool {
	if expectedVal == "" {
		return false
	}
	// 结构化 answers 优先：精确绑定 question_id 与 option_value，杜绝跨题 YES 串台
	if len(ctx.Answers) > 0 {
		if qID != "" {
			if ansVal, ok := ctx.Answers[qID]; ok {
				return optionMatchesValue(ansVal, expectedVal)
			}
		} else {
			for _, qKey := range sortedAnswerKeys(ctx.Answers) {
				if optionMatchesValue(ctx.Answers[qKey], expectedVal) {
					return true
				}
			}
		}
		// 若已有结构化 answers，且仅针对未答题的 Legacy ConfirmedSigns，仅允许非泛用布尔值精确匹配
		if expectedVal != "YES" && expectedVal != "NO" && expectedVal != "UNCERTAIN" {
			for _, cs := range ctx.ConfirmedSigns {
				if strings.TrimSpace(cs) == expectedVal || (qID != "" && strings.TrimSpace(cs) == qID+"_"+expectedVal) {
					return true
				}
			}
		}
		return false
	}

	// 纯 Legacy 调用方（仅传入 ConfirmedSigns）：严格精确等值匹配
	for _, cs := range ctx.ConfirmedSigns {
		trimmed := strings.TrimSpace(cs)
		if trimmed == expectedVal || (qID != "" && trimmed == qID+"_"+expectedVal) {
			return true
		}
	}
	return false
}

func ruleTriggerMatched(rule *SafetyRuleNode, symCodeSet map[string]bool, ctx ClinicalContext) bool {
	cond := rule.TriggerCondition
	if len(cond) == 0 {
		// 安全铁律：绝不允许空触发条件的规则自动返回 true 导致误报 120 急救
		return false
	}

	if conceptsRaw, ok := cond["any_of_concepts"]; ok {
		if concepts, ok := conceptsRaw.([]interface{}); ok {
			for _, c := range concepts {
				if cStr, ok := c.(string); ok && symCodeSet[cStr] {
					return true
				}
			}
		}
	}

	if answersRaw, ok := cond["any_of_answers"]; ok {
		if answers, ok := answersRaw.([]interface{}); ok {
			for _, ans := range answers {
				if ansMap, ok := ans.(map[string]interface{}); ok {
					qID, _ := ansMap["question_id"].(string)
					val, _ := ansMap["value"].(string)
					if matchesAnswerCondition(qID, val, ctx) {
						return true
					}
				}
			}
		}
	}

	if signsRaw, ok := cond["any_of_signs"]; ok {
		if signs, ok := signsRaw.([]interface{}); ok {
			for _, sign := range signs {
				signStr, _ := sign.(string)
				if signStr == "" {
					continue
				}
				for _, cs := range ctx.ConfirmedSigns {
					if strings.TrimSpace(cs) == signStr {
						return true
					}
				}
				for _, ansVal := range ctx.Answers {
					if optionMatchesValue(ansVal, signStr) {
						return true
					}
				}
			}
		}
	}

	if medHistoryRaw, ok := cond["any_of_medical_history"]; ok {
		if histList, ok := medHistoryRaw.([]interface{}); ok {
			for _, h := range histList {
				hStr, _ := h.(string)
				if hStr == "" {
					continue
				}
				for _, mh := range ctx.MedicalHistory {
					if strings.Contains(mh, hStr) {
						return true
					}
				}
			}
		}
	}

	return false
}

func buildAlertFromRule(rule *SafetyRuleNode) *EmergencyAlert {
	safetyState := normalizeRuleSafetyState(rule)
	call120 := safetyState == "EMERGENCY_NOW"
	recDept := "急诊医学科"
	if safetyState == "URGENT_ASSESSMENT" {
		recDept = "急诊医学科 / 专科优先门诊"
	}
	guideText := rule.WarningText
	if action := rule.EmergencyAction; len(action) > 0 {
		if c120, ok := action["call_120"].(bool); ok {
			call120 = c120
		}
		if dept, ok := action["recommended_dept_name"].(string); ok && dept != "" {
			recDept = dept
		}
		if gt, ok := action["guide_text"].(string); ok && gt != "" {
			guideText = gt
		}
	}

	return &EmergencyAlert{
		RuleID:               rule.RuleID,
		RuleName:             rule.Name,
		Level:                rule.Level,
		Title:                rule.Title,
		WarningText:          rule.WarningText,
		Call120:              call120,
		RecommendedDeptName:  recDept,
		RecommendedGuideText: guideText,
	}
}

func questionHasRedFlagOption(q *QuestionNode) bool {
	if q == nil {
		return false
	}
	for _, opt := range q.Options {
		if opt.IsRedFlag {
			return true
		}
	}
	return false
}

func (p *InferencePipeline) resolveServiceName(svcCode string) string {
	if svcCode == "" {
		return ""
	}
	if svc, ok := p.snapshot.Services[svcCode]; ok && svc != nil && svc.ServiceName != "" {
		return svc.ServiceName
	}
	return ""
}

// evaluateSafety 评估安全规则与选项级/不确定级红旗兜底
func (p *InferencePipeline) evaluateSafety(symptoms []*SymptomNode, ctx ClinicalContext) safetyEvaluation {
	res := safetyEvaluation{SafetyState: "ROUTINE_PATHWAY"}

	symCodeSet := make(map[string]bool, len(symptoms))
	for _, s := range symptoms {
		symCodeSet[s.ConceptCode] = true
	}

	// 1. 确定性顺序评估已注册规则（CRITICAL > URGENT，按 RuleID 升序；极危主诉由 RULE_CRIT_THUNDERCLAP 等 CONCEPT_DIRECT 规则数据驱动）
	for _, rule := range sortSafetyRules(p.snapshot.SafetyRules) {
		if !ruleMatchesSymptoms(rule, symCodeSet) {
			continue
		}
		if !ruleTriggerMatched(rule, symCodeSet, ctx) {
			continue
		}
		state := normalizeRuleSafetyState(rule)
		alert := buildAlertFromRule(rule)
		if state == "EMERGENCY_NOW" {
			return safetyEvaluation{
				SafetyState: "EMERGENCY_NOW",
				Alert:       alert,
			}
		}
		// URGENT 规则命中：记录优先级最高的 URGENT 规则，继续排查其余已选答案中是否命中 CRITICAL 规则或兜底
		if res.Alert == nil {
			res.Alert = alert
		}
		res.SafetyState = "URGENT_ASSESSMENT"
		res.PositiveUrgentHit = true
	}

	// 3. 运行时选项级安全双保险 + UNCERTAIN 安全兜底
	for _, qID := range sortedAnswerKeys(ctx.Answers) {
		ansVal := strings.TrimSpace(ctx.Answers[qID])
		if ansVal == "" {
			continue
		}
		q, ok := p.snapshot.Questions[qID]
		if !ok || q == nil {
			continue
		}
		isRedFlagQuestion := q.Category == "RED_FLAG" || questionHasRedFlagOption(q)
		for _, opt := range q.Options {
			if !optionMatchesValue(ansVal, opt.Value) {
				continue
			}
			// 3a. 选中了未绑定 CRITICAL 规则的 is_red_flag 选项：至少升级至 URGENT_ASSESSMENT
			if opt.IsRedFlag {
				res.SafetyState = "URGENT_ASSESSMENT"
				res.PositiveUrgentHit = true
				if res.Alert == nil {
					recDept := p.resolveServiceName(opt.ServiceHint)
					if recDept == "" {
						recDept = "急诊医学科 / 对应专科急诊"
					}
					res.Alert = &EmergencyAlert{
						RuleID:               "RULE_FALLBACK_RED_FLAG",
						RuleName:             "警示征象运行时安全兜底预警",
						Level:                "URGENT",
						Title:                "检测到重要医学警示征象，建议尽快就诊",
						WarningText:          fmt.Sprintf("您在问诊中确认了高危警示表现（%s），请尽快前往医院急诊或对应专科进一步排查。", opt.Label),
						Call120:              false,
						RecommendedDeptName:  recDept,
						RecommendedGuideText: "请于当日尽快前往医院急诊或专科门诊完成医生评估；若症状持续加重请立即前往急诊科。",
					}
				}
			}
			// 3b. 红旗题选择了“不确定”(UNCERTAIN)：安全底线至少提升至 URGENT_ASSESSMENT
			if isRedFlagQuestion && (opt.IsUncertain || opt.Value == "UNCERTAIN") {
				res.SafetyState = "URGENT_ASSESSMENT"
				res.UncertainHit = true
				if res.Alert == nil {
					res.Alert = &EmergencyAlert{
						RuleID:               "RULE_UNCERTAIN_RED_FLAG_FLOOR",
						RuleName:             "高危征象不确定安全兜底预警",
						Level:                "URGENT",
						Title:                "高危警示征象未能排除，建议尽快线下评估",
						WarningText:          "您对关键危重警示症状选择了“不确定”，为确保就医安全，系统无法完全排除急性高危风险，建议优先前往急诊或相应专科尽快由医生面诊排查。",
						Call120:              false,
						RecommendedDeptName:  "急诊医学科 / 对应专科优先门诊",
						RecommendedGuideText: "请优先前往医院接受线下检查以排除急危重症；若出现剧烈疼痛、呼吸困难或意识改变请立即拨打120。",
					}
				}
			}
		}
	}

	return res
}

func symptomDemographicMultiplier(sym *SymptomNode, ctx ClinicalContext) float64 {
	if sym == nil {
		return 1.0
	}
	mul := 1.0
	gender := normalizeGenderInput(ctx.Gender)
	limitGender := strings.ToUpper(strings.TrimSpace(sym.GenderLimit))
	if gender == "MALE" && limitGender == "FEMALE" {
		return 0.15
	}
	if gender == "FEMALE" && limitGender == "MALE" {
		return 0.15
	}
	ageGroup := normalizeAgeGroupTag(sym.RecommendedAgeGroup)
	if ctx.Age > 0 {
		if ageGroup == "PEDIATRIC" {
			if ctx.Age <= 14 {
				mul *= 1.3
			} else if ctx.Age > 18 {
				mul *= 0.35
			}
		} else if ageGroup == "ELDERLY" {
			if ctx.Age >= 60 {
				mul *= 1.2
			} else if ctx.Age < 50 {
				mul *= 0.65
			}
		}
	}
	return mul
}

// spreadingActivation 受激扩散加权算子（含人群性别/年龄硬加权 + ACCOMPANIED_BY/伴随特征 2 跳扩散）
func (p *InferencePipeline) spreadingActivation(symptoms []*SymptomNode, ctx ClinicalContext) []ScoredDisease {
	if p.config.MaxDistance <= 0 {
		return nil
	}
	diseaseScores := make(map[string]float64)
	diseaseHits := make(map[string]int) // 记录多主诉协同汇聚次数
	signWeights := make(map[string]float64)

	// 人群特征加权
	demographicsWeight := func(d *DiseaseNode) float64 {
		weight := 1.0
		// 年龄偏好
		if ctx.Age > 0 && ctx.Age < 14 && strings.Contains(d.DiseaseName, "小儿") {
			weight *= 1.3
		}
		if ctx.Age >= 60 && (strings.Contains(d.DiseaseName, "脑卒中") || strings.Contains(d.DiseaseName, "冠心病") || strings.Contains(d.DiseaseName, "高血压")) {
			weight *= 1.2
		}
		// 性别限制
		if ctx.Gender == "MALE" && (strings.Contains(d.DiseaseName, "子宫") || strings.Contains(d.DiseaseName, "卵巢") || strings.Contains(d.DiseaseName, "妊娠")) {
			return 0.0
		}
		if ctx.Gender == "FEMALE" && (strings.Contains(d.DiseaseName, "前列腺") || strings.Contains(d.DiseaseName, "睾丸")) {
			return 0.0
		}
		return weight
	}

	decayFactor := 1.0
	if p.config.Decay > 0 && p.config.Decay < 1.0 {
		decayFactor = 1.0 - p.config.Decay
	}

	// 第 1 跳：主诉直连疾病扩散遍历 + 收集 ACCOMPANIED_BY 2 跳节点
	for _, sym := range symptoms {
		symDemo := symptomDemographicMultiplier(sym, ctx)
		outEdges := p.snapshot.OutEdges[sym.ConceptCode]
		for _, edge := range outEdges {
			if edge.TargetType == "DISEASE" {
				disCode := edge.TargetCode
				diseaseNode, exists := p.snapshot.Diseases[disCode]
				if !exists {
					continue
				}

				demoW := demographicsWeight(diseaseNode) * symDemo
				if demoW <= 0.0 {
					continue
				}

				hitIdx := diseaseHits[disCode]
				hopWeight := 1.0
				if hitIdx > 0 {
					hopWeight = decayFactor
				}
				scoreContribution := edge.Weight * demoW * hopWeight
				diseaseScores[disCode] += scoreContribution
				diseaseHits[disCode]++
			} else if edge.RelationType == "ACCOMPANIED_BY" && edge.TargetCode != "" {
				w := edge.Weight * symDemo
				if w > signWeights[edge.TargetCode] {
					signWeights[edge.TargetCode] = w
				}
			}
		}
		if p.config.MaxDistance >= 2 {
			for _, comp := range sym.AssociatedComplications {
				if comp.Code != "" {
					if 0.45*symDemo > signWeights[comp.Code] {
						signWeights[comp.Code] = 0.45 * symDemo
					}
				}
			}
		}
	}

	// 第 2 跳：经由已确认体征/伴随症状向关联疾病传导扩散
	if p.config.MaxDistance >= 2 {
		for _, cs := range ctx.ConfirmedSigns {
			trimmed := strings.TrimSpace(cs)
			if trimmed != "" && signWeights[trimmed] < 0.7 {
				signWeights[trimmed] = 0.7
			}
		}
		for hopNodeCode, hopBaseWeight := range signWeights {
			hopEdges := p.snapshot.OutEdges[hopNodeCode]
			for _, edge := range hopEdges {
				if edge.TargetType != "DISEASE" {
					continue
				}
				disCode := edge.TargetCode
				diseaseNode, exists := p.snapshot.Diseases[disCode]
				if !exists {
					continue
				}
				demoW := demographicsWeight(diseaseNode)
				if demoW <= 0.0 {
					continue
				}
				contrib := hopBaseWeight * edge.Weight * decayFactor * 0.5 * demoW
				diseaseScores[disCode] += contrib
				diseaseHits[disCode]++
			}
		}
	}

	// 多主诉协同汇聚指数级加权 (Co-occurrence Convergence)
	var result []ScoredDisease
	for disCode, baseScore := range diseaseScores {
		hits := diseaseHits[disCode]
		coBoost := 0.3 * decayFactor
		if p.config.MaxDistance <= 1 {
			coBoost = 0.0
		}
		finalScore := baseScore * (1.0 + coBoost*float64(hits-1))

		// 伴随体征强化与排除
		for _, cs := range ctx.ConfirmedSigns {
			if strings.Contains(disCode, cs) || (p.snapshot.Diseases[disCode] != nil && strings.Contains(p.snapshot.Diseases[disCode].Description, cs)) {
				finalScore += 0.25
			}
		}
		for _, ns := range ctx.NegativeSigns {
			if p.snapshot.Diseases[disCode] != nil && strings.Contains(p.snapshot.Diseases[disCode].Description, ns) {
				finalScore -= 0.30
			}
		}

		if finalScore <= 0 || (p.config.SpreadingThreshold > 0 && finalScore < p.config.SpreadingThreshold) {
			continue
		}

		node := p.snapshot.Diseases[disCode]
		if node != nil {
			result = append(result, ScoredDisease{
				DiseaseCode: disCode,
				DiseaseName: node.DiseaseName,
				Score:       math.Round(finalScore*100) / 100,
				IsCritical:  node.IsCritical,
				ServiceCode: node.DeptServiceCode,
				Description: node.Description,
			})
		}
	}

	// 按得分降序排列，分数相同时按 DiseaseCode 字典序保证确定性
	sort.Slice(result, func(i, j int) bool {
		if result[i].Score != result[j].Score {
			return result[i].Score > result[j].Score
		}
		return result[i].DiseaseCode < result[j].DiseaseCode
	})

	if len(result) > 10 {
		result = result[:10]
	}
	return result
}

const differentialOptionBoost = 3.5

// routeWesternServices 路由现代医学科室服务（融合主诉、疾病激活及已选追问服务倾向）
func (p *InferencePipeline) routeWesternServices(diseases []ScoredDisease, symptoms []*SymptomNode, ctx ClinicalContext) []RankedService {
	serviceScores := make(map[string]float64)

	// 1. 主诉默认服务兜底基础分 + BOOSTS_SERVICE 显式边（受性别/年龄人口学权重调节）
	for _, sym := range symptoms {
		symDemo := symptomDemographicMultiplier(sym, ctx)
		if sym.DefaultServiceCode != "" {
			serviceScores[sym.DefaultServiceCode] += 1.0 * symDemo
		}
		for _, diffSvc := range sym.DifferentialServices {
			serviceScores[diffSvc] += 0.5 * symDemo
		}
		for _, edge := range p.snapshot.OutEdges[sym.ConceptCode] {
			if edge.RelationType == "BOOSTS_SERVICE" && edge.TargetCode != "" {
				w := edge.Weight
				if w <= 0 {
					w = 0.5
				}
				serviceScores[edge.TargetCode] += w * symDemo
			}
		}
	}

	// 2. 疾病激活加权（同一科室取最高分疾病主导权重 + 次要疾病边际递减增益，防止同科室多条边堆叠淹没鉴别答题信号）
	maxDiseaseBySvc := make(map[string]float64)
	diseaseCountBySvc := make(map[string]int)
	for _, d := range diseases {
		if d.ServiceCode == "" {
			continue
		}
		diseaseCountBySvc[d.ServiceCode]++
		if d.Score > maxDiseaseBySvc[d.ServiceCode] {
			maxDiseaseBySvc[d.ServiceCode] = d.Score
		}
	}
	for svcCode, topScore := range maxDiseaseBySvc {
		extraHits := diseaseCountBySvc[svcCode] - 1
		if extraHits < 0 {
			extraHits = 0
		}
		serviceScores[svcCode] += topScore*0.8 + float64(extraHits)*0.12
	}

	// 3. 融合结构化追问答案中的 ServiceHint（结合 opt.BoostWeight，鉴别题与红旗确认项强加权，常规题微量平手决胜）
	for _, qID := range sortedAnswerKeys(ctx.Answers) {
		ansVal := strings.TrimSpace(ctx.Answers[qID])
		if ansVal == "" {
			continue
		}
		q, ok := p.snapshot.Questions[qID]
		if !ok || q == nil {
			continue
		}
		for _, opt := range q.Options {
			if !optionMatchesValue(ansVal, opt.Value) || opt.ServiceHint == "" {
				continue
			}
			if opt.IsUncertain || opt.Value == "UNCERTAIN" {
				continue
			}
			weightScale := opt.BoostWeight
			if weightScale <= 0 {
				weightScale = 1.0
			}
			if q.Category == "DIFFERENTIAL" || opt.IsRedFlag {
				serviceScores[opt.ServiceHint] += differentialOptionBoost * weightScale
			} else {
				serviceScores[opt.ServiceHint] += 0.05 * weightScale
			}
		}
	}

	var ranked []RankedService
	for svcCode, score := range serviceScores {
		svcNode := p.snapshot.Services[svcCode]
		name := svcCode
		desc := ""
		var priKeywords, fallKeywords []string
		if svcNode != nil {
			name = svcNode.ServiceName
			desc = svcNode.Description
			priKeywords = svcNode.PrimaryDeptKeywords
			fallKeywords = svcNode.FallbackDeptKeywords
		}

		ranked = append(ranked, RankedService{
			ServiceCode:          svcCode,
			ServiceName:          name,
			Score:                math.Round(score*100) / 100,
			IsPrimary:            false,
			TrackType:            "WESTERN_GENERAL",
			Description:          desc,
			PrimaryDeptKeywords:  priKeywords,
			FallbackDeptKeywords: fallKeywords,
		})
	}

	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].Score != ranked[j].Score {
			return ranked[i].Score > ranked[j].Score
		}
		return ranked[i].ServiceCode < ranked[j].ServiceCode
	})

	if len(ranked) > 0 {
		ranked[0].IsPrimary = true
	}
	if len(ranked) > 5 {
		ranked = ranked[:5]
	}

	return ranked
}

// inferTCMTrack 中医双轨辨证与专病门诊推演（支持四诊选项 TCMSyndromeHints + SYNDROME_TO_CLINIC 2 跳聚合 + 儿童专病年龄拦截）
func (p *InferencePipeline) inferTCMTrack(symptoms []*SymptomNode, diseases []ScoredDisease, ctx ClinicalContext) *TCMRecommendation {
	rec := &TCMRecommendation{
		Syndromes: make([]TCMSyndromeResult, 0),
		Clinics:   make([]TCMClinicResult, 0),
		Therapies: make([]TCMTherapyResult, 0),
	}

	syndromeScores := make(map[string]float64)
	syndromeEvidence := make(map[string]string)
	clinicScores := make(map[string]float64)
	therapySet := make(map[string]bool)

	for _, sym := range symptoms {
		edges := p.snapshot.OutEdges[sym.ConceptCode]
		for _, e := range edges {
			if e.RelationType == "SYMPTOM_TO_SYNDROME" {
				syndromeScores[e.TargetCode] += e.Weight
				if e.EvidenceQuote != "" {
					syndromeEvidence[e.TargetCode] = e.EvidenceQuote
				}
			} else if e.RelationType == "SYMPTOM_TO_CLINIC" {
				clinicScores[e.TargetCode] += e.Weight
			} else if e.RelationType == "SYMPTOM_TO_THERAPY" {
				therapySet[e.TargetCode] = true
			}
		}
	}

	// 四诊追问选项对中医证候的增量辨证激活
	for _, qID := range sortedAnswerKeys(ctx.Answers) {
		ansVal := strings.TrimSpace(ctx.Answers[qID])
		if ansVal == "" {
			continue
		}
		q, ok := p.snapshot.Questions[qID]
		if !ok || q == nil {
			continue
		}
		for _, opt := range q.Options {
			if !optionMatchesValue(ansVal, opt.Value) || opt.IsUncertain || opt.Value == "UNCERTAIN" {
				continue
			}
			boost := opt.BoostWeight
			if boost <= 0 {
				boost = 1.0
			}
			for _, synCode := range opt.TCMSyndromeHints {
				synCode = strings.TrimSpace(synCode)
				if synCode == "" {
					continue
				}
				syndromeScores[synCode] += 0.65 * boost
				if syndromeEvidence[synCode] == "" && opt.Label != "" {
					syndromeEvidence[synCode] = "四诊追问命中「" + opt.Label + "」"
				}
			}
		}
	}

	// 第 2 跳：证候 -> 专病门诊 (SYNDROME_TO_CLINIC)
	for synCode, synScore := range syndromeScores {
		for _, e := range p.snapshot.OutEdges[synCode] {
			if e.RelationType == "SYNDROME_TO_CLINIC" && e.TargetCode != "" {
				w := e.Weight
				if w <= 0 {
					w = 0.8
				}
				clinicScores[e.TargetCode] += synScore * w * 0.7
			}
		}
	}

	// 1. 组装中医证候
	for synCode, score := range syndromeScores {
		node := p.snapshot.TCMSyndromes[synCode]
		if node != nil {
			rec.Syndromes = append(rec.Syndromes, TCMSyndromeResult{
				SyndromeCode:  synCode,
				SyndromeName:  node.SyndromeName,
				OrganSystem:   node.OrganSystem,
				Pathogen:      node.PathogenNature,
				Score:         math.Round(score*100) / 100,
				TypicalSigns:  node.TypicalSigns,
				Description:   node.Description,
				EvidenceQuote: syndromeEvidence[synCode],
			})
		}
	}
	sort.Slice(rec.Syndromes, func(i, j int) bool {
		if rec.Syndromes[i].Score != rec.Syndromes[j].Score {
			return rec.Syndromes[i].Score > rec.Syndromes[j].Score
		}
		return rec.Syndromes[i].SyndromeCode < rec.Syndromes[j].SyndromeCode
	})

	// 2. 组装专病门诊（成人 age > 14 自动过滤小儿推拿门诊）
	for clinicCode, score := range clinicScores {
		if clinicCode == "TCM_CLINIC_PEDIATRIC_MASSAGE" && ctx.Age > 14 {
			continue
		}
		node := p.snapshot.TCMClinics[clinicCode]
		if node != nil {
			rec.Clinics = append(rec.Clinics, TCMClinicResult{
				ClinicCode:         clinicCode,
				ClinicName:         node.ClinicName,
				DeptName:           node.DeptName,
				ExpertTitles:       node.ExpertTitles,
				SpecialityTags:     node.SpecialityTags,
				TargetDeptKeywords: node.TargetDeptKeywords,
				BookingURL:         node.BookingURL,
				Score:              math.Round(score*100) / 100,
			})
		}
	}
	sort.Slice(rec.Clinics, func(i, j int) bool {
		if rec.Clinics[i].Score != rec.Clinics[j].Score {
			return rec.Clinics[i].Score > rec.Clinics[j].Score
		}
		return rec.Clinics[i].ClinicCode < rec.Clinics[j].ClinicCode
	})

	// 3. 组装外治疗法
	for thCode := range therapySet {
		node := p.snapshot.TCMTherapies[thCode]
		if node != nil {
			rec.Therapies = append(rec.Therapies, TCMTherapyResult{
				TherapyCode:          thCode,
				TherapyName:          node.TherapyName,
				Category:             node.Category,
				Indications:          node.Indications,
				ClinicRecommendation: node.ClinicRecommendation,
				Description:          node.Description,
			})
		}
	}
	sort.Slice(rec.Therapies, func(i, j int) bool {
		return rec.Therapies[i].TherapyCode < rec.Therapies[j].TherapyCode
	})

	return rec
}

// selectNextInquiry 信息熵最小化追问选择算子
type candidateQuestion struct {
	node         *QuestionNode
	tier         int
	matchCount   int
	canFlipSvc   bool
	distinctHint int
}

func (p *InferencePipeline) collectCriticalRuleQuestionIDs(symCodeSet map[string]bool) map[string]bool {
	critQIDs := make(map[string]bool)
	for _, rule := range p.snapshot.SafetyRules {
		if rule == nil || normalizeRuleSafetyState(rule) != "EMERGENCY_NOW" {
			continue
		}
		if !ruleMatchesSymptoms(rule, symCodeSet) {
			continue
		}
		if answersRaw, ok := rule.TriggerCondition["any_of_answers"]; ok {
			if answers, ok := answersRaw.([]interface{}); ok {
				for _, ans := range answers {
					if ansMap, ok := ans.(map[string]interface{}); ok {
						if qID, _ := ansMap["question_id"].(string); qID != "" {
							critQIDs[qID] = true
						}
					}
				}
			}
		}
	}
	return critQIDs
}

func collectAnsweredQuestionIDs(applicable []*QuestionNode, ctx ClinicalContext) map[string]bool {
	answered := make(map[string]bool, len(ctx.Answers))
	for qID, ansVal := range ctx.Answers {
		if strings.TrimSpace(ansVal) != "" {
			answered[qID] = true
		}
	}

	// 仅在无结构化 Answers 时兼容 Legacy ConfirmedSigns / NegativeSigns，防止泛用 YES/NO 跨题误杀
	if len(answered) == 0 && (len(ctx.ConfirmedSigns) > 0 || len(ctx.NegativeSigns) > 0) {
		legacySigns := make(map[string]bool, len(ctx.ConfirmedSigns)+len(ctx.NegativeSigns))
		for _, s := range ctx.ConfirmedSigns {
			if trimmed := strings.TrimSpace(s); trimmed != "" {
				legacySigns[trimmed] = true
			}
		}
		for _, s := range ctx.NegativeSigns {
			if trimmed := strings.TrimSpace(s); trimmed != "" {
				legacySigns[trimmed] = true
			}
		}
		for _, q := range applicable {
			if legacySigns[q.QuestionID] {
				answered[q.QuestionID] = true
				continue
			}
			for _, opt := range q.Options {
				if legacySigns[q.QuestionID+"_"+opt.Value] {
					answered[q.QuestionID] = true
					break
				}
				isGenericBool := opt.Value == "YES" || opt.Value == "NO" || opt.Value == "UNCERTAIN"
				if legacySigns[opt.Value] && (!isGenericBool || len(applicable) == 1) {
					answered[q.QuestionID] = true
					break
				}
			}
		}
	}

	return answered
}

func questionTier(q *QuestionNode, critQIDs map[string]bool) int {
	isRedFlag := q.Category == "RED_FLAG" || questionHasRedFlagOption(q)
	if isRedFlag && critQIDs[q.QuestionID] {
		return 3
	}
	if isRedFlag {
		return 2
	}
	return 1
}

func questionCanFlipService(ranked []RankedService, q *QuestionNode) bool {
	if len(ranked) == 0 || q == nil {
		return false
	}
	topCode := ranked[0].ServiceCode
	topScore := ranked[0].Score
	scoreByCode := make(map[string]float64, len(ranked))
	for _, svc := range ranked {
		scoreByCode[svc.ServiceCode] = svc.Score
	}

	// 鉴别题或红旗选项在 routeWesternServices 中的最高加权为 differentialOptionBoost * weightScale
	for _, opt := range q.Options {
		if opt.ServiceHint == "" || opt.ServiceHint == topCode {
			continue
		}
		if opt.IsUncertain || opt.Value == "UNCERTAIN" {
			continue
		}
		weightScale := opt.BoostWeight
		if weightScale <= 0 {
			weightScale = 1.0
		}
		if scoreByCode[opt.ServiceHint]+differentialOptionBoost*weightScale >= topScore {
			return true
		}
	}
	return false
}

func countDistinctServiceHints(q *QuestionNode) int {
	hints := make(map[string]bool)
	for _, opt := range q.Options {
		if opt.ServiceHint != "" && !opt.IsUncertain && opt.Value != "UNCERTAIN" {
			hints[opt.ServiceHint] = true
		}
	}
	return len(hints)
}

func anyTier1CanFlipService(ranked []RankedService, tier1 []candidateQuestion) bool {
	for _, c := range tier1 {
		if questionCanFlipService(ranked, c.node) {
			return true
		}
	}
	return false
}

func isSP2UncertainConvergence(ranked []RankedService, ctx ClinicalContext, safety safetyEvaluation, entropyDiffMin float64) bool {
	if ctx.UncertainStreak < 2 {
		return false
	}
	if safety.SafetyState != "ROUTINE_PATHWAY" || safety.UncertainHit || safety.PositiveUrgentHit {
		return false
	}
	if len(ranked) == 0 {
		return true
	}
	if len(ranked) == 1 {
		return false
	}
	diffThreshold := entropyDiffMin
	if diffThreshold <= 0 {
		diffThreshold = 0.25
	}
	return (ranked[0].Score - ranked[1].Score) < diffThreshold
}

func promoteGeneralInternalFallback(decision *TriageDecision, snap *GraphSnapshot) {
	if decision == nil {
		return
	}
	const fallbackCode = "GENERAL_INTERNAL_ASSESSMENT"
	svcName := "综合普通内科服务（全院兜底）"
	desc := "多次追问无法明确具体亚专科倾向，建议先由全科/普通内科线下评估"
	priKeywords := []string{"普通内科", "全科医疗科", "综合内科"}
	var fallKeywords []string
	if snap != nil {
		if svc, ok := snap.Services[fallbackCode]; ok && svc != nil {
			if svc.ServiceName != "" {
				svcName = svc.ServiceName
			}
			if svc.Description != "" {
				desc = svc.Description + "（连续不确定安全收敛兜底）"
			}
			if len(svc.PrimaryDeptKeywords) > 0 {
				priKeywords = svc.PrimaryDeptKeywords
			}
			fallKeywords = svc.FallbackDeptKeywords
		}
	}
	topScore := 1.0
	if len(decision.RankedServices) > 0 && decision.RankedServices[0].Score >= topScore {
		topScore = math.Round((decision.RankedServices[0].Score+0.25)*100) / 100
	}
	fallbackSvc := RankedService{
		ServiceCode:          fallbackCode,
		ServiceName:          svcName,
		Score:                topScore,
		IsPrimary:            true,
		TrackType:            "WESTERN_GENERAL",
		Description:          desc,
		PrimaryDeptKeywords:  priKeywords,
		FallbackDeptKeywords: fallKeywords,
	}
	reordered := make([]RankedService, 0, len(decision.RankedServices)+1)
	reordered = append(reordered, fallbackSvc)
	for _, item := range decision.RankedServices {
		if item.ServiceCode == fallbackCode {
			continue
		}
		item.IsPrimary = false
		reordered = append(reordered, item)
	}
	if len(reordered) > 5 {
		reordered = reordered[:5]
	}
	decision.RankedServices = reordered
}

// selectNextInquiry 信息熵与安全分级追问算子，返回 (下一题, 是否收敛, 红旗是否已清零, 收敛原因)
func (p *InferencePipeline) selectNextInquiry(diseases []ScoredDisease, westernServices []RankedService, symptoms []*SymptomNode, ctx ClinicalContext, safety safetyEvaluation) (*InquiryStep, bool, bool, string) {
	symSet := make(map[string]bool, len(symptoms))
	allDirectSingleHop := len(symptoms) > 0
	for _, s := range symptoms {
		symSet[s.ConceptCode] = true
		if strings.ToUpper(strings.TrimSpace(s.SkipFollowupMode)) != "DIRECT_SINGLE_HOP" {
			allDirectSingleHop = false
		}
	}

	// 1. 收集适用于当前主诉集的全部题库并按 QuestionID 确定性排序
	applicable := make([]*QuestionNode, 0)
	matchCounts := make(map[string]int)
	for _, q := range p.snapshot.Questions {
		if q == nil {
			continue
		}
		cnt := 0
		for _, appSym := range q.ApplicableSymptoms {
			if symSet[appSym] {
				cnt++
			}
		}
		if cnt > 0 {
			applicable = append(applicable, q)
			matchCounts[q.QuestionID] = cnt
		}
	}
	sort.Slice(applicable, func(i, j int) bool {
		return applicable[i].QuestionID < applicable[j].QuestionID
	})

	// 2. 按 question_id 精确去重 + 会话 slot_key 跨卡去重（红旗题遵守安全铁律，不被 KnownSlots 泛化静音）
	answeredQIDs := collectAnsweredQuestionIDs(applicable, ctx)
	answeredAnswerSlots := make(map[string]bool)
	for qID, ansVal := range ctx.Answers {
		if strings.TrimSpace(ansVal) == "" {
			continue
		}
		if q, ok := p.snapshot.Questions[qID]; ok && q != nil && strings.TrimSpace(q.SlotKey) != "" {
			answeredAnswerSlots[strings.TrimSpace(q.SlotKey)] = true
		}
	}
	knownSlots := make(map[string]bool, len(ctx.KnownSlots))
	for k, v := range ctx.KnownSlots {
		if strings.TrimSpace(v) != "" {
			knownSlots[strings.TrimSpace(k)] = true
		}
	}

	critQIDs := p.collectCriticalRuleQuestionIDs(symSet)
	gender := normalizeGenderInput(ctx.Gender)
	genderUnknown := gender == "UNKNOWN"

	candidates := make([]candidateQuestion, 0, len(applicable))
	hasUnaskedRedFlag := false
	for _, q := range applicable {
		if answeredQIDs[q.QuestionID] {
			continue
		}
		slotKey := strings.TrimSpace(q.SlotKey)
		if slotKey == "is_pregnant" && gender == "MALE" {
			continue
		}
		tier := questionTier(q, critQIDs)
		if slotKey != "" {
			if tier < 2 {
				if answeredAnswerSlots[slotKey] || knownSlots[slotKey] {
					continue
				}
			} else {
				// tier >= 2 红旗题仅当本轮已显式回答同 slot 题目，或安全白名单阴性/明确槽位（has_fever / onset_acute）才可跳过
				safeKnownSlot := knownSlots[slotKey] && (slotKey == "has_fever" || slotKey == "onset_acute")
				if answeredAnswerSlots[slotKey] || safeKnownSlot {
					continue
				}
			}
		}
		if tier >= 2 {
			hasUnaskedRedFlag = true
		}
		candidates = append(candidates, candidateQuestion{
			node:         q,
			tier:         tier,
			matchCount:   matchCounts[q.QuestionID],
			canFlipSvc:   questionCanFlipService(westernServices, q),
			distinctHint: countDistinctServiceHints(q),
		})
	}
	redFlagsCleared := !hasUnaskedRedFlag

	// 3. 若已命中显式 URGENT 规则或选了红旗选项：仅继续排查剩余 CRITICAL 红旗题（无则跳过常规题直接收敛）
	if safety.PositiveUrgentHit {
		critOnly := make([]candidateQuestion, 0)
		for _, c := range candidates {
			if c.tier == 3 {
				critOnly = append(critOnly, c)
			}
		}
		candidates = critOnly
		if len(candidates) == 0 {
			return nil, true, redFlagsCleared, "URGENT_RULE_HIT"
		}
	}

	// 3.2 红旗已清零且全部主诉标注 DIRECT_SINGLE_HOP（单跳直达专科）：跳过常规鉴别追问直接收敛（含 0 题目场景）
	if !hasUnaskedRedFlag && allDirectSingleHop {
		return nil, true, true, "DIRECT_SINGLE_HOP"
	}

	// 3.3 SP-2 连续非红旗不确定 >= 2 次：在红旗已清零前提下终止继续追问并收敛（含第 2 题恰好耗尽候选场景）
	if !hasUnaskedRedFlag && ctx.UncertainStreak >= 2 {
		return nil, true, true, "UNCERTAIN_STREAK"
	}

	if len(candidates) == 0 {
		return nil, true, redFlagsCleared, "CONVERGED"
	}

	// 3.5 若已达最大临床追问步数 max_steps，在非 CRITICAL 红旗下强制收敛结束临床追问
	if p.config.MaxSteps > 0 && len(answeredQIDs) >= p.config.MaxSteps {
		return nil, true, redFlagsCleared, "MAX_STEPS"
	}

	// 4. 在所有红旗题已清零后，判定是否已达到科室/疾病级收敛条件 (受 allow_early_convergence 控制)
	if p.config.AllowEarlyConvergence && !hasUnaskedRedFlag {
		canFlip := anyTier1CanFlipService(westernServices, candidates)
		hasAnsweredAny := len(ctx.Answers) > 0 || len(ctx.ConfirmedSigns) > 0
		if hasAnsweredAny && len(westernServices) > 0 && !canFlip {
			return nil, true, true, "CONVERGED"
		}
		if !canFlip {
			diseasesConverged := len(diseases) == 0 ||
				(len(diseases) == 1 && diseases[0].Score >= p.config.ConfidenceThreshold) ||
				(len(diseases) >= 2 && (diseases[0].Score-diseases[1].Score) >= p.config.EntropyDiffMin && diseases[0].Score >= p.config.ConfidenceThreshold)
			if diseasesConverged {
				return nil, true, true, "CONVERGED"
			}
		}
	}

	// 5. 确定性优先级排序：CRITICAL 红旗(3) > 普通红旗(2) > 鉴别追问(1) -> 未知性别下孕产排查优先 -> 能翻转推荐科室 -> 多科室区分度 -> 症状命中数 -> QuestionID
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].tier != candidates[j].tier {
			return candidates[i].tier > candidates[j].tier
		}
		if genderUnknown {
			iPreg := strings.TrimSpace(candidates[i].node.SlotKey) == "is_pregnant"
			jPreg := strings.TrimSpace(candidates[j].node.SlotKey) == "is_pregnant"
			if iPreg != jPreg {
				return iPreg
			}
		}
		if candidates[i].canFlipSvc != candidates[j].canFlipSvc {
			return candidates[i].canFlipSvc
		}
		if candidates[i].distinctHint != candidates[j].distinctHint {
			return candidates[i].distinctHint > candidates[j].distinctHint
		}
		if candidates[i].matchCount != candidates[j].matchCount {
			return candidates[i].matchCount > candidates[j].matchCount
		}
		return candidates[i].node.QuestionID < candidates[j].node.QuestionID
	})

	picked := candidates[0]
	infoGain := 0.82
	if picked.tier == 3 {
		infoGain = 0.98
	} else if picked.tier == 2 {
		infoGain = 0.92
	}

	return &InquiryStep{
		QuestionID:  picked.node.QuestionID,
		Title:       picked.node.Title,
		Type:        picked.node.Type,
		Category:    picked.node.Category,
		SlotKey:     picked.node.SlotKey,
		Options:     picked.node.Options,
		Information: infoGain,
	}, false, redFlagsCleared, ""
}

// assembleEvidence 装配人卫教材逐字原句出处（优先命中推演疾病，无重叠时兜底 EvidenceSummary 与其余条目）
func (p *InferencePipeline) assembleEvidence(decision *TriageDecision, symptoms []*SymptomNode, diseases []ScoredDisease) {
	seenQuotes := make(map[string]bool)

	appendChain := func(s *SymptomNode, rec *EvidenceRecord, target, quote string) bool {
		if quote == "" || seenQuotes[quote] {
			return false
		}
		seenQuotes[quote] = true
		decision.EvidenceChains = append(decision.EvidenceChains, EvidenceChain{
			SymptomCode:   s.ConceptCode,
			SymptomName:   s.ConceptName,
			Textbook:      rec.TextbookSource,
			Chapter:       rec.Chapter,
			Page:          rec.Page,
			Quote:         quote,
			MatchedTarget: target,
		})
		return len(decision.EvidenceChains) >= 5
	}

	for _, s := range symptoms {
		records := p.snapshot.EvidenceIndex[s.ConceptCode]
		for _, rec := range records {
			for _, diff := range rec.DiffDiseases {
				disName, _ := diff["disease_name"].(string)
				quote, _ := diff["distinguishing_features"].(string)
				if quote == "" {
					continue
				}
				isHit := len(diseases) == 0
				if !isHit {
					for _, d := range diseases {
						if strings.Contains(d.DiseaseName, disName) || strings.Contains(disName, d.DiseaseName) {
							isHit = true
							break
						}
					}
				}
				if isHit && appendChain(s, rec, disName, quote) {
					return
				}
			}
		}
	}

	if len(decision.EvidenceChains) == 0 {
		for _, s := range symptoms {
			records := p.snapshot.EvidenceIndex[s.ConceptCode]
			for _, rec := range records {
				if rec.EvidenceSummary != "" {
					if appendChain(s, rec, s.ConceptName, rec.EvidenceSummary) {
						return
					}
				}
				for _, diff := range rec.DiffDiseases {
					disName, _ := diff["disease_name"].(string)
					quote, _ := diff["distinguishing_features"].(string)
					if appendChain(s, rec, disName, quote) {
						return
					}
				}
			}
		}
	}
}
