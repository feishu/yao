package triagekb

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestPaths() (string, string) {
	generalDB := "/Users/L/Desktop/Code/yao_projects/syd/service/data/triage/kb/kb_triage_general.db"
	tcmDB := "/Users/L/Desktop/Code/yao_projects/syd/service/data/triage/kb/kb_triage_tcm.db"
	return generalDB, tcmDB
}

func prepareEngine(t *testing.T) *Engine {
	genDB, tcmDB := getTestPaths()
	loader := NewFederatedLoader(FederatedGraphDSL{
		Version: "2.1.0",
	})
	snap, err := loader.LoadFromFiles(genDB, tcmDB)
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	engine := GetEngine()
	engine.SetSnapshot(snap)
	return engine
}

// 1. 测试图快照与倒排索引检索
func TestSnapshotAndSearch(t *testing.T) {
	engine := prepareEngine(t)
	snap := engine.GetSnapshot()

	assert.Equal(t, "2.1.0", snap.Version)
	assert.True(t, len(snap.Symptoms) >= 434, "主诉数量应>=434")
	assert.True(t, len(snap.Diseases) >= 1000, "鉴别疾病应>=1000")
	assert.True(t, len(snap.SafetyRules) >= 40, "安全规则应>=40")
	assert.True(t, len(snap.TCMSyndromes) >= 10, "中医证候应>=10")

	// 拼音简拼检索测试
	hits := engine.SearchSymptoms("ptt", 5)
	assert.NotEmpty(t, hits)
	assert.Equal(t, "SYM_HEADACHE_THROBBING", hits[0].ConceptCode)

	// 中文模糊检索
	hitsChest := engine.SearchSymptoms("胸痛", 5)
	assert.NotEmpty(t, hitsChest)
}

// 2. 测试危急重症 100% 熔断拦截 (Safety Short-Circuit)
func TestSafetyShortCircuit(t *testing.T) {
	engine := prepareEngine(t)

	testCases := []struct {
		name       string
		symptoms   []string
		confirmed  []string
		expectRule string
	}{
		{
			name:       "胸痛伴压榨感与向左肩放射",
			symptoms:   []string{"SYM_CHEST_PAIN"},
			confirmed:  []string{"LEFT_ARM_BACK", "YES"},
			expectRule: "RULE_CRIT_ACS_MI",
		},
		{
			name:       "剧烈胸痛出冷汗",
			symptoms:   []string{"SYM_CHEST_PAIN"},
			confirmed:  []string{"YES"},
			expectRule: "RULE_CRIT_ACS_MI",
		},
		{
			name:       "突发霹雳样剧烈头痛伴意识障碍",
			symptoms:   []string{"SYM_HEADACHE_THUNDERCLAP"},
			confirmed:  []string{},
			expectRule: "", // 会命中头痛危象或蛛网膜下腔出血
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dec, err := engine.Evaluate(ClinicalContext{
				Symptoms:       tc.symptoms,
				ConfirmedSigns: tc.confirmed,
				Age:            55,
				Gender:         "MALE",
			})
			assert.NoError(t, err)
			assert.NotNil(t, dec)
			assert.Equal(t, "EMERGENCY", dec.State, "危急病例必须被短路拦截为 EMERGENCY")
			assert.NotNil(t, dec.EmergencyAlert)
			assert.True(t, dec.EmergencyAlert.Call120)
			assert.NotEmpty(t, dec.RankedServices)
			assert.True(t, dec.RankedServices[0].IsPrimary)
		})
	}
}

// 3. 测试多主诉协同推演与鉴别诊断 (Spreading Activation & Differentiation)
func TestMultiSymptomDifferentiation(t *testing.T) {
	engine := prepareEngine(t)

	// Case A: 胸痛 + 反酸烧心 + 平卧加重 -> 协同指向反流性食管炎 / 消化内科
	decDigestive, err := engine.Evaluate(ClinicalContext{
		Symptoms: []string{"SYM_ACID_REFLUX", "SYM_EPIGASTRIC_PAIN"},
		Age:      40,
		Gender:   "FEMALE",
	})
	assert.NoError(t, err)
	assert.NotNil(t, decDigestive)
	assert.NotEmpty(t, decDigestive.RankedServices)
	topSvc := decDigestive.RankedServices[0]
	assert.Contains(t, []string{"GASTRO_INTERNAL_ASSESSMENT", "DIGESTIVE_INTERNAL_ASSESSMENT", "GENERAL_INTERNAL_ASSESSMENT"}, topSvc.ServiceCode)

	// Case B: 胃痛胃胀 + 畏寒肢冷 -> 同时激活消化内科与中医脾胃病科
	decDual, err := engine.Evaluate(ClinicalContext{
		Symptoms: []string{"SYM_EPIGASTRIC_PAIN", "SYM_ABDOMINAL_DISTENSION"},
		Age:      35,
		Gender:   "MALE",
	})
	assert.NoError(t, err)
	assert.NotNil(t, decDual)
	assert.NotNil(t, decDual.TCM, "必须输出中医推荐")
	assert.NotEmpty(t, decDual.TCM.Syndromes)
	assert.NotEmpty(t, decDual.TCM.Clinics)
	assert.NotEmpty(t, decDual.TCM.Therapies)
	// 验证专病门诊存在
	hasHPClinic := false
	for _, c := range decDual.TCM.Clinics {
		if c.ClinicCode == "TCM_CLINIC_HP_REGULATION" || c.ClinicCode == "TCM_CLINIC_CHRONIC_GASTRITIS_REVERSAL" {
			hasHPClinic = true
			break
		}
	}
	assert.True(t, hasHPClinic, "胃部不适应激活幽门螺杆菌耐药调理或胃炎阻断门诊")

	// Case C: 面部麻木口角歪斜 -> 命中针灸特色专病门诊
	decFacial, err := engine.Evaluate(ClinicalContext{
		Symptoms: []string{"SYM_FACIAL_NUMBNESS"},
		Age:      30,
		Gender:   "FEMALE",
	})
	assert.NoError(t, err)
	assert.NotNil(t, decFacial)
	assert.NotNil(t, decFacial.TCM)
	assert.NotEmpty(t, decFacial.TCM.Clinics)
	assert.Equal(t, "TCM_CLINIC_FACIAL_ACUTE_FAST", decFacial.TCM.Clinics[0].ClinicCode)
}

// 4. 测试并发高吞吐与热重载无抖动测试 (Zero-Lock Read & Concurrency)
func TestHighConcurrencyAndZeroLockHotReload(t *testing.T) {
	engine := prepareEngine(t)
	genDB, tcmDB := getTestPaths()

	concurrency := 100
	requestsPerRoutine := 50
	var wg sync.WaitGroup
	wg.Add(concurrency)

	stopReload := make(chan bool)

	// 并发热重载 Goroutine
	go func() {
		loader := NewFederatedLoader(FederatedGraphDSL{Version: "2.1.0"})
		for {
			select {
			case <-stopReload:
				return
			default:
				time.Sleep(20 * time.Millisecond)
				newSnap, err := loader.LoadFromFiles(genDB, tcmDB)
				if err == nil {
					engine.SetSnapshot(newSnap)
				}
			}
		}
	}()

	start := time.Now()
	for i := 0; i < concurrency; i++ {
		go func(routineIdx int) {
			defer wg.Done()
			for j := 0; j < requestsPerRoutine; j++ {
				sym := "SYM_COUGH"
				if j%2 == 0 {
					sym = "SYM_EPIGASTRIC_PAIN"
				}
				dec, err := engine.Evaluate(ClinicalContext{
					Symptoms: []string{sym},
					Age:      30,
					Gender:   "MALE",
				})
				assert.NoError(t, err)
				assert.NotNil(t, dec)
				assert.NotEmpty(t, dec.RankedServices)
			}
		}(i)
	}

	wg.Wait()
	close(stopReload)
	duration := time.Since(start)

	totalRequests := concurrency * requestsPerRoutine
	qps := float64(totalRequests) / duration.Seconds()
	t.Logf("并发压测完成: %d 次请求耗时 %v, 吞吐量 QPS: %.2f (热重载期间零Panic/零失败)", totalRequests, duration, qps)
	assert.True(t, qps > 1000, "微秒级内存无锁读取吞吐量应轻松超 1000 QPS")
}

// 5. 测试结膜炎/青光眼红旗分诊：不重复追问、URGENT 分级升级与 UNCERTAIN 安全兜底
func TestConjunctivitisRedEyeAndGlaucomaSafety(t *testing.T) {
	engine := prepareEngine(t)

	// Step 0: 首次确认主诉 -> 返回 Q_EYE_PRESSURE_HEADACHE，无红旗清零
	step0, err := engine.Evaluate(ClinicalContext{
		Symptoms: []string{"SYM_ACUTE_CONJUNCTIVITIS_RED_EYE"},
	})
	assert.NoError(t, err)
	assert.Equal(t, "NORMAL", step0.State)
	assert.Equal(t, "ROUTINE_PATHWAY", step0.SafetyState)
	assert.False(t, step0.CanEarlyTerminate)
	if assert.NotNil(t, step0.NextInquiry) {
		assert.Equal(t, "Q_EYE_PRESSURE_HEADACHE", step0.NextInquiry.QuestionID)
	}

	// 分支 A: 回答 YES -> 命中 URGENT 规则 RULE_CRIT_ACUTE_GLAUCOMA，不再重复追问并立即收敛
	stepYes, err := engine.Evaluate(ClinicalContext{
		Symptoms: []string{"SYM_ACUTE_CONJUNCTIVITIS_RED_EYE"},
		Answers:  map[string]string{"Q_EYE_PRESSURE_HEADACHE": "YES"},
	})
	assert.NoError(t, err)
	assert.Equal(t, "CONVERGED", stepYes.State)
	assert.Equal(t, "URGENT_ASSESSMENT", stepYes.SafetyState)
	assert.Nil(t, stepYes.NextInquiry, "已回答题目绝不能再次出现在 NextInquiry 中")
	if assert.NotNil(t, stepYes.EmergencyAlert) {
		assert.Equal(t, "RULE_CRIT_ACUTE_GLAUCOMA", stepYes.EmergencyAlert.RuleID)
		assert.Equal(t, "URGENT", stepYes.EmergencyAlert.Level)
	}
	assert.Contains(t, stepYes.AnsweredLabels["Q_EYE_PRESSURE_HEADACHE"], "虹")
	if assert.NotEmpty(t, stepYes.RankedServices) {
		assert.Equal(t, "OPHTHALMOLOGY_ASSESSMENT", stepYes.RankedServices[0].ServiceCode)
	}

	// 分支 B: 回答 NO -> 常规路径收敛并允许提前查看
	stepNo, err := engine.Evaluate(ClinicalContext{
		Symptoms: []string{"SYM_ACUTE_CONJUNCTIVITIS_RED_EYE"},
		Answers:  map[string]string{"Q_EYE_PRESSURE_HEADACHE": "NO"},
	})
	assert.NoError(t, err)
	assert.Equal(t, "CONVERGED", stepNo.State)
	assert.Equal(t, "ROUTINE_PATHWAY", stepNo.SafetyState)
	assert.Nil(t, stepNo.EmergencyAlert)
	assert.Nil(t, stepNo.NextInquiry)
	assert.True(t, stepNo.CanEarlyTerminate)

	// 分支 C: 回答 UNCERTAIN -> 安全底线抬升至 URGENT_ASSESSMENT
	stepUncertain, err := engine.Evaluate(ClinicalContext{
		Symptoms: []string{"SYM_ACUTE_CONJUNCTIVITIS_RED_EYE"},
		Answers:  map[string]string{"Q_EYE_PRESSURE_HEADACHE": "UNCERTAIN"},
	})
	assert.NoError(t, err)
	assert.Equal(t, "CONVERGED", stepUncertain.State)
	assert.Equal(t, "URGENT_ASSESSMENT", stepUncertain.SafetyState)
	if assert.NotNil(t, stepUncertain.EmergencyAlert) {
		assert.Equal(t, "URGENT", stepUncertain.EmergencyAlert.Level)
	}
}

// 6. 测试多轮问诊无死循环、红旗优先及鉴别选项引导科室切换 (如胸痛转消化内科)
func TestMultiTurnNoRepeatAndDifferentialRouting(t *testing.T) {
	engine := prepareEngine(t)

	answers := make(map[string]string)
	asked := make(map[string]bool)

	for step := 0; step < 8; step++ {
		dec, err := engine.Evaluate(ClinicalContext{
			Symptoms: []string{"SYM_CHEST_PAIN"},
			Answers:  answers,
		})
		assert.NoError(t, err)
		if dec.NextInquiry == nil {
			assert.Equal(t, "CONVERGED", dec.State)
			break
		}
		qID := dec.NextInquiry.QuestionID
		assert.False(t, asked[qID], "题目 %s 不应被重复追问", qID)
		asked[qID] = true

		switch qID {
		case "Q_CHEST_RADIATION":
			answers[qID] = "LOCALIZED"
		case "Q_CHEST_COLD_SWEAT":
			answers[qID] = "NO"
		case "Q_CHEST_NATURE":
			answers[qID] = "BURNING_REFLUX"
		default:
			answers[qID] = dec.NextInquiry.Options[len(dec.NextInquiry.Options)-1].Value
		}
	}

	finalDec, err := engine.Evaluate(ClinicalContext{
		Symptoms: []string{"SYM_CHEST_PAIN"},
		Answers:  answers,
	})
	assert.NoError(t, err)
	assert.Equal(t, "ROUTINE_PATHWAY", finalDec.SafetyState)
	assert.Equal(t, "CONVERGED", finalDec.State)
	if assert.NotEmpty(t, finalDec.RankedServices) {
		assert.Equal(t, "GASTRO_INTERNAL_ASSESSMENT", finalDec.RankedServices[0].ServiceCode)
	}

	// 验证上腹痛多轮问诊无重复且快速收敛
	epiAnswers := make(map[string]string)
	epiAsked := make(map[string]bool)
	converged := false
	for step := 0; step < 6; step++ {
		dec, err := engine.Evaluate(ClinicalContext{
			Symptoms: []string{"SYM_EPIGASTRIC_PAIN"},
			Answers:  epiAnswers,
		})
		assert.NoError(t, err)
		if dec.NextInquiry == nil {
			converged = true
			break
		}
		qID := dec.NextInquiry.QuestionID
		assert.False(t, epiAsked[qID], "SYM_EPIGASTRIC_PAIN 题目 %s 重复", qID)
		epiAsked[qID] = true
		for _, opt := range dec.NextInquiry.Options {
			if !opt.IsRedFlag && !opt.IsUncertain && opt.Value != "UNCERTAIN" {
				epiAnswers[qID] = opt.Value
				break
			}
		}
		if _, ok := epiAnswers[qID]; !ok && len(dec.NextInquiry.Options) > 0 {
			epiAnswers[qID] = dec.NextInquiry.Options[0].Value
		}
	}
	assert.True(t, converged, "SYM_EPIGASTRIC_PAIN 应在6步内正常收敛")
}

// 7. 测试确定性输出（50次执行题序与科室排名完全一致）
func TestDeterministicOrdering(t *testing.T) {
	engine := prepareEngine(t)
	ctx := ClinicalContext{
		Symptoms: []string{"SYM_CHEST_PAIN", "SYM_EPIGASTRIC_PAIN"},
	}

	first, err := engine.Evaluate(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, first.NextInquiry)

	for i := 0; i < 50; i++ {
		cur, err := engine.Evaluate(ctx)
		assert.NoError(t, err)
		assert.Equal(t, first.NextInquiry.QuestionID, cur.NextInquiry.QuestionID)
		assert.Equal(t, len(first.RankedServices), len(cur.RankedServices))
		for j := range first.RankedServices {
			assert.Equal(t, first.RankedServices[j].ServiceCode, cur.RankedServices[j].ServiceCode)
			assert.Equal(t, first.RankedServices[j].Score, cur.RankedServices[j].Score)
		}
	}
}

// 8. 测试跨题泛用值隔离、运行时 is_red_flag 兜底与 URGENT 提前跳过常规鉴别题
func TestCrossQuestionIsolationAndFallbackSafety(t *testing.T) {
	snap := NewGraphSnapshot("2.1.0")
	snap.Services["OPHTHALMOLOGY_ASSESSMENT"] = &ServiceNode{
		ServiceCode: "OPHTHALMOLOGY_ASSESSMENT",
		ServiceName: "眼科诊疗服务",
	}
	snap.Symptoms["SYM_SYNTH_EYE"] = &SymptomNode{
		ConceptCode:        "SYM_SYNTH_EYE",
		ConceptName:        "合成眼红测试症状",
		DefaultServiceCode: "OPHTHALMOLOGY_ASSESSMENT",
	}
	snap.Diseases["DIS_SYNTH_1"] = &DiseaseNode{
		DiseaseCode:     "DIS_SYNTH_1",
		DiseaseName:     "急性结膜炎",
		DeptServiceCode: "OPHTHALMOLOGY_ASSESSMENT",
	}
	snap.AddEdge(&GraphEdge{
		SourceType:   "SYMPTOM",
		SourceCode:   "SYM_SYNTH_EYE",
		TargetType:   "DISEASE",
		TargetCode:   "DIS_SYNTH_1",
		RelationType: "POSSIBLE_DISEASE",
		Weight:       0.6,
	})
	snap.Questions["Q_EYE_RED_URGENT"] = &QuestionNode{
		QuestionID:         "Q_EYE_RED_URGENT",
		Title:              "是否伴剧烈眼痛及虹视？",
		ApplicableSymptoms: []string{"SYM_SYNTH_EYE"},
		Type:               "SINGLE_CHOICE",
		Category:           "RED_FLAG",
		Options: []InquiryOption{
			{Label: "是，伴眼痛虹视", Value: "YES", IsRedFlag: true, ServiceHint: "OPHTHALMOLOGY_ASSESSMENT"},
			{Label: "否", Value: "NO", IsRedFlag: false},
			{Label: "不确定", Value: "UNCERTAIN", IsUncertain: true},
		},
	}
	snap.Questions["Q_BENIGN_YES"] = &QuestionNode{
		QuestionID:         "Q_BENIGN_YES",
		Title:              "是否佩戴过隐形眼镜？",
		ApplicableSymptoms: []string{"SYM_SYNTH_EYE"},
		Type:               "SINGLE_CHOICE",
		Category:           "DIFFERENTIAL",
		Options: []InquiryOption{
			{Label: "是", Value: "YES", IsRedFlag: false, ServiceHint: "OPHTHALMOLOGY_ASSESSMENT"},
			{Label: "否", Value: "NO", IsRedFlag: false},
		},
	}
	snap.SafetyRules["RULE_SYNTH_GLAUCOMA"] = &SafetyRuleNode{
		RuleID:          "RULE_SYNTH_GLAUCOMA",
		Name:            "急性闭角型青光眼预警",
		Level:           "URGENT",
		SafetyState:     "URGENT_ASSESSMENT",
		PrimarySymptoms: []string{"SYM_SYNTH_EYE"},
		TriggerCondition: map[string]interface{}{
			"any_of_answers": []interface{}{
				map[string]interface{}{"question_id": "Q_EYE_RED_URGENT", "value": "YES"},
			},
		},
		Title:       "警惕高眼压青光眼",
		WarningText: "需尽快眼科测眼压处理",
	}

	pipe := NewInferencePipeline(snap, DefaultPipelineConfig())

	// 场景 A: 回答了 Q_BENIGN_YES = YES，Q_EYE_RED_URGENT = NO -> 绝不能触发 Q_EYE_RED_URGENT 的 YES 规则
	decBenign, err := pipe.Execute(ClinicalContext{
		Symptoms: []string{"SYM_SYNTH_EYE"},
		Answers: map[string]string{
			"Q_BENIGN_YES":     "YES",
			"Q_EYE_RED_URGENT": "NO",
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "ROUTINE_PATHWAY", decBenign.SafetyState)
	assert.Nil(t, decBenign.EmergencyAlert)
	assert.Equal(t, "CONVERGED", decBenign.State)

	// 场景 B: 回答了 Q_EYE_RED_URGENT = YES (URGENT 规则) 且仅剩常规题 Q_BENIGN_YES -> 跳过常规题直接收敛
	decUrgent, err := pipe.Execute(ClinicalContext{
		Symptoms: []string{"SYM_SYNTH_EYE"},
		Answers: map[string]string{
			"Q_EYE_RED_URGENT": "YES",
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "URGENT_ASSESSMENT", decUrgent.SafetyState)
	assert.Equal(t, "CONVERGED", decUrgent.State)
	assert.Nil(t, decUrgent.NextInquiry, "命中 URGENT 且无剩余 CRITICAL 红旗题时应跳过常规鉴别题")

	// 场景 C: 删除规则仅靠 option.IsRedFlag=true，运行时兜底也必须将安全态提升至 URGENT_ASSESSMENT
	snapNoRule := NewGraphSnapshot("2.1.0")
	snapNoRule.Symptoms["SYM_SYNTH_EYE"] = snap.Symptoms["SYM_SYNTH_EYE"]
	snapNoRule.Questions["Q_EYE_RED_URGENT"] = snap.Questions["Q_EYE_RED_URGENT"]
	snapNoRule.Services["OPHTHALMOLOGY_ASSESSMENT"] = snap.Services["OPHTHALMOLOGY_ASSESSMENT"]
	pipeNoRule := NewInferencePipeline(snapNoRule, DefaultPipelineConfig())

	decFallback, err := pipeNoRule.Execute(ClinicalContext{
		Symptoms: []string{"SYM_SYNTH_EYE"},
		Answers:  map[string]string{"Q_EYE_RED_URGENT": "YES"},
	})
	assert.NoError(t, err)
	assert.Equal(t, "URGENT_ASSESSMENT", decFallback.SafetyState)
	if assert.NotNil(t, decFallback.EmergencyAlert) {
		assert.Equal(t, "RULE_FALLBACK_RED_FLAG", decFallback.EmergencyAlert.RuleID)
	}
}

func TestRuntimeHyperparametersAndManifestOverride(t *testing.T) {
	genDB, tcmDB := getTestPaths()
	tmpRoot := t.TempDir()
	kbsDir := filepath.Join(tmpRoot, "kbs")
	runtimeDir := filepath.Join(tmpRoot, "data", "triage", "runtime")
	require.NoError(t, os.MkdirAll(kbsDir, 0755))
	require.NoError(t, os.MkdirAll(runtimeDir, 0755))

	// 1. 构建默认 kbs/triage.kb.yao 与运行态覆盖文件 data/triage/runtime/triage.runtime.kb.json
	baseManifest := map[string]interface{}{
		"name":           "Base Test KB",
		"version":        "2.1.0",
		"active_version": "v2.1.0",
		"storage":        "federated_sqlite_graph",
		"layers": []map[string]interface{}{
			{"id": "overlay_tcm", "name": "TCM", "db_file": tcmDB, "type": "OVERLAY", "priority": 200, "enabled": true},
			{"id": "base_general", "name": "General", "db_file": genDB, "type": "BASE", "priority": 100, "enabled": true},
		},
		"runtime": map[string]interface{}{
			"max_steps":               5,
			"confidence_threshold":    0.85,
			"allow_early_convergence": true,
			"spreading_activation":    map[string]interface{}{"decay": 0.2, "max_distance": 2, "threshold": 0.35},
			"dual_track":              map[string]interface{}{"enable_tcm": true, "prefer_specialist_clinic": true},
			"audit_persistence":       true,
		},
	}
	baseBytes, err := json.MarshalIndent(baseManifest, "", "  ")
	require.NoError(t, err)
	packagedPath := filepath.Join(kbsDir, "triage.kb.yao")
	require.NoError(t, os.WriteFile(packagedPath, baseBytes, 0644))

	// 运行态清单尚不存在时，应回退到 kbs/triage.kb.yao
	assert.Equal(t, packagedPath, ResolveEffectiveDSLPath(tmpRoot, ""))

	// 写入运行态持久化覆盖清单：验证其优先级高于打包文件，且禁用 prefer_specialist_clinic、调整 max_steps=1、audit_persistence=false、高扩散阈值
	runtimeManifest := map[string]interface{}{
		"name":           "Runtime Override KB",
		"version":        "2.1.9",
		"active_version": "v2.1.9",
		"storage":        "federated_sqlite_graph",
		"layers": []map[string]interface{}{
			{"id": "overlay_tcm", "name": "TCM", "db_file": tcmDB, "type": "OVERLAY", "priority": 200, "enabled": true},
			{"id": "base_general", "name": "General", "db_file": genDB, "type": "BASE", "priority": 100, "enabled": true},
		},
		"runtime": map[string]interface{}{
			"max_steps":               1,
			"confidence_threshold":    0.92,
			"allow_early_convergence": false,
			"spreading_activation":    map[string]interface{}{"decay": 0.2, "max_distance": 2, "threshold": 9.9},
			"dual_track":              map[string]interface{}{"enable_tcm": true, "prefer_specialist_clinic": false},
			"audit_persistence":       false,
		},
	}
	rtBytes, err := json.MarshalIndent(runtimeManifest, "", "  ")
	require.NoError(t, err)
	runtimePath := filepath.Join(runtimeDir, "triage.runtime.kb.json")
	require.NoError(t, os.WriteFile(runtimePath, rtBytes, 0644))

	assert.Equal(t, runtimePath, ResolveEffectiveDSLPath(tmpRoot, ""))
	assert.Equal(t, runtimePath, ResolveEffectiveDSLPath(tmpRoot, "kbs/triage.kb.yao"))

	engine := GetEngine()
	defer func() {
		// 恢复引擎默认状态避免影响同包内其它测试
		engine.SetAppRoot("")
		_, _ = engine.LoadFromDSLFile(packagedPath)
	}()

	snap, err := engine.LoadFromDSLFile(runtimePath)
	require.NoError(t, err)
	assert.Equal(t, "v2.1.9", snap.ActiveVersion)
	assert.Equal(t, runtimePath, snap.DSLPath)
	assert.NotEmpty(t, snap.ManifestHash)

	// Layer 按 BASE(100) 先于 OVERLAY(200) 加载
	require.Len(t, snap.LayerStatuses, 2)
	assert.Equal(t, "base_general", snap.LayerStatuses[0].ID)
	assert.Equal(t, "overlay_tcm", snap.LayerStatuses[1].ID)

	// 验证 Evaluate 应用了所有超参数：threshold=9.9 应过滤全部 SuspectedDiseases，prefer_specialist_clinic=false 不应注入 TCM_SPECIALIST 服务，audit_persistence=false，max_steps=1
	dec, err := engine.Evaluate(ClinicalContext{
		Symptoms: []string{"SYM_COUGH"},
		Answers:  map[string]string{"Q_COUGH_DURATION": "ACUTE"},
		Age:      30,
		Gender:   "FEMALE",
	})
	require.NoError(t, err)
	assert.Equal(t, "v2.1.9", dec.KBVersion)
	assert.Equal(t, snap.ManifestHash, dec.KBHash)
	assert.False(t, dec.PreferSpecialistClinic)
	assert.False(t, dec.AuditPersistence)
	assert.False(t, dec.CanEarlyTerminate, "allow_early_convergence=false 时 CanEarlyTerminate 必须为 false")
	assert.Empty(t, dec.SuspectedDiseases, "spreading_activation.threshold=9.9 应过滤低分疾病节点")
	for _, svc := range dec.RankedServices {
		assert.NotEqual(t, "TCM_SPECIALIST", svc.TrackType, "prefer_specialist_clinic=false 不应将中医专病门诊混入服务推荐列表")
	}
	assert.Equal(t, "CONVERGED", dec.State, "已答数达到 max_steps=1 时应收敛")
	assert.Nil(t, dec.NextInquiry)

	// 验证 MetadataPayload 暴露完整标准服务清单且不返回 nil
	meta := snap.MetadataPayload()
	svcs, ok := meta["services"].([]*ServiceNode)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(svcs), 30)

	// 验证 enable_tcm=false 关闭中医轨道
	runtimeManifest["runtime"].(map[string]interface{})["dual_track"] = map[string]interface{}{
		"enable_tcm":               false,
		"prefer_specialist_clinic": false,
	}
	rtBytes2, _ := json.MarshalIndent(runtimeManifest, "", "  ")
	require.NoError(t, os.WriteFile(runtimePath, rtBytes2, 0644))
	_, err = engine.LoadFromDSLFile(runtimePath)
	require.NoError(t, err)
	decNoTCM, err := engine.Evaluate(ClinicalContext{Symptoms: []string{"SYM_COUGH"}})
	require.NoError(t, err)
	assert.Nil(t, decNoTCM.TCM, "dual_track.enable_tcm=false 时不应输出中医双轨结果")
}
