package triagekb

// ClinicalContext 患者临床上下文契约
type ClinicalContext struct {
	Symptoms       []string `json:"symptoms"`                  // 激活的主诉/症状 concept_code
	ConfirmedSigns []string `json:"confirmed_signs,omitempty"` // 确认伴随体征
	NegativeSigns  []string `json:"negative_signs,omitempty"`  // 排除的体征/阴性特征
	Age            int      `json:"age"`                       // 年龄
	Gender         string   `json:"gender"`                    // MALE / FEMALE
	MedicalHistory []string `json:"medical_history,omitempty"` // 既往史标签
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
	Label       string `json:"label"`
	Value       string `json:"value"`
	IsRedFlag   bool   `json:"is_red_flag"`
	ServiceHint string `json:"service_hint,omitempty"`
}

// InquiryStep 信息熵歧义追问推荐
type InquiryStep struct {
	QuestionID  string          `json:"question_id"`
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
	ClinicCode     string   `json:"clinic_code"`
	ClinicName     string   `json:"clinic_name"`
	DeptName       string   `json:"dept_name"`
	ExpertTitles   []string `json:"expert_titles"`
	SpecialityTags []string `json:"speciality_tags"`
	BookingURL     string   `json:"booking_url"`
	Score          float64  `json:"score"`
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
	State             string             `json:"state"` // NORMAL / EMERGENCY / CONVERGED
	EmergencyAlert    *EmergencyAlert    `json:"emergency_alert,omitempty"`
	RankedServices    []RankedService    `json:"ranked_services"`
	SuspectedDiseases []ScoredDisease    `json:"suspected_diseases"`
	EvidenceChains    []EvidenceChain    `json:"evidence_chains"`
	NextInquiry       *InquiryStep       `json:"next_inquiry,omitempty"`
	TCM               *TCMRecommendation `json:"tcm,omitempty"`
}

// --- 图谱内存实体定义 ---

// SymptomNode 症状/主诉节点
type SymptomNode struct {
	ConceptCode          string   `json:"concept_code"`
	ConceptName          string   `json:"concept_name"`
	BodyPart             string   `json:"body_part"`
	Pinyin               string   `json:"pinyin"`
	PinyinInitials       string   `json:"pinyin_initials"`
	Synonyms             []string `json:"synonyms"`
	DefaultServiceCode   string   `json:"default_service_code"`
	DifferentialServices []string `json:"differential_service_codes"`
	IsRedFlagCandidate   bool     `json:"is_red_flag_candidate"`
	RecommendedAgeGroup  string   `json:"recommended_age_group"`
	GenderLimit          string   `json:"gender_limit"`
	EvidenceProtocolID   string   `json:"evidence_protocol_id"`
	ICD11Code            string   `json:"icd11_code"`
	ICD11Title           string   `json:"icd11_title"`
	TextbookSource       string   `json:"textbook_source"`
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
	ClinicCode     string   `json:"clinic_code"`
	ClinicName     string   `json:"clinic_name"`
	DeptName       string   `json:"dept_name"`
	ExpertTitles   []string `json:"expert_titles"`
	SpecialityTags []string `json:"speciality_tags"`
	BookingURL     string   `json:"booking_url"`
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
	TypicalSigns    []string                 `json:"typical_signs"`
	TypicalSymptoms []string                 `json:"typical_symptoms"`
	DiffDiseases    []map[string]interface{} `json:"differential_diseases"`
}
