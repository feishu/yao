package triagekb

import (
	"archive/zip"
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
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

const (
	runtimeDirRel       = "data/triage/runtime"
	snapshotsDirRel     = "data/triage/runtime/snapshots"
	releasesFileRel     = "data/triage/runtime/releases.json"
	exportsDirRel       = "data/triage/runtime/exports"
	defaultGeneralDBRel = "data/triage/kb/kb_triage_general.db"
	defaultTCMDBRel     = "data/triage/kb/kb_triage_tcm.db"
	draftGeneralRel     = "data/triage/runtime/kb_triage_general.draft.db"
	draftTCMRel         = "data/triage/runtime/kb_triage_tcm.draft.db"
)

var adminMu sync.Mutex

// ReleaseRecord 知识库发布与回滚快照台账
type ReleaseRecord struct {
	ReleaseID     string                   `json:"release_id"`
	Version       string                   `json:"version"`
	Action        string                   `json:"action"` // PUBLISH / ROLLBACK / RUNTIME_TUNING
	Operator      string                   `json:"operator,omitempty"`
	Note          string                   `json:"note,omitempty"`
	CreatedAt     string                   `json:"created_at"`
	GeneralDBFile string                   `json:"general_db_file"`
	GeneralSHA256 string                   `json:"general_sha256"`
	TCMDBFile     string                   `json:"tcm_db_file"`
	TCMSHA256     string                   `json:"tcm_sha256"`
	ManifestHash  string                   `json:"manifest_hash"`
	Counts        map[string]int           `json:"counts,omitempty"`
	Runtime       map[string]interface{}   `json:"runtime,omitempty"`
	Layers        []LayerConfig            `json:"layers,omitempty"`
}

// RegisterAdminProcesses 注册工作区 B 平台知识库直管与发布回滚 Process
func RegisterAdminProcesses() {
	process.Register("triage.kb.status", ProcessKBStatus)
	process.Register("triage.kb.init_draft", ProcessKBInitDraft)
	process.Register("triage.kb.list_entities", ProcessKBListEntities)
	process.Register("triage.kb.get_entity", ProcessKBGetEntity)
	process.Register("triage.kb.save_entity", ProcessKBSaveEntity)
	process.Register("triage.kb.delete_entity", ProcessKBDeleteEntity)
	process.Register("triage.kb.validate", ProcessKBValidate)
	process.Register("triage.kb.publish", ProcessKBPublish)
	process.Register("triage.kb.rollback", ProcessKBRollback)
	process.Register("triage.kb.releases", ProcessKBReleases)
	process.Register("triage.kb.get_runtime_config", ProcessKBGetRuntimeConfig)
	process.Register("triage.kb.update_runtime_params", ProcessKBUpdateRuntimeParams)
	process.Register("triage.kb.export_seed", ProcessKBExportSeed)
	process.Register("triage.kb.playbooks.list", ProcessKBPlaybooksList)
	process.Register("triage.kb.playbooks.get", ProcessKBPlaybooksGet)
	process.Register("triage.kb.playbooks.save", ProcessKBPlaybooksSave)
	process.Register("triage.kb.playbooks.batch_disposition", ProcessKBPlaybooksBatchDisposition)
	process.Register("triage.kb.playbooks.import", ProcessKBPlaybooksImport)
	process.Register("triage.kb.question_packs.list", ProcessKBQuestionPacksList)
	process.Register("triage.kb.question_packs.save", ProcessKBQuestionPacksSave)
	process.Register("triage.kb.sandbox_run", ProcessKBSandboxRun)
	process.Register("triage.kb.sandbox_golden_diff", ProcessKBSandboxGoldenDiff)
}

func resolveAdminAppRoot() string {
	if root := GetEngine().GetAppRoot(); root != "" {
		return root
	}
	if application.App != nil {
		if root := application.App.Root(); root != "" {
			return root
		}
	}
	cwd, _ := os.Getwd()
	return cwd
}

func resolveDraftDBPaths(appRoot string) (string, string) {
	genPath := resolveConnectorFile("triage_general_draft", appRoot, filepath.Join(appRoot, "kbs"))
	if genPath == "" {
		genPath = draftGeneralRel
	}
	tcmPath := resolveConnectorFile("triage_tcm_draft", appRoot, filepath.Join(appRoot, "kbs"))
	if tcmPath == "" {
		tcmPath = draftTCMRel
	}
	if !filepath.IsAbs(genPath) {
		genPath = filepath.Join(appRoot, genPath)
	}
	if !filepath.IsAbs(tcmPath) {
		tcmPath = filepath.Join(appRoot, tcmPath)
	}
	return genPath, tcmPath
}

func loadEffectiveDSL(appRoot string) (FederatedGraphDSL, string, error) {
	effectivePath := ResolveEffectiveDSLPath(appRoot, "")
	raw, err := os.ReadFile(effectivePath)
	if err != nil {
		return FederatedGraphDSL{}, effectivePath, err
	}
	var dsl FederatedGraphDSL
	if err := json.Unmarshal(raw, &dsl); err != nil {
		return FederatedGraphDSL{}, effectivePath, err
	}
	return dsl, effectivePath, nil
}

func resolveActiveDBPaths(appRoot string) (string, string) {
	gen := filepath.Join(appRoot, defaultGeneralDBRel)
	tcm := filepath.Join(appRoot, defaultTCMDBRel)
	if dsl, dslPath, err := loadEffectiveDSL(appRoot); err == nil {
		for _, layer := range dsl.Layers {
			resolved := resolveLayerDBPath(layer, appRoot, filepath.Dir(dslPath))
			if resolved == "" {
				continue
			}
			if _, statErr := os.Stat(resolved); statErr != nil {
				continue
			}
			if strings.EqualFold(layer.Type, "BASE") {
				gen = resolved
			} else if strings.EqualFold(layer.Type, "OVERLAY") {
				tcm = resolved
			}
		}
	}
	return gen, tcm
}

func openSQLiteRW(dbPath string) (*sql.DB, error) {
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(absPath); err != nil {
		return nil, fmt.Errorf("SQLite 草稿库不存在: %s (%w)", absPath, err)
	}
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL", absPath))
	if err != nil {
		return nil, err
	}
	return db, nil
}

func openSQLiteRO(dbPath string) (*sql.DB, error) {
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(absPath); err != nil {
		return nil, fmt.Errorf("SQLite 数据库不存在: %s (%w)", absPath, err)
	}
	return sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=ro&cache=shared", absPath))
}

func writeFileAtomic(targetPath string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}
	tmpPath := fmt.Sprintf("%s.tmp.%d", targetPath, time.Now().UnixNano())
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, targetPath)
}

func copyFileSafe(srcPath, dstPath string) error {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return err
	}
	in, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer in.Close()

	tmpPath := fmt.Sprintf("%s.tmp.%d", dstPath, time.Now().UnixNano())
	out, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, dstPath)
}

func ensureDraftSchemaAndSeeds(appRoot, genDraftPath, tcmDraftPath string) error {
	genDB, err := openSQLiteRW(genDraftPath)
	if err != nil {
		return err
	}
	defer genDB.Close()

	cols := sqliteTableColumns(genDB, "symptoms")
	addedNeedsReview := false
	if !cols["needs_review"] {
		if _, err := genDB.Exec(`ALTER TABLE symptoms ADD COLUMN needs_review INTEGER NOT NULL DEFAULT 0`); err != nil {
			return fmt.Errorf("升级 symptoms.needs_review 列失败: %w", err)
		}
		addedNeedsReview = true
	}
	addedComplications := false
	if !cols["associated_complications_json"] {
		if _, err := genDB.Exec(`ALTER TABLE symptoms ADD COLUMN associated_complications_json TEXT NOT NULL DEFAULT '[]'`); err != nil {
			return fmt.Errorf("升级 symptoms.associated_complications_json 列失败: %w", err)
		}
		addedComplications = true
	}

	// 如从旧版基座初始化且缺失 needs_review / associated_complications，则按 seed/symptoms.json 同步补全
	if addedNeedsReview || addedComplications {
		syncSymptomExtrasFromSeed(appRoot, genDB)
	}

	// 创建扩展平台词典表
	dictDDLs := []string{
		`CREATE TABLE IF NOT EXISTS kb_hot_symptoms (
			symptom_code VARCHAR(64) PRIMARY KEY,
			label VARCHAR(128) NOT NULL,
			icon VARCHAR(64) NOT NULL DEFAULT 'stethoscope',
			body_part VARCHAR(64) NOT NULL DEFAULT 'GENERAL',
			sort_order INTEGER NOT NULL DEFAULT 100,
			is_active INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS kb_body_parts (
			part_code VARCHAR(64) PRIMARY KEY,
			label VARCHAR(64) NOT NULL,
			symptom_codes_json TEXT NOT NULL DEFAULT '[]',
			sort_order INTEGER NOT NULL DEFAULT 100,
			is_active INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS kb_keyword_synonyms (
			term VARCHAR(128) PRIMARY KEY,
			synonyms_json TEXT NOT NULL DEFAULT '[]'
		)`,
		`CREATE TABLE IF NOT EXISTS kb_fallback_keywords (
			service_code VARCHAR(64) PRIMARY KEY,
			keywords_json TEXT NOT NULL DEFAULT '[]'
		)`,
	}
	for _, ddl := range dictDDLs {
		if _, err := genDB.Exec(ddl); err != nil {
			return fmt.Errorf("初始化词典表失败: %w", err)
		}
	}
	seedDictionariesIfEmpty(appRoot, genDB)

	tcmDB, err := openSQLiteRW(tcmDraftPath)
	if err != nil {
		return err
	}
	defer tcmDB.Close()

	tcmCols := sqliteTableColumns(tcmDB, "tcm_specialist_clinics")
	if !tcmCols["target_dept_keywords_json"] {
		if _, err := tcmDB.Exec(`ALTER TABLE tcm_specialist_clinics ADD COLUMN target_dept_keywords_json TEXT NOT NULL DEFAULT '[]'`); err != nil {
			return fmt.Errorf("升级 tcm_specialist_clinics.target_dept_keywords_json 列失败: %w", err)
		}
		_, _ = tcmDB.Exec(`UPDATE tcm_specialist_clinics SET target_dept_keywords_json = json_array(dept_name) WHERE (target_dept_keywords_json IS NULL OR target_dept_keywords_json = '[]') AND dept_name != ''`)
	}

	if err := EnsurePlaybookSchema(genDB, tcmDB); err != nil {
		return fmt.Errorf("初始化分诊剧本表结构失败: %w", err)
	}
	var cardCount int
	_ = genDB.QueryRow(`SELECT COUNT(1) FROM playbook_cards`).Scan(&cardCount)
	if cardCount == 0 {
		if _, err := BootstrapPlaybooksFromLegacy(genDB, tcmDB); err != nil {
			return fmt.Errorf("冷启动逆向初始化分诊剧本卡失败: %w", err)
		}
	}
	return nil
}

func syncSymptomExtrasFromSeed(appRoot string, genDB *sql.DB) {
	seedPath := filepath.Join(appRoot, "data/triage/seed/symptoms.json")
	raw, err := os.ReadFile(seedPath)
	if err != nil {
		return
	}
	var parsed struct {
		Symptoms []struct {
			ConceptCode             string                       `json:"concept_code"`
			NeedsReview             bool                         `json:"needs_review"`
			AssociatedComplications []AssociatedComplicationItem `json:"associated_complications"`
		} `json:"symptoms"`
	}
	if json.Unmarshal(raw, &parsed) != nil {
		return
	}
	tx, err := genDB.Begin()
	if err != nil {
		return
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.Prepare(`UPDATE symptoms SET needs_review = ?, associated_complications_json = ? WHERE concept_code = ?`)
	if err != nil {
		return
	}
	defer stmt.Close()
	for _, s := range parsed.Symptoms {
		compBytes, _ := json.Marshal(s.AssociatedComplications)
		if len(compBytes) == 0 || string(compBytes) == "null" {
			compBytes = []byte("[]")
		}
		nr := 0
		if s.NeedsReview {
			nr = 1
		}
		_, _ = stmt.Exec(nr, string(compBytes), s.ConceptCode)
	}
	_ = tx.Commit()
}

func seedDictionariesIfEmpty(appRoot string, genDB *sql.DB) {
	var hotCount int
	_ = genDB.QueryRow(`SELECT COUNT(1) FROM kb_hot_symptoms`).Scan(&hotCount)
	var synCount int
	_ = genDB.QueryRow(`SELECT COUNT(1) FROM kb_keyword_synonyms`).Scan(&synCount)
	if hotCount > 0 && synCount > 0 {
		return
	}

	seedPath := filepath.Join(appRoot, "data/triage/seed/dictionaries_seed.json")
	raw, err := os.ReadFile(seedPath)
	if err != nil {
		return
	}
	var dictSeed struct {
		HotSymptoms      []HotSymptomItem    `json:"hot_symptoms"`
		BodyParts        []BodyPartItem      `json:"body_parts"`
		KeywordSynonyms  map[string][]string `json:"keyword_synonyms"`
		FallbackKeywords map[string][]string `json:"fallback_keywords"`
	}
	if json.Unmarshal(raw, &dictSeed) != nil {
		return
	}
	tx, err := genDB.Begin()
	if err != nil {
		return
	}
	defer func() { _ = tx.Rollback() }()

	if hotCount == 0 {
		for _, h := range dictSeed.HotSymptoms {
			active := 1
			if !h.IsActive {
				active = 0
			}
			_, _ = tx.Exec(`INSERT OR IGNORE INTO kb_hot_symptoms (symptom_code, label, icon, body_part, sort_order, is_active) VALUES (?, ?, ?, ?, ?, ?)`,
				h.SymptomCode, h.Label, h.Icon, h.BodyPart, h.SortOrder, active)
		}
	}

	var bpCount int
	_ = tx.QueryRow(`SELECT COUNT(1) FROM kb_body_parts`).Scan(&bpCount)
	if bpCount == 0 {
		for _, bp := range dictSeed.BodyParts {
			active := 1
			if !bp.IsActive {
				active = 0
			}
			codesJSON := mustMarshalJSON(bp.SymptomCodes, "[]")
			_, _ = tx.Exec(`INSERT OR IGNORE INTO kb_body_parts (part_code, label, symptom_codes_json, sort_order, is_active) VALUES (?, ?, ?, ?, ?)`,
				bp.PartCode, bp.Label, codesJSON, bp.SortOrder, active)
		}
	}

	if synCount == 0 {
		for term, list := range dictSeed.KeywordSynonyms {
			_, _ = tx.Exec(`INSERT OR IGNORE INTO kb_keyword_synonyms (term, synonyms_json) VALUES (?, ?)`,
				term, mustMarshalJSON(list, "[]"))
		}
	}

	var fbCount int
	_ = tx.QueryRow(`SELECT COUNT(1) FROM kb_fallback_keywords`).Scan(&fbCount)
	if fbCount == 0 {
		for svcCode, list := range dictSeed.FallbackKeywords {
			_, _ = tx.Exec(`INSERT OR IGNORE INTO kb_fallback_keywords (service_code, keywords_json) VALUES (?, ?)`,
				svcCode, mustMarshalJSON(list, "[]"))
		}
	}

	_ = tx.Commit()
}

func ensureDraftInitializedLocked(appRoot string, forceFromActive bool) (string, string, error) {
	genDraft, tcmDraft := resolveDraftDBPaths(appRoot)
	activeGen, activeTCM := resolveActiveDBPaths(appRoot)

	if forceFromActive {
		_ = copyFileSafe(activeGen, genDraft)
		_ = copyFileSafe(activeTCM, tcmDraft)
		_ = os.Remove(genDraft + "-wal")
		_ = os.Remove(genDraft + "-shm")
		_ = os.Remove(tcmDraft + "-wal")
		_ = os.Remove(tcmDraft + "-shm")
	} else {
		if _, err := os.Stat(genDraft); os.IsNotExist(err) {
			if err := copyFileSafe(activeGen, genDraft); err != nil {
				return "", "", fmt.Errorf("初始化全科草稿库失败: %w", err)
			}
		}
		if _, err := os.Stat(tcmDraft); os.IsNotExist(err) {
			if err := copyFileSafe(activeTCM, tcmDraft); err != nil {
				return "", "", fmt.Errorf("初始化中医草稿库失败: %w", err)
			}
		}
	}

	if err := ensureDraftSchemaAndSeeds(appRoot, genDraft, tcmDraft); err != nil {
		return "", "", err
	}
	return genDraft, tcmDraft, nil
}

func loadReleaseHistory(appRoot string) []ReleaseRecord {
	path := filepath.Join(appRoot, releasesFileRel)
	raw, err := os.ReadFile(path)
	if err != nil {
		return []ReleaseRecord{}
	}
	var list []ReleaseRecord
	if json.Unmarshal(raw, &list) != nil {
		return []ReleaseRecord{}
	}
	return list
}

func saveReleaseHistory(appRoot string, list []ReleaseRecord) error {
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(filepath.Join(appRoot, releasesFileRel), data)
}

func queryTableCounts(dbPath string, tables []string) map[string]int {
	counts := make(map[string]int, len(tables))
	db, err := openSQLiteRO(dbPath)
	if err != nil {
		return counts
	}
	defer db.Close()
	for _, tbl := range tables {
		if !sqliteHasTable(db, tbl) {
			continue
		}
		var cnt int
		if db.QueryRow(fmt.Sprintf(`SELECT COUNT(1) FROM "%s"`, tbl)).Scan(&cnt) == nil {
			counts[tbl] = cnt
		}
	}
	return counts
}

// ProcessKBInitDraft 初始化或从当前激活版本重置 Staging .draft.db
func ProcessKBInitDraft(proc *process.Process) interface{} {
	force := false
	if proc.NumOfArgs() > 0 {
		if m, ok := proc.Args[0].(map[string]interface{}); ok {
			if f, ok := m["force"].(bool); ok {
				force = f
			}
		} else if b, ok := proc.Args[0].(bool); ok {
			force = b
		}
	}
	appRoot := resolveAdminAppRoot()
	adminMu.Lock()
	defer adminMu.Unlock()

	genDraft, tcmDraft, err := ensureDraftInitializedLocked(appRoot, force)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	return map[string]interface{}{
		"success":          true,
		"general_draft_db": genDraft,
		"tcm_draft_db":     tcmDraft,
		"forced":           force,
		"counts":           collectDraftCounts(genDraft, tcmDraft),
	}
}

func collectDraftCounts(genDraft, tcmDraft string) map[string]int {
	merged := queryTableCounts(genDraft, []string{
		"standard_services", "symptoms", "questions", "safety_rules",
		"diseases", "textbook_evidence", "edges",
		"kb_hot_symptoms", "kb_body_parts", "kb_keyword_synonyms", "kb_fallback_keywords",
	})
	for k, v := range queryTableCounts(tcmDraft, []string{
		"tcm_syndromes", "tcm_diseases", "tcm_specialist_clinics", "tcm_therapies", "tcm_edges",
	}) {
		merged[k] = v
	}
	if db, err := openSQLiteRO(genDraft); err == nil {
		defer db.Close()
		cols := sqliteTableColumns(db, "symptoms")
		if cols["needs_review"] {
			var nr int
			if db.QueryRow(`SELECT COUNT(1) FROM symptoms WHERE needs_review = 1`).Scan(&nr) == nil {
				merged["symptoms_needs_review"] = nr
			}
		}
	}
	return merged
}

// ProcessKBStatus 返回运行态、草稿库状态及发布台账概况
func ProcessKBStatus(proc *process.Process) interface{} {
	appRoot := resolveAdminAppRoot()
	adminMu.Lock()
	defer adminMu.Unlock()

	genDraft, tcmDraft := resolveDraftDBPaths(appRoot)
	_, genExists := os.Stat(genDraft)
	_, tcmExists := os.Stat(tcmDraft)
	runtimeManifest := filepath.Join(appRoot, RuntimeManifestRelPath)
	_, hasRuntimeOverride := os.Stat(runtimeManifest)
	dsl, dslPath, _ := loadEffectiveDSL(appRoot)

	snap := GetEngine().GetSnapshot()
	engineStatus := map[string]interface{}{"loaded": snap != nil}
	if snap != nil {
		engineStatus["version"] = snap.Version
		engineStatus["active_version"] = snap.ActiveVersion
		engineStatus["manifest_hash"] = snap.ManifestHash
		engineStatus["dsl_path"] = snap.DSLPath
		engineStatus["layer_statuses"] = snap.LayerStatuses
		engineStatus["services_count"] = len(snap.Services)
		engineStatus["symptoms_count"] = len(snap.Symptoms)
		engineStatus["questions_count"] = len(snap.Questions)
		engineStatus["rules_count"] = len(snap.SafetyRules)
		engineStatus["diseases_count"] = len(snap.Diseases)
		engineStatus["tcm_clinics"] = len(snap.TCMClinics)
	}

	var draftCounts map[string]int
	var genDraftSHA, tcmDraftSHA string
	if genExists == nil && tcmExists == nil {
		_ = ensureDraftSchemaAndSeeds(appRoot, genDraft, tcmDraft)
		draftCounts = collectDraftCounts(genDraft, tcmDraft)
		genDraftSHA, _ = fileSHA256(genDraft)
		tcmDraftSHA, _ = fileSHA256(tcmDraft)
	}

	releases := loadReleaseHistory(appRoot)
	return map[string]interface{}{
		"draft_initialized":    genExists == nil && tcmExists == nil,
		"general_draft_db":     genDraft,
		"tcm_draft_db":         tcmDraft,
		"general_draft_sha256": genDraftSHA,
		"tcm_draft_sha256":     tcmDraftSHA,
		"draft_counts":         draftCounts,
		"has_runtime_override": hasRuntimeOverride == nil,
		"effective_dsl_path":   dslPath,
		"dsl_config":           dsl,
		"engine_status":        engineStatus,
		"releases_count":       len(releases),
	}
}

// ProcessKBPublish 门禁校验后 VACUUM INTO 生成不可变版本快照并原子切换引擎
func ProcessKBPublish(proc *process.Process) interface{} {
	params := map[string]interface{}{}
	if proc.NumOfArgs() > 0 {
		if m, ok := proc.Args[0].(map[string]interface{}); ok {
			params = m
		}
	}

	appRoot := resolveAdminAppRoot()
	adminMu.Lock()
	defer adminMu.Unlock()

	genDraft, tcmDraft, err := ensureDraftInitializedLocked(appRoot, false)
	if err != nil {
		exception.New(err.Error(), 500).Throw()
	}

	report, err := validateDatabases(genDraft, tcmDraft)
	if err != nil {
		exception.New(fmt.Sprintf("执行发布门禁校验失败: %v", err), 500).Throw()
	}
	if !report.Pass {
		exception.New(fmt.Sprintf("发布门禁校验未通过，存在 %d 项阻断性错误: %s",
			len(report.BlockingErrors), strings.Join(report.BlockingErrors, "; ")), 400).Throw()
	}

	releases := loadReleaseHistory(appRoot)
	seq := len(releases) + 1
	now := time.Now()
	releaseID := fmt.Sprintf("rel_%s_%03d", now.Format("20060102_150405"), seq)
	versionLabel, _ := params["version_label"].(string)
	versionLabel = strings.TrimSpace(versionLabel)
	if versionLabel == "" {
		versionLabel = fmt.Sprintf("v2.2.%d", seq)
	}
	operator, _ := params["operator"].(string)
	note, _ := params["note"].(string)

	snapDir := filepath.Join(appRoot, snapshotsDirRel)
	if err := os.MkdirAll(snapDir, 0755); err != nil {
		exception.New(fmt.Sprintf("创建快照目录失败: %v", err), 500).Throw()
	}

	genSnapRel := fmt.Sprintf("./%s/kb_triage_general_%s.db", snapshotsDirRel, releaseID)
	tcmSnapRel := fmt.Sprintf("./%s/kb_triage_tcm_%s.db", snapshotsDirRel, releaseID)
	genSnapAbs := filepath.Join(appRoot, strings.TrimPrefix(genSnapRel, "./"))
	tcmSnapAbs := filepath.Join(appRoot, strings.TrimPrefix(tcmSnapRel, "./"))
	_ = os.Remove(genSnapAbs)
	_ = os.Remove(tcmSnapAbs)

	if err := vacuumIntoSnapshot(genDraft, genSnapAbs); err != nil {
		exception.New(fmt.Sprintf("生成全科快照失败: %v", err), 500).Throw()
	}
	if err := vacuumIntoSnapshot(tcmDraft, tcmSnapAbs); err != nil {
		exception.New(fmt.Sprintf("生成中医快照失败: %v", err), 500).Throw()
	}

	genSHA, err := fileSHA256(genSnapAbs)
	if err != nil {
		exception.New(fmt.Sprintf("计算全科快照摘要失败: %v", err), 500).Throw()
	}
	tcmSHA, err := fileSHA256(tcmSnapAbs)
	if err != nil {
		exception.New(fmt.Sprintf("计算中医快照摘要失败: %v", err), 500).Throw()
	}

	dsl, _, dslErr := loadEffectiveDSL(appRoot)
	if dslErr != nil {
		dsl = defaultDSLTemplate()
	}
	dsl.Version = strings.TrimPrefix(versionLabel, "v")
	dsl.ActiveVersion = versionLabel
	dsl.Storage = "federated_sqlite_graph"

	if rtOverride, ok := params["runtime"].(map[string]interface{}); ok && len(rtOverride) > 0 {
		dsl.Runtime = mergeRuntimeConfig(dsl.Runtime, rtOverride)
	}
	if rawLayers, ok := params["layers"]; ok {
		dsl.Layers = mergeLayerConfigs(dsl.Layers, rawLayers)
	}
	for i := range dsl.Layers {
		if strings.EqualFold(dsl.Layers[i].Type, "BASE") {
			dsl.Layers[i].DBFile = genSnapRel
		} else if strings.EqualFold(dsl.Layers[i].Type, "OVERLAY") {
			dsl.Layers[i].DBFile = tcmSnapRel
		}
	}

	runtimeAbs := filepath.Join(appRoot, RuntimeManifestRelPath)
	dslBytes, _ := json.MarshalIndent(dsl, "", "  ")
	if err := writeFileAtomic(runtimeAbs, dslBytes); err != nil {
		exception.New(fmt.Sprintf("写入运行态持久化清单失败: %v", err), 500).Throw()
	}

	snap, loadErr := GetEngine().LoadFromDSLFile(runtimeAbs)
	if loadErr != nil {
		exception.New(fmt.Sprintf("快照热加载失败: %v", loadErr), 500).Throw()
	}

	record := ReleaseRecord{
		ReleaseID:     releaseID,
		Version:       versionLabel,
		Action:        "PUBLISH",
		Operator:      operator,
		Note:          note,
		CreatedAt:     now.Format(time.RFC3339),
		GeneralDBFile: genSnapRel,
		GeneralSHA256: genSHA,
		TCMDBFile:     tcmSnapRel,
		TCMSHA256:     tcmSHA,
		ManifestHash:  snap.ManifestHash,
		Counts:        report.Counts,
		Runtime:       dsl.Runtime,
		Layers:        dsl.Layers,
	}
	releases = append([]ReleaseRecord{record}, releases...)
	if err := saveReleaseHistory(appRoot, releases); err != nil {
		exception.New(fmt.Sprintf("写入发布台账失败: %v", err), 500).Throw()
	}

	return map[string]interface{}{
		"success":        true,
		"release":        record,
		"gate_report":    report,
		"active_version": snap.ActiveVersion,
		"manifest_hash":  snap.ManifestHash,
	}
}

func vacuumIntoSnapshot(srcPath, destPath string) error {
	db, err := openSQLiteRW(srcPath)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(`VACUUM INTO ?`, destPath)
	return err
}

// ProcessKBRollback 一键秒级回滚到指定历史发布版本快照
func ProcessKBRollback(proc *process.Process) interface{} {
	releaseID := ""
	operator := ""
	if proc.NumOfArgs() > 0 {
		switch v := proc.Args[0].(type) {
		case string:
			releaseID = v
		case map[string]interface{}:
			releaseID, _ = v["release_id"].(string)
			operator, _ = v["operator"].(string)
		}
	}
	releaseID = strings.TrimSpace(releaseID)
	if releaseID == "" {
		exception.New("缺少 release_id 参数", 400).Throw()
	}

	appRoot := resolveAdminAppRoot()
	adminMu.Lock()
	defer adminMu.Unlock()

	releases := loadReleaseHistory(appRoot)
	var target *ReleaseRecord
	for i := range releases {
		if releases[i].ReleaseID == releaseID {
			target = &releases[i]
			break
		}
	}
	if target == nil {
		exception.New(fmt.Sprintf("未找到发布版本 %s", releaseID), 404).Throw()
	}

	genAbs := filepath.Join(appRoot, strings.TrimPrefix(target.GeneralDBFile, "./"))
	tcmAbs := filepath.Join(appRoot, strings.TrimPrefix(target.TCMDBFile, "./"))
	genSHA, err := fileSHA256(genAbs)
	if err != nil {
		exception.New(fmt.Sprintf("目标全科快照文件不可读: %v", err), 404).Throw()
	}
	tcmSHA, err := fileSHA256(tcmAbs)
	if err != nil {
		exception.New(fmt.Sprintf("目标中医快照文件不可读: %v", err), 404).Throw()
	}
	if target.GeneralSHA256 != "" && genSHA != target.GeneralSHA256 {
		exception.New("全科快照 SHA-256 校验不匹配，拒绝回滚损坏文件", 409).Throw()
	}
	if target.TCMSHA256 != "" && tcmSHA != target.TCMSHA256 {
		exception.New("中医快照 SHA-256 校验不匹配，拒绝回滚损坏文件", 409).Throw()
	}

	dsl, _, err := loadEffectiveDSL(appRoot)
	if err != nil {
		dsl = defaultDSLTemplate()
	}
	dsl.Version = strings.TrimPrefix(target.Version, "v")
	dsl.ActiveVersion = target.Version
	if len(target.Runtime) > 0 {
		dsl.Runtime = target.Runtime
	}
	if len(target.Layers) > 0 {
		dsl.Layers = target.Layers
	}
	for i := range dsl.Layers {
		if strings.EqualFold(dsl.Layers[i].Type, "BASE") {
			dsl.Layers[i].DBFile = target.GeneralDBFile
		} else if strings.EqualFold(dsl.Layers[i].Type, "OVERLAY") {
			dsl.Layers[i].DBFile = target.TCMDBFile
		}
	}

	runtimeAbs := filepath.Join(appRoot, RuntimeManifestRelPath)
	dslBytes, _ := json.MarshalIndent(dsl, "", "  ")
	if err := writeFileAtomic(runtimeAbs, dslBytes); err != nil {
		exception.New(fmt.Sprintf("更新运行态清单失败: %v", err), 500).Throw()
	}

	snap, loadErr := GetEngine().LoadFromDSLFile(runtimeAbs)
	if loadErr != nil {
		exception.New(fmt.Sprintf("回滚热加载失败: %v", loadErr), 500).Throw()
	}

	rollbackEntry := ReleaseRecord{
		ReleaseID:     fmt.Sprintf("rb_%s_%03d", time.Now().Format("20060102_150405"), len(releases)+1),
		Version:       target.Version,
		Action:        "ROLLBACK",
		Operator:      operator,
		Note:          fmt.Sprintf("回滚至快照 %s (%s)", target.ReleaseID, target.Version),
		CreatedAt:     time.Now().Format(time.RFC3339),
		GeneralDBFile: target.GeneralDBFile,
		GeneralSHA256: genSHA,
		TCMDBFile:     target.TCMDBFile,
		TCMSHA256:     tcmSHA,
		ManifestHash:  snap.ManifestHash,
		Counts:        target.Counts,
		Runtime:       dsl.Runtime,
		Layers:        dsl.Layers,
	}
	releases = append([]ReleaseRecord{rollbackEntry}, releases...)
	_ = saveReleaseHistory(appRoot, releases)

	return map[string]interface{}{
		"success":        true,
		"rollback":       rollbackEntry,
		"active_version": snap.ActiveVersion,
		"manifest_hash":  snap.ManifestHash,
	}
}

// ProcessKBReleases 返回版本发布历史台账
func ProcessKBReleases(proc *process.Process) interface{} {
	appRoot := resolveAdminAppRoot()
	adminMu.Lock()
	defer adminMu.Unlock()

	snap := GetEngine().GetSnapshot()
	activeHash := ""
	activeVer := ""
	if snap != nil {
		activeHash = snap.ManifestHash
		activeVer = snap.ActiveVersion
	}
	return map[string]interface{}{
		"active_version": activeVer,
		"manifest_hash":  activeHash,
		"releases":       loadReleaseHistory(appRoot),
	}
}

// ProcessKBGetRuntimeConfig 返回当前生效的 kbs 运行时超参数与层列表
func ProcessKBGetRuntimeConfig(proc *process.Process) interface{} {
	appRoot := resolveAdminAppRoot()
	dsl, dslPath, err := loadEffectiveDSL(appRoot)
	if err != nil {
		exception.New(fmt.Sprintf("读取 kbs 清单失败: %v", err), 500).Throw()
	}
	return map[string]interface{}{
		"dsl_path":       dslPath,
		"version":        dsl.Version,
		"active_version": dsl.ActiveVersion,
		"layers":         dsl.Layers,
		"runtime":        dsl.Runtime,
		"pipeline_cfg":   GetEngine().GetConfig(),
	}
}

// ProcessKBUpdateRuntimeParams 调优 kbs/triage.kb.yao runtime 超参数并原子写入运行态覆盖清单生效
func ProcessKBUpdateRuntimeParams(proc *process.Process) interface{} {
	if proc.NumOfArgs() < 1 {
		exception.New("缺少 runtime 参数配置", 400).Throw()
	}
	payload, ok := proc.Args[0].(map[string]interface{})
	if !ok {
		exception.New("参数格式错误", 400).Throw()
	}

	appRoot := resolveAdminAppRoot()
	adminMu.Lock()
	defer adminMu.Unlock()

	dsl, _, err := loadEffectiveDSL(appRoot)
	if err != nil {
		dsl = defaultDSLTemplate()
	}

	if rtMap, ok := payload["runtime"].(map[string]interface{}); ok {
		if v, exists := rtMap["max_steps"]; exists {
			if n, ok := asInt(v); !ok || n < 1 || n > 15 {
				exception.New("max_steps 必须在 1~15 之间", 400).Throw()
			}
		}
		if v, exists := rtMap["confidence_threshold"]; exists {
			if f, ok := asFloat64(v); !ok || f <= 0 || f > 1 {
				exception.New("confidence_threshold 必须在 (0, 1] 之间", 400).Throw()
			}
		}
		if sp, ok := rtMap["spreading_activation"].(map[string]interface{}); ok {
			if v, exists := sp["decay"]; exists {
				if f, ok := asFloat64(v); !ok || f < 0 || f >= 1 {
					exception.New("spreading_activation.decay 必须在 [0, 1) 之间", 400).Throw()
				}
			}
			if v, exists := sp["max_distance"]; exists {
				if n, ok := asInt(v); !ok || n < 0 || n > 5 {
					exception.New("spreading_activation.max_distance 必须在 0~5 之间", 400).Throw()
				}
			}
			if v, exists := sp["threshold"]; exists {
				if f, ok := asFloat64(v); !ok || f < 0 || f > 5 {
					exception.New("spreading_activation.threshold 必须在 [0, 5] 之间", 400).Throw()
				}
			}
		}
		dsl.Runtime = mergeRuntimeConfig(dsl.Runtime, rtMap)
	}
	if rawLayers, ok := payload["layers"]; ok {
		dsl.Layers = mergeLayerConfigs(dsl.Layers, rawLayers)
	}
	// 严格校验超参数合法区间
	cfg := buildPipelineConfigFromRuntime(dsl.Runtime)
	if cfg.MaxSteps < 1 || cfg.MaxSteps > 15 {
		exception.New("max_steps 必须在 1~15 之间", 400).Throw()
	}
	if cfg.ConfidenceThreshold <= 0 || cfg.ConfidenceThreshold > 1 {
		exception.New("confidence_threshold 必须在 (0, 1] 之间", 400).Throw()
	}
	if cfg.Decay < 0 || cfg.Decay >= 1 {
		exception.New("spreading_activation.decay 必须在 [0, 1) 之间", 400).Throw()
	}
	if cfg.MaxDistance < 0 || cfg.MaxDistance > 5 {
		exception.New("spreading_activation.max_distance 必须在 0~5 之间", 400).Throw()
	}

	runtimeAbs := filepath.Join(appRoot, RuntimeManifestRelPath)
	dslBytes, _ := json.MarshalIndent(dsl, "", "  ")
	if err := writeFileAtomic(runtimeAbs, dslBytes); err != nil {
		exception.New(fmt.Sprintf("写入运行态清单失败: %v", err), 500).Throw()
	}

	snap, err := GetEngine().LoadFromDSLFile(runtimeAbs)
	if err != nil {
		exception.New(fmt.Sprintf("热重载运行参数失败: %v", err), 500).Throw()
	}

	genActive, tcmActive := resolveActiveDBPaths(appRoot)
	genSHA, _ := fileSHA256(genActive)
	tcmSHA, _ := fileSHA256(tcmActive)
	releases := loadReleaseHistory(appRoot)
	note, _ := payload["note"].(string)
	operator, _ := payload["operator"].(string)
	entry := ReleaseRecord{
		ReleaseID:     fmt.Sprintf("rt_%s_%03d", time.Now().Format("20060102_150405"), len(releases)+1),
		Version:       snap.ActiveVersion,
		Action:        "RUNTIME_TUNING",
		Operator:      operator,
		Note:          note,
		CreatedAt:     time.Now().Format(time.RFC3339),
		GeneralDBFile: toRelOrKeep(appRoot, genActive),
		GeneralSHA256: genSHA,
		TCMDBFile:     toRelOrKeep(appRoot, tcmActive),
		TCMSHA256:     tcmSHA,
		ManifestHash:  snap.ManifestHash,
		Runtime:       dsl.Runtime,
		Layers:        dsl.Layers,
	}
	releases = append([]ReleaseRecord{entry}, releases...)
	_ = saveReleaseHistory(appRoot, releases)

	return map[string]interface{}{
		"success":       true,
		"manifest_hash": snap.ManifestHash,
		"pipeline_cfg":  GetEngine().GetConfig(),
		"runtime":       dsl.Runtime,
		"layers":        dsl.Layers,
	}
}

func toRelOrKeep(appRoot, target string) string {
	if rel, err := filepath.Rel(appRoot, target); err == nil && !strings.HasPrefix(rel, "..") {
		return "./" + rel
	}
	return target
}

func defaultDSLTemplate() FederatedGraphDSL {
	return FederatedGraphDSL{
		Name:          "Clinical Triage Federated Knowledge Graph",
		Label:         "临床多源联邦知识图谱导诊库",
		Version:       "2.1.0",
		ActiveVersion: "v2.1.0",
		Storage:       "federated_sqlite_graph",
		Layers: []LayerConfig{
			{ID: "base_general", Name: "全科现代医学临床基座库", Connector: "connectors.triage_general", DBFile: "data/triage/kb/kb_triage_general.db", Type: "BASE", Priority: 100},
			{ID: "overlay_tcm", Name: "数智贵中医专病与辨证扩展库", Connector: "connectors.triage_tcm", DBFile: "data/triage/kb/kb_triage_tcm.db", Type: "OVERLAY", Priority: 200},
		},
		Runtime: map[string]interface{}{
			"engine":                  "FEDERATED_GRAPH",
			"max_steps":               5,
			"confidence_threshold":    0.85,
			"allow_early_convergence": true,
			"spreading_activation": map[string]interface{}{
				"decay":        0.2,
				"max_distance": 2,
				"threshold":    0.35,
			},
			"dual_track": map[string]interface{}{
				"enable_tcm":               true,
				"prefer_specialist_clinic": true,
			},
			"audit_persistence": true,
		},
	}
}

func mergeRuntimeConfig(base map[string]interface{}, patch map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range base {
		out[k] = v
	}
	for k, v := range patch {
		if nestedPatch, ok := v.(map[string]interface{}); ok {
			if nestedBase, ok2 := out[k].(map[string]interface{}); ok2 {
				merged := make(map[string]interface{})
				for bk, bv := range nestedBase {
					merged[bk] = bv
				}
				for pk, pv := range nestedPatch {
					merged[pk] = pv
				}
				out[k] = merged
				continue
			}
		}
		out[k] = v
	}
	return out
}

func mergeLayerConfigs(existing []LayerConfig, rawPatch interface{}) []LayerConfig {
	bytes, err := json.Marshal(rawPatch)
	if err != nil {
		return existing
	}
	var patches []LayerConfig
	if json.Unmarshal(bytes, &patches) != nil || len(patches) == 0 {
		return existing
	}
	patchByID := make(map[string]LayerConfig, len(patches))
	for _, p := range patches {
		patchByID[p.ID] = p
	}
	out := make([]LayerConfig, 0, len(existing))
	for _, l := range existing {
		if p, ok := patchByID[l.ID]; ok {
			if p.Priority != 0 {
				l.Priority = p.Priority
			}
			if p.Enabled != nil {
				l.Enabled = p.Enabled
			}
			if p.Name != "" {
				l.Name = p.Name
			}
		}
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Priority < out[j].Priority
	})
	return out
}

// ProcessKBExportSeed 导出全量标准化 seed/*.json 与 .db 压缩包供代码库归档回流
func ProcessKBExportSeed(proc *process.Process) interface{} {
	source := "draft"
	if proc.NumOfArgs() > 0 {
		if m, ok := proc.Args[0].(map[string]interface{}); ok {
			if s, ok := m["source"].(string); ok && s != "" {
				source = s
			}
		}
	}

	appRoot := resolveAdminAppRoot()
	adminMu.Lock()
	defer adminMu.Unlock()

	var genPath, tcmPath string
	var err error
	if source == "active" {
		genPath, tcmPath = resolveActiveDBPaths(appRoot)
	} else {
		genPath, tcmPath, err = ensureDraftInitializedLocked(appRoot, false)
		if err != nil {
			exception.New(err.Error(), 500).Throw()
		}
	}

	seedFiles, err := exportSeedPayloads(genPath, tcmPath)
	if err != nil {
		exception.New(fmt.Sprintf("导出 seed JSON 失败: %v", err), 500).Throw()
	}

	exportDir := filepath.Join(appRoot, exportsDirRel)
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		exception.New(err.Error(), 500).Throw()
	}
	zipName := fmt.Sprintf("triage_kb_bundle_%s.zip", time.Now().Format("20060102_150405"))
	zipPath := filepath.Join(exportDir, zipName)
	if err := writeSeedAndDBZip(zipPath, seedFiles, genPath, tcmPath); err != nil {
		exception.New(fmt.Sprintf("生成导出压缩包失败: %v", err), 500).Throw()
	}
	zipSHA, _ := fileSHA256(zipPath)

	return map[string]interface{}{
		"success":    true,
		"source":     source,
		"zip_path":   toRelOrKeep(appRoot, zipPath),
		"zip_name":   zipName,
		"zip_sha256": zipSHA,
		"files":      sortedMapKeys(seedFiles),
	}
}

func sortedMapKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func writeSeedAndDBZip(zipPath string, seedFiles map[string][]byte, genPath, tcmPath string) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	for _, name := range sortedMapKeys(seedFiles) {
		w, err := zw.Create("seed/" + name)
		if err != nil {
			return err
		}
		if _, err := w.Write(seedFiles[name]); err != nil {
			return err
		}
	}
	for _, item := range []struct {
		zipEntry string
		diskPath string
	}{
		{"kb/kb_triage_general.db", genPath},
		{"kb/kb_triage_tcm.db", tcmPath},
	} {
		w, err := zw.Create(item.zipEntry)
		if err != nil {
			return err
		}
		src, err := os.Open(item.diskPath)
		if err != nil {
			return err
		}
		if _, err := io.Copy(w, src); err != nil {
			_ = src.Close()
			return err
		}
		_ = src.Close()
	}
	return nil
}

func exportSeedPayloads(genPath, tcmPath string) (map[string][]byte, error) {
	files := make(map[string][]byte)
	services, err := listAllRowsFromDB(genPath, "services")
	if err != nil {
		return nil, err
	}
	symptoms, err := listAllRowsFromDB(genPath, "symptoms")
	if err != nil {
		return nil, err
	}
	questions, err := listAllRowsFromDB(genPath, "questions")
	if err != nil {
		return nil, err
	}
	rules, err := listAllRowsFromDB(genPath, "safety_rules")
	if err != nil {
		return nil, err
	}
	evidences, err := listAllRowsFromDB(genPath, "textbook_evidence")
	if err != nil {
		return nil, err
	}
	hotSymptoms, _ := listAllRowsFromDB(genPath, "hot_symptoms")
	bodyParts, _ := listAllRowsFromDB(genPath, "body_parts")
	synonymsList, _ := listAllRowsFromDB(genPath, "keyword_synonyms")
	fallbacksList, _ := listAllRowsFromDB(genPath, "fallback_keywords")

	tcmSyndromes, _ := listAllRowsFromDB(tcmPath, "tcm_syndromes")
	tcmDiseases, _ := listAllRowsFromDB(tcmPath, "tcm_diseases")
	tcmClinics, _ := listAllRowsFromDB(tcmPath, "tcm_specialist_clinics")
	tcmTherapies, _ := listAllRowsFromDB(tcmPath, "tcm_therapies")
	tcmEdges, _ := listAllRowsFromDB(tcmPath, "tcm_edges")

	evidBySym := make(map[string][]map[string]interface{})
	for _, row := range evidences {
		symCode, _ := row["symptom_code"].(string)
		evidBySym[symCode] = append(evidBySym[symCode], row)
	}

	synMap := make(map[string]interface{})
	for _, row := range synonymsList {
		term, _ := row["term"].(string)
		synMap[term] = row["synonyms"]
	}
	fbMap := make(map[string]interface{})
	for _, row := range fallbacksList {
		svcCode, _ := row["service_code"].(string)
		fbMap[svcCode] = row["keywords"]
	}

	files["standard_services.json"], _ = json.MarshalIndent(services, "", "  ")
	files["symptoms.json"], _ = json.MarshalIndent(map[string]interface{}{
		"version":  "2.2.0",
		"symptoms": symptoms,
	}, "", "  ")
	files["question_tree.json"], _ = json.MarshalIndent(questions, "", "  ")
	files["safety_rules.json"], _ = json.MarshalIndent(rules, "", "  ")
	files["textbook_evidence.json"], _ = json.MarshalIndent(evidBySym, "", "  ")
	files["tcm_seed.json"], _ = json.MarshalIndent(map[string]interface{}{
		"tcm_syndromes":          tcmSyndromes,
		"tcm_diseases":           tcmDiseases,
		"tcm_specialist_clinics": tcmClinics,
		"tcm_therapies":          tcmTherapies,
		"tcm_edges":              tcmEdges,
	}, "", "  ")
	files["dictionaries_seed.json"], _ = json.MarshalIndent(map[string]interface{}{
		"hot_symptoms":      hotSymptoms,
		"body_parts":        bodyParts,
		"keyword_synonyms":  synMap,
		"fallback_keywords": fbMap,
	}, "", "  ")

	checksums := make(map[string]string, len(files))
	for name, content := range files {
		sum := sha256.Sum256(content)
		checksums[name] = "sha256:" + hex.EncodeToString(sum[:])
	}
	manifest := map[string]interface{}{
		"version":     "2.2.0",
		"exported_at": time.Now().Format(time.RFC3339),
		"statistics": map[string]int{
			"standard_services_count": len(services),
			"symptoms_count":          len(symptoms),
			"questions_count":         len(questions),
			"safety_rules_count":      len(rules),
			"textbook_evidence_count": len(evidences),
			"tcm_clinics_count":       len(tcmClinics),
		},
		"checksums": checksums,
	}
	files["manifest.json"], _ = json.MarshalIndent(manifest, "", "  ")
	return files, nil
}
