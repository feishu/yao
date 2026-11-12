package triagekb

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// NLUPatientContext 患者侧基础已知上下文（未填写性别默认为 UNKNOWN，严禁伪造 MALE）
type NLUPatientContext struct {
	Age        int    `json:"age,omitempty"`
	Gender     string `json:"gender,omitempty"`
	IsPregnant *bool  `json:"is_pregnant,omitempty"`
}

// NLUMatchedConcept NLU 命中概念明细
type NLUMatchedConcept struct {
	ConceptCode string  `json:"concept_code"`
	ConceptName string  `json:"concept_name"`
	TriageRole  string  `json:"triage_role"`
	MatchedText string  `json:"matched_text"`
	MatchScore  float64 `json:"match_score"`
	IsCompanion bool    `json:"is_companion,omitempty"`
}

// NLUParseResult 6 级 Zero-LLM 确定性自然语言解析输出
type NLUParseResult struct {
	OriginalText        string              `json:"original_text"`
	NormalizedText      string              `json:"normalized_text"`
	ExtractedAge        *int                `json:"extracted_age,omitempty"`
	InferredAgeGroup    string              `json:"inferred_age_group,omitempty"`
	ExtractedGender     string              `json:"extracted_gender"`
	ExtractedPregnant   *bool               `json:"extracted_pregnant,omitempty"`
	Onset               string              `json:"onset,omitempty"`
	SeedConcepts        []string            `json:"seed_concepts"`
	CompanionConcepts   []string            `json:"companion_concepts"`
	MatchedDetails      []NLUMatchedConcept `json:"matched_details"`
	NegatedSlots        []string            `json:"negated_slots"`
	NegatedConcepts     []string            `json:"negated_concepts"`
	KnownSlots          map[string]string   `json:"known_slots"`
	PreConfirmedAnswers map[string]string   `json:"pre_confirmed_answers"`
	ConfirmedSigns      []string            `json:"confirmed_signs"`
	SummaryFacts        []string            `json:"summary_facts"`
	ElapsedMs           float64             `json:"elapsed_ms"`
}

type nluPhraseEntry struct {
	phrase      string
	runeLen     int
	conceptCode string
	baseWeight  float64
}

type nluMatcherIndex struct {
	manifestKey string
	phrases     []nluPhraseEntry
}

var (
	nluIndexMu    sync.RWMutex
	nluCachedIdx  *nluMatcherIndex
	ageYearRegex  = regexp.MustCompile(`(\d{1,3})\s*(?:周岁|岁)`)
	ageMonthRegex = regexp.MustCompile(`(\d{1,2})\s*(?:个月|月龄)`)
	tempHighRegex = regexp.MustCompile(`(38\.[5-9]|39\.\d?|4[0-2](?:\.\d)?)\s*[度℃°]?`)
	tempNumRegex  = regexp.MustCompile(`体温\s*(3[7-9](?:\.\d)?|4[0-2](?:\.\d)?)`)
	clauseSplitRe = regexp.MustCompile(`[，,。；;！!？?\n]+|但是|不过|可是|然而|却|而且`)
)

// curatedColloquialAnchors 常见患者口语表达到核心概念编码的确定性兜底映射（按存在性动态校验）
var curatedColloquialAnchors = []struct {
	phrase string
	codes  []string
	weight float64
}{
	{"胸口闷痛", []string{"SYM_CHEST_PAIN", "SYM_CHEST_TIGHTNESS"}, 1.0},
	{"胸闷痛", []string{"SYM_CHEST_PAIN", "SYM_CHEST_TIGHTNESS"}, 0.98},
	{"心口窝疼", []string{"SYM_CHEST_PAIN"}, 0.98},
	{"心口窝痛", []string{"SYM_CHEST_PAIN"}, 0.98},
	{"心口疼", []string{"SYM_CHEST_PAIN"}, 0.95},
	{"心口痛", []string{"SYM_CHEST_PAIN"}, 0.95},
	{"胸口痛", []string{"SYM_CHEST_PAIN"}, 0.95},
	{"胸口疼", []string{"SYM_CHEST_PAIN"}, 0.95},
	{"胸痛", []string{"SYM_CHEST_PAIN"}, 0.95},
	{"胸口闷", []string{"SYM_CHEST_TIGHTNESS"}, 0.92},
	{"胸闷", []string{"SYM_CHEST_TIGHTNESS"}, 0.9},
	{"喘不上气", []string{"SYM_DYSPNEA", "SYM_CHEST_TIGHTNESS"}, 0.98},
	{"喘不过气", []string{"SYM_DYSPNEA", "SYM_CHEST_TIGHTNESS"}, 0.98},
	{"气喘", []string{"SYM_DYSPNEA"}, 0.9},
	{"呼吸困难", []string{"SYM_DYSPNEA"}, 0.98},
	{"憋气", []string{"SYM_CHEST_TIGHTNESS"}, 0.9},
	{"突发下腹剧痛", []string{"SYM_LOWER_ABDOMEN_SUDDEN_TEARING"}, 0.96},
	{"下腹剧痛", []string{"SYM_LOWER_ABDOMEN_SUDDEN_TEARING"}, 0.95},
	{"小腹剧痛", []string{"SYM_LOWER_ABDOMEN_SUDDEN_TEARING"}, 0.95},
	{"下腹痛", []string{"SYM_CHRONIC_LOWER_ABDOMINAL_PAIN"}, 0.9},
	{"小腹痛", []string{"SYM_CHRONIC_LOWER_ABDOMINAL_PAIN"}, 0.9},
	{"肚子疼", []string{"SYM_EPIGASTRIC_PAIN"}, 0.85},
	{"肚子痛", []string{"SYM_EPIGASTRIC_PAIN"}, 0.85},
	{"胃疼", []string{"SYM_EPIGASTRIC_PAIN"}, 0.92},
	{"胃痛", []string{"SYM_EPIGASTRIC_PAIN"}, 0.92},
	{"右下腹痛", []string{"SYM_RLQ_MIGRATORY_PAIN"}, 0.95},
	{"右上腹痛", []string{"SYM_RUQ_BILIARY_COLIC"}, 0.95},
	{"反酸烧心", []string{"SYM_HEARTBURN_REFLUX"}, 0.96},
	{"烧心反酸", []string{"SYM_HEARTBURN_REFLUX"}, 0.96},
	{"天旋地转", []string{"SYM_DIZZINESS_VERTIGO"}, 0.96},
	{"头晕", []string{"SYM_DIZZINESS_VERTIGO"}, 0.85},
	{"剧烈头痛", []string{"SYM_HEADACHE_THUNDERCLAP"}, 0.95},
	{"炸裂样头痛", []string{"SYM_HEADACHE_THUNDERCLAP"}, 0.99},
	{"偏头痛", []string{"SYM_HEADACHE_THROBBING"}, 0.94},
	{"拉肚子", []string{"SYM_DIARRHEA_WATERY"}, 0.9},
	{"水样腹泻", []string{"SYM_DIARRHEA_WATERY"}, 0.95},
	{"心慌", []string{"SYM_PALPITATION"}, 0.92},
	{"心跳快", []string{"SYM_PALPITATION"}, 0.9},
	{"青一块紫一块", []string{"SYM_TB_SKIN_MUCOUS_MEMBRANE_BLEEDING"}, 0.95},
	{"瘀斑", []string{"SYM_TB_SKIN_MUCOUS_MEMBRANE_BLEEDING"}, 0.94},
	{"紫癜", []string{"SYM_TB_SKIN_MUCOUS_MEMBRANE_BLEEDING"}, 0.94},
	{"牙龈出血", []string{"SYM_TB_SKIN_MUCOUS_MEMBRANE_BLEEDING"}, 0.88},
	{"咳嗽", []string{"SYM_COUGH"}, 0.88},
}

func getOrBuildNLUMatcherIndex(snap *GraphSnapshot) *nluMatcherIndex {
	if snap == nil {
		return &nluMatcherIndex{}
	}
	key := fmt.Sprintf("%p|%s|%d|%s", snap, snap.ManifestHash, len(snap.Symptoms), snap.ActiveVersion)
	nluIndexMu.RLock()
	if nluCachedIdx != nil && nluCachedIdx.manifestKey == key {
		idx := nluCachedIdx
		nluIndexMu.RUnlock()
		return idx
	}
	nluIndexMu.RUnlock()

	idx := buildNLUMatcherIndex(snap, key)
	nluIndexMu.Lock()
	nluCachedIdx = idx
	nluIndexMu.Unlock()
	return idx
}

func buildNLUMatcherIndex(snap *GraphSnapshot, key string) *nluMatcherIndex {
	seen := make(map[string]bool)
	var entries []nluPhraseEntry

	addPhrase := func(phrase, code string, weight float64) {
		clean := strings.TrimSpace(strings.ToLower(phrase))
		runeLen := utf8.RuneCountInString(clean)
		if clean == "" || runeLen < 2 {
			return
		}
		if _, ok := snap.Symptoms[code]; !ok {
			return
		}
		k := clean + "->" + code
		if seen[k] {
			return
		}
		seen[k] = true
		entries = append(entries, nluPhraseEntry{
			phrase:      clean,
			runeLen:     runeLen,
			conceptCode: code,
			baseWeight:  weight,
		})
	}

	// 1. 从全图谱症状库加载规范名称拆词 + synonyms + colloquial_aliases
	var codes []string
	for code := range snap.Symptoms {
		codes = append(codes, code)
	}
	sort.Strings(codes)

	for _, code := range codes {
		sym := snap.Symptoms[code]
		if sym == nil || sym.TriageRole == TriageRoleArchived {
			continue
		}
		addPhrase(sym.ConceptName, code, 1.0)
		// 拆分 "/"、"（"、"、" 获取直观短词，如 "胸痛/心前区压榨性疼痛" -> "胸痛"
		for _, part := range splitConceptParts(sym.ConceptName) {
			if utf8.RuneCountInString(part) >= 2 && utf8.RuneCountInString(part) <= 12 {
				addPhrase(part, code, 0.92)
			}
		}
		for _, syn := range sym.Synonyms {
			addPhrase(syn, code, 0.95)
			for _, sub := range splitConceptParts(syn) {
				if utf8.RuneCountInString(sub) >= 2 && utf8.RuneCountInString(sub) <= 10 {
					addPhrase(sub, code, 0.88)
				}
			}
		}
		for _, alias := range sym.ColloquialAliases {
			addPhrase(alias, code, 0.98)
		}
	}

	// 2. 注入经过存在性验证的高频口语锚点
	for _, anc := range curatedColloquialAnchors {
		for _, code := range anc.codes {
			addPhrase(anc.phrase, code, anc.weight)
		}
	}

	// 最长优先排列，确定性排序避免非稳定匹配
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].runeLen != entries[j].runeLen {
			return entries[i].runeLen > entries[j].runeLen
		}
		if entries[i].baseWeight != entries[j].baseWeight {
			return entries[i].baseWeight > entries[j].baseWeight
		}
		return entries[i].conceptCode < entries[j].conceptCode
	})

	return &nluMatcherIndex{
		manifestKey: key,
		phrases:     entries,
	}
}

func splitConceptParts(raw string) []string {
	replacer := strings.NewReplacer("/", "|", "、", "|", "（", "|", "）", "|", "(", "|", ")", "|", "，", "|", "：", "|")
	norm := replacer.Replace(raw)
	rawParts := strings.Split(norm, "|")
	var out []string
	for _, p := range rawParts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func normalizeUtteranceText(text string) string {
	replacer := strings.NewReplacer(
		"發燒", "发烧",
		"發熱", "发热",
		"噁心", "恶心",
		"疼痛", "痛",
		" ", "",
		"\t", "",
	)
	_ = replacer
	return strings.TrimSpace(text)
}

func normalizeGenderInput(g string) string {
	switch strings.ToUpper(strings.TrimSpace(g)) {
	case "MALE", "M", "男":
		return "MALE"
	case "FEMALE", "F", "女":
		return "FEMALE"
	default:
		return "UNKNOWN"
	}
}

// AnalyzeUtterance 执行 Zero-LLM 6 级确定性患者口语多槽位解析（总耗时 < 15ms）
func AnalyzeUtterance(snap *GraphSnapshot, text string, patientCtx NLUPatientContext) NLUParseResult {
	startTime := time.Now()

	normalized := strings.TrimSpace(text)
	result := NLUParseResult{
		OriginalText:        text,
		NormalizedText:      normalized,
		ExtractedGender:     normalizeGenderInput(patientCtx.Gender),
		ExtractedPregnant:   patientCtx.IsPregnant,
		SeedConcepts:        []string{},
		CompanionConcepts:   []string{},
		MatchedDetails:      []NLUMatchedConcept{},
		NegatedSlots:        []string{},
		NegatedConcepts:     []string{},
		KnownSlots:          make(map[string]string),
		PreConfirmedAnswers: make(map[string]string),
		ConfirmedSigns:      []string{},
		SummaryFacts:        []string{},
	}

	if patientCtx.Age > 0 {
		ageCopy := patientCtx.Age
		result.ExtractedAge = &ageCopy
		result.InferredAgeGroup = classifyAgeGroupByYears(ageCopy)
	}
	if normalized == "" {
		result.ElapsedMs = float64(time.Since(startTime).Microseconds()) / 1000.0
		return result
	}

	// Stage 2: 人口学（年龄/亲属/显式性别/孕周）与急慢性时长正则抽取
	extractDemographicsAndOnset(normalized, &result)

	// Stage 3 & Stage 4: 按标点与转折词切分子句，运行 0~6 字局部否定窗口 + 最长优先症状匹配
	clauses := clauseSplitRe.Split(normalized, -1)
	matcher := getOrBuildNLUMatcherIndex(snap)

	seedSeen := make(map[string]bool)
	companionSeen := make(map[string]bool)
	negSlotSeen := make(map[string]bool)
	negConceptSeen := make(map[string]bool)

	addNegSlot := func(slotKey, slotVal string) {
		result.KnownSlots[slotKey] = slotVal
		if !negSlotSeen[slotKey] {
			negSlotSeen[slotKey] = true
			result.NegatedSlots = append(result.NegatedSlots, slotKey)
		}
	}

	// 若文本已提取急慢性，写入已知槽位供跳过常规询问
	if result.Onset == "ACUTE" {
		result.KnownSlots["onset_acute"] = "ONSET_ACUTE"
	} else if result.Onset == "SUBACUTE" {
		result.KnownSlots["onset_acute"] = "DAYS_TO_WEEKS"
	} else if result.Onset == "CHRONIC" {
		result.KnownSlots["onset_acute"] = "ONSET_CHRONIC"
	}
	if result.ExtractedPregnant != nil && *result.ExtractedPregnant {
		result.KnownSlots["is_pregnant"] = "PREGNANT_YES"
	}

	var positiveClauses []string
	for _, rawClause := range clauses {
		clause := strings.TrimSpace(rawClause)
		if clause == "" {
			continue
		}
		posClause, clauseNegConcepts := processClauseNegations(clause, matcher, &result, addNegSlot)
		for _, nc := range clauseNegConcepts {
			if !negConceptSeen[nc] {
				negConceptSeen[nc] = true
				result.NegatedConcepts = append(result.NegatedConcepts, nc)
			}
		}

		// 检查正向发热（仅限非否定子句里正面提起）
		checkPositiveFeverInClause(posClause, &result)

		if posClause != "" {
			positiveClauses = append(positiveClauses, posClause)
			// 在正向子句片段内执行最长优先概念匹配（支持同子句命中多个症状，如“胸口闷痛喘不上气”）
			matchConceptsInPositiveClause(snap, posClause, matcher, negConceptSeen, seedSeen, companionSeen, &result)
		}
	}

	// Stage 4.5: 若仅匹配出通用伴随词/没有命中独立症状且未覆盖全文部位，保留干净排序
	if len(result.SeedConcepts) > 0 && result.Onset == "ACUTE" {
		// 对同时存在的相似急重症状做自然部位权重排序，不丢弃任何独立主诉
		sortMatchedSeedPriority(snap, positiveClauses, &result)
	}

	// Stage 5: 首句高确定性红旗字面精确命中（严谨保守：模糊词绝不预填跳过）
	matchLiteralRedFlags(snap, positiveClauses, &result)

	// 组装可阅读语义提取摘要，供 E3 沙盒及 H5 首句回显胶囊展示
	buildNLUSummaryFacts(&result)

	result.ElapsedMs = float64(time.Since(startTime).Microseconds()) / 1000.0
	return result
}

func classifyAgeGroupByYears(age int) string {
	switch {
	case age <= 14:
		return "PEDIATRIC"
	case age >= 65:
		return "ELDERLY"
	default:
		return "ADULT"
	}
}

func extractDemographicsAndOnset(text string, res *NLUParseResult) {
	// 1. 文本显式年龄数字优先抽取（覆盖外部兜底默认年龄如 30 岁，适配代挂号场景如“爷爷72岁”）
	if m := ageYearRegex.FindStringSubmatch(text); len(m) >= 2 {
		if ageVal, err := strconv.Atoi(m[1]); err == nil && ageVal >= 0 && ageVal <= 120 {
			res.ExtractedAge = &ageVal
			res.InferredAgeGroup = classifyAgeGroupByYears(ageVal)
		}
	} else if m := ageMonthRegex.FindStringSubmatch(text); len(m) >= 2 {
		if months, err := strconv.Atoi(m[1]); err == nil && months >= 0 && months <= 48 {
			ageVal := 1
			if months < 12 {
				ageVal = 0
			}
			res.ExtractedAge = &ageVal
			res.InferredAgeGroup = "PEDIATRIC"
		}
	}

	// 2. 亲属/称谓先验（仅对主语年龄段及明显称谓生效，无线索则绝不猜性别）
	pediatricTerms := []string{"宝宝", "婴儿", "患儿", "新生儿", "幼童", "小孩", "小朋友", "小儿", "男宝", "女宝", "男孩", "女孩", "男童", "女童"}
	for _, kw := range pediatricTerms {
		if strings.Contains(text, kw) {
			if res.InferredAgeGroup == "" {
				res.InferredAgeGroup = "PEDIATRIC"
			}
			break
		}
	}
	maleFamilyTerms := []string{"爷爷", "姥爷", "外公", "男宝", "男性", "男生", "男孩", "男童", "小伙", "丈夫", "老公", "父亲", "爸爸", "儿子"}
	femaleFamilyTerms := []string{"奶奶", "姥姥", "外婆", "女宝", "女性", "女生", "女孩", "女童", "姑娘", "妻子", "老婆", "母亲", "妈妈", "女儿", "孕妇", "怀孕", "孕期", "产妇"}

	if res.ExtractedGender == "UNKNOWN" {
		for _, kw := range femaleFamilyTerms {
			if strings.Contains(text, kw) {
				res.ExtractedGender = "FEMALE"
				break
			}
		}
	}
	if res.ExtractedGender == "UNKNOWN" {
		for _, kw := range maleFamilyTerms {
			if strings.Contains(text, kw) {
				res.ExtractedGender = "MALE"
				break
			}
		}
	}
	if res.InferredAgeGroup == "" {
		for _, kw := range []string{"爷爷", "奶奶", "姥爷", "姥姥", "外公", "外婆", "老人", "高龄"} {
			if strings.Contains(text, kw) {
				res.InferredAgeGroup = "ELDERLY"
				break
			}
		}
	}

	// 3. 显式孕产线索
	pregTerms := []string{"孕妇", "怀孕", "孕周", "孕期", "妊娠", "胎动", "预产期"}
	for _, kw := range pregTerms {
		if strings.Contains(text, kw) {
			yes := true
			res.ExtractedPregnant = &yes
			res.ExtractedGender = "FEMALE"
			break
		}
	}

	// 4. 急慢性起病时间正则
	acutePatterns := []string{"刚才", "刚刚", "突然", "突发", "忽然", "昨晚", "昨天", "昨夜", "今天", "今早", "今晨", "半天", "几小时", "几个小时", "几分钟"}
	subacutePatterns := []string{"两天", "2天", "三天", "3天", "几天", "一周", "1周", "两周", "2周", "数天"}
	chronicPatterns := []string{"半年", "好几年", "几年", "几个月", "数月", "长期", "一直反复", "好几个月", "慢性"}
	for _, kw := range acutePatterns {
		if strings.Contains(text, kw) {
			res.Onset = "ACUTE"
			break
		}
	}
	if res.Onset == "" {
		for _, kw := range chronicPatterns {
			if strings.Contains(text, kw) {
				res.Onset = "CHRONIC"
				break
			}
		}
	}
	if res.Onset == "" {
		for _, kw := range subacutePatterns {
			if strings.Contains(text, kw) {
				res.Onset = "SUBACUTE"
				break
			}
		}
	}
}

var negationMarkers = []string{"没有", "未见", "并未", "否认", "不是", "无明显", "也不", "没", "不", "无"}

type negationRule struct {
	keywords []string
	slotKey  string
	slotVal  string
	concepts []string
}

var slotNegationRules = []negationRule{
	{keywords: []string{"发烧", "发热", "高热", "低热", "低烧", "高烧", "寒战"}, slotKey: "has_fever", slotVal: "NO_FEVER", concepts: []string{"SYM_TB_INFECTIOUS_FEVER", "SYM_FUO_FEVER_OF_UNKNOWN_ORIGIN", "SYM_UNEXPLAINED_CONTINUOUS_FEVER"}},
	{keywords: []string{"胸痛", "胸口痛", "胸闷痛"}, slotKey: "chest_pain", slotVal: "NO", concepts: []string{"SYM_CHEST_PAIN"}},
	{keywords: []string{"咳嗽", "干咳", "咳痰"}, slotKey: "cough_present", slotVal: "NO", concepts: []string{"SYM_COUGH"}},
	{keywords: []string{"外伤", "摔伤", "撞伤", "磕碰", "跌倒"}, slotKey: "trauma_mech", slotVal: "NO_TRAUMA"},
	{keywords: []string{"出血", "流血", "呕血", "咯血", "黑便", "便血"}, slotKey: "bleeding_alarm", slotVal: "NO_BLEEDING"},
	{keywords: []string{"头痛", "头疼"}, slotKey: "headache_present", slotVal: "NO", concepts: []string{"SYM_HEADACHE_THROBBING", "SYM_HEADACHE_THUNDERCLAP", "SYM_HEADACHE_TENSION"}},
	{keywords: []string{"恶心", "呕吐"}, slotKey: "nausea_present", slotVal: "NO", concepts: []string{"SYM_NAUSEA_VOMITING"}},
}

// processClauseNegations 按 0~6 字安全否定窗口剔除否定片段并返回子句内保留的正向文本 + 阴性概念
func processClauseNegations(clause string, matcher *nluMatcherIndex, res *NLUParseResult, addNegSlot func(string, string)) (string, []string) {
	lower := strings.ToLower(clause)
	runes := []rune(lower)
	masked := make([]bool, len(runes))
	var negConcepts []string

	i := 0
	for i < len(runes) {
		matchedMarkerLen := 0
		for _, mk := range negationMarkers {
			mkRunes := []rune(mk)
			if i+len(mkRunes) <= len(runes) && string(runes[i:i+len(mkRunes)]) == mk {
				matchedMarkerLen = len(mkRunes)
				break
			}
		}
		if matchedMarkerLen == 0 {
			i++
			continue
		}

		// 安全窗口仅限当前子句内否定词后的 0~6 个字，绝不跨句误伤前序主诉
		winStart := i + matchedMarkerLen
		winEnd := winStart + 6
		if winEnd > len(runes) {
			winEnd = len(runes)
		}
		windowText := string(runes[winStart:winEnd])

		// 遮蔽否定词本身及窗口内的否定描述
		for k := i; k < winEnd; k++ {
			masked[k] = true
		}

		for _, rule := range slotNegationRules {
			for _, kw := range rule.keywords {
				if strings.Contains(windowText, kw) {
					addNegSlot(rule.slotKey, rule.slotVal)
					negConcepts = append(negConcepts, rule.concepts...)
					break
				}
			}
		}
		// 检查窗口内所有命中短语对应的症状概念，禁止激活为阳性
		for _, entry := range matcher.phrases {
			if entry.runeLen > 6 {
				continue
			}
			if strings.Contains(windowText, entry.phrase) {
				negConcepts = append(negConcepts, entry.conceptCode)
			}
		}
		i = winEnd
	}

	var posBuilder strings.Builder
	for idx, r := range runes {
		if !masked[idx] {
			posBuilder.WriteRune(r)
		}
	}
	return strings.TrimSpace(posBuilder.String()), negConcepts
}

func checkPositiveFeverInClause(posClause string, res *NLUParseResult) {
	if posClause == "" {
		return
	}
	if _, alreadySet := res.KnownSlots["has_fever"]; alreadySet {
		return
	}
	if tempHighRegex.MatchString(posClause) || strings.Contains(posClause, "高烧") || strings.Contains(posClause, "高热") || strings.Contains(posClause, "寒战") {
		res.KnownSlots["has_fever"] = "HIGH_FEVER"
		return
	}
	if tempNumRegex.MatchString(posClause) || strings.Contains(posClause, "发烧") || strings.Contains(posClause, "发热") || strings.Contains(posClause, "低热") || strings.Contains(posClause, "低烧") {
		res.KnownSlots["has_fever"] = "LOW_FEVER"
	}
}

func matchConceptsInPositiveClause(
	snap *GraphSnapshot,
	posClause string,
	matcher *nluMatcherIndex,
	negConceptSeen map[string]bool,
	seedSeen map[string]bool,
	companionSeen map[string]bool,
	res *NLUParseResult,
) {
	remaining := posClause
	for len(matcher.phrases) > 0 && remaining != "" {
		bestIdx := -1
		bestPos := -1
		for idx, entry := range matcher.phrases {
			pos := strings.Index(remaining, entry.phrase)
			if pos >= 0 {
				bestIdx = idx
				bestPos = pos
				break // 已按字长降序排列，首个命中的就是当前最精准/最长匹配
			}
		}
		if bestIdx < 0 {
			break
		}

		matchedPhrase := matcher.phrases[bestIdx].phrase
		matchedCode := matcher.phrases[bestIdx].conceptCode
		matchedWeight := matcher.phrases[bestIdx].baseWeight

		// 同步采集在同一位置精确等长匹配该短语的所有锚点（如“胸口闷痛”同时激活胸痛与胸闷）
		for _, entry := range matcher.phrases {
			if entry.phrase != matchedPhrase {
				continue
			}
			code := entry.conceptCode
			if negConceptSeen[code] {
				continue
			}
			sym := snap.Symptoms[code]
			if sym == nil || sym.TriageRole == TriageRoleArchived {
				continue
			}
			if sym.TriageRole == TriageRoleCompanionOnly {
				if !companionSeen[code] {
					companionSeen[code] = true
					res.CompanionConcepts = append(res.CompanionConcepts, code)
					res.ConfirmedSigns = append(res.ConfirmedSigns, code)
					res.MatchedDetails = append(res.MatchedDetails, NLUMatchedConcept{
						ConceptCode: code,
						ConceptName: sym.ConceptName,
						TriageRole:  sym.TriageRole,
						MatchedText: matchedPhrase,
						MatchScore:  entry.baseWeight,
						IsCompanion: true,
					})
				}
				continue
			}

			// TB_DEFERRED 在尚未转正前若非显式别名命中，优先 CORE_COMPLAINT；若无其它核心主诉也允许匹配但标注角色
			if !seedSeen[code] {
				seedSeen[code] = true
				res.SeedConcepts = append(res.SeedConcepts, code)
				res.MatchedDetails = append(res.MatchedDetails, NLUMatchedConcept{
					ConceptCode: code,
					ConceptName: sym.ConceptName,
					TriageRole:  sym.TriageRole,
					MatchedText: matchedPhrase,
					MatchScore:  entry.baseWeight,
				})
			}
		}
		_ = matchedCode
		_ = matchedWeight

		// 切掉已消耗的片段并保留左右未消费文本，防重叠重复消费同时允许相邻短语识别（“胸口闷痛” + “喘不上气”）
		left := strings.TrimSpace(remaining[:bestPos])
		right := strings.TrimSpace(remaining[bestPos+len(matchedPhrase):])
		if left != "" && right != "" {
			remaining = left + " " + right
		} else if left != "" {
			remaining = left
		} else {
			remaining = right
		}
	}
}

func sortMatchedSeedPriority(snap *GraphSnapshot, positiveClauses []string, res *NLUParseResult) {
	if len(res.SeedConcepts) <= 1 || snap == nil {
		return
	}
	joined := strings.Join(positiveClauses, " ")
	scoreMap := make(map[string]float64, len(res.SeedConcepts))
	for _, detail := range res.MatchedDetails {
		if detail.IsCompanion {
			continue
		}
		bonus := detail.MatchScore
		sym := snap.Symptoms[detail.ConceptCode]
		if sym != nil {
			if sym.TriageRole == TriageRoleCoreComplaint {
				bonus += 0.2
			}
			// 部位上下文消歧：如句子含“心口/胸口”+“胃/肚子”区分权重
			if sym.BodyPart == "CHEST" && (strings.Contains(joined, "胸") || strings.Contains(joined, "心口") || strings.Contains(joined, "喘")) {
				bonus += 0.15
			}
			if sym.BodyPart == "EPIGASTRIC" && (strings.Contains(joined, "胃") || strings.Contains(joined, "肚子") || strings.Contains(joined, "反酸")) {
				bonus += 0.15
			}
		}
		if bonus > scoreMap[detail.ConceptCode] {
			scoreMap[detail.ConceptCode] = bonus
		}
	}
	sort.SliceStable(res.SeedConcepts, func(i, j int) bool {
		return scoreMap[res.SeedConcepts[i]] > scoreMap[res.SeedConcepts[j]]
	})
}

func matchLiteralRedFlags(snap *GraphSnapshot, positiveClauses []string, res *NLUParseResult) {
	if snap == nil || len(res.SeedConcepts) == 0 || len(positiveClauses) == 0 {
		return
	}
	seedSet := make(map[string]bool, len(res.SeedConcepts))
	for _, c := range res.SeedConcepts {
		seedSet[c] = true
	}

	// 字面高确定性红旗短语字典（仅当同一个未否定正向子句字面精确出现，严禁对模糊表述做预设）
	literalMarkers := []struct {
		slotKey  string
		phrases  []string
		optMatch []string // option_value substring or exact value
	}{
		{slotKey: "chest_cold_sweat", phrases: []string{"大汗淋漓", "浑身出冷汗", "满身冷汗", "冷汗湿透", "濒死感"}, optMatch: []string{"YES", "CRUSHING_SWEAT"}},
		{slotKey: "chest_radiation", phrases: []string{"向左肩放射", "向左臂放射", "放散到左肩", "压榨样痛", "大石头压着"}, optMatch: []string{"LEFT_ARM_BACK", "CRUSHING_SWEAT"}},
		{slotKey: "dyspnea_sev", phrases: []string{"说不出完整的话", "无法说完整一句话", "嘴唇发紫", "口唇发绀", "不能平卧坐着喘"}, optMatch: []string{"CANNOT_SPEAK", "AIRWAY_DISTRESS"}},
		{slotKey: "abd_alarm", phrases: []string{"肚子硬得像木板", "板状腹", "柏油样黑便", "大口吐血", "呕血"}, optMatch: []string{"RIGID_ABDOMEN", "GI_BLEED"}},
		{slotKey: "neuro_fast", phrases: []string{"口角歪斜", "嘴巴歪了", "一侧手脚麻木", "偏瘫", "突然说不出话", "吐字不清", "霹雳样剧痛"}, optMatch: []string{"FAST_POSITIVE"}},
	}

	for _, marker := range literalMarkers {
		matchedLiteral := ""
		for _, clause := range positiveClauses {
			for _, ph := range marker.phrases {
				if strings.Contains(clause, ph) {
					matchedLiteral = ph
					break
				}
			}
			if matchedLiteral != "" {
				break
			}
		}
		if matchedLiteral == "" {
			continue
		}

		// 查找当前命中种子主诉对应的实际问题与选项值
		for _, q := range snap.Questions {
			if q == nil {
				continue
			}
			qSlot := q.SlotKey
			if qSlot == "" {
				qSlot = inferLegacySlotKey(q.QuestionID)
			}
			if qSlot != marker.slotKey {
				continue
			}
			applicable := false
			for _, sym := range q.ApplicableSymptoms {
				if seedSet[sym] {
					applicable = true
					break
				}
			}
			if !applicable {
				continue
			}
			for _, opt := range q.Options {
				if !opt.IsRedFlag {
					continue
				}
				for _, targetHint := range marker.optMatch {
					if opt.Value == targetHint || strings.Contains(opt.Value, targetHint) {
						res.PreConfirmedAnswers[q.QuestionID] = opt.Value
						res.KnownSlots[marker.slotKey] = opt.Value
						break
					}
				}
			}
		}
	}
}

func buildNLUSummaryFacts(res *NLUParseResult) {
	var facts []string
	if res.ExtractedAge != nil {
		facts = append(facts, fmt.Sprintf("年龄:%d岁(%s)", *res.ExtractedAge, res.InferredAgeGroup))
	} else if res.InferredAgeGroup != "" {
		facts = append(facts, fmt.Sprintf("人群:%s", res.InferredAgeGroup))
	}
	if res.ExtractedGender != "" && res.ExtractedGender != "UNKNOWN" {
		facts = append(facts, fmt.Sprintf("性别:%s", res.ExtractedGender))
	} else {
		facts = append(facts, "性别:未指定(UNKNOWN)")
	}
	if res.Onset == "ACUTE" {
		facts = append(facts, "起病:急性")
	} else if res.Onset == "CHRONIC" {
		facts = append(facts, "起病:慢性反复")
	}
	if val, ok := res.KnownSlots["has_fever"]; ok {
		if val == "NO_FEVER" {
			facts = append(facts, "已确认:无发热(跳过发热筛查)")
		} else {
			facts = append(facts, fmt.Sprintf("发热:%s", val))
		}
	}
	if len(res.SeedConcepts) > 0 {
		facts = append(facts, fmt.Sprintf("识别主诉:%d项", len(res.SeedConcepts)))
	}
	if len(res.PreConfirmedAnswers) > 0 {
		facts = append(facts, fmt.Sprintf("字面红旗直击:%d项", len(res.PreConfirmedAnswers)))
	}
	res.SummaryFacts = facts
}
