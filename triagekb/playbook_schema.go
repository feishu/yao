package triagekb

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// TriageRole 常量：定义主诉节点在导诊系统中的角色定位
const (
	TriageRoleCoreComplaint = "CORE_COMPLAINT"
	TriageRoleCompanionOnly = "COMPANION_ONLY"
	TriageRoleTBDeferred    = "TB_DEFERRED"
	TriageRoleArchived      = "ARCHIVED"
)

// SkipFollowupMode 常量：定义追问策略
const (
	SkipFollowupRequired        = "REQUIRED"
	SkipFollowupDirectSingleHop = "DIRECT_SINGLE_HOP"
)

// ReviewStatus 常量：定义剧本卡审核状态
const (
	ReviewStatusApproved    = "APPROVED"
	ReviewStatusNeedsReview = "NEEDS_REVIEW"
	ReviewStatusAISuggested = "AI_SUGGESTED"
)

// RedFlagTriggerMode 常量：红旗规则触发模式（严禁空触发条件）
const (
	TriggerModeOptionAnswer  = "OPTION_ANSWER"
	TriggerModeConceptDirect = "CONCEPT_DIRECT"
	TriggerModeSignMatch     = "SIGN_MATCH"
)

// PlaybookQuestionOption 通用问诊包与专属追问题统一选项规范（QuestionOptionV1）
type PlaybookQuestionOption struct {
	OptionKey        string   `json:"option_key"`
	Value            string   `json:"value,omitempty"`
	Label            string   `json:"label"`
	IsRedFlag        bool     `json:"is_red_flag,omitempty"`
	IsUncertain      bool     `json:"is_uncertain,omitempty"`
	ServiceHint      string   `json:"service_hint,omitempty"`
	BoostWeight      float64  `json:"boost_weight,omitempty"`
	TCMSyndromeHints []string `json:"tcm_syndrome_hints,omitempty"`
}

// PlaybookQuestionTemplate 问诊包模板题或剧本卡专属追问题定义
type PlaybookQuestionTemplate struct {
	QuestionID        string                   `json:"question_id,omitempty"`
	SlotKey           string                   `json:"slot_key"`
	Title             string                   `json:"title"`
	Category          string                   `json:"category"` // RED_FLAG | DIFFERENTIAL | COURSE_OF_DISEASE | TCM | DEMOGRAPHIC
	Type              string                   `json:"type"`     // SINGLE_CHOICE | MULTI_CHOICE
	EvidenceDimension string                   `json:"evidence_dimension,omitempty"`
	Options           []PlaybookQuestionOption `json:"options"`
}

// PlaybookRedFlagRule 剧本卡急诊红旗定义（结构化编译为 safety_rules）
type PlaybookRedFlagRule struct {
	RuleID              string   `json:"rule_id,omitempty"`
	RuleSuffix          string   `json:"rule_suffix"`
	Title               string   `json:"title"`
	Level               string   `json:"level"`        // CRITICAL | URGENT
	SafetyState         string   `json:"safety_state"` // EMERGENCY_NOW | URGENT_ASSESSMENT
	TriggerMode         string   `json:"trigger_mode"` // OPTION_ANSWER | CONCEPT_DIRECT | SIGN_MATCH
	BindQuestionID      string   `json:"bind_question_id,omitempty"`
	BindSlotKey         string   `json:"bind_slot_key,omitempty"`
	BindOptionKey       string   `json:"bind_option_key,omitempty"`
	BindOptionValue     string   `json:"bind_option_value,omitempty"`
	TriggerConcepts     []string `json:"trigger_concepts,omitempty"`
	TriggerSignCodes    []string `json:"trigger_sign_codes,omitempty"`
	RawTriggerJSON      string   `json:"raw_trigger_json,omitempty"`
	SymptomKeywords     []string `json:"symptom_keywords,omitempty"`
	RecommendedDeptName string   `json:"recommended_dept_name"`
	FallbackServiceCode string   `json:"fallback_service_code,omitempty"`
	Call120             bool     `json:"call_120"`
	WarningText         string   `json:"warning_text"`
	GuideText           string   `json:"guide_text"`
}

// PlaybookTCMBinding 中医辨证与 6 大可挂号专病门诊绑定
type PlaybookTCMBinding struct {
	Syndromes     []string `json:"syndromes,omitempty"`
	Clinics       []string `json:"clinics,omitempty"`
	Therapies     []string `json:"therapies,omitempty"`
	EvidenceQuote string   `json:"evidence_quote,omitempty"`
}

// PlaybookDiffNote 教材鉴别诊断依据条目
type PlaybookDiffNote struct {
	DiseaseName           string `json:"disease_name"`
	DistinguishingFeatures string `json:"distinguishing_features"`
}

// PlaybookTextbookCitation 教材来源与摘要依据
type PlaybookTextbookCitation struct {
	TextbookSource    string             `json:"textbook_source,omitempty"`
	Chapter           string             `json:"chapter,omitempty"`
	Page              string             `json:"page,omitempty"`
	EvidenceSummary   string             `json:"evidence_summary,omitempty"`
	DifferentialNotes []PlaybookDiffNote `json:"differential_notes,omitempty"`
}

// PlaybookMismatchAlert 存量主诉错配纠偏提示或教材对齐记录
type PlaybookMismatchAlert struct {
	HasAlert             bool   `json:"has_alert,omitempty"`
	PreviousServiceCode  string `json:"previous_service_code,omitempty"`
	SuggestedServiceCode string `json:"suggested_service_code,omitempty"`
	Reason               string `json:"reason,omitempty"`
	Severity             string `json:"severity,omitempty"`
}

// PlaybookCard 三段式分诊剧本卡（Approach C 医务编排核心源表模型）
type PlaybookCard struct {
	ConceptCode        string                     `json:"concept_code"`
	ConceptName        string                     `json:"concept_name"`
	TriageRole         string                     `json:"triage_role"`
	SkipFollowupMode   string                     `json:"skip_followup_mode"`
	DirectTriageNote   string                     `json:"direct_triage_note"`
	BodyPart           string                     `json:"body_part"`
	ColloquialAliases  []string                   `json:"colloquial_aliases"`
	GenderLimit        string                     `json:"gender_limit"`
	AgeGroupLimit      string                     `json:"age_group_limit"`
	DefaultServiceCode string                     `json:"default_service_code"`
	BackupServiceCodes []string                   `json:"backup_service_codes"`
	RedFlagRules       []PlaybookRedFlagRule      `json:"red_flag_rules"`
	AttachedPackCodes  []string                   `json:"attached_pack_codes"`
	CustomQuestions    []PlaybookQuestionTemplate `json:"custom_questions"`
	TCMBinding         PlaybookTCMBinding         `json:"tcm_binding"`
	TextbookCitation   PlaybookTextbookCitation   `json:"textbook_citation"`
	ReviewStatus       string                     `json:"review_status"`
	MismatchAlert      PlaybookMismatchAlert      `json:"mismatch_alert"`
	VersionRev         int                        `json:"version_rev"`
	UpdatedAt          string                     `json:"updated_at"`
	UpdatedBy          string                     `json:"updated_by"`
	DraftRevision      int                        `json:"draft_revision,omitempty"`
	NeedsReviewFlag    bool                       `json:"needs_review,omitempty"`
}

// QuestionPack 标准化临床通用问诊包（~20 套可复用问诊模块）
type QuestionPack struct {
	PackCode          string                     `json:"pack_code"`
	PackName          string                     `json:"pack_name"`
	ClinicalDomain    string                     `json:"clinical_domain"`
	Description       string                     `json:"description"`
	QuestionsTemplate []PlaybookQuestionTemplate `json:"questions_template"`
	VersionRev        int                        `json:"version_rev"`
	UpdatedAt         string                     `json:"updated_at"`
	RefCount          int                        `json:"ref_count,omitempty"`
}

// PlaybookDraftMeta 草稿乐观锁与沙盒/回归对比元数据
type PlaybookDraftMeta struct {
	DraftRevision         int                    `json:"draft_revision"`
	LastCompiledAt        string                 `json:"last_compiled_at"`
	CompilerVersion       string                 `json:"compiler_version"`
	LastGoldenDiffSummary map[string]interface{} `json:"last_golden_diff_summary"`
}

// EnsurePlaybookSchema 幂等创建表里分离业务源表、向后兼容扩列并 Seed 20 套标准临床问诊包
func EnsurePlaybookSchema(genDB *sql.DB, _ *sql.DB) error {
	if genDB == nil {
		return fmt.Errorf("genDB 不能为空")
	}

	// 1. 向后兼容扩列 symptoms / questions / textbook_evidence
	if err := ensureColumnExists(genDB, "symptoms", "triage_role", "TEXT NOT NULL DEFAULT 'CORE_COMPLAINT'"); err != nil {
		return err
	}
	if err := ensureColumnExists(genDB, "symptoms", "skip_followup_mode", "TEXT NOT NULL DEFAULT 'REQUIRED'"); err != nil {
		return err
	}
	if err := ensureColumnExists(genDB, "symptoms", "direct_triage_note", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumnExists(genDB, "symptoms", "colloquial_aliases_json", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
		return err
	}
	if err := ensureColumnExists(genDB, "questions", "slot_key", "VARCHAR(64) NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumnExists(genDB, "textbook_evidence", "evidence_summary", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}

	// 2. 新建 3 张表里分离核心表
	ddlStatements := []string{
		`CREATE TABLE IF NOT EXISTS playbook_cards (
			concept_code VARCHAR(64) PRIMARY KEY,
			concept_name VARCHAR(128) NOT NULL,
			triage_role VARCHAR(32) NOT NULL DEFAULT 'CORE_COMPLAINT',
			skip_followup_mode VARCHAR(32) NOT NULL DEFAULT 'REQUIRED',
			direct_triage_note TEXT NOT NULL DEFAULT '',
			body_part VARCHAR(64) NOT NULL DEFAULT 'GENERAL',
			colloquial_aliases_json TEXT NOT NULL DEFAULT '[]',
			gender_limit VARCHAR(32) NOT NULL DEFAULT 'ALL',
			age_group_limit VARCHAR(32) NOT NULL DEFAULT 'ALL',
			default_service_code VARCHAR(64) NOT NULL,
			backup_service_codes_json TEXT NOT NULL DEFAULT '[]',
			red_flag_rules_json TEXT NOT NULL DEFAULT '[]',
			attached_pack_codes_json TEXT NOT NULL DEFAULT '[]',
			custom_questions_json TEXT NOT NULL DEFAULT '[]',
			tcm_binding_json TEXT NOT NULL DEFAULT '{}',
			textbook_citation_json TEXT NOT NULL DEFAULT '{}',
			review_status VARCHAR(32) NOT NULL DEFAULT 'NEEDS_REVIEW',
			mismatch_alert_json TEXT NOT NULL DEFAULT '{}',
			version_rev INTEGER NOT NULL DEFAULT 1,
			updated_at VARCHAR(64) NOT NULL DEFAULT '',
			updated_by VARCHAR(64) NOT NULL DEFAULT ''
		);`,
		`CREATE INDEX IF NOT EXISTS idx_playbook_role ON playbook_cards(triage_role);`,
		`CREATE INDEX IF NOT EXISTS idx_playbook_review ON playbook_cards(review_status);`,
		`CREATE INDEX IF NOT EXISTS idx_playbook_service ON playbook_cards(default_service_code);`,
		`CREATE TABLE IF NOT EXISTS question_packs (
			pack_code VARCHAR(64) PRIMARY KEY,
			pack_name VARCHAR(128) NOT NULL,
			clinical_domain VARCHAR(64) NOT NULL DEFAULT 'GENERAL',
			description TEXT NOT NULL DEFAULT '',
			questions_template_json TEXT NOT NULL DEFAULT '[]',
			version_rev INTEGER NOT NULL DEFAULT 1,
			updated_at VARCHAR(64) NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE IF NOT EXISTS playbook_draft_meta (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			draft_revision INTEGER NOT NULL DEFAULT 1,
			last_compiled_at VARCHAR(64) NOT NULL DEFAULT '',
			compiler_version VARCHAR(32) NOT NULL DEFAULT 'playbook-v1',
			last_golden_diff_summary_json TEXT NOT NULL DEFAULT '{}'
		);`,
		`INSERT OR IGNORE INTO playbook_draft_meta (id, draft_revision, compiler_version, last_golden_diff_summary_json)
		 VALUES (1, 1, 'playbook-v1', '{}');`,
	}

	for _, stmt := range ddlStatements {
		if _, err := genDB.Exec(stmt); err != nil {
			return fmt.Errorf("初始化 Playbook 表结构失败: %w", err)
		}
	}

	// 3. 幂等写入 ~20 套标准临床通用问诊包
	if err := seedDefaultQuestionPacks(genDB); err != nil {
		return err
	}
	return nil
}

func ensureColumnExists(db *sql.DB, tableName, colName, alterDef string) error {
	cols := sqliteTableColumns(db, tableName)
	if len(cols) == 0 || cols[colName] {
		return nil
	}
	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, colName, alterDef)
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("向 %s 扩列 %s 失败: %w", tableName, colName, err)
	}
	return nil
}

func seedDefaultQuestionPacks(genDB *sql.DB) error {
	now := time.Now().Format(time.RFC3339)
	packs := buildDefaultQuestionPacks(now)
	for _, pack := range packs {
		tplBytes, err := json.Marshal(pack.QuestionsTemplate)
		if err != nil {
			return fmt.Errorf("序列化问诊包 %s 失败: %w", pack.PackCode, err)
		}
		if _, err := genDB.Exec(`
			INSERT OR IGNORE INTO question_packs (
				pack_code, pack_name, clinical_domain, description, questions_template_json, version_rev, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?)
		`, pack.PackCode, pack.PackName, pack.ClinicalDomain, pack.Description, string(tplBytes), 1, now); err != nil {
			return fmt.Errorf("写入内置问诊包 %s 失败: %w", pack.PackCode, err)
		}
	}
	return nil
}

// buildDefaultQuestionPacks 返回 PRD 第 5.2 节规范的 20 套标准通用临床问诊包
func buildDefaultQuestionPacks(now string) []QuestionPack {
	return []QuestionPack{
		{
			PackCode:       "PACK_PAIN_OLDCARTS",
			PackName:       "疼痛性质与放射特征评估",
			ClinicalDomain: "GENERAL",
			Description:    "用于区分绞痛、钝痛、压榨放射痛与按压肌肉骨痛",
			VersionRev:     1,
			UpdatedAt:      now,
			QuestionsTemplate: []PlaybookQuestionTemplate{
				{
					SlotKey:           "pain_radiation",
					Title:             "疼痛的感觉及是否向其他部位放射？",
					Category:          "DIFFERENTIAL",
					Type:              "SINGLE_CHOICE",
					EvidenceDimension: "DIM_PAIN_OLDCARTS",
					Options: []PlaybookQuestionOption{
						{OptionKey: "CRUSHING_RADIATING", Label: "沉重压榨感，并向肩背/左臂/下颌放射", IsRedFlag: true, ServiceHint: "CARDIO_ASSESSMENT", BoostWeight: 0.9},
						{OptionKey: "SPASM_COLIC", Label: "阵发性绞痛或钻顶样痛，常伴恶心呕吐", ServiceHint: "GASTRO_INTERNAL_ASSESSMENT", BoostWeight: 0.8},
						{OptionKey: "FIXED_TENDERNESS", Label: "按压某一位置更痛，或随翻身抬手加重", ServiceHint: "ORTHOPEDIC_SPINE_ASSESSMENT", BoostWeight: 0.85},
						{OptionKey: "DULL_ACHE", Label: "隐隐作痛或酸胀，无明确放射", ServiceHint: "GENERAL_INTERNAL_ASSESSMENT", BoostWeight: 0.5},
					},
				},
			},
		},
		{
			PackCode:       "PACK_PAIN_DURATION",
			PackName:       "起病缓急与持续时长",
			ClinicalDomain: "COURSE",
			Description:    "用于识别突发急症与慢性复发过程，并在 E1 自动依据口语时长跳过",
			VersionRev:     1,
			UpdatedAt:      now,
			QuestionsTemplate: []PlaybookQuestionTemplate{
				{
					SlotKey:           "onset_acute",
					Title:             "这次不适是从什么时候开始的？",
					Category:          "COURSE_OF_DISEASE",
					Type:              "SINGLE_CHOICE",
					EvidenceDimension: "DIM_ONSET_DURATION",
					Options: []PlaybookQuestionOption{
						{OptionKey: "ONSET_ACUTE", Label: "突发或 24 小时内急剧出现/加重", BoostWeight: 0.6},
						{OptionKey: "DAYS_TO_WEEKS", Label: "持续数天到 2 周左右", BoostWeight: 0.4},
						{OptionKey: "ONSET_CHRONIC", Label: "断断续续超过 1 个月（反复慢性不适）", ServiceHint: "GENERAL_INTERNAL_ASSESSMENT", BoostWeight: 0.35},
					},
				},
			},
		},
		{
			PackCode:       "PACK_FEVER_RED_FLAG",
			PackName:       "发热寒战与神志筛查",
			ClinicalDomain: "INFECTIOUS",
			Description:    "通用发热红旗筛查包，支持口语“没有发烧/发高烧”自动跳过",
			VersionRev:     1,
			UpdatedAt:      now,
			QuestionsTemplate: []PlaybookQuestionTemplate{
				{
					SlotKey:           "has_fever",
					Title:             "是否伴有发热（体温超过 37.3℃）、寒战或精神萎靡？",
					Category:          "RED_FLAG",
					Type:              "SINGLE_CHOICE",
					EvidenceDimension: "DIM_FEVER_SCREEN",
					Options: []PlaybookQuestionOption{
						{OptionKey: "HIGH_FEVER", Label: "高热（38.5℃以上）或伴寒战打哆嗦、神志不清", IsRedFlag: true, ServiceHint: "INFECTIOUS_ASSESSMENT", BoostWeight: 0.9},
						{OptionKey: "LOW_FEVER", Label: "低热或中度发热（37.3℃~38.4℃），意识清醒", ServiceHint: "INFECTIOUS_ASSESSMENT", BoostWeight: 0.7},
						{OptionKey: "NO_FEVER", Label: "没有发烧，体温正常", BoostWeight: 0.0},
					},
				},
			},
		},
		{
			PackCode:       "PACK_BLEEDING_RISK",
			PackName:       "出血部位、出血量与休克征",
			ClinicalDomain: "EMERGENCY",
			Description:    "识别大出血、头晕冷汗休克前驱与黏膜反复出血倾向",
			VersionRev:     1,
			UpdatedAt:      now,
			QuestionsTemplate: []PlaybookQuestionTemplate{
				{
					SlotKey:           "bleeding_alarm",
					Title:             "出血量有多少，是否伴有头晕心慌、出冷汗或晕厥？",
					Category:          "RED_FLAG",
					Type:              "SINGLE_CHOICE",
					EvidenceDimension: "DIM_BLEEDING_RISK",
					Options: []PlaybookQuestionOption{
						{OptionKey: "MASSIVE_SHOCK", Label: "出血量大且伴头晕眼前发黑、冷汗心慌或休克感", IsRedFlag: true, ServiceHint: "GASTRO_INTERNAL_ASSESSMENT", BoostWeight: 0.95},
						{OptionKey: "MODERATE_PERSIST", Label: "反复多部位或持续渗血超过 20 分钟难止", IsRedFlag: true, ServiceHint: "HEMATOLOGY_ASSESSMENT", BoostWeight: 0.85},
						{OptionKey: "MILD_SPOT", Label: "仅有轻微血丝或少量点状出血，无头晕心慌", ServiceHint: "HEMATOLOGY_ASSESSMENT", BoostWeight: 0.6},
						{OptionKey: "NO_BLEEDING", Label: "没有出血症状", BoostWeight: 0.0},
					},
				},
			},
		},
		{
			PackCode:       "PACK_TRAUMA_MECHANISM",
			PackName:       "外伤机制与肢体活动功能",
			ClinicalDomain: "TRAUMA",
			Description:    "区分高能量外伤骨折风险、开放出血与一般软组织挫伤",
			VersionRev:     1,
			UpdatedAt:      now,
			QuestionsTemplate: []PlaybookQuestionTemplate{
				{
					SlotKey:           "trauma_mech",
					Title:             "是否有明确外伤或跌倒史？受伤部位外观与活动情况如何？",
					Category:          "RED_FLAG",
					Type:              "SINGLE_CHOICE",
					EvidenceDimension: "DIM_TRAUMA_MECHANISM",
					Options: []PlaybookQuestionOption{
						{OptionKey: "SEVERE_TRAUMA", Label: "高处跌落/车祸撞击，伴明显畸形、骨擦音或完全无法活动", IsRedFlag: true, ServiceHint: "TRAUMA_SURGERY_ASSESSMENT", BoostWeight: 0.95},
						{OptionKey: "MODERATE_WOUND", Label: "开放裂口出血或明显肿胀剧痛、勉强着地", ServiceHint: "TRAUMA_SURGERY_ASSESSMENT", BoostWeight: 0.85},
						{OptionKey: "MILD_CONTUSION", Label: "轻微扭伤或磕碰，关节可正常活动", ServiceHint: "ORTHOPEDIC_SPINE_ASSESSMENT", BoostWeight: 0.6},
						{OptionKey: "NO_TRAUMA", Label: "无外伤或碰撞史", BoostWeight: 0.0},
					},
				},
			},
		},
		{
			PackCode:       "PACK_CHEST_CARDIOPULMONARY",
			PackName:       "心前区压榨感、放射与冷汗",
			ClinicalDomain: "CARDIOPULMONARY",
			Description:    "用于胸痛/胸闷鉴别心源性危象、肺部胸膜痛与反流性胃食管痛",
			VersionRev:     1,
			UpdatedAt:      now,
			QuestionsTemplate: []PlaybookQuestionTemplate{
				{
					SlotKey:           "chest_radiation",
					Title:             "胸部不适的具体性质及伴随特征更符合哪一项？",
					Category:          "RED_FLAG",
					Type:              "SINGLE_CHOICE",
					EvidenceDimension: "DIM_CHEST_CHARACTER",
					Options: []PlaybookQuestionOption{
						{OptionKey: "CRUSHING_SWEAT", Label: "大石压榨紧缩感，伴大汗淋漓或向左肩背放射", IsRedFlag: true, ServiceHint: "CARDIO_ASSESSMENT", BoostWeight: 1.0},
						{OptionKey: "TEARING_BACK", Label: "突发如刀割撕裂样剧痛，直达肩胛间/后背", IsRedFlag: true, ServiceHint: "THORACIC_SURGERY_ASSESSMENT", BoostWeight: 1.0},
						{OptionKey: "PLEURITIC", Label: "咳嗽或深吸气时明显牵扯刺痛", ServiceHint: "RESPIRATORY_ASSESSMENT", BoostWeight: 0.8},
						{OptionKey: "REFLUX_BURNING", Label: "胸骨后火辣辣烧灼感，伴反酸、饱餐后平卧加重", ServiceHint: "GASTRO_INTERNAL_ASSESSMENT", BoostWeight: 0.85},
						{OptionKey: "WALL_PRESS", Label: "按压胸肋骨某一点明显痛", ServiceHint: "THORACIC_SURGERY_ASSESSMENT", BoostWeight: 0.75},
					},
				},
			},
		},
		{
			PackCode:       "PACK_ABDOMINAL_ALARM",
			PackName:       "急腹症肌紧张、转移痛与黑便",
			ClinicalDomain: "GASTROINTESTINAL",
			Description:    "快速区分腹膜炎急腹症、外科阑尾/胆道绞痛、消化道出血与内科消化不良",
			VersionRev:     1,
			UpdatedAt:      now,
			QuestionsTemplate: []PlaybookQuestionTemplate{
				{
					SlotKey:           "abd_alarm",
					Title:             "肚子痛的位置特点及按压反应最符合以下哪种？",
					Category:          "RED_FLAG",
					Type:              "SINGLE_CHOICE",
					EvidenceDimension: "DIM_ABDOMINAL_ALARM",
					Options: []PlaybookQuestionOption{
						{OptionKey: "RIGID_ABDOMEN", Label: "全腹剧痛，肚子摸起来像木板一样硬（拒绝按压）", IsRedFlag: true, ServiceHint: "GENERAL_SURGERY_ASSESSMENT", BoostWeight: 1.0},
						{OptionKey: "MIGRATORY_COLIC", Label: "从上腹转移到右下腹痛，或右上腹绞痛向右肩背放射", ServiceHint: "GASTRO_SURGERY_ASSESSMENT", BoostWeight: 0.9},
						{OptionKey: "GI_BLEED", Label: "伴有柏油样黑便、暗红血便或呕血/咖啡样物", IsRedFlag: true, ServiceHint: "GASTRO_INTERNAL_ASSESSMENT", BoostWeight: 0.95},
						{OptionKey: "EPIGASTRIC_DYSPEPSIA", Label: "上腹隐痛腹胀、反酸嗳气或轻度腹泻", ServiceHint: "GASTRO_INTERNAL_ASSESSMENT", BoostWeight: 0.85},
					},
				},
			},
		},
		{
			PackCode:       "PACK_NEURO_STROKE_FAST",
			PackName:       "脑卒中 FAST 与突发剧痛筛查",
			ClinicalDomain: "NEUROLOGY",
			Description:    "筛查口角歪斜、单侧肢体无力、言语含糊及颅高压危象",
			VersionRev:     1,
			UpdatedAt:      now,
			QuestionsTemplate: []PlaybookQuestionTemplate{
				{
					SlotKey:           "neuro_fast",
					Title:             "是否突发以下任一神经系统警报表现？",
					Category:          "RED_FLAG",
					Type:              "SINGLE_CHOICE",
					EvidenceDimension: "DIM_NEURO_FAST",
					Options: []PlaybookQuestionOption{
						{OptionKey: "FAST_POSITIVE", Label: "突然口角歪斜、一侧手脚麻木无力、吐字不清或霹雳样炸裂剧痛", IsRedFlag: true, ServiceHint: "NEURO_INTERNAL_ASSESSMENT", BoostWeight: 1.0, TCMSyndromeHints: []string{"TCM_SYN_WIND_PHLEGM_STRIKE"}},
						{OptionKey: "POSITIONAL_VERTIGO", Label: "翻身或转头时天旋地转，无偏瘫口斜和言语障碍", ServiceHint: "ENT_ASSESSMENT", BoostWeight: 0.8},
						{OptionKey: "TYPICAL_HEADACHE", Label: "无偏瘫口斜，仅搏动性胀痛或紧箍感头痛", ServiceHint: "NEURO_INTERNAL_ASSESSMENT", BoostWeight: 0.75, TCMSyndromeHints: []string{"TCM_SYN_QI_STAGNATION_BLOOD_STASIS"}},
					},
				},
			},
		},
		{
			PackCode:       "PACK_RESPIRATORY_DIFF",
			PackName:       "呼吸困难、喘息、发绀与端坐呼吸",
			ClinicalDomain: "RESPIRATORY",
			Description:    "识别急性呼吸窘迫/窒息与端坐呼吸心源性哮喘",
			VersionRev:     1,
			UpdatedAt:      now,
			QuestionsTemplate: []PlaybookQuestionTemplate{
				{
					SlotKey:           "dyspnea_sev",
					Title:             "气短、喘息或呼吸不畅对说话和躺卧有什么影响？",
					Category:          "RED_FLAG",
					Type:              "SINGLE_CHOICE",
					EvidenceDimension: "DIM_RESP_DISTRESS",
					Options: []PlaybookQuestionOption{
						{OptionKey: "AIRWAY_DISTRESS", Label: "喘不过气无法说完整一句话、嘴唇发紫，或必须坐着才能呼吸", IsRedFlag: true, ServiceHint: "RESPIRATORY_ASSESSMENT", BoostWeight: 1.0},
						{OptionKey: "EXERTIONAL_WHEEZE", Label: "活动或快走后气急喘息，安静休息稍缓解", ServiceHint: "RESPIRATORY_ASSESSMENT", BoostWeight: 0.85},
						{OptionKey: "MILD_DYSPNEA", Label: "仅偶尔感觉胸闷气短、需要长出气", ServiceHint: "CARDIO_ASSESSMENT", BoostWeight: 0.65},
					},
				},
			},
		},
		{
			PackCode:       "PACK_GYN_OB_PREGNANCY",
			PackName:       "生理性别、末次月经与孕产状态确认",
			ClinicalDomain: "GYN_OB",
			Description:    "用于未知性别/孕龄期腹痛出血患者，安全甄别产科与妇科危症",
			VersionRev:     1,
			UpdatedAt:      now,
			QuestionsTemplate: []PlaybookQuestionTemplate{
				{
					SlotKey:           "is_pregnant",
					Title:             "为保障用药与检查安全，请确认您的生理性别及当前是否有怀孕/停经情况：",
					Category:          "DIFFERENTIAL",
					Type:              "SINGLE_CHOICE",
					EvidenceDimension: "DIM_GYN_OB_STATUS",
					Options: []PlaybookQuestionOption{
						{OptionKey: "PREGNANT_BLEED_PAIN", Label: "女性，已确认怀孕或停经，且伴随腹痛或阴道出血", IsRedFlag: true, ServiceHint: "OBSTETRICS_ASSESSMENT", BoostWeight: 1.0},
						{OptionKey: "PREGNANT_YES", Label: "女性，目前怀孕中（无急剧腹痛出血）", ServiceHint: "OBSTETRICS_ASSESSMENT", BoostWeight: 0.9},
						{OptionKey: "FEMALE_NON_PREG", Label: "女性，明确未怀孕且月经规律", ServiceHint: "GYNECOLOGY_ASSESSMENT", BoostWeight: 0.8},
						{OptionKey: "MALE_PATIENT", Label: "男性患者", ServiceHint: "GENERAL_INTERNAL_ASSESSMENT", BoostWeight: 0.35},
					},
				},
			},
		},
		{
			PackCode:       "PACK_PEDIATRIC_TRIAGE",
			PackName:       "儿科精神反应、拒奶与高热惊厥",
			ClinicalDomain: "PEDIATRICS",
			Description:    "专用于 14 岁以下患儿精神萎靡、犬吠样咳及惊厥拦截",
			VersionRev:     1,
			UpdatedAt:      now,
			QuestionsTemplate: []PlaybookQuestionTemplate{
				{
					SlotKey:           "ped_alarm",
					Title:             "孩子的精神状态、呼吸声和抽搐情况如何？",
					Category:          "RED_FLAG",
					Type:              "SINGLE_CHOICE",
					EvidenceDimension: "DIM_PED_ALARM",
					Options: []PlaybookQuestionOption{
						{OptionKey: "PED_CRITICAL", Label: "精神萎靡嗜睡、拒奶、高热惊厥抽风或声音嘶哑犬吠样咳嗽", IsRedFlag: true, ServiceHint: "PEDIATRIC_INTERNAL_ASSESSMENT", BoostWeight: 1.0},
						{OptionKey: "PED_SURGICAL", Label: "持续剧烈哭闹腹部包块、果酱样大便或外伤骨折", IsRedFlag: true, ServiceHint: "PEDIATRIC_SURGERY_ASSESSMENT", BoostWeight: 0.95},
						{OptionKey: "PED_MILD", Label: "精神玩耍尚可、进食基本正常", ServiceHint: "PEDIATRIC_INTERNAL_ASSESSMENT", BoostWeight: 0.85},
					},
				},
			},
		},
		{
			PackCode:       "PACK_URINARY_RENAL",
			PackName:       "肉眼血尿、尿痛尿急与腰肋绞痛",
			ClinicalDomain: "URINARY",
			Description:    "区分泌尿外科结石/尿路刺激与肾内科水肿蛋白尿",
			VersionRev:     1,
			UpdatedAt:      now,
			QuestionsTemplate: []PlaybookQuestionTemplate{
				{
					SlotKey:           "urinary_red",
					Title:             "排尿或腰部不适伴有以下哪种表现？",
					Category:          "DIFFERENTIAL",
					Type:              "SINGLE_CHOICE",
					EvidenceDimension: "DIM_URINARY_RENAL",
					Options: []PlaybookQuestionOption{
						{OptionKey: "RENAL_COLIC_HEMATURIA", Label: "突发单侧腰腹部剧烈绞痛向会阴放射，或肉眼血尿伴血块", IsRedFlag: true, ServiceHint: "UROLOGY_ASSESSMENT", BoostWeight: 0.95},
						{OptionKey: "UTI_IRRITATION", Label: "尿频、尿急、排尿灼痛或排尿困难", ServiceHint: "UROLOGY_ASSESSMENT", BoostWeight: 0.85},
						{OptionKey: "NEPHRITIC_EDEMA", Label: "眼睑/双下肢明显浮肿、尿中大量持久泡沫或尿量骤减", ServiceHint: "NEPHROLOGY_ASSESSMENT", BoostWeight: 0.9},
					},
				},
			},
		},
		{
			PackCode:       "PACK_ENT_AIRWAY",
			PackName:       "咽喉梗阻呼吸不畅、突聋与鼻衄难止",
			ClinicalDomain: "ENT",
			Description:    "用于耳鼻咽喉急症如急性会厌炎吸气性呼吸困难与 72 小时突聋",
			VersionRev:     1,
			UpdatedAt:      now,
			QuestionsTemplate: []PlaybookQuestionTemplate{
				{
					SlotKey:           "ent_urgent",
					Title:             "耳鼻咽喉症状是否伴随呼吸吞咽受阻或剧烈出血？",
					Category:          "RED_FLAG",
					Type:              "SINGLE_CHOICE",
					EvidenceDimension: "DIM_ENT_URGENT",
					Options: []PlaybookQuestionOption{
						{OptionKey: "AIRWAY_OR_BLEED", Label: "咽喉像被卡住喘气费力、流口水无法吞咽，或鼻血按压 20 分钟不止", IsRedFlag: true, ServiceHint: "ENT_ASSESSMENT", BoostWeight: 1.0},
						{OptionKey: "SUDDEN_DEAFNESS", Label: "72 小时内单侧或双侧听力突然下降或伴耳闷眩晕", IsRedFlag: true, ServiceHint: "ENT_ASSESSMENT", BoostWeight: 0.95},
						{OptionKey: "COMMON_ENT", Label: "普通咽痛、鼻塞流涕或慢性耳鸣", ServiceHint: "ENT_ASSESSMENT", BoostWeight: 0.8},
					},
				},
			},
		},
		{
			PackCode:       "PACK_OPHTHALMOLOGY_URGENT",
			PackName:       "突发视力骤降、眼剧痛与外伤异物",
			ClinicalDomain: "OPHTHALMOLOGY",
			Description:    "筛查急性闭角型青光眼、视网膜脱离与眼化学/穿通伤",
			VersionRev:     1,
			UpdatedAt:      now,
			QuestionsTemplate: []PlaybookQuestionTemplate{
				{
					SlotKey:           "eye_urgent",
					Title:             "眼部症状是否影响视力或伴有同侧头痛恶心？",
					Category:          "RED_FLAG",
					Type:              "SINGLE_CHOICE",
					EvidenceDimension: "DIM_EYE_URGENT",
					Options: []PlaybookQuestionOption{
						{OptionKey: "VISION_LOSS_OR_TRAUMA", Label: "视力突然大幅下降/黑影遮挡，或眼外伤/酸碱液体溅入", IsRedFlag: true, ServiceHint: "OPHTHALMOLOGY_ASSESSMENT", BoostWeight: 1.0},
						{OptionKey: "ACUTE_GLAUCOMA_RISK", Label: "眼睛胀痛剧烈，伴同侧头痛、恶心呕吐或看灯光有彩虹圈", IsRedFlag: true, ServiceHint: "OPHTHALMOLOGY_ASSESSMENT", BoostWeight: 0.95},
						{OptionKey: "COMMON_EYE", Label: "眼红分泌物多、干涩异物感，视力清晰稳定", ServiceHint: "OPHTHALMOLOGY_ASSESSMENT", BoostWeight: 0.8},
					},
				},
			},
		},
		{
			PackCode:       "PACK_SKIN_ALLERGY_ANAPHYLAXIS",
			PackName:       "全身风团水肿、咽喉发紧与药疹脱皮",
			ClinicalDomain: "DERMATOLOGY",
			Description:    "第一时间拦截过敏性休克/喉头水肿与重症药疹",
			VersionRev:     1,
			UpdatedAt:      now,
			QuestionsTemplate: []PlaybookQuestionTemplate{
				{
					SlotKey:           "allergy_airway",
					Title:             "皮疹或皮肤红肿是否伴有呼吸发紧或大面积水疱脱皮？",
					Category:          "RED_FLAG",
					Type:              "SINGLE_CHOICE",
					EvidenceDimension: "DIM_ALLERGY_ALARM",
					Options: []PlaybookQuestionOption{
						{OptionKey: "ANAPHYLAXIS_SIGNS", Label: "全身大片风团迅速增多，且伴嗓子发紧喘不过气、嘴唇肿或头晕", IsRedFlag: true, ServiceHint: "DERMATOLOGY_ASSESSMENT", BoostWeight: 1.0},
						{OptionKey: "SEVERE_DRUG_ERUPTION", Label: "近期服药后全身起水疱、皮肤大片剥脱或口腔黏膜溃烂", IsRedFlag: true, ServiceHint: "DERMATOLOGY_ASSESSMENT", BoostWeight: 0.95},
						{OptionKey: "LOCAL_DERMATITIS", Label: "局部皮肤发红、瘙痒、丘疹或脱屑，呼吸正常", ServiceHint: "DERMATOLOGY_ASSESSMENT", BoostWeight: 0.85},
					},
				},
			},
		},
		{
			PackCode:       "PACK_ENDOCRINE_METABOLIC",
			PackName:       "三多一少、低血糖冷汗手抖与甲状腺表现",
			ClinicalDomain: "ENDOCRINE",
			Description:    "鉴别酮症酸中毒急症、低血糖发作与慢性内分泌代谢评估",
			VersionRev:     1,
			UpdatedAt:      now,
			QuestionsTemplate: []PlaybookQuestionTemplate{
				{
					SlotKey:           "endo_screen",
					Title:             "是否伴有代谢紊乱相关危象或典型症状？",
					Category:          "DIFFERENTIAL",
					Type:              "SINGLE_CHOICE",
					EvidenceDimension: "DIM_ENDO_SCREEN",
					Options: []PlaybookQuestionOption{
						{OptionKey: "DKA_RISK", Label: "既往糖尿病且呼吸深大喘气、呼气有烂苹果味或意识嗜睡", IsRedFlag: true, ServiceHint: "ENDOCRINE_ASSESSMENT", BoostWeight: 1.0},
						{OptionKey: "HYPO_OR_THYROID", Label: "心慌手抖出冷汗饥饿感，或明显怕热多汗体重快速下降", ServiceHint: "ENDOCRINE_ASSESSMENT", BoostWeight: 0.9},
						{OptionKey: "METABOLIC_CHRONIC", Label: "口渴多饮多尿，或常规体检发现血糖/血脂/尿酸升高", ServiceHint: "ENDOCRINE_ASSESSMENT", BoostWeight: 0.85},
					},
				},
			},
		},
		{
			PackCode:       "PACK_HEMATOLOGY_SYSTEMIC",
			PackName:       "自发瘀斑压不褪色、黏膜渗血与淋巴肿大",
			ClinicalDomain: "HEMATOLOGY",
			Description:    "鉴别系统性出血/紫癜/血液病（防误导至皮肤科）与普通皮肤过敏",
			VersionRev:     1,
			UpdatedAt:      now,
			QuestionsTemplate: []PlaybookQuestionTemplate{
				{
					SlotKey:           "hema_screen",
					Title:             "皮下瘀斑/红点按压是否褪色，是否伴随鼻衄、牙龈出血或乏力？",
					Category:          "DIFFERENTIAL",
					Type:              "SINGLE_CHOICE",
					EvidenceDimension: "DIM_HEMATOLOGY_SCREEN",
					Options: []PlaybookQuestionOption{
						{OptionKey: "SYSTEMIC_HEMA_ALARM", Label: "全身大片自发瘀斑融合，且伴有持续高热、面色苍白乏力或意识变差", IsRedFlag: true, ServiceHint: "HEMATOLOGY_ASSESSMENT", BoostWeight: 1.0},
						{OptionKey: "PURPURA_BLEED", Label: "身上小红点/瘀斑用手按压不褪色，或常有流鼻血、刷牙牙龈渗血、月经过多", ServiceHint: "HEMATOLOGY_ASSESSMENT", BoostWeight: 0.95},
						{OptionKey: "BLANCHING_SKIN", Label: "红斑按压会褪色、明显瘙痒，无黏膜渗血或贫血表现", ServiceHint: "DERMATOLOGY_ASSESSMENT", BoostWeight: 0.8},
					},
				},
			},
		},
		{
			PackCode:       "PACK_ORTHO_SPINE_RED",
			PackName:       "脊柱外伤、肢体肌力下降与马尾二便障碍",
			ClinicalDomain: "ORTHOPEDICS",
			Description:    "鉴别马尾神经综合征急症、根性神经卡压与慢性颈腰康复需求",
			VersionRev:     1,
			UpdatedAt:      now,
			QuestionsTemplate: []PlaybookQuestionTemplate{
				{
					SlotKey:           "spine_red",
					Title:             "颈肩或腰腿不适是否伴有以下神经受压或外伤红旗表现？",
					Category:          "RED_FLAG",
					Type:              "SINGLE_CHOICE",
					EvidenceDimension: "DIM_SPINE_RED_FLAG",
					Options: []PlaybookQuestionOption{
						{OptionKey: "CAUDA_EQUINA", Label: "会阴肛门周围麻木、大小便失禁或排尿困难、双腿踩棉花感迅速加重", IsRedFlag: true, ServiceHint: "ORTHOPEDIC_SPINE_ASSESSMENT", BoostWeight: 1.0},
						{OptionKey: "RADICULOPATHY", Label: "明显像过电一样向单侧手臂/手指或臀部小腿放射麻痛", ServiceHint: "ORTHOPEDIC_SPINE_ASSESSMENT", BoostWeight: 0.9},
						{OptionKey: "CHRONIC_STRAIN", Label: "主要是颈肩腰部僵硬酸胀（遇风寒加重），无肢体麻木无力", ServiceHint: "REHABILITATION_ASSESSMENT", BoostWeight: 0.8, TCMSyndromeHints: []string{"TCM_SYN_WIND_COLD_DAMP_BI"}},
					},
				},
			},
		},
		{
			PackCode:       "PACK_PSYCH_ORGANIC_EXCLUDE",
			PackName:       "惊恐心悸器质性排除与睡眠情绪评估",
			ClinicalDomain: "PSYCHIATRY",
			Description:    "先排筛心源性器质危急症，再导诊精神心理或中医睡眠调理门诊",
			VersionRev:     1,
			UpdatedAt:      now,
			QuestionsTemplate: []PlaybookQuestionTemplate{
				{
					SlotKey:           "psych_organic",
					Title:             "惊慌、心悸或失眠发作的临床背景更接近以下哪一种？",
					Category:          "DIFFERENTIAL",
					Type:              "SINGLE_CHOICE",
					EvidenceDimension: "DIM_PSYCH_ORGANIC",
					Options: []PlaybookQuestionOption{
						{OptionKey: "ORGANIC_URGENT", Label: "首次突发剧烈心慌窒息，伴压榨胸痛、晕厥或冠心病史", IsRedFlag: true, ServiceHint: "CARDIO_ASSESSMENT", BoostWeight: 0.9},
						{OptionKey: "PSYCH_MOOD", Label: "反复情绪低落、焦虑紧张、入睡困难早醒，既往检查心肺未见严重异常", ServiceHint: "PSYCHIATRY_ASSESSMENT", BoostWeight: 0.9, TCMSyndromeHints: []string{"TCM_SYN_HEART_SPLEEN_DEFICIENCY", "TCM_SYN_PHLEGM_HEAT_DISTURB_MIND"}},
					},
				},
			},
		},
		{
			PackCode:       "PACK_TCM_FOUR_DIAGNOSTIC",
			PackName:       "中医四诊补充（寒热口苦、二便与睡眠）",
			ClinicalDomain: "TCM",
			Description:    "采集四诊偏性并直接输出可预约的 6 大贵中医专病与辨证证候",
			VersionRev:     1,
			UpdatedAt:      now,
			QuestionsTemplate: []PlaybookQuestionTemplate{
				{
					SlotKey:           "tcm_signs",
					Title:             "【中医四诊补充】结合您平时的体质表现，是否有以下特征（可协助匹配贵中医特色专病门诊）？",
					Category:          "TCM",
					Type:              "SINGLE_CHOICE",
					EvidenceDimension: "DIM_TCM_FOUR_DIAGNOSTIC",
					Options: []PlaybookQuestionOption{
						{OptionKey: "COLD_STOMACH_LIVER", Label: "胃脘怕冷喜温敷、吃生冷加重，或情绪抑郁伴叹气胁胀", ServiceHint: "GASTRO_INTERNAL_ASSESSMENT", BoostWeight: 0.4, TCMSyndromeHints: []string{"TCM_SYN_SPLEEN_STOMACH_COLD_DEFICIENCY", "TCM_SYN_LIVER_QI_STAGNATION_SPLEEN"}},
						{OptionKey: "DAMP_HEAT_YELLOW", Label: "口苦口臭、烦热口渴、大便黏滞不爽或小便黄赤", ServiceHint: "GASTRO_INTERNAL_ASSESSMENT", BoostWeight: 0.4, TCMSyndromeHints: []string{"TCM_SYN_SPLEEN_STOMACH_DAMP_HEAT", "TCM_SYN_PHLEGM_HEAT_DISTURB_MIND"}},
						{OptionKey: "WIND_PHLEGM_COLD_BI", Label: "颈肩腰腿怕风怕冷遇寒疼重，或突发面部麻木僵硬口角微歪", ServiceHint: "REHABILITATION_ASSESSMENT", BoostWeight: 0.4, TCMSyndromeHints: []string{"TCM_SYN_WIND_COLD_DAMP_BI", "TCM_SYN_WIND_PHLEGM_STRIKE"}},
						{OptionKey: "SLEEP_DEFICIENCY", Label: "入睡困难多梦易醒、心悸健忘、头晕耳鸣或腰膝酸软", ServiceHint: "NEURO_INTERNAL_ASSESSMENT", BoostWeight: 0.4, TCMSyndromeHints: []string{"TCM_SYN_HEART_SPLEEN_DEFICIENCY", "TCM_SYN_LIVER_KIDNEY_YIN_DEFICIENCY"}},
						{OptionKey: "NONE_OR_NORMAL", Label: "无明显上述寒热二便睡眠偏性", BoostWeight: 0.0},
					},
				},
			},
		},
	}
}
