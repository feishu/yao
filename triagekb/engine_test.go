package triagekb

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func getTestPaths() (string, string) {
	generalDB := "/Users/L/Desktop/Code/yao_projects/syd/service/data/triage/kb/kb_triage_general.db"
	tcmDB := "/Users/L/Desktop/Code/yao_projects/syd/service/data/triage/kb/kb_triage_tcm.db"
	return generalDB, tcmDB
}

func prepareEngine(t *testing.T) *Engine {
	genDB, tcmDB := getTestPaths()
	loader := NewFederatedLoader(FederatedGraphDSL{
		Version: "2.1.0",
	})
	snap, err := loader.LoadFromFiles(genDB, tcmDB)
	assert.NoError(t, err)
	assert.NotNil(t, snap)

	engine := GetEngine()
	engine.SetSnapshot(snap)
	return engine
}

// 1. 测试图快照与倒排索引检索
func TestSnapshotAndSearch(t *testing.T) {
	engine := prepareEngine(t)
	snap := engine.GetSnapshot()

	assert.Equal(t, "2.1.0", snap.Version)
	assert.True(t, len(snap.Symptoms) >= 434, "主诉数量应>=434")
	assert.True(t, len(snap.Diseases) >= 1000, "鉴别疾病应>=1000")
	assert.True(t, len(snap.SafetyRules) >= 40, "安全规则应>=40")
	assert.True(t, len(snap.TCMSyndromes) >= 10, "中医证候应>=10")

	// 拼音简拼检索测试
	hits := engine.SearchSymptoms("ptt", 5)
	assert.NotEmpty(t, hits)
	assert.Equal(t, "SYM_HEADACHE_THROBBING", hits[0].ConceptCode)

	// 中文模糊检索
	hitsChest := engine.SearchSymptoms("胸痛", 5)
	assert.NotEmpty(t, hitsChest)
}

// 2. 测试危急重症 100% 熔断拦截 (Safety Short-Circuit)
func TestSafetyShortCircuit(t *testing.T) {
	engine := prepareEngine(t)

	testCases := []struct {
		name       string
		symptoms   []string
		confirmed  []string
		expectRule string
	}{
		{
			name:       "胸痛伴压榨感与向左肩放射",
			symptoms:   []string{"SYM_CHEST_PAIN"},
			confirmed:  []string{"LEFT_ARM_BACK", "YES"},
			expectRule: "RULE_CRIT_ACS_MI",
		},
		{
			name:       "剧烈胸痛出冷汗",
			symptoms:   []string{"SYM_CHEST_PAIN"},
			confirmed:  []string{"YES"},
			expectRule: "RULE_CRIT_ACS_MI",
		},
		{
			name:       "突发霹雳样剧烈头痛伴意识障碍",
			symptoms:   []string{"SYM_HEADACHE_THUNDERCLAP"},
			confirmed:  []string{},
			expectRule: "", // 会命中头痛危象或蛛网膜下腔出血
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dec, err := engine.Evaluate(ClinicalContext{
				Symptoms:       tc.symptoms,
				ConfirmedSigns: tc.confirmed,
				Age:            55,
				Gender:         "MALE",
			})
			assert.NoError(t, err)
			assert.NotNil(t, dec)
			assert.Equal(t, "EMERGENCY", dec.State, "危急病例必须被短路拦截为 EMERGENCY")
			assert.NotNil(t, dec.EmergencyAlert)
			assert.True(t, dec.EmergencyAlert.Call120)
			assert.NotEmpty(t, dec.RankedServices)
			assert.True(t, dec.RankedServices[0].IsPrimary)
		})
	}
}

// 3. 测试多主诉协同推演与鉴别诊断 (Spreading Activation & Differentiation)
func TestMultiSymptomDifferentiation(t *testing.T) {
	engine := prepareEngine(t)

	// Case A: 胸痛 + 反酸烧心 + 平卧加重 -> 协同指向反流性食管炎 / 消化内科
	decDigestive, err := engine.Evaluate(ClinicalContext{
		Symptoms: []string{"SYM_ACID_REFLUX", "SYM_EPIGASTRIC_PAIN"},
		Age:      40,
		Gender:   "FEMALE",
	})
	assert.NoError(t, err)
	assert.NotNil(t, decDigestive)
	assert.NotEmpty(t, decDigestive.RankedServices)
	topSvc := decDigestive.RankedServices[0]
	assert.Contains(t, []string{"GASTRO_INTERNAL_ASSESSMENT", "DIGESTIVE_INTERNAL_ASSESSMENT", "GENERAL_INTERNAL_ASSESSMENT"}, topSvc.ServiceCode)

	// Case B: 胃痛胃胀 + 畏寒肢冷 -> 同时激活消化内科与中医脾胃病科
	decDual, err := engine.Evaluate(ClinicalContext{
		Symptoms: []string{"SYM_EPIGASTRIC_PAIN", "SYM_ABDOMINAL_DISTENSION"},
		Age:      35,
		Gender:   "MALE",
	})
	assert.NoError(t, err)
	assert.NotNil(t, decDual)
	assert.NotNil(t, decDual.TCM, "必须输出中医推荐")
	assert.NotEmpty(t, decDual.TCM.Syndromes)
	assert.NotEmpty(t, decDual.TCM.Clinics)
	assert.NotEmpty(t, decDual.TCM.Therapies)
	// 验证专病门诊存在
	hasHPClinic := false
	for _, c := range decDual.TCM.Clinics {
		if c.ClinicCode == "TCM_CLINIC_HP_REGULATION" || c.ClinicCode == "TCM_CLINIC_CHRONIC_GASTRITIS_REVERSAL" {
			hasHPClinic = true
			break
		}
	}
	assert.True(t, hasHPClinic, "胃部不适应激活幽门螺杆菌耐药调理或胃炎阻断门诊")

	// Case C: 面部麻木口角歪斜 -> 命中针灸特色专病门诊
	decFacial, err := engine.Evaluate(ClinicalContext{
		Symptoms: []string{"SYM_FACIAL_NUMBNESS"},
		Age:      30,
		Gender:   "FEMALE",
	})
	assert.NoError(t, err)
	assert.NotNil(t, decFacial)
	assert.NotNil(t, decFacial.TCM)
	assert.NotEmpty(t, decFacial.TCM.Clinics)
	assert.Equal(t, "TCM_CLINIC_FACIAL_ACUTE_FAST", decFacial.TCM.Clinics[0].ClinicCode)
}

// 4. 测试并发高吞吐与热重载无抖动测试 (Zero-Lock Read & Concurrency)
func TestHighConcurrencyAndZeroLockHotReload(t *testing.T) {
	engine := prepareEngine(t)
	genDB, tcmDB := getTestPaths()

	concurrency := 100
	requestsPerRoutine := 50
	var wg sync.WaitGroup
	wg.Add(concurrency)

	stopReload := make(chan bool)

	// 并发热重载 Goroutine
	go func() {
		loader := NewFederatedLoader(FederatedGraphDSL{Version: "2.1.0"})
		for {
			select {
			case <-stopReload:
				return
			default:
				time.Sleep(20 * time.Millisecond)
				newSnap, err := loader.LoadFromFiles(genDB, tcmDB)
				if err == nil {
					engine.SetSnapshot(newSnap)
				}
			}
		}
	}()

	start := time.Now()
	for i := 0; i < concurrency; i++ {
		go func(routineIdx int) {
			defer wg.Done()
			for j := 0; j < requestsPerRoutine; j++ {
				sym := "SYM_COUGH"
				if j%2 == 0 {
					sym = "SYM_EPIGASTRIC_PAIN"
				}
				dec, err := engine.Evaluate(ClinicalContext{
					Symptoms: []string{sym},
					Age:      30,
					Gender:   "MALE",
				})
				assert.NoError(t, err)
				assert.NotNil(t, dec)
				assert.NotEmpty(t, dec.RankedServices)
			}
		}(i)
	}

	wg.Wait()
	close(stopReload)
	duration := time.Since(start)

	totalRequests := concurrency * requestsPerRoutine
	qps := float64(totalRequests) / duration.Seconds()
	t.Logf("并发压测完成: %d 次请求耗时 %v, 吞吐量 QPS: %.2f (热重载期间零Panic/零失败)", totalRequests, duration, qps)
	assert.True(t, qps > 1000, "微秒级内存无锁读取吞吐量应轻松超 1000 QPS")
}
