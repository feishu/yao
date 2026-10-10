package triagekb

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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

// LoadFromFiles 从基座库和扩展库文件构建快照
func (fl *FederatedLoader) LoadFromFiles(generalDBPath, tcmDBPath string) (*GraphSnapshot, error) {
	snapshot := NewGraphSnapshot(fl.dsl.Version)

	// 1. 加载全科现代医学基座库 (Base Layer)
	if generalDBPath != "" {
		if err := fl.loadGeneralDB(snapshot, generalDBPath); err != nil {
			return nil, fmt.Errorf("加载全科基座库失败: %w", err)
		}
		snapshot.Layers = append(snapshot.Layers, "base_general")
	}

	// 2. 加载数智贵中医专病扩展库 (Overlay Layer - Graph Union)
	if tcmDBPath != "" {
		if err := fl.loadTCMDB(snapshot, tcmDBPath); err != nil {
			return nil, fmt.Errorf("加载中医专病扩展库失败: %w", err)
		}
		snapshot.Layers = append(snapshot.Layers, "overlay_tcm")
	}

	// 3. 构建倒排索引
	snapshot.BuildInvertedIndex()

	return snapshot, nil
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

	// 1. 读取 symptoms
	symRows, err := db.Query(`
		SELECT concept_code, concept_name, body_part, pinyin, pinyin_initials,
		       synonyms_json, default_service_code, differential_service_codes_json,
		       is_red_flag_candidate, recommended_age_group, gender_limit,
		       evidence_protocol_id, icd11_code, icd11_title, textbook_source
		FROM symptoms
	`)
	if err != nil {
		return err
	}
	defer symRows.Close()

	for symRows.Next() {
		var s SymptomNode
		var synJSON, diffJSON string
		var isRedFlag int
		if err := symRows.Scan(
			&s.ConceptCode, &s.ConceptName, &s.BodyPart, &s.Pinyin, &s.PinyinInitials,
			&synJSON, &s.DefaultServiceCode, &diffJSON, &isRedFlag,
			&s.RecommendedAgeGroup, &s.GenderLimit, &s.EvidenceProtocolID,
			&s.ICD11Code, &s.ICD11Title, &s.TextbookSource,
		); err != nil {
			return err
		}
		s.IsRedFlagCandidate = isRedFlag == 1
		_ = json.Unmarshal([]byte(synJSON), &s.Synonyms)
		_ = json.Unmarshal([]byte(diffJSON), &s.DifferentialServices)
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

	// 4. 读取 questions
	qRows, err := db.Query(`
		SELECT question_id, title, applicable_symptoms_json, type,
		       category, evidence_dimension, options_json
		FROM questions
	`)
	if err != nil {
		return err
	}
	defer qRows.Close()

	for qRows.Next() {
		var q QuestionNode
		var symsJSON, optJSON string
		if err := qRows.Scan(
			&q.QuestionID, &q.Title, &symsJSON, &q.Type,
			&q.Category, &q.EvidenceDimension, &optJSON,
		); err != nil {
			return err
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
		       relation_type, weight, evidence_quote, evidence_book, evidence_chapter
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

	// 7. 读取 textbook_evidence
	evidRows, err := db.Query(`
		SELECT chunk_id, symptom_code, concept_name, textbook_source,
		       chapter, page, typical_signs_json, typical_symptoms_json, differential_diseases_json
		FROM textbook_evidence
	`)
	if err != nil {
		return err
	}
	defer evidRows.Close()

	for evidRows.Next() {
		var rec EvidenceRecord
		var signsJSON, symsJSON, diffsJSON string
		if err := evidRows.Scan(
			&rec.ChunkID, &rec.SymptomCode, &rec.ConceptName, &rec.TextbookSource,
			&rec.Chapter, &rec.Page, &signsJSON, &symsJSON, &diffsJSON,
		); err != nil {
			return err
		}
		_ = json.Unmarshal([]byte(signsJSON), &rec.TypicalSigns)
		_ = json.Unmarshal([]byte(symsJSON), &rec.TypicalSymptoms)
		_ = json.Unmarshal([]byte(diffsJSON), &rec.DiffDiseases)
		g.EvidenceIndex[rec.SymptomCode] = append(g.EvidenceIndex[rec.SymptomCode], &rec)
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

	// 2. 贵中医专病门诊
	clinicRows, err := db.Query(`
		SELECT clinic_code, clinic_name, dept_name, expert_titles_json, speciality_tags_json, booking_url
		FROM tcm_specialist_clinics
	`)
	if err == nil {
		defer clinicRows.Close()
		for clinicRows.Next() {
			var c TCMClinicNode
			var expJSON, tagsJSON string
			if err := clinicRows.Scan(
				&c.ClinicCode, &c.ClinicName, &c.DeptName,
				&expJSON, &tagsJSON, &c.BookingURL,
			); err == nil {
				_ = json.Unmarshal([]byte(expJSON), &c.ExpertTitles)
				_ = json.Unmarshal([]byte(tagsJSON), &c.SpecialityTags)
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

	// 4. 中医拓扑边 (合并入全局图谱)
	edgeRows, err := db.Query(`
		SELECT source_code, target_code, relation_type, weight, evidence_quote, source_ref
		FROM tcm_edges
	`)
	if err == nil {
		defer edgeRows.Close()
		for edgeRows.Next() {
			var e GraphEdge
			e.SourceType = "SYMPTOM"
			e.TargetType = "TCM_ENTITY"
			if err := edgeRows.Scan(
				&e.SourceCode, &e.TargetCode, &e.RelationType,
				&e.Weight, &e.EvidenceQuote, &e.EvidenceBook,
			); err == nil {
				e.EvidenceChapter = "中医辨证与特色专科规范"
				g.AddEdge(&e)
			}
		}
	}

	return nil
}
