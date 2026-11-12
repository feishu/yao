package triagekb

import (
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/process"
)

func TestPlaybookBootstrapSchemaAndSecurityIsolation(t *testing.T) {
	tmpRoot := setupIsolatedKBWorkspace(t)
	RegisterProcesses()
	RegisterAdminProcesses()

	initRes, err := process.New("triage.kb.init_draft").Exec()
	require.NoError(t, err)
	assert.NotNil(t, initRes)

	genDraft, tcmDraft := resolveDraftDBPaths(tmpRoot)

	// 1. 验证 SYM_TB_* 降级为 TB_DEFERRED，不阻断核心发布门禁
	listTBRes, err := process.New("triage.kb.playbooks.list", map[string]interface{}{
		"triage_role": "TB_DEFERRED",
		"page":        1,
		"page_size":   50,
	}).Exec()
	require.NoError(t, err)
	tbMap := listTBRes.(map[string]interface{})
	assert.GreaterOrEqual(t, tbMap["total"].(int), 100)

	// 2. 验证 20 套标准化问诊包已初始化 Seed
	packsRes, err := process.New("triage.kb.question_packs.list").Exec()
	require.NoError(t, err)
	packsMap := packsRes.(map[string]interface{})
	packs := packsMap["packs"].([]QuestionPack)
	assert.GreaterOrEqual(t, len(packs), 20)

	// 3. 验证 legacy 问题已自动补全 OPT_UNCERTAIN 与 slot_key，且 RULE_CRIT_THUNDERCLAP 已入库
	genDB, err := openSQLiteRO(genDraft)
	require.NoError(t, err)
	defer genDB.Close()

	var missingUncertain int
	err = genDB.QueryRow(`SELECT COUNT(1) FROM questions WHERE options_json NOT LIKE '%UNCERTAIN%'`).Scan(&missingUncertain)
	require.NoError(t, err)
	assert.Equal(t, 0, missingUncertain, "Bootstrap 后所有存量题目都应具备 UNCERTAIN 选项")

	var thunderTrig string
	err = genDB.QueryRow(`SELECT trigger_condition_json FROM safety_rules WHERE rule_id = 'RULE_CRIT_THUNDERCLAP'`).Scan(&thunderTrig)
	require.NoError(t, err)
	assert.Contains(t, thunderTrig, "SYM_HEADACHE_THUNDERCLAP")

	// 4. 验证中医库 SYNDROME_TO_CLINIC 2跳边已初始化
	tcmDB, err := openSQLiteRO(tcmDraft)
	require.NoError(t, err)
	defer tcmDB.Close()
	var synToClinic int
	err = tcmDB.QueryRow(`SELECT COUNT(1) FROM tcm_edges WHERE relation_type = 'SYNDROME_TO_CLINIC'`).Scan(&synToClinic)
	require.NoError(t, err)
	assert.Greater(t, synToClinic, 0)

	// 5. 验证幂等性：再次执行 Bootstrap 不应重复创建或报错
	rwGen, err := openSQLiteRW(genDraft)
	require.NoError(t, err)
	defer rwGen.Close()
	rwTCM, err := openSQLiteRW(tcmDraft)
	require.NoError(t, err)
	defer rwTCM.Close()
	created2, err := BootstrapPlaybooksFromLegacy(rwGen, rwTCM)
	require.NoError(t, err)
	assert.Equal(t, 0, created2)

	// 6. 验证黄金基线门禁默认通过（legacy backlog 仅输出 warnings，不发生冷启动死锁）
	valRes, err := process.New("triage.kb.validate", "draft").Exec()
	require.NoError(t, err)
	report := valRes.(*KBGateReport)
	assert.True(t, report.Pass, "冷启动核心门禁应通过: %v", report.BlockingErrors)
	assert.NotEmpty(t, report.Warnings)
}

func TestPlaybookCompileTxIsolationOptimisticLockAndAntiDualWrite(t *testing.T) {
	tmpRoot := setupIsolatedKBWorkspace(t)
	RegisterProcesses()
	RegisterAdminProcesses()

	_, err := process.New("triage.kb.init_draft").Exec()
	require.NoError(t, err)
	genDraft, tcmDraft := resolveDraftDBPaths(tmpRoot)

	// 1. 获取当前 draft_revision
	listRes, err := process.New("triage.kb.playbooks.list", map[string]interface{}{"page": 1, "page_size": 5}).Exec()
	require.NoError(t, err)
	rev1 := listRes.(map[string]interface{})["draft_revision"].(int)

	// 2. 非法服务名应触发单事务回滚，不写入脏数据、不自增 draft_revision
	_, badErr := CompilePlaybookCard(genDraft, tcmDraft, PlaybookCard{
		ConceptCode:        "SYM_PB_INVALID_SVC_TEST",
		ConceptName:        "测试非法专科",
		TriageRole:         TriageRoleCoreComplaint,
		SkipFollowupMode:   SkipFollowupRequired,
		DefaultServiceCode: "EMERGENCY_RESCUE", // 非法不在 30 标准服务内
		AttachedPackCodes:  []string{"PACK_PAIN_DURATION"},
		ReviewStatus:       ReviewStatusApproved,
	}, rev1)
	require.Error(t, badErr)
	var ce *CompileError
	require.True(t, errors.As(badErr, &ce))
	assert.Equal(t, 400, ce.Code)

	roDB, err := openSQLiteRO(genDraft)
	require.NoError(t, err)
	var existCnt int
	_ = roDB.QueryRow(`SELECT COUNT(1) FROM playbook_cards WHERE concept_code = 'SYM_PB_INVALID_SVC_TEST'`).Scan(&existCnt)
	_ = roDB.Close()
	assert.Equal(t, 0, existCnt, "编译失败必须整事务回滚")

	// 3. 编译两张共享相同 PACK_PAIN_DURATION 的剧本卡，验证卡作用域 Q_PB_* 隔离互不覆盖
	cardA, err := CompilePlaybookCard(genDraft, tcmDraft, PlaybookCard{
		ConceptCode:        "SYM_PB_ALPHA_TEST",
		ConceptName:        "测试症状甲",
		TriageRole:         TriageRoleCoreComplaint,
		SkipFollowupMode:   SkipFollowupRequired,
		BodyPart:           "HEAD",
		ColloquialAliases:  []string{"甲不适", "脑壳甲痛"},
		DefaultServiceCode: "NEURO_INTERNAL_ASSESSMENT",
		AttachedPackCodes:  []string{"PACK_PAIN_DURATION"},
		TextbookCitation: PlaybookTextbookCitation{
			TextbookSource:  "《内科学》第9版",
			EvidenceSummary: "甲测试症状神经内科排查要点。",
		},
		ReviewStatus: ReviewStatusApproved,
		UpdatedBy:    "tester",
	}, rev1)
	require.NoError(t, err)
	assert.Greater(t, cardA.DraftRevision, rev1)

	cardB, err := CompilePlaybookCard(genDraft, tcmDraft, PlaybookCard{
		ConceptCode:        "SYM_PB_BETA_TEST",
		ConceptName:        "测试症状乙",
		TriageRole:         TriageRoleCoreComplaint,
		SkipFollowupMode:   SkipFollowupRequired,
		BodyPart:           "ABDOMEN",
		ColloquialAliases:  []string{"乙肚子疼"},
		DefaultServiceCode: "GASTRO_INTERNAL_ASSESSMENT",
		AttachedPackCodes:  []string{"PACK_PAIN_DURATION"},
		CustomQuestions: []PlaybookQuestionTemplate{
			{
				SlotKey:  "beta_specific",
				Title:    "是否进食油腻后加重？",
				Category: "DIFFERENTIAL",
				Type:     "SINGLE_CHOICE",
				Options: []PlaybookQuestionOption{
					{OptionKey: "GREASY_YES", Label: "油腻后明显右上腹痛", IsRedFlag: true, ServiceHint: "HEPATOBILIARY_SURGERY_ASSESSMENT", BoostWeight: 1.2},
					{OptionKey: "GREASY_NO", Label: "无明显诱因"},
				},
			},
		},
		RedFlagRules: []PlaybookRedFlagRule{
			{
				RuleSuffix:          "BILIARY_ACUTE",
				Title:               "急性胆道急腹症排查",
				Level:               "URGENT",
				SafetyState:         "URGENT_ASSESSMENT",
				TriggerMode:         TriggerModeOptionAnswer,
				BindSlotKey:         "beta_specific",
				BindOptionKey:       "GREASY_YES",
				RecommendedDeptName: "肝胆胰外科",
				WarningText:         "油腻后剧烈右上腹痛警惕急性胆囊炎/胆绞痛",
				GuideText:           "建议今日肝胆胰外科或急诊外科评估",
			},
		},
		ReviewStatus: ReviewStatusApproved,
		UpdatedBy:    "tester",
	}, cardA.DraftRevision)
	require.NoError(t, err)
	assert.Greater(t, cardB.DraftRevision, cardA.DraftRevision)

	ro2, err := openSQLiteRO(genDraft)
	require.NoError(t, err)
	defer ro2.Close()

	var qAID, qBID, qBOptJ string
	err = ro2.QueryRow(`SELECT question_id FROM questions WHERE question_id = 'Q_PB_SYM_PB_ALPHA_TEST_ONSET_ACUTE'`).Scan(&qAID)
	require.NoError(t, err)
	err = ro2.QueryRow(`SELECT question_id FROM questions WHERE question_id = 'Q_PB_SYM_PB_BETA_TEST_ONSET_ACUTE'`).Scan(&qBID)
	require.NoError(t, err)
	assert.NotEqual(t, qAID, qBID, "同问诊包在不同卡编译必须生成独立 Q_PB_<CONCEPT>_<SLOT>")

	err = ro2.QueryRow(`SELECT options_json FROM questions WHERE question_id = 'Q_PB_SYM_PB_BETA_TEST_BETA_SPECIFIC'`).Scan(&qBOptJ)
	require.NoError(t, err)
	assert.Contains(t, qBOptJ, "OPT_UNCERTAIN", "未手写不确定选项时编译器应自动补齐 OPT_UNCERTAIN")

	var trigJSON string
	err = ro2.QueryRow(`SELECT trigger_condition_json FROM safety_rules WHERE rule_id = 'RULE_PB_SYM_PB_BETA_TEST_BILIARY_ACUTE'`).Scan(&trigJSON)
	require.NoError(t, err)
	assert.Contains(t, trigJSON, "Q_PB_SYM_PB_BETA_TEST_BETA_SPECIFIC")
	assert.Contains(t, trigJSON, "OPT_BETA_SPECIFIC_GREASY_YES")

	// 4. 乐观锁冲突检查（传入过期 draft_revision 返回 409）
	_, lockErr := CompilePlaybookCard(genDraft, tcmDraft, *cardA, rev1)
	require.Error(t, lockErr)
	require.True(t, errors.As(lockErr, &ce))
	assert.Equal(t, 409, ce.Code)

	// 5. 防双写约束检查：禁止通过旧底层 CRUD 接口修改或删除已受剧本卡托管的症状 / Q_PB_* / RULE_PB_* / RULE_CRIT_THUNDERCLAP
	_, dualSymErr := process.New("triage.kb.save_entity", "symptoms", map[string]interface{}{
		"concept_code":         "SYM_PB_ALPHA_TEST",
		"concept_name":         "绕过编译器修改",
		"default_service_code": "NEURO_INTERNAL_ASSESSMENT",
	}).Exec()
	require.Error(t, dualSymErr)
	assert.Contains(t, dualSymErr.Error(), "409")

	_, dualQErr := process.New("triage.kb.delete_entity", "questions", "Q_PB_SYM_PB_ALPHA_TEST_ONSET_ACUTE").Exec()
	require.Error(t, dualQErr)
	assert.Contains(t, dualQErr.Error(), "409")

	_, dualRErr := process.New("triage.kb.delete_entity", "safety_rules", "RULE_PB_SYM_PB_BETA_TEST_BILIARY_ACUTE").Exec()
	require.Error(t, dualRErr)
	assert.Contains(t, dualRErr.Error(), "409")

	_, dualThunderErr := process.New("triage.kb.delete_entity", "safety_rules", "RULE_CRIT_THUNDERCLAP").Exec()
	require.Error(t, dualThunderErr)
	assert.Contains(t, dualThunderErr.Error(), "409")

	// 6. 问诊包修改单事务级联重编所有引用它的卡
	packs, revNow, err := ListQuestionPacks(genDraft)
	require.NoError(t, err)
	var onsetPack QuestionPack
	for _, p := range packs {
		if p.PackCode == "PACK_PAIN_DURATION" {
			onsetPack = p
			break
		}
	}
	require.NotEmpty(t, onsetPack.PackCode)
	onsetPack.QuestionsTemplate[0].Title = "这次不适症状持续多久了？（级联更新验证）"
	savedPack, cascadedCount, newRev, err := SaveQuestionPackAndCascade(genDraft, tcmDraft, onsetPack, revNow)
	require.NoError(t, err)
	assert.Equal(t, 2, cascadedCount, "应同时级联重编 SYM_PB_ALPHA_TEST 与 SYM_PB_BETA_TEST 两张卡")
	assert.Greater(t, newRev, revNow)
	assert.Equal(t, onsetPack.VersionRev+1, savedPack.VersionRev)

	var newQTitle string
	_ = ro2.QueryRow(`SELECT title FROM questions WHERE question_id = 'Q_PB_SYM_PB_ALPHA_TEST_ONSET_ACUTE'`).Scan(&newQTitle)
	assert.Contains(t, newQTitle, "级联更新验证")

	// 7. 严格门禁校验：若某新版剧本卡为 needs_review=1 且设为 CORE_COMPLAINT，或 REQUIRED 且 0 题目，应阻断发布
	rwDB, err := openSQLiteRW(genDraft)
	require.NoError(t, err)
	_, err = rwDB.Exec(`UPDATE symptoms SET needs_review = 1 WHERE concept_code = 'SYM_PB_ALPHA_TEST'`)
	require.NoError(t, err)
	_ = rwDB.Close()

	valFailRes, err := process.New("triage.kb.validate", "draft").Exec()
	require.NoError(t, err)
	valFail := valFailRes.(*KBGateReport)
	assert.False(t, valFail.Pass)
	assert.True(t, func() bool {
		for _, b := range valFail.BlockingErrors {
			if strings.Contains(b, "SYM_PB_ALPHA_TEST") {
				return true
			}
		}
		return false
	}())
}

func TestPlaybookImportValidationBatchDispositionAndSandbox(t *testing.T) {
	tmpRoot := setupIsolatedKBWorkspace(t)
	RegisterProcesses()
	RegisterAdminProcesses()

	_, err := process.New("triage.kb.init_draft").Exec()
	require.NoError(t, err)
	genDraft, tcmDraft := resolveDraftDBPaths(tmpRoot)

	// 1. 测试 PlaybookImportV1 校验拦截：使用不存在科室 EMERGENCY_RESCUE 必须被 dry_run 拒绝
	badImportRes, badImportErr := ImportPlaybooksBatch(genDraft, tcmDraft, true, PlaybookImportPayload{
		ImportBatchID: "batch_bad_01",
		SchemaVersion: "PlaybookImportV1",
		DryRun:        true,
		Items: []PlaybookImportItem{
			{
				ConceptCode:        "SYM_TB_SKIN_MUCOUS_MEMBRANE_BLEEDING",
				ConceptName:        "皮肤黏膜出血",
				TriageRole:         TriageRoleCoreComplaint,
				SkipFollowupMode:   SkipFollowupRequired,
				DefaultServiceCode: "EMERGENCY_RESCUE", // 非法编码
				ReviewStatus:       ReviewStatusApproved,
			},
		},
	}, "importer_qa")
	if badImportErr == nil {
		// 若返回结构体则 valid 必须为 false / rejected_count > 0
		assert.Equal(t, false, badImportRes["valid"])
	}

	// 2. 测试合法 PlaybookImportV1：dry_run 不落盘，正式提交写入并解除 TB_DEFERRED
	validItem := PlaybookImportItem{
		ConceptCode:        "SYM_TB_SKIN_MUCOUS_MEMBRANE_BLEEDING",
		ConceptName:        "皮肤黏膜出血",
		TriageRole:         TriageRoleCoreComplaint,
		SkipFollowupMode:   SkipFollowupRequired,
		BodyPart:           "SKIN",
		ColloquialAliases:  []string{"身上莫名青紫瘀斑", "青紫瘀斑", "刷牙老流血止不住", "皮下出血点"},
		GenderLimit:        "ALL",
		AgeGroupLimit:      "ALL",
		DefaultServiceCode: "HEMATOLOGY_ASSESSMENT",
		BackupServiceCodes: []string{"RHEUMATOLOGY_ASSESSMENT", "DERMATOLOGY_ASSESSMENT"},
		AttachedPackCodes:  []string{"PACK_PAIN_DURATION"},
		CustomQuestions: []PlaybookQuestionTemplate{
			{
				SlotKey:  "bleeding_site_scope",
				Title:    "出血是仅局限于牙龈/皮肤小瘀点，还是伴有黑便、血尿或鼻出血难止？",
				Category: "RED_FLAG",
				Type:     "SINGLE_CHOICE",
				Options: []PlaybookQuestionOption{
					{OptionKey: "MULTI_CAVITY", Label: "多部位广泛出血或伴头晕乏力", IsRedFlag: true, ServiceHint: "HEMATOLOGY_ASSESSMENT", BoostWeight: 1.5},
					{OptionKey: "LOCAL_SPOT", Label: "仅局部少量出血点/牙龈轻度渗血", ServiceHint: "HEMATOLOGY_ASSESSMENT", BoostWeight: 0.9},
				},
			},
		},
		RedFlagRules: []PlaybookRedFlagRule{
			{
				RuleSuffix:          "SYSTEMIC_BLEED",
				Title:               "多部位黏膜出血警惕急性出凝血障碍",
				Level:               "CRITICAL",
				SafetyState:         "EMERGENCY_NOW",
				TriggerMode:         TriggerModeOptionAnswer,
				BindSlotKey:         "bleeding_site_scope",
				BindOptionKey:       "MULTI_CAVITY",
				RecommendedDeptName: "急诊内科 / 血液内科",
				Call120:             false,
				WarningText:         "多部位黏膜广泛出血伴头晕警惕血小板极度减少或凝血危象，请立即急诊评估",
				GuideText:           "避免剧烈活动或磕碰，立即前往急诊查血常规及凝血功能",
			},
		},
		TextbookCitation: PlaybookTextbookCitation{
			TextbookSource:  "《诊断学》第9版",
			Chapter:         "第一篇 常见症状",
			Page:            "p.38",
			EvidenceSummary: "广泛皮肤黏膜出血点瘀斑伴腔道出血需优先血液内科急诊排查血小板及凝血机制异常。",
		},
		ReviewStatus: ReviewStatusApproved,
	}

	dryRes, err := ImportPlaybooksBatch(genDraft, tcmDraft, true, PlaybookImportPayload{
		ImportBatchID: "batch_dry_01",
		SchemaVersion: "PlaybookImportV1",
		DryRun:        true,
		Items:         []PlaybookImportItem{validItem},
	}, "importer_qa")
	require.NoError(t, err)
	assert.Equal(t, true, dryRes["dry_run"])

	// 确保 dry_run 没有把 SYM_TB_SKIN_MUCOUS_MEMBRANE_BLEEDING 改成 CORE_COMPLAINT
	cardBefore, err := GetPlaybookCard(genDraft, "SYM_TB_SKIN_MUCOUS_MEMBRANE_BLEEDING")
	require.NoError(t, err)
	assert.Equal(t, TriageRoleTBDeferred, cardBefore.TriageRole)

	// 正式导入提交
	commitRes, err := process.New("triage.kb.playbooks.import", map[string]interface{}{
		"import_batch_id": "batch_commit_01",
		"schema_version":  "PlaybookImportV1",
		"dry_run":         false,
		"items":           []PlaybookImportItem{validItem},
	}).Exec()
	require.NoError(t, err)
	assert.Equal(t, 1, commitRes.(map[string]interface{})["imported_count"])

	cardAfter, err := GetPlaybookCard(genDraft, "SYM_TB_SKIN_MUCOUS_MEMBRANE_BLEEDING")
	require.NoError(t, err)
	assert.Equal(t, TriageRoleCoreComplaint, cardAfter.TriageRole)
	assert.False(t, cardAfter.NeedsReviewFlag)

	// 3. 测试 2A 三桶批量处置（DIRECT_SINGLE_HOP）
	dispRes, err := process.New("triage.kb.playbooks.batch_disposition", map[string]interface{}{
		"concept_codes":      []string{"SYM_EYE_TEARING"},
		"action":             "SET_FOLLOWUP_MODE",
		"skip_followup_mode": "DIRECT_SINGLE_HOP",
		"direct_triage_note": "迎风流泪症状指向明确，直接推荐眼科常规评估",
	}).Exec()
	require.NoError(t, err)
	assert.Equal(t, 1, dispRes.(map[string]interface{})["updated_count"])

	// 4. 测试 E3 实时患者沙盒 (Staging 热快照) + skipped_questions 踪迹
	sbRes, err := process.New("triage.kb.sandbox_run", map[string]interface{}{
		"utterance": "45岁患者，皮肤莫名青紫瘀斑2周，没有发烧",
		"patient_context": map[string]interface{}{
			"age":    45,
			"gender": "UNKNOWN",
		},
	}).Exec()
	require.NoError(t, err)
	sbMap := sbRes.(map[string]interface{})
	sbDecision := sbMap["decision"].(*TriageDecision)
	require.NotEmpty(t, sbDecision.RankedServices)
	assert.Equal(t, "HEMATOLOGY_ASSESSMENT", sbDecision.RankedServices[0].ServiceCode)
	skipped := sbMap["skipped_questions"].([]SkippedQuestionTrace)
	var skippedJSON []byte
	skippedJSON, _ = json.Marshal(skipped)
	assert.Contains(t, string(skippedJSON), "Q_PB_SYM_TB_SKIN_MUCOUS_MEMBRANE_BLEEDING_ONSET_ACUTE")

	// 5. 测试 E3 Golden Regression Diff
	diffRes, err := process.New("triage.kb.sandbox_golden_diff").Exec()
	require.NoError(t, err)
	diffMap := diffRes.(map[string]interface{})
	assert.GreaterOrEqual(t, diffMap["total_cases"].(int), 20)
	assert.True(t, diffMap["all_critical_passed"].(bool), "核心安全回归集不得出现降级回归: %v", diffMap["diffs"])
	assert.FileExists(t, genDraft)
	_ = filepath.Base(tmpRoot)

	_ = (*sql.DB)(nil)
}
