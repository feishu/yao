package payment

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/paypal"
)

// PaymentProvider 支付提供商
type PaymentProvider string

const (
	ProviderWechat PaymentProvider = "wechat"
	ProviderAlipay PaymentProvider = "alipay"
	ProviderPayPal PaymentProvider = "paypal"
)

// PaymentStatus 支付状态
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "PENDING"
	PaymentStatusSuccess   PaymentStatus = "SUCCESS"
	PaymentStatusFailed    PaymentStatus = "FAILED"
	PaymentStatusCancelled PaymentStatus = "CANCELLED"
	PaymentStatusRefunded  PaymentStatus = "REFUNDED"
)

// PaymentConfig 支付配置
type PaymentConfig struct {
	Provider    PaymentProvider `json:"provider"`
	AppID       string          `json:"app_id"`
	AppSecret   string          `json:"app_secret"`
	MchID       string          `json:"mch_id"`
	APIKey      string          `json:"api_key"`
	PrivateKey  string          `json:"private_key"`  // 支付宝V3私钥
	IsProd      bool            `json:"is_prod"`
	NotifyURL   string          `json:"notify_url"`
	ReturnURL   string          `json:"return_url"`
}

// PaymentRequest 支付请求
type PaymentRequest struct {
	Provider    PaymentProvider `json:"provider"`
	AppID       string          `json:"app_id"`
	OutTradeNo  string          `json:"out_trade_no"`
	Amount      float64         `json:"amount"`      // 元为单位
	Subject     string          `json:"subject"`
	Body        string          `json:"body"`
	UserID      string          `json:"user_id"`     // 用户ID或OpenID
	NotifyURL   string          `json:"notify_url"`
	ReturnURL   string          `json:"return_url"`
}

// PaymentResponse 支付响应
type PaymentResponse struct {
	Success     bool          `json:"success"`
	TradeNo     string        `json:"trade_no"`
	OutTradeNo  string        `json:"out_trade_no"`
	Amount      string        `json:"amount"`
	Status      PaymentStatus `json:"status"`
	PayURL      string        `json:"pay_url"`
	CreatedTime string        `json:"created_time"`
}

// RefundRequest 退款请求
type RefundRequest struct {
	Provider     PaymentProvider `json:"provider"`
	OutTradeNo   string          `json:"out_trade_no"`
	RefundAmount float64         `json:"refund_amount"` // 元为单位
	Reason       string          `json:"reason"`
}

// RefundResponse 退款响应
type RefundResponse struct {
	Success      bool    `json:"success"`
	RefundNo     string  `json:"refund_no"`
	OutTradeNo   string  `json:"out_trade_no"`
	RefundAmount float64 `json:"refund_amount"`
	RefundStatus string  `json:"refund_status"`
}

// PaymentManager 支付管理器
type PaymentManager struct {
	configs map[PaymentProvider]*PaymentConfig
	clients map[PaymentProvider]interface{}
}

var manager *PaymentManager

// GetManager 获取支付管理器单例
func GetManager() *PaymentManager {
	if manager == nil {
		manager = &PaymentManager{
			configs: make(map[PaymentProvider]*PaymentConfig),
			clients: make(map[PaymentProvider]interface{}),
		}
	}
	return manager
}

// AddConfig 添加支付配置
func (pm *PaymentManager) AddConfig(config *PaymentConfig) error {
	pm.configs[config.Provider] = config
	return pm.initClient(config.Provider)
}

// initClient 初始化客户端
func (pm *PaymentManager) initClient(provider PaymentProvider) error {
	config := pm.configs[provider]
	if config == nil {
		return fmt.Errorf("config not found for provider: %s", provider)
	}

	switch provider {
	case ProviderWechat:
		// 初始化微信支付V3客户端
		wechatConfig := &WeChatV3Config{
			MchID:       config.MchID,
			APIv3Key:    config.APIKey,
			PrivateKey:  config.PrivateKey,
			IsProd:      config.IsProd,
			NotifyURL:   config.NotifyURL,
		}
		client, err := NewWeChatV3Client(wechatConfig)
		if err != nil {
			return fmt.Errorf("failed to create wechat client: %v", err)
		}
		pm.clients[provider] = client

	case ProviderAlipay:
		// 初始化支付宝V3客户端
		alipayConfig := &AliPayConfig{
			AppID:      config.AppID,
			PrivateKey: config.PrivateKey,
			IsProd:     config.IsProd,
			NotifyURL:  config.NotifyURL,
			ReturnURL:  config.ReturnURL,
		}
		client, err := NewAliPayV3Client(alipayConfig)
		if err != nil {
			return fmt.Errorf("failed to create alipay client: %v", err)
		}
		pm.clients[provider] = client

	case ProviderPayPal:
		// 初始化PayPal客户端
		client, err := paypal.NewClient(config.AppID, config.AppSecret, config.IsProd)
		if err != nil {
			return fmt.Errorf("failed to create paypal client: %v", err)
		}
		pm.clients[provider] = client

	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}

	return nil
}

// GetClient 获取客户端
func (pm *PaymentManager) GetClient(provider PaymentProvider) (interface{}, error) {
	client, exists := pm.clients[provider]
	if !exists {
		return nil, fmt.Errorf("client not found for provider: %s", provider)
	}
	return client, nil
}

// CreatePayment 创建支付订单
func (pm *PaymentManager) CreatePayment(req *PaymentRequest) (*PaymentResponse, error) {
	client, err := pm.GetClient(req.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	switch req.Provider {
	case ProviderWechat:
		return pm.createWechatPayment(ctx, client.(*WeChatV3Client), req)
	case ProviderAlipay:
		return pm.createAlipayPayment(ctx, client.(*AliPayV3Client), req)
	case ProviderPayPal:
		return pm.createPayPalPayment(ctx, client.(*paypal.Client), req)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", req.Provider)
	}
}

// createWechatPayment 创建微信支付订单
func (pm *PaymentManager) createWechatPayment(ctx context.Context, client *WeChatV3Client, req *PaymentRequest) (*PaymentResponse, error) {
	wxReq := &WeChatV3PayRequest{
		AppID:       req.AppID,
		OutTradeNo:  req.OutTradeNo,
		Description: req.Subject,
		TotalAmount: int64(req.Amount * 100), // 转换为分
		OpenID:      req.UserID,
		NotifyURL:   req.NotifyURL,
	}

	resp, err := client.JSAPIPayment(ctx, wxReq)
	if err != nil {
		return nil, err
	}

	return &PaymentResponse{
		Success:     resp.PrepayID != "",
		TradeNo:     resp.PrepayID,
		OutTradeNo:  req.OutTradeNo,
		PayURL:      resp.CodeURL,
		Amount:      fmt.Sprintf("%.2f", req.Amount),
		Status:      PaymentStatusPending,
		CreatedTime: time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

// createAlipayPayment 创建支付宝支付订单
func (pm *PaymentManager) createAlipayPayment(ctx context.Context, client *AliPayV3Client, req *PaymentRequest) (*PaymentResponse, error) {
	params := &TradePrecreateParams{
		OutTradeNo:  req.OutTradeNo,
		TotalAmount: fmt.Sprintf("%.2f", req.Amount),
		Subject:     req.Subject,
		Body:        req.Body,
	}

	resp, err := client.TradePrecreate(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("alipay trade precreate failed: %v", err)
	}

	return &PaymentResponse{
		Success:     resp.Code == "10000",
		TradeNo:     resp.QrCode,
		OutTradeNo:  req.OutTradeNo,
		PayURL:      resp.QrCode,
		Amount:      fmt.Sprintf("%.2f", req.Amount),
		Status:      PaymentStatusPending,
		CreatedTime: time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

// createPayPalPayment 创建PayPal支付订单
func (pm *PaymentManager) createPayPalPayment(ctx context.Context, client *paypal.Client, req *PaymentRequest) (*PaymentResponse, error) {
	bm := make(gopay.BodyMap)
	bm.Set("intent", "CAPTURE")
	bm.SetBodyMap("purchase_units", func(bm gopay.BodyMap) {
		bm.SetBodyMap("amount", func(bm gopay.BodyMap) {
			bm.Set("currency_code", "USD")
			bm.Set("value", fmt.Sprintf("%.2f", req.Amount))
		})
	})

	ppRsp, err := client.CreateOrder(ctx, bm)
	if err != nil {
		return nil, fmt.Errorf("create paypal order failed: %v", err)
	}

	return &PaymentResponse{
		Success:     ppRsp.Code == paypal.Success,
		TradeNo:     ppRsp.Response.Id,
		OutTradeNo:  req.OutTradeNo,
		Amount:      fmt.Sprintf("%.2f", req.Amount),
		Status:      PaymentStatusPending,
		CreatedTime: time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

// QueryPayment 查询支付订单
func (pm *PaymentManager) QueryPayment(provider PaymentProvider, outTradeNo string) (*PaymentResponse, error) {
	client, err := pm.GetClient(provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	switch provider {
	case ProviderWechat:
		return pm.queryWechatPayment(ctx, client.(*WeChatV3Client), outTradeNo)
	case ProviderAlipay:
		return pm.queryAlipayPayment(ctx, client.(*AliPayV3Client), outTradeNo)
	case ProviderPayPal:
		return pm.queryPayPalPayment(ctx, client.(*paypal.Client), outTradeNo)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// queryWechatPayment 查询微信支付订单
func (pm *PaymentManager) queryWechatPayment(ctx context.Context, client *WeChatV3Client, outTradeNo string) (*PaymentResponse, error) {
	resp, err := client.QueryOrder(ctx, outTradeNo)
	if err != nil {
		return nil, err
	}

	var status PaymentStatus
	switch resp.Response.TradeState {
	case "SUCCESS":
		status = PaymentStatusSuccess
	case "CLOSED":
		status = PaymentStatusCancelled
	default:
		status = PaymentStatusPending
	}

	totalAmount, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", float64(resp.Response.Amount.Total)/100), 64)

	return &PaymentResponse{
		Success:    resp.Response.TradeState == "SUCCESS",
		TradeNo:    resp.Response.TransactionId,
		OutTradeNo: outTradeNo,
		Amount:     fmt.Sprintf("%.2f", totalAmount),
		Status:     status,
	}, nil
}

// queryAlipayPayment 查询支付宝支付订单
func (pm *PaymentManager) queryAlipayPayment(ctx context.Context, client *AliPayV3Client, outTradeNo string) (*PaymentResponse, error) {
	params := &TradeQueryParams{
		OutTradeNo: outTradeNo,
	}

	resp, err := client.TradeQuery(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("alipay trade query failed: %v", err)
	}

	var status PaymentStatus
	switch resp.TradeStatus {
	case "TRADE_SUCCESS":
		status = PaymentStatusSuccess
	case "TRADE_CLOSED":
		status = PaymentStatusCancelled
	default:
		status = PaymentStatusPending
	}

	totalAmount, _ := strconv.ParseFloat(resp.TotalAmount, 64)

	return &PaymentResponse{
		Success:    resp.TradeStatus == "TRADE_SUCCESS",
		TradeNo:    resp.TradeNo,
		OutTradeNo: outTradeNo,
		Amount:     fmt.Sprintf("%.2f", totalAmount),
		Status:     status,
	}, nil
}

// queryPayPalPayment 查询PayPal支付订单
func (pm *PaymentManager) queryPayPalPayment(ctx context.Context, client *paypal.Client, orderID string) (*PaymentResponse, error) {
	ppRsp, err := client.OrderDetail(ctx, orderID, nil)
	if err != nil {
		return nil, fmt.Errorf("query paypal order failed: %v", err)
	}

	var status PaymentStatus
	switch ppRsp.Response.Status {
	case "COMPLETED":
		status = PaymentStatusSuccess
	case "CANCELLED":
		status = PaymentStatusCancelled
	default:
		status = PaymentStatusPending
	}

	return &PaymentResponse{
		Success:    ppRsp.Response.Status == "COMPLETED",
		TradeNo:    ppRsp.Response.Id,
		OutTradeNo: orderID,
		Status:     status,
	}, nil
}

// RefundPayment 申请退款
func (pm *PaymentManager) RefundPayment(req *RefundRequest) (*RefundResponse, error) {
	client, err := pm.GetClient(req.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	switch req.Provider {
	case ProviderWechat:
		return pm.refundWechatPayment(ctx, client.(*WeChatV3Client), req)
	case ProviderAlipay:
		return pm.refundAlipayPayment(ctx, client.(*AliPayV3Client), req)
	case ProviderPayPal:
		return pm.refundPayPalPayment(ctx, client.(*paypal.Client), req)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", req.Provider)
	}
}

// refundWechatPayment 微信支付退款
func (pm *PaymentManager) refundWechatPayment(ctx context.Context, client *WeChatV3Client, req *RefundRequest) (*RefundResponse, error) {
	refundNo := fmt.Sprintf("refund_%s_%d", req.OutTradeNo, time.Now().Unix())
	
	resp, err := client.Refund(ctx, req.OutTradeNo, refundNo, int64(req.RefundAmount*100), int64(req.RefundAmount*100), req.Reason)
	if err != nil {
		return nil, err
	}

	return &RefundResponse{
		Success:      resp.Response.Status == "SUCCESS",
		RefundNo:     resp.Response.OutRefundNo,
		OutTradeNo:   req.OutTradeNo,
		RefundAmount: req.RefundAmount,
		RefundStatus: resp.Response.Status,
	}, nil
}

// refundAlipayPayment 支付宝退款
func (pm *PaymentManager) refundAlipayPayment(ctx context.Context, client *AliPayV3Client, req *RefundRequest) (*RefundResponse, error) {
	params := &TradeRefundParams{
		OutTradeNo:   req.OutTradeNo,
		RefundAmount: fmt.Sprintf("%.2f", req.RefundAmount),
		RefundReason: req.Reason,
		OutRequestNo: fmt.Sprintf("refund_%s_%d", req.OutTradeNo, time.Now().Unix()),
	}

	resp, err := client.TradeRefund(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("alipay trade refund failed: %v", err)
	}

	return &RefundResponse{
		Success:      resp.Code == "10000",
		RefundNo:     resp.TradeNo,
		OutTradeNo:   req.OutTradeNo,
		RefundAmount: req.RefundAmount,
		RefundStatus: "SUCCESS",
	}, nil
}

// refundPayPalPayment PayPal退款
func (pm *PaymentManager) refundPayPalPayment(ctx context.Context, client *paypal.Client, req *RefundRequest) (*RefundResponse, error) {
	// 首先获取订单详情
	orderRsp, err := client.OrderDetail(ctx, req.OutTradeNo, nil)
	if err != nil {
		return nil, fmt.Errorf("get paypal order detail failed: %v", err)
	}

	// 获取capture ID进行退款
	if len(orderRsp.Response.PurchaseUnits) == 0 || len(orderRsp.Response.PurchaseUnits[0].Payments.Captures) == 0 {
		return nil, fmt.Errorf("no capture found for order: %s", req.OutTradeNo)
	}

	captureID := orderRsp.Response.PurchaseUnits[0].Payments.Captures[0].Id

	bm := make(gopay.BodyMap)
	bm.SetBodyMap("amount", func(bm gopay.BodyMap) {
		bm.Set("currency_code", "USD")
		bm.Set("value", fmt.Sprintf("%.2f", req.RefundAmount))
	})

	refundRsp, err := client.PaymentCaptureRefund(ctx, captureID, bm)
	if err != nil {
		return nil, fmt.Errorf("paypal refund failed: %v", err)
	}

	return &RefundResponse{
		Success:      refundRsp.Code == paypal.Success,
		RefundNo:     refundRsp.Response.Id,
		OutTradeNo:   req.OutTradeNo,
		RefundAmount: req.RefundAmount,
		RefundStatus: refundRsp.Response.Status,
	}, nil
}

// GetSupportedProviders 获取支持的支付提供商
func GetSupportedProviders() []PaymentProvider {
	return []PaymentProvider{ProviderWechat, ProviderAlipay, ProviderPayPal}
}

// ValidateNotify 验证支付通知
func (pm *PaymentManager) ValidateNotify(provider PaymentProvider, data map[string]interface{}) (bool, error) {
	switch provider {
	case ProviderWechat:
		// 微信支付通知验证逻辑
		return true, nil
	case ProviderAlipay:
		// 支付宝通知验证逻辑
		return true, nil
	case ProviderPayPal:
		// PayPal通知验证逻辑
		return true, nil
	default:
		return false, fmt.Errorf("unsupported provider: %s", provider)
	}
}