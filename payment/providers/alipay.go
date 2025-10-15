package providers

import (
	"github.com/yaoapp/yao/payment/types"
	"context"
	"fmt"
	"strconv"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"
	"github.com/yaoapp/kun/log"
)

// AlipayProvider 支付宝支付提供商
type AlipayProvider struct {
	client *alipay.Client
	config *AlipayConfig
}

// AlipayConfig 支付宝配置
type AlipayConfig struct {
	AppID            string `json:"app_id"`              // 应用ID
	PrivateKey       string `json:"private_key"`         // 应用私钥
	PublicKey        string `json:"public_key"`          // 支付宝公钥
	IsSandbox        bool   `json:"is_sandbox"`          // 是否沙箱环境
	SignType         string `json:"sign_type"`           // 签名类型
	AppCertSN        string `json:"app_cert_sn"`         // 应用公钥证书SN
	AlipayCertSN     string `json:"alipay_cert_sn"`      // 支付宝公钥证书SN
	AlipayRootCertSN string `json:"alipay_root_cert_sn"` // 支付宝根证书SN
}

// NewAlipayProvider 创建支付宝支付提供商
func NewAlipayProvider(config map[string]interface{}) (types.PaymentProvider, error) {
	// 如果配置为空，返回默认实例（用于注册）
	if config == nil {
		return &AlipayProvider{}, nil
	}

	// 解析配置
	alipayConfig := &AlipayConfig{}
	if appID, ok := config["app_id"].(string); ok {
		alipayConfig.AppID = appID
	}
	if privateKey, ok := config["private_key"].(string); ok {
		alipayConfig.PrivateKey = privateKey
	}
	if publicKey, ok := config["public_key"].(string); ok {
		alipayConfig.PublicKey = publicKey
	}
	if isSandbox, ok := config["is_sandbox"].(bool); ok {
		alipayConfig.IsSandbox = isSandbox
	}
	if signType, ok := config["sign_type"].(string); ok {
		alipayConfig.SignType = signType
	}

	// 验证必要配置
	if alipayConfig.AppID == "" {
		return nil, fmt.Errorf("alipay app_id is required")
	}
	if alipayConfig.PrivateKey == "" {
		return nil, fmt.Errorf("alipay private_key is required")
	}
	if alipayConfig.PublicKey == "" {
		return nil, fmt.Errorf("alipay public_key is required")
	}

	// 设置默认值
	if alipayConfig.SignType == "" {
		alipayConfig.SignType = "RSA2"
	}

	// 创建支付宝客户端
	client, err := alipay.NewClient(alipayConfig.AppID, alipayConfig.PrivateKey, alipayConfig.IsSandbox)
	if err != nil {
		return nil, fmt.Errorf("create alipay client failed: %v", err)
	}

	// 设置支付宝公钥 - 使用自动验签功能
	if alipayConfig.PublicKey != "" {
		client.AutoVerifySign([]byte(alipayConfig.PublicKey))
	}

	// 设置证书SN（如果提供）
	if alipayConfig.AppCertSN != "" && alipayConfig.AlipayCertSN != "" && alipayConfig.AlipayRootCertSN != "" {
		client.SetCertSnByContent([]byte(alipayConfig.AppCertSN), []byte(alipayConfig.AlipayCertSN), []byte(alipayConfig.AlipayRootCertSN))
	}

	log.Info("Alipay provider initialized successfully")

	return &AlipayProvider{
		client: client,
		config: alipayConfig,
	}, nil
}

// CreateOrder 创建支付订单
func (ap *AlipayProvider) CreateOrder(params *types.CreateOrderParams) (*types.CreateOrderResponse, error) {
	ctx := context.Background()

	// 根据交易类型选择不同的支付方式
	switch types.TradeType(params.TradeType) {
	case types.TradeTypeNative:
		return ap.createNativeOrder(ctx, params)
	case types.TradeTypeWAP:
		return ap.createWAPOrder(ctx, params)
	case types.TradeTypeApp:
		return ap.createAppOrder(ctx, params)
	default:
		return &types.CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("不支持的交易类型: %s", params.TradeType),
		}, fmt.Errorf("unsupported trade type: %s", params.TradeType)
	}
}

// createNativeOrder 创建扫码支付订单
func (ap *AlipayProvider) createNativeOrder(ctx context.Context, params *types.CreateOrderParams) (*types.CreateOrderResponse, error) {
	// 构建请求参数
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", params.OutTradeNo)
	bm.Set("total_amount", fmt.Sprintf("%.2f", float64(params.Amount)/100))
	bm.Set("subject", params.Subject)

	if params.Body != "" {
		bm.Set("body", params.Body)
	}

	bm.Set("product_code", "FACE_TO_FACE_PAYMENT") // 默认扫码支付产品码

	// 设置过期时间
	if params.ExpireTime != "" {
		bm.Set("timeout_express", params.ExpireTime)
	}

	// 调用支付宝预下单接口
	aliRsp, err := ap.client.TradePrecreate(ctx, bm)
	if err != nil {
		return &types.CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("调用支付宝预下单接口失败: %v", err),
		}, err
	}

	if aliRsp.Response.Code != "10000" {
		return &types.CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("支付宝预下单失败: %s - %s", aliRsp.Response.Code, aliRsp.Response.Msg),
		}, fmt.Errorf("alipay precreate failed: %s", aliRsp.Response.Msg)
	}

	// 构建响应
	payInfo := map[string]interface{}{
		"out_trade_no": aliRsp.Response.OutTradeNo,
		"qr_code":      aliRsp.Response.QrCode,
	}

	return &types.CreateOrderResponse{
		Success:    true,
		OrderID:    aliRsp.Response.OutTradeNo, // 使用商户订单号作为订单ID
		OutTradeNo: aliRsp.Response.OutTradeNo,
		PayInfo:    payInfo,
		QRCode:     aliRsp.Response.QrCode,
		Message:    "订单创建成功",
	}, nil
}

// createWAPOrder 创建WAP支付订单
func (ap *AlipayProvider) createWAPOrder(ctx context.Context, params *types.CreateOrderParams) (*types.CreateOrderResponse, error) {
	// 构建请求参数
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", params.OutTradeNo)
	bm.Set("total_amount", fmt.Sprintf("%.2f", float64(params.Amount)/100))
	bm.Set("subject", params.Subject)
	bm.Set("product_code", "QUICK_WAP_WAY") // WAP支付产品码

	if params.Body != "" {
		bm.Set("body", params.Body)
	}

	if params.ReturnURL != "" {
		bm.Set("return_url", params.ReturnURL)
	}

	if params.NotifyURL != "" {
		bm.Set("notify_url", params.NotifyURL)
	}

	// 设置支付宝特有参数
	if params.AlipayParams != nil {
		if params.AlipayParams.TimeoutExpress != "" {
			bm.Set("timeout_express", params.AlipayParams.TimeoutExpress)
		}

		if params.AlipayParams.ExtendParams != "" {
			bm.Set("extend_params", params.AlipayParams.ExtendParams)
		}
	}

	// 设置过期时间
	if params.ExpireTime != "" {
		bm.Set("timeout_express", params.ExpireTime)
	}

	// 调用支付宝WAP支付接口
	payURL, err := ap.client.TradeWapPay(ctx, bm)
	if err != nil {
		return &types.CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("调用支付宝WAP支付接口失败: %v", err),
		}, err
	}

	// 构建响应
	payInfo := map[string]interface{}{
		"out_trade_no": params.OutTradeNo,
		"pay_url":      payURL,
	}

	return &types.CreateOrderResponse{
		Success:    true,
		OrderID:    params.OutTradeNo,
		OutTradeNo: params.OutTradeNo,
		PayInfo:    payInfo,
		PayURL:     payURL,
		Message:    "订单创建成功",
	}, nil
}

// createAppOrder 创建APP支付订单
func (ap *AlipayProvider) createAppOrder(ctx context.Context, params *types.CreateOrderParams) (*types.CreateOrderResponse, error) {
	// 构建请求参数
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", params.OutTradeNo)
	bm.Set("total_amount", fmt.Sprintf("%.2f", float64(params.Amount)/100))
	bm.Set("subject", params.Subject)
	bm.Set("product_code", "QUICK_MSECURITY_PAY") // APP支付产品码

	if params.Body != "" {
		bm.Set("body", params.Body)
	}

	if params.NotifyURL != "" {
		bm.Set("notify_url", params.NotifyURL)
	}

	// 设置支付宝特有参数
	if params.AlipayParams != nil {
		if params.AlipayParams.TimeoutExpress != "" {
			bm.Set("timeout_express", params.AlipayParams.TimeoutExpress)
		}

		if params.AlipayParams.ExtendParams != "" {
			bm.Set("extend_params", params.AlipayParams.ExtendParams)
		}
	}

	// 设置过期时间
	if params.ExpireTime != "" {
		bm.Set("timeout_express", params.ExpireTime)
	}

	// 调用支付宝APP支付接口
	payParam, err := ap.client.TradeAppPay(ctx, bm)
	if err != nil {
		return &types.CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("调用支付宝APP支付接口失败: %v", err),
		}, err
	}

	// 构建响应
	payInfo := map[string]interface{}{
		"out_trade_no": params.OutTradeNo,
		"pay_param":    payParam,
	}

	return &types.CreateOrderResponse{
		Success:    true,
		OrderID:    params.OutTradeNo,
		OutTradeNo: params.OutTradeNo,
		PayInfo:    payInfo,
		Message:    "订单创建成功",
	}, nil
}

// QueryOrder 查询订单状态
func (ap *AlipayProvider) QueryOrder(params *types.QueryOrderParams) (*types.QueryOrderResponse, error) {
	ctx := context.Background()

	// 构建请求参数
	bm := make(gopay.BodyMap)
	if params.OutTradeNo != "" {
		bm.Set("out_trade_no", params.OutTradeNo)
	}
	if params.TradeNo != "" {
		bm.Set("trade_no", params.TradeNo)
	}

	// 调用支付宝查询接口
	aliRsp, err := ap.client.TradeQuery(ctx, bm)
	if err != nil {
		return &types.QueryOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("调用支付宝查询接口失败: %v", err),
		}, err
	}

	if aliRsp.Response.Code != "10000" {
		return &types.QueryOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("支付宝查询失败: %s - %s", aliRsp.Response.Code, aliRsp.Response.Msg),
		}, fmt.Errorf("alipay query failed: %s", aliRsp.Response.Msg)
	}

	// 转换金额（元转分）
	amount, _ := strconv.ParseFloat(aliRsp.Response.TotalAmount, 64)
	paidAmount, _ := strconv.ParseFloat(aliRsp.Response.ReceiptAmount, 64)

	return &types.QueryOrderResponse{
		Success:    true,
		OutTradeNo: aliRsp.Response.OutTradeNo,
		TradeNo:    aliRsp.Response.TradeNo,
		Status:     convertAlipayOrderStatus(aliRsp.Response.TradeStatus),
		Amount:     int64(amount * 100),
		PaidAmount: int64(paidAmount * 100),
		PayTime:    aliRsp.Response.SendPayDate,
		Message:    "查询成功",
	}, nil
}

// CreateRefund 创建退款
func (ap *AlipayProvider) CreateRefund(params *types.CreateRefundParams) (*types.CreateRefundResponse, error) {
	ctx := context.Background()

	// 构建请求参数
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", params.OutTradeNo)
	bm.Set("out_request_no", params.OutRefundNo)
	bm.Set("refund_amount", fmt.Sprintf("%.2f", float64(params.RefundAmount)/100))

	if params.Reason != "" {
		bm.Set("refund_reason", params.Reason)
	}

	// 调用支付宝退款接口
	aliRsp, err := ap.client.TradeRefund(ctx, bm)
	if err != nil {
		return &types.CreateRefundResponse{
			Success: false,
			Error:   fmt.Sprintf("调用支付宝退款接口失败: %v", err),
		}, err
	}

	if aliRsp.Response.Code != "10000" {
		return &types.CreateRefundResponse{
			Success: false,
			Error:   fmt.Sprintf("支付宝退款失败: %s - %s", aliRsp.Response.Code, aliRsp.Response.Msg),
		}, fmt.Errorf("alipay refund failed: %s", aliRsp.Response.Msg)
	}

	return &types.CreateRefundResponse{
		Success:     true,
		OutRefundNo: params.OutRefundNo,
		RefundID:    params.OutRefundNo,  // 使用 OutRefundNo 作为 RefundID
		Status:      types.RefundStatusSuccess, // 支付宝退款是同步的
		Message:     "退款成功",
	}, nil
}

// QueryRefund 查询退款状态
func (ap *AlipayProvider) QueryRefund(params *types.QueryRefundParams) (*types.QueryRefundResponse, error) {
	ctx := context.Background()

	// 构建请求参数
	bm := make(gopay.BodyMap)

	// 支付宝退款查询需要订单号和退款单号
	if params.OutTradeNo != "" {
		bm.Set("out_trade_no", params.OutTradeNo)
	}
	if params.OutRefundNo != "" {
		bm.Set("out_request_no", params.OutRefundNo)
	}

	// 调用支付宝退款查询接口
	aliRsp, err := ap.client.TradeFastPayRefundQuery(ctx, bm)
	if err != nil {
		return &types.QueryRefundResponse{
			Success: false,
			Error:   fmt.Sprintf("调用支付宝退款查询接口失败: %v", err),
		}, err
	}

	if aliRsp.Response.Code != "10000" {
		return &types.QueryRefundResponse{
			Success: false,
			Error:   fmt.Sprintf("支付宝退款查询失败: %s - %s", aliRsp.Response.Code, aliRsp.Response.Msg),
		}, fmt.Errorf("alipay query refund failed: %s", aliRsp.Response.Msg)
	}

	// 转换退款金额（元转分）
	refundAmount, _ := strconv.ParseFloat(aliRsp.Response.RefundAmount, 64)

	return &types.QueryRefundResponse{
		Success:      true,
		OutRefundNo:  params.OutRefundNo,
		RefundID:     params.OutRefundNo,
		RefundAmount: int64(refundAmount * 100),
		Status:       types.RefundStatusSuccess, // 支付宝退款查询成功即表示退款已完成
		RefundTime:   aliRsp.Response.GmtRefundPay,
		Message:      "查询成功",
	}, nil
}

// HandleNotify 处理异步通知
func (ap *AlipayProvider) HandleNotify(params *types.HandleNotifyParams) (*types.HandleNotifyResponse, error) {
	// 检查是否传入了HTTP请求对象
	if params.Request == nil {
		return &types.HandleNotifyResponse{
			Success: false,
			Error:   "HTTP请求对象不能为空",
		}, fmt.Errorf("HTTP request is required")
	}

	// 解析支付宝通知参数
	notifyReq, err := alipay.ParseNotifyToBodyMap(params.Request)
	if err != nil {
		return &types.HandleNotifyResponse{
			Success: false,
			Error:   fmt.Sprintf("解析通知数据失败: %v", err),
		}, err
	}

	// 验证签名（使用支付宝公钥）
	ok, err := alipay.VerifySign(ap.config.PublicKey, notifyReq)
	if err != nil {
		return &types.HandleNotifyResponse{
			Success: false,
			Error:   fmt.Sprintf("签名验证失败: %v", err),
		}, err
	}
	if !ok {
		return &types.HandleNotifyResponse{
			Success: false,
			Error:   "签名验证不通过",
		}, fmt.Errorf("sign verification failed")
	}

	// 提取通知数据
	outTradeNo := notifyReq.GetString("out_trade_no")
	tradeNo := notifyReq.GetString("trade_no")
	tradeStatus := notifyReq.GetString("trade_status")
	totalAmount := notifyReq.GetString("total_amount")
	gmtPayment := notifyReq.GetString("gmt_payment")

	// 转换金额（元转分）
	amount, _ := strconv.ParseFloat(totalAmount, 64)

	// 转换为map用于返回
	notifyData := make(map[string]interface{})
	for k, v := range notifyReq {
		notifyData[k] = v
	}

	return &types.HandleNotifyResponse{
		Success:    true,
		OutTradeNo: outTradeNo,
		TradeNo:    tradeNo,
		Status:     convertAlipayOrderStatus(tradeStatus),
		Amount:     int64(amount * 100),
		PayTime:    gmtPayment,
		NotifyData: notifyData,
		Message:    "通知处理成功",
	}, nil
}

// DownloadBill 下载对账单
func (ap *AlipayProvider) DownloadBill(params *types.DownloadBillParams) (*types.DownloadBillResponse, error) {
	ctx := context.Background()

	// 构建请求参数
	bm := make(gopay.BodyMap)
	bm.Set("bill_type", params.BillType) // trade（商户基于支付宝交易收单的商户经营账单）
	bm.Set("bill_date", params.BillDate) // 格式：YYYY-MM-DD

	// 设置默认账单类型
	if params.BillType == "" {
		bm.Set("bill_type", "trade")
	}

	// 调用支付宝对账单下载接口
	aliRsp, err := ap.client.DataBillDownloadUrlQuery(ctx, bm)
	if err != nil {
		return &types.DownloadBillResponse{
			Success: false,
			Error:   fmt.Sprintf("调用支付宝对账单接口失败: %v", err),
		}, err
	}

	if aliRsp.Response.Code != "10000" {
		return &types.DownloadBillResponse{
			Success: false,
			Error:   fmt.Sprintf("支付宝获取对账单失败: %s - %s", aliRsp.Response.Code, aliRsp.Response.Msg),
		}, fmt.Errorf("alipay download bill failed: %s", aliRsp.Response.Msg)
	}

	// 返回对账单下载链接
	return &types.DownloadBillResponse{
		Success:  true,
		BillData: aliRsp.Response.BillDownloadUrl, // 返回下载链接
		Message:  "对账单获取成功",
	}, nil
}

// convertAlipayOrderStatus 转换支付宝订单状态
func convertAlipayOrderStatus(alipayStatus string) types.OrderStatus {
	switch alipayStatus {
	case "WAIT_BUYER_PAY":
		return types.OrderStatusPending
	case "TRADE_SUCCESS":
		return types.OrderStatusPaid
	case "TRADE_CLOSED":
		return types.OrderStatusClosed
	case "TRADE_FINISHED":
		return types.OrderStatusPaid
	default:
		return types.OrderStatusPending
	}
}
