package utils

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/utils/conn"
	"github.com/yaoapp/yao/utils/datetime"
	"github.com/yaoapp/yao/utils/fmt"
	"github.com/yaoapp/yao/utils/json"
	"github.com/yaoapp/yao/utils/redis"
	"github.com/yaoapp/yao/utils/str"
	"github.com/yaoapp/yao/utils/template"
	"github.com/yaoapp/yao/utils/throw"
	"github.com/yaoapp/yao/utils/tree"
	"github.com/yaoapp/yao/utils/url"
	"github.com/yaoapp/yao/utils/x2j"
)

// Init the utils
func Init() {
	// 加载所有 Redis 连接器客户端
	redis.LoadAllClients()

	process.Alias("xiang.helper.Captcha", "yao.utils.Captcha")                 // deprecated
	process.Alias("xiang.helper.CaptchaValidate", "yao.utils.CaptchaValidate") // deprecated

	// ****************************************
	// * Processes Version 0.10.4+
	// ****************************************
	process.Register("utils.throw.Forbidden", throw.Forbidden)
	process.Register("utils.throw.Unauthorized", throw.Unauthorized)
	process.Register("utils.throw.NotFound", throw.NotFound)
	process.Register("utils.throw.BadRequest", throw.BadRequest)
	process.Register("utils.throw.InternalError", throw.InternalError)
	process.Register("utils.throw.Exception", throw.Exception)

	// ****************************************
	// * Migrate Processes Version 0.10.2+
	// ****************************************

	// FMT
	process.Alias("xiang.helper.Print", "utils.fmt.Print")
	process.Register("utils.fmt.Printf", fmt.ProcessPrintf)
	process.Register("utils.fmt.ColorPrintf", fmt.ProcessColorPrintf)

	// ENV
	process.Alias("xiang.helper.EnvSet", "utils.env.Set")
	process.Alias("xiang.helper.EnvGet", "utils.env.Get")
	process.Alias("xiang.helper.EnvMultiSet", "utils.env.SetMany")
	process.Alias("xiang.helper.EnvMultiGet", "utils.env.GetMany")

	// Flow
	process.Alias("xiang.helper.For", "utils.flow.For")
	process.Alias("xiang.helper.Each", "utils.flow.Each")
	process.Alias("xiang.helper.Case", "utils.flow.Case")
	process.Alias("xiang.helper.IF", "utils.flow.IF")
	process.Alias("xiang.helper.Throw", "utils.flow.Throw")
	process.Alias("xiang.helper.Return", "utils.flow.Return")

	// JWT
	process.Alias("xiang.helper.JwtMake", "utils.jwt.Make")
	process.Alias("xiang.helper.JwtValidate", "utils.jwt.Verify")

	// Password
	// utils.pwd.Hash
	process.Alias("xiang.helper.PasswordValidate", "utils.pwd.Verify")

	// Captcha
	process.Alias("xiang.helper.Captcha", "utils.captcha.Make")
	process.Alias("xiang.helper.CaptchaValidate", "utils.captcha.Verify")

	// String
	process.Alias("xiang.helper.StrConcat", "utils.str.Concat")
	process.Alias("xiang.helper.HexToString", "utils.str.Hex")
	process.Register("utils.str.Join", str.ProcessJoin)
	process.Register("utils.str.JoinPath", str.ProcessJoinPath)
	process.Register("utils.str.UUID", str.ProcessUUID)
	process.Register("utils.str.Pinyin", str.ProcessPinyin)

	// Array
	process.Alias("xiang.helper.ArrayPluck", "utils.arr.Pluck")
	process.Alias("xiang.helper.ArraySplit", "utils.arr.Split")
	process.Alias("xiang.helper.ArrayTree", "utils.arr.Tree")
	process.Alias("xiang.helper.ArrayUnique", "utils.arr.Unique")
	process.Alias("xiang.helper.ArrayIndexes", "utils.arr.Indexes")
	process.Alias("xiang.helper.ArrayGet", "utils.arr.Get")
	process.Alias("xiang.helper.ArrayColumn", "utils.arr.Column") // doc
	process.Alias("xiang.helper.ArrayKeep", "utils.arr.Keep")
	process.Alias("xiang.helper.ArrayMapSet", "utils.arr.MapSet")

	// Tree
	process.Register("utils.tree.Flatten", tree.ProcessFlatten)

	// Map
	process.Alias("xiang.helper.MapGet", "utils.map.Get")
	process.Alias("xiang.helper.MapSet", "utils.map.Set")
	process.Alias("xiang.helper.MapDel", "utils.map.Del")
	process.Alias("xiang.helper.MapDel", "utils.map.DelMany")
	process.Alias("xiang.helper.MapKeys", "utils.map.Keys")
	process.Alias("xiang.helper.MapValues", "utils.map.Values")
	process.Alias("xiang.helper.MapToArray", "utils.map.Array") // doc
	// utils.map.Merge

	// Time
	process.Alias("xiang.flow.Sleep", "utils.time.Sleep")
	process.Register("utils.now.Time", datetime.ProcessTime)
	process.Register("utils.now.Date", datetime.ProcessDate)
	process.Register("utils.now.DateTime", datetime.ProcessDateTime)
	process.Register("utils.now.Timestamp", datetime.ProcessTimestamp)
	process.Register("utils.now.Timestampms", datetime.ProcessTimestampms)

	// URL
	process.Register("utils.url.ParseQuery", url.ProcessParseQuery)
	process.Register("utils.url.QueryParam", url.ProcessQueryParam)
	process.Register("utils.url.ParseURL", url.ProcessParseURL)

	// JSON
	process.Register("utils.json.Validate", json.ProcessValidate)

	// Template
	process.RegisterGroup("utils.template", map[string]process.Handler{
		"render":        template.ProcessRender,
		"renderContent": template.ProcessRenderContent,
		"register":      template.ProcessRegister,
		"get":           template.ProcessGet,
		"list":          template.ProcessList,
		"remove":        template.ProcessRemove,
		"clear":         template.ProcessClear,
		"init":          template.ProcessInit,
	})

	// Xml2Json
	process.RegisterGroup("utils.x2j", map[string]process.Handler{
		"XmlToJson":            x2j.ProcessXmlToJson,
		"XmlToMap":             x2j.ProcessXmlToMap,
		"MapToXml":             x2j.ProcessMapToXml,
		"XmlValuesForTag":      x2j.ProcessXmlValuesForTag,
		"XmlPathsForTag":       x2j.ProcessXmlPathsForTag,
		"XmlUpdateValsForPath": x2j.ProcessXmlUpdateValsForPath,
	})

	// Connector
	process.RegisterGroup("utils.connector", map[string]process.Handler{
		"select": conn.ProcessSelectConnector,
	})

	// Redis - 实用的高级方法
	process.RegisterGroup("utils.redis", map[string]process.Handler{
		// JSON 操作 - 便捷存取 JSON 对象
		"SetJSON": redis.ProcessSetJSON,
		"GetJSON": redis.ProcessGetJSON,

		// 批量操作 - 一次操作多个键
		"MSet": redis.ProcessMSet,
		"MGet": redis.ProcessMGet,

		// 计数器 - 页面访问、点赞等
		"CounterIncr":  redis.ProcessCounterIncr,
		"CounterDecr":  redis.ProcessCounterDecr,
		"CounterGet":   redis.ProcessCounterGet,
		"CounterReset": redis.ProcessCounterReset,

		// 排行榜 - 游戏分数、热度排名等
		"RankingAdd":     redis.ProcessRankingAdd,
		"RankingIncrBy":  redis.ProcessRankingIncrBy,
		"RankingTop":     redis.ProcessRankingTop,
		"RankingGetRank": redis.ProcessRankingGetRank,
		"RankingRemove":  redis.ProcessRankingRemove,

		// 批量操作 - Pipeline 简化版
		"Batch": redis.ProcessBatch,

		// 清理工具
		"ClearPattern": redis.ProcessClearPattern,

		// 基础操作（保留最常用的）
		"Get":    redis.ProcessGet,
		"Set":    redis.ProcessSet,
		"Del":    redis.ProcessDel,
		"Unlink": redis.ProcessUnlink,
		"Exists": redis.ProcessExists,
		"Expire": redis.ProcessExpire,
		"TTL":    redis.ProcessTTL,
		"Keys":   redis.ProcessKeys,

		// Hash 操作
		"HGet":    redis.ProcessHGet,
		"HSet":    redis.ProcessHSet,
		"HMSet":   redis.ProcessHMSet,
		"HGetAll": redis.ProcessHGetAll,
		"HDel":    redis.ProcessHDel,

		// List 操作
		"LPush":  redis.ProcessLPush,
		"RPush":  redis.ProcessRPush,
		"LPop":   redis.ProcessLPop,
		"RPop":   redis.ProcessRPop,
		"LRange": redis.ProcessLRange,
		"LLen":   redis.ProcessLLen,

		// Set 操作
		"SAdd":     redis.ProcessSAdd,
		"SRem":     redis.ProcessSRem,
		"SMembers": redis.ProcessSMembers,

		// Sorted Set 操作
		"ZAdd":      redis.ProcessZAdd,
		"ZRem":      redis.ProcessZRem,
		"ZRange":    redis.ProcessZRange,
		"ZRevRange": redis.ProcessZRevRange,
		"ZScore":    redis.ProcessZScore,
		"ZCard":     redis.ProcessZCard,
		"ZIncrBy":   redis.ProcessZIncrBy,
		"ZRank":     redis.ProcessZRank,
		"ZRevRank":  redis.ProcessZRevRank,
	})
}
