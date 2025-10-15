package payment

import (
	"github.com/yaoapp/yao/payment/types"
	"fmt"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

// ProcessSetMerchantConfig 设置商户配置的 Process 接口
// process.New("payment.SetMerchantConfig", merchantID, channel, config).Run()
//
// 参数:
//   args[0] string: 商户ID
//   args[1] string: 支付渠道 (alipay/wechat)
//   args[2] map[string]interface{}: 配置参数
//
// 配置参数说明:
//   支付宝 (alipay):
//     - app_id (必需): 应用ID
//     - private_key (必需): 应用私钥
//     - public_key (必需): 支付宝公钥
//     - sign_type (可选): 签名类型，默认RSA2
//     - is_sandbox (可选): 是否沙箱环境，默认false
//
//   微信 (wechat):
//     - app_id (必需): 应用ID
//     - mch_id (必需): 商户号
//     - apiv3_key (必需): APIv3密钥
//     - private_key (必需): 商户私钥
//     - serial_no (必需): 证书序列号
//     - is_sandbox (可选): 是否沙箱环境，默认false
//
// 返回: nil (成功) 或抛出异常
//
// 示例:
//   // 设置支付宝配置
//   process.New("payment.SetMerchantConfig", "merchant001", "alipay", map[string]interface{}{
//       "app_id": "2021001234567890",
//       "private_key": "-----BEGIN RSA PRIVATE KEY-----...",
//       "public_key": "-----BEGIN PUBLIC KEY-----...",
//       "is_sandbox": true,
//   }).Run()
//
//   // 设置微信配置
//   process.New("payment.SetMerchantConfig", "merchant002", "wechat", map[string]interface{}{
//       "app_id": "wx1234567890",
//       "mch_id": "1234567890",
//       "apiv3_key": "your_apiv3_key",
//       "private_key": "-----BEGIN PRIVATE KEY-----...",
//       "serial_no": "1234567890ABCDEF",
//   }).Run()
func ProcessSetMerchantConfig(proc *process.Process) interface{} {
	proc.ValidateArgNums(3)

	// 获取参数
	merchantID := proc.ArgsString(0)
	channelStr := proc.ArgsString(1)
	config := proc.ArgsMap(2)

	// 验证参数
	if merchantID == "" {
		exception.New("merchant_id is required", 400).Throw()
	}
	if channelStr == "" {
		exception.New("channel is required", 400).Throw()
	}
	if len(config) == 0 {
		exception.New("config is required", 400).Throw()
	}

	// 解析渠道
	channel := types.PaymentChannel(channelStr)
	if !isValidPaymentChannel(channelStr) {
		exception.New(fmt.Sprintf("invalid channel: %s (支持: alipay, wechat)", channelStr), 400).Throw()
	}

	// 设置配置
	err := Manager.SetMerchantConfig(merchantID, channel, config)
	if err != nil {
		exception.New(fmt.Sprintf("设置商户配置失败: %v", err), 500).Throw()
	}

	return nil
}

// ProcessGetMerchantConfig 获取商户配置的 Process 接口
// process.New("payment.GetMerchantConfig", merchantID, channel).Run()
//
// 参数:
//   args[0] string: 商户ID
//   args[1] string: 支付渠道 (alipay/wechat)
//
// 返回: map[string]interface{} 配置参数
//
// 示例:
//   config := process.New("payment.GetMerchantConfig", "merchant001", "alipay").Run()
func ProcessGetMerchantConfig(proc *process.Process) interface{} {
	proc.ValidateArgNums(2)

	// 获取参数
	merchantID := proc.ArgsString(0)
	channelStr := proc.ArgsString(1)

	// 验证参数
	if merchantID == "" {
		exception.New("merchant_id is required", 400).Throw()
	}
	if channelStr == "" {
		exception.New("channel is required", 400).Throw()
	}

	// 解析渠道
	channel := types.PaymentChannel(channelStr)
	if !isValidPaymentChannel(channelStr) {
		exception.New(fmt.Sprintf("invalid channel: %s (支持: alipay, wechat)", channelStr), 400).Throw()
	}

	// 获取配置
	config, err := Manager.GetMerchantConfig(merchantID, channel)
	if err != nil {
		exception.New(fmt.Sprintf("获取商户配置失败: %v", err), 404).Throw()
	}

	return config
}

// ProcessListMerchants 列出所有已配置的商户
// process.New("payment.ListMerchants").Run()
//
// 返回: []map[string]interface{} 商户列表
//
// 示例:
//   merchants := process.New("payment.ListMerchants").Run()
//   // [
//   //   {"merchant_id": "merchant001", "channel": "alipay", "has_config": true},
//   //   {"merchant_id": "merchant002", "channel": "wechat", "has_config": true},
//   // ]
func ProcessListMerchants(proc *process.Process) interface{} {
	proc.ValidateArgNums(0)

	Manager.mutex.RLock()
	defer Manager.mutex.RUnlock()

	merchants := make([]map[string]interface{}, 0, len(Manager.certConfigs))
	for key, certConfig := range Manager.certConfigs {
		merchants = append(merchants, map[string]interface{}{
			"key":         key,
			"merchant_id": certConfig.MerchantID,
			"channel":     string(certConfig.Channel),
			"has_app_id":  certConfig.AppID != "",
			"has_mch_id":  certConfig.MchID != "",
			"has_cert":    certConfig.PrivateKey != "" && certConfig.PublicKey != "",
		})
	}

	return merchants
}
