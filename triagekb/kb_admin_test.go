package triagekb

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/process"
)

func setupIsolatedKBWorkspace(t *testing.T) string {
	t.Helper()
	serviceRoot := "/Users/L/Desktop/Code/yao_projects/syd/service"
	tmpRoot := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(tmpRoot, "kbs"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpRoot, "connectors"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpRoot, "data", "triage", "kb"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpRoot, "data", "triage", "seed"), 0755))

	// 复制主库与种子 symptoms.json
	require.NoError(t, copyFileSafe(
		filepath.Join(serviceRoot, "data/triage/kb/kb_triage_general.db"),
		filepath.Join(tmpRoot, "data/triage/kb/kb_triage_general.db"),
	))
	require.NoError(t, copyFileSafe(
		filepath.Join(serviceRoot, "data/triage/kb/kb_triage_tcm.db"),
		filepath.Join(tmpRoot, "data/triage/kb/kb_triage_tcm.db"),
	))
	require.NoError(t, copyFileSafe(
		filepath.Join(serviceRoot, "data/triage/seed/symptoms.json"),
		filepath.Join(tmpRoot, "data/triage/seed/symptoms.json"),
	))

	// 写入测试用 dictionaries_seed.json
	dictSeed := map[string]interface{}{
		"hot_symptoms": []map[string]interface{}{
			{"symptom_code": "SYM_COUGH", "label": "咳嗽", "icon": "lungs", "body_part": "CHEST", "sort_order": 1, "is_active": true},
		},
		"body_parts": []map[string]interface{}{
			{"part_code": "HEAD", "label": "头面部", "symptom_codes": []string{"SYM_HEADACHE_TENSION"}, "sort_order": 1, "is_active": true},
			{"part_code": "CHEST", "label": "胸部", "symptom_codes": []string{"SYM_COUGH", "SYM_CHEST_PAIN"}, "sort_order": 2, "is_active": true},
		},
		"keyword_synonyms": map[string][]string{
			"普通内科": {"普内科", "内科门诊"},
		},
		"fallback_keywords": map[string][]string{
			"GENERAL_INTERNAL_ASSESSMENT": {"普内科", "全科医学科", "全科门诊"},
		},
	}
	dictBytes, _ := json.MarshalIndent(dictSeed, "", "  ")
	require.NoError(t, os.WriteFile(filepath.Join(tmpRoot, "data/triage/seed/dictionaries_seed.json"), dictBytes, 0644))

	// 写入 kbs/triage.kb.yao 与草稿连接器
	dslBytes, _ := json.MarshalIndent(defaultDSLTemplate(), "", "  ")
	require.NoError(t, os.WriteFile(filepath.Join(tmpRoot, "kbs/triage.kb.yao"), dslBytes, 0644))

	for name, fileRel := range map[string]string{
		"triage_general_draft": "./data/triage/runtime/kb_triage_general.draft.db",
		"triage_tcm_draft":     "./data/triage/runtime/kb_triage_tcm.draft.db",
	} {
		conn := map[string]interface{}{
			"name":    name,
			"version": "1.0.0",
			"type":    "sqlite3",
			"options": map[string]string{"file": fileRel},
		}
		b, _ := json.MarshalIndent(conn, "", "  ")
		require.NoError(t, os.WriteFile(filepath.Join(tmpRoot, "connectors", name+".conn.yao"), b, 0644))
	}

	RegisterProcesses()
	GetEngine().SetAppRoot(tmpRoot)
	_, err := process.New("triage.engine.load").Exec()
	require.NoError(t, err)

	t.Cleanup(func() {
		GetEngine().SetAppRoot("")
		_, _ = GetEngine().LoadFromDSLFile(filepath.Join(serviceRoot, "kbs/triage.kb.yao"))
	})
	return tmpRoot
}

func TestKBAdminDraftCRUDGatePublishRollback(t *testing.T) {
	tmpRoot := setupIsolatedKBWorkspace(t)

	// 1. 初始化 Staging 草稿库并校验 schema 自动升级与 seed 灌入
	initRes, err := process.New("triage.kb.init_draft", map[string]interface{}{"force": true}).Exec()
	require.NoError(t, err)
	initMap := initRes.(map[string]interface{})
	assert.True(t, initMap["success"].(bool))
	counts := initMap["counts"].(map[string]int)
	assert.GreaterOrEqual(t, counts["standard_services"], 30)
	assert.Equal(t, 112, counts["symptoms_needs_review"], "应自动从 symptoms.json 回填 112 条 needs_review=1")
	assert.GreaterOrEqual(t, counts["kb_hot_symptoms"], 1)
	assert.GreaterOrEqual(t, counts["kb_body_parts"], 2)

	// 清零可能残留在基线 DB 中的废弃假 /registration/specialist-clinic/ 链接以便后续正常发布
	tcmListRes, err := process.New("triage.kb.list_entities", "tcm_specialist_clinics", map[string]interface{}{"page_size": 100}).Exec()
	require.NoError(t, err)
	tcmList := tcmListRes.(map[string]interface{})["items"].([]map[string]interface{})
	hasFakeURL := false
	for _, clinic := range tcmList {
		if bURL, _ := clinic["booking_url"].(string); bURL != "" {
			hasFakeURL = true
			_, err := process.New("triage.kb.save_entity", "tcm_specialist_clinics", map[string]interface{}{
				"clinic_code":          clinic["clinic_code"],
				"clinic_name":          clinic["clinic_name"],
				"dept_name":            clinic["dept_name"],
				"expert_titles":        clinic["expert_titles"],
				"speciality_tags":      clinic["speciality_tags"],
				"booking_url":          "",
				"target_dept_keywords": []string{clinic["dept_name"].(string), "中医科"},
			}).Exec()
			require.NoError(t, err)
		}
	}
	if hasFakeURL {
		// 测试门禁拦截保存假 /registration/specialist-clinic/ 链接
		_, fakeSaveErr := process.New("triage.kb.save_entity", "tcm_specialist_clinics", map[string]interface{}{
			"clinic_code": "TCM_CLINIC_FAKE_TEST",
			"clinic_name": "测试专病门诊",
			"dept_name":   "中医脾胃科",
			"booking_url": "/registration/specialist-clinic/TCM_CLINIC_FAKE_TEST",
		}).Exec()
		assert.Error(t, fakeSaveErr, "保存废弃假路由应被阻断")
	}

	// 2. 测试 symptoms CRUD + DEFAULT_SERVICE & INDICATES 边事务级联 + 确定性 SHA-256 DIS_ 编码
	saveSymPayload := map[string]interface{}{
		"concept_code":         "SYM_TEST_VERTIGO",
		"concept_name":         "测试性眩晕",
		"body_part":            "HEAD",
		"default_service_code": "NEURO_INTERNAL_ASSESSMENT",
		"needs_review":         false,
		"indicative_diseases": []map[string]interface{}{
			{
				"disease_name":    "良性阵发性位置性眩晕(耳石症)",
				"dept_service_code": "ENT_ASSESSMENT",
				"weight":          0.91,
				"is_critical":     false,
				"evidence_quote":  "体位改变诱发短暂旋转性眩晕，优先耳鼻咽喉头颈外科",
			},
		},
	}
	saveRes1, err := process.New("triage.kb.save_entity", "symptoms", saveSymPayload).Exec()
	require.NoError(t, err)
	assert.Equal(t, 1, saveRes1.(map[string]interface{})["edge_sync"].(map[string]int)["indicates_edges"])

	detailRes1, err := process.New("triage.kb.get_entity", "symptoms", "SYM_TEST_VERTIGO").Exec()
	require.NoError(t, err)
	detailMap1 := detailRes1.(map[string]interface{})
	ind1 := detailMap1["indicative_diseases"].([]map[string]interface{})
	require.Len(t, ind1, 1)
	disCode1 := ind1[0]["disease_code"].(string)
	assert.Contains(t, disCode1, "DIS_")
	assert.Len(t, disCode1, 16)

	// 再次保存相同疾病名称，DIS_ 编码必须完全一致（禁止随机 hash 漂移）
	_, err = process.New("triage.kb.save_entity", "symptoms", saveSymPayload).Exec()
	require.NoError(t, err)
	detailRes2, err := process.New("triage.kb.get_entity", "symptoms", "SYM_TEST_VERTIGO").Exec()
	require.NoError(t, err)
	ind2 := detailRes2.(map[string]interface{})["indicative_diseases"].([]map[string]interface{})
	assert.Equal(t, disCode1, ind2[0]["disease_code"].(string), "同名鉴别疾病多次保存应生成完全确定的 DIS_<HEX12>")

	// 3. 测试门禁阻断：悬空红旗选项无对应 safety_rules 必须阻断 publish
	_, err = process.New("triage.kb.save_entity", "questions", map[string]interface{}{
		"question_id":         "Q_TEST_VERTIGO_RED",
		"title":               "是否突发言语不清或单侧肢体无力？",
		"applicable_symptoms": []string{"SYM_TEST_VERTIGO"},
		"type":                "SINGLE_CHOICE",
		"category":            "RED_FLAG",
		"options": []map[string]interface{}{
			{"label": "是", "value": "STROKE_SIGN", "is_red_flag": true, "service_hint": "NEURO_INTERNAL_ASSESSMENT"},
			{"label": "否", "value": "NO"},
		},
	}).Exec()
	require.NoError(t, err)

	valBlockedRes, err := process.New("triage.kb.validate", "draft").Exec()
	require.NoError(t, err)
	valBlocked := valBlockedRes.(*KBGateReport)
	assert.False(t, valBlocked.Pass, "悬空红旗选项必须使门禁失败")

	_, pubBlockedErr := process.New("triage.kb.publish", map[string]interface{}{"version_label": "v2.2.bad"}).Exec()
	assert.Error(t, pubBlockedErr, "门禁未通过时禁止发布")

	// 4. 测试只读表删除保护与关联问题删除保护，随后补齐安全规则让门禁通过
	_, delSvcErr := process.New("triage.kb.delete_entity", "services", "GENERAL_INTERNAL_ASSESSMENT").Exec()
	assert.Error(t, delSvcErr)
	_, delDisErr := process.New("triage.kb.delete_entity", "diseases", disCode1).Exec()
	assert.Error(t, delDisErr)

	ruleRes, err := process.New("triage.kb.save_entity", "safety_rules", map[string]interface{}{
		"rule_id":          "RULE_TEST_VERTIGO_STROKE",
		"name":             "后循环卒中测试预警",
		"level":            "CRITICAL",
		"safety_state":     "EMERGENCY_NOW",
		"primary_symptoms": []string{"SYM_TEST_VERTIGO"},
		"trigger_condition": map[string]interface{}{
			"any_of_answers": []map[string]string{
				{"question_id": "Q_TEST_VERTIGO_RED", "value": "STROKE_SIGN"},
			},
		},
		"title":        "高度警惕急性脑卒中风险",
		"warning_text": "请立即拨打120急救电话前往卒中中心急诊科",
		"emergency_action": map[string]interface{}{
			"call_120":              true,
			"recommended_dept_name": "急诊神经科",
		},
	}).Exec()
	require.NoError(t, err)
	assert.Equal(t, 1, ruleRes.(map[string]interface{})["edge_sync"].(map[string]int)["triggers_safety_edges"])

	// 被安全规则引用的问题禁止删除
	_, delQErr := process.New("triage.kb.delete_entity", "questions", "Q_TEST_VERTIGO_RED").Exec()
	assert.Error(t, delQErr)

	// 门禁应通过
	valPassRes, err := process.New("triage.kb.validate", "draft").Exec()
	require.NoError(t, err)
	valPass := valPassRes.(*KBGateReport)
	assert.True(t, valPass.Pass, "补齐安全规则闭环后门禁应通过: %v", valPass.BlockingErrors)

	// 5. 发布第 1 个不可变快照版本并热加载到引擎
	pubRes1, err := process.New("triage.kb.publish", map[string]interface{}{
		"version_label": "v2.2.1",
		"operator":      "platform_qa",
		"note":          "首次测试发布",
	}).Exec()
	require.NoError(t, err)
	pub1Map := pubRes1.(map[string]interface{})
	rel1 := pub1Map["release"].(ReleaseRecord)
	assert.Equal(t, "v2.2.1", pub1Map["active_version"])
	assert.FileExists(t, filepath.Join(tmpRoot, rel1.GeneralDBFile[2:]))
	assert.FileExists(t, filepath.Join(tmpRoot, rel1.TCMDBFile[2:]))
	assert.FileExists(t, filepath.Join(tmpRoot, RuntimeManifestRelPath))

	// 验证激活引擎已即时生效新症状与安全规则
	evalRes, err := process.New("triage.engine.evaluate", map[string]interface{}{
		"symptoms": []string{"SYM_TEST_VERTIGO"},
		"answers":  map[string]string{"Q_TEST_VERTIGO_RED": "STROKE_SIGN"},
	}).Exec()
	require.NoError(t, err)
	dec1 := evalRes.(*TriageDecision)
	assert.Equal(t, "EMERGENCY", dec1.State)
	assert.Equal(t, "v2.2.1", dec1.KBVersion)

	// 6. 调整草稿再次发布 v2.2.2，然后一键回滚到 v2.2.1
	_, err = process.New("triage.kb.save_entity", "hot_symptoms", map[string]interface{}{
		"symptom_code": "SYM_TEST_VERTIGO",
		"label":        "测试眩晕",
		"icon":         "brain",
		"body_part":    "HEAD",
		"sort_order":   99,
		"is_active":    true,
	}).Exec()
	require.NoError(t, err)

	pubRes2, err := process.New("triage.kb.publish", map[string]interface{}{
		"version_label": "v2.2.2",
		"operator":      "platform_qa",
		"note":          "第二次发布",
	}).Exec()
	require.NoError(t, err)
	assert.Equal(t, "v2.2.2", pubRes2.(map[string]interface{})["active_version"])

	rbRes, err := process.New("triage.kb.rollback", map[string]interface{}{
		"release_id": rel1.ReleaseID,
		"operator":   "platform_qa",
	}).Exec()
	require.NoError(t, err)
	rbMap := rbRes.(map[string]interface{})
	assert.Equal(t, "v2.2.1", rbMap["active_version"])
	assert.Equal(t, rel1.ManifestHash, rbMap["manifest_hash"])

	relsRes, err := process.New("triage.kb.releases").Exec()
	require.NoError(t, err)
	relsList := relsRes.(map[string]interface{})["releases"].([]ReleaseRecord)
	require.GreaterOrEqual(t, len(relsList), 3)
	assert.Equal(t, "ROLLBACK", relsList[0].Action)

	// 7. 测试运行时参数热调优与越界校验
	_, badParamErr := process.New("triage.kb.update_runtime_params", map[string]interface{}{
		"runtime": map[string]interface{}{"max_steps": 99},
	}).Exec()
	assert.Error(t, badParamErr)

	rtRes, err := process.New("triage.kb.update_runtime_params", map[string]interface{}{
		"operator": "platform_qa",
		"note":     "关闭直跳专病",
		"runtime": map[string]interface{}{
			"max_steps": 4,
			"dual_track": map[string]interface{}{
				"enable_tcm":               true,
				"prefer_specialist_clinic": false,
			},
		},
	}).Exec()
	require.NoError(t, err)
	rtCfg := rtRes.(map[string]interface{})["pipeline_cfg"].(PipelineConfig)
	assert.Equal(t, 4, rtCfg.MaxSteps)
	assert.False(t, rtCfg.PreferSpecialistClinic)

	// 8. 测试标准化 seed/*.json 与 .db ZIP 导出
	expRes, err := process.New("triage.kb.export_seed", map[string]interface{}{"source": "draft"}).Exec()
	require.NoError(t, err)
	expMap := expRes.(map[string]interface{})
	zipRel := expMap["zip_path"].(string)
	zipAbs := filepath.Join(tmpRoot, zipRel[2:])
	assert.FileExists(t, zipAbs)

	zr, err := zip.OpenReader(zipAbs)
	require.NoError(t, err)
	defer zr.Close()
	entrySet := make(map[string]bool)
	for _, f := range zr.File {
		entrySet[f.Name] = true
	}
	assert.True(t, entrySet["seed/symptoms.json"])
	assert.True(t, entrySet["seed/tcm_seed.json"])
	assert.True(t, entrySet["seed/dictionaries_seed.json"])
	assert.True(t, entrySet["kb/kb_triage_general.db"])
	assert.True(t, entrySet["kb/kb_triage_tcm.db"])
}
