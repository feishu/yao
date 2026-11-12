package triagekb

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/mozillazg/go-pinyin"
)

// CompileError 带 HTTP 状态码的编译器业务错误（例如 400 参数非法 / 409 乐观锁冲突）
type CompileError struct {
	Code    int
	Message string
}

func (e *CompileError) Error() string {
	return e.Message
}

func newCompileError(code int, format string, args ...interface{}) error {
	return &CompileError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

func httpCodeFromError(err error, fallback int) int {
	if err == nil {
		return 200
	}
	var ce *CompileError
	if errors.As(err, &ce) && ce.Code > 0 {
		return ce.Code
	}
	return fallback
}

// PlaybookListFilter 分诊剧本卡列表过滤条件
type PlaybookListFilter struct {
	TriageRole   string `json:"triage_role,omitempty"`
	ReviewStatus string `json:"review_status,omitempty"`
	OnlyMismatch bool   `json:"only_mismatch,omitempty"`
	OnlyZeroQ    bool   `json:"only_zero_questions,omitempty"`
	Search       string `json:"search,omitempty"`
	Page         int    `json:"page,omitempty"`
	PageSize     int    `json:"page_size,omitempty"`
}

// BatchDispositionRequest 2A 三桶批量处置请求
type BatchDispositionRequest struct {
	ConceptCodes        []string `json:"concept_codes"`
	Action              string   `json:"action"` // SET_ROLE | SET_FOLLOWUP_MODE | APPLY_MISMATCH_FIX | APPROVE_REVIEW | ATTACH_PACK | UPDATE_SERVICE
	TriageRole          string   `json:"triage_role,omitempty"`
	SkipFollowupMode    string   `json:"skip_followup_mode,omitempty"`
	DirectTriageNote    string   `json:"direct_triage_note,omitempty"`
	DefaultServiceCode  string   `json:"default_service_code,omitempty"`
	BackupServiceCodes  []string `json:"backup_service_codes,omitempty"`
	AttachPackCodes     []string `json:"attach_pack_codes,omitempty"`
	ApproveAfterFix     bool     `json:"approve_after_fix,omitempty"`
	ExpectedDraftRev    int      `json:"expected_draft_revision,omitempty"`
	Operator            string   `json:"operator,omitempty"`
}

// PlaybookImportItem 第 6 节离线教材/人工标准导入单条数据结构（PlaybookImportV1）
type PlaybookImportItem struct {
	ConceptCode        string                     `json:"concept_code"`
	ConceptName        string                     `json:"concept_name"`
	TriageRole         string                     `json:"triage_role"`
	SkipFollowupMode   string                     `json:"skip_followup_mode"`
	DirectTriageNote   string                     `json:"direct_triage_note,omitempty"`
	BodyPart           string                     `json:"body_part"`
	ColloquialAliases  []string                   `json:"colloquial_aliases"`
	GenderLimit        string                     `json:"gender_limit"`
	AgeGroupLimit      string                     `json:"age_group_limit"`
	DefaultServiceCode string                     `json:"default_service_code"`
	BackupServiceCodes []string                   `json:"backup_service_codes"`
	MismatchCorrection *PlaybookMismatchAlert     `json:"mismatch_correction,omitempty"`
	MismatchAlert      *PlaybookMismatchAlert     `json:"mismatch_alert,omitempty"`
	AttachedPackCodes  []string                   `json:"attached_pack_codes"`
	CustomQuestions    []PlaybookQuestionTemplate `json:"custom_questions"`
	RedFlags           []PlaybookRedFlagRule      `json:"red_flags"`
	RedFlagRules       []PlaybookRedFlagRule      `json:"red_flag_rules,omitempty"`
	TCMBinding         PlaybookTCMBinding         `json:"tcm_binding"`
	TextbookCitation   PlaybookTextbookCitation   `json:"textbook_citation"`
	ReviewStatus       string                     `json:"review_status"`
}

// PlaybookImportPayload 第 6 节批量导入顶层请求体
type PlaybookImportPayload struct {
	ImportBatchID string               `json:"import_batch_id"`
	SchemaVersion string               `json:"schema_version"`
	DryRun        bool                 `json:"dry_run"`
	Items         []PlaybookImportItem `json:"items"`
}

type compilerWhitelists struct {
	Services     map[string]bool
	Packs        map[string]QuestionPack
	TCMClinics   map[string]bool
	TCMSyndromes map[string]bool
	TCMTherapies map[string]bool
}

type compiledQuestionLookup struct {
	QuestionID string
	SlotKey    string
	OptionVals map[string]string // option_key -> value & option_value -> value
}

func buildChinesePinyin(name string) (string, string) {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return "", ""
	}
	args := pinyin.NewArgs()
	args.Style = pinyin.Normal
	args.Fallback = func(r rune, _ pinyin.Args) []string {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return []string{strings.ToLower(string(r))}
		}
		return nil
	}
	syllables := pinyin.Pinyin(clean, args)
	var fullParts []string
	var initParts []string
	for _, item := range syllables {
		if len(item) == 0 || item[0] == "" {
			continue
		}
		syl := strings.ToLower(strings.TrimSpace(item[0]))
		if syl == "" {
			continue
		}
		fullParts = append(fullParts, syl)
		initParts = append(initParts, string([]rune(syl)[0]))
	}
	// 若全为 ASCII/未提取出拼音，退回字符保留
	if len(fullParts) == 0 {
		var alpha []rune
		for _, r := range strings.ToLower(clean) {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				alpha = append(alpha, r)
			}
		}
		s := string(alpha)
		return s, s
	}
	return strings.Join(fullParts, ""), strings.Join(initParts, "")
}

func normalizeUpperToken(raw string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range strings.ToUpper(strings.TrimSpace(raw)) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastUnderscore = false
		} else if !lastUnderscore && b.Len() > 0 {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	res := strings.Trim(b.String(), "_")
	if res == "" {
		return "SLOT"
	}
	return res
}

func normalizeSlotKey(raw string) string {
	s := strings.TrimSpace(strings.ToLower(raw))
	var b strings.Builder
	lastUnder := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastUnder = false
		} else if !lastUnder && b.Len() > 0 {
			b.WriteByte('_')
			lastUnder = true
		}
	}
	return strings.Trim(b.String(), "_")
}

func uniqueNonEmptyStrings(items []string) []string {
	seen := make(map[string]bool, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}

func loadWhitelistsTx(genTx, tcmTx *sql.Tx) (*compilerWhitelists, error) {
	wl := &compilerWhitelists{
		Services:     make(map[string]bool),
		Packs:        make(map[string]QuestionPack),
		TCMClinics:   make(map[string]bool),
		TCMSyndromes: make(map[string]bool),
		TCMTherapies: make(map[string]bool),
	}

	sRows, err := genTx.Query(`SELECT service_code FROM standard_services`)
	if err != nil {
		return nil, fmt.Errorf("读取 standard_services 白名单失败: %w", err)
	}
	for sRows.Next() {
		var code string
		if scanErr := sRows.Scan(&code); scanErr == nil && strings.TrimSpace(code) != "" {
			wl.Services[strings.TrimSpace(code)] = true
		}
	}
	_ = sRows.Close()

	pRows, err := genTx.Query(`SELECT pack_code, pack_name, clinical_domain, description, COALESCE(questions_template_json, '[]'), COALESCE(version_rev, 1), COALESCE(updated_at, '') FROM question_packs`)
	if err != nil {
		return nil, fmt.Errorf("读取 question_packs 失败: %w", err)
	}
	for pRows.Next() {
		var p QuestionPack
		var tplJSON string
		if scanErr := pRows.Scan(&p.PackCode, &p.PackName, &p.ClinicalDomain, &p.Description, &tplJSON, &p.VersionRev, &p.UpdatedAt); scanErr == nil {
			_ = json.Unmarshal([]byte(tplJSON), &p.QuestionsTemplate)
			wl.Packs[p.PackCode] = p
		}
	}
	_ = pRows.Close()

	if tcmTx != nil {
		cRows, err := tcmTx.Query(`SELECT clinic_code FROM tcm_specialist_clinics`)
		if err == nil {
			for cRows.Next() {
				var code string
				if scanErr := cRows.Scan(&code); scanErr == nil && strings.TrimSpace(code) != "" {
					wl.TCMClinics[strings.TrimSpace(code)] = true
				}
			}
			_ = cRows.Close()
		}
		synRows, err := tcmTx.Query(`SELECT syndrome_code FROM tcm_syndromes`)
		if err == nil {
			for synRows.Next() {
				var code string
				if scanErr := synRows.Scan(&code); scanErr == nil && strings.TrimSpace(code) != "" {
					wl.TCMSyndromes[strings.TrimSpace(code)] = true
				}
			}
			_ = synRows.Close()
		}
		thRows, err := tcmTx.Query(`SELECT therapy_code FROM tcm_therapies`)
		if err == nil {
			for thRows.Next() {
				var code string
				if scanErr := thRows.Scan(&code); scanErr == nil && strings.TrimSpace(code) != "" {
					wl.TCMTherapies[strings.TrimSpace(code)] = true
				}
			}
			_ = thRows.Close()
		}
	}

	return wl, nil
}

func validateCardAndNormalize(card *PlaybookCard, wl *compilerWhitelists) error {
	card.ConceptCode = strings.TrimSpace(card.ConceptCode)
	card.ConceptName = strings.TrimSpace(card.ConceptName)
	if card.ConceptCode == "" || card.ConceptName == "" {
		return newCompileError(400, "concept_code 与 concept_name 必填")
	}

	if card.TriageRole == "" {
		card.TriageRole = TriageRoleCoreComplaint
	}
	switch card.TriageRole {
	case TriageRoleCoreComplaint, TriageRoleCompanionOnly, TriageRoleTBDeferred, TriageRoleArchived:
	default:
		return newCompileError(400, "非法 triage_role: %s（仅支持 CORE_COMPLAINT / COMPANION_ONLY / TB_DEFERRED / ARCHIVED）", card.TriageRole)
	}

	if card.SkipFollowupMode == "" {
		card.SkipFollowupMode = SkipFollowupRequired
	}
	switch card.SkipFollowupMode {
	case SkipFollowupRequired, SkipFollowupDirectSingleHop:
	default:
		return newCompileError(400, "非法 skip_followup_mode: %s（仅支持 REQUIRED / DIRECT_SINGLE_HOP）", card.SkipFollowupMode)
	}

	if card.BodyPart == "" {
		card.BodyPart = "GENERAL"
	}
	if card.GenderLimit == "" {
		card.GenderLimit = "ALL"
	}
	switch card.GenderLimit {
	case "ALL", "MALE", "FEMALE":
	default:
		return newCompileError(400, "非法 gender_limit: %s", card.GenderLimit)
	}
	if card.AgeGroupLimit == "" {
		card.AgeGroupLimit = "ALL"
	}
	card.AgeGroupLimit = normalizeAgeGroupTag(card.AgeGroupLimit)

	if card.ReviewStatus == "" {
		card.ReviewStatus = ReviewStatusNeedsReview
	}
	switch card.ReviewStatus {
	case ReviewStatusApproved, ReviewStatusNeedsReview, ReviewStatusAISuggested:
	default:
		return newCompileError(400, "非法 review_status: %s", card.ReviewStatus)
	}

	card.DefaultServiceCode = strings.TrimSpace(card.DefaultServiceCode)
	if !wl.Services[card.DefaultServiceCode] {
		return newCompileError(400, "首选临床服务 %s 不在 30 项标准服务白名单内，严禁写入", card.DefaultServiceCode)
	}

	card.BackupServiceCodes = uniqueNonEmptyStrings(card.BackupServiceCodes)
	var cleanBackups []string
	for _, svc := range card.BackupServiceCodes {
		if svc == card.DefaultServiceCode {
			continue
		}
		if !wl.Services[svc] {
			return newCompileError(400, "备选临床服务 %s 不在 30 项标准服务白名单内", svc)
		}
		cleanBackups = append(cleanBackups, svc)
	}
	card.BackupServiceCodes = cleanBackups

	card.ColloquialAliases = uniqueNonEmptyStrings(card.ColloquialAliases)
	card.AttachedPackCodes = uniqueNonEmptyStrings(card.AttachedPackCodes)
	for _, packCode := range card.AttachedPackCodes {
		if _, ok := wl.Packs[packCode]; !ok {
			return newCompileError(400, "问诊包 %s 不存在于 question_packs 表中", packCode)
		}
	}

	for qIdx := range card.CustomQuestions {
		q := &card.CustomQuestions[qIdx]
		q.SlotKey = normalizeSlotKey(q.SlotKey)
		if q.SlotKey == "" {
			q.SlotKey = normalizeSlotKey(strings.TrimPrefix(q.QuestionID, "Q_"))
		}
		if q.SlotKey == "" || strings.TrimSpace(q.Title) == "" {
			return newCompileError(400, "专属追问题缺少 slot_key 或 title")
		}
		if q.Category == "" {
			q.Category = "DIFFERENTIAL"
		}
		if q.Type == "" {
			q.Type = "SINGLE_CHOICE"
		}
		for oIdx := range q.Options {
			opt := &q.Options[oIdx]
			if strings.TrimSpace(opt.Label) == "" {
				return newCompileError(400, "问题 %s 存在空选项 label", q.SlotKey)
			}
			if strings.TrimSpace(opt.ServiceHint) != "" && !wl.Services[strings.TrimSpace(opt.ServiceHint)] {
				return newCompileError(400, "问题 %s 选项 %s 的 service_hint=%s 不在 30 项标准服务白名单内", q.SlotKey, opt.Label, opt.ServiceHint)
			}
			for _, syn := range opt.TCMSyndromeHints {
				if len(wl.TCMSyndromes) > 0 && !wl.TCMSyndromes[strings.TrimSpace(syn)] {
					return newCompileError(400, "选项中医证候提示 %s 不在 tcm_syndromes 白名单内", syn)
				}
			}
		}
	}

	for rIdx := range card.RedFlagRules {
		rf := &card.RedFlagRules[rIdx]
		rf.Level = strings.ToUpper(strings.TrimSpace(rf.Level))
		if rf.Level != "CRITICAL" && rf.Level != "URGENT" {
			return newCompileError(400, "红旗规则 %s level=%s 非法（仅允许 CRITICAL / URGENT）", rf.RuleSuffix, rf.Level)
		}
		if rf.Level == "CRITICAL" {
			rf.SafetyState = "EMERGENCY_NOW"
		} else {
			rf.SafetyState = "URGENT_ASSESSMENT"
		}
		if strings.TrimSpace(rf.FallbackServiceCode) != "" && !wl.Services[strings.TrimSpace(rf.FallbackServiceCode)] {
			return newCompileError(400, "红旗规则 %s fallback_service_code=%s 不在白名单内", rf.RuleSuffix, rf.FallbackServiceCode)
		}
		if strings.TrimSpace(rf.Title) == "" || strings.TrimSpace(rf.WarningText) == "" {
			return newCompileError(400, "红旗规则 %s 的 title 与 warning_text 必填", rf.RuleSuffix)
		}
		switch rf.TriggerMode {
		case TriggerModeOptionAnswer, TriggerModeConceptDirect, TriggerModeSignMatch:
		default:
			return newCompileError(400, "红旗规则 %s 的 trigger_mode=%s 非法（仅支持 OPTION_ANSWER / CONCEPT_DIRECT / SIGN_MATCH）", rf.RuleSuffix, rf.TriggerMode)
		}
	}

	if len(wl.TCMClinics) > 0 {
		for _, clinic := range card.TCMBinding.Clinics {
			if !wl.TCMClinics[strings.TrimSpace(clinic)] {
				return newCompileError(400, "贵中医专病门诊 %s 不在 6 项在库白名单内", clinic)
			}
		}
	}
	if len(wl.TCMSyndromes) > 0 {
		for _, syn := range card.TCMBinding.Syndromes {
			if !wl.TCMSyndromes[strings.TrimSpace(syn)] {
				return newCompileError(400, "中医证候 %s 不在 tcm_syndromes 白名单内", syn)
			}
		}
	}
	if len(wl.TCMTherapies) > 0 {
		for _, th := range card.TCMBinding.Therapies {
			if !wl.TCMTherapies[strings.TrimSpace(th)] {
				return newCompileError(400, "中医治法 %s 不在 tcm_therapies 白名单内", th)
			}
		}
	}

	card.NeedsReviewFlag = card.ReviewStatus != ReviewStatusApproved
	return nil
}

func readDraftRevisionTx(genTx *sql.Tx) int {
	var rev int
	_ = genTx.QueryRow(`SELECT COALESCE(draft_revision, 1) FROM playbook_draft_meta WHERE id = 1`).Scan(&rev)
	if rev <= 0 {
		rev = 1
	}
	return rev
}

func bumpDraftRevisionTx(genTx *sql.Tx, now string) (int, error) {
	rev := readDraftRevisionTx(genTx) + 1
	if _, err := genTx.Exec(`
		INSERT INTO playbook_draft_meta (id, draft_revision, last_compiled_at, compiler_version, last_golden_diff_summary_json)
		VALUES (1, ?, ?, 'playbook-v1', '{}')
		ON CONFLICT(id) DO UPDATE SET
			draft_revision = excluded.draft_revision,
			last_compiled_at = excluded.last_compiled_at
	`, rev, now); err != nil {
		return 0, fmt.Errorf("更新草稿版本号失败: %w", err)
	}
	return rev, nil
}

// CompilePlaybookCard 单事务保存 PlaybookCard（含 draft_revision 乐观锁）并单向编译写入 Staging 底表
func CompilePlaybookCard(genPath, tcmPath string, card PlaybookCard, expectedDraftRev int) (*PlaybookCard, error) {
	genDB, err := openSQLiteRW(genPath)
	if err != nil {
		return nil, err
	}
	defer genDB.Close()

	var tcmDB *sql.DB
	if strings.TrimSpace(tcmPath) != "" {
		if tdb, tErr := openSQLiteRW(tcmPath); tErr == nil {
			tcmDB = tdb
			defer tcmDB.Close()
		}
	}

	if err := EnsurePlaybookSchema(genDB, tcmDB); err != nil {
		return nil, err
	}

	genTx, err := genDB.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = genTx.Rollback() }()

	var tcmTx *sql.Tx
	if tcmDB != nil {
		tcmTx, err = tcmDB.Begin()
		if err != nil {
			return nil, err
		}
		defer func() { _ = tcmTx.Rollback() }()
	}

	curDraftRev := readDraftRevisionTx(genTx)
	if expectedDraftRev > 0 && expectedDraftRev != curDraftRev {
		return nil, newCompileError(409, "Staging 草稿版本号已变更（当前 v%d，请求预期 v%d），请刷新后重试", curDraftRev, expectedDraftRev)
	}

	var existingVer int
	err = genTx.QueryRow(`SELECT COALESCE(version_rev, 0) FROM playbook_cards WHERE concept_code = ?`, strings.TrimSpace(card.ConceptCode)).Scan(&existingVer)
	if err == nil && card.VersionRev > 0 && card.VersionRev != existingVer {
		return nil, newCompileError(409, "剧本卡 %s 已被其他管理员修改（当前卡版本 %d，提交版本 %d）", card.ConceptCode, existingVer, card.VersionRev)
	}
	card.VersionRev = existingVer + 1
	if card.UpdatedAt == "" {
		card.UpdatedAt = time.Now().Format(time.RFC3339)
	}

	wl, err := loadWhitelistsTx(genTx, tcmTx)
	if err != nil {
		return nil, err
	}
	if err := compileCardArtifactsTx(genTx, tcmTx, &card, wl); err != nil {
		return nil, err
	}

	newDraftRev, err := bumpDraftRevisionTx(genTx, card.UpdatedAt)
	if err != nil {
		return nil, err
	}
	card.DraftRevision = newDraftRev

	if tcmTx != nil {
		if err := tcmTx.Commit(); err != nil {
			return nil, fmt.Errorf("提交中医库事务失败: %w", err)
		}
	}
	if err := genTx.Commit(); err != nil {
		return nil, fmt.Errorf("提交全科库事务失败: %w", err)
	}
	return &card, nil
}

func compileCardArtifactsTx(genTx, tcmTx *sql.Tx, card *PlaybookCard, wl *compilerWhitelists) error {
	if err := validateCardAndNormalize(card, wl); err != nil {
		return err
	}

	// 1. 写入 playbook_cards 源表
	if err := upsertPlaybookCardRowTx(genTx, card); err != nil {
		return err
	}

	// 2. 更新/写入 symptoms 底表并同步拼音与别名
	if err := compileSymptomRowTx(genTx, card); err != nil {
		return err
	}

	// 3. 展开 attached_pack_codes + custom_questions 生成卡作用域 Q_PB_* 题目
	compiledQs, err := compileCardQuestionsTx(genTx, card, wl)
	if err != nil {
		return err
	}

	// 4. 依 trigger_mode 精准编译 safety_rules 与 TRIGGERS_SAFETY 边（禁止空条件）
	if err := compileCardSafetyRulesTx(genTx, card, compiledQs); err != nil {
		return err
	}

	// 5. 编译 DEFAULT_SERVICE + BOOSTS_SERVICE 边
	if err := compileCardServiceEdgesTx(genTx, card, compiledQs); err != nil {
		return err
	}

	// 6. 编译中医 1~2 跳边 (SYMPTOM->SYNDROME->CLINIC)
	if tcmTx != nil {
		if err := compileCardTCMEdgesTx(tcmTx, card); err != nil {
			return err
		}
	}

	// 7. 同步 textbook_evidence
	if err := compileCardEvidenceTx(genTx, card); err != nil {
		return err
	}

	return nil
}

func upsertPlaybookCardRowTx(genTx *sql.Tx, card *PlaybookCard) error {
	aliasesJ := mustMarshalJSON(card.ColloquialAliases, "[]")
	backupsJ := mustMarshalJSON(card.BackupServiceCodes, "[]")
	rfJ := mustMarshalJSON(card.RedFlagRules, "[]")
	packsJ := mustMarshalJSON(card.AttachedPackCodes, "[]")
	customQsJ := mustMarshalJSON(card.CustomQuestions, "[]")
	tcmJ := mustMarshalJSON(card.TCMBinding, "{}")
	citeJ := mustMarshalJSON(card.TextbookCitation, "{}")
	mismatchJ := mustMarshalJSON(card.MismatchAlert, "{}")

	_, err := genTx.Exec(`
		INSERT INTO playbook_cards (
			concept_code, concept_name, triage_role, skip_followup_mode, direct_triage_note,
			body_part, colloquial_aliases_json, gender_limit, age_group_limit,
			default_service_code, backup_service_codes_json, red_flag_rules_json,
			attached_pack_codes_json, custom_questions_json, tcm_binding_json,
			textbook_citation_json, review_status, mismatch_alert_json,
			version_rev, updated_at, updated_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(concept_code) DO UPDATE SET
			concept_name = excluded.concept_name,
			triage_role = excluded.triage_role,
			skip_followup_mode = excluded.skip_followup_mode,
			direct_triage_note = excluded.direct_triage_note,
			body_part = excluded.body_part,
			colloquial_aliases_json = excluded.colloquial_aliases_json,
			gender_limit = excluded.gender_limit,
			age_group_limit = excluded.age_group_limit,
			default_service_code = excluded.default_service_code,
			backup_service_codes_json = excluded.backup_service_codes_json,
			red_flag_rules_json = excluded.red_flag_rules_json,
			attached_pack_codes_json = excluded.attached_pack_codes_json,
			custom_questions_json = excluded.custom_questions_json,
			tcm_binding_json = excluded.tcm_binding_json,
			textbook_citation_json = excluded.textbook_citation_json,
			review_status = excluded.review_status,
			mismatch_alert_json = excluded.mismatch_alert_json,
			version_rev = excluded.version_rev,
			updated_at = excluded.updated_at,
			updated_by = excluded.updated_by
	`,
		card.ConceptCode, card.ConceptName, card.TriageRole, card.SkipFollowupMode, card.DirectTriageNote,
		card.BodyPart, aliasesJ, card.GenderLimit, card.AgeGroupLimit,
		card.DefaultServiceCode, backupsJ, rfJ,
		packsJ, customQsJ, tcmJ,
		citeJ, card.ReviewStatus, mismatchJ,
		card.VersionRev, card.UpdatedAt, card.UpdatedBy,
	)
	if err != nil {
		return fmt.Errorf("保存 playbook_cards 失败: %w", err)
	}
	return nil
}

func compileSymptomRowTx(genTx *sql.Tx, card *PlaybookCard) error {
	var existingSynJ, existingPinyin, existingInit, existingProto, existingICDCode, existingICDTitle, existingCompsJ string
	err := genTx.QueryRow(`
		SELECT COALESCE(synonyms_json, '[]'), COALESCE(pinyin, ''), COALESCE(pinyin_initials, ''),
			COALESCE(evidence_protocol_id, ''), COALESCE(icd11_code, ''), COALESCE(icd11_title, ''),
			COALESCE(associated_complications_json, '[]')
		FROM symptoms WHERE concept_code = ?
	`, card.ConceptCode).Scan(&existingSynJ, &existingPinyin, &existingInit, &existingProto, &existingICDCode, &existingICDTitle, &existingCompsJ)

	var mergedSyns []string
	if err == nil && existingSynJ != "" {
		_ = json.Unmarshal([]byte(existingSynJ), &mergedSyns)
	}
	mergedSyns = uniqueNonEmptyStrings(append(mergedSyns, card.ColloquialAliases...))

	pinyinFull, pinyinInit := buildChinesePinyin(card.ConceptName)
	if pinyinFull == "" {
		pinyinFull = existingPinyin
	}
	if pinyinInit == "" {
		pinyinInit = existingInit
	}
	if existingCompsJ == "" {
		existingCompsJ = "[]"
	}

	isRedFlagInt := 0
	if len(card.RedFlagRules) > 0 {
		isRedFlagInt = 1
	}
	needsReviewInt := 1
	if card.ReviewStatus == ReviewStatusApproved {
		needsReviewInt = 0
	}

	textbookSource := strings.TrimSpace(card.TextbookCitation.TextbookSource)
	aliasesJ := mustMarshalJSON(card.ColloquialAliases, "[]")
	synonymsJ := mustMarshalJSON(mergedSyns, "[]")
	diffsJ := mustMarshalJSON(card.BackupServiceCodes, "[]")

	_, err = genTx.Exec(`
		INSERT INTO symptoms (
			concept_code, concept_name, body_part, pinyin, pinyin_initials,
			synonyms_json, default_service_code, differential_service_codes_json,
			is_red_flag_candidate, recommended_age_group, gender_limit,
			evidence_protocol_id, icd11_code, icd11_title, textbook_source,
			needs_review, associated_complications_json,
			triage_role, skip_followup_mode, direct_triage_note, colloquial_aliases_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(concept_code) DO UPDATE SET
			concept_name = excluded.concept_name,
			body_part = excluded.body_part,
			pinyin = excluded.pinyin,
			pinyin_initials = excluded.pinyin_initials,
			synonyms_json = excluded.synonyms_json,
			default_service_code = excluded.default_service_code,
			differential_service_codes_json = excluded.differential_service_codes_json,
			is_red_flag_candidate = excluded.is_red_flag_candidate,
			recommended_age_group = excluded.recommended_age_group,
			gender_limit = excluded.gender_limit,
			textbook_source = CASE WHEN excluded.textbook_source <> '' THEN excluded.textbook_source ELSE symptoms.textbook_source END,
			needs_review = excluded.needs_review,
			triage_role = excluded.triage_role,
			skip_followup_mode = excluded.skip_followup_mode,
			direct_triage_note = excluded.direct_triage_note,
			colloquial_aliases_json = excluded.colloquial_aliases_json
	`,
		card.ConceptCode, card.ConceptName, card.BodyPart, pinyinFull, pinyinInit,
		synonymsJ, card.DefaultServiceCode, diffsJ,
		isRedFlagInt, card.AgeGroupLimit, card.GenderLimit,
		existingProto, existingICDCode, existingICDTitle, textbookSource,
		needsReviewInt, existingCompsJ,
		card.TriageRole, card.SkipFollowupMode, card.DirectTriageNote, aliasesJ,
	)
	if err != nil {
		return fmt.Errorf("编译写入 symptoms 失败: %w", err)
	}
	return nil
}

func compileCardQuestionsTx(genTx *sql.Tx, card *PlaybookCard, wl *compilerWhitelists) (map[string]compiledQuestionLookup, error) {
	// 1. 先清理本卡之前编译的 Q_PB_<CONCEPT>_% 题目
	pbPrefix := fmt.Sprintf("Q_PB_%s_%%", card.ConceptCode)
	if _, err := genTx.Exec(`DELETE FROM questions WHERE question_id LIKE ?`, pbPrefix); err != nil {
		return nil, fmt.Errorf("清理旧编译题目失败: %w", err)
	}

	// 2. 合并问诊包模板 + 专属追问题（同 slot_key 时专属追问覆盖通用包）
	orderedSlots := make([]string, 0)
	slotTemplates := make(map[string]PlaybookQuestionTemplate)

	addTemplate := func(tpl PlaybookQuestionTemplate, fromPack bool) {
		slot := normalizeSlotKey(tpl.SlotKey)
		if slot == "" {
			return
		}
		tpl.SlotKey = slot
		if fromPack {
			// 来自问诊包的题目永远强制走卡作用域新 ID，确保零跨卡冲突
			tpl.QuestionID = fmt.Sprintf("Q_PB_%s_%s", card.ConceptCode, normalizeUpperToken(slot))
		} else if tpl.QuestionID == "" || strings.HasPrefix(tpl.QuestionID, "Q_PB_") {
			tpl.QuestionID = fmt.Sprintf("Q_PB_%s_%s", card.ConceptCode, normalizeUpperToken(slot))
		}
		if _, exists := slotTemplates[slot]; !exists {
			orderedSlots = append(orderedSlots, slot)
		}
		slotTemplates[slot] = tpl
	}

	for _, packCode := range card.AttachedPackCodes {
		pack, ok := wl.Packs[packCode]
		if !ok {
			continue
		}
		for _, tpl := range pack.QuestionsTemplate {
			addTemplate(tpl, true)
		}
	}
	for _, tpl := range card.CustomQuestions {
		addTemplate(tpl, false)
	}

	lookupBySlot := make(map[string]compiledQuestionLookup, len(orderedSlots))
	for _, slot := range orderedSlots {
		tpl := slotTemplates[slot]
		qID := tpl.QuestionID
		hasUncertain := false
		compiledOpts := make([]InquiryOption, 0, len(tpl.Options)+1)
		optValMap := make(map[string]string)

		for _, opt := range tpl.Options {
			if opt.IsUncertain || opt.Value == "UNCERTAIN" || opt.Value == "OPT_UNCERTAIN" {
				val := strings.TrimSpace(opt.Value)
				if val == "" {
					val = "OPT_UNCERTAIN"
				}
				hasUncertain = true
				compiledOpts = append(compiledOpts, InquiryOption{
					Label:       opt.Label,
					Value:       val,
					IsUncertain: true,
				})
				optValMap["UNCERTAIN"] = val
				optValMap[val] = val
				continue
			}

			val := strings.TrimSpace(opt.Value)
			if val == "" {
				optKey := normalizeUpperToken(opt.OptionKey)
				if optKey == "" {
					optKey = normalizeUpperToken(opt.Label)
				}
				val = fmt.Sprintf("OPT_%s_%s", normalizeUpperToken(slot), optKey)
			}
			compiledOpts = append(compiledOpts, InquiryOption{
				Label:            strings.TrimSpace(opt.Label),
				Value:            val,
				IsRedFlag:        opt.IsRedFlag,
				ServiceHint:      strings.TrimSpace(opt.ServiceHint),
				BoostWeight:      opt.BoostWeight,
				TCMSyndromeHints: uniqueNonEmptyStrings(opt.TCMSyndromeHints),
			})
			if strings.TrimSpace(opt.OptionKey) != "" {
				optValMap[strings.TrimSpace(opt.OptionKey)] = val
				optValMap[normalizeUpperToken(opt.OptionKey)] = val
			}
			optValMap[val] = val
		}

		if !hasUncertain {
			compiledOpts = append(compiledOpts, InquiryOption{
				Label:       "不清楚 / 我也不确定",
				Value:       "OPT_UNCERTAIN",
				IsUncertain: true,
			})
		}

		// 若沿用非 Q_PB_ 遗留题，保留它原本已有的其他适用症状，防止破坏历史共用规则
		appSymptoms := []string{card.ConceptCode}
		if !strings.HasPrefix(qID, "Q_PB_") {
			var existingAppJ string
			if err := genTx.QueryRow(`SELECT COALESCE(applicable_symptoms_json, '[]') FROM questions WHERE question_id = ?`, qID).Scan(&existingAppJ); err == nil {
				var existingSyms []string
				if json.Unmarshal([]byte(existingAppJ), &existingSyms) == nil {
					appSymptoms = uniqueNonEmptyStrings(append(existingSyms, card.ConceptCode))
				}
			}
		}

		optsBytes, _ := json.Marshal(compiledOpts)
		appBytes, _ := json.Marshal(appSymptoms)
		if _, err := genTx.Exec(`
			INSERT INTO questions (question_id, title, applicable_symptoms_json, type, category, evidence_dimension, options_json, slot_key)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(question_id) DO UPDATE SET
				title = excluded.title,
				applicable_symptoms_json = excluded.applicable_symptoms_json,
				type = excluded.type,
				category = excluded.category,
				evidence_dimension = excluded.evidence_dimension,
				options_json = excluded.options_json,
				slot_key = excluded.slot_key
		`, qID, tpl.Title, string(appBytes), tpl.Type, tpl.Category, tpl.EvidenceDimension, string(optsBytes), slot); err != nil {
			return nil, fmt.Errorf("编译写入问题 %s 失败: %w", qID, err)
		}

		lookupBySlot[slot] = compiledQuestionLookup{
			QuestionID: qID,
			SlotKey:    slot,
			OptionVals: optValMap,
		}
	}

	return lookupBySlot, nil
}

func compileCardSafetyRulesTx(genTx *sql.Tx, card *PlaybookCard, compiledQs map[string]compiledQuestionLookup) error {
	rulePrefix := fmt.Sprintf("RULE_PB_%s_%%", card.ConceptCode)
	if _, err := genTx.Exec(`DELETE FROM edges WHERE source_type='SYMPTOM' AND source_code=? AND relation_type='TRIGGERS_SAFETY' AND target_code LIKE ?`, card.ConceptCode, rulePrefix); err != nil {
		return err
	}
	if _, err := genTx.Exec(`DELETE FROM safety_rules WHERE rule_id LIKE ?`, rulePrefix); err != nil {
		return err
	}

	for idx, rf := range card.RedFlagRules {
		suffix := normalizeUpperToken(rf.RuleSuffix)
		if suffix == "" {
			suffix = fmt.Sprintf("RF_%d", idx+1)
		}
		ruleID := strings.TrimSpace(rf.RuleID)
		if ruleID == "" || strings.HasPrefix(ruleID, "RULE_PB_") {
			ruleID = fmt.Sprintf("RULE_PB_%s_%s", card.ConceptCode, suffix)
		}

		triggerMap := make(map[string]interface{})
		switch rf.TriggerMode {
		case TriggerModeConceptDirect:
			concepts := uniqueNonEmptyStrings(rf.TriggerConcepts)
			if len(concepts) == 0 {
				concepts = []string{card.ConceptCode}
			}
			triggerMap["any_of_concepts"] = concepts

		case TriggerModeSignMatch:
			signs := uniqueNonEmptyStrings(rf.TriggerSignCodes)
			if len(signs) == 0 {
				return newCompileError(400, "红旗规则 %s 使用 SIGN_MATCH 模式但未提供 trigger_sign_codes", ruleID)
			}
			triggerMap["any_of_signs"] = signs

		case TriggerModeOptionAnswer:
			slot := normalizeSlotKey(rf.BindSlotKey)
			resolvedQID := strings.TrimSpace(rf.BindQuestionID)
			resolvedVal := strings.TrimSpace(rf.BindOptionValue)

			if slot != "" {
				if cq, ok := compiledQs[slot]; ok {
					resolvedQID = cq.QuestionID
					key := strings.TrimSpace(rf.BindOptionKey)
					if key != "" {
						if mapped, exists := cq.OptionVals[key]; exists {
							resolvedVal = mapped
						} else if mappedUpper, existsUpper := cq.OptionVals[normalizeUpperToken(key)]; existsUpper {
							resolvedVal = mappedUpper
						}
					}
					if resolvedVal == "" && strings.TrimSpace(rf.BindOptionValue) != "" {
						if mapped, exists := cq.OptionVals[strings.TrimSpace(rf.BindOptionValue)]; exists {
							resolvedVal = mapped
						}
					}
				}
			} else if resolvedQID != "" {
				// 允许直接按 legacy question_id 反查
				for _, cq := range compiledQs {
					if cq.QuestionID == resolvedQID {
						if resolvedVal == "" && strings.TrimSpace(rf.BindOptionKey) != "" {
							resolvedVal = cq.OptionVals[strings.TrimSpace(rf.BindOptionKey)]
						}
						break
					}
				}
			}

			if resolvedQID == "" || resolvedVal == "" {
				return newCompileError(400, "红旗规则 %s 未能对齐有效题目(%s)或选项(%s/%s)", ruleID, rf.BindSlotKey, rf.BindOptionKey, rf.BindOptionValue)
			}
			triggerMap["any_of_answers"] = []map[string]string{
				{
					"question_id": resolvedQID,
					"value":       resolvedVal,
				},
			}
		}

		if len(triggerMap) == 0 {
			return newCompileError(400, "红旗规则 %s 触发条件不能为空", ruleID)
		}
		trigBytes, _ := json.Marshal(triggerMap)
		// 对沿用存量多句共享红旗规则（如 RULE_CRIT_ACS_MI），若保留原多子句 trigger_condition 且结构合法则无损保留
		if !strings.HasPrefix(ruleID, "RULE_PB_") && strings.TrimSpace(rf.RawTriggerJSON) != "" {
			rawMap := parseJSONMap(rf.RawTriggerJSON)
			if len(rawMap) > 0 && (rawMap["any_of_answers"] != nil || rawMap["any_of_concepts"] != nil || rawMap["any_of_signs"] != nil || rawMap["any_of_medical_history"] != nil) {
				trigBytes = []byte(rf.RawTriggerJSON)
			}
		}

		primarySyms := []string{card.ConceptCode}
		if !strings.HasPrefix(ruleID, "RULE_PB_") {
			var existingPrimJ string
			if err := genTx.QueryRow(`SELECT COALESCE(primary_symptoms_json, '[]') FROM safety_rules WHERE rule_id = ?`, ruleID).Scan(&existingPrimJ); err == nil {
				var existingSyms []string
				if json.Unmarshal([]byte(existingPrimJ), &existingSyms) == nil {
					primarySyms = uniqueNonEmptyStrings(append(existingSyms, card.ConceptCode))
				}
			}
		}
		primBytes, _ := json.Marshal(primarySyms)

		recDept := strings.TrimSpace(rf.RecommendedDeptName)
		if recDept == "" {
			if rf.Level == "CRITICAL" {
				recDept = "急诊抢救中心（急诊医学科）"
			} else {
				recDept = "急诊医学科 / 专科优先门诊"
			}
		}
		actionPayload := map[string]interface{}{
			"call_120":              rf.Call120,
			"recommended_dept_name": recDept,
			"guide_text":            strings.TrimSpace(rf.GuideText),
		}
		if strings.TrimSpace(rf.FallbackServiceCode) != "" {
			actionPayload["fallback_service_code"] = strings.TrimSpace(rf.FallbackServiceCode)
		}
		if len(rf.SymptomKeywords) > 0 {
			actionPayload["symptom_keywords"] = uniqueNonEmptyStrings(rf.SymptomKeywords)
		}
		actionBytes, _ := json.Marshal(actionPayload)

		if _, err := genTx.Exec(`
			INSERT INTO safety_rules (
				rule_id, name, level, safety_state, primary_symptoms_json,
				trigger_condition_json, title, warning_text, emergency_action_json
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(rule_id) DO UPDATE SET
				name = excluded.name,
				level = excluded.level,
				safety_state = excluded.safety_state,
				primary_symptoms_json = excluded.primary_symptoms_json,
				trigger_condition_json = excluded.trigger_condition_json,
				title = excluded.title,
				warning_text = excluded.warning_text,
				emergency_action_json = excluded.emergency_action_json
		`, ruleID, rf.Title, rf.Level, rf.SafetyState, string(primBytes), string(trigBytes), rf.Title, rf.WarningText, string(actionBytes)); err != nil {
			return fmt.Errorf("编译写入 safety_rules %s 失败: %w", ruleID, err)
		}

		if _, err := genTx.Exec(`DELETE FROM edges WHERE source_type='SYMPTOM' AND source_code=? AND target_type='RED_FLAG_RULE' AND target_code=? AND relation_type='TRIGGERS_SAFETY'`, card.ConceptCode, ruleID); err != nil {
			return err
		}
		if _, err := genTx.Exec(`
			INSERT INTO edges (source_type, source_code, target_type, target_code, relation_type, weight, evidence_quote)
			VALUES ('SYMPTOM', ?, 'RED_FLAG_RULE', ?, 'TRIGGERS_SAFETY', 1.0, ?)
		`, card.ConceptCode, ruleID, rf.Title); err != nil {
			return err
		}
	}
	return nil
}

func compileCardServiceEdgesTx(genTx *sql.Tx, card *PlaybookCard, compiledQs map[string]compiledQuestionLookup) error {
	if _, err := genTx.Exec(`
		DELETE FROM edges
		WHERE source_type = 'SYMPTOM' AND source_code = ? AND relation_type IN ('DEFAULT_SERVICE', 'BOOSTS_SERVICE')
	`, card.ConceptCode); err != nil {
		return fmt.Errorf("清理服务关联边失败: %w", err)
	}

	if _, err := genTx.Exec(`
		INSERT INTO edges (source_type, source_code, target_type, target_code, relation_type, weight, evidence_quote)
		VALUES ('SYMPTOM', ?, 'SERVICE', ?, 'DEFAULT_SERVICE', 1.0, 'Playbook 首选服务映射')
	`, card.ConceptCode, card.DefaultServiceCode); err != nil {
		return fmt.Errorf("写入 DEFAULT_SERVICE 边失败: %w", err)
	}

	boostWeights := make(map[string]float64)
	for _, bSvc := range card.BackupServiceCodes {
		if bSvc == "" || bSvc == card.DefaultServiceCode {
			continue
		}
		// BackupServiceCodes 已在 symptoms.differential_service_codes_json 中贡献 +0.5，
		// 此处仅写入 0.15 低权重辅助排序，确保未答题前总分 (0.65) 严格低于首推服务 (1.0)。
		if boostWeights[bSvc] < 0.15 {
			boostWeights[bSvc] = 0.15
		}
	}
	_ = compiledQs

	var boostKeys []string
	for k := range boostWeights {
		boostKeys = append(boostKeys, k)
	}
	sort.Strings(boostKeys)
	for _, targetSvc := range boostKeys {
		if _, err := genTx.Exec(`
			INSERT INTO edges (source_type, source_code, target_type, target_code, relation_type, weight, evidence_quote)
			VALUES ('SYMPTOM', ?, 'SERVICE', ?, 'BOOSTS_SERVICE', ?, 'Playbook 鉴别与备选服务加权')
		`, card.ConceptCode, targetSvc, boostWeights[targetSvc]); err != nil {
			return fmt.Errorf("写入 BOOSTS_SERVICE 边失败: %w", err)
		}
	}
	return nil
}

func compileCardTCMEdgesTx(tcmTx *sql.Tx, card *PlaybookCard) error {
	if _, err := tcmTx.Exec(`
		DELETE FROM tcm_edges
		WHERE source_code = ? AND relation_type IN ('SYMPTOM_TO_SYNDROME', 'SYMPTOM_TO_CLINIC', 'SYMPTOM_TO_THERAPY')
	`, card.ConceptCode); err != nil {
		return fmt.Errorf("清理中医边失败: %w", err)
	}

	quote := strings.TrimSpace(card.TCMBinding.EvidenceQuote)
	syndromes := uniqueNonEmptyStrings(card.TCMBinding.Syndromes)
	clinics := uniqueNonEmptyStrings(card.TCMBinding.Clinics)
	therapies := uniqueNonEmptyStrings(card.TCMBinding.Therapies)

	for _, syn := range syndromes {
		if _, err := tcmTx.Exec(`
			INSERT INTO tcm_edges (source_code, target_code, relation_type, weight, evidence_quote, source_ref)
			VALUES (?, ?, 'SYMPTOM_TO_SYNDROME', 0.9, ?, 'PlaybookCompiler')
		`, card.ConceptCode, syn, quote); err != nil {
			return err
		}
	}
	for _, clinic := range clinics {
		if _, err := tcmTx.Exec(`
			INSERT INTO tcm_edges (source_code, target_code, relation_type, weight, evidence_quote, source_ref)
			VALUES (?, ?, 'SYMPTOM_TO_CLINIC', 0.95, ?, 'PlaybookCompiler')
		`, card.ConceptCode, clinic, quote); err != nil {
			return err
		}
	}
	for _, therapy := range therapies {
		if _, err := tcmTx.Exec(`
			INSERT INTO tcm_edges (source_code, target_code, relation_type, weight, evidence_quote, source_ref)
			VALUES (?, ?, 'SYMPTOM_TO_THERAPY', 0.85, ?, 'PlaybookCompiler')
		`, card.ConceptCode, therapy, quote); err != nil {
			return err
		}
	}
	// 同步补齐 SYNDROME_TO_CLINIC 两跳连边，让四诊答题激活证候时也可直通专病门诊
	for _, syn := range syndromes {
		for _, clinic := range clinics {
			var cnt int
			_ = tcmTx.QueryRow(`SELECT COUNT(*) FROM tcm_edges WHERE source_code = ? AND target_code = ? AND relation_type = 'SYNDROME_TO_CLINIC'`, syn, clinic).Scan(&cnt)
			if cnt == 0 {
				if _, err := tcmTx.Exec(`
					INSERT INTO tcm_edges (source_code, target_code, relation_type, weight, evidence_quote, source_ref)
					VALUES (?, ?, 'SYNDROME_TO_CLINIC', 0.9, ?, 'PlaybookCompiler')
				`, syn, clinic, quote); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func compileCardEvidenceTx(genTx *sql.Tx, card *PlaybookCard) error {
	cite := card.TextbookCitation
	if strings.TrimSpace(cite.TextbookSource) == "" && strings.TrimSpace(cite.EvidenceSummary) == "" && len(cite.DifferentialNotes) == 0 {
		return nil
	}
	chunkID := "PB_" + card.ConceptCode
	if _, err := genTx.Exec(`DELETE FROM textbook_evidence WHERE chunk_id = ?`, chunkID); err != nil {
		return err
	}
	srcName := strings.TrimSpace(cite.TextbookSource)
	if srcName == "" {
		srcName = "临床分诊剧本共识依据"
	}
	diffBytes, _ := json.Marshal(cite.DifferentialNotes)
	_, err := genTx.Exec(`
		INSERT INTO textbook_evidence (
			chunk_id, symptom_code, concept_name, textbook_source, chapter, page,
			typical_signs_json, typical_symptoms_json, differential_diseases_json, associated_complications_json, evidence_summary
		) VALUES (?, ?, ?, ?, ?, ?, '[]', '[]', ?, '[]', ?)
	`, chunkID, card.ConceptCode, card.ConceptName, srcName, cite.Chapter, cite.Page, string(diffBytes), cite.EvidenceSummary)
	return err
}

// SaveQuestionPackAndCascade 保存通用问诊包模板并在同一事务内级联重编所有引用它的剧本卡
func SaveQuestionPackAndCascade(genPath, tcmPath string, pack QuestionPack, expectedDraftRev int) (*QuestionPack, int, int, error) {
	pack.PackCode = strings.TrimSpace(pack.PackCode)
	pack.PackName = strings.TrimSpace(pack.PackName)
	if pack.PackCode == "" || pack.PackName == "" {
		return nil, 0, 0, newCompileError(400, "pack_code 与 pack_name 必填")
	}
	if len(pack.QuestionsTemplate) == 0 {
		return nil, 0, 0, newCompileError(400, "questions_template 至少包含 1 道标准化问题")
	}

	genDB, err := openSQLiteRW(genPath)
	if err != nil {
		return nil, 0, 0, err
	}
	defer genDB.Close()

	var tcmDB *sql.DB
	if strings.TrimSpace(tcmPath) != "" {
		if tdb, tErr := openSQLiteRW(tcmPath); tErr == nil {
			tcmDB = tdb
			defer tcmDB.Close()
		}
	}

	if err := EnsurePlaybookSchema(genDB, tcmDB); err != nil {
		return nil, 0, 0, err
	}

	genTx, err := genDB.Begin()
	if err != nil {
		return nil, 0, 0, err
	}
	defer func() { _ = genTx.Rollback() }()

	var tcmTx *sql.Tx
	if tcmDB != nil {
		tcmTx, err = tcmDB.Begin()
		if err != nil {
			return nil, 0, 0, err
		}
		defer func() { _ = tcmTx.Rollback() }()
	}

	curRev := readDraftRevisionTx(genTx)
	if expectedDraftRev > 0 && expectedDraftRev != curRev {
		return nil, 0, 0, newCompileError(409, "草稿版本号冲突（当前 v%d，请求 v%d）", curRev, expectedDraftRev)
	}

	wl, err := loadWhitelistsTx(genTx, tcmTx)
	if err != nil {
		return nil, 0, 0, err
	}

	for qIdx := range pack.QuestionsTemplate {
		qt := &pack.QuestionsTemplate[qIdx]
		qt.SlotKey = normalizeSlotKey(qt.SlotKey)
		if qt.SlotKey == "" || strings.TrimSpace(qt.Title) == "" {
			return nil, 0, 0, newCompileError(400, "问诊包 %s 的问题缺少 slot_key 或 title", pack.PackCode)
		}
		if len(qt.Options) == 0 {
			return nil, 0, 0, newCompileError(400, "问诊包 %s 问题 %s 选项为空", pack.PackCode, qt.SlotKey)
		}
		for oIdx := range qt.Options {
			opt := &qt.Options[oIdx]
			if strings.TrimSpace(opt.Label) == "" || (strings.TrimSpace(opt.OptionKey) == "" && !opt.IsUncertain) {
				return nil, 0, 0, newCompileError(400, "问诊包 %s 问题 %s 存在缺少 option_key/label 的选项", pack.PackCode, qt.SlotKey)
			}
			if hint := strings.TrimSpace(opt.ServiceHint); hint != "" && !wl.Services[hint] {
				return nil, 0, 0, newCompileError(400, "问诊包 %s 选项 service_hint=%s 不在 30 项标准服务白名单内", pack.PackCode, hint)
			}
		}
	}

	var curPackVer int
	_ = genTx.QueryRow(`SELECT COALESCE(version_rev, 0) FROM question_packs WHERE pack_code = ?`, pack.PackCode).Scan(&curPackVer)
	pack.VersionRev = curPackVer + 1
	pack.UpdatedAt = time.Now().Format(time.RFC3339)
	if pack.ClinicalDomain == "" {
		pack.ClinicalDomain = "GENERAL"
	}

	tplBytes, _ := json.Marshal(pack.QuestionsTemplate)
	if _, err := genTx.Exec(`
		INSERT INTO question_packs (pack_code, pack_name, clinical_domain, description, questions_template_json, version_rev, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(pack_code) DO UPDATE SET
			pack_name = excluded.pack_name,
			clinical_domain = excluded.clinical_domain,
			description = excluded.description,
			questions_template_json = excluded.questions_template_json,
			version_rev = excluded.version_rev,
			updated_at = excluded.updated_at
	`, pack.PackCode, pack.PackName, pack.ClinicalDomain, pack.Description, string(tplBytes), pack.VersionRev, pack.UpdatedAt); err != nil {
		return nil, 0, 0, fmt.Errorf("保存 question_pack 失败: %w", err)
	}

	// 将更新后的问诊包放回内存白名单后再查出全部引用卡重编
	wl.Packs[pack.PackCode] = pack
	cards, err := loadCardsByPackCodeTx(genTx, pack.PackCode)
	if err != nil {
		return nil, 0, 0, err
	}

	cascadedCount := 0
	for i := range cards {
		cards[i].VersionRev++
		cards[i].UpdatedAt = pack.UpdatedAt
		if err := compileCardArtifactsTx(genTx, tcmTx, &cards[i], wl); err != nil {
			return nil, 0, 0, fmt.Errorf("级联重编剧本卡 %s 失败: %w", cards[i].ConceptCode, err)
		}
		cascadedCount++
	}

	newDraftRev, err := bumpDraftRevisionTx(genTx, pack.UpdatedAt)
	if err != nil {
		return nil, 0, 0, err
	}

	if tcmTx != nil {
		if err := tcmTx.Commit(); err != nil {
			return nil, 0, 0, err
		}
	}
	if err := genTx.Commit(); err != nil {
		return nil, 0, 0, err
	}
	return &pack, cascadedCount, newDraftRev, nil
}

func loadCardsByPackCodeTx(genTx *sql.Tx, packCode string) ([]PlaybookCard, error) {
	rows, err := genTx.Query(`SELECT ` + playbookCardSelectCols + ` FROM playbook_cards`)
	if err != nil {
		return nil, err
	}
	allCards, err := scanPlaybookCardRows(rows)
	_ = rows.Close()
	if err != nil {
		return nil, err
	}
	var matched []PlaybookCard
	for _, c := range allCards {
		for _, p := range c.AttachedPackCodes {
			if p == packCode {
				matched = append(matched, c)
				break
			}
		}
	}
	return matched, nil
}

// BootstrapPlaybooksFromLegacy 从现有底表幂等初始化 playbook_cards，补齐 THUNDERCLAP 规则、OPT_UNCERTAIN 与 SYM_TB 过渡态
func BootstrapPlaybooksFromLegacy(genDB, tcmDB *sql.DB) (int, error) {
	if genDB == nil {
		return 0, fmt.Errorf("genDB 不能为空")
	}
	if err := EnsurePlaybookSchema(genDB, tcmDB); err != nil {
		return 0, err
	}

	// 1. 保证 legacy 题目包含 OPT_UNCERTAIN 与标准 slot_key
	if err := ensureLegacyQuestionsNormalized(genDB); err != nil {
		return 0, err
	}

	// 2. 保证 SYM_HEADACHE_THUNDERCLAP 具备显式 CONCEPT_DIRECT 安全规则 RULE_CRIT_THUNDERCLAP
	if err := ensureThunderclapRuleExists(genDB); err != nil {
		return 0, err
	}

	// 3. 若中医库尚无 SYNDROME_TO_CLINIC 边，补全 10 证候→6 专病 2 跳连接
	if tcmDB != nil {
		if err := ensureLegacySyndromeClinicEdges(tcmDB); err != nil {
			return 0, err
		}
	}

	// 4. 将尚未建立 playbook_cards 的症状反向抽象写入 playbook_cards（仅补差量，不覆写已配置卡）
	existingCards := make(map[string]bool)
	eRows, err := genDB.Query(`SELECT concept_code FROM playbook_cards`)
	if err == nil {
		for eRows.Next() {
			var code string
			if eRows.Scan(&code) == nil {
				existingCards[code] = true
			}
		}
		_ = eRows.Close()
	}

	// 预加载 questions / safety_rules / textbook_evidence / tcm_edges 索引
	qBySymptom := loadLegacyQuestionsBySymptom(genDB)
	rBySymptom := loadLegacyRulesBySymptom(genDB)
	eBySymptom := loadLegacyEvidenceBySymptom(genDB)
	tcmBySymptom := loadLegacyTCMBindingsBySymptom(tcmDB)

	curatedMismatches := curatedLegacyMismatches()
	now := time.Now().Format(time.RFC3339)

	sRows, err := genDB.Query(`
		SELECT concept_code, concept_name, body_part, COALESCE(synonyms_json, '[]'),
			default_service_code, COALESCE(differential_service_codes_json, '[]'),
			COALESCE(recommended_age_group, 'ALL'), COALESCE(gender_limit, 'ALL'),
			COALESCE(needs_review, 0), COALESCE(textbook_source, ''),
			COALESCE(triage_role, 'CORE_COMPLAINT'), COALESCE(skip_followup_mode, 'REQUIRED'),
			COALESCE(direct_triage_note, ''), COALESCE(colloquial_aliases_json, '[]')
		FROM symptoms
		ORDER BY concept_code ASC
	`)
	if err != nil {
		return 0, err
	}

	type symRow struct {
		code, name, bodyPart, synJ, defSvc, diffJ, ageGrp, genLim, tbSrc, role, skipMode, directNote, colJ string
		needsRev                                                                                           int
	}
	var symRows []symRow
	for sRows.Next() {
		var r symRow
		if scanErr := sRows.Scan(&r.code, &r.name, &r.bodyPart, &r.synJ, &r.defSvc, &r.diffJ, &r.ageGrp, &r.genLim, &r.needsRev, &r.tbSrc, &r.role, &r.skipMode, &r.directNote, &r.colJ); scanErr == nil {
			symRows = append(symRows, r)
		}
	}
	_ = sRows.Close()

	tx, err := genDB.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	created := 0
	for _, r := range symRows {
		if existingCards[r.code] {
			continue
		}

		role := r.role
		if strings.HasPrefix(r.code, "SYM_TB_") && r.needsRev == 1 {
			role = TriageRoleTBDeferred
			_, _ = tx.Exec(`UPDATE symptoms SET triage_role = ? WHERE concept_code = ?`, role, r.code)
		} else if role == "" {
			role = TriageRoleCoreComplaint
		}

		reviewStatus := ReviewStatusApproved
		if r.needsRev == 1 {
			reviewStatus = ReviewStatusNeedsReview
		}

		var aliases []string
		_ = json.Unmarshal([]byte(r.colJ), &aliases)
		if len(aliases) == 0 {
			_ = json.Unmarshal([]byte(r.synJ), &aliases)
		}
		var backups []string
		_ = json.Unmarshal([]byte(r.diffJ), &backups)

		card := PlaybookCard{
			ConceptCode:        r.code,
			ConceptName:        r.name,
			TriageRole:         role,
			SkipFollowupMode:   r.skipMode,
			DirectTriageNote:   r.directNote,
			BodyPart:           r.bodyPart,
			ColloquialAliases:  uniqueNonEmptyStrings(aliases),
			GenderLimit:        r.genLim,
			AgeGroupLimit:      normalizeAgeGroupTag(r.ageGrp),
			DefaultServiceCode: r.defSvc,
			BackupServiceCodes: uniqueNonEmptyStrings(backups),
			RedFlagRules:       rBySymptom[r.code],
			AttachedPackCodes:  []string{},
			CustomQuestions:    qBySymptom[r.code],
			TCMBinding:         tcmBySymptom[r.code],
			TextbookCitation:   eBySymptom[r.code],
			ReviewStatus:       reviewStatus,
			VersionRev:         1,
			UpdatedAt:          now,
			UpdatedBy:          "system:bootstrap",
		}
		if card.TextbookCitation.TextbookSource == "" {
			card.TextbookCitation.TextbookSource = r.tbSrc
		}
		if alert, ok := curatedMismatches[r.code]; ok {
			card.MismatchAlert = alert
		}

		if err := upsertPlaybookCardRowTx(tx, &card); err != nil {
			return created, err
		}
		created++
	}

	if err := tx.Commit(); err != nil {
		return created, err
	}
	return created, nil
}

func inferLegacySlotKey(questionID string) string {
	switch questionID {
	case "Q_CHEST_RADIATION":
		return "chest_radiation"
	case "Q_CHEST_COLD_SWEAT":
		return "chest_cold_sweat"
	case "Q_CHEST_NATURE":
		return "chest_nature"
	case "Q_DYSPNEA_SEVERITY":
		return "dyspnea_sev"
	case "Q_DYSPNEA_POSTURE":
		return "dyspnea_posture"
	case "Q_ABDOMEN_RIGIDITY":
		return "abd_alarm"
	case "Q_ABDOMEN_TRANSFER":
		return "abd_transfer"
	case "Q_STROKE_FAST_TIME", "Q_HEADACHE_ONSET_SEVERITY":
		return "neuro_fast"
	case "Q_FEMALE_PREGNANCY_HISTORY", "Q_GYN_PREGNANCY_BLEEDING":
		return "is_pregnant"
	case "Q_PREGNANCY_WEEKS":
		return "pregnancy_weeks"
	case "Q_COUGH_DURATION":
		return "onset_acute"
	case "Q_ANAPHYLAXIS_DYSPNEA":
		return "allergy_airway"
	case "Q_SPINE_RADICULAR_PAIN":
		return "spine_red"
	case "Q_THROAT_BREATHING":
		return "ent_urgent"
	case "Q_VISION_DROP_SPEED", "Q_EYE_PAIN_HEADACHE", "Q_EYE_PRESSURE_HEADACHE":
		return "eye_urgent"
	case "Q_PED_EMERGENCY_SIGNS":
		return "ped_alarm"
	default:
		clean := strings.TrimPrefix(questionID, "Q_")
		return normalizeSlotKey(clean)
	}
}

func ensureLegacyQuestionsNormalized(genDB *sql.DB) error {
	rows, err := genDB.Query(`SELECT question_id, COALESCE(options_json, '[]'), COALESCE(slot_key, '') FROM questions`)
	if err != nil {
		return err
	}
	type qPatch struct {
		id      string
		slotKey string
		optsJ   string
		dirty   bool
	}
	var patches []qPatch
	for rows.Next() {
		var id, optsJ, curSlot string
		if rows.Scan(&id, &optsJ, &curSlot) != nil {
			continue
		}
		var opts []InquiryOption
		_ = json.Unmarshal([]byte(optsJ), &opts)
		hasUncertain := false
		for _, o := range opts {
			if o.IsUncertain || o.Value == "UNCERTAIN" || o.Value == "OPT_UNCERTAIN" {
				hasUncertain = true
				break
			}
		}
		newSlot := strings.TrimSpace(curSlot)
		if newSlot == "" {
			newSlot = inferLegacySlotKey(id)
		}
		dirty := newSlot != curSlot
		if !hasUncertain {
			opts = append(opts, InquiryOption{
				Label:       "不清楚 / 我也不确定",
				Value:       "OPT_UNCERTAIN",
				IsUncertain: true,
			})
			b, _ := json.Marshal(opts)
			optsJ = string(b)
			dirty = true
		}
		if dirty {
			patches = append(patches, qPatch{id: id, slotKey: newSlot, optsJ: optsJ, dirty: true})
		}
	}
	_ = rows.Close()

	if len(patches) == 0 {
		return nil
	}
	tx, err := genDB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, p := range patches {
		if _, err := tx.Exec(`UPDATE questions SET slot_key = ?, options_json = ? WHERE question_id = ?`, p.slotKey, p.optsJ, p.id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func ensureThunderclapRuleExists(genDB *sql.DB) error {
	var cnt int
	_ = genDB.QueryRow(`SELECT COUNT(*) FROM safety_rules WHERE rule_id = 'RULE_CRIT_THUNDERCLAP'`).Scan(&cnt)
	if cnt > 0 {
		return nil
	}
	actionJ := mustMarshalJSON(map[string]interface{}{
		"call_120":              true,
		"recommended_dept_name": "急诊抢救中心（神经外科 / 急诊医学科）",
		"guide_text":            "请立即停止活动平卧休息，保持呼吸道通畅，马上拨打 120 急救电话或由家属急救送医，警惕动脉瘤破裂！",
	}, "{}")
	trigJ := mustMarshalJSON(map[string]interface{}{
		"any_of_concepts": []string{"SYM_HEADACHE_THUNDERCLAP"},
	}, "{}")
	primJ := mustMarshalJSON([]string{"SYM_HEADACHE_THUNDERCLAP"}, "[]")

	_, err := genDB.Exec(`
		INSERT INTO safety_rules (
			rule_id, name, level, safety_state, primary_symptoms_json,
			trigger_condition_json, title, warning_text, emergency_action_json
		) VALUES (
			'RULE_CRIT_THUNDERCLAP',
			'突发雷击样剧痛高度警惕蛛网膜下腔出血',
			'CRITICAL',
			'EMERGENCY_NOW',
			?, ?,
			'突发剧烈头痛高危警报（疑似蛛网膜下腔出血）',
			'突发达到顶峰的雷击样炸裂剧痛属于神经科急危重症，必须立即急诊排查！',
			?
		)
	`, primJ, trigJ, actionJ)
	if err != nil {
		return fmt.Errorf("写入 RULE_CRIT_THUNDERCLAP 失败: %w", err)
	}
	_, _ = genDB.Exec(`
		INSERT OR IGNORE INTO edges (source_type, source_code, target_type, target_code, relation_type, weight, evidence_quote)
		VALUES ('SYMPTOM', 'SYM_HEADACHE_THUNDERCLAP', 'RED_FLAG_RULE', 'RULE_CRIT_THUNDERCLAP', 'TRIGGERS_SAFETY', 1.0, '雷击样剧烈头痛直连急救规则')
	`)
	return nil
}

func ensureLegacySyndromeClinicEdges(tcmDB *sql.DB) error {
	var cnt int
	if err := tcmDB.QueryRow(`SELECT COUNT(*) FROM tcm_edges WHERE relation_type = 'SYNDROME_TO_CLINIC'`).Scan(&cnt); err != nil {
		return nil
	}
	if cnt > 0 {
		return nil
	}
	links := []struct {
		syn, clinic, quote string
		w                  float64
	}{
		{"TCM_SYN_SPLEEN_STOMACH_COLD_DEFICIENCY", "TCM_CLINIC_HP_REGULATION", "脾胃虚寒证辨证关联幽门螺杆菌/脾胃调理专病", 0.9},
		{"TCM_SYN_SPLEEN_STOMACH_DAMP_HEAT", "TCM_CLINIC_HP_REGULATION", "脾胃湿热证辨证关联幽门螺杆菌中药特色专病", 0.9},
		{"TCM_SYN_LIVER_QI_STAGNATION_SPLEEN", "TCM_CLINIC_CHRONIC_GASTRITIS_REVERSAL", "肝郁脾虚证辨证关联慢性萎缩性胃炎逆转门诊", 0.92},
		{"TCM_SYN_WIND_COLD_DAMP_BI", "TCM_CLINIC_CERVICAL_SPINE_REPAIR", "风寒湿痹证辨证关联颈肩腰腿痛针刀专病门诊", 0.95},
		{"TCM_SYN_WIND_PHLEGM_STRIKE", "TCM_CLINIC_FACIAL_ACUTE_FAST", "风痰阻络证辨证关联周围性面瘫针灸急救通道", 0.95},
		{"TCM_SYN_HEART_SPLEEN_DEFICIENCY", "TCM_CLINIC_SLEEP_HERBAL", "心脾两虚证辨证关联顽固性失眠经方调理门诊", 0.95},
		{"TCM_SYN_PHLEGM_HEAT_DISTURB_MIND", "TCM_CLINIC_SLEEP_HERBAL", "痰热扰心证辨证关联顽固性失眠经方调理门诊", 0.92},
		{"TCM_SYN_LIVER_KIDNEY_YIN_DEFICIENCY", "TCM_CLINIC_SLEEP_HERBAL", "肝肾阴虚证辨证关联失眠经方调理门诊", 0.85},
	}
	tx, err := tcmDB.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, lk := range links {
		if _, err := tx.Exec(`
			INSERT INTO tcm_edges (source_code, target_code, relation_type, weight, evidence_quote, source_ref)
			VALUES (?, ?, 'SYNDROME_TO_CLINIC', ?, ?, 'Bootstrap2Hop')
		`, lk.syn, lk.clinic, lk.w, lk.quote); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func curatedLegacyMismatches() map[string]PlaybookMismatchAlert {
	return map[string]PlaybookMismatchAlert{
		"SYM_TB_SKIN_MUCOUS_MEMBRANE_BLEEDING": {
			HasAlert:             true,
			PreviousServiceCode:  "DERMATOLOGY_ASSESSMENT",
			SuggestedServiceCode: "HEMATOLOGY_ASSESSMENT",
			Reason:               "自发皮肤黏膜出血/瘀斑紫癜首要病因为血小板或凝血机制异常，原配皮肤科属典型教材导入错配，建议调整为血液内科",
			Severity:             "HIGH",
		},
		"SYM_TB_MULTISITE_HEMORRHAGE": {
			HasAlert:             true,
			PreviousServiceCode:  "DERMATOLOGY_ASSESSMENT",
			SuggestedServiceCode: "HEMATOLOGY_ASSESSMENT",
			Reason:               "多部位自发出血属于全身凝血/血液系统警报，原配皮肤科属错配，建议调整为血液内科",
			Severity:             "HIGH",
		},
		"SYM_TB_UNDULANT_FEVER": {
			HasAlert:             true,
			PreviousServiceCode:  "DERMATOLOGY_ASSESSMENT",
			SuggestedServiceCode: "INFECTIOUS_ASSESSMENT",
			Reason:               "波状热属于典型感染性发热（如布鲁氏菌病/感染热型），原配皮肤科属错配，建议调整为感染科",
			Severity:             "HIGH",
		},
		"SYM_TB_SCARLATINIFORM_RASH": {
			HasAlert:             true,
			PreviousServiceCode:  "DERMATOLOGY_ASSESSMENT",
			SuggestedServiceCode: "INFECTIOUS_ASSESSMENT",
			Reason:               "猩红热样皮疹常伴传染性发热（亦需与药疹鉴别），建议结合发热问诊包关联感染科",
			Severity:             "MEDIUM",
		},
		"SYM_TB_LOCALIZED_CONVULSION": {
			HasAlert:             true,
			PreviousServiceCode:  "DERMATOLOGY_ASSESSMENT",
			SuggestedServiceCode: "NEURO_INTERNAL_ASSESSMENT",
			Reason:               "局限性抽搐属神经系统异常放电表现，原配皮肤科错配，建议调整为神经内科",
			Severity:             "HIGH",
		},
		"SYM_TB_CLONIC_CONVULSION": {
			HasAlert:             true,
			PreviousServiceCode:  "DERMATOLOGY_ASSESSMENT",
			SuggestedServiceCode: "NEURO_INTERNAL_ASSESSMENT",
			Reason:               "阵挛性抽搐属癫痫/神经系统表现，原配皮肤科错配，建议调整为神经内科",
			Severity:             "HIGH",
		},
		"SYM_TB_TEMPERATURE_SENSE_IMPAIRMENT": {
			HasAlert:             true,
			PreviousServiceCode:  "DERMATOLOGY_ASSESSMENT",
			SuggestedServiceCode: "NEURO_INTERNAL_ASSESSMENT",
			Reason:               "温度觉障碍属感觉神经通路受损体征（或作为伴随查体词），建议改配神经内科或转 COMPNAION_ONLY",
			Severity:             "HIGH",
		},
		"SYM_TB_BLOOD_LIPID_ELEVATED": {
			HasAlert:             true,
			PreviousServiceCode:  "DERMATOLOGY_ASSESSMENT",
			SuggestedServiceCode: "ENDOCRINE_ASSESSMENT",
			Reason:               "血脂增高属代谢异常（或单跳直达内分泌/心血管评估），原配皮肤科错配",
			Severity:             "MEDIUM",
		},
		"SYM_TB_LIPID_METABOLISM_DISORDER": {
			HasAlert:             true,
			PreviousServiceCode:  "DERMATOLOGY_ASSESSMENT",
			SuggestedServiceCode: "ENDOCRINE_ASSESSMENT",
			Reason:               "脂代谢异常属内分泌代谢疾病谱，原配皮肤科错配",
			Severity:             "MEDIUM",
		},
		"SYM_TB_REM_SLEEP_BEHAVIOR_DISORDER": {
			HasAlert:             true,
			PreviousServiceCode:  "DERMATOLOGY_ASSESSMENT",
			SuggestedServiceCode: "NEURO_INTERNAL_ASSESSMENT",
			Reason:               "快速眼动睡眠行为异常属睡眠/神经退行性相关评估，原配皮肤科错配",
			Severity:             "MEDIUM",
		},
		"SYM_TB_HOT_FLASHES": {
			HasAlert:             true,
			PreviousServiceCode:  "DERMATOLOGY_ASSESSMENT",
			SuggestedServiceCode: "GYNECOLOGY_ASSESSMENT",
			Reason:               "潮热常关联更年期妇科内分泌变化，原配皮肤科错配，建议调整为妇科或内分泌科",
			Severity:             "MEDIUM",
		},
		"SYM_TB_ABSENT_SECONDARY_SEXUAL_CHARACTERISTICS": {
			HasAlert:             true,
			PreviousServiceCode:  "DERMATOLOGY_ASSESSMENT",
			SuggestedServiceCode: "ENDOCRINE_ASSESSMENT",
			Reason:               "第二性征缺乏属下丘脑-垂体-性腺轴内分泌发育问题，原配皮肤科错配",
			Severity:             "MEDIUM",
		},
	}
}

func loadLegacyQuestionsBySymptom(genDB *sql.DB) map[string][]PlaybookQuestionTemplate {
	res := make(map[string][]PlaybookQuestionTemplate)
	rows, err := genDB.Query(`SELECT question_id, title, applicable_symptoms_json, type, category, COALESCE(evidence_dimension, ''), options_json, COALESCE(slot_key, '') FROM questions ORDER BY question_id ASC`)
	if err != nil {
		return res
	}
	defer rows.Close()
	for rows.Next() {
		var qID, title, appJ, qType, cat, dim, optsJ, slotKey string
		if rows.Scan(&qID, &title, &appJ, &qType, &cat, &dim, &optsJ, &slotKey) != nil {
			continue
		}
		var syms []string
		_ = json.Unmarshal([]byte(appJ), &syms)
		var rawOpts []InquiryOption
		_ = json.Unmarshal([]byte(optsJ), &rawOpts)
		pbOpts := make([]PlaybookQuestionOption, 0, len(rawOpts))
		for _, o := range rawOpts {
			pbOpts = append(pbOpts, PlaybookQuestionOption{
				OptionKey:        o.Value,
				Value:            o.Value,
				Label:            o.Label,
				IsRedFlag:        o.IsRedFlag,
				IsUncertain:      o.IsUncertain,
				ServiceHint:      o.ServiceHint,
				BoostWeight:      o.BoostWeight,
				TCMSyndromeHints: o.TCMSyndromeHints,
			})
		}
		if slotKey == "" {
			slotKey = inferLegacySlotKey(qID)
		}
		tpl := PlaybookQuestionTemplate{
			QuestionID:        qID,
			SlotKey:           slotKey,
			Title:             title,
			Category:          cat,
			Type:              qType,
			EvidenceDimension: dim,
			Options:           pbOpts,
		}
		for _, s := range syms {
			res[s] = append(res[s], tpl)
		}
	}
	return res
}

func loadLegacyRulesBySymptom(genDB *sql.DB) map[string][]PlaybookRedFlagRule {
	res := make(map[string][]PlaybookRedFlagRule)
	rows, err := genDB.Query(`SELECT rule_id, name, level, safety_state, primary_symptoms_json, trigger_condition_json, title, warning_text, emergency_action_json FROM safety_rules ORDER BY rule_id ASC`)
	if err != nil {
		return res
	}
	defer rows.Close()
	for rows.Next() {
		var rID, name, level, sState, primJ, trigJ, title, warnText, actJ string
		if rows.Scan(&rID, &name, &level, &sState, &primJ, &trigJ, &title, &warnText, &actJ) != nil {
			continue
		}
		var syms []string
		_ = json.Unmarshal([]byte(primJ), &syms)
		trigMap := parseJSONMap(trigJ)
		actMap := parseJSONMap(actJ)

		rf := PlaybookRedFlagRule{
			RuleID:              rID,
			RuleSuffix:          strings.TrimPrefix(rID, "RULE_"),
			Title:               title,
			Level:               level,
			SafetyState:         sState,
			RawTriggerJSON:      trigJ,
			WarningText:         warnText,
			RecommendedDeptName: toString(actMap["recommended_dept_name"]),
			FallbackServiceCode: toString(actMap["fallback_service_code"]),
			Call120:             toBoolInt(actMap["call_120"], 0) == 1,
			GuideText:           toString(actMap["guide_text"]),
		}
		if ansList, ok := trigMap["any_of_answers"].([]interface{}); ok && len(ansList) > 0 {
			rf.TriggerMode = TriggerModeOptionAnswer
			if first, ok := ansList[0].(map[string]interface{}); ok {
				rf.BindQuestionID = toString(first["question_id"])
				rf.BindSlotKey = inferLegacySlotKey(rf.BindQuestionID)
				rf.BindOptionValue = toString(first["value"])
				rf.BindOptionKey = rf.BindOptionValue
			}
		} else if cList := parseJSONInterfaceSlice(trigMap["any_of_concepts"]); len(cList) > 0 {
			rf.TriggerMode = TriggerModeConceptDirect
			rf.TriggerConcepts = cList
		} else if sList := parseJSONInterfaceSlice(trigMap["any_of_signs"]); len(sList) > 0 {
			rf.TriggerMode = TriggerModeSignMatch
			rf.TriggerSignCodes = sList
		} else {
			rf.TriggerMode = TriggerModeOptionAnswer
		}
		for _, s := range syms {
			res[s] = append(res[s], rf)
		}
	}
	return res
}

func parseJSONInterfaceSlice(raw interface{}) []string {
	arr, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	var out []string
	for _, item := range arr {
		if s := toString(item); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func loadLegacyEvidenceBySymptom(genDB *sql.DB) map[string]PlaybookTextbookCitation {
	res := make(map[string]PlaybookTextbookCitation)
	cols := sqliteTableColumns(genDB, "textbook_evidence")
	sumCol := "'' AS evidence_summary"
	if cols["evidence_summary"] {
		sumCol = "COALESCE(evidence_summary, '')"
	}
	rows, err := genDB.Query(fmt.Sprintf(`SELECT symptom_code, COALESCE(textbook_source, ''), COALESCE(chapter, ''), COALESCE(page, ''), COALESCE(differential_diseases_json, '[]'), %s FROM textbook_evidence ORDER BY id ASC`, sumCol))
	if err != nil {
		return res
	}
	defer rows.Close()
	for rows.Next() {
		var code, src, ch, pg, diffJ, sumText string
		if rows.Scan(&code, &src, &ch, &pg, &diffJ, &sumText) != nil {
			continue
		}
		if _, exists := res[code]; exists {
			continue
		}
		var notes []PlaybookDiffNote
		var rawArr []map[string]interface{}
		if json.Unmarshal([]byte(diffJ), &rawArr) == nil {
			for _, item := range rawArr {
				dName := toString(item["disease_name"])
				if dName == "" {
					dName = toString(item["disease"])
				}
				feat := toString(item["distinguishing_features"])
				if feat == "" {
					feat = toString(item["feature"])
				}
				if dName != "" {
					notes = append(notes, PlaybookDiffNote{DiseaseName: dName, DistinguishingFeatures: feat})
				}
			}
		}
		res[code] = PlaybookTextbookCitation{
			TextbookSource:    src,
			Chapter:           ch,
			Page:              pg,
			EvidenceSummary:   sumText,
			DifferentialNotes: notes,
		}
	}
	return res
}

func loadLegacyTCMBindingsBySymptom(tcmDB *sql.DB) map[string]PlaybookTCMBinding {
	res := make(map[string]PlaybookTCMBinding)
	if tcmDB == nil {
		return res
	}
	rows, err := tcmDB.Query(`SELECT source_code, target_code, relation_type, COALESCE(evidence_quote, '') FROM tcm_edges WHERE relation_type IN ('SYMPTOM_TO_SYNDROME', 'SYMPTOM_TO_CLINIC', 'SYMPTOM_TO_THERAPY')`)
	if err != nil {
		return res
	}
	defer rows.Close()
	for rows.Next() {
		var src, tgt, rel, quote string
		if rows.Scan(&src, &tgt, &rel, &quote) != nil {
			continue
		}
		b := res[src]
		if quote != "" && b.EvidenceQuote == "" {
			b.EvidenceQuote = quote
		}
		switch rel {
		case "SYMPTOM_TO_SYNDROME":
			b.Syndromes = uniqueNonEmptyStrings(append(b.Syndromes, tgt))
		case "SYMPTOM_TO_CLINIC":
			b.Clinics = uniqueNonEmptyStrings(append(b.Clinics, tgt))
		case "SYMPTOM_TO_THERAPY":
			b.Therapies = uniqueNonEmptyStrings(append(b.Therapies, tgt))
		}
		res[src] = b
	}
	return res
}

const playbookCardSelectCols = `
	concept_code, concept_name, COALESCE(triage_role, 'CORE_COMPLAINT'),
	COALESCE(skip_followup_mode, 'REQUIRED'), COALESCE(direct_triage_note, ''),
	COALESCE(body_part, 'GENERAL'), COALESCE(colloquial_aliases_json, '[]'),
	COALESCE(gender_limit, 'ALL'), COALESCE(age_group_limit, 'ALL'),
	default_service_code, COALESCE(backup_service_codes_json, '[]'),
	COALESCE(red_flag_rules_json, '[]'), COALESCE(attached_pack_codes_json, '[]'),
	COALESCE(custom_questions_json, '[]'), COALESCE(tcm_binding_json, '{}'),
	COALESCE(textbook_citation_json, '{}'), COALESCE(review_status, 'NEEDS_REVIEW'),
	COALESCE(mismatch_alert_json, '{}'), COALESCE(version_rev, 1),
	COALESCE(updated_at, ''), COALESCE(updated_by, '')
`

func scanPlaybookCardRows(rows *sql.Rows) ([]PlaybookCard, error) {
	var cards []PlaybookCard
	for rows.Next() {
		var c PlaybookCard
		var aliasesJ, backupsJ, rfJ, packsJ, customQsJ, tcmJ, citeJ, mismatchJ string
		if err := rows.Scan(
			&c.ConceptCode, &c.ConceptName, &c.TriageRole,
			&c.SkipFollowupMode, &c.DirectTriageNote,
			&c.BodyPart, &aliasesJ,
			&c.GenderLimit, &c.AgeGroupLimit,
			&c.DefaultServiceCode, &backupsJ,
			&rfJ, &packsJ,
			&customQsJ, &tcmJ,
			&citeJ, &c.ReviewStatus,
			&mismatchJ, &c.VersionRev,
			&c.UpdatedAt, &c.UpdatedBy,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(aliasesJ), &c.ColloquialAliases)
		_ = json.Unmarshal([]byte(backupsJ), &c.BackupServiceCodes)
		_ = json.Unmarshal([]byte(rfJ), &c.RedFlagRules)
		_ = json.Unmarshal([]byte(packsJ), &c.AttachedPackCodes)
		_ = json.Unmarshal([]byte(customQsJ), &c.CustomQuestions)
		_ = json.Unmarshal([]byte(tcmJ), &c.TCMBinding)
		_ = json.Unmarshal([]byte(citeJ), &c.TextbookCitation)
		_ = json.Unmarshal([]byte(mismatchJ), &c.MismatchAlert)
		c.NeedsReviewFlag = c.ReviewStatus != ReviewStatusApproved
		cards = append(cards, c)
	}
	return cards, nil
}

// ListPlaybookCards 分页筛选分诊剧本卡并返回全局统计与当前 draft_revision
func ListPlaybookCards(genPath string, filter PlaybookListFilter) (map[string]interface{}, error) {
	db, err := openSQLiteRW(genPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if err := EnsurePlaybookSchema(db, nil); err != nil {
		return nil, err
	}

	rows, err := db.Query(`SELECT ` + playbookCardSelectCols + ` FROM playbook_cards ORDER BY concept_code ASC`)
	if err != nil {
		return nil, err
	}
	allCards, err := scanPlaybookCardRows(rows)
	_ = rows.Close()
	if err != nil {
		return nil, err
	}

	var draftRev int
	_ = db.QueryRow(`SELECT COALESCE(draft_revision, 1) FROM playbook_draft_meta WHERE id = 1`).Scan(&draftRev)

	stats := map[string]int{
		"total":            len(allCards),
		"core_complaint":   0,
		"companion_only":   0,
		"tb_deferred":      0,
		"archived":         0,
		"needs_review":     0,
		"mismatch_alerts":  0,
		"zero_questions":   0,
		"direct_single":    0,
		"approved_active":  0,
	}

	searchLower := strings.ToLower(strings.TrimSpace(filter.Search))
	var filtered []PlaybookCard
	for _, c := range allCards {
		switch c.TriageRole {
		case TriageRoleCoreComplaint:
			stats["core_complaint"]++
			if c.ReviewStatus == ReviewStatusApproved {
				stats["approved_active"]++
			}
		case TriageRoleCompanionOnly:
			stats["companion_only"]++
		case TriageRoleTBDeferred:
			stats["tb_deferred"]++
		case TriageRoleArchived:
			stats["archived"]++
		}
		if c.ReviewStatus != ReviewStatusApproved {
			stats["needs_review"]++
		}
		if c.MismatchAlert.HasAlert {
			stats["mismatch_alerts"]++
		}
		hasQuestions := len(c.AttachedPackCodes) > 0 || len(c.CustomQuestions) > 0
		if !hasQuestions && c.SkipFollowupMode != SkipFollowupDirectSingleHop {
			stats["zero_questions"]++
		}
		if c.SkipFollowupMode == SkipFollowupDirectSingleHop {
			stats["direct_single"]++
		}

		if filter.TriageRole != "" && c.TriageRole != filter.TriageRole {
			continue
		}
		if filter.ReviewStatus != "" && c.ReviewStatus != filter.ReviewStatus {
			continue
		}
		if filter.OnlyMismatch && !c.MismatchAlert.HasAlert {
			continue
		}
		if filter.OnlyZeroQ && (hasQuestions || c.SkipFollowupMode == SkipFollowupDirectSingleHop) {
			continue
		}
		if searchLower != "" {
			matched := strings.Contains(strings.ToLower(c.ConceptCode), searchLower) ||
				strings.Contains(strings.ToLower(c.ConceptName), searchLower) ||
				strings.Contains(strings.ToLower(c.DefaultServiceCode), searchLower)
			if !matched {
				for _, al := range c.ColloquialAliases {
					if strings.Contains(strings.ToLower(al), searchLower) {
						matched = true
						break
					}
				}
			}
			if !matched {
				continue
			}
		}
		c.DraftRevision = draftRev
		filtered = append(filtered, c)
	}

	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 5000 {
		pageSize = 5000
	}

	total := len(filtered)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	return map[string]interface{}{
		"items":          filtered[start:end],
		"total":          total,
		"page":           page,
		"page_size":      pageSize,
		"draft_revision": draftRev,
		"stats":          stats,
	}, nil
}

// GetPlaybookCard 按主诉编码获取单张剧本卡详情
func GetPlaybookCard(genPath, conceptCode string) (*PlaybookCard, error) {
	db, err := openSQLiteRW(genPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := EnsurePlaybookSchema(db, nil); err != nil {
		return nil, err
	}

	rows, err := db.Query(`SELECT `+playbookCardSelectCols+` FROM playbook_cards WHERE concept_code = ?`, strings.TrimSpace(conceptCode))
	if err != nil {
		return nil, err
	}
	cards, err := scanPlaybookCardRows(rows)
	_ = rows.Close()
	if err != nil {
		return nil, err
	}
	if len(cards) == 0 {
		return nil, newCompileError(404, "未找到剧本卡: %s", conceptCode)
	}
	_ = db.QueryRow(`SELECT COALESCE(draft_revision, 1) FROM playbook_draft_meta WHERE id = 1`).Scan(&cards[0].DraftRevision)
	return &cards[0], nil
}

// ListQuestionPacks 列出通用临床问诊包及全站引用次数统计
func ListQuestionPacks(genPath string) ([]QuestionPack, int, error) {
	db, err := openSQLiteRW(genPath)
	if err != nil {
		return nil, 0, err
	}
	defer db.Close()
	if err := EnsurePlaybookSchema(db, nil); err != nil {
		return nil, 0, err
	}

	refCounts := make(map[string]int)
	cRows, err := db.Query(`SELECT COALESCE(attached_pack_codes_json, '[]') FROM playbook_cards`)
	if err == nil {
		for cRows.Next() {
			var pJ string
			if cRows.Scan(&pJ) == nil {
				var pList []string
				_ = json.Unmarshal([]byte(pJ), &pList)
				for _, pc := range pList {
					refCounts[pc]++
				}
			}
		}
		_ = cRows.Close()
	}

	rows, err := db.Query(`SELECT pack_code, pack_name, clinical_domain, description, COALESCE(questions_template_json, '[]'), COALESCE(version_rev, 1), COALESCE(updated_at, '') FROM question_packs ORDER BY pack_code ASC`)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var packs []QuestionPack
	for rows.Next() {
		var p QuestionPack
		var tplJ string
		if rows.Scan(&p.PackCode, &p.PackName, &p.ClinicalDomain, &p.Description, &tplJ, &p.VersionRev, &p.UpdatedAt) == nil {
			_ = json.Unmarshal([]byte(tplJ), &p.QuestionsTemplate)
			p.RefCount = refCounts[p.PackCode]
			packs = append(packs, p)
		}
	}
	var draftRev int
	_ = db.QueryRow(`SELECT COALESCE(draft_revision, 1) FROM playbook_draft_meta WHERE id = 1`).Scan(&draftRev)
	return packs, draftRev, nil
}

// BatchDispositionPlaybooks 执行 2A 三桶批量处置并单事务重新编译受影响卡片
func BatchDispositionPlaybooks(genPath, tcmPath string, req BatchDispositionRequest) (map[string]interface{}, error) {
	codes := uniqueNonEmptyStrings(req.ConceptCodes)
	if len(codes) == 0 {
		return nil, newCompileError(400, "concept_codes 不能为空")
	}

	genDB, err := openSQLiteRW(genPath)
	if err != nil {
		return nil, err
	}
	defer genDB.Close()

	var tcmDB *sql.DB
	if strings.TrimSpace(tcmPath) != "" {
		if tdb, tErr := openSQLiteRW(tcmPath); tErr == nil {
			tcmDB = tdb
			defer tcmDB.Close()
		}
	}
	if err := EnsurePlaybookSchema(genDB, tcmDB); err != nil {
		return nil, err
	}

	genTx, err := genDB.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = genTx.Rollback() }()

	var tcmTx *sql.Tx
	if tcmDB != nil {
		tcmTx, err = tcmDB.Begin()
		if err != nil {
			return nil, err
		}
		defer func() { _ = tcmTx.Rollback() }()
	}

	curRev := readDraftRevisionTx(genTx)
	if req.ExpectedDraftRev > 0 && req.ExpectedDraftRev != curRev {
		return nil, newCompileError(409, "草稿版本冲突（当前 v%d，请求 v%d）", curRev, req.ExpectedDraftRev)
	}

	wl, err := loadWhitelistsTx(genTx, tcmTx)
	if err != nil {
		return nil, err
	}

	now := time.Now().Format(time.RFC3339)
	updated := make([]string, 0, len(codes))
	for _, code := range codes {
		rows, qErr := genTx.Query(`SELECT `+playbookCardSelectCols+` FROM playbook_cards WHERE concept_code = ?`, code)
		if qErr != nil {
			return nil, qErr
		}
		cards, sErr := scanPlaybookCardRows(rows)
		_ = rows.Close()
		if sErr != nil {
			return nil, sErr
		}
		if len(cards) == 0 {
			return nil, newCompileError(404, "批量处置未找到剧本卡 %s", code)
		}
		card := cards[0]

		switch req.Action {
		case "SET_ROLE":
			if req.TriageRole == "" {
				return nil, newCompileError(400, "SET_ROLE 需要传入 triage_role")
			}
			card.TriageRole = req.TriageRole
			if req.ApproveAfterFix {
				card.ReviewStatus = ReviewStatusApproved
			}
		case "SET_FOLLOWUP_MODE":
			if req.SkipFollowupMode == "" {
				return nil, newCompileError(400, "SET_FOLLOWUP_MODE 需要传入 skip_followup_mode")
			}
			card.SkipFollowupMode = req.SkipFollowupMode
			if strings.TrimSpace(req.DirectTriageNote) != "" {
				card.DirectTriageNote = strings.TrimSpace(req.DirectTriageNote)
			}
			if req.ApproveAfterFix {
				card.ReviewStatus = ReviewStatusApproved
			}
		case "APPLY_MISMATCH_FIX":
			targetSvc := strings.TrimSpace(req.DefaultServiceCode)
			if targetSvc == "" {
				targetSvc = strings.TrimSpace(card.MismatchAlert.SuggestedServiceCode)
			}
			if targetSvc == "" {
				return nil, newCompileError(400, "主诉 %s 无可用错配纠正建议服务", code)
			}
			oldSvc := card.DefaultServiceCode
			card.DefaultServiceCode = targetSvc
			if oldSvc != "" && oldSvc != targetSvc {
				card.BackupServiceCodes = uniqueNonEmptyStrings(append([]string{oldSvc}, card.BackupServiceCodes...))
			}
			card.MismatchAlert.HasAlert = false
			if req.ApproveAfterFix {
				card.ReviewStatus = ReviewStatusApproved
			}
		case "APPROVE_REVIEW":
			card.ReviewStatus = ReviewStatusApproved
			card.MismatchAlert.HasAlert = false
		case "ATTACH_PACK":
			card.AttachedPackCodes = uniqueNonEmptyStrings(append(card.AttachedPackCodes, req.AttachPackCodes...))
		case "UPDATE_SERVICE":
			if strings.TrimSpace(req.DefaultServiceCode) != "" {
				card.DefaultServiceCode = strings.TrimSpace(req.DefaultServiceCode)
			}
			if len(req.BackupServiceCodes) > 0 {
				card.BackupServiceCodes = req.BackupServiceCodes
			}
			if req.ApproveAfterFix {
				card.ReviewStatus = ReviewStatusApproved
			}
		default:
			return nil, newCompileError(400, "不支持的批量处置动作: %s", req.Action)
		}

		card.VersionRev++
		card.UpdatedAt = now
		if req.Operator != "" {
			card.UpdatedBy = req.Operator
		}
		if err := compileCardArtifactsTx(genTx, tcmTx, &card, wl); err != nil {
			return nil, err
		}
		updated = append(updated, code)
	}

	newRev, err := bumpDraftRevisionTx(genTx, now)
	if err != nil {
		return nil, err
	}
	if tcmTx != nil {
		if err := tcmTx.Commit(); err != nil {
			return nil, err
		}
	}
	if err := genTx.Commit(); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success":        true,
		"updated_codes":  updated,
		"updated_count":  len(updated),
		"draft_revision": newRev,
	}, nil
}

// ImportPlaybooksBatch 执行第 6 节标准 PlaybookImportV1 批量入库（支持 dry_run 预览校验不落盘）
func ImportPlaybooksBatch(genPath, tcmPath string, dryRun bool, payload PlaybookImportPayload, operator string) (map[string]interface{}, error) {
	if payload.SchemaVersion != "" && payload.SchemaVersion != "PlaybookImportV1" {
		return nil, newCompileError(400, "不支持的导入 schema_version: %s（仅支持 PlaybookImportV1）", payload.SchemaVersion)
	}
	if len(payload.Items) == 0 {
		return nil, newCompileError(400, "导入 items 不能为空")
	}

	genDB, err := openSQLiteRW(genPath)
	if err != nil {
		return nil, err
	}
	defer genDB.Close()

	var tcmDB *sql.DB
	if strings.TrimSpace(tcmPath) != "" {
		if tdb, tErr := openSQLiteRW(tcmPath); tErr == nil {
			tcmDB = tdb
			defer tcmDB.Close()
		}
	}
	if err := EnsurePlaybookSchema(genDB, tcmDB); err != nil {
		return nil, err
	}

	genTx, err := genDB.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = genTx.Rollback() }()

	var tcmTx *sql.Tx
	if tcmDB != nil {
		tcmTx, err = tcmDB.Begin()
		if err != nil {
			return nil, err
		}
		defer func() { _ = tcmTx.Rollback() }()
	}

	wl, err := loadWhitelistsTx(genTx, tcmTx)
	if err != nil {
		return nil, err
	}

	now := time.Now().Format(time.RFC3339)
	diffRows := make([]map[string]interface{}, 0, len(payload.Items))

	for idx, item := range payload.Items {
		rfs := item.RedFlags
		if len(rfs) == 0 && len(item.RedFlagRules) > 0 {
			rfs = item.RedFlagRules
		}
		revStatus := strings.TrimSpace(item.ReviewStatus)
		if revStatus == "" {
			revStatus = ReviewStatusAISuggested
		}

		var mismatch PlaybookMismatchAlert
		if item.MismatchAlert != nil {
			mismatch = *item.MismatchAlert
		} else if item.MismatchCorrection != nil {
			mismatch = *item.MismatchCorrection
			if mismatch.PreviousServiceCode != "" && mismatch.PreviousServiceCode != item.DefaultServiceCode {
				mismatch.HasAlert = true
				if mismatch.SuggestedServiceCode == "" {
					mismatch.SuggestedServiceCode = item.DefaultServiceCode
				}
			}
		}

		var prevService, prevRole string
		var prevRev int
		existed := false
		if err := genTx.QueryRow(`SELECT default_service_code, triage_role, COALESCE(version_rev, 0) FROM playbook_cards WHERE concept_code = ?`, strings.TrimSpace(item.ConceptCode)).Scan(&prevService, &prevRole, &prevRev); err == nil {
			existed = true
		} else if err := genTx.QueryRow(`SELECT default_service_code, COALESCE(triage_role, 'CORE_COMPLAINT') FROM symptoms WHERE concept_code = ?`, strings.TrimSpace(item.ConceptCode)).Scan(&prevService, &prevRole); err == nil {
			existed = true
		}

		card := PlaybookCard{
			ConceptCode:        item.ConceptCode,
			ConceptName:        item.ConceptName,
			TriageRole:         item.TriageRole,
			SkipFollowupMode:   item.SkipFollowupMode,
			DirectTriageNote:   item.DirectTriageNote,
			BodyPart:           item.BodyPart,
			ColloquialAliases:  item.ColloquialAliases,
			GenderLimit:        item.GenderLimit,
			AgeGroupLimit:      item.AgeGroupLimit,
			DefaultServiceCode: item.DefaultServiceCode,
			BackupServiceCodes: item.BackupServiceCodes,
			RedFlagRules:       rfs,
			AttachedPackCodes:  item.AttachedPackCodes,
			CustomQuestions:    item.CustomQuestions,
			TCMBinding:         item.TCMBinding,
			TextbookCitation:   item.TextbookCitation,
			ReviewStatus:       revStatus,
			MismatchAlert:      mismatch,
			VersionRev:         prevRev + 1,
			UpdatedAt:          now,
			UpdatedBy:          operator,
		}

		// 不论 dry_run 是否为 true，都运行完整事务级编译校验（包含题目与红旗绑定的存在性检验）
		if err := compileCardArtifactsTx(genTx, tcmTx, &card, wl); err != nil {
			return nil, newCompileError(httpCodeFromError(err, 400), "导入第 %d 项 (%s) 失败: %v", idx+1, item.ConceptCode, err)
		}

		diffRows = append(diffRows, map[string]interface{}{
			"concept_code":         card.ConceptCode,
			"concept_name":         card.ConceptName,
			"is_new":               !existed,
			"previous_service":     prevService,
			"next_service":         card.DefaultServiceCode,
			"previous_role":        prevRole,
			"next_role":            card.TriageRole,
			"compiled_packs":       len(card.AttachedPackCodes),
			"compiled_custom_q":    len(card.CustomQuestions),
			"compiled_red_flags":   len(card.RedFlagRules),
			"review_status":        card.ReviewStatus,
		})
	}

	if dryRun {
		// Dry-Run 不提交事务（由 defer Rollback 自动还原所有更改）
		return map[string]interface{}{
			"success":         true,
			"dry_run":         true,
			"import_batch_id": payload.ImportBatchID,
			"item_count":      len(diffRows),
			"preview_diff":    diffRows,
			"draft_revision":  readDraftRevisionTx(genTx),
		}, nil
	}

	newDraftRev, err := bumpDraftRevisionTx(genTx, now)
	if err != nil {
		return nil, err
	}
	if tcmTx != nil {
		if err := tcmTx.Commit(); err != nil {
			return nil, err
		}
	}
	if err := genTx.Commit(); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success":         true,
		"dry_run":         false,
		"import_batch_id": payload.ImportBatchID,
		"imported_count":  len(diffRows),
		"preview_diff":    diffRows,
		"draft_revision":  newDraftRev,
	}, nil
}

func normalizeAgeGroupTag(raw string) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "", "ALL":
		return "ALL"
	case "CHILD", "PEDIATRIC":
		return "PEDIATRIC"
	case "ADULT":
		return "ADULT"
	case "GERIATRIC", "ELDERLY", "SENIOR":
		return "ELDERLY"
	default:
		return strings.ToUpper(strings.TrimSpace(raw))
	}
}
