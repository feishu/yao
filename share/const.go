package share

// VERSION Yao App Engine Version
const VERSION = "1.0.0"

<<<<<<< HEAD
// PRVERSION Yao App Engine PR Commit
const PRVERSION = "DEV"
=======
// PRVERSION  PreRelease Version
const PRVERSION = "f393289387f0-2025-03-24T14:30:18+0800-debug"
>>>>>>> d29d425f (在IM服务客户端中添加了对AccessKey和SecretKey的打印输出，便于调试和验证服务配置。)

// CUI Version
const CUI = "1.0.0"

// PRCUI CUI PR Commit
const PRCUI = "DEV"

// BUILDIN If true, the application will be built into a single artifact
const BUILDIN = false

// BUILDNAME The name of the artifact
const BUILDNAME = "yao"

// MoapiHosts the master mirror
var MoapiHosts = []string{
	"master.moapi.ai",
	"master-moon.moapi.ai",
	"master-earth.moapi.ai",
	"master-mars.moapi.ai",
	"master-venus.moapi.ai",
	"master-mercury.moapi.ai",
	"master-jupiter.moapi.ai",
	"master-saturn.moapi.ai",
	"master-uranus.moapi.ai",
	"master-neptune.moapi.ai",
	"master-pluto.moapi.ai",
}
