package triagekb

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// LayerConfig 知识图谱单层配置
type LayerConfig struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Connector string `json:"connector"`
	DBFile    string `json:"db_file"`
	Type      string `json:"type"` // BASE / OVERLAY
	Priority  int    `json:"priority"`
	Enabled   *bool  `json:"enabled,omitempty"`
}

// IsEnabled 返回该图谱层是否启用（缺省 nil 默认为 true）
func (l LayerConfig) IsEnabled() bool {
	return l.Enabled == nil || *l.Enabled
}

// FederatedGraphDSL 联邦知识库声明 DSL
type FederatedGraphDSL struct {
	Name          string                 `json:"name"`
	Label         string                 `json:"label"`
	Version       string                 `json:"version"`
	ActiveVersion string                 `json:"active_version"`
	Storage       string                 `json:"storage"`
	Layers        []LayerConfig          `json:"layers"`
	Runtime       map[string]interface{} `json:"runtime"`
}

// FederatedLoader 多源联邦加载器
type FederatedLoader struct {
	dsl FederatedGraphDSL
}

// NewFederatedLoader 创建加载器
func NewFederatedLoader(dsl FederatedGraphDSL) *FederatedLoader {
	return &FederatedLoader{dsl: dsl}
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func sqliteHasTable(db *sql.DB, tableName string) bool {
	var count int
	err := db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?`, tableName).Scan(&count)
	return err == nil && count > 0
}

func sqliteTableColumns(db *sql.DB, tableName string) map[string]bool {
	cols := make(map[string]bool)
	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info("%s")`, strings.ReplaceAll(tableName, `"`, `""`)))
	if err != nil {
		return cols
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err == nil {
			cols[name] = true
		}
	}
	return cols
}

func resolveConnectorFile(connectorName, appRoot, dslDir string) string {
	cleaned := strings.TrimPrefix(strings.TrimSpace(connectorName), "connectors.")
	if cleaned == "" {
		return ""
	}
	rel := filepath.Join("connectors", strings.ReplaceAll(cleaned, ".", string(filepath.Separator))+".conn.yao")
	candidates := make([]string, 0, 3)
	if appRoot != "" {
		candidates = append(candidates, filepath.Join(appRoot, rel))
	}
	if dslDir != "" {
		candidates = append(candidates, filepath.Join(filepath.Dir(dslDir), rel))
	}
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		candidates = append(candidates, filepath.Join(cwd, rel))
	}
	for _, cand := range candidates {
		raw, err := os.ReadFile(cand)
		if err != nil {
			continue
		}
		var parsed struct {
			Options struct {
				File string `json:"file"`
			} `json:"options"`
		}
		if json.Unmarshal(raw, &parsed) == nil && parsed.Options.File != "" {
			return parsed.Options.File
		}
	}
	return ""
}

func resolveLayerDBPath(layer LayerConfig, appRoot, dslDir string) string {
	rawPath := strings.TrimSpace(layer.DBFile)
	if rawPath == "" && layer.Connector != "" {
		rawPath = resolveConnectorFile(layer.Connector, appRoot, dslDir)
	}
	if rawPath == "" {
		return ""
	}
	if filepath.IsAbs(rawPath) {
		return rawPath
	}

	candidates := make([]string, 0, 4)
	if appRoot != "" {
		candidates = append(candidates, filepath.Join(appRoot, rawPath))
	}
	if dslDir != "" {
		candidates = append(candidates, filepath.Join(dslDir, rawPath))
		candidates = append(candidates, filepath.Join(filepath.Dir(dslDir), rawPath))
	}
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		candidates = append(candidates, filepath.Join(cwd, rawPath))
	}
	for _, cand := range candidates {
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
	}
	if len(candidates) > 0 {
		return candidates[0]
	}
	return rawPath
}

// LoadFromDSL 按层 priority 与类型合并加载所有启用层（支持多 OVERLAY 扩展与 SHA-256 指纹追踪）
func (fl *FederatedLoader) LoadFromDSL(appRoot, dslDir string) (*GraphSnapshot, error) {
	activeVer := strings.TrimSpace(fl.dsl.ActiveVersion)
	if activeVer == "" {
		activeVer = fl.dsl.Version
	}
	snapshot := NewGraphSnapshot(fl.dsl.Version)
	snapshot.ActiveVersion = activeVer
	snapshot.RuntimeConfig = fl.dsl.Runtime

	// 排序启用的层：BASE 优先于 OVERLAY，同类别按 Priority 升序，ID 作为确定性平手决胜
	sortedLayers := make([]LayerConfig, len(fl.dsl.Layers))
	copy(sortedLayers, fl.dsl.Layers)
	sort.Slice(sortedLayers, func(i, j int) bool {
		ti := strings.ToUpper(sortedLayers[i].Type)
		tj := strings.ToUpper(sortedLayers[j].Type)
		if ti != tj {
			if ti == "BASE" {
				return true
			}
			if tj == "BASE" {
				return false
			}
		}
		if sortedLayers[i].Priority != sortedLayers[j].Priority {
			return sortedLayers[i].Priority < sortedLayers[j].Priority
		}
		return sortedLayers[i].ID < sortedLayers[j].ID
	})

	if len(sortedLayers) == 0 {
		sortedLayers = []LayerConfig{
			{ID: "base_general", Name: "全科现代医学临床基座库", Type: "BASE", Priority: 100, DBFile: "data/triage/kb/kb_triage_general.db"},
			{ID: "overlay_tcm", Name: "数智贵中医专病与辨证扩展库", Type: "OVERLAY", Priority: 200, DBFile: "data/triage/kb/kb_triage_tcm.db"},
		}
	}

	hashInput := sha256.New()
	_, _ = hashInput.Write([]byte(snapshot.Version + "|" + snapshot.ActiveVersion + "|"))

	for _, layer := range sortedLayers {
		resolvedPath := resolveLayerDBPath(layer, appRoot, dslDir)
		status := LayerStatus{
			ID:        layer.ID,
			Name:      layer.Name,
			Type:      strings.ToUpper(layer.Type),
			Priority:  layer.Priority,
			Enabled:   layer.IsEnabled(),
			Connector: layer.Connector,
			DBFile:    layer.DBFile,
			Loaded:    false,
		}
		if !layer.IsEnabled() || resolvedPath == "" {
			snapshot.LayerStatuses = append(snapshot.LayerStatuses, status)
			continue
		}

		layerType := strings.ToUpper(layer.Type)
		if layerType == "BASE" {
			if err := fl.loadGeneralDB(snapshot, resolvedPath); err != nil {
				return nil, fmt.Errorf("加载全科基座层(%s)失败: %w", layer.ID, err)
			}
		} else {
			if err := fl.loadTCMDB(snapshot, resolvedPath); err != nil {
				return nil, fmt.Errorf("加载扩展叠加层(%s)失败: %w", layer.ID, err)
			}
		}

		if sum, err := fileSHA256(resolvedPath); err == nil {
			status.SHA256 = sum
			_, _ = hashInput.Write([]byte(layer.ID + ":" + sum + "|"))
		}
		status.DBFile = resolvedPath
		status.Loaded = true
		snapshot.LayerStatuses = append(snapshot.LayerStatuses, status)
		snapshot.Layers = append(snapshot.Layers, layer.ID)
	}

	if rtBytes, err := json.Marshal(fl.dsl.Runtime); err == nil {
		_, _ = hashInput.Write(rtBytes)
	}
	snapshot.ManifestHash = hex.EncodeToString(hashInput.Sum(nil))
	snapshot.BuildInvertedIndex()
	return snapshot, nil
}

// LoadFromFiles 从基座库和扩展库文件构建快照 (兼容旧调用签名)
func (fl *FederatedLoader) LoadFromFiles(generalDBPath, tcmDBPath string) (*GraphSnapshot, error) {
	dslCopy := fl.dsl
	dslCopy.Layers = make([]LayerConfig, 0, 2)
	if generalDBPath != "" {
		dslCopy.Layers = append(dslCopy.Layers, LayerConfig{
			ID:       "base_general",
			Name:     "全科现代医学临床基座库",
			Type:     "BASE",
			Priority: 100,
			DBFile:   generalDBPath,
		})
	}
	if tcmDBPath != "" {
		dslCopy.Layers = append(dslCopy.Layers, LayerConfig{
			ID:       "overlay_tcm",
			Name:     "数智贵中医专病与辨证扩展库",
			Type:     "OVERLAY",
			Priority: 200,
			DBFile:   tcmDBPath,
		})
	}
	return NewFederatedLoader(dslCopy).LoadFromDSL("", "")
}

// loadGeneralDB 加载全科基座库
func (fl *FederatedLoader) loadGeneralDB(g *GraphSnapshot, dbPath string) error {
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("数据库文件不存在: %s", absPath)
	}

	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=ro&cache=shared", absPath))
	if err != nil {
		return err
	}
	defer db.Close()

	// 1. 读取 symptoms（兼容扩展列 needs_review、associated_complications_json 与表里分离扩列）
	symCols := sqliteTableColumns(db, "symptoms")
	hasNeedsReview := symCols["needs_review"]
	hasAssocComp := symCols["associated_complications_json"]
	selectCols := `concept_code, concept_name, body_part, pinyin, pinyin_initials,
		synonyms_json, default_service_code, differential_service_codes_json,
		is_red_flag_candidate, recommended_age_group, gender_limit,
		evidence_protocol_id, icd11_code, icd11_title, textbook_source`
	if hasNeedsReview {
		selectCols += `, needs_review`
	} else {
		selectCols += `, 0 AS needs_review`
	}
	if hasAssocComp {
		selectCols += `, associated_complications_json`
	} else {
		selectCols += `, '[]' AS associated_complications_json`
	}
	if symCols["triage_role"] {
		selectCols += `, COALESCE(triage_role, 'CORE_COMPLAINT')`
	} else {
		selectCols += `, 'CORE_COMPLAINT' AS triage_role`
	}
	if symCols["skip_followup_mode"] {
		selectCols += `, COALESCE(skip_followup_mode, 'REQUIRED')`
	} else {
		selectCols += `, 'REQUIRED' AS skip_followup_mode`
	}
	if symCols["direct_triage_note"] {
		selectCols += `, COALESCE(direct_triage_note, '')`
	} else {
		selectCols += `, '' AS direct_triage_note`
	}
	if symCols["colloquial_aliases_json"] {
		selectCols += `, COALESCE(colloquial_aliases_json, '[]')`
	} else {
		selectCols += `, '[]' AS colloquial_aliases_json`
	}

	symRows, err := db.Query("SELECT " + selectCols + " FROM symptoms")
	if err != nil {
		return err
	}
	defer symRows.Close()

	for symRows.Next() {
		var s SymptomNode
		var synJSON, diffJSON, compJSON, aliasJSON string
		var isRedFlag, needsReview int
		if err := symRows.Scan(
			&s.ConceptCode, &s.ConceptName, &s.BodyPart, &s.Pinyin, &s.PinyinInitials,
			&synJSON, &s.DefaultServiceCode, &diffJSON, &isRedFlag,
			&s.RecommendedAgeGroup, &s.GenderLimit, &s.EvidenceProtocolID,
			&s.ICD11Code, &s.ICD11Title, &s.TextbookSource,
			&needsReview, &compJSON,
			&s.TriageRole, &s.SkipFollowupMode, &s.DirectTriageNote, &aliasJSON,
		); err != nil {
			return err
		}
		s.IsRedFlagCandidate = isRedFlag == 1
		s.NeedsReview = needsReview == 1
		if s.TriageRole == "" {
			s.TriageRole = TriageRoleCoreComplaint
		}
		if s.SkipFollowupMode == "" {
			s.SkipFollowupMode = SkipFollowupRequired
		}
		_ = json.Unmarshal([]byte(synJSON), &s.Synonyms)
		_ = json.Unmarshal([]byte(diffJSON), &s.DifferentialServices)
		if strings.TrimSpace(compJSON) != "" {
			_ = json.Unmarshal([]byte(compJSON), &s.AssociatedComplications)
		}
		if strings.TrimSpace(aliasJSON) != "" {
			_ = json.Unmarshal([]byte(aliasJSON), &s.ColloquialAliases)
		}
		g.Symptoms[s.ConceptCode] = &s
	}

	// 2. 读取 diseases
	disRows, err := db.Query(`
		SELECT disease_code, disease_name, dept_service_code, icd10_code, icd11_code,
		       is_critical, category, description
		FROM diseases
	`)
	if err != nil {
		return err
	}
	defer disRows.Close()

	for disRows.Next() {
		var d DiseaseNode
		var isCrit int
		if err := disRows.Scan(
			&d.DiseaseCode, &d.DiseaseName, &d.DeptServiceCode,
			&d.ICD10Code, &d.ICD11Code, &isCrit, &d.Category, &d.Description,
		); err != nil {
			return err
		}
		d.IsCritical = isCrit == 1
		g.Diseases[d.DiseaseCode] = &d
	}

	// 3. 读取 safety_rules
	ruleRows, err := db.Query(`
		SELECT rule_id, name, level, safety_state, primary_symptoms_json,
		       trigger_condition_json, title, warning_text, emergency_action_json
		FROM safety_rules
	`)
	if err != nil {
		return err
	}
	defer ruleRows.Close()

	for ruleRows.Next() {
		var r SafetyRuleNode
		var symsJSON, trigJSON, actionJSON string
		if err := ruleRows.Scan(
			&r.RuleID, &r.Name, &r.Level, &r.SafetyState,
			&symsJSON, &trigJSON, &r.Title, &r.WarningText, &actionJSON,
		); err != nil {
			return err
		}
		_ = json.Unmarshal([]byte(symsJSON), &r.PrimarySymptoms)
		_ = json.Unmarshal([]byte(trigJSON), &r.TriggerCondition)
		_ = json.Unmarshal([]byte(actionJSON), &r.EmergencyAction)
		g.SafetyRules[r.RuleID] = &r
	}

	// 4. 读取 questions（向后兼容 slot_key 扩列）
	qCols := sqliteTableColumns(db, "questions")
	slotCol := `'' AS slot_key`
	if qCols["slot_key"] {
		slotCol = `COALESCE(slot_key, '')`
	}
	qRows, err := db.Query(fmt.Sprintf(`
		SELECT question_id, title, applicable_symptoms_json, type,
		       category, evidence_dimension, options_json, %s
		FROM questions
	`, slotCol))
	if err != nil {
		return err
	}
	defer qRows.Close()

	for qRows.Next() {
		var q QuestionNode
		var symsJSON, optJSON string
		if err := qRows.Scan(
			&q.QuestionID, &q.Title, &symsJSON, &q.Type,
			&q.Category, &q.EvidenceDimension, &optJSON, &q.SlotKey,
		); err != nil {
			return err
		}
		if q.SlotKey == "" {
			q.SlotKey = inferLegacySlotKey(q.QuestionID)
		}
		_ = json.Unmarshal([]byte(symsJSON), &q.ApplicableSymptoms)
		_ = json.Unmarshal([]byte(optJSON), &q.Options)
		g.Questions[q.QuestionID] = &q
	}

	// 5. 读取 standard_services
	svcRows, err := db.Query(`
		SELECT service_code, service_name, description,
		       primary_dept_keywords_json, fallback_dept_keywords_json, requires_emergency_support
		FROM standard_services
	`)
	if err != nil {
		return err
	}
	defer svcRows.Close()

	for svcRows.Next() {
		var s ServiceNode
		var priJSON, fallJSON string
		var reqEmerg int
		if err := svcRows.Scan(
			&s.ServiceCode, &s.ServiceName, &s.Description,
			&priJSON, &fallJSON, &reqEmerg,
		); err != nil {
			return err
		}
		s.RequiresEmergencySupport = reqEmerg == 1
		_ = json.Unmarshal([]byte(priJSON), &s.PrimaryDeptKeywords)
		_ = json.Unmarshal([]byte(fallJSON), &s.FallbackDeptKeywords)
		g.Services[s.ServiceCode] = &s
	}

	// 6. 读取 edges
	edgeRows, err := db.Query(`
		SELECT source_type, source_code, target_type, target_code,
		       relation_type, COALESCE(weight, 0), COALESCE(evidence_quote, ''), COALESCE(evidence_book, ''), COALESCE(evidence_chapter, '')
		FROM edges
	`)
	if err != nil {
		return err
	}
	defer edgeRows.Close()

	for edgeRows.Next() {
		var e GraphEdge
		if err := edgeRows.Scan(
			&e.SourceType, &e.SourceCode, &e.TargetType, &e.TargetCode,
			&e.RelationType, &e.Weight, &e.EvidenceQuote, &e.EvidenceBook, &e.EvidenceChapter,
		); err != nil {
			return err
		}
		g.AddEdge(&e)
	}

	// 7. 读取 textbook_evidence（兼容 evidence_summary 扩列）
	evCols := sqliteTableColumns(db, "textbook_evidence")
	evSumCol := `'' AS evidence_summary`
	if evCols["evidence_summary"] {
		evSumCol = `COALESCE(evidence_summary, '')`
	}
	evidRows, err := db.Query(fmt.Sprintf(`
		SELECT chunk_id, symptom_code, COALESCE(concept_name, ''), COALESCE(textbook_source, ''),
		       COALESCE(chapter, ''), COALESCE(page, 0), COALESCE(typical_signs_json, '[]'), COALESCE(typical_symptoms_json, '[]'), COALESCE(differential_diseases_json, '[]'), %s
		FROM textbook_evidence
	`, evSumCol))
	if err != nil {
		return err
	}
	defer evidRows.Close()

	for evidRows.Next() {
		var rec EvidenceRecord
		var signsJSON, symsJSON, diffsJSON string
		if err := evidRows.Scan(
			&rec.ChunkID, &rec.SymptomCode, &rec.ConceptName, &rec.TextbookSource,
			&rec.Chapter, &rec.Page, &signsJSON, &symsJSON, &diffsJSON, &rec.EvidenceSummary,
		); err != nil {
			return err
		}
		_ = json.Unmarshal([]byte(signsJSON), &rec.TypicalSigns)
		_ = json.Unmarshal([]byte(symsJSON), &rec.TypicalSymptoms)
		_ = json.Unmarshal([]byte(diffsJSON), &rec.DiffDiseases)
		g.EvidenceIndex[rec.SymptomCode] = append(g.EvidenceIndex[rec.SymptomCode], &rec)
	}

	// 8. 读取扩展平台词典表（向后兼容旧版无词典表 .db）
	if sqliteHasTable(db, "kb_hot_symptoms") {
		hotRows, hErr := db.Query(`
			SELECT symptom_code, label, icon, body_part, sort_order, is_active
			FROM kb_hot_symptoms ORDER BY sort_order ASC, symptom_code ASC
		`)
		if hErr == nil {
			defer hotRows.Close()
			for hotRows.Next() {
				var item HotSymptomItem
				var active int
				if err := hotRows.Scan(&item.SymptomCode, &item.Label, &item.Icon, &item.BodyPart, &item.SortOrder, &active); err == nil {
					item.IsActive = active == 1
					g.HotSymptoms = append(g.HotSymptoms, item)
				}
			}
		}
	}

	if sqliteHasTable(db, "kb_body_parts") {
		bpRows, bErr := db.Query(`
			SELECT part_code, label, symptom_codes_json, sort_order, is_active
			FROM kb_body_parts ORDER BY sort_order ASC, part_code ASC
		`)
		if bErr == nil {
			defer bpRows.Close()
			for bpRows.Next() {
				var item BodyPartItem
				var codesJSON string
				var active int
				if err := bpRows.Scan(&item.PartCode, &item.Label, &codesJSON, &item.SortOrder, &active); err == nil {
					item.IsActive = active == 1
					_ = json.Unmarshal([]byte(codesJSON), &item.SymptomCodes)
					g.BodyParts = append(g.BodyParts, item)
				}
			}
		}
	}

	if sqliteHasTable(db, "kb_keyword_synonyms") {
		synRows, sErr := db.Query(`SELECT term, synonyms_json FROM kb_keyword_synonyms`)
		if sErr == nil {
			defer synRows.Close()
			for synRows.Next() {
				var term, listJSON string
				if err := synRows.Scan(&term, &listJSON); err == nil {
					var list []string
					_ = json.Unmarshal([]byte(listJSON), &list)
					g.KeywordSynonyms[term] = list
				}
			}
		}
	}

	if sqliteHasTable(db, "kb_fallback_keywords") {
		fbRows, fErr := db.Query(`SELECT service_code, keywords_json FROM kb_fallback_keywords`)
		if fErr == nil {
			defer fbRows.Close()
			for fbRows.Next() {
				var svcCode, kwJSON string
				if err := fbRows.Scan(&svcCode, &kwJSON); err == nil {
					var list []string
					_ = json.Unmarshal([]byte(kwJSON), &list)
					g.FallbackKeywords[svcCode] = list
				}
			}
		}
	}

	return nil
}

// loadTCMDB 加载数智贵中医专病与辨证扩展库 (Graph Union)
func (fl *FederatedLoader) loadTCMDB(g *GraphSnapshot, dbPath string) error {
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("数据库文件不存在: %s", absPath)
	}

	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=ro&cache=shared", absPath))
	if err != nil {
		return err
	}
	defer db.Close()

	// 1. 中医证候
	synRows, err := db.Query(`
		SELECT syndrome_code, syndrome_name, organ_system, pathogen_nature,
		       typical_signs_json, description
		FROM tcm_syndromes
	`)
	if err == nil {
		defer synRows.Close()
		for synRows.Next() {
			var syn TCMSyndromeNode
			var signsJSON string
			if err := synRows.Scan(
				&syn.SyndromeCode, &syn.SyndromeName, &syn.OrganSystem,
				&syn.PathogenNature, &signsJSON, &syn.Description,
			); err == nil {
				_ = json.Unmarshal([]byte(signsJSON), &syn.TypicalSigns)
				g.TCMSyndromes[syn.SyndromeCode] = &syn
			}
		}
	}

	// 1.5 中医优势病种
	if sqliteHasTable(db, "tcm_diseases") {
		tdRows, tdErr := db.Query(`
			SELECT disease_code, disease_name, COALESCE(modern_disease_name, ''),
			       COALESCE(modern_disease_code, ''), preferred_dept_name, COALESCE(description, '')
			FROM tcm_diseases
		`)
		if tdErr == nil {
			defer tdRows.Close()
			for tdRows.Next() {
				var td TCMDiseaseNode
				if err := tdRows.Scan(
					&td.DiseaseCode, &td.DiseaseName, &td.ModernDiseaseName,
					&td.ModernDiseaseCode, &td.PreferredDeptName, &td.Description,
				); err == nil {
					g.TCMDiseases[td.DiseaseCode] = &td
				}
			}
		}
	}

	// 2. 贵中医专病门诊（兼容 target_dept_keywords_json 扩展列）
	clinicCols := sqliteTableColumns(db, "tcm_specialist_clinics")
	clinicQuery := `SELECT clinic_code, clinic_name, dept_name, expert_titles_json, speciality_tags_json, COALESCE(booking_url, '')`
	if clinicCols["target_dept_keywords_json"] {
		clinicQuery += `, COALESCE(target_dept_keywords_json, '[]')`
	} else {
		clinicQuery += `, '[]'`
	}
	clinicQuery += ` FROM tcm_specialist_clinics`

	clinicRows, err := db.Query(clinicQuery)
	if err == nil {
		defer clinicRows.Close()
		for clinicRows.Next() {
			var c TCMClinicNode
			var expJSON, tagsJSON, kwJSON string
			if err := clinicRows.Scan(
				&c.ClinicCode, &c.ClinicName, &c.DeptName,
				&expJSON, &tagsJSON, &c.BookingURL, &kwJSON,
			); err == nil {
				_ = json.Unmarshal([]byte(expJSON), &c.ExpertTitles)
				_ = json.Unmarshal([]byte(tagsJSON), &c.SpecialityTags)
				_ = json.Unmarshal([]byte(kwJSON), &c.TargetDeptKeywords)
				if len(c.TargetDeptKeywords) == 0 && c.DeptName != "" {
					c.TargetDeptKeywords = []string{c.DeptName}
				}
				g.TCMClinics[c.ClinicCode] = &c
			}
		}
	}

	// 3. 中医外治疗法
	thRows, err := db.Query(`
		SELECT therapy_code, therapy_name, category, indications_json, clinic_recommendation, description
		FROM tcm_therapies
	`)
	if err == nil {
		defer thRows.Close()
		for thRows.Next() {
			var th TCMTherapyNode
			var indJSON string
			if err := thRows.Scan(
				&th.TherapyCode, &th.TherapyName, &th.Category,
				&indJSON, &th.ClinicRecommendation, &th.Description,
			); err == nil {
				_ = json.Unmarshal([]byte(indJSON), &th.Indications)
				g.TCMTherapies[th.TherapyCode] = &th
			}
		}
	}

	// 4. 中医拓扑边 (按编码前缀与边类型区分 SYMPTOM vs SYNDROME 源类型，打通 SYNDROME_TO_CLINIC 两跳推理)
	edgeRows, err := db.Query(`
		SELECT source_code, target_code, relation_type, weight, evidence_quote, source_ref
		FROM tcm_edges
	`)
	if err == nil {
		defer edgeRows.Close()
		for edgeRows.Next() {
			var e GraphEdge
			if err := edgeRows.Scan(
				&e.SourceCode, &e.TargetCode, &e.RelationType,
				&e.Weight, &e.EvidenceQuote, &e.EvidenceBook,
			); err == nil {
				if strings.HasPrefix(e.SourceCode, "TCM_SYN_") || e.RelationType == "SYNDROME_TO_CLINIC" {
					e.SourceType = "SYNDROME"
				} else {
					e.SourceType = "SYMPTOM"
				}
				e.TargetType = "TCM_ENTITY"
				e.EvidenceChapter = "中医辨证与特色专科规范"
				g.AddEdge(&e)
			}
		}
	}

	return nil
}
