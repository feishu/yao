package triagekb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNLUDeterministicPerformanceNegationAndCompanions(t *testing.T) {
	eng := prepareEngine(t)
	snap := eng.GetSnapshot()
	require.NotNil(t, snap)

	// 1. 口语短语映射 + Zero-LLM < 15ms 性能断言 + 只输出真实存在概念（绝不虚构 SYM_DIARRHEA / SYM_FEVER）
	utts := []string{
		"28岁女性突发拉肚子水样便伴腹痛",
		"头晕天旋地转，不敢睁眼",
		"心慌气短难受好几天了",
		"拉黑便呕血头晕冷汗",
		"这几天一直咳嗽有痰，但绝对没有胸痛，也不发烧",
	}
	start := time.Now()
	iterations := 100
	for i := 0; i < iterations; i++ {
		_ = AnalyzeUtterance(snap, utts[i%len(utts)], NLUPatientContext{Gender: "UNKNOWN"})
	}
	avgMs := float64(time.Since(start).Microseconds()) / float64(iterations) / 1000.0
	assert.Less(t, avgMs, 15.0, "Zero-LLM E1 平均解析延迟必须低于 15ms (实测 %.3fms)", avgMs)

	// 验证真实数据库编码
	resDiarrhea := AnalyzeUtterance(snap, "昨晚吃坏东西一直拉肚子，水样大便", NLUPatientContext{})
	assert.Contains(t, resDiarrhea.SeedConcepts, "SYM_DIARRHEA_WATERY")
	assert.NotContains(t, resDiarrhea.SeedConcepts, "SYM_DIARRHEA", "严禁捏造不存在的概念编码")
	assert.Equal(t, "UNKNOWN", resDiarrhea.ExtractedGender, "未明确提及性别时必须保真为 UNKNOWN，严禁伪造 MALE")

	// 验证 ALL 自动归一化为 UNKNOWN
	resAllGender := AnalyzeUtterance(snap, "头痛难受", NLUPatientContext{Gender: "ALL"})
	assert.Equal(t, "UNKNOWN", resAllGender.ExtractedGender)

	// 2. 分句切分 + 0~6 字符否定窗口，严防“不发烧/无胸痛/否认黑便”污染主诉
	resNeg := AnalyzeUtterance(snap, "连续咳嗽三天，但是不发烧，也没有胸痛，否认咯血、黑便", NLUPatientContext{Gender: "FEMALE"})
	assert.Equal(t, "FEMALE", resNeg.ExtractedGender)
	assert.Equal(t, "NO_FEVER", resNeg.KnownSlots["has_fever"])
	assert.Equal(t, "NO", resNeg.KnownSlots["chest_pain"])
	for _, negCode := range resNeg.NegatedConcepts {
		assert.NotContains(t, resNeg.SeedConcepts, negCode, "被否定主诉 %s 不得泄露入 SeedConcepts", negCode)
	}
	assert.NotContains(t, resNeg.SeedConcepts, "SYM_CHEST_PAIN")
	assert.NotContains(t, resNeg.SeedConcepts, "SYM_MELENA_BLACK_STOOL")

	// 3. 人口学与病程起病提取
	resDemo := AnalyzeUtterance(snap, "32岁女性突发剧烈头痛3小时，怀孕3个月", NLUPatientContext{})
	require.NotNil(t, resDemo.ExtractedAge)
	assert.Equal(t, 32, *resDemo.ExtractedAge)
	assert.Equal(t, "FEMALE", resDemo.ExtractedGender)
	require.NotNil(t, resDemo.ExtractedPregnant)
	assert.True(t, *resDemo.ExtractedPregnant)
	assert.Equal(t, "ONSET_ACUTE", resDemo.KnownSlots["onset_acute"])

	// 文本显式年龄必须覆盖外部默认 30 岁兜底（代挂号场景）
	resProxyElderly := AnalyzeUtterance(snap, "爷爷72岁，昨天开始胸口闷痛喘不上气，没有发烧", NLUPatientContext{Age: 30, Gender: "UNKNOWN"})
	require.NotNil(t, resProxyElderly.ExtractedAge)
	assert.Equal(t, 72, *resProxyElderly.ExtractedAge)
	assert.Equal(t, "ELDERLY", resProxyElderly.InferredAgeGroup)
	assert.Equal(t, "MALE", resProxyElderly.ExtractedGender)
	assert.Equal(t, "NO_FEVER", resProxyElderly.KnownSlots["has_fever"])
	assert.Contains(t, resProxyElderly.SeedConcepts, "SYM_CHEST_PAIN")

	resKid := AnalyzeUtterance(snap, "6岁男孩反复咳嗽", NLUPatientContext{})
	require.NotNil(t, resKid.ExtractedAge)
	assert.Equal(t, 6, *resKid.ExtractedAge)
	assert.Equal(t, "PEDIATRIC", resKid.InferredAgeGroup)
	assert.Equal(t, "MALE", resKid.ExtractedGender)

	// 4. 字面红旗直出预确认（雷击样头痛 / 劈裂样胸痛）
	resRed := AnalyzeUtterance(snap, "突发雷击样剧烈头痛，像炸裂一样", NLUPatientContext{})
	assert.Contains(t, resRed.SeedConcepts, "SYM_HEADACHE_THUNDERCLAP")

	// 5. COMPANION_ONLY 伴随症状隔离：即便命中陪伴特征也不得反客为主压倒 CORE_COMPLAINT
	snapClone := *snap
	symMap := make(map[string]*SymptomNode, len(snap.Symptoms)+1)
	for k, v := range snap.Symptoms {
		symMap[k] = v
	}
	symMap["SYM_COMPANION_CHILLS"] = &SymptomNode{
		ConceptCode:        "SYM_COMPANION_CHILLS",
		ConceptName:        "畏寒怕冷",
		TriageRole:         TriageRoleCompanionOnly,
		ColloquialAliases:  []string{"怕冷"},
		DefaultServiceCode: "INFECTIOUS_ASSESSMENT",
	}
	snapClone.Symptoms = symMap
	snapClone.BuildInvertedIndex()

	resComp := AnalyzeUtterance(&snapClone, "怕冷而且剧烈咳嗽", NLUPatientContext{})
	assert.Contains(t, resComp.CompanionConcepts, "SYM_COMPANION_CHILLS")
	assert.NotContains(t, resComp.SeedConcepts, "SYM_COMPANION_CHILLS", "COMPANION_ONLY 不得作为核心主诉进入 SeedConcepts")
}
