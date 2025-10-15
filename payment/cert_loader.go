package payment

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
)

// CertConfig 证书配置结构（统一配置入口）
type CertConfig struct {
	// 基本信息
	MerchantID string         // 商户ID
	Channel    PaymentChannel // 支付渠道

	// 证书文件（从文件加载）
	PrivateKey string // 私钥内容
	PublicKey  string // 公钥内容
	AppCert    string // 应用公钥证书（支付宝）
	RootCert   string // 支付宝根证书

	// 支付宝特有配置
	AppID     string // 应用ID
	SignType  string // 签名类型 (RSA2/RSA)
	IsSandbox bool   // 是否沙箱环境

	// 微信特有配置
	MchID    string // 商户号
	APIv3Key string // APIv3密钥
	SerialNo string // 证书序列号

	// 扩展字段（其他任意配置）
	ExtraFields map[string]interface{} // 用于存储额外配置
	ExtraFiles  map[string]string      // 其他证书文件
}

// LoadCertsFromDirectory 从目录自动加载证书
// 目录结构: certs/[商户号]/[渠道]/private_key.pem、public_key.pem
func LoadCertsFromDirectory() (map[string]*CertConfig, error) {
	if config.Conf.Root == "" {
		return nil, fmt.Errorf("application root not initialized")
	}

	certsDir := filepath.Join(config.Conf.Root, "certs")

	// 检查certs目录是否存在
	if _, err := os.Stat(certsDir); os.IsNotExist(err) {
		log.Warn("Certs directory not found: %s, skip auto-loading certificates", certsDir)
		return make(map[string]*CertConfig), nil
	}

	log.Info("Loading certificates from directory: %s", certsDir)

	certConfigs := make(map[string]*CertConfig)

	// 遍历商户目录
	merchantDirs, err := os.ReadDir(certsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read certs directory: %v", err)
	}

	for _, merchantDir := range merchantDirs {
		if !merchantDir.IsDir() {
			continue
		}

		merchantID := merchantDir.Name()
		merchantPath := filepath.Join(certsDir, merchantID)

		log.Debug("Scanning merchant directory: %s", merchantID)

		// 遍历渠道目录
		channelDirs, err := os.ReadDir(merchantPath)
		if err != nil {
			log.Warn("Failed to read merchant directory %s: %v", merchantID, err)
			continue
		}

		for _, channelDir := range channelDirs {
			if !channelDir.IsDir() {
				continue
			}

			channelName := channelDir.Name()
			channel := parseChannel(channelName)

			if channel == "" {
				log.Warn("Unknown channel directory: %s/%s, skipping", merchantID, channelName)
				continue
			}

			channelPath := filepath.Join(merchantPath, channelName)

			log.Debug("Loading certificates for %s/%s", merchantID, channelName)

			// 加载证书文件
			certConfig, err := loadCertFiles(merchantID, channel, channelPath)
			if err != nil {
				log.Error("Failed to load certificates for %s/%s: %v", merchantID, channelName, err)
				continue
			}

			// 存储配置
			key := fmt.Sprintf("%s_%s", merchantID, string(channel))
			certConfigs[key] = certConfig

			log.Info("✓ Loaded certificates for merchant: %s, channel: %s", merchantID, channelName)
		}
	}

	log.Info("Certificate auto-loading completed: %d configurations loaded", len(certConfigs))

	return certConfigs, nil
}

// loadCertFiles 从指定路径加载证书文件
func loadCertFiles(merchantID string, channel PaymentChannel, certPath string) (*CertConfig, error) {
	certConfig := &CertConfig{
		MerchantID:  merchantID,
		Channel:     channel,
		ExtraFiles:  make(map[string]string),
		ExtraFields: make(map[string]interface{}),
	}

	// 支持的文件扩展名
	certExtensions := []string{".pem", ".key", ".pub", ".crt", ".cer", ".p12", ".pfx"}

	// 读取私钥（必需）
	privateKeyPath, err := findCertFile(certPath, "private_key", certExtensions)
	if err != nil {
		return nil, fmt.Errorf("failed to find private key file: %v", err)
	}
	privateKeyContent, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key from %s: %v", privateKeyPath, err)
	}
	certConfig.PrivateKey = string(privateKeyContent)
	log.Debug("Loaded private key from: %s", privateKeyPath)

	// 读取公钥（必需）
	publicKeyPath, err := findCertFile(certPath, "public_key", certExtensions)

	if publicKeyPath != "" {
		if err != nil {
			return nil, fmt.Errorf("failed to find public key file: %v", err)
		}
		publicKeyContent, err := os.ReadFile(publicKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read public key from %s: %v", publicKeyPath, err)
		}
		certConfig.PublicKey = string(publicKeyContent)
		log.Debug("Loaded public key from: %s", publicKeyPath)
	} else {
		log.Warn("No public key file found for %s/%s, continue loading without public key", merchantID, channel)
	}

	// 根据渠道加载额外文件
	switch channel {
	case ChannelAlipay:
		// 支付宝可能需要证书模式的文件
		certConfig.loadAlipayExtraFiles(certPath)
	case ChannelWechat:
		// 微信可能需要额外的证书文件
		certConfig.loadWechatExtraFiles(certPath)
	}

	return certConfig, nil
}

// loadAlipayExtraFiles 加载支付宝额外证书文件（证书模式）
func (c *CertConfig) loadAlipayExtraFiles(certPath string) {
	certExtensions := []string{".crt", ".cer", ".pem"}

	// 应用公钥证书（可选）
	if appCertPath, err := findCertFile(certPath, "app_cert", certExtensions); err == nil {
		if content, err := os.ReadFile(appCertPath); err == nil {
			c.AppCert = string(content)
			c.ExtraFiles["app_cert"] = string(content)
			log.Debug("Loaded app cert from: %s", appCertPath)
		}
	}

	// 支付宝公钥证书（可选）
	if alipayCertPath, err := findCertFile(certPath, "alipay_cert", certExtensions); err == nil {
		if content, err := os.ReadFile(alipayCertPath); err == nil {
			c.ExtraFiles["alipay_cert"] = string(content)
			log.Debug("Loaded alipay cert from: %s", alipayCertPath)
		}
	}
	// 也尝试 alipay_public_cert 命名
	if alipayCertPath, err := findCertFile(certPath, "alipay_public_cert", certExtensions); err == nil {
		if content, err := os.ReadFile(alipayCertPath); err == nil {
			c.ExtraFiles["alipay_public_cert"] = string(content)
			log.Debug("Loaded alipay public cert from: %s", alipayCertPath)
		}
	}

	// 支付宝根证书（可选）
	if rootCertPath, err := findCertFile(certPath, "alipay_root_cert", certExtensions); err == nil {
		if content, err := os.ReadFile(rootCertPath); err == nil {
			c.RootCert = string(content)
			c.ExtraFiles["alipay_root_cert"] = string(content)
			log.Debug("Loaded alipay root cert from: %s", rootCertPath)
		}
	}
	// 也尝试 root_cert 命名
	if rootCertPath, err := findCertFile(certPath, "root_cert", certExtensions); err == nil {
		if content, err := os.ReadFile(rootCertPath); err == nil {
			if c.RootCert == "" { // 只有在还没有加载的时候才加载
				c.RootCert = string(content)
				c.ExtraFiles["root_cert"] = string(content)
				log.Debug("Loaded root cert from: %s", rootCertPath)
			}
		}
	}
}

// loadWechatExtraFiles 加载微信额外证书文件
func (c *CertConfig) loadWechatExtraFiles(certPath string) {
	certExtensions := []string{".pem", ".crt", ".cer"}

	// 微信API证书文件（可选）
	if certFilePath, err := findCertFile(certPath, "apiclient_cert", certExtensions); err == nil {
		if content, err := os.ReadFile(certFilePath); err == nil {
			c.ExtraFiles["apiclient_cert"] = string(content)
			log.Debug("Loaded apiclient cert from: %s", certFilePath)
		}
	}

	// 微信API密钥文件（可选）
	keyExtensions := []string{".pem", ".key"}
	if keyFilePath, err := findCertFile(certPath, "apiclient_key", keyExtensions); err == nil {
		if content, err := os.ReadFile(keyFilePath); err == nil {
			c.ExtraFiles["apiclient_key"] = string(content)
			log.Debug("Loaded apiclient key from: %s", keyFilePath)
		}
	}

	// P12证书文件（可选）
	p12Extensions := []string{".p12", ".pfx"}
	if p12Path, err := findCertFile(certPath, "apiclient_cert", p12Extensions); err == nil {
		if content, err := os.ReadFile(p12Path); err == nil {
			c.ExtraFiles["apiclient_cert_p12"] = string(content)
			log.Debug("Loaded apiclient p12 from: %s", p12Path)
		}
	}
}

// findCertFile 查找证书文件，支持多种扩展名
// 例如：findCertFile("/path", "private_key", [".pem", ".key"])
// 会尝试查找：private_key.pem, private_key.key
func findCertFile(certPath, baseName string, extensions []string) (string, error) {
	// 尝试所有可能的扩展名
	for _, ext := range extensions {
		filePath := filepath.Join(certPath, baseName+ext)
		if _, err := os.Stat(filePath); err == nil {
			return filePath, nil
		}
	}

	// 如果没有找到，返回错误信息
	return "", fmt.Errorf("file not found: %s with extensions %v in %s", baseName, extensions, certPath)
}

// parseChannel 解析渠道名称
func parseChannel(channelName string) PaymentChannel {
	// 标准化渠道名称
	channelName = strings.ToLower(channelName)

	switch channelName {
	case "alipay", "支付宝":
		return ChannelAlipay
	case "wechat", "wechatpay", "wxpay", "微信", "微信支付":
		return ChannelWechat
	default:
		return ""
	}
}

// GetCertConfig 从缓存中获取证书配置
func (pm *PaymentManager) GetCertConfig(merchantID string, channel PaymentChannel) (*CertConfig, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	key := fmt.Sprintf("%s_%s", merchantID, string(channel))

	if config, exists := pm.certConfigs[key]; exists {
		return config, nil
	}

	return nil, fmt.Errorf("certificate config not found for %s/%s", merchantID, string(channel))
}

// ToProviderConfig 将证书配置转换为Provider配置
func (c *CertConfig) ToProviderConfig() map[string]interface{} {
	config := map[string]interface{}{}

	// 证书文件
	if c.PrivateKey != "" {
		config["private_key"] = c.PrivateKey
	}
	if c.PublicKey != "" {
		config["public_key"] = c.PublicKey
	}

	// 支付宝特有配置
	if c.Channel == ChannelAlipay {
		if c.AppID != "" {
			config["app_id"] = c.AppID
		}
		if c.SignType != "" {
			config["sign_type"] = c.SignType
		}
		config["is_sandbox"] = c.IsSandbox

		if c.AppCert != "" {
			config["app_cert"] = c.AppCert
		}
		if c.RootCert != "" {
			config["root_cert"] = c.RootCert
		}
	}

	// 微信特有配置
	if c.Channel == ChannelWechat {
		if c.AppID != "" {
			config["app_id"] = c.AppID
		}
		if c.MchID != "" {
			config["mch_id"] = c.MchID
		}
		if c.APIv3Key != "" {
			config["apiv3_key"] = c.APIv3Key
		}
		if c.SerialNo != "" {
			config["serial_no"] = c.SerialNo
		}
		config["is_sandbox"] = c.IsSandbox
	}

	// 添加额外证书文件
	for key, value := range c.ExtraFiles {
		config[key] = value
	}

	// 添加扩展字段
	for key, value := range c.ExtraFields {
		config[key] = value
	}

	return config
}
