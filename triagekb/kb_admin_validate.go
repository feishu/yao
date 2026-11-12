package triagekb

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

// KBGateReport 知识库发布门禁核验报告
type KBGateReport struct {
	Pass           bool           `json:"pass"`
	BlockingErrors []string       `json:"blocking_errors"`
	Warnings       []string       `json:"warnings"`
	Counts         map[string]int `json:"counts"`
}

// ProcessKBValidate 执行草稿库或当前激活库发布前全量完整性门禁校验
func ProcessKBValidate(proc *process.Process) interface{} {
	target := "draft"
	if proc.NumOfArgs() > 0 && strings.TrimSpace(proc.ArgsString(0)) != "" {
		target = strings.TrimSpace(proc.ArgsString(0))
	}

	appRoot := resolveAdminAppRoot()
	adminMu.Lock()
	var genPath, tcmPath string
	var err error
	if target == "active" {
		genPath, tcmPath = resolveActiveDBPaths(appRoot)
	} else {
		genPath, tcmPath, err = ensureDraftInitializedLocked(appRoot, false)
	}
	adminMu.Unlock()
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	report, err := validateDatabases(genPath, tcmPath)
	if err != nil {
		exception.New(fmt.Sprintf("执行门禁校验异常: %v", err), 500).Throw()
	}
	return report
}

func validateDatabases(genPath, tcmPath string) (*KBGateReport, error) {
	genDB, err := openSQLiteRO(genPath)
	if err != nil {
		return nil, err
	}
	defer genDB.Close()

	report := &KBGateReport{
		Pass:           true,
		BlockingErrors: make([]string, 0),
		Warnings:       make([]string, 0),
		Counts:         collectDraftCounts(genPath, tcmPath),
	}

	// 1. 服务字典全集
	svcCodes := make(map[string]bool)
	sRows, err := genDB.Query(`SELECT service_code FROM standard_services`)
	if err != nil {
		return nil, err
	}
	for sRows.Next() {
		var code string
		if sRows.Scan(&code) == nil && code != "" {
			svcCodes[code] = true
		}
	}
	sRows.Close()
	if len(svcCodes) == 0 {
		report.BlockingErrors = append(report.BlockingErrors, "[standard_services] 临床专科方向为空")
	}

	// 2. 主诉字典与分诊角色分层校验
	symCols := sqliteTableColumns(genDB, "symptoms")
	nrSelect := "0"
	if symCols["needs_review"] {
		nrSelect = "COALESCE(needs_review, 0)"
	}
	roleSelect := "'CORE_COMPLAINT'"
	if symCols["triage_role"] {
		roleSelect = "COALESCE(NULLIF(triage_role, ''), 'CORE_COMPLAINT')"
	}
	skipSelect := "'REQUIRED'"
	if symCols["skip_followup_mode"] {
		skipSelect = "COALESCE(NULLIF(skip_followup_mode, ''), 'REQUIRED')"
	}
	symRows, err := genDB.Query(fmt.Sprintf(`SELECT concept_code, concept_name, default_service_code, is_red_flag_candidate, %s, %s, %s FROM symptoms`, nrSelect, roleSelect, skipSelect))
	if err != nil {
		return nil, err
	}
	type symMeta struct {
		code     string
		name     string
		defSvc   string
		redFlag  int
		nr       int
		role     string
		skipMode string
	}
	symCodes := make(map[string]bool)
	symList := make([]symMeta, 0)
	redCandidates := make(map[string]bool)
	deferredNeedsReview := 0
	for symRows.Next() {
		var item symMeta
		if symRows.Scan(&item.code, &item.name, &item.defSvc, &item.redFlag, &item.nr, &item.role, &item.skipMode) != nil {
			continue
		}
		item.role = strings.ToUpper(strings.TrimSpace(item.role))
		if item.role == "" {
			item.role = TriageRoleCoreComplaint
		}
		item.skipMode = strings.ToUpper(strings.TrimSpace(item.skipMode))
		if item.skipMode == "" {
			item.skipMode = SkipFollowupRequired
		}
		symCodes[item.code] = true
		symList = append(symList, item)
		if item.redFlag == 1 && item.role == TriageRoleCoreComplaint {
			redCandidates[item.code] = true
		}
		if item.nr == 1 && item.role != TriageRoleCoreComplaint {
			deferredNeedsReview++
		}
		if item.role != TriageRoleArchived && !svcCodes[item.defSvc] {
			report.Warnings = append(report.Warnings, fmt.Sprintf("[symptoms] %s 默认专科 %s 不在 standard_services 中", item.code, item.defSvc))
		}
	}
	symRows.Close()
	if deferredNeedsReview > 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("[symptoms] 共有 %d 条非核心活跃主诉(TB_DEFERRED/COMPANION_ONLY/ARCHIVED)处于待复核态，不阻断核心集发布", deferredNeedsReview))
	}

	// 3. 追问题库与选项校验（仅对非空 service_hint 做白名单校验，不误杀阴性选项如 NO_FEVER/MILD）
	qRows, err := genDB.Query(`SELECT question_id, applicable_symptoms_json, options_json FROM questions`)
	if err != nil {
		return nil, err
	}
	type qMeta struct {
		appSyms map[string]bool
		optVals map[string]bool
		redOpts []string
	}
	qMap := make(map[string]qMeta)
	qCoveredSyms := make(map[string]bool)
	for qRows.Next() {
		var qID, appJ, optJ string
		if qRows.Scan(&qID, &appJ, &optJ) != nil {
			continue
		}
		var apps []string
		var opts []InquiryOption
		_ = json.Unmarshal([]byte(appJ), &apps)
		_ = json.Unmarshal([]byte(optJ), &opts)
		appSet := make(map[string]bool, len(apps))
		for _, s := range apps {
			appSet[s] = true
			qCoveredSyms[s] = true
			if !symCodes[s] {
				report.BlockingErrors = append(report.BlockingErrors, fmt.Sprintf("[questions] %s 引用未知主诉 %s", qID, s))
			}
		}
		optSet := make(map[string]bool, len(opts))
		var redVals []string
		for _, o := range opts {
			optSet[o.Value] = true
			if o.IsRedFlag {
				redVals = append(redVals, o.Value)
			}
			if o.ServiceHint != "" && !o.IsUncertain && !svcCodes[o.ServiceHint] {
				report.BlockingErrors = append(report.BlockingErrors, fmt.Sprintf("[questions] %s.%s 引用未知专科服务 %s", qID, o.Value, o.ServiceHint))
			}
		}
		qMap[qID] = qMeta{appSyms: appSet, optVals: optSet, redOpts: redVals}
	}
	qRows.Close()

	// 4. 安全规则与 4 类触发条件完整校验
	rRows, err := genDB.Query(`SELECT rule_id, level, primary_symptoms_json, trigger_condition_json FROM safety_rules`)
	if err != nil {
		return nil, err
	}
	coveredRedFlags := make(map[string]bool)
	ruleCoveredSyms := make(map[string]bool)
	for rRows.Next() {
		var rID, level, primJ, trigJ string
		if rRows.Scan(&rID, &level, &primJ, &trigJ) != nil {
			continue
		}
		upperLvl := strings.ToUpper(level)
		if upperLvl != "CRITICAL" && upperLvl != "URGENT" {
			report.BlockingErrors = append(report.BlockingErrors, fmt.Sprintf("[safety_rules] %s level 非 CRITICAL/URGENT: %s", rID, level))
		}
		var prims []string
		_ = json.Unmarshal([]byte(primJ), &prims)
		if len(prims) == 0 {
			report.BlockingErrors = append(report.BlockingErrors, fmt.Sprintf("[safety_rules] %s primary_symptoms 为空", rID))
		}
		primSet := make(map[string]bool, len(prims))
		for _, s := range prims {
			primSet[s] = true
			ruleCoveredSyms[s] = true
			if !symCodes[s] {
				report.BlockingErrors = append(report.BlockingErrors, fmt.Sprintf("[safety_rules] %s 引用未知主诉 %s", rID, s))
			}
		}
		var rawTC map[string]interface{}
		if unmErr := json.Unmarshal([]byte(trigJ), &rawTC); unmErr != nil || len(rawTC) == 0 {
			report.BlockingErrors = append(report.BlockingErrors, fmt.Sprintf("[safety_rules] %s trigger_condition_json 为空或非法（禁止无条件恒真规则）", rID))
			continue
		}
		var tc struct {
			AnyOfAnswers []struct {
				QuestionID string `json:"question_id"`
				Value      string `json:"value"`
			} `json:"any_of_answers"`
			AnyOfConcepts       []string `json:"any_of_concepts"`
			AnyOfSigns          []string `json:"any_of_signs"`
			AnyOfMedicalHistory []string `json:"any_of_medical_history"`
		}
		_ = json.Unmarshal([]byte(trigJ), &tc)
		totalConditions := len(tc.AnyOfAnswers) + len(tc.AnyOfConcepts) + len(tc.AnyOfSigns) + len(tc.AnyOfMedicalHistory)
		if totalConditions == 0 {
			report.BlockingErrors = append(report.BlockingErrors, fmt.Sprintf("[safety_rules] %s trigger_condition_json 未配置任何有效触发项", rID))
			continue
		}
		for _, ans := range tc.AnyOfAnswers {
			qm, exists := qMap[ans.QuestionID]
			if !exists {
				report.BlockingErrors = append(report.BlockingErrors, fmt.Sprintf("[safety_rules] %s 引用不存在的问题 %s", rID, ans.QuestionID))
				continue
			}
			if !qm.optVals[ans.Value] {
				report.BlockingErrors = append(report.BlockingErrors, fmt.Sprintf("[safety_rules] %s 引用 %s 不存在的选项值 %s", rID, ans.QuestionID, ans.Value))
				continue
			}
			coveredRedFlags[ans.QuestionID+"="+ans.Value] = true
			for appSym := range qm.appSyms {
				if !primSet[appSym] {
					report.BlockingErrors = append(report.BlockingErrors, fmt.Sprintf("[safety_rules] %s primary_symptoms 遗漏关联问题 %s 的适用主诉 %s", rID, ans.QuestionID, appSym))
				}
			}
		}
		for _, concept := range tc.AnyOfConcepts {
			trimmed := strings.TrimSpace(concept)
			if trimmed == "" || !symCodes[trimmed] {
				report.BlockingErrors = append(report.BlockingErrors, fmt.Sprintf("[safety_rules] %s any_of_concepts 引用未知主诉 %s", rID, concept))
				continue
			}
			ruleCoveredSyms[trimmed] = true
		}
		for _, sign := range tc.AnyOfSigns {
			if strings.TrimSpace(sign) == "" {
				report.BlockingErrors = append(report.BlockingErrors, fmt.Sprintf("[safety_rules] %s any_of_signs 包含空条目", rID))
			}
		}
		for _, hist := range tc.AnyOfMedicalHistory {
			if strings.TrimSpace(hist) == "" {
				report.BlockingErrors = append(report.BlockingErrors, fmt.Sprintf("[safety_rules] %s any_of_medical_history 包含空条目", rID))
			}
		}
	}
	rRows.Close()

	for qID, qm := range qMap {
		for _, redVal := range qm.redOpts {
			if !coveredRedFlags[qID+"="+redVal] {
				report.BlockingErrors = append(report.BlockingErrors, fmt.Sprintf("[questions] 悬空红旗选项未定义安全规则: %s=%s", qID, redVal))
			}
		}
	}

	// 4.5 核心主诉（CORE_COMPLAINT）严格发布门禁（TB_DEFERRED / COMPANION_ONLY / ARCHIVED 不阻断发布；未触达的历史 system:bootstrap 卡先报 Warning 以免冷启动死锁）
	bootstrappedUnedited := make(map[string]bool)
	if pbRows, pbErr := genDB.Query(`SELECT concept_code, COALESCE(updated_by, '') FROM playbook_cards`); pbErr == nil {
		for pbRows.Next() {
			var code, updatedBy string
			if pbRows.Scan(&code, &updatedBy) == nil && updatedBy == "system:bootstrap" {
				bootstrappedUnedited[code] = true
			}
		}
		pbRows.Close()
	}

	deferredZeroQ := 0
	legacyZeroQ := 0
	legacyRedUncovered := 0
	for _, sym := range symList {
		if sym.role != TriageRoleCoreComplaint {
			if !qCoveredSyms[sym.code] {
				deferredZeroQ++
			}
			continue
		}
		if sym.nr == 1 {
			report.BlockingErrors = append(report.BlockingErrors, fmt.Sprintf("[gate] 核心主诉 %s(%s) 仍处于待复核(needs_review=1)，禁止发布", sym.code, sym.name))
		}
		if sym.skipMode == SkipFollowupRequired && !qCoveredSyms[sym.code] {
			if bootstrappedUnedited[sym.code] {
				legacyZeroQ++
			} else {
				report.BlockingErrors = append(report.BlockingErrors, fmt.Sprintf("[gate] 核心主诉 %s(%s) 为必填追问(REQUIRED)但未关联任何题目", sym.code, sym.name))
			}
		}
		if sym.redFlag == 1 && !ruleCoveredSyms[sym.code] {
			if bootstrappedUnedited[sym.code] {
				legacyRedUncovered++
			} else {
				report.BlockingErrors = append(report.BlockingErrors, fmt.Sprintf("[gate] 高危候选核心主诉 %s(%s) 未配置安全规则", sym.code, sym.name))
			}
		}
	}
	if legacyZeroQ > 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("[coverage] 共有 %d 条历史初始化核心主诉暂未挂载追问（可在工作区 B 2A 待处置看板批量转为单跳直达/伴随症状或挂载问诊包）", legacyZeroQ))
	}
	if legacyRedUncovered > 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("[coverage] 共有 %d 条历史初始化高危候选主诉尚未挂载 TRIGGERS_SAFETY 规则", legacyRedUncovered))
	}
	if deferredZeroQ > 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("[coverage] %d 条非核心/暂缓主诉未关联题目（已隔离，不阻断核心发布）", deferredZeroQ))
	}

	// 5. 中医库校验（强制清零废弃假 /registration/specialist-clinic/* 链接）
	tcmDB, tErr := openSQLiteRO(tcmPath)
	if tErr != nil {
		report.BlockingErrors = append(report.BlockingErrors, fmt.Sprintf("[tcm] 中医库不可读取: %v", tErr))
	} else {
		defer tcmDB.Close()
		cRows, cErr := tcmDB.Query(`SELECT clinic_code, clinic_name, dept_name, COALESCE(booking_url, '') FROM tcm_specialist_clinics`)
		if cErr != nil {
			report.BlockingErrors = append(report.BlockingErrors, fmt.Sprintf("[tcm] 查询专病门诊失败: %v", cErr))
		} else {
			clinicCount := 0
			for cRows.Next() {
				var cCode, cName, deptName, bURL string
				if cRows.Scan(&cCode, &cName, &deptName, &bURL) == nil {
					clinicCount++
					if strings.Contains(bURL, "/registration/specialist-clinic/") {
						report.BlockingErrors = append(report.BlockingErrors, fmt.Sprintf("[tcm_specialist_clinics] %s(%s) 仍包含失效假路由 %s", cCode, cName, bURL))
					}
					if strings.TrimSpace(deptName) == "" {
						report.BlockingErrors = append(report.BlockingErrors, fmt.Sprintf("[tcm_specialist_clinics] %s 缺少推荐科室方向名称 dept_name", cCode))
					}
				}
			}
			cRows.Close()
			if clinicCount == 0 {
				report.BlockingErrors = append(report.BlockingErrors, "[tcm_specialist_clinics] 中医专病门诊列表为空")
			}
		}
	}

	report.Pass = len(report.BlockingErrors) == 0
	return report, nil
}

