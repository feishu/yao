package triagekb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cloneSnapshotForUpgradeTest(base *GraphSnapshot) *GraphSnapshot {
	snap := &GraphSnapshot{
		Version:      "test-upgrade",
		Services:     make(map[string]*ServiceNode, len(base.Services)),
		Symptoms:     make(map[string]*SymptomNode, len(base.Symptoms)),
		Diseases:     make(map[string]*DiseaseNode, len(base.Diseases)),
		Questions:    make(map[string]*QuestionNode, len(base.Questions)),
		SafetyRules:  make(map[string]*SafetyRuleNode, len(base.SafetyRules)),
		OutEdges:     make(map[string][]*GraphEdge, len(base.OutEdges)),
		EvidenceIndex: make(map[string][]*EvidenceRecord, len(base.EvidenceIndex)),
		TCMSyndromes: make(map[string]*TCMSyndromeNode, len(base.TCMSyndromes)),
		TCMClinics:   make(map[string]*TCMClinicNode, len(base.TCMClinics)),
		TCMTherapies: make(map[string]*TCMTherapyNode, len(base.TCMTherapies)),
	}
	for k, v := range base.Services {
		snap.Services[k] = v
	}
	for k, v := range base.Symptoms {
		copyNode := *v
		snap.Symptoms[k] = &copyNode
	}
	for k, v := range base.Diseases {
		snap.Diseases[k] = v
	}
	for k, v := range base.Questions {
		copyQ := *v
		snap.Questions[k] = &copyQ
	}
	for k, v := range base.SafetyRules {
		snap.SafetyRules[k] = v
	}
	for k, v := range base.OutEdges {
		edges := make([]*GraphEdge, len(v))
		copy(edges, v)
		snap.OutEdges[k] = edges
	}
	for k, v := range base.EvidenceIndex {
		snap.EvidenceIndex[k] = v
	}
	for k, v := range base.TCMSyndromes {
		snap.TCMSyndromes[k] = v
	}
	for k, v := range base.TCMClinics {
		snap.TCMClinics[k] = v
	}
	for k, v := range base.TCMTherapies {
		snap.TCMTherapies[k] = v
	}
	snap.BuildInvertedIndex()
	return snap
}

func TestPipelineSlotKeyDedupSafetyIronRuleAndUnknownGenderPregnancy(t *testing.T) {
	eng := prepareEngine(t)
	snap := cloneSnapshotForUpgradeTest(eng.GetSnapshot())

	// 构造含以下特点的合成测试症状：
	// - Tier 1 常规槽位 has_fever（应被 KnownSlots["has_fever"]="NO" 跳过）
	// - Tier 2 红旗槽位 severe_disp（不应被 KnownSlots 随意静音！必须明确回答）
	// - Tier 2 孕产槽位 is_pregnant（Gender=UNKNOWN 时优先问，Gender=MALE 时跳过）
	snap.Symptoms["SYM_TEST_ABD"] = &SymptomNode{
		ConceptCode:        "SYM_TEST_ABD",
		ConceptName:        "下腹部疼痛",
		TriageRole:         TriageRoleCoreComplaint,
		SkipFollowupMode:   SkipFollowupRequired,
		DefaultServiceCode: "GASTRO_INTERNAL_ASSESSMENT",
	}
	snap.Questions["Q_TEST_FEVER"] = &QuestionNode{
		QuestionID:         "Q_TEST_FEVER",
		Title:              "是否发烧？",
		Category:           "DIFFERENTIAL",
		SlotKey:            "has_fever",
		ApplicableSymptoms: []string{"SYM_TEST_ABD"},
		Options: []InquiryOption{
			{Label: "高热", Value: "YES", ServiceHint: "INFECTIOUS_ASSESSMENT", BoostWeight: 1.0},
			{Label: "无热", Value: "NO"},
		},
	}
	snap.Questions["Q_TEST_RED_SHOCK"] = &QuestionNode{
		QuestionID:         "Q_TEST_RED_SHOCK",
		Title:              "是否伴晕厥出冷汗？",
		Category:           "RED_FLAG",
		SlotKey:            "syncope_cold_sweat",
		ApplicableSymptoms: []string{"SYM_TEST_ABD"},
		Options: []InquiryOption{
			{Label: "是", Value: "YES", IsRedFlag: true, ServiceHint: "GASTRO_SURGERY_ASSESSMENT", BoostWeight: 1.5},
			{Label: "否", Value: "NO"},
		},
	}
	snap.Questions["Q_TEST_PREG"] = &QuestionNode{
		QuestionID:         "Q_TEST_PREG",
		Title:              "是否存在停经或可能怀孕？",
		Category:           "RED_FLAG",
		SlotKey:            "is_pregnant",
		ApplicableSymptoms: []string{"SYM_TEST_ABD"},
		Options: []InquiryOption{
			{Label: "可能怀孕", Value: "YES", IsRedFlag: true, ServiceHint: "OBSTETRICS_ASSESSMENT", BoostWeight: 2.0},
			{Label: "无可能", Value: "NO"},
		},
	}

	pipe := NewInferencePipeline(snap, DefaultPipelineConfig())

	// 1. 铁律测试：即便 KnownSlots 宣称 syncope_cold_sweat="NO"，Tier>=2 不可靠模糊槽位跳过；
	// 并且 Gender=UNKNOWN 下 is_pregnant 同 Tier 优先级排第一！
	decUnknown, err := pipe.Execute(ClinicalContext{
		Symptoms: []string{"SYM_TEST_ABD"},
		Gender:   "UNKNOWN",
		KnownSlots: map[string]string{
			"has_fever":          "NO",
			"syncope_cold_sweat": "NO",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, decUnknown.NextInquiry)
	assert.Equal(t, "Q_TEST_PREG", decUnknown.NextInquiry.QuestionID, "未知性别下应优先排查育龄停经/怀孕红旗")

	// 2. 回答 is_pregnant="NO" 后，下一步必须问 Q_TEST_RED_SHOCK（不能被 KnownSlots 静音）
	decAfterPreg, err := pipe.Execute(ClinicalContext{
		Symptoms: []string{"SYM_TEST_ABD"},
		Gender:   "UNKNOWN",
		KnownSlots: map[string]string{
			"has_fever":          "NO",
			"syncope_cold_sweat": "NO",
		},
		Answers: map[string]string{
			"Q_TEST_PREG": "NO",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, decAfterPreg.NextInquiry)
	assert.Equal(t, "Q_TEST_RED_SHOCK", decAfterPreg.NextInquiry.QuestionID, "红旗题不得被非白名单 KnownSlots 跳过")

	// 3. 回答 Q_TEST_RED_SHOCK="NO" 后，Q_TEST_FEVER(Tier 1) 因 KnownSlots["has_fever"]="NO" 被成功跳过并直接收敛！
	decDone, err := pipe.Execute(ClinicalContext{
		Symptoms: []string{"SYM_TEST_ABD"},
		Gender:   "UNKNOWN",
		KnownSlots: map[string]string{
			"has_fever":          "NO",
			"syncope_cold_sweat": "NO",
		},
		Answers: map[string]string{
			"Q_TEST_PREG":      "NO",
			"Q_TEST_RED_SHOCK": "NO",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "CONVERGED", decDone.State)
	assert.Nil(t, decDone.NextInquiry)

	// 4. 男性患者应自动跳过 is_pregnant 提问
	decMale, err := pipe.Execute(ClinicalContext{
		Symptoms: []string{"SYM_TEST_ABD"},
		Gender:   "MALE",
	})
	require.NoError(t, err)
	require.NotNil(t, decMale.NextInquiry)
	assert.Equal(t, "Q_TEST_RED_SHOCK", decMale.NextInquiry.QuestionID, "男性患者不应再追问怀孕")
}

func TestPipelineDirectSingleHopSP2VsRedFlagUncertainAnd2HopTCM(t *testing.T) {
	eng := prepareEngine(t)
	snap := cloneSnapshotForUpgradeTest(eng.GetSnapshot())

	// 1. 测试 DIRECT_SINGLE_HOP 单跳直达专科：在无未清零红旗时，跳过普通鉴别追问直接收敛
	snap.Symptoms["SYM_TEST_DIRECT"] = &SymptomNode{
		ConceptCode:        "SYM_TEST_DIRECT",
		ConceptName:        "迎风流泪",
		TriageRole:         TriageRoleCoreComplaint,
		SkipFollowupMode:   SkipFollowupDirectSingleHop,
		DefaultServiceCode: "OPHTHALMOLOGY_ASSESSMENT",
	}
	snap.Questions["Q_DIRECT_DIFF"] = &QuestionNode{
		QuestionID:         "Q_DIRECT_DIFF",
		Title:              "持续几天了？",
		Category:           "DIFFERENTIAL",
		SlotKey:            "onset_acute",
		ApplicableSymptoms: []string{"SYM_TEST_DIRECT"},
		Options:            []InquiryOption{{Label: "几天", Value: "DAYS"}},
	}

	pipe := NewInferencePipeline(snap, DefaultPipelineConfig())
	decDirect, err := pipe.Execute(ClinicalContext{Symptoms: []string{"SYM_TEST_DIRECT"}})
	require.NoError(t, err)
	assert.Equal(t, "CONVERGED", decDirect.State)
	assert.Equal(t, "DIRECT_SINGLE_HOP", decDirect.ConvergenceReason)
	assert.Nil(t, decDirect.NextInquiry)
	assert.Equal(t, "OPHTHALMOLOGY_ASSESSMENT", decDirect.RankedServices[0].ServiceCode)

	// 0 题纯单跳剧本卡也应返回 DIRECT_SINGLE_HOP
	snap.Symptoms["SYM_TEST_DIRECT_EMPTY"] = &SymptomNode{
		ConceptCode:        "SYM_TEST_DIRECT_EMPTY",
		ConceptName:        "单纯失眠",
		TriageRole:         TriageRoleCoreComplaint,
		SkipFollowupMode:   SkipFollowupDirectSingleHop,
		DefaultServiceCode: "NEURO_INTERNAL_ASSESSMENT",
	}
	decDirectEmpty, err := pipe.Execute(ClinicalContext{Symptoms: []string{"SYM_TEST_DIRECT_EMPTY"}})
	require.NoError(t, err)
	assert.Equal(t, "CONVERGED", decDirectEmpty.State)
	assert.Equal(t, "DIRECT_SINGLE_HOP", decDirectEmpty.ConvergenceReason)

	// 2. 测试 SP-2 vs 红旗 UNCERTAIN 分层安全边界
	snap.Symptoms["SYM_TEST_AMBIG"] = &SymptomNode{
		ConceptCode:          "SYM_TEST_AMBIG",
		ConceptName:          "非特异性乏力不适",
		TriageRole:           TriageRoleCoreComplaint,
		SkipFollowupMode:     SkipFollowupRequired,
		DefaultServiceCode:   "ENDOCRINE_ASSESSMENT",
		DifferentialServices: []string{"NEURO_INTERNAL_ASSESSMENT"},
	}
	// 使两个竞争科室分差极小，且挂载两道 DIFFERENTIAL + 一道 RED_FLAG
	snap.OutEdges["SYM_TEST_AMBIG"] = []*GraphEdge{
		{SourceType: "SYMPTOM", SourceCode: "SYM_TEST_AMBIG", TargetType: "SERVICE", TargetCode: "NEURO_INTERNAL_ASSESSMENT", RelationType: "BOOSTS_SERVICE", Weight: 0.48},
	}
	snap.Questions["Q_AMBIG_1"] = &QuestionNode{
		QuestionID:         "Q_AMBIG_1",
		Title:              "更偏重头昏还是多饮多尿？",
		Category:           "DIFFERENTIAL",
		SlotKey:            "ambig_s1",
		ApplicableSymptoms: []string{"SYM_TEST_AMBIG"},
		Options: []InquiryOption{
			{Label: "多饮", Value: "THIRST", ServiceHint: "ENDOCRINE_ASSESSMENT"},
			{Label: "不清楚", Value: "OPT_UNCERTAIN", IsUncertain: true},
		},
	}
	snap.Questions["Q_AMBIG_RED"] = &QuestionNode{
		QuestionID:         "Q_AMBIG_RED",
		Title:              "是否突发偏瘫昏迷？",
		Category:           "RED_FLAG",
		SlotKey:            "neuro_red",
		ApplicableSymptoms: []string{"SYM_TEST_AMBIG"},
		Options: []InquiryOption{
			{Label: "无", Value: "NO"},
			{Label: "不确定", Value: "OPT_UNCERTAIN", IsUncertain: true},
		},
	}

	cfg := DefaultPipelineConfig()
	cfg.EntropyDiffMin = 0.25
	pipe2 := NewInferencePipeline(snap, cfg)

	// 2a. 红旗题选 NO，但非红旗连续 2 次不确定 -> SP-2 触发全科兜底 GENERAL_INTERNAL_ASSESSMENT
	decSP2, err := pipe2.Execute(ClinicalContext{
		Symptoms:        []string{"SYM_TEST_AMBIG"},
		UncertainStreak: 2,
		Answers: map[string]string{
			"Q_AMBIG_RED": "NO",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "CONVERGED", decSP2.State)
	assert.Equal(t, "UNCERTAIN_STREAK", decSP2.ConvergenceReason)
	assert.Equal(t, "ROUTINE_PATHWAY", decSP2.SafetyState)
	require.NotEmpty(t, decSP2.RankedServices)
	assert.Equal(t, "GENERAL_INTERNAL_ASSESSMENT", decSP2.RankedServices[0].ServiceCode)

	// 2b. 若红旗题本身选了 UNCERTAIN，绝对禁止降级到全科，必须维持 URGENT_ASSESSMENT！
	decRedUncertain, err := pipe2.Execute(ClinicalContext{
		Symptoms:        []string{"SYM_TEST_AMBIG"},
		UncertainStreak: 2,
		Answers: map[string]string{
			"Q_AMBIG_RED": "OPT_UNCERTAIN",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "URGENT_ASSESSMENT", decRedUncertain.SafetyState)
	require.NotNil(t, decRedUncertain.EmergencyAlert)
	assert.Equal(t, "RULE_UNCERTAIN_RED_FLAG_FLOOR", decRedUncertain.EmergencyAlert.RuleID)
	if len(decRedUncertain.RankedServices) > 0 {
		assert.NotEqual(t, "GENERAL_INTERNAL_ASSESSMENT", decRedUncertain.RankedServices[0].ServiceCode)
	}

	// 3. 测试 2 跳中医证候 -> 专病门诊推理 + 小儿推拿门诊对成人自动过滤
	snap.Questions["Q_TCM_TEST"] = &QuestionNode{
		QuestionID:         "Q_TCM_TEST",
		Title:              "近期舌苔脉象及伴随感受？",
		Category:           "TCM",
		SlotKey:            "tcm_tongue",
		ApplicableSymptoms: []string{"SYM_TEST_AMBIG"},
		Options: []InquiryOption{
			{
				Label:            "心烦多梦、舌红少苔",
				Value:            "INSOMNIA_HEAT",
				BoostWeight:      1.2,
				TCMSyndromeHints: []string{"TCM_SYN_TEST_YIN_DEF"},
			},
		},
	}
	snap.TCMSyndromes["TCM_SYN_TEST_YIN_DEF"] = &TCMSyndromeNode{
		SyndromeCode: "TCM_SYN_TEST_YIN_DEF",
		SyndromeName: "阴虚火旺证",
	}
	snap.TCMClinics["TCM_CLINIC_SLEEP_HERBAL"] = &TCMClinicNode{
		ClinicCode: "TCM_CLINIC_SLEEP_HERBAL",
		ClinicName: "贵中医睡眠与草本调理专病门诊",
		DeptName:   "中医治未病与神志睡眠科",
	}
	snap.TCMClinics["TCM_CLINIC_PEDIATRIC_MASSAGE"] = &TCMClinicNode{
		ClinicCode: "TCM_CLINIC_PEDIATRIC_MASSAGE",
		ClinicName: "贵中医小儿推拿专病门诊",
		DeptName:   "小儿推拿科",
	}
	snap.OutEdges["TCM_SYN_TEST_YIN_DEF"] = []*GraphEdge{
		{SourceType: "SYNDROME", SourceCode: "TCM_SYN_TEST_YIN_DEF", TargetType: "CLINIC", TargetCode: "TCM_CLINIC_SLEEP_HERBAL", RelationType: "SYNDROME_TO_CLINIC", Weight: 0.9},
		{SourceType: "SYNDROME", SourceCode: "TCM_SYN_TEST_YIN_DEF", TargetType: "CLINIC", TargetCode: "TCM_CLINIC_PEDIATRIC_MASSAGE", RelationType: "SYNDROME_TO_CLINIC", Weight: 0.8},
	}

	pipeTCM := NewInferencePipeline(snap, cfg)
	// 成人（35岁）应推导出 SLEEP_HERBAL 并过滤掉 PEDIATRIC_MASSAGE
	decAdultTCM, err := pipeTCM.Execute(ClinicalContext{
		Symptoms: []string{"SYM_TEST_AMBIG"},
		Age:      35,
		Answers: map[string]string{
			"Q_AMBIG_RED": "NO",
			"Q_TCM_TEST":  "INSOMNIA_HEAT",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, decAdultTCM.TCM)
	require.NotEmpty(t, decAdultTCM.TCM.Syndromes)
	assert.Equal(t, "TCM_SYN_TEST_YIN_DEF", decAdultTCM.TCM.Syndromes[0].SyndromeCode)
	var adultClinicCodes []string
	for _, c := range decAdultTCM.TCM.Clinics {
		adultClinicCodes = append(adultClinicCodes, c.ClinicCode)
	}
	assert.Contains(t, adultClinicCodes, "TCM_CLINIC_SLEEP_HERBAL")
	assert.NotContains(t, adultClinicCodes, "TCM_CLINIC_PEDIATRIC_MASSAGE", "成人患者禁止推荐小儿推拿专病门诊")

	// 儿童（8岁）应保留小儿推拿专病门诊
	decChildTCM, err := pipeTCM.Execute(ClinicalContext{
		Symptoms: []string{"SYM_TEST_AMBIG"},
		Age:      8,
		Answers: map[string]string{
			"Q_AMBIG_RED": "NO",
			"Q_TCM_TEST":  "INSOMNIA_HEAT",
		},
	})
	require.NoError(t, err)
	var childClinicCodes []string
	for _, c := range decChildTCM.TCM.Clinics {
		childClinicCodes = append(childClinicCodes, c.ClinicCode)
	}
	assert.Contains(t, childClinicCodes, "TCM_CLINIC_PEDIATRIC_MASSAGE")

	// 4. 性别权重保护：MALE 遇到 FEMALE 专属症状被抑制，UNKNOWN 不抑制女性专属症状
	snap.Symptoms["SYM_TEST_GYN"] = &SymptomNode{
		ConceptCode:        "SYM_TEST_GYN",
		ConceptName:        "非经期异常阴道出血",
		GenderLimit:        "FEMALE",
		TriageRole:         TriageRoleCoreComplaint,
		SkipFollowupMode:   SkipFollowupDirectSingleHop,
		DefaultServiceCode: "GYNECOLOGY_ASSESSMENT",
	}
	pipeGender := NewInferencePipeline(snap, cfg)
	decGynUnknown, err := pipeGender.Execute(ClinicalContext{
		Symptoms: []string{"SYM_TEST_GYN"},
		Gender:   "UNKNOWN",
	})
	require.NoError(t, err)
	require.NotEmpty(t, decGynUnknown.RankedServices)
	assert.GreaterOrEqual(t, decGynUnknown.RankedServices[0].Score, 1.0)
	decGynMale, err := pipeGender.Execute(ClinicalContext{
		Symptoms: []string{"SYM_TEST_GYN"},
		Gender:   "MALE",
	})
	require.NoError(t, err)
	require.NotEmpty(t, decGynMale.RankedServices)
	assert.Less(t, decGynMale.RankedServices[0].Score, 0.5)
}

