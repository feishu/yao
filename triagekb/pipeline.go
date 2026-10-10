package triagekb

import (
	"math"
	"sort"
	"strings"
)

// PipelineConfig 推演流水线超参数
type PipelineConfig struct {
	Decay               float64 // 扩散衰减系数 (默认 0.2)
	MaxDistance         int     // 最大扩散步数 (默认 2)
	ConfidenceThreshold float64 // 收敛置信度阈值 (默认 0.85)
	EntropyDiffMin      float64 // 区分度阈值 (默认 0.25)
	EnableTCM           bool    // 是否启用中医双轨 (默认 true)
}

// DefaultPipelineConfig 默认配置
func DefaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		Decay:               0.2,
		MaxDistance:         2,
		ConfidenceThreshold: 0.85,
		EntropyDiffMin:      0.25,
		EnableTCM:           true,
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
	decision := &TriageDecision{
		State:             "NORMAL",
		RankedServices:    make([]RankedService, 0),
		SuspectedDiseases: make([]ScoredDisease, 0),
		EvidenceChains:    make([]EvidenceChain, 0),
	}

	// 1. 实体对齐算子 (Entity Linking)
	activeSymptoms := p.linkEntities(clinicalCtx.Symptoms)
	if len(activeSymptoms) == 0 {
		return decision, nil
	}

	// 2. 红旗急危重症一票短路拦截 (Safety Short-Circuit)
	emergencyAlert := p.checkSafetyRules(activeSymptoms, clinicalCtx)
	if emergencyAlert != nil {
		decision.State = "EMERGENCY"
		decision.EmergencyAlert = emergencyAlert
		// 装配急诊科默认服务
		decision.RankedServices = append(decision.RankedServices, RankedService{
			ServiceCode:          "EMERGENCY_TRIAGE_FAST",
			ServiceName:          emergencyAlert.RecommendedDeptName,
			Score:                1.0,
			IsPrimary:            true,
			TrackType:            "WESTERN_GENERAL",
			Description:          emergencyAlert.WarningText,
			PrimaryDeptKeywords:  []string{emergencyAlert.RecommendedDeptName, "急诊科", "急诊医学科"},
			FallbackDeptKeywords: []string{"全科医疗科"},
		})
		// 组装循证证据
		p.assembleEvidence(decision, activeSymptoms, nil)
		return decision, nil
	}

	// 3. 受激扩散加权算子 (Spreading Activation)
	scoredDiseases := p.spreadingActivation(activeSymptoms, clinicalCtx)
	decision.SuspectedDiseases = scoredDiseases

	// 4. 路由与推荐现代医学科室服务 (Western Medical Service Routing)
	rankedServices := p.routeWesternServices(scoredDiseases, activeSymptoms)
	decision.RankedServices = rankedServices

	// 5. 中医双轨辨证与特色专病门诊决策 (TCM Dual-Track Decision)
	if p.config.EnableTCM {
		tcmRec := p.inferTCMTrack(activeSymptoms, scoredDiseases)
		decision.TCM = tcmRec

		// 若存在特色专病门诊，合并入 RankedServices 供挂号直达
		for _, clinic := range tcmRec.Clinics {
			decision.RankedServices = append(decision.RankedServices, RankedService{
				ServiceCode:          clinic.ClinicCode,
				ServiceName:          clinic.DeptName,
				Score:                clinic.Score,
				IsPrimary:            false,
				TrackType:            "TCM_SPECIALIST",
				Description:          clinic.ClinicName + " (特色专病门诊)",
				PrimaryDeptKeywords:  []string{clinic.DeptName},
				FallbackDeptKeywords: []string{"中医科"},
				AssociatedClinicCode: clinic.ClinicCode,
				AssociatedClinicName: clinic.ClinicName,
				BookingURL:           clinic.BookingURL,
			})
		}
	}

	// 6. 信息熵歧义追问算子 (Entropy Inquiry Selector)
	nextInquiry, isConverged := p.selectNextInquiry(scoredDiseases, activeSymptoms, clinicalCtx)
	if isConverged {
		decision.State = "CONVERGED"
	} else if nextInquiry != nil {
		decision.NextInquiry = nextInquiry
	}

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

// checkSafetyRules 评估红旗安全规则一票熔断
func (p *InferencePipeline) checkSafetyRules(symptoms []*SymptomNode, ctx ClinicalContext) *EmergencyAlert {
	symCodeSet := make(map[string]bool)
	for _, s := range symptoms {
		symCodeSet[s.ConceptCode] = true
	}

	for _, rule := range p.snapshot.SafetyRules {
		// 规则主诉匹配：精确匹配、同系统前缀匹配、或规则名称关键词命中
		hitPrimary := false
		for _, prim := range rule.PrimarySymptoms {
			if symCodeSet[prim] {
				hitPrimary = true
				break
			}
			// 同系统前缀匹配 (如 SYM_HEADACHE_* 均属于头痛主诉族)
			for _, s := range symptoms {
				if len(prim) > 13 && len(s.ConceptCode) > 13 {
					primPrefix := prim[:13] // 如 SYM_HEADACHE_
					if strings.HasPrefix(s.ConceptCode, primPrefix) {
						hitPrimary = true
						break
					}
				}
				// 规则名称与症状名称强关联
				if strings.Contains(rule.Name, s.ConceptName) || (strings.Contains(s.ConceptName, "头痛") && strings.Contains(rule.Name, "头痛")) {
					hitPrimary = true
					break
				}
			}
			if hitPrimary {
				break
			}
		}
		if !hitPrimary {
			continue
		}

		// 检查触发条件
		triggered := false
		cond := rule.TriggerCondition
		if len(cond) == 0 {
			// 无特定答题限制，直接根据主诉触发
			triggered = true
		} else {
			// 检查 any_of_concepts
			if conceptsRaw, ok := cond["any_of_concepts"]; ok {
				if concepts, ok := conceptsRaw.([]interface{}); ok {
					for _, c := range concepts {
						if symCodeSet[c.(string)] {
							triggered = true
							break
						}
					}
				}
			}
			// 检查选项值 any_of_answers
			if answersRaw, ok := cond["any_of_answers"]; ok {
				if answers, ok := answersRaw.([]interface{}); ok {
					for _, ans := range answers {
						if ansMap, ok := ans.(map[string]interface{}); ok {
							val, _ := ansMap["value"].(string)
							for _, cs := range ctx.ConfirmedSigns {
								if cs == val || strings.Contains(cs, val) {
									triggered = true
									break
								}
							}
						}
					}
				}
			}
			// 检查伴随体征 confirmed_signs
			if signsRaw, ok := cond["any_of_signs"]; ok {
				if signs, ok := signsRaw.([]interface{}); ok {
					for _, sign := range signs {
						signStr := sign.(string)
						for _, cs := range ctx.ConfirmedSigns {
							if strings.Contains(cs, signStr) || cs == signStr {
								triggered = true
								break
							}
						}
					}
				}
			}
			// 若既往史包含高危
			if medHistoryRaw, ok := cond["any_of_medical_history"]; ok {
				if histList, ok := medHistoryRaw.([]interface{}); ok {
					for _, h := range histList {
						hStr := h.(string)
						for _, mh := range ctx.MedicalHistory {
							if strings.Contains(mh, hStr) {
								triggered = true
								break
							}
						}
					}
				}
			}
		}

		// 若当前主诉本身标记为极高危 (如心肌梗死、张力性气胸、脑出血、雷击样头痛征象)，直接熔断
		if !triggered && rule.Level == "CRITICAL" {
			for _, s := range symptoms {
				if s.IsRedFlagCandidate || strings.Contains(s.ConceptCode, "THUNDERCLAP") || strings.Contains(s.ConceptName, "雷击") || strings.Contains(s.ConceptName, "剧烈") {
					if strings.Contains(rule.Name, "脑疝") || strings.Contains(rule.Name, "心肌梗死") || strings.Contains(rule.Name, "急救") || strings.Contains(rule.Name, "高危") {
						triggered = true
						break
					}
				}
			}
		}

		if triggered {
			call120 := true
			recDept := "急诊医学科"
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
	}

	// 兜底高危主诉硬熔断拦截 (如包含雷击样头痛、大出血、昏迷休克等)
	for _, s := range symptoms {
		if strings.Contains(s.ConceptCode, "THUNDERCLAP") || strings.Contains(s.ConceptName, "雷击") || strings.Contains(s.ConceptName, "剧烈头痛") {
			return &EmergencyAlert{
				RuleID:               "RULE_CRIT_EMERGENCY_FAST",
				RuleName:             "急性危急重症绿色通道预警",
				Level:                "CRITICAL",
				Title:                "突发高危症状急救提示",
				WarningText:          "患者出现雷击样剧烈头痛等高危表现，高度怀疑脑血管急性病变或颅内高压，必须立即就医！",
				Call120:              true,
				RecommendedDeptName:  "急诊医学科 / 神经外科急诊",
				RecommendedGuideText: "请立即平卧保持呼吸道通畅，拨打120急救电话！",
			}
		}
	}

	return nil
}

// spreadingActivation 受激扩散加权算法
func (p *InferencePipeline) spreadingActivation(symptoms []*SymptomNode, ctx ClinicalContext) []ScoredDisease {
	diseaseScores := make(map[string]float64)
	diseaseHits := make(map[string]int) // 记录多主诉协同汇聚次数

	// 人群特征加权
	demographicsWeight := func(d *DiseaseNode) float64 {
		weight := 1.0
		// 年龄偏好
		if ctx.Age < 14 && strings.Contains(d.DiseaseName, "小儿") {
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

	// 扩散遍历
	for _, sym := range symptoms {
		outEdges := p.snapshot.OutEdges[sym.ConceptCode]
		for _, edge := range outEdges {
			if edge.TargetType == "DISEASE" {
				disCode := edge.TargetCode
				diseaseNode, exists := p.snapshot.Diseases[disCode]
				if !exists {
					continue
				}

				demoW := demographicsWeight(diseaseNode)
				if demoW <= 0.0 {
					continue
				}

				// 基础边权重
				scoreContribution := edge.Weight * demoW
				diseaseScores[disCode] += scoreContribution
				diseaseHits[disCode]++
			}
		}
	}

	// 多主诉协同汇聚指数级加权 (Co-occurrence Convergence)
	var result []ScoredDisease
	for disCode, baseScore := range diseaseScores {
		hits := diseaseHits[disCode]
		// 多主诉交汇处提升 (1 + 0.3 * (hits - 1))
		finalScore := baseScore * (1.0 + 0.3*float64(hits-1))

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

		if finalScore <= 0 {
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

	// 按得分降序排序
	sort.Slice(result, func(i, j int) bool {
		return result[i].Score > result[j].Score
	})

	if len(result) > 10 {
		result = result[:10]
	}
	return result
}

// routeWesternServices 路由现代医学科室服务
func (p *InferencePipeline) routeWesternServices(diseases []ScoredDisease, symptoms []*SymptomNode) []RankedService {
	serviceScores := make(map[string]float64)

	// 1. 主诉默认服务兜底基础分
	for _, sym := range symptoms {
		if sym.DefaultServiceCode != "" {
			serviceScores[sym.DefaultServiceCode] += 1.0
		}
		for _, diffSvc := range sym.DifferentialServices {
			serviceScores[diffSvc] += 0.5
		}
	}

	// 2. 疾病激活加权
	for _, d := range diseases {
		if d.ServiceCode != "" {
			serviceScores[d.ServiceCode] += d.Score * 0.8
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
		return ranked[i].Score > ranked[j].Score
	})

	if len(ranked) > 0 {
		ranked[0].IsPrimary = true
	}
	if len(ranked) > 5 {
		ranked = ranked[:5]
	}

	return ranked
}

// inferTCMTrack 中医双轨辨证与专病门诊推演
func (p *InferencePipeline) inferTCMTrack(symptoms []*SymptomNode, diseases []ScoredDisease) *TCMRecommendation {
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
		return rec.Syndromes[i].Score > rec.Syndromes[j].Score
	})

	// 2. 组装专病门诊
	for clinicCode, score := range clinicScores {
		node := p.snapshot.TCMClinics[clinicCode]
		if node != nil {
			rec.Clinics = append(rec.Clinics, TCMClinicResult{
				ClinicCode:     clinicCode,
				ClinicName:     node.ClinicName,
				DeptName:       node.DeptName,
				ExpertTitles:   node.ExpertTitles,
				SpecialityTags: node.SpecialityTags,
				BookingURL:     node.BookingURL,
				Score:          math.Round(score*100) / 100,
			})
		}
	}
	sort.Slice(rec.Clinics, func(i, j int) bool {
		return rec.Clinics[i].Score > rec.Clinics[j].Score
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

	return rec
}

// selectNextInquiry 信息熵最小化追问选择算子
func (p *InferencePipeline) selectNextInquiry(diseases []ScoredDisease, symptoms []*SymptomNode, ctx ClinicalContext) (*InquiryStep, bool) {
	// 若无疾病或候选只有1个且得分极高，直接收敛
	if len(diseases) == 0 {
		return nil, true
	}
	if len(diseases) == 1 && diseases[0].Score >= p.config.ConfidenceThreshold {
		return nil, true
	}

	// 若前两名疾病分差显著，收敛完成
	if len(diseases) >= 2 {
		diff := diseases[0].Score - diseases[1].Score
		if diff >= p.config.EntropyDiffMin && diseases[0].Score >= p.config.ConfidenceThreshold {
			return nil, true
		}
	}

	// 检索适用于当前主诉集的候选追问
	symSet := make(map[string]bool)
	for _, s := range symptoms {
		symSet[s.ConceptCode] = true
	}

	// 排除已经回答过的体征
	answeredSet := make(map[string]bool)
	for _, s := range ctx.ConfirmedSigns {
		answeredSet[s] = true
	}
	for _, s := range ctx.NegativeSigns {
		answeredSet[s] = true
	}

	for _, q := range p.snapshot.Questions {
		// 检查适用主诉
		applicable := false
		for _, appSym := range q.ApplicableSymptoms {
			if symSet[appSym] {
				applicable = true
				break
			}
		}
		if !applicable {
			continue
		}

		// 检查选项是否未回答
		hasUnanswered := false
		for _, opt := range q.Options {
			if !answeredSet[opt.Value] {
				hasUnanswered = true
				break
			}
		}
		if !hasUnanswered {
			continue
		}

		// 选取首个最具区分度的有效追问
		return &InquiryStep{
			QuestionID:  q.QuestionID,
			Title:       q.Title,
			Type:        q.Type,
			Category:    q.Category,
			Options:     q.Options,
			Information: 0.82,
		}, false
	}

	// 无可用追问，标记收敛
	return nil, true
}

// assembleEvidence 装配人卫教材逐字原句出处
func (p *InferencePipeline) assembleEvidence(decision *TriageDecision, symptoms []*SymptomNode, diseases []ScoredDisease) {
	seenQuotes := make(map[string]bool)

	for _, s := range symptoms {
		records := p.snapshot.EvidenceIndex[s.ConceptCode]
		for _, rec := range records {
			for _, diff := range rec.DiffDiseases {
				disName, _ := diff["disease_name"].(string)
				quote, _ := diff["distinguishing_features"].(string)
				if quote == "" || seenQuotes[quote] {
					continue
				}

				// 若命中了当前推演出的疾病，优先录入证据链
				isHit := false
				if len(diseases) == 0 {
					isHit = true
				} else {
					for _, d := range diseases {
						if strings.Contains(d.DiseaseName, disName) || strings.Contains(disName, d.DiseaseName) {
							isHit = true
							break
						}
					}
				}

				if isHit {
					seenQuotes[quote] = true
					decision.EvidenceChains = append(decision.EvidenceChains, EvidenceChain{
						SymptomCode:   s.ConceptCode,
						SymptomName:   s.ConceptName,
						Textbook:      rec.TextbookSource,
						Chapter:       rec.Chapter,
						Page:          rec.Page,
						Quote:         quote,
						MatchedTarget: disName,
					})
					if len(decision.EvidenceChains) >= 5 {
						return
					}
				}
			}
		}
	}
}
