package triagekb

import (
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func ensurePlaybookDraftReadyLocked(appRoot string) (string, string, error) {
	genDraft, tcmDraft, err := ensureDraftInitializedLocked(appRoot, false)
	if err != nil {
		return "", "", err
	}
	if err := ensureDraftSchemaAndSeeds(appRoot, genDraft, tcmDraft); err != nil {
		return "", "", err
	}
	return genDraft, tcmDraft, nil
}

func decodeProcessMapArg(proc *process.Process, idx int) map[string]interface{} {
	if proc.NumOfArgs() <= idx || proc.Args[idx] == nil {
		return map[string]interface{}{}
	}
	if m, ok := proc.Args[idx].(map[string]interface{}); ok {
		return m
	}
	if raw, err := json.Marshal(proc.Args[idx]); err == nil {
		var out map[string]interface{}
		if json.Unmarshal(raw, &out) == nil && out != nil {
			return out
		}
	}
	return map[string]interface{}{}
}

func remarshalAsStruct(src interface{}, dst interface{}) error {
	raw, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dst)
}

func extractExpectedRevision(rawMap map[string]interface{}, fallback int) int {
	if rawMap == nil {
		return fallback
	}
	if v, ok := rawMap["expected_draft_revision"]; ok {
		if parsed := toInt(v, 0); parsed > 0 {
			return parsed
		}
	}
	if v, ok := rawMap["draft_revision"]; ok {
		if parsed := toInt(v, 0); parsed > 0 {
			return parsed
		}
	}
	return fallback
}

// checkPlaybookOwnershipConflict 防止双写：拦截通过旧底表 CRUD 接口修改/删除已受剧本卡托管的症状、Q_PB_* 题目、RULE_PB_* / RULE_CRIT_THUNDERCLAP 安全规则
func checkPlaybookOwnershipConflict(genDBPath, entityType, key string, payload map[string]interface{}) error {
	if strings.TrimSpace(genDBPath) == "" {
		return nil
	}
	normalizedType := strings.ToLower(strings.TrimSpace(entityType))
	if normalizedType != "symptoms" && normalizedType != "questions" && normalizedType != "safety_rules" {
		return nil
	}

	db, err := openSQLiteRO(genDBPath)
	if err != nil {
		return nil
	}
	defer db.Close()

	var tableExists int
	_ = db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name='playbook_cards'`).Scan(&tableExists)
	if tableExists == 0 {
		return nil
	}

	hasPlaybookCard := func(code string) bool {
		code = strings.TrimSpace(code)
		if code == "" {
			return false
		}
		var cnt int
		if err := db.QueryRow(`SELECT COUNT(1) FROM playbook_cards WHERE concept_code = ?`, code).Scan(&cnt); err == nil && cnt > 0 {
			return true
		}
		return false
	}

	extractSymptomListFromMap := func(m map[string]interface{}, field, jsonField string) []string {
		if m == nil {
			return nil
		}
		if rawList, ok := m[field]; ok && rawList != nil {
			return parseJSONInterfaceSlice(rawList)
		}
		if rawStr := strings.TrimSpace(toString(m[jsonField])); rawStr != "" {
			var arr []interface{}
			if json.Unmarshal([]byte(rawStr), &arr) == nil {
				return parseJSONInterfaceSlice(arr)
			}
		}
		return nil
	}

	switch normalizedType {
	case "symptoms":
		code := strings.TrimSpace(key)
		if code == "" && payload != nil {
			code = strings.TrimSpace(toString(payload["concept_code"]))
		}
		if hasPlaybookCard(code) {
			return newCompileError(409, "主诉「%s」已由分诊剧本卡托管，请在「分诊剧本卡」工作台编辑，禁止直接改写底表", code)
		}

	case "questions":
		qID := strings.TrimSpace(key)
		if qID == "" && payload != nil {
			qID = strings.TrimSpace(toString(payload["question_id"]))
		}
		if strings.HasPrefix(qID, "Q_PB_") {
			return newCompileError(409, "题目「%s」由分诊剧本卡编译器自动生成并托管，禁止直接编辑或删除", qID)
		}
		for _, symCode := range extractSymptomListFromMap(payload, "applicable_symptoms", "applicable_symptoms_json") {
			if hasPlaybookCard(symCode) {
				return newCompileError(409, "题目「%s」关联了分诊剧本卡「%s」，请通过剧本卡编辑，禁止绕过编译器直接修改", qID, symCode)
			}
		}
		if qID != "" {
			var appJSON string
			if db.QueryRow(`SELECT COALESCE(applicable_symptoms_json, '[]') FROM questions WHERE question_id = ?`, qID).Scan(&appJSON) == nil {
				var arr []interface{}
				if json.Unmarshal([]byte(appJSON), &arr) == nil {
					for _, symCode := range parseJSONInterfaceSlice(arr) {
						if hasPlaybookCard(symCode) {
							return newCompileError(409, "题目「%s」在草稿中已关联分诊剧本卡「%s」，请通过剧本卡工作台编辑或删除", qID, symCode)
						}
					}
				}
			}
		}

	case "safety_rules":
		rID := strings.TrimSpace(key)
		if rID == "" && payload != nil {
			rID = strings.TrimSpace(toString(payload["rule_id"]))
		}
		if strings.HasPrefix(rID, "RULE_PB_") || rID == "RULE_CRIT_THUNDERCLAP" {
			return newCompileError(409, "安全规则「%s」受分诊剧本卡编译器或系统雷击安全门禁托管，禁止直接编辑或删除", rID)
		}
		for _, symCode := range extractSymptomListFromMap(payload, "primary_symptoms", "primary_symptoms_json") {
			if hasPlaybookCard(symCode) {
				return newCompileError(409, "安全规则「%s」关联了分诊剧本卡「%s」，请通过剧本卡工作台修改", rID, symCode)
			}
		}
		if rID != "" {
			var priJSON string
			if db.QueryRow(`SELECT COALESCE(primary_symptoms_json, '[]') FROM safety_rules WHERE rule_id = ?`, rID).Scan(&priJSON) == nil {
				var arr []interface{}
				if json.Unmarshal([]byte(priJSON), &arr) == nil {
					for _, symCode := range parseJSONInterfaceSlice(arr) {
						if hasPlaybookCard(symCode) {
							return newCompileError(409, "安全规则「%s」在草稿中已关联分诊剧本卡「%s」，请通过剧本卡工作台修改或删除", rID, symCode)
						}
					}
				}
			}
		}
	}

	return nil
}

// ProcessKBPlaybooksList 列出分诊剧本卡列表与分层统计
func ProcessKBPlaybooksList(proc *process.Process) interface{} {
	argMap := decodeProcessMapArg(proc, 0)
	var filter PlaybookListFilter
	if len(argMap) > 0 {
		_ = remarshalAsStruct(argMap, &filter)
	}

	appRoot := resolveAdminAppRoot()
	adminMu.Lock()
	genDraft, _, err := ensurePlaybookDraftReadyLocked(appRoot)
	adminMu.Unlock()
	if err != nil {
		exception.New(err.Error(), httpCodeFromError(err, 500)).Throw()
	}

	res, err := ListPlaybookCards(genDraft, filter)
	if err != nil {
		exception.New(err.Error(), httpCodeFromError(err, 500)).Throw()
	}
	return res
}

// ProcessKBPlaybooksGet 获取单张分诊剧本卡完整配置
func ProcessKBPlaybooksGet(proc *process.Process) interface{} {
	var conceptCode string
	if proc.NumOfArgs() > 0 {
		if s, ok := proc.Args[0].(string); ok {
			conceptCode = strings.TrimSpace(s)
		} else {
			m := decodeProcessMapArg(proc, 0)
			conceptCode = strings.TrimSpace(toString(m["concept_code"]))
		}
	}
	if conceptCode == "" {
		exception.New("concept_code 必填", 400).Throw()
	}

	appRoot := resolveAdminAppRoot()
	adminMu.Lock()
	genDraft, _, err := ensurePlaybookDraftReadyLocked(appRoot)
	adminMu.Unlock()
	if err != nil {
		exception.New(err.Error(), httpCodeFromError(err, 500)).Throw()
	}

	card, err := GetPlaybookCard(genDraft, conceptCode)
	if err != nil {
		exception.New(err.Error(), httpCodeFromError(err, 404)).Throw()
	}
	return map[string]interface{}{
		"card":           card,
		"draft_revision": card.DraftRevision,
	}
}

// ProcessKBPlaybooksSave 单事务保存并编译单张分诊剧本卡（带 draft_revision 乐观锁）
func ProcessKBPlaybooksSave(proc *process.Process) interface{} {
	rawMap := decodeProcessMapArg(proc, 0)
	if len(rawMap) == 0 {
		exception.New("剧本卡请求体不能为空", 400).Throw()
	}

	var card PlaybookCard
	if err := remarshalAsStruct(rawMap, &card); err != nil {
		exception.New("剧本卡数据格式错误: "+err.Error(), 400).Throw()
	}
	expectedRev := extractExpectedRevision(rawMap, card.DraftRevision)

	appRoot := resolveAdminAppRoot()
	adminMu.Lock()
	defer adminMu.Unlock()

	genDraft, tcmDraft, err := ensurePlaybookDraftReadyLocked(appRoot)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	compiled, err := CompilePlaybookCard(genDraft, tcmDraft, card, expectedRev)
	if err != nil {
		exception.New(err.Error(), httpCodeFromError(err, 400)).Throw()
	}
	return map[string]interface{}{
		"success":        true,
		"card":           compiled,
		"draft_revision": compiled.DraftRevision,
	}
}

// ProcessKBPlaybooksBatchDisposition 执行 2A 三桶批量处置
func ProcessKBPlaybooksBatchDisposition(proc *process.Process) interface{} {
	rawMap := decodeProcessMapArg(proc, 0)
	if len(rawMap) == 0 {
		exception.New("批量处置请求体不能为空", 400).Throw()
	}

	var req BatchDispositionRequest
	if err := remarshalAsStruct(rawMap, &req); err != nil {
		exception.New("批量处置参数错误: "+err.Error(), 400).Throw()
	}
	req.ExpectedDraftRev = extractExpectedRevision(rawMap, req.ExpectedDraftRev)

	appRoot := resolveAdminAppRoot()
	adminMu.Lock()
	defer adminMu.Unlock()

	genDraft, tcmDraft, err := ensurePlaybookDraftReadyLocked(appRoot)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	res, err := BatchDispositionPlaybooks(genDraft, tcmDraft, req)
	if err != nil {
		exception.New(err.Error(), httpCodeFromError(err, 400)).Throw()
	}
	return res
}

// ProcessKBPlaybooksImport 执行第 6 节离线标准剧本批量导入（支持 dry_run）
func ProcessKBPlaybooksImport(proc *process.Process) interface{} {
	rawMap := decodeProcessMapArg(proc, 0)
	if len(rawMap) == 0 {
		exception.New("导入请求体不能为空", 400).Throw()
	}

	var payload PlaybookImportPayload
	if err := remarshalAsStruct(rawMap, &payload); err != nil {
		exception.New("导入数据解析失败: "+err.Error(), 400).Throw()
	}
	operator := strings.TrimSpace(toString(rawMap["operator"]))
	if proc.NumOfArgs() > 1 {
		if op := strings.TrimSpace(proc.ArgsString(1)); op != "" {
			operator = op
		}
	}

	appRoot := resolveAdminAppRoot()
	adminMu.Lock()
	defer adminMu.Unlock()

	genDraft, tcmDraft, err := ensurePlaybookDraftReadyLocked(appRoot)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	res, err := ImportPlaybooksBatch(genDraft, tcmDraft, payload.DryRun, payload, operator)
	if err != nil {
		exception.New(err.Error(), httpCodeFromError(err, 400)).Throw()
	}
	return res
}

// ProcessKBQuestionPacksList 列出全部标准化问诊包及其引用计数
func ProcessKBQuestionPacksList(proc *process.Process) interface{} {
	appRoot := resolveAdminAppRoot()
	adminMu.Lock()
	genDraft, _, err := ensurePlaybookDraftReadyLocked(appRoot)
	adminMu.Unlock()
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	packs, rev, err := ListQuestionPacks(genDraft)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return map[string]interface{}{
		"packs":          packs,
		"total":          len(packs),
		"draft_revision": rev,
	}
}

// ProcessKBQuestionPacksSave 保存通用问诊包模板并单事务级联重编所有引用它的剧本卡
func ProcessKBQuestionPacksSave(proc *process.Process) interface{} {
	rawMap := decodeProcessMapArg(proc, 0)
	if len(rawMap) == 0 {
		exception.New("问诊包请求体不能为空", 400).Throw()
	}

	var pack QuestionPack
	if err := remarshalAsStruct(rawMap, &pack); err != nil {
		exception.New("问诊包格式错误: "+err.Error(), 400).Throw()
	}
	expectedRev := extractExpectedRevision(rawMap, 0)

	appRoot := resolveAdminAppRoot()
	adminMu.Lock()
	defer adminMu.Unlock()

	genDraft, tcmDraft, err := ensurePlaybookDraftReadyLocked(appRoot)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	saved, cascaded, newRev, err := SaveQuestionPackAndCascade(genDraft, tcmDraft, pack, expectedRev)
	if err != nil {
		exception.New(err.Error(), httpCodeFromError(err, 400)).Throw()
	}
	return map[string]interface{}{
		"success":        true,
		"pack":           saved,
		"cascaded_count": cascaded,
		"draft_revision": newRev,
	}
}

// ProcessKBSandboxRun E3 右侧实时患者沙盒推演（直接加载 Staging 草稿热快照）
func ProcessKBSandboxRun(proc *process.Process) interface{} {
	rawMap := decodeProcessMapArg(proc, 0)
	var req SandboxRunRequest
	if len(rawMap) > 0 {
		if err := remarshalAsStruct(rawMap, &req); err != nil {
			exception.New("沙盒请求参数错误: "+err.Error(), 400).Throw()
		}
	}
	appRoot := resolveAdminAppRoot()
	res, err := RunSandbox(appRoot, req)
	if err != nil {
		exception.New(err.Error(), httpCodeFromError(err, 500)).Throw()
	}
	return res
}

// ProcessKBSandboxGoldenDiff E3 对回归基线集执行发布前前后对比差异检测
func ProcessKBSandboxGoldenDiff(proc *process.Process) interface{} {
	appRoot := resolveAdminAppRoot()
	res, err := RunGoldenDiff(appRoot)
	if err != nil {
		exception.New(err.Error(), httpCodeFromError(err, 500)).Throw()
	}
	return res
}

var _ = (*sql.DB)(nil)

