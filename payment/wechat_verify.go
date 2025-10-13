package payment

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"

	"github.com/go-pay/gopay/wechat/v3"
	"github.com/yaoapp/kun/log"
)

// NotifyRequest 微信支付异步通知请求结构
type NotifyRequest struct {
	ID           string                 `json:"id"`
	CreateTime   string                 `json:"create_time"`
	ResourceType string                 `json:"resource_type"`
	EventType    string                 `json:"event_type"`
	Summary      string                 `json:"summary"`
	Resource     *NotifyResource        `json:"resource"`
	Extra        map[string]interface{} `json:"-"`
}

// NotifyResource 微信支付异步通知资源结构
type NotifyResource struct {
	OriginalType   string `json:"original_type"`
	Algorithm      string `json:"algorithm"`
	Ciphertext     string `json:"ciphertext"`
	AssociatedData string `json:"associated_data"`
	Nonce          string `json:"nonce"`
}

// PayNotification 支付通知结构
type PayNotification struct {
	AppID              string `json:"appid"`
	MchID              string `json:"mchid"`
	OutTradeNo         string `json:"out_trade_no"`
	TransactionID      string `json:"transaction_id"`
	TradeType          string `json:"trade_type"`
	TradeState         string `json:"trade_state"`
	TradeStateDesc     string `json:"trade_state_desc"`
	BankType           string `json:"bank_type"`
	Attach             string `json:"attach"`
	SuccessTime        string `json:"success_time"`
	Payer              *Payer `json:"payer"`
	Amount             *Amount `json:"amount"`
	SceneInfo          *WeChatSceneInfo `json:"scene_info"`
	PromotionDetail    []PromotionDetail `json:"promotion_detail"`
}

// RefundNotification 退款通知结构
type RefundNotification struct {
	MchID               string `json:"mchid"`
	OutTradeNo          string `json:"out_trade_no"`
	TransactionID       string `json:"transaction_id"`
	OutRefundNo         string `json:"out_refund_no"`
	RefundID            string `json:"refund_id"`
	RefundStatus        string `json:"refund_status"`
	SuccessTime         string `json:"success_time"`
	RecvAccount         string `json:"recv_account"`
	UserReceivedAccount string `json:"user_received_account"`
	Amount              *RefundAmount `json:"amount"`
}

// Payer 支付者信息
type Payer struct {
	OpenID string `json:"openid"`
}

// Amount 金额信息
type Amount struct {
	Total         int    `json:"total"`
	PayerTotal    int    `json:"payer_total"`
	Currency      string `json:"currency"`
	PayerCurrency string `json:"payer_currency"`
}

// RefundAmount 退款金额信息
type RefundAmount struct {
	Total       int `json:"total"`
	Refund      int `json:"refund"`
	PayerTotal  int `json:"payer_total"`
	PayerRefund int `json:"payer_refund"`
}

// WeChatSceneInfo 微信验签场景信息
type WeChatSceneInfo struct {
	DeviceID string `json:"device_id"`
}

// PromotionDetail 优惠详情
type PromotionDetail struct {
	CouponID            string `json:"coupon_id"`
	Name                string `json:"name"`
	Scope               string `json:"scope"`
	Type                string `json:"type"`
	Amount              int    `json:"amount"`
	StockID             string `json:"stock_id"`
	WechatpayContribute int    `json:"wechatpay_contribute"`
	MerchantContribute  int    `json:"merchant_contribute"`
	OtherContribute     int    `json:"other_contribute"`
	Currency            string `json:"currency"`
	GoodsDetail         []GoodsDetail `json:"goods_detail"`
}

// GoodsDetail 商品详情
type GoodsDetail struct {
	GoodsID        string `json:"goods_id"`
	Quantity       int    `json:"quantity"`
	UnitPrice      int    `json:"unit_price"`
	DiscountAmount int    `json:"discount_amount"`
	GoodsRemark    string `json:"goods_remark"`
}

// VerifySignByPublicKey 使用微信支付公钥验证签名（同步返回验签）
func (w *WeChatV3Client) VerifySignByPublicKey(ctx context.Context, publicKey *rsa.PublicKey, signInfo *wechat.SignInfo) error {
	if publicKey == nil {
		return fmt.Errorf("微信支付公钥不能为空")
	}

	if signInfo == nil {
		return fmt.Errorf("签名信息不能为空")
	}

	// 使用公钥验证签名
	err := wechat.V3VerifySignByPK(signInfo.HeaderTimestamp, signInfo.HeaderNonce, signInfo.SignBody, signInfo.HeaderSignature, publicKey)
	if err != nil {
		log.Error("微信支付V3同步验签失败: %v", err)
		return fmt.Errorf("微信支付V3同步验签失败: %v", err)
	}

	log.Info("微信支付V3同步验签成功")
	return nil
}

// ParseNotify 解析异步通知
func (w *WeChatV3Client) ParseNotify(req *http.Request) (*wechat.V3NotifyReq, error) {
	if req == nil {
		return nil, fmt.Errorf("HTTP请求不能为空")
	}

	// 解析异步通知
	notifyReq, err := wechat.V3ParseNotify(req)
	if err != nil {
		log.Error("解析微信支付V3异步通知失败: %v", err)
		return nil, fmt.Errorf("解析微信支付V3异步通知失败: %v", err)
	}

	log.Info("解析微信支付V3异步通知成功")
	return notifyReq, nil
}

// VerifyNotifySign 验证异步通知签名
func (w *WeChatV3Client) VerifyNotifySign(req *http.Request) error {
	if req == nil {
		return fmt.Errorf("HTTP请求不能为空")
	}

	// 解析异步通知
	notifyReq, err := wechat.V3ParseNotify(req)
	if err != nil {
		log.Error("解析微信支付V3异步通知失败: %v", err)
		return fmt.Errorf("解析微信支付V3异步通知失败: %v", err)
	}

	// 获取微信平台证书
	certMap := w.client.WxPublicKeyMap()
	
	// 验证异步通知签名
	err = notifyReq.VerifySignByPKMap(certMap)
	if err != nil {
		log.Error("微信支付V3异步通知验签失败: %v", err)
		return fmt.Errorf("微信支付V3异步通知验签失败: %v", err)
	}

	log.Info("微信支付V3异步通知验签成功")
	return nil
}

// DecryptPayNotify 解密支付通知
func (w *WeChatV3Client) DecryptPayNotify(notifyReq *wechat.V3NotifyReq, apiV3Key string) (*PayNotification, error) {
	if notifyReq == nil {
		return nil, fmt.Errorf("异步通知请求不能为空")
	}

	if apiV3Key == "" {
		return nil, fmt.Errorf("API V3密钥不能为空")
	}

	// 解密支付通知
	decryptResult, err := notifyReq.DecryptPayCipherText(apiV3Key)
	if err != nil {
		log.Error("微信支付V3支付通知解密失败: %v", err)
		return nil, fmt.Errorf("微信支付V3支付通知解密失败: %v", err)
	}

	// 转换为我们的结构体
	result := &PayNotification{
		AppID:          decryptResult.Appid,
		MchID:          decryptResult.Mchid,
		OutTradeNo:     decryptResult.OutTradeNo,
		TransactionID:  decryptResult.TransactionId,
		TradeType:      decryptResult.TradeType,
		TradeState:     decryptResult.TradeState,
		TradeStateDesc: decryptResult.TradeStateDesc,
		BankType:       decryptResult.BankType,
		Attach:         decryptResult.Attach,
		SuccessTime:    decryptResult.SuccessTime,
	}

	log.Info("微信支付V3支付通知解密成功")
	return result, nil
}

// DecryptRefundNotify 解密退款通知
func (w *WeChatV3Client) DecryptRefundNotify(notifyReq *wechat.V3NotifyReq, apiV3Key string) (*RefundNotification, error) {
	if notifyReq == nil {
		return nil, fmt.Errorf("异步通知请求不能为空")
	}

	if apiV3Key == "" {
		return nil, fmt.Errorf("API V3密钥不能为空")
	}

	// 解密退款通知
	decryptResult, err := notifyReq.DecryptRefundCipherText(apiV3Key)
	if err != nil {
		log.Error("微信支付V3退款通知解密失败: %v", err)
		return nil, fmt.Errorf("微信支付V3退款通知解密失败: %v", err)
	}

	// 转换为我们的结构体
	result := &RefundNotification{
		MchID:         decryptResult.Mchid,
		OutTradeNo:    decryptResult.OutTradeNo,
		TransactionID: decryptResult.TransactionId,
		OutRefundNo:   decryptResult.OutRefundNo,
		RefundID:      decryptResult.RefundId,
		RefundStatus:  decryptResult.RefundStatus,
		SuccessTime:   decryptResult.SuccessTime,
	}

	log.Info("微信支付V3退款通知解密成功")
	return result, nil
}

// EncryptSensitiveInfo 加密敏感信息
func (w *WeChatV3Client) EncryptSensitiveInfo(ctx context.Context, plaintext string) (string, error) {
	if plaintext == "" {
		return "", fmt.Errorf("明文不能为空")
	}

	// 获取最新的微信支付平台证书
	serialNo, snCertMap, err := w.client.GetAndSelectNewestCert()
	if err != nil {
		log.Error("获取微信支付平台证书失败: %v", err)
		return "", fmt.Errorf("获取微信支付平台证书失败: %v", err)
	}

	// 使用最新证书的序列号获取证书内容
	var certContent string
	if cert, exists := snCertMap[serialNo]; exists {
		certContent = cert
	} else {
		return "", fmt.Errorf("未找到有效的平台证书")
	}

	// 使用全局函数加密敏感信息
	ciphertext, err := wechat.V3EncryptText(plaintext, []byte(certContent))
	if err != nil {
		log.Error("微信支付V3敏感信息加密失败: %v", err)
		return "", fmt.Errorf("微信支付V3敏感信息加密失败: %v", err)
	}

	log.Info("微信支付V3敏感信息加密成功")
	return ciphertext, nil
}

// DecryptSensitiveInfo 解密敏感信息
func (w *WeChatV3Client) DecryptSensitiveInfo(ctx context.Context, ciphertext, apiV3Key string) ([]byte, error) {
	if ciphertext == "" {
		return nil, fmt.Errorf("密文不能为空")
	}

	if apiV3Key == "" {
		return nil, fmt.Errorf("API V3密钥不能为空")
	}

	// 使用全局函数解密敏感信息
	plaintext, err := wechat.V3DecryptText(ciphertext, []byte(apiV3Key))
	if err != nil {
		log.Error("微信支付V3敏感信息解密失败: %v", err)
		return nil, fmt.Errorf("微信支付V3敏感信息解密失败: %v", err)
	}

	log.Info("微信支付V3敏感信息解密成功")
	return []byte(plaintext), nil
}

// GetPlatformCerts 获取微信支付平台证书
func (w *WeChatV3Client) GetPlatformCerts(ctx context.Context) (map[string]string, error) {
	// 获取微信支付平台证书
	_, snCertMap, err := w.client.GetAndSelectNewestCert()
	if err != nil {
		log.Error("获取微信支付平台证书失败: %v", err)
		return nil, fmt.Errorf("获取微信支付平台证书失败: %v", err)
	}

	log.Info("获取微信支付平台证书成功")
	return snCertMap, nil
}

// GetNewestCert 获取最新的微信支付平台证书
func (w *WeChatV3Client) GetNewestCert(ctx context.Context) (string, error) {
	// 获取最新的微信支付平台证书
	serialNo, snCertMap, err := w.client.GetAndSelectNewestCert()
	if err != nil {
		log.Error("获取最新微信支付平台证书失败: %v", err)
		return "", fmt.Errorf("获取最新微信支付平台证书失败: %v", err)
	}

	// 获取最新证书内容
	if cert, exists := snCertMap[serialNo]; exists {
		log.Info("获取最新微信支付平台证书成功")
		return cert, nil
	}

	return "", fmt.Errorf("未找到有效的平台证书")
}

// NotifyResponse 异步通知响应
func (w *WeChatV3Client) NotifyResponse() *wechat.V3NotifyRsp {
	return &wechat.V3NotifyRsp{Code: "SUCCESS", Message: "成功"}
}

// NotifyErrorResponse 异步通知错误响应
func (w *WeChatV3Client) NotifyErrorResponse(message string) *wechat.V3NotifyRsp {
	return &wechat.V3NotifyRsp{Code: "FAIL", Message: message}
}