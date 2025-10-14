package payment

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
)

// CertConfig 证书配置结构
type CertConfig struct {
	MerchantID  string            // 商户ID
	Channel     PaymentChannel    // 支付渠道
	PrivateKey  string            // 私钥内容
	PublicKey   string            // 公钥内容（支付宝）或证书内容（微信）
	AppCert     string            // 应用公钥证书（支付宝证书模式）
	RootCert    string            // 支付宝根证书
	ExtraFiles  map[string]string // 其他证书文件
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
		MerchantID: merchantID,
		Channel:    channel,
		ExtraFiles: make(map[string]string),
	}

	// 必需文件
	privateKeyPath := filepath.Join(certPath, "private_key.pem")
	publicKeyPath := filepath.Join(certPath, "public_key.pem")

	// 读取私钥（必需）
	privateKeyContent, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private_key.pem: %v", err)
	}
	certConfig.PrivateKey = string(privateKeyContent)

	// 读取公钥（必需）
	publicKeyContent, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public_key.pem: %v", err)
	}
	certConfig.PublicKey = string(publicKeyContent)

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
	// 应用公钥证书（可选）
	appCertPath := filepath.Join(certPath, "app_cert.crt")
	if content, err := os.ReadFile(appCertPath); err == nil {
		c.AppCert = string(content)
		c.ExtraFiles["app_cert"] = string(content)
		log.Debug("Loaded app_cert.crt for %s/alipay", c.MerchantID)
	}

	// 支付宝公钥证书（可选）
	alipayCertPath := filepath.Join(certPath, "alipay_cert.crt")
	if content, err := os.ReadFile(alipayCertPath); err == nil {
		c.ExtraFiles["alipay_cert"] = string(content)
		log.Debug("Loaded alipay_cert.crt for %s/alipay", c.MerchantID)
	}

	// 支付宝根证书（可选）
	rootCertPath := filepath.Join(certPath, "alipay_root_cert.crt")
	if content, err := os.ReadFile(rootCertPath); err == nil {
		c.RootCert = string(content)
		c.ExtraFiles["alipay_root_cert"] = string(content)
		log.Debug("Loaded alipay_root_cert.crt for %s/alipay", c.MerchantID)
	}
}

// loadWechatExtraFiles 加载微信额外证书文件
func (c *CertConfig) loadWechatExtraFiles(certPath string) {
	// 微信证书文件（可选）
	certFilePath := filepath.Join(certPath, "apiclient_cert.pem")
	if content, err := os.ReadFile(certFilePath); err == nil {
		c.ExtraFiles["apiclient_cert"] = string(content)
		log.Debug("Loaded apiclient_cert.pem for %s/wechat", c.MerchantID)
	}

	// P12证书文件（可选）
	p12Path := filepath.Join(certPath, "apiclient_cert.p12")
	if content, err := os.ReadFile(p12Path); err == nil {
		c.ExtraFiles["apiclient_cert_p12"] = string(content)
		log.Debug("Loaded apiclient_cert.p12 for %s/wechat", c.MerchantID)
	}
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
	config := map[string]interface{}{
		"private_key": c.PrivateKey,
		"public_key":  c.PublicKey,
	}

	// 添加额外文件
	for key, value := range c.ExtraFiles {
		config[key] = value
	}

	// 支付宝特有配置
	if c.Channel == ChannelAlipay {
		if c.AppCert != "" {
			config["app_cert"] = c.AppCert
		}
		if c.RootCert != "" {
			config["root_cert"] = c.RootCert
		}
	}

	return config
}
