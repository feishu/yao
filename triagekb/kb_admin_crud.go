package triagekb

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func deterministicCode(prefix, raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return prefix + strings.ToUpper(hex.EncodeToString(sum[:]))[:12]
}

func mustMarshalJSON(v interface{}, fallback string) string {
	if v == nil {
		return fallback
	}
	if s, ok := v.(string); ok {
		trimmed := strings.TrimSpace(s)
		if trimmed == "" {
			return fallback
		}
		return trimmed
	}
	b, err := json.Marshal(v)
	if err != nil || len(b) == 0 || string(b) == "null" {
		return fallback
	}
	return string(b)
}

func parseJSONArrayString(raw string) []interface{} {
	out := make([]interface{}, 0)
	if strings.TrimSpace(raw) == "" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func parseJSONMap(raw string) map[string]interface{} {
	out := make(map[string]interface{})
	if strings.TrimSpace(raw) == "" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func toBoolInt(v interface{}, dflt int) int {
	switch val := v.(type) {
	case bool:
		if val {
			return 1
		}
		return 0
	case int:
		if val != 0 {
			return 1
		}
		return 0
	case int64:
		if val != 0 {
			return 1
		}
		return 0
	case float64:
		if val != 0 {
			return 1
		}
		return 0
	case string:
		if val == "1" || strings.EqualFold(val, "true") {
			return 1
		}
		if val == "0" || strings.EqualFold(val, "false") {
			return 0
		}
	}
	return dflt
}

func toFloat(v interface{}, dflt float64) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
			return f
		}
	}
	return dflt
}

func toInt(v interface{}, dflt int) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
			return n
		}
	}
	return dflt
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

func isTCMEntity(entityType string) bool {
	return strings.HasPrefix(entityType, "tcm_")
}

func resolveEntityDB(entityType string, useActive bool) (string, error) {
	appRoot := resolveAdminAppRoot()
	if useActive {
		genPath, tcmPath := resolveActiveDBPaths(appRoot)
		if isTCMEntity(entityType) {
			return tcmPath, nil
		}
		return genPath, nil
	}
	genDraft, tcmDraft, err := ensureDraftInitializedLocked(appRoot, false)
	if err != nil {
		return "", err
	}
	if isTCMEntity(entityType) {
		return tcmDraft, nil
	}
	return genDraft, nil
}

// ProcessKBListEntities 分页与条件查询 Staging 草稿库或激活库实体
func ProcessKBListEntities(proc *process.Process) interface{} {
	if proc.NumOfArgs() < 1 {
		exception.New("缺少 entity_type 参数", 400).Throw()
	}
	entityType := strings.TrimSpace(proc.ArgsString(0))
	params := map[string]interface{}{}
	if proc.NumOfArgs() > 1 {
		if m, ok := proc.Args[1].(map[string]interface{}); ok {
			params = m
		}
	}

	useActive := toString(params["source"]) == "active"
	page := toInt(params["page"], 1)
	if page < 1 {
		page = 1
	}
	pageSize := toInt(params["page_size"], 50)
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 5000 {
		pageSize = 5000
	}

	appRoot := resolveAdminAppRoot()
	adminMu.Lock()
	dbPath, err := resolveEntityDB(entityType, useActive)
	if err != nil {
		adminMu.Unlock()
		exception.New(err.Error(), 500).Throw()
	}
	if !useActive {
		genDraft, tcmDraft := resolveDraftDBPaths(appRoot)
		_ = ensureDraftSchemaAndSeeds(appRoot, genDraft, tcmDraft)
	}
	adminMu.Unlock()

	items, total, qErr := queryEntityList(dbPath, entityType, params, page, pageSize)
	if qErr != nil {
		exception.New(qErr.Error(), 400).Throw()
	}

	return map[string]interface{}{
		"entity_type": entityType,
		"source":      map[bool]string{true: "active", false: "draft"}[useActive],
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"items":       items,
	}
}

func listAllRowsFromDB(dbPath, entityType string) ([]map[string]interface{}, error) {
	items, _, err := queryEntityList(dbPath, entityType, map[string]interface{}{}, 1, 100000)
	return items, err
}

type entityQuerySpec struct {
	table      string
	selectSQL  string
	orderSQL   string
	searchCols []string
	mapper     func(*sql.Rows) (map[string]interface{}, error)
}

func buildEntityQuerySpec(db *sql.DB, entityType string) (*entityQuerySpec, error) {
	switch entityType {
	case "services":
		return &entityQuerySpec{
			table:      "standard_services",
			selectSQL:  `SELECT service_code, service_name, description, primary_dept_keywords_json, fallback_dept_keywords_json, requires_emergency_support FROM standard_services`,
			orderSQL:   `ORDER BY service_code ASC`,
			searchCols: []string{"service_code", "service_name", "description", "primary_dept_keywords_json", "fallback_dept_keywords_json"},
			mapper: func(r *sql.Rows) (map[string]interface{}, error) {
				var code, name, desc, priJSON, fbJSON string
				var reqEmerg int
				if err := r.Scan(&code, &name, &desc, &priJSON, &fbJSON, &reqEmerg); err != nil {
					return nil, err
				}
				return map[string]interface{}{
					"service_code":               code,
					"service_name":               name,
					"description":                desc,
					"primary_dept_keywords":      parseJSONArrayString(priJSON),
					"fallback_dept_keywords":     parseJSONArrayString(fbJSON),
					"requires_emergency_support": reqEmerg == 1,
				}, nil
			},
		}, nil

	case "symptoms":
		cols := sqliteTableColumns(db, "symptoms")
		nrCol := "0 AS needs_review"
		if cols["needs_review"] {
			nrCol = "needs_review"
		}
		acCol := "'[]' AS associated_complications_json"
		if cols["associated_complications_json"] {
			acCol = "associated_complications_json"
		}
		return &entityQuerySpec{
			table: "symptoms",
			selectSQL: fmt.Sprintf(`SELECT concept_code, concept_name, body_part, pinyin, pinyin_initials,
				synonyms_json, default_service_code, differential_service_codes_json,
				is_red_flag_candidate, recommended_age_group, gender_limit,
				COALESCE(evidence_protocol_id, ''), COALESCE(icd11_code, ''), COALESCE(icd11_title, ''),
				COALESCE(textbook_source, ''), %s, %s FROM symptoms`, nrCol, acCol),
			orderSQL:   `ORDER BY needs_review DESC, concept_code ASC`,
			searchCols: []string{"concept_code", "concept_name", "pinyin", "pinyin_initials", "synonyms_json"},
			mapper: func(r *sql.Rows) (map[string]interface{}, error) {
				var code, name, bodyPart, pinyin, pyInit, synJSON, defSvc, diffJSON, ageGrp, gender, protID, icdCode, icdTitle, book, compJSON string
				var redFlag, nr int
				if err := r.Scan(&code, &name, &bodyPart, &pinyin, &pyInit, &synJSON, &defSvc, &diffJSON, &redFlag, &ageGrp, &gender, &protID, &icdCode, &icdTitle, &book, &nr, &compJSON); err != nil {
					return nil, err
				}
				return map[string]interface{}{
					"concept_code":               code,
					"concept_name":               name,
					"body_part":                  bodyPart,
					"pinyin":                     pinyin,
					"pinyin_initials":            pyInit,
					"synonyms":                   parseJSONArrayString(synJSON),
					"default_service_code":       defSvc,
					"differential_service_codes": parseJSONArrayString(diffJSON),
					"is_red_flag_candidate":      redFlag == 1,
					"needs_review":               nr == 1,
					"recommended_age_group":      ageGrp,
					"gender_limit":               gender,
					"evidence_protocol_id":       protID,
					"icd11_code":                 icdCode,
					"icd11_title":                icdTitle,
					"textbook_source":            book,
					"associated_complications":   parseJSONArrayString(compJSON),
				}, nil
			},
		}, nil

	case "questions":
		return &entityQuerySpec{
			table:      "questions",
			selectSQL:  `SELECT question_id, title, applicable_symptoms_json, type, category, COALESCE(evidence_dimension, ''), options_json FROM questions`,
			orderSQL:   `ORDER BY question_id ASC`,
			searchCols: []string{"question_id", "title", "applicable_symptoms_json", "options_json"},
			mapper: func(r *sql.Rows) (map[string]interface{}, error) {
				var qID, title, appJSON, qType, cat, dim, optJSON string
				if err := r.Scan(&qID, &title, &appJSON, &qType, &cat, &dim, &optJSON); err != nil {
					return nil, err
				}
				return map[string]interface{}{
					"question_id":         qID,
					"title":               title,
					"applicable_symptoms": parseJSONArrayString(appJSON),
					"type":                qType,
					"category":            cat,
					"evidence_dimension":  dim,
					"options":             parseJSONArrayString(optJSON),
				}, nil
			},
		}, nil

	case "safety_rules":
		return &entityQuerySpec{
			table:      "safety_rules",
			selectSQL:  `SELECT rule_id, name, level, safety_state, primary_symptoms_json, trigger_condition_json, title, warning_text, emergency_action_json FROM safety_rules`,
			orderSQL:   `ORDER BY CASE WHEN level='CRITICAL' THEN 0 ELSE 1 END, rule_id ASC`,
			searchCols: []string{"rule_id", "name", "title", "warning_text", "primary_symptoms_json"},
			mapper: func(r *sql.Rows) (map[string]interface{}, error) {
				var rID, name, level, state, primJSON, trigJSON, title, warnText, actJSON string
				if err := r.Scan(&rID, &name, &level, &state, &primJSON, &trigJSON, &title, &warnText, &actJSON); err != nil {
					return nil, err
				}
				return map[string]interface{}{
					"rule_id":           rID,
					"name":              name,
					"level":             level,
					"safety_state":      state,
					"primary_symptoms":  parseJSONArrayString(primJSON),
					"trigger_condition": parseJSONMap(trigJSON),
					"title":             title,
					"warning_text":      warnText,
					"emergency_action":  parseJSONMap(actJSON),
				}, nil
			},
		}, nil

	case "diseases":
		return &entityQuerySpec{
			table:      "diseases",
			selectSQL:  `SELECT disease_code, disease_name, dept_service_code, COALESCE(icd10_code, ''), COALESCE(icd11_code, ''), is_critical, category, COALESCE(description, '') FROM diseases`,
			orderSQL:   `ORDER BY is_critical DESC, disease_code ASC`,
			searchCols: []string{"disease_code", "disease_name", "dept_service_code", "description"},
			mapper: func(r *sql.Rows) (map[string]interface{}, error) {
				var code, name, svc, icd10, icd11, cat, desc string
				var isCrit int
				if err := r.Scan(&code, &name, &svc, &icd10, &icd11, &isCrit, &cat, &desc); err != nil {
					return nil, err
				}
				return map[string]interface{}{
					"disease_code":      code,
					"disease_name":      name,
					"dept_service_code": svc,
					"icd10_code":        icd10,
					"icd11_code":        icd11,
					"is_critical":       isCrit == 1,
					"category":          cat,
					"description":       desc,
				}, nil
			},
		}, nil

	case "textbook_evidence":
		return &entityQuerySpec{
			table:      "textbook_evidence",
			selectSQL:  `SELECT id, COALESCE(chunk_id, ''), symptom_code, concept_name, textbook_source, COALESCE(chapter, ''), COALESCE(page, ''), COALESCE(typical_signs_json, '[]'), COALESCE(typical_symptoms_json, '[]'), COALESCE(differential_diseases_json, '[]'), COALESCE(associated_complications_json, '[]') FROM textbook_evidence`,
			orderSQL:   `ORDER BY id ASC`,
			searchCols: []string{"chunk_id", "symptom_code", "concept_name", "textbook_source", "chapter", "differential_diseases_json"},
			mapper: func(r *sql.Rows) (map[string]interface{}, error) {
				var id int
				var chunkID, symCode, name, book, chap, page, signsJ, symsJ, diffJ, compJ string
				if err := r.Scan(&id, &chunkID, &symCode, &name, &book, &chap, &page, &signsJ, &symsJ, &diffJ, &compJ); err != nil {
					return nil, err
				}
				return map[string]interface{}{
					"id":                       id,
					"chunk_id":                 chunkID,
					"symptom_code":             symCode,
					"concept_name":             name,
					"textbook_source":          book,
					"chapter":                  chap,
					"page":                     page,
					"typical_signs":            parseJSONArrayString(signsJ),
					"typical_symptoms":         parseJSONArrayString(symsJ),
					"differential_diseases":    parseJSONArrayString(diffJ),
					"associated_complications": parseJSONArrayString(compJ),
				}, nil
			},
		}, nil

	case "edges":
		return &entityQuerySpec{
			table:      "edges",
			selectSQL:  `SELECT id, source_type, source_code, target_type, target_code, relation_type, weight, COALESCE(evidence_quote, ''), COALESCE(evidence_book, ''), COALESCE(evidence_chapter, '') FROM edges`,
			orderSQL:   `ORDER BY id ASC`,
			searchCols: []string{"source_code", "target_code", "relation_type", "evidence_quote"},
			mapper: func(r *sql.Rows) (map[string]interface{}, error) {
				var id int
				var sType, sCode, tType, tCode, rel, quote, book, chap string
				var weight float64
				if err := r.Scan(&id, &sType, &sCode, &tType, &tCode, &rel, &weight, &quote, &book, &chap); err != nil {
					return nil, err
				}
				return map[string]interface{}{
					"id":               id,
					"source_type":      sType,
					"source_code":      sCode,
					"target_type":      tType,
					"target_code":      tCode,
					"relation_type":    rel,
					"weight":           weight,
					"evidence_quote":   quote,
					"evidence_book":    book,
					"evidence_chapter": chap,
				}, nil
			},
		}, nil

	case "hot_symptoms":
		return &entityQuerySpec{
			table:      "kb_hot_symptoms",
			selectSQL:  `SELECT symptom_code, label, icon, body_part, sort_order, is_active FROM kb_hot_symptoms`,
			orderSQL:   `ORDER BY sort_order ASC, symptom_code ASC`,
			searchCols: []string{"symptom_code", "label", "body_part"},
			mapper: func(r *sql.Rows) (map[string]interface{}, error) {
				var code, label, icon, part string
				var sortOrder, active int
				if err := r.Scan(&code, &label, &icon, &part, &sortOrder, &active); err != nil {
					return nil, err
				}
				return map[string]interface{}{
					"symptom_code": code,
					"label":        label,
					"icon":         icon,
					"body_part":    part,
					"sort_order":   sortOrder,
					"is_active":    active == 1,
				}, nil
			},
		}, nil

	case "body_parts":
		return &entityQuerySpec{
			table:      "kb_body_parts",
			selectSQL:  `SELECT part_code, label, symptom_codes_json, sort_order, is_active FROM kb_body_parts`,
			orderSQL:   `ORDER BY sort_order ASC, part_code ASC`,
			searchCols: []string{"part_code", "label", "symptom_codes_json"},
			mapper: func(r *sql.Rows) (map[string]interface{}, error) {
				var code, label, codesJSON string
				var sortOrder, active int
				if err := r.Scan(&code, &label, &codesJSON, &sortOrder, &active); err != nil {
					return nil, err
				}
				return map[string]interface{}{
					"part_code":     code,
					"label":         label,
					"symptom_codes": parseJSONArrayString(codesJSON),
					"sort_order":    sortOrder,
					"is_active":     active == 1,
				}, nil
			},
		}, nil

	case "keyword_synonyms":
		return &entityQuerySpec{
			table:      "kb_keyword_synonyms",
			selectSQL:  `SELECT term, synonyms_json FROM kb_keyword_synonyms`,
			orderSQL:   `ORDER BY term ASC`,
			searchCols: []string{"term", "synonyms_json"},
			mapper: func(r *sql.Rows) (map[string]interface{}, error) {
				var term, synJSON string
				if err := r.Scan(&term, &synJSON); err != nil {
					return nil, err
				}
				return map[string]interface{}{
					"term":     term,
					"synonyms": parseJSONArrayString(synJSON),
				}, nil
			},
		}, nil

	case "fallback_keywords":
		return &entityQuerySpec{
			table:      "kb_fallback_keywords",
			selectSQL:  `SELECT service_code, keywords_json FROM kb_fallback_keywords`,
			orderSQL:   `ORDER BY service_code ASC`,
			searchCols: []string{"service_code", "keywords_json"},
			mapper: func(r *sql.Rows) (map[string]interface{}, error) {
				var svcCode, kwJSON string
				if err := r.Scan(&svcCode, &kwJSON); err != nil {
					return nil, err
				}
				return map[string]interface{}{
					"service_code": svcCode,
					"keywords":     parseJSONArrayString(kwJSON),
				}, nil
			},
		}, nil

	case "tcm_syndromes":
		return &entityQuerySpec{
			table:      "tcm_syndromes",
			selectSQL:  `SELECT syndrome_code, syndrome_name, organ_system, pathogen_nature, COALESCE(typical_signs_json, '[]'), COALESCE(description, '') FROM tcm_syndromes`,
			orderSQL:   `ORDER BY syndrome_code ASC`,
			searchCols: []string{"syndrome_code", "syndrome_name", "organ_system", "description"},
			mapper: func(r *sql.Rows) (map[string]interface{}, error) {
				var code, name, organ, patho, signsJ, desc string
				if err := r.Scan(&code, &name, &organ, &patho, &signsJ, &desc); err != nil {
					return nil, err
				}
				return map[string]interface{}{
					"syndrome_code":   code,
					"syndrome_name":   name,
					"organ_system":    organ,
					"pathogen_nature": patho,
					"typical_signs":   parseJSONArrayString(signsJ),
					"description":     desc,
				}, nil
			},
		}, nil

	case "tcm_diseases":
		return &entityQuerySpec{
			table:      "tcm_diseases",
			selectSQL:  `SELECT disease_code, disease_name, COALESCE(modern_disease_name, ''), COALESCE(modern_disease_code, ''), preferred_dept_name, COALESCE(description, '') FROM tcm_diseases`,
			orderSQL:   `ORDER BY disease_code ASC`,
			searchCols: []string{"disease_code", "disease_name", "modern_disease_name", "preferred_dept_name"},
			mapper: func(r *sql.Rows) (map[string]interface{}, error) {
				var code, name, modName, modCode, dept, desc string
				if err := r.Scan(&code, &name, &modName, &modCode, &dept, &desc); err != nil {
					return nil, err
				}
				return map[string]interface{}{
					"disease_code":        code,
					"disease_name":        name,
					"modern_disease_name": modName,
					"modern_disease_code": modCode,
					"preferred_dept_name": dept,
					"description":         desc,
				}, nil
			},
		}, nil

	case "tcm_specialist_clinics":
		cols := sqliteTableColumns(db, "tcm_specialist_clinics")
		kwCol := "'[]' AS target_dept_keywords_json"
		if cols["target_dept_keywords_json"] {
			kwCol = "COALESCE(target_dept_keywords_json, '[]')"
		}
		return &entityQuerySpec{
			table:      "tcm_specialist_clinics",
			selectSQL:  fmt.Sprintf(`SELECT clinic_code, clinic_name, dept_name, COALESCE(expert_titles_json, '[]'), COALESCE(speciality_tags_json, '[]'), COALESCE(booking_url, ''), %s FROM tcm_specialist_clinics`, kwCol),
			orderSQL:   `ORDER BY clinic_code ASC`,
			searchCols: []string{"clinic_code", "clinic_name", "dept_name", "speciality_tags_json"},
			mapper: func(r *sql.Rows) (map[string]interface{}, error) {
				var code, name, dept, expJ, tagJ, url, kwJ string
				if err := r.Scan(&code, &name, &dept, &expJ, &tagJ, &url, &kwJ); err != nil {
					return nil, err
				}
				kwArr := parseJSONArrayString(kwJ)
				if len(kwArr) == 0 && dept != "" {
					kwArr = []interface{}{dept}
				}
				return map[string]interface{}{
					"clinic_code":          code,
					"clinic_name":          name,
					"dept_name":            dept,
					"expert_titles":        parseJSONArrayString(expJ),
					"speciality_tags":      parseJSONArrayString(tagJ),
					"booking_url":          url,
					"target_dept_keywords": kwArr,
				}, nil
			},
		}, nil

	case "tcm_therapies":
		return &entityQuerySpec{
			table:      "tcm_therapies",
			selectSQL:  `SELECT therapy_code, therapy_name, category, COALESCE(indications_json, '[]'), COALESCE(clinic_recommendation, ''), COALESCE(description, '') FROM tcm_therapies`,
			orderSQL:   `ORDER BY therapy_code ASC`,
			searchCols: []string{"therapy_code", "therapy_name", "category", "indications_json"},
			mapper: func(r *sql.Rows) (map[string]interface{}, error) {
				var code, name, cat, indJ, clinicRec, desc string
				if err := r.Scan(&code, &name, &cat, &indJ, &clinicRec, &desc); err != nil {
					return nil, err
				}
				return map[string]interface{}{
					"therapy_code":          code,
					"therapy_name":          name,
					"category":              cat,
					"indications":           parseJSONArrayString(indJ),
					"clinic_recommendation": clinicRec,
					"description":           desc,
				}, nil
			},
		}, nil

	case "tcm_edges":
		return &entityQuerySpec{
			table:      "tcm_edges",
			selectSQL:  `SELECT id, source_code, target_code, relation_type, weight, COALESCE(evidence_quote, ''), COALESCE(source_ref, '') FROM tcm_edges`,
			orderSQL:   `ORDER BY id ASC`,
			searchCols: []string{"source_code", "target_code", "relation_type", "evidence_quote"},
			mapper: func(r *sql.Rows) (map[string]interface{}, error) {
				var id int
				var src, dst, rel, quote, ref string
				var w float64
				if err := r.Scan(&id, &src, &dst, &rel, &w, &quote, &ref); err != nil {
					return nil, err
				}
				return map[string]interface{}{
					"id":             id,
					"source_code":    src,
					"target_code":    dst,
					"relation_type":  rel,
					"weight":         w,
					"evidence_quote": quote,
					"source_ref":     ref,
				}, nil
			},
		}, nil
	}

	return nil, fmt.Errorf("不支持的知识实体类型: %s", entityType)
}

func queryEntityList(dbPath, entityType string, params map[string]interface{}, page, pageSize int) ([]map[string]interface{}, int, error) {
	db, err := openSQLiteRO(dbPath)
	if err != nil {
		return nil, 0, err
	}
	defer db.Close()

	spec, err := buildEntityQuerySpec(db, entityType)
	if err != nil {
		return nil, 0, err
	}
	if !sqliteHasTable(db, spec.table) {
		return []map[string]interface{}{}, 0, nil
	}

	whereClauses := []string{"1=1"}
	args := make([]interface{}, 0)

	if kw := toString(params["keyword"]); kw != "" && len(spec.searchCols) > 0 {
		pattern := "%" + kw + "%"
		orParts := make([]string, 0, len(spec.searchCols))
		for _, col := range spec.searchCols {
			orParts = append(orParts, fmt.Sprintf("%s LIKE ?", col))
			args = append(args, pattern)
		}
		whereClauses = append(whereClauses, "("+strings.Join(orParts, " OR ")+")")
	}

	// 专用过滤条件
	switch entityType {
	case "symptoms":
		if nrStr, ok := params["needs_review"]; ok && nrStr != nil && toString(nrStr) != "" {
			cols := sqliteTableColumns(db, "symptoms")
			if cols["needs_review"] {
				whereClauses = append(whereClauses, "needs_review = ?")
				args = append(args, toBoolInt(nrStr, 0))
			}
		}
		if rfStr, ok := params["is_red_flag_candidate"]; ok && rfStr != nil && toString(rfStr) != "" {
			whereClauses = append(whereClauses, "is_red_flag_candidate = ?")
			args = append(args, toBoolInt(rfStr, 0))
		}
		if svc := toString(params["service_code"]); svc != "" {
			whereClauses = append(whereClauses, "default_service_code = ?")
			args = append(args, svc)
		}
	case "questions":
		if sym := toString(params["symptom_code"]); sym != "" {
			whereClauses = append(whereClauses, "applicable_symptoms_json LIKE ?")
			args = append(args, "%"+sym+"%")
		}
		if cat := toString(params["category"]); cat != "" {
			whereClauses = append(whereClauses, "category = ?")
			args = append(args, cat)
		}
	case "safety_rules":
		if level := toString(params["level"]); level != "" {
			whereClauses = append(whereClauses, "level = ?")
			args = append(args, level)
		}
		if sym := toString(params["symptom_code"]); sym != "" {
			whereClauses = append(whereClauses, "primary_symptoms_json LIKE ?")
			args = append(args, "%"+sym+"%")
		}
	case "diseases":
		if svc := toString(params["service_code"]); svc != "" {
			whereClauses = append(whereClauses, "dept_service_code = ?")
			args = append(args, svc)
		}
	case "textbook_evidence":
		if sym := toString(params["symptom_code"]); sym != "" {
			whereClauses = append(whereClauses, "symptom_code = ?")
			args = append(args, sym)
		}
	case "edges":
		if sym := toString(params["source_code"]); sym != "" {
			whereClauses = append(whereClauses, "source_code = ?")
			args = append(args, sym)
		}
		if rel := toString(params["relation_type"]); rel != "" {
			whereClauses = append(whereClauses, "relation_type = ?")
			args = append(args, rel)
		}
	case "tcm_edges":
		if src := toString(params["source_code"]); src != "" {
			whereClauses = append(whereClauses, "source_code = ?")
			args = append(args, src)
		}
		if dst := toString(params["target_code"]); dst != "" {
			whereClauses = append(whereClauses, "target_code = ?")
			args = append(args, dst)
		}
	}

	whereSQL := " WHERE " + strings.Join(whereClauses, " AND ")
	var total int
	if err := db.QueryRow(fmt.Sprintf(`SELECT COUNT(1) FROM "%s"%s`, spec.table, whereSQL), args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	queryArgs := append(append([]interface{}{}, args...), pageSize, offset)
	rows, err := db.Query(spec.selectSQL+whereSQL+" "+spec.orderSQL+" LIMIT ? OFFSET ?", queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]map[string]interface{}, 0)
	for rows.Next() {
		rowMap, mErr := spec.mapper(rows)
		if mErr != nil {
			return nil, 0, mErr
		}
		items = append(items, rowMap)
	}
	return items, total, nil
}

// ProcessKBGetEntity 查询单条知识实体详情（症状额外聚合鉴别疾病权重、教材循证、追问与安全规则）
func ProcessKBGetEntity(proc *process.Process) interface{} {
	if proc.NumOfArgs() < 2 {
		exception.New("缺少 entity_type 或 code 参数", 400).Throw()
	}
	entityType := strings.TrimSpace(proc.ArgsString(0))
	code := strings.TrimSpace(proc.ArgsString(1))

	adminMu.Lock()
	dbPath, err := resolveEntityDB(entityType, false)
	adminMu.Unlock()
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	db, err := openSQLiteRO(dbPath)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	defer db.Close()

	spec, err := buildEntityQuerySpec(db, entityType)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}

	pkCol := primaryKeyColumn(entityType)
	rows, err := db.Query(fmt.Sprintf("%s WHERE %s = ? LIMIT 1", spec.selectSQL, pkCol), code)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	var item map[string]interface{}
	if rows.Next() {
		item, err = spec.mapper(rows)
	}
	rows.Close()
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	if item == nil {
		exception.New(fmt.Sprintf("未找到 %s: %s", entityType, code), 404).Throw()
	}

	// 症状详情内嵌鉴别疾病 INDICATES 权重、教材原文片段、关联追问与规则
	if entityType == "symptoms" {
		indRows, iErr := db.Query(`
			SELECT e.target_code, COALESCE(d.disease_name, e.target_code),
			       COALESCE(d.dept_service_code, ''), COALESCE(d.is_critical, 0),
			       COALESCE(d.description, ''), e.weight, COALESCE(e.evidence_quote, ''),
			       COALESCE(e.evidence_book, ''), COALESCE(e.evidence_chapter, '')
			FROM edges e
			LEFT JOIN diseases d ON d.disease_code = e.target_code
			WHERE e.source_type = 'SYMPTOM' AND e.source_code = ? AND e.relation_type = 'INDICATES'
			ORDER BY e.weight DESC, e.target_code ASC
		`, code)
		indicatives := make([]map[string]interface{}, 0)
		if iErr == nil {
			defer indRows.Close()
			for indRows.Next() {
				var dCode, dName, dSvc, dDesc, quote, book, chap string
				var isCrit int
				var w float64
				if indRows.Scan(&dCode, &dName, &dSvc, &isCrit, &dDesc, &w, &quote, &book, &chap) == nil {
					indicatives = append(indicatives, map[string]interface{}{
						"disease_code":      dCode,
						"disease_name":      dName,
						"dept_service_code": dSvc,
						"is_critical":       isCrit == 1,
						"description":       dDesc,
						"weight":            w,
						"evidence_quote":    quote,
						"evidence_book":     book,
						"evidence_chapter":  chap,
					})
				}
			}
		}
		item["indicative_diseases"] = indicatives

		evidences, _, _ := queryEntityList(dbPath, "textbook_evidence", map[string]interface{}{"symptom_code": code}, 1, 20)
		item["textbook_evidence"] = evidences
		questions, _, _ := queryEntityList(dbPath, "questions", map[string]interface{}{"symptom_code": code}, 1, 50)
		item["linked_questions"] = questions
		rules, _, _ := queryEntityList(dbPath, "safety_rules", map[string]interface{}{"symptom_code": code}, 1, 50)
		item["linked_safety_rules"] = rules
	}

	if entityType == "tcm_specialist_clinics" || entityType == "tcm_syndromes" || entityType == "tcm_therapies" {
		edges, _, _ := queryEntityList(dbPath, "tcm_edges", map[string]interface{}{"target_code": code}, 1, 100)
		item["tcm_edges"] = edges
	}

	return item
}

func primaryKeyColumn(entityType string) string {
	switch entityType {
	case "services":
		return "service_code"
	case "symptoms":
		return "concept_code"
	case "questions":
		return "question_id"
	case "safety_rules":
		return "rule_id"
	case "diseases":
		return "disease_code"
	case "textbook_evidence":
		return "id"
	case "edges":
		return "id"
	case "hot_symptoms":
		return "symptom_code"
	case "body_parts":
		return "part_code"
	case "keyword_synonyms":
		return "term"
	case "fallback_keywords":
		return "service_code"
	case "tcm_syndromes":
		return "syndrome_code"
	case "tcm_diseases":
		return "disease_code"
	case "tcm_specialist_clinics":
		return "clinic_code"
	case "tcm_therapies":
		return "therapy_code"
	case "tcm_edges":
		return "id"
	default:
		return "id"
	}
}

// ProcessKBSaveEntity 在单事务中保存知识库实体并自动级联重算衍生边（废除只读表误写）
func ProcessKBSaveEntity(proc *process.Process) interface{} {
	if proc.NumOfArgs() < 2 {
		exception.New("缺少 entity_type 或 payload 参数", 400).Throw()
	}
	entityType := strings.TrimSpace(proc.ArgsString(0))
	payload, ok := proc.Args[1].(map[string]interface{})
	if !ok {
		exception.New("payload 必须为对象", 400).Throw()
	}

	if entityType == "diseases" || entityType == "textbook_evidence" || entityType == "edges" {
		exception.New("疾病与教材原文主表仅支持只读反查，请在症状详情内联维护 INDICATES 鉴别疾病权重边", 400).Throw()
	}

	appRoot := resolveAdminAppRoot()
	adminMu.Lock()
	defer adminMu.Unlock()

	genDraftPath, _, initErr := ensureDraftInitializedLocked(appRoot, false)
	if initErr != nil {
		exception.New(initErr.Error(), 500).Throw()
	}
	dbPath, err := resolveEntityDB(entityType, false)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	db, err := openSQLiteRW(dbPath)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	defer db.Close()

	if err := checkPlaybookOwnershipConflict(genDraftPath, entityType, "", payload); err != nil {
		exception.New(err.Error(), httpCodeFromError(err, 409)).Throw()
	}
	tx, err := db.Begin()
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	defer func() { _ = tx.Rollback() }()

	savedCode, edgeStats, saveErr := saveEntityInTx(tx, entityType, payload)
	if saveErr != nil {
		exception.New(saveErr.Error(), 400).Throw()
	}
	if err := tx.Commit(); err != nil {
		exception.New(fmt.Sprintf("提交事务失败: %v", err), 500).Throw()
	}

	return map[string]interface{}{
		"success":     true,
		"entity_type": entityType,
		"code":        savedCode,
		"edge_sync":   edgeStats,
	}
}

func saveEntityInTx(tx *sql.Tx, entityType string, p map[string]interface{}) (string, map[string]int, error) {
	edgeStats := map[string]int{}

	switch entityType {
	case "services":
		code := toString(p["service_code"])
		name := toString(p["service_name"])
		if code == "" || name == "" {
			return "", nil, fmt.Errorf("service_code 与 service_name 必填")
		}
		_, err := tx.Exec(`
			INSERT INTO standard_services (service_code, service_name, description, primary_dept_keywords_json, fallback_dept_keywords_json, requires_emergency_support)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(service_code) DO UPDATE SET
				service_name = excluded.service_name,
				description = excluded.description,
				primary_dept_keywords_json = excluded.primary_dept_keywords_json,
				fallback_dept_keywords_json = excluded.fallback_dept_keywords_json,
				requires_emergency_support = excluded.requires_emergency_support
		`, code, name, toString(p["description"]),
			mustMarshalJSON(p["primary_dept_keywords"], "[]"),
			mustMarshalJSON(p["fallback_dept_keywords"], "[]"),
			toBoolInt(p["requires_emergency_support"], 0))
		return code, edgeStats, err

	case "symptoms":
		code := toString(p["concept_code"])
		name := toString(p["concept_name"])
		if code == "" || name == "" {
			return "", nil, fmt.Errorf("concept_code 与 concept_name 必填")
		}
		defSvc := toString(p["default_service_code"])
		if defSvc == "" {
			defSvc = "GENERAL_INTERNAL_ASSESSMENT"
		}
		bodyPart := toString(p["body_part"])
		if bodyPart == "" {
			bodyPart = "GENERAL"
		}
		ageGroup := toString(p["recommended_age_group"])
		if ageGroup == "" {
			ageGroup = "ALL"
		}
		genderLimit := toString(p["gender_limit"])
		if genderLimit == "" {
			genderLimit = "ALL"
		}
		textbook := toString(p["textbook_source"])
		if textbook == "" {
			textbook = "平台医学编辑维护"
		}
		_, err := tx.Exec(`
			INSERT INTO symptoms (
				concept_code, concept_name, body_part, pinyin, pinyin_initials,
				synonyms_json, default_service_code, differential_service_codes_json,
				is_red_flag_candidate, recommended_age_group, gender_limit,
				evidence_protocol_id, icd11_code, icd11_title, textbook_source,
				needs_review, associated_complications_json
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
				evidence_protocol_id = excluded.evidence_protocol_id,
				icd11_code = excluded.icd11_code,
				icd11_title = excluded.icd11_title,
				textbook_source = excluded.textbook_source,
				needs_review = excluded.needs_review,
				associated_complications_json = excluded.associated_complications_json
		`, code, name, bodyPart, toString(p["pinyin"]), toString(p["pinyin_initials"]),
			mustMarshalJSON(p["synonyms"], "[]"), defSvc, mustMarshalJSON(p["differential_service_codes"], "[]"),
			toBoolInt(p["is_red_flag_candidate"], 0), ageGroup, genderLimit,
			toString(p["evidence_protocol_id"]), toString(p["icd11_code"]), toString(p["icd11_title"]),
			textbook, toBoolInt(p["needs_review"], 0), mustMarshalJSON(p["associated_complications"], "[]"))
		if err != nil {
			return "", nil, err
		}

		// 事务级联重建 DEFAULT_SERVICE 边
		if _, err := tx.Exec(`DELETE FROM edges WHERE source_type = 'SYMPTOM' AND source_code = ? AND relation_type = 'DEFAULT_SERVICE'`, code); err != nil {
			return "", nil, err
		}
		if _, err := tx.Exec(`
			INSERT INTO edges (source_type, source_code, target_type, target_code, relation_type, weight, evidence_quote, evidence_book, evidence_chapter)
			VALUES ('SYMPTOM', ?, 'SERVICE', ?, 'DEFAULT_SERVICE', 1.0, ?, ?, '服务映射规范')
		`, code, defSvc, "主诉默认初诊服务匹配: "+defSvc, textbook); err != nil {
			return "", nil, err
		}
		edgeStats["default_service_edges"] = 1

		// 若显式提交了 indicative_diseases，则事务级联重建该症状的 INDICATES 疾病鉴别边（确定性 SHA-256 编码）
		if rawInd, hasInd := p["indicative_diseases"]; hasInd && rawInd != nil {
			var indList []map[string]interface{}
			_ = json.Unmarshal([]byte(mustMarshalJSON(rawInd, "[]")), &indList)
			if _, err := tx.Exec(`DELETE FROM edges WHERE source_type = 'SYMPTOM' AND source_code = ? AND relation_type = 'INDICATES'`, code); err != nil {
				return "", nil, err
			}
			insertedInd := 0
			for _, item := range indList {
				disName := toString(item["disease_name"])
				disCode := toString(item["disease_code"])
				if disName == "" && disCode == "" {
					continue
				}
				if disCode == "" {
					disCode = deterministicCode("DIS_", disName)
				}
				disSvc := toString(item["dept_service_code"])
				if disSvc == "" {
					disSvc = defSvc
				}
				isCrit := toBoolInt(item["is_critical"], 0)
				cat := "COMMON"
				if isCrit == 1 {
					cat = "CRITICAL"
				}
				disDesc := toString(item["description"])
				if _, err := tx.Exec(`
					INSERT INTO diseases (disease_code, disease_name, dept_service_code, icd10_code, icd11_code, is_critical, category, description)
					VALUES (?, ?, ?, '', '', ?, ?, ?)
					ON CONFLICT(disease_code) DO UPDATE SET
						disease_name = COALESCE(NULLIF(excluded.disease_name, ''), diseases.disease_name),
						dept_service_code = excluded.dept_service_code,
						is_critical = excluded.is_critical,
						category = excluded.category,
						description = CASE WHEN excluded.description != '' THEN excluded.description ELSE diseases.description END
				`, disCode, disName, disSvc, isCrit, cat, disDesc); err != nil {
					return "", nil, err
				}
				w := toFloat(item["weight"], 0.70)
				if w <= 0 {
					w = 0.70
				}
				quote := toString(item["evidence_quote"])
				if quote == "" {
					quote = disDesc
				}
				book := toString(item["evidence_book"])
				if book == "" {
					book = textbook
				}
				chap := toString(item["evidence_chapter"])
				if _, err := tx.Exec(`
					INSERT INTO edges (source_type, source_code, target_type, target_code, relation_type, weight, evidence_quote, evidence_book, evidence_chapter)
					VALUES ('SYMPTOM', ?, 'DISEASE', ?, 'INDICATES', ?, ?, ?, ?)
				`, code, disCode, w, quote, book, chap); err != nil {
					return "", nil, err
				}
				insertedInd++
			}
			edgeStats["indicates_edges"] = insertedInd
		}
		return code, edgeStats, nil

	case "questions":
		qID := toString(p["question_id"])
		title := toString(p["title"])
		if qID == "" || title == "" {
			return "", nil, fmt.Errorf("question_id 与 title 必填")
		}
		qType := toString(p["type"])
		if qType == "" {
			qType = "SINGLE_CHOICE"
		}
		cat := toString(p["category"])
		if cat == "" {
			cat = "DIFFERENTIAL"
		}
		_, err := tx.Exec(`
			INSERT INTO questions (question_id, title, applicable_symptoms_json, type, category, evidence_dimension, options_json)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(question_id) DO UPDATE SET
				title = excluded.title,
				applicable_symptoms_json = excluded.applicable_symptoms_json,
				type = excluded.type,
				category = excluded.category,
				evidence_dimension = excluded.evidence_dimension,
				options_json = excluded.options_json
		`, qID, title, mustMarshalJSON(p["applicable_symptoms"], "[]"), qType, cat,
			toString(p["evidence_dimension"]), mustMarshalJSON(p["options"], "[]"))
		return qID, edgeStats, err

	case "safety_rules":
		rID := toString(p["rule_id"])
		name := toString(p["name"])
		if rID == "" || name == "" {
			return "", nil, fmt.Errorf("rule_id 与 name 必填")
		}
		level := strings.ToUpper(toString(p["level"]))
		if level == "" {
			level = "URGENT"
		}
		safetyState := strings.ToUpper(toString(p["safety_state"]))
		if safetyState == "" {
			if level == "CRITICAL" {
				safetyState = "EMERGENCY_NOW"
			} else {
				safetyState = "URGENT_ASSESSMENT"
			}
		}
		primJSON := mustMarshalJSON(p["primary_symptoms"], "[]")
		warnText := toString(p["warning_text"])
		_, err := tx.Exec(`
			INSERT INTO safety_rules (rule_id, name, level, safety_state, primary_symptoms_json, trigger_condition_json, title, warning_text, emergency_action_json)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(rule_id) DO UPDATE SET
				name = excluded.name,
				level = excluded.level,
				safety_state = excluded.safety_state,
				primary_symptoms_json = excluded.primary_symptoms_json,
				trigger_condition_json = excluded.trigger_condition_json,
				title = excluded.title,
				warning_text = excluded.warning_text,
				emergency_action_json = excluded.emergency_action_json
		`, rID, name, level, safetyState, primJSON, mustMarshalJSON(p["trigger_condition"], "{}"),
			toString(p["title"]), warnText, mustMarshalJSON(p["emergency_action"], "{}"))
		if err != nil {
			return "", nil, err
		}

		// 事务级联重建 TRIGGERS_SAFETY 边
		if _, err := tx.Exec(`DELETE FROM edges WHERE target_type = 'RED_FLAG_RULE' AND target_code = ? AND relation_type = 'TRIGGERS_SAFETY'`, rID); err != nil {
			return "", nil, err
		}
		var primSyms []string
		_ = json.Unmarshal([]byte(primJSON), &primSyms)
		inserted := 0
		for _, symCode := range primSyms {
			symCode = strings.TrimSpace(symCode)
			if symCode == "" {
				continue
			}
			if _, err := tx.Exec(`
				INSERT INTO edges (source_type, source_code, target_type, target_code, relation_type, weight, evidence_quote, evidence_book, evidence_chapter)
				VALUES ('SYMPTOM', ?, 'RED_FLAG_RULE', ?, 'TRIGGERS_SAFETY', 1.0, ?, '临床安全预警规范', ?)
			`, symCode, rID, warnText, name); err != nil {
				return "", nil, err
			}
			inserted++
		}
		edgeStats["triggers_safety_edges"] = inserted
		return rID, edgeStats, nil

	case "hot_symptoms":
		code := toString(p["symptom_code"])
		label := toString(p["label"])
		if code == "" || label == "" {
			return "", nil, fmt.Errorf("symptom_code 与 label 必填")
		}
		icon := toString(p["icon"])
		if icon == "" {
			icon = "stethoscope"
		}
		part := toString(p["body_part"])
		if part == "" {
			part = "GENERAL"
		}
		_, err := tx.Exec(`
			INSERT INTO kb_hot_symptoms (symptom_code, label, icon, body_part, sort_order, is_active)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(symptom_code) DO UPDATE SET
				label = excluded.label, icon = excluded.icon, body_part = excluded.body_part,
				sort_order = excluded.sort_order, is_active = excluded.is_active
		`, code, label, icon, part, toInt(p["sort_order"], 100), toBoolInt(p["is_active"], 1))
		return code, edgeStats, err

	case "body_parts":
		code := toString(p["part_code"])
		label := toString(p["label"])
		if code == "" || label == "" {
			return "", nil, fmt.Errorf("part_code 与 label 必填")
		}
		_, err := tx.Exec(`
			INSERT INTO kb_body_parts (part_code, label, symptom_codes_json, sort_order, is_active)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(part_code) DO UPDATE SET
				label = excluded.label, symptom_codes_json = excluded.symptom_codes_json,
				sort_order = excluded.sort_order, is_active = excluded.is_active
		`, code, label, mustMarshalJSON(p["symptom_codes"], "[]"), toInt(p["sort_order"], 100), toBoolInt(p["is_active"], 1))
		return code, edgeStats, err

	case "keyword_synonyms":
		term := toString(p["term"])
		if term == "" {
			return "", nil, fmt.Errorf("term 必填")
		}
		_, err := tx.Exec(`
			INSERT INTO kb_keyword_synonyms (term, synonyms_json) VALUES (?, ?)
			ON CONFLICT(term) DO UPDATE SET synonyms_json = excluded.synonyms_json
		`, term, mustMarshalJSON(p["synonyms"], "[]"))
		return term, edgeStats, err

	case "fallback_keywords":
		code := toString(p["service_code"])
		if code == "" {
			return "", nil, fmt.Errorf("service_code 必填")
		}
		_, err := tx.Exec(`
			INSERT INTO kb_fallback_keywords (service_code, keywords_json) VALUES (?, ?)
			ON CONFLICT(service_code) DO UPDATE SET keywords_json = excluded.keywords_json
		`, code, mustMarshalJSON(p["keywords"], "[]"))
		return code, edgeStats, err

	case "tcm_syndromes":
		code := toString(p["syndrome_code"])
		name := toString(p["syndrome_name"])
		if code == "" || name == "" {
			return "", nil, fmt.Errorf("syndrome_code 与 syndrome_name 必填")
		}
		_, err := tx.Exec(`
			INSERT INTO tcm_syndromes (syndrome_code, syndrome_name, organ_system, pathogen_nature, typical_signs_json, description)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(syndrome_code) DO UPDATE SET
				syndrome_name = excluded.syndrome_name, organ_system = excluded.organ_system,
				pathogen_nature = excluded.pathogen_nature, typical_signs_json = excluded.typical_signs_json,
				description = excluded.description
		`, code, name, toString(p["organ_system"]), toString(p["pathogen_nature"]),
			mustMarshalJSON(p["typical_signs"], "[]"), toString(p["description"]))
		return code, edgeStats, err

	case "tcm_diseases":
		code := toString(p["disease_code"])
		name := toString(p["disease_name"])
		if code == "" || name == "" {
			return "", nil, fmt.Errorf("disease_code 与 disease_name 必填")
		}
		_, err := tx.Exec(`
			INSERT INTO tcm_diseases (disease_code, disease_name, modern_disease_name, modern_disease_code, preferred_dept_name, description)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(disease_code) DO UPDATE SET
				disease_name = excluded.disease_name, modern_disease_name = excluded.modern_disease_name,
				modern_disease_code = excluded.modern_disease_code, preferred_dept_name = excluded.preferred_dept_name,
				description = excluded.description
		`, code, name, toString(p["modern_disease_name"]), toString(p["modern_disease_code"]),
			toString(p["preferred_dept_name"]), toString(p["description"]))
		return code, edgeStats, err

	case "tcm_specialist_clinics":
		code := toString(p["clinic_code"])
		name := toString(p["clinic_name"])
		dept := toString(p["dept_name"])
		if code == "" || name == "" || dept == "" {
			return "", nil, fmt.Errorf("clinic_code、clinic_name 与 dept_name 必填")
		}
		bookingURL := toString(p["booking_url"])
		if strings.Contains(bookingURL, "/registration/specialist-clinic/") {
			return "", nil, fmt.Errorf("禁止使用废弃假挂号链接 %s，请置空并改配 target_dept_keywords", bookingURL)
		}
		kwJSON := mustMarshalJSON(p["target_dept_keywords"], "[]")
		if kwJSON == "[]" && dept != "" {
			kwJSON = mustMarshalJSON([]string{dept}, "[]")
		}
		_, err := tx.Exec(`
			INSERT INTO tcm_specialist_clinics (clinic_code, clinic_name, dept_name, expert_titles_json, speciality_tags_json, booking_url, target_dept_keywords_json)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(clinic_code) DO UPDATE SET
				clinic_name = excluded.clinic_name, dept_name = excluded.dept_name,
				expert_titles_json = excluded.expert_titles_json, speciality_tags_json = excluded.speciality_tags_json,
				booking_url = excluded.booking_url, target_dept_keywords_json = excluded.target_dept_keywords_json
		`, code, name, dept, mustMarshalJSON(p["expert_titles"], "[]"),
			mustMarshalJSON(p["speciality_tags"], "[]"), bookingURL, kwJSON)
		if err != nil {
			return "", nil, err
		}
		if rawEdges, hasEdges := p["tcm_edges"]; hasEdges && rawEdges != nil {
			if _, err := tx.Exec(`DELETE FROM tcm_edges WHERE target_code = ? AND relation_type = 'SYMPTOM_TO_CLINIC'`, code); err != nil {
				return "", nil, err
			}
			var edgeList []map[string]interface{}
			_ = json.Unmarshal([]byte(mustMarshalJSON(rawEdges, "[]")), &edgeList)
			for _, e := range edgeList {
				src := toString(e["source_code"])
				if src == "" {
					continue
				}
				w := toFloat(e["weight"], 0.85)
				if _, err := tx.Exec(`
					INSERT INTO tcm_edges (source_code, target_code, relation_type, weight, evidence_quote, source_ref)
					VALUES (?, ?, 'SYMPTOM_TO_CLINIC', ?, ?, ?)
				`, src, code, w, toString(e["evidence_quote"]), toString(e["source_ref"])); err != nil {
					return "", nil, err
				}
				edgeStats["tcm_edges"]++
			}
		}
		return code, edgeStats, nil

	case "tcm_therapies":
		code := toString(p["therapy_code"])
		name := toString(p["therapy_name"])
		if code == "" || name == "" {
			return "", nil, fmt.Errorf("therapy_code 与 therapy_name 必填")
		}
		_, err := tx.Exec(`
			INSERT INTO tcm_therapies (therapy_code, therapy_name, category, indications_json, clinic_recommendation, description)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(therapy_code) DO UPDATE SET
				therapy_name = excluded.therapy_name, category = excluded.category,
				indications_json = excluded.indications_json, clinic_recommendation = excluded.clinic_recommendation,
				description = excluded.description
		`, code, name, toString(p["category"]), mustMarshalJSON(p["indications"], "[]"),
			toString(p["clinic_recommendation"]), toString(p["description"]))
		return code, edgeStats, err

	case "tcm_edges":
		src := toString(p["source_code"])
		dst := toString(p["target_code"])
		rel := toString(p["relation_type"])
		if src == "" || dst == "" || rel == "" {
			return "", nil, fmt.Errorf("source_code、target_code 与 relation_type 必填")
		}
		idVal := toInt(p["id"], 0)
		w := toFloat(p["weight"], 0.85)
		if idVal > 0 {
			_, err := tx.Exec(`UPDATE tcm_edges SET source_code=?, target_code=?, relation_type=?, weight=?, evidence_quote=?, source_ref=? WHERE id=?`,
				src, dst, rel, w, toString(p["evidence_quote"]), toString(p["source_ref"]), idVal)
			return strconv.Itoa(idVal), edgeStats, err
		}
		res, err := tx.Exec(`INSERT INTO tcm_edges (source_code, target_code, relation_type, weight, evidence_quote, source_ref) VALUES (?, ?, ?, ?, ?, ?)`,
			src, dst, rel, w, toString(p["evidence_quote"]), toString(p["source_ref"]))
		if err != nil {
			return "", nil, err
		}
		lastID, _ := res.LastInsertId()
		return strconv.FormatInt(lastID, 10), edgeStats, nil
	}

	return "", nil, fmt.Errorf("不支持保存的实体类型: %s", entityType)
}

// ProcessKBDeleteEntity 删除 Staging 草稿库中允许删除的实体并校验引用完整性
func ProcessKBDeleteEntity(proc *process.Process) interface{} {
	if proc.NumOfArgs() < 2 {
		exception.New("缺少 entity_type 或 code 参数", 400).Throw()
	}
	entityType := strings.TrimSpace(proc.ArgsString(0))
	code := strings.TrimSpace(proc.ArgsString(1))

	if entityType == "diseases" || entityType == "textbook_evidence" || entityType == "edges" || entityType == "services" {
		exception.New("标准临床方向与底层循证数据受完整性保护，不允许直接物理删除", 400).Throw()
	}

	appRoot := resolveAdminAppRoot()
	adminMu.Lock()
	defer adminMu.Unlock()

	genDraftPath, _, initErr := ensureDraftInitializedLocked(appRoot, false)
	if initErr != nil {
		exception.New(initErr.Error(), 500).Throw()
	}
	dbPath, err := resolveEntityDB(entityType, false)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	db, err := openSQLiteRW(dbPath)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	defer db.Close()

	if err := checkPlaybookOwnershipConflict(genDraftPath, entityType, code, nil); err != nil {
		exception.New(err.Error(), httpCodeFromError(err, 409)).Throw()
	}
	tx, err := db.Begin()
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	defer func() { _ = tx.Rollback() }()

	switch entityType {
	case "questions":
		var refCnt int
		_ = tx.QueryRow(`SELECT COUNT(1) FROM safety_rules WHERE trigger_condition_json LIKE ?`, "%"+code+"%").Scan(&refCnt)
		if refCnt > 0 {
			exception.New(fmt.Sprintf("问题 %s 正被 %d 条安全规则引用，请先调整关联安全规则后再删", code, refCnt), 400).Throw()
		}
		_, err = tx.Exec(`DELETE FROM questions WHERE question_id = ?`, code)
	case "safety_rules":
		_, _ = tx.Exec(`DELETE FROM edges WHERE target_type = 'RED_FLAG_RULE' AND target_code = ? AND relation_type = 'TRIGGERS_SAFETY'`, code)
		_, err = tx.Exec(`DELETE FROM safety_rules WHERE rule_id = ?`, code)
	case "symptoms":
		_, _ = tx.Exec(`DELETE FROM edges WHERE source_type = 'SYMPTOM' AND source_code = ?`, code)
		_, err = tx.Exec(`DELETE FROM symptoms WHERE concept_code = ?`, code)
	case "tcm_specialist_clinics":
		_, _ = tx.Exec(`DELETE FROM tcm_edges WHERE target_code = ? OR source_code = ?`, code, code)
		_, err = tx.Exec(`DELETE FROM tcm_specialist_clinics WHERE clinic_code = ?`, code)
	case "tcm_syndromes", "tcm_therapies":
		_, _ = tx.Exec(`DELETE FROM tcm_edges WHERE source_code = ? OR target_code = ?`, code, code)
		spec, sErr := buildEntityQuerySpec(db, entityType)
		if sErr != nil {
			exception.New(sErr.Error(), 400).Throw()
		}
		pkCol := primaryKeyColumn(entityType)
		_, err = tx.Exec(fmt.Sprintf(`DELETE FROM "%s" WHERE %s = ?`, spec.table, pkCol), code)
	default:
		spec, sErr := buildEntityQuerySpec(db, entityType)
		if sErr != nil {
			exception.New(sErr.Error(), 400).Throw()
		}
		pkCol := primaryKeyColumn(entityType)
		_, err = tx.Exec(fmt.Sprintf(`DELETE FROM "%s" WHERE %s = ?`, spec.table, pkCol), code)
	}
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	if err := tx.Commit(); err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	return map[string]interface{}{"success": true, "entity_type": entityType, "code": code}
}
