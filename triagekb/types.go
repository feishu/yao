package triagekb

// ClinicalContext 患者临床上下文契约
type ClinicalContext struct {
	Symptoms        []string          `json:"symptoms"`                   // 激活的主诉/症状 concept_code
	Answers         map[string]string `json:"answers,omitempty"`          // 结构化已答追问映射 question_id -> option_value
	KnownSlots      map[string]string `json:"known_slots,omitempty"`      // E1/NLU 提取的高置信已知槽位 (slot_key -> value)
	ConfirmedSigns  []string          `json:"confirmed_signs,omitempty"`  // 确认伴随体征
	NegativeSigns   []string          `json:"negative_signs,omitempty"`   // 排除的体征/阴性特征
	Age             int               `json:"age"`                        // 年龄
	Gender          string            `json:"gender"`                     // MALE / FEMALE / UNKNOWN
	IsPregnant      *bool             `json:"is_pregnant,omitempty"`      // 明确孕产状态
	UncertainStreak int               `json:"uncertain_streak,omitempty"` // 连续不确定答题次数 (SP-2)
	RawUtterance    string            `json:"raw_utterance,omitempty"`    // 原始患者口述
	MedicalHistory  []string          `json:"medical_history,omitempty"`  // 既往史标签
}

// EmergencyAlert 急危重症预警信息
type EmergencyAlert struct {
	RuleID               string `json:"rule_id"`
	RuleName             string `json:"rule_name"`
	Level                string `json:"level"` // CRITICAL / URGENT
	Title                string `json:"title"`
	WarningText          string `json:"warning_text"`
	Call120              bool   `json:"call_120"`
	RecommendedDeptName  string `json:"recommended_dept_name"`
	RecommendedGuideText string `json:"guide_text"`
}

// RankedService 推荐服务与科室条目 (现代医学全科与专病门诊)
type RankedService struct {
	ServiceCode          string   `json:"service_code"`
	ServiceName          string   `json:"service_name"`
	Score                float64  `json:"score"`
	IsPrimary            bool     `json:"is_primary"`
	TrackType            string   `json:"track_type"` // WESTERN_GENERAL / TCM_SPECIALIST
	Description          string   `json:"description"`
	PrimaryDeptKeywords  []string `json:"primary_dept_keywords"`
	FallbackDeptKeywords []string `json:"fallback_dept_keywords"`
	AssociatedClinicCode string   `json:"associated_clinic_code,omitempty"`
	AssociatedClinicName string   `json:"associated_clinic_name,omitempty"`
	BookingURL           string   `json:"booking_url,omitempty"`
}

// ScoredDisease 评分疾病实体
type ScoredDisease struct {
	DiseaseCode string  `json:"disease_code"`
	DiseaseName string  `json:"disease_name"`
	Score       float64 `json:"score"`
	IsCritical  bool    `json:"is_critical"`
	ServiceCode string  `json:"service_code"`
	Description string  `json:"description"`
}

// EvidenceChain 人卫教材逐字原句出处
type EvidenceChain struct {
	SymptomCode   string `json:"symptom_code"`
	SymptomName   string `json:"symptom_name"`
	Textbook      string `json:"textbook"`
	Chapter       string `json:"chapter"`
	Page          string `json:"page,omitempty"`
	Quote         string `json:"quote"`
	MatchedTarget string `json:"matched_target"`
}

// InquiryOption 动态追问选项
type InquiryOption struct {
	Label            string   `json:"label"`
	Value            string   `json:"value"`
	IsRedFlag        bool     `json:"is_red_flag"`
	IsUncertain      bool     `json:"is_uncertain,omitempty"`
	ServiceHint      string   `json:"service_hint,omitempty"`
	BoostWeight      float64  `json:"boost_weight,omitempty"`
	TCMSyndromeHints []string `json:"tcm_syndrome_hints,omitempty"`
}

// InquiryStep 信息熵歧义追问推荐
type InquiryStep struct {
	QuestionID  string          `json:"question_id"`
	SlotKey     string          `json:"slot_key,omitempty"`
	Title       string          `json:"title"`
	Type        string          `json:"type"`     // SINGLE_CHOICE / MULTI_CHOICE
	Category    string          `json:"category"` // DIFFERENTIAL / RED_FLAG
	Options     []InquiryOption `json:"options"`
	Information float64         `json:"information_gain,omitempty"` // 信息增益
}

// TCMSyndromeResult 中医辨证结果
type TCMSyndromeResult struct {
	SyndromeCode  string   `json:"syndrome_code"`
	SyndromeName  string   `json:"syndrome_name"`
	OrganSystem   string   `json:"organ_system"`
	Pathogen      string   `json:"pathogen_nature"`
	Score         float64  `json:"score"`
	TypicalSigns  []string `json:"typical_signs"`
	Description   string   `json:"description"`
	EvidenceQuote string   `json:"evidence_quote"`
}

// TCMClinicResult 贵中医特色专病门诊
type TCMClinicResult struct {
	ClinicCode         string   `json:"clinic_code"`
	ClinicName         string   `json:"clinic_name"`
	DeptName           string   `json:"dept_name"`
	ExpertTitles       []string `json:"expert_titles"`
	SpecialityTags     []string `json:"speciality_tags"`
	TargetDeptKeywords []string `json:"target_dept_keywords,omitempty"`
	BookingURL         string   `json:"booking_url"`
	Score              float64  `json:"score"`
}

// TCMTherapyResult 中医非药物外治疗法推荐
type TCMTherapyResult struct {
	TherapyCode          string   `json:"therapy_code"`
	TherapyName          string   `json:"therapy_name"`
	Category             string   `json:"category"`
	Indications          []string `json:"indications"`
	ClinicRecommendation string   `json:"clinic_recommendation"`
	Description          string   `json:"description"`
}

// TCMRecommendation 完整中医双轨推荐体
type TCMRecommendation struct {
	Syndromes []TCMSyndromeResult `json:"syndromes"`
	Clinics   []TCMClinicResult   `json:"clinics"`
	Therapies []TCMTherapyResult  `json:"therapies"`
}

// TriageDecision 导诊推演决策最终契约
type TriageDecision struct {
	State                  string             `json:"state"`                        // NORMAL / EMERGENCY / CONVERGED
	SafetyState            string             `json:"safety_state,omitempty"`       // ROUTINE_PATHWAY / URGENT_ASSESSMENT / EMERGENCY_NOW
	ConvergenceReason      string             `json:"convergence_reason,omitempty"` // EMERGENCY_SHORT_CIRCUIT / URGENT_RULE_HIT / DIRECT_SINGLE_HOP / UNCERTAIN_STREAK / MAX_STEPS / CONVERGED
	CanEarlyTerminate      bool               `json:"can_early_terminate"`          // 是否允许用户提前查看常规推荐结果（红旗清零后）
	EmergencyAlert         *EmergencyAlert    `json:"emergency_alert,omitempty"`
	AnsweredLabels         map[string]string  `json:"answered_labels,omitempty"` // question_id -> 中文选项标签
	RankedServices         []RankedService    `json:"ranked_services"`
	SuspectedDiseases      []ScoredDisease    `json:"suspected_diseases"`
	EvidenceChains         []EvidenceChain    `json:"evidence_chains"`
	NextInquiry            *InquiryStep       `json:"next_inquiry,omitempty"`
	TCM                    *TCMRecommendation `json:"tcm,omitempty"`
	KBVersion              string             `json:"kb_version,omitempty"`
	KBHash                 string             `json:"kb_hash,omitempty"`
	PreferSpecialistClinic bool               `json:"prefer_specialist_clinic,omitempty"`
	AuditPersistence       bool               `json:"audit_persistence,omitempty"`
}

// --- 图谱内存实体定义 ---

// AssociatedComplicationItem 症状关联并发症条目
type AssociatedComplicationItem struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// SymptomNode 症状/主诉节点
type SymptomNode struct {
	ConceptCode             string                       `json:"concept_code"`
	ConceptName             string                       `json:"concept_name"`
	BodyPart                string                       `json:"body_part"`
	Pinyin                  string                       `json:"pinyin"`
	PinyinInitials          string                       `json:"pinyin_initials"`
	Synonyms                []string                     `json:"synonyms"`
	ColloquialAliases       []string                     `json:"colloquial_aliases,omitempty"`
	TriageRole              string                       `json:"triage_role,omitempty"`
	SkipFollowupMode        string                       `json:"skip_followup_mode,omitempty"`
	DirectTriageNote        string                       `json:"direct_triage_note,omitempty"`
	DefaultServiceCode      string                       `json:"default_service_code"`
	DifferentialServices    []string                     `json:"differential_service_codes"`
	IsRedFlagCandidate      bool                         `json:"is_red_flag_candidate"`
	NeedsReview             bool                         `json:"needs_review"`
	RecommendedAgeGroup     string                       `json:"recommended_age_group"`
	GenderLimit             string                       `json:"gender_limit"`
	EvidenceProtocolID      string                       `json:"evidence_protocol_id"`
	ICD11Code               string                       `json:"icd11_code"`
	ICD11Title              string                       `json:"icd11_title"`
	TextbookSource          string                       `json:"textbook_source"`
	AssociatedComplications []AssociatedComplicationItem `json:"associated_complications,omitempty"`
}

// DiseaseNode 鉴别疾病节点
type DiseaseNode struct {
	DiseaseCode     string `json:"disease_code"`
	DiseaseName     string `json:"disease_name"`
	DeptServiceCode string `json:"dept_service_code"`
	ICD10Code       string `json:"icd10_code"`
	ICD11Code       string `json:"icd11_code"`
	IsCritical      bool   `json:"is_critical"`
	Category        string `json:"category"`
	Description     string `json:"description"`
}

// SafetyRuleNode 安全急救规则节点
type SafetyRuleNode struct {
	RuleID           string                 `json:"rule_id"`
	Name             string                 `json:"name"`
	Level            string                 `json:"level"`
	SafetyState      string                 `json:"safety_state"`
	PrimarySymptoms  []string               `json:"primary_symptoms"`
	TriggerCondition map[string]interface{} `json:"trigger_condition"`
	Title            string                 `json:"title"`
	WarningText      string                 `json:"warning_text"`
	EmergencyAction  map[string]interface{} `json:"emergency_action"`
}

// QuestionNode 追问题目节点
type QuestionNode struct {
	QuestionID         string          `json:"question_id"`
	SlotKey            string          `json:"slot_key,omitempty"`
	Title              string          `json:"title"`
	ApplicableSymptoms []string        `json:"applicable_symptoms"`
	Type               string          `json:"type"`
	Category           string          `json:"category"`
	EvidenceDimension  string          `json:"evidence_dimension"`
	Options            []InquiryOption `json:"options"`
}

// ServiceNode 标准诊疗服务节点
type ServiceNode struct {
	ServiceCode              string   `json:"service_code"`
	ServiceName              string   `json:"service_name"`
	Description              string   `json:"description"`
	PrimaryDeptKeywords      []string `json:"primary_dept_keywords"`
	FallbackDeptKeywords     []string `json:"fallback_dept_keywords"`
	RequiresEmergencySupport bool     `json:"requires_emergency_support"`
}

// TCMSyndromeNode 中医证候节点
type TCMSyndromeNode struct {
	SyndromeCode   string   `json:"syndrome_code"`
	SyndromeName   string   `json:"syndrome_name"`
	OrganSystem    string   `json:"organ_system"`
	PathogenNature string   `json:"pathogen_nature"`
	TypicalSigns   []string `json:"typical_signs"`
	Description    string   `json:"description"`
}

// TCMClinicNode 贵中医专病门诊节点
type TCMClinicNode struct {
	ClinicCode         string   `json:"clinic_code"`
	ClinicName         string   `json:"clinic_name"`
	DeptName           string   `json:"dept_name"`
	ExpertTitles       []string `json:"expert_titles"`
	SpecialityTags     []string `json:"speciality_tags"`
	TargetDeptKeywords []string `json:"target_dept_keywords"`
	BookingURL         string   `json:"booking_url"`
}

// TCMDiseaseNode 中医优势病种节点
type TCMDiseaseNode struct {
	DiseaseCode       string `json:"disease_code"`
	DiseaseName       string `json:"disease_name"`
	ModernDiseaseName string `json:"modern_disease_name"`
	ModernDiseaseCode string `json:"modern_disease_code"`
	PreferredDeptName string `json:"preferred_dept_name"`
	Description       string `json:"description"`
}

// HotSymptomItem 平台热门症状入口词典项
type HotSymptomItem struct {
	SymptomCode string `json:"symptom_code"`
	Label       string `json:"label"`
	Icon        string `json:"icon"`
	BodyPart    string `json:"body_part"`
	SortOrder   int    `json:"sort_order"`
	IsActive    bool   `json:"is_active"`
}

// BodyPartItem 平台解剖部位分组入口词典项
type BodyPartItem struct {
	PartCode     string   `json:"part_code"`
	Label        string   `json:"label"`
	SymptomCodes []string `json:"symptom_codes"`
	SortOrder    int      `json:"sort_order"`
	IsActive     bool     `json:"is_active"`
}

// LayerStatus 联邦图谱加载层状态与指纹
type LayerStatus struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Priority  int    `json:"priority"`
	Enabled   bool   `json:"enabled"`
	Connector string `json:"connector,omitempty"`
	DBFile    string `json:"db_file"`
	SHA256    string `json:"sha256,omitempty"`
	Loaded    bool   `json:"loaded"`
}

// TCMTherapyNode 中医非药物疗法节点
type TCMTherapyNode struct {
	TherapyCode          string   `json:"therapy_code"`
	TherapyName          string   `json:"therapy_name"`
	Category             string   `json:"category"`
	Indications          []string `json:"indications"`
	ClinicRecommendation string   `json:"clinic_recommendation"`
	Description          string   `json:"description"`
}

// GraphEdge 内存拓扑边
type GraphEdge struct {
	SourceType      string  `json:"source_type"`
	SourceCode      string  `json:"source_code"`
	TargetType      string  `json:"target_type"`
	TargetCode      string  `json:"target_code"`
	RelationType    string  `json:"relation_type"`
	Weight          float64 `json:"weight"`
	EvidenceQuote   string  `json:"evidence_quote"`
	EvidenceBook    string  `json:"evidence_book"`
	EvidenceChapter string  `json:"evidence_chapter"`
}

// EvidenceRecord 证据记录
type EvidenceRecord struct {
	ChunkID         string                   `json:"chunk_id"`
	SymptomCode     string                   `json:"symptom_code"`
	ConceptName     string                   `json:"concept_name"`
	TextbookSource  string                   `json:"textbook_source"`
	Chapter         string                   `json:"chapter"`
	Page            string                   `json:"page"`
	EvidenceSummary string                   `json:"evidence_summary,omitempty"`
	TypicalSigns    []string                 `json:"typical_signs"`
	TypicalSymptoms []string                 `json:"typical_symptoms"`
	DiffDiseases    []map[string]interface{} `json:"differential_diseases"`
}
