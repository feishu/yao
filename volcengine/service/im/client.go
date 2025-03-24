package im

// client 这个文件由 Codegen 生成，但后续不再更新（如果没有，则会重新生成），包含配置结构体和创建服务实例的代码
// 开发者可以在这里给服务结构体添加自定义扩展方法

import (
	"fmt"

	"github.com/yaoapp/yao/volcengine"
	common "github.com/yaoapp/yao/volcengine/base"
)

type Im struct {
	*common.Client
}

func NewInstance() *Im {
	return NewInstanceWithRegion("cn-north-1")
}

func NewInstanceWithRegion(region string) *Im {
	serviceInfo, ok := ServiceInfoMap[region]
	if !ok {
		panic(fmt.Errorf("Im not support region %s", region))
	}
	instance := &Im{
		Client: common.NewClient(&serviceInfo, ApiListInfo),
	}

	instance.SetCredential(common.Credentials{
		AccessKeyID:     volcengine.VolcEngine.Creds.AccessKeyID,
		SecretAccessKey: volcengine.VolcEngine.Creds.AccessKeySecret,
	})

	return instance
}
