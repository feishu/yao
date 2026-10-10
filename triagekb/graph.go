package triagekb

import (
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
	TCMClinics   map[string]*TCMClinicNode
	TCMTherapies map[string]*TCMTherapyNode

	// 拓扑出边：SourceCode -> []*GraphEdge
	OutEdges map[string][]*GraphEdge

	// 拓扑入边：TargetCode -> []*GraphEdge
	InEdges map[string][]*GraphEdge

	// 拼音与关键词倒排索引：keyword -> []SymptomCode
	InvertedIndex map[string][]string

	// 教材循证记录索引：SymptomCode -> []*EvidenceRecord
	EvidenceIndex map[string][]*EvidenceRecord

	// 元数据信息
	Version string
	Layers  []string
	mu      sync.RWMutex // 仅在构建期使用，快照生成后不可变
}

// NewGraphSnapshot 创建新快照容器
func NewGraphSnapshot(version string) *GraphSnapshot {
	return &GraphSnapshot{
		Symptoms:      make(map[string]*SymptomNode),
		Diseases:      make(map[string]*DiseaseNode),
		SafetyRules:   make(map[string]*SafetyRuleNode),
		Questions:     make(map[string]*QuestionNode),
		Services:      make(map[string]*ServiceNode),
		TCMSyndromes:  make(map[string]*TCMSyndromeNode),
		TCMClinics:    make(map[string]*TCMClinicNode),
		TCMTherapies:  make(map[string]*TCMTherapyNode),
		OutEdges:      make(map[string][]*GraphEdge),
		InEdges:       make(map[string][]*GraphEdge),
		InvertedIndex: make(map[string][]string),
		EvidenceIndex: make(map[string][]*EvidenceRecord),
		Version:       version,
		Layers:        make([]string, 0),
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
		// 1. 概念编码自身
		addToken(code, code)
		// 2. 概念标准中文名称
		addToken(s.ConceptName, code)
		// 3. 拼音全拼与首字母
		addToken(s.Pinyin, code)
		addToken(s.PinyinInitials, code)
		// 4. 全部同义词与近义词
		for _, syn := range s.Synonyms {
			addToken(syn, code)
		}
	}

	g.InvertedIndex = index
}

// SearchSymptoms 通过拼音首字母、全拼或中文关键词进行 O(1) 前缀与倒排检索
func (g *GraphSnapshot) SearchSymptoms(query string, limit int) []*SymptomNode {
	query = strings.ToLower(strings.TrimSpace(query))
	if limit <= 0 {
		limit = 10
	}

	resultMap := make(map[string]*SymptomNode)
	var results []*SymptomNode

	if query == "" {
		for _, sym := range g.Symptoms {
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
			if sym, exists := g.Symptoms[code]; exists {
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
