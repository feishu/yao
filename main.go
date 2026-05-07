package main

import (
	"github.com/yaoapp/yao/cmd"
	"github.com/yaoapp/yao/utils"

	// 导入模块以触发 init 函数
	_ "github.com/yaoapp/gou/diff"
	_ "github.com/yaoapp/gou/encoding"

	// _ "github.com/yaoapp/yao/aigc"
	_ "github.com/yaoapp/yao/attachment"
	_ "github.com/yaoapp/yao/crypto"
	_ "github.com/yaoapp/yao/excel"
	_ "github.com/yaoapp/yao/helper"
	_ "github.com/yaoapp/yao/openai"
	_ "github.com/yaoapp/yao/payment"
	_ "github.com/yaoapp/yao/volcengine/service/coze"
	_ "github.com/yaoapp/yao/volcengine/service/im"
	_ "github.com/yaoapp/yao/volcengine/service/rtc"
	_ "github.com/yaoapp/yao/wework"
	// _ "net/http/pprof"
)

func main() {
	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()
	utils.Init()
	cmd.Execute()
}
