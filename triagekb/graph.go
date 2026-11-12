package triagekb

import (
	"sort"
	"strings"
	"sync"
)

// GraphSnapshot 只读不可变图快照容器 (Immutable Graph Snapshot)
// 支持 atomic.Pointer 托管，所有并发读取绝对零锁，彻底杜绝读写争用。
type GraphSnapshot struct {
	// 实体索引
	Symptoms     map[string]*SymptomNode
	Diseases     map[string]*DiseaseNode
	SafetyRules  map[string]*SafetyRuleNode
	Questions    map[string]*QuestionNode
	Services     map[string]*ServiceNode
	TCMSyndromes map[string]*TCMSyndromeNode
	TCMDiseases  map[string]*TCMDiseaseNode
	TCMClinics   map[string]*TCMClinicNode
	TCMTherapies map[string]*TCMTherapyNode

	// 扩展词典
	HotSymptoms      []HotSymptomItem
	BodyParts        []BodyPartItem
	KeywordSynonyms  map[string][]string
	FallbackKeywords map[string][]string

	// 拓扑出边：SourceCode -> []*GraphEdge
	OutEdges map[string][]*GraphEdge

	// 拓扑入边：TargetCode -> []*GraphEdge
	InEdges map[string][]*GraphEdge

	// 拼音与关键词倒排索引：keyword -> []SymptomCode
	InvertedIndex map[string][]string

	// 教材循证记录索引：SymptomCode -> []*EvidenceRecord
	EvidenceIndex map[string][]*EvidenceRecord

	// 元数据信息
	Version       string
	ActiveVersion string
	DSLPath       string
	ManifestHash  string
	RuntimeConfig map[string]interface{}
	LayerStatuses []LayerStatus
	Layers        []string
	mu            sync.RWMutex // 仅在构建期使用，快照生成后不可变
}

// NewGraphSnapshot 创建新快照容器
func NewGraphSnapshot(version string) *GraphSnapshot {
	return &GraphSnapshot{
		Symptoms:         make(map[string]*SymptomNode),
		Diseases:         make(map[string]*DiseaseNode),
		SafetyRules:      make(map[string]*SafetyRuleNode),
		Questions:        make(map[string]*QuestionNode),
		Services:         make(map[string]*ServiceNode),
		TCMSyndromes:     make(map[string]*TCMSyndromeNode),
		TCMDiseases:      make(map[string]*TCMDiseaseNode),
		TCMClinics:       make(map[string]*TCMClinicNode),
		TCMTherapies:     make(map[string]*TCMTherapyNode),
		HotSymptoms:      make([]HotSymptomItem, 0),
		BodyParts:        make([]BodyPartItem, 0),
		KeywordSynonyms:  make(map[string][]string),
		FallbackKeywords: make(map[string][]string),
		OutEdges:         make(map[string][]*GraphEdge),
		InEdges:          make(map[string][]*GraphEdge),
		InvertedIndex:    make(map[string][]string),
		EvidenceIndex:    make(map[string][]*EvidenceRecord),
		Version:          version,
		RuntimeConfig:    make(map[string]interface{}),
		LayerStatuses:    make([]LayerStatus, 0),
		Layers:           make([]string, 0),
	}
}

// AddEdge 添加拓扑边并建立出边和入边双向索
func (g *GraphSnapshot) AddEdge(edge *GraphEdge) {
	if edge == nil {
		return
	}
	g.OutEdges[edge.SourceCode] = append(g.OutEdges[edge.SourceCode], edge)
	g.InEdges[edge.TargetCode] = append(g.InEdges[edge.TargetCode], edge)
}

// BuildInvertedIndex 构建症状实体拼音与同义词高频倒排索引
func (g *GraphSnapshot) BuildInvertedIndex() {
	index := make(map[string][]string)

	addToken := func(token, code string) {
		token = strings.ToLower(strings.TrimSpace(token))
		if token == "" {
			return
		}
		list := index[token]
		found := false
		for _, existing := range list {
			if existing == code {
				found = true
				break
			}
		}
		if !found {
			index[token] = append(list, code)
		}
	}

	for code, s := range g.Symptoms {
		if s == nil || s.TriageRole == TriageRoleArchived {
			continue
		}
		// 1. 概念编码自身
		addToken(code, code)
		// 2. 概念标准中文名称
		addToken(s.ConceptName, code)
		// 3. 拼音全拼与首字母
		addToken(s.Pinyin, code)
		addToken(s.PinyinInitials, code)
		// 4. 全部同义词与口语别名
		for _, syn := range s.Synonyms {
			addToken(syn, code)
		}
		for _, alias := range s.ColloquialAliases {
			addToken(alias, code)
		}
	}

	g.InvertedIndex = index
}

// SearchSymptoms 通过拼音首字母、全拼或中文关键词进行 O(1) 前缀与倒排检索（严格屏蔽 COMPANION_ONLY 与 ARCHIVED）
func (g *GraphSnapshot) SearchSymptoms(query string, limit int) []*SymptomNode {
	return g.searchSymptomsFiltered(query, limit, false)
}

// SearchAllForNLU 为确定性 NLU 与伴随特征激活保留 COMPANION_ONLY 检索能力（仍屏蔽 ARCHIVED）
func (g *GraphSnapshot) SearchAllForNLU(query string, limit int) []*SymptomNode {
	return g.searchSymptomsFiltered(query, limit, true)
}

func (g *GraphSnapshot) searchSymptomsFiltered(query string, limit int, allowCompanion bool) []*SymptomNode {
	query = strings.ToLower(strings.TrimSpace(query))
	if limit <= 0 {
		limit = 10
	}

	isAllowed := func(sym *SymptomNode) bool {
		if sym == nil || sym.TriageRole == TriageRoleArchived {
			return false
		}
		if !allowCompanion && sym.TriageRole == TriageRoleCompanionOnly {
			return false
		}
		return true
	}

	resultMap := make(map[string]*SymptomNode)
	var results []*SymptomNode

	if query == "" {
		for _, sym := range g.Symptoms {
			if !isAllowed(sym) {
				continue
			}
			results = append(results, sym)
			if len(results) >= limit {
				break
			}
		}
		return results
	}

	// 1. 精确倒排索引命中
	if codes, ok := g.InvertedIndex[query]; ok {
		for _, code := range codes {
			if sym, exists := g.Symptoms[code]; exists && isAllowed(sym) {
				if _, added := resultMap[code]; !added {
					resultMap[code] = sym
					results = append(results, sym)
					if len(results) >= limit {
						return results
					}
				}
			}
		}
	}

	// 2. 模糊包含与拼音前缀匹配
	for _, sym := range g.Symptoms {
		if !isAllowed(sym) {
			continue
		}
		if _, added := resultMap[sym.ConceptCode]; added {
			continue
		}
		match := false
		if strings.Contains(strings.ToLower(sym.ConceptName), query) {
			match = true
		} else if strings.HasPrefix(strings.ToLower(sym.Pinyin), query) {
			match = true
		} else if strings.HasPrefix(strings.ToLower(sym.PinyinInitials), query) {
			match = true
		} else {
			for _, syn := range sym.Synonyms {
				if strings.Contains(strings.ToLower(syn), query) {
					match = true
					break
				}
			}
			if !match {
				for _, alias := range sym.ColloquialAliases {
					if strings.Contains(strings.ToLower(alias), query) {
						match = true
						break
					}
				}
			}
		}

		if match {
			resultMap[sym.ConceptCode] = sym
			results = append(results, sym)
			if len(results) >= limit {
				break
			}
		}
	}

	return results
}

// MetadataPayload 导出确定性排序后的全量只读元数据与扩展词典，供 TS 运行时与管理后台消费
func (g *GraphSnapshot) MetadataPayload() map[string]interface{} {
	services := make([]*ServiceNode, 0, len(g.Services))
	for _, item := range g.Services {
		services = append(services, item)
	}
	sort.Slice(services, func(i, j int) bool {
		return services[i].ServiceCode < services[j].ServiceCode
	})

	clinics := make([]*TCMClinicNode, 0, len(g.TCMClinics))
	for _, item := range g.TCMClinics {
		clinics = append(clinics, item)
	}
	sort.Slice(clinics, func(i, j int) bool {
		return clinics[i].ClinicCode < clinics[j].ClinicCode
	})

	syndromes := make([]*TCMSyndromeNode, 0, len(g.TCMSyndromes))
	for _, item := range g.TCMSyndromes {
		syndromes = append(syndromes, item)
	}
	sort.Slice(syndromes, func(i, j int) bool {
		return syndromes[i].SyndromeCode < syndromes[j].SyndromeCode
	})

	tcmDiseases := make([]*TCMDiseaseNode, 0, len(g.TCMDiseases))
	for _, item := range g.TCMDiseases {
		tcmDiseases = append(tcmDiseases, item)
	}
	sort.Slice(tcmDiseases, func(i, j int) bool {
		return tcmDiseases[i].DiseaseCode < tcmDiseases[j].DiseaseCode
	})

	symptoms := make([]*SymptomNode, 0, len(g.Symptoms))
	for _, item := range g.Symptoms {
		symptoms = append(symptoms, item)
	}
	sort.Slice(symptoms, func(i, j int) bool {
		return symptoms[i].ConceptCode < symptoms[j].ConceptCode
	})

	questions := make([]*QuestionNode, 0, len(g.Questions))
	for _, item := range g.Questions {
		questions = append(questions, item)
	}
	sort.Slice(questions, func(i, j int) bool {
		return questions[i].QuestionID < questions[j].QuestionID
	})

	rules := make([]*SafetyRuleNode, 0, len(g.SafetyRules))
	for _, item := range g.SafetyRules {
		rules = append(rules, item)
	}
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].RuleID < rules[j].RuleID
	})

	return map[string]interface{}{
		"version":           g.Version,
		"active_version":    g.ActiveVersion,
		"manifest_hash":     g.ManifestHash,
		"dsl_path":          g.DSLPath,
		"runtime":           g.RuntimeConfig,
		"layer_statuses":    g.LayerStatuses,
		"layers":            g.Layers,
		"services":          services,
		"tcm_clinics":       clinics,
		"tcm_syndromes":     syndromes,
		"tcm_diseases":      tcmDiseases,
		"symptoms":          symptoms,
		"questions":         questions,
		"safety_rules":      rules,
		"hot_symptoms":      g.HotSymptoms,
		"body_parts":        g.BodyParts,
		"keyword_synonyms":  g.KeywordSynonyms,
		"fallback_keywords": g.FallbackKeywords,
	}
}
