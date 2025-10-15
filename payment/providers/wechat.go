package providers

import (
	"context"
	"fmt"

	"github.com/yaoapp/yao/payment/types"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/wechat/v3"
	"github.com/yaoapp/kun/log"
)

// WechatProvider 微信支付提供商
type WechatProvider struct {
	client *wechat.ClientV3
	config *WechatConfig
}

// WechatConfig 微信支付配置
type WechatConfig struct {
	AppID      string `json:"app_id"`      // 应用ID
	MchID      string `json:"mch_id"`      // 商户号
	APIv3Key   string `json:"apiv3_key"`   // APIv3密钥
	PrivateKey string `json:"private_key"` // 商户私钥
	SerialNo   string `json:"serial_no"`   // 商户证书序列号
	PublicKey  string `json:"public_key"`  // 微信支付公钥
	IsSandbox  bool   `json:"is_sandbox"`  // 是否沙箱环境
}

// NewWechatProvider 创建微信支付提供商
func NewWechatProvider(config map[string]interface{}) (types.PaymentProvider, error) {
	// 如果配置为空，返回默认实例
	if config == nil {
		return &WechatProvider{}, nil
	}

	// 解析配置参数
	wechatConfig := &WechatConfig{}

	if appID, ok := config["app_id"].(string); ok {
		wechatConfig.AppID = appID
	}
	if mchID, ok := config["mch_id"].(string); ok {
		wechatConfig.MchID = mchID
	}
	if apiv3Key, ok := config["apiv3_key"].(string); ok {
		wechatConfig.APIv3Key = apiv3Key
	}
	if privateKey, ok := config["private_key"].(string); ok {
		wechatConfig.PrivateKey = privateKey
	}
	if serialNo, ok := config["serial_no"].(string); ok {
		wechatConfig.SerialNo = serialNo
	}
	if publicKey, ok := config["public_key"].(string); ok {
		wechatConfig.PublicKey = publicKey
	}
	if isSandbox, ok := config["is_sandbox"].(bool); ok {
		wechatConfig.IsSandbox = isSandbox
	}

	// 验证必要配置
	if wechatConfig.AppID == "" {
		return nil, fmt.Errorf("wechat app_id is required")
	}
	if wechatConfig.MchID == "" {
		return nil, fmt.Errorf("wechat mch_id is required")
	}
	if wechatConfig.APIv3Key == "" {
		return nil, fmt.Errorf("wechat apiv3_key is required")
	}
	if wechatConfig.PrivateKey == "" {
		return nil, fmt.Errorf("wechat private_key is required")
	}
	if wechatConfig.SerialNo == "" {
		return nil, fmt.Errorf("wechat serial_no is required")
	}

	// 创建微信支付客户端
	client, err := wechat.NewClientV3(wechatConfig.MchID, wechatConfig.SerialNo, wechatConfig.APIv3Key, wechatConfig.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("create wechat client failed: %v", err)
	}

	// 设置沙箱环境
	if wechatConfig.IsSandbox {
		client.DebugSwitch = gopay.DebugOn
	}

	log.Info("Wechat provider initialized successfully")

	return &WechatProvider{
		client: client,
		config: wechatConfig,
	}, nil
}

// CreateOrder 创建支付订单
func (wp *WechatProvider) CreateOrder(params *types.CreateOrderParams) (*types.CreateOrderResponse, error) {
	ctx := context.Background()

	// 根据交易类型调用不同的创建方法
	switch types.TradeType(params.TradeType) {
	case types.TradeTypeJSAPI:
		return wp.createJSAPIOrder(ctx, params)
	case types.TradeTypeNative:
		return wp.createNativeOrder(ctx, params)
	case types.TradeTypeApp:
		return wp.createAppOrder(ctx, params)
	case types.TradeTypeH5:
		return wp.createH5Order(ctx, params)
	default:
		return &types.CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("不支持的交易类型: %s", params.TradeType),
		}, fmt.Errorf("unsupported trade type: %s", params.TradeType)
	}
}

// createJSAPIOrder 创建JSAPI支付订单
func (wp *WechatProvider) createJSAPIOrder(ctx context.Context, params *types.CreateOrderParams) (*types.CreateOrderResponse, error) {

	// 构建请求参数
	bm := make(gopay.BodyMap)
	bm.Set("appid", wp.config.AppID).
		Set("mchid", wp.config.MchID).
		Set("description", params.Subject).
		Set("out_trade_no", params.OutTradeNo).
		Set("notify_url", params.NotifyURL)

	// 设置金额信息
	bm.SetBodyMap("amount", func(bm gopay.BodyMap) {
		bm.Set("total", params.Amount)
		bm.Set("currency", "CNY")
	})

	// 设置支付者信息
	if params.WechatParams != nil && params.WechatParams.OpenID != "" {
		bm.SetBodyMap("payer", func(bm gopay.BodyMap) {
			bm.Set("openid", params.WechatParams.OpenID)
		})
	}

	// 调用微信JSAPI支付接口
	wxRsp, err := wp.client.V3TransactionJsapi(ctx, bm)
	if err != nil {
		return &types.CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("调用微信JSAPI支付接口失败: %v", err),
		}, err
	}

	if wxRsp.Code != wechat.Success {
		return &types.CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("微信JSAPI支付失败: %s", wxRsp.Error),
		}, fmt.Errorf("wechat jsapi payment failed: %s", wxRsp.Error)
	}

	// 生成JSAPI支付参数
	jsapiParams, err := wp.client.PaySignOfJSAPI(wp.config.AppID, wxRsp.Response.PrepayId)
	if err != nil {
		return &types.CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("生成JSAPI支付参数失败: %v", err),
		}, err
	}

	// 转换为 map[string]interface{}
	payInfo := map[string]interface{}{
		"appId":     jsapiParams.AppId,
		"timeStamp": jsapiParams.TimeStamp,
		"nonceStr":  jsapiParams.NonceStr,
		"package":   jsapiParams.Package,
		"signType":  jsapiParams.SignType,
		"paySign":   jsapiParams.PaySign,
	}

	return &types.CreateOrderResponse{
		Success:    true,
		OrderID:    params.OutTradeNo,
		OutTradeNo: params.OutTradeNo,
		PayInfo:    payInfo,
		Message:    "订单创建成功",
	}, nil
}

// createNativeOrder 创建Native支付订单
func (wp *WechatProvider) createNativeOrder(ctx context.Context, params *types.CreateOrderParams) (*types.CreateOrderResponse, error) {
	// 构建请求参数
	bm := make(gopay.BodyMap)
	bm.Set("appid", wp.config.AppID)
	bm.Set("mchid", wp.config.MchID)
	bm.Set("description", params.Subject)
	bm.Set("out_trade_no", params.OutTradeNo)
	bm.Set("notify_url", params.NotifyURL)

	// 设置金额信息
	bm.SetBodyMap("amount", func(bm gopay.BodyMap) {
		bm.Set("total", params.Amount)
		bm.Set("currency", "CNY")
	})

	// 调用微信Native支付接口
	wxRsp, err := wp.client.V3TransactionNative(ctx, bm)
	if err != nil {
		return &types.CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("调用微信Native支付接口失败: %v", err),
		}, err
	}

	if wxRsp.Code != wechat.Success {
		return &types.CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("微信Native支付失败: %s", wxRsp.Error),
		}, fmt.Errorf("wechat native payment failed: %s", wxRsp.Error)
	}

	return &types.CreateOrderResponse{
		Success:    true,
		OrderID:    params.OutTradeNo,
		OutTradeNo: params.OutTradeNo,
		PayInfo: map[string]interface{}{
			"code_url": wxRsp.Response.CodeUrl,
		},
		Message: "订单创建成功",
	}, nil
}

// createAppOrder 创建APP支付订单
func (wp *WechatProvider) createAppOrder(ctx context.Context, params *types.CreateOrderParams) (*types.CreateOrderResponse, error) {
	// 构建请求参数
	bm := make(gopay.BodyMap)
	bm.Set("appid", wp.config.AppID)
	bm.Set("mchid", wp.config.MchID)
	bm.Set("description", params.Subject)
	bm.Set("out_trade_no", params.OutTradeNo)
	bm.Set("notify_url", params.NotifyURL)

	// 设置金额信息
	bm.SetBodyMap("amount", func(bm gopay.BodyMap) {
		bm.Set("total", params.Amount)
		bm.Set("currency", "CNY")
	})

	// 调用微信APP支付接口
	wxRsp, err := wp.client.V3TransactionApp(ctx, bm)
	if err != nil {
		return &types.CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("调用微信APP支付接口失败: %v", err),
		}, err
	}

	if wxRsp.Code != wechat.Success {
		return &types.CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("微信APP支付失败: %s", wxRsp.Error),
		}, fmt.Errorf("wechat app payment failed: %s", wxRsp.Error)
	}

	// 生成APP支付参数
	appParams, err := wp.client.PaySignOfApp(wp.config.AppID, wxRsp.Response.PrepayId)
	if err != nil {
		return &types.CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("生成APP支付参数失败: %v", err),
		}, err
	}

	// 转换为 map[string]interface{}
	payInfo := map[string]interface{}{
		"appid":     appParams.Appid,
		"partnerid": appParams.Partnerid,
		"prepayid":  appParams.Prepayid,
		"package":   appParams.Package,
		"noncestr":  appParams.Noncestr,
		"timestamp": appParams.Timestamp,
		"sign":      appParams.Sign,
	}

	return &types.CreateOrderResponse{
		Success:    true,
		OrderID:    params.OutTradeNo,
		OutTradeNo: params.OutTradeNo,
		PayInfo:    payInfo,
		Message:    "订单创建成功",
	}, nil
}

// createH5Order 创建H5支付订单
func (wp *WechatProvider) createH5Order(ctx context.Context, params *types.CreateOrderParams) (*types.CreateOrderResponse, error) {
	// H5支付需要场景信息
	var sceneInfo map[string]interface{}
	if params.WechatParams != nil && params.WechatParams.SceneInfo != nil {
		// 直接使用场景信息map
		sceneInfo = params.WechatParams.SceneInfo
	}

	if sceneInfo == nil {
		sceneInfo = map[string]interface{}{
			"payer_client_ip": "127.0.0.1",
			"h5_info": map[string]interface{}{
				"type": "Wap",
			},
		}
	}

	// 构建请求参数
	bm := make(gopay.BodyMap)
	bm.Set("appid", wp.config.AppID)
	bm.Set("mchid", wp.config.MchID)
	bm.Set("description", params.Subject)
	bm.Set("out_trade_no", params.OutTradeNo)
	bm.Set("notify_url", params.NotifyURL)

	// 设置金额信息
	bm.SetBodyMap("amount", func(bm gopay.BodyMap) {
		bm.Set("total", params.Amount)
		bm.Set("currency", "CNY")
	})

	// 设置场景信息
	bm.SetBodyMap("scene_info", func(bm gopay.BodyMap) {
		if payerClientIP, ok := sceneInfo["payer_client_ip"].(string); ok {
			bm.Set("payer_client_ip", payerClientIP)
		}
		if h5Info, ok := sceneInfo["h5_info"].(map[string]interface{}); ok {
			bm.SetBodyMap("h5_info", func(bm gopay.BodyMap) {
				if typ, ok := h5Info["type"].(string); ok {
					bm.Set("type", typ)
				}
				if appName, ok := h5Info["app_name"].(string); ok {
					bm.Set("app_name", appName)
				}
				if appURL, ok := h5Info["app_url"].(string); ok {
					bm.Set("app_url", appURL)
				}
			})
		}
	})

	// 调用微信H5支付接口
	wxRsp, err := wp.client.V3TransactionH5(ctx, bm)
	if err != nil {
		return &types.CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("调用微信H5支付接口失败: %v", err),
		}, err
	}

	if wxRsp.Code != wechat.Success {
		return &types.CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("微信H5支付失败: %s", wxRsp.Error),
		}, fmt.Errorf("wechat h5 payment failed: %s", wxRsp.Error)
	}

	return &types.CreateOrderResponse{
		Success:    true,
		OrderID:    params.OutTradeNo,
		OutTradeNo: params.OutTradeNo,
		PayInfo: map[string]interface{}{
			"h5_url": wxRsp.Response.H5Url,
		},
		Message: "订单创建成功",
	}, nil
}

// QueryOrder 查询支付订单
func (wp *WechatProvider) QueryOrder(params *types.QueryOrderParams) (*types.QueryOrderResponse, error) {
	ctx := context.Background()

	// 微信支付V3支持使用商户订单号或微信支付订单号查询
	var wxRsp *wechat.QueryOrderRsp
	var err error

	if params.OutTradeNo != "" {
		// 使用商户订单号查询
		wxRsp, err = wp.client.V3TransactionQueryOrder(ctx, wechat.OutTradeNo, params.OutTradeNo)
	} else if params.TransactionID != "" {
		// 使用微信支付订单号查询
		wxRsp, err = wp.client.V3TransactionQueryOrder(ctx, wechat.TransactionId, params.TransactionID)
	} else {
		return &types.QueryOrderResponse{
			Success: false,
			Error:   "out_trade_no 或 transaction_id 必须提供一个",
		}, fmt.Errorf("out_trade_no or transaction_id is required")
	}

	if err != nil {
		return &types.QueryOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("查询订单失败: %v", err),
		}, err
	}

	if wxRsp.Code != wechat.Success {
		return &types.QueryOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("微信查询订单失败: %s", wxRsp.Error),
		}, fmt.Errorf("wechat query order failed: %s", wxRsp.Error)
	}

	// 转换订单状态
	status := convertWechatOrderStatus(wxRsp.Response.TradeState)

	// 获取支付时间
	var paidAt string
	if wxRsp.Response.SuccessTime != "" {
		paidAt = wxRsp.Response.SuccessTime
	}

	return &types.QueryOrderResponse{
		Success:       true,
		OrderID:       wxRsp.Response.OutTradeNo,
		OutTradeNo:    wxRsp.Response.OutTradeNo,
		TransactionID: wxRsp.Response.TransactionId,
		TradeNo:       wxRsp.Response.TransactionId,
		Channel:       types.ChannelWechat,
		TradeType:     types.TradeType(wxRsp.Response.TradeType),
		Status:        status,
		Amount:        int64(wxRsp.Response.Amount.Total),
		PaidAmount:    int64(wxRsp.Response.Amount.PayerTotal),
		PaidAt:        paidAt,
		PayTime:       paidAt,
		CreatedAt:     "",
		Message:       "查询成功",
	}, nil
}

// convertWechatOrderStatus 转换微信订单状态
func convertWechatOrderStatus(wechatStatus string) types.OrderStatus {
	switch wechatStatus {
	case "SUCCESS":
		return types.OrderStatusPaid
	case "REFUND":
		return types.OrderStatusRefund
	case "NOTPAY":
		return types.OrderStatusPending
	case "CLOSED":
		return types.OrderStatusClosed
	case "REVOKED":
		return types.OrderStatusClosed
	case "USERPAYING":
		return types.OrderStatusPending
	case "PAYERROR":
		return types.OrderStatusClosed
	default:
		return types.OrderStatusPending
	}
}

// CreateRefund 创建退款
func (wp *WechatProvider) CreateRefund(params *types.CreateRefundParams) (*types.CreateRefundResponse, error) {
	ctx := context.Background()

	// 构建请求参数
	bm := make(gopay.BodyMap)

	// 设置订单号（微信支持使用商户订单号或微信订单号）
	if params.OutTradeNo != "" {
		bm.Set("out_trade_no", params.OutTradeNo)
	} else if params.TransactionID != "" {
		bm.Set("transaction_id", params.TransactionID)
	}

	bm.Set("out_refund_no", params.OutRefundNo)

	// 设置退款原因
	if params.Reason != "" {
		bm.Set("reason", params.Reason)
	} else if params.RefundReason != "" {
		bm.Set("reason", params.RefundReason)
	}

	// 设置退款通知URL
	if params.NotifyURL != "" {
		bm.Set("notify_url", params.NotifyURL)
	}

	// 设置金额信息
	bm.SetBodyMap("amount", func(bm gopay.BodyMap) {
		bm.Set("refund", params.RefundAmount) // 退款金额（分）
		bm.Set("total", params.TotalAmount)   // 原订单金额（分）
		bm.Set("currency", "CNY")             // 货币类型
	})

	// 调用微信退款接口
	wxRsp, err := wp.client.V3Refund(ctx, bm)
	if err != nil {
		return &types.CreateRefundResponse{
			Success: false,
			Error:   fmt.Sprintf("调用微信退款接口失败: %v", err),
		}, err
	}

	if wxRsp.Code != wechat.Success {
		return &types.CreateRefundResponse{
			Success: false,
			Error:   fmt.Sprintf("微信退款失败: %s", wxRsp.Error),
		}, fmt.Errorf("wechat refund failed: %s", wxRsp.Error)
	}

	// 转换退款状态
	status := convertWechatRefundStatus(wxRsp.Response.Status)

	return &types.CreateRefundResponse{
		Success:      true,
		RefundID:     wxRsp.Response.RefundId,
		OutRefundNo:  wxRsp.Response.OutRefundNo,
		RefundAmount: int64(wxRsp.Response.Amount.Refund),
		Status:       status,
		Message:      "退款申请成功",
	}, nil
}

// convertWechatRefundStatus 转换微信退款状态
func convertWechatRefundStatus(wechatStatus string) types.RefundStatus {
	switch wechatStatus {
	case "SUCCESS":
		return types.RefundStatusSuccess
	case "CLOSED":
		return types.RefundStatusClosed
	case "PROCESSING":
		return types.RefundStatusProcessing
	case "ABNORMAL":
		return types.RefundStatusAbnormal
	default:
		return types.RefundStatusPending
	}
}

// QueryRefund 查询退款
func (wp *WechatProvider) QueryRefund(params *types.QueryRefundParams) (*types.QueryRefundResponse, error) {
	ctx := context.Background()

	// 使用商户退款单号查询
	if params.OutRefundNo == "" {
		return &types.QueryRefundResponse{
			Success: false,
			Error:   "out_refund_no 不能为空",
		}, fmt.Errorf("out_refund_no is required")
	}

	// 调用微信查询退款接口（需要传入空的 BodyMap）
	wxRsp, err := wp.client.V3RefundQuery(ctx, params.OutRefundNo, nil)
	if err != nil {
		return &types.QueryRefundResponse{
			Success: false,
			Error:   fmt.Sprintf("查询退款失败: %v", err),
		}, err
	}

	if wxRsp.Code != wechat.Success {
		return &types.QueryRefundResponse{
			Success: false,
			Error:   fmt.Sprintf("微信查询退款失败: %s", wxRsp.Error),
		}, fmt.Errorf("wechat query refund failed: %s", wxRsp.Error)
	}

	// 转换退款状态
	status := convertWechatRefundStatus(wxRsp.Response.Status)

	// 获取退款时间
	var refundTime string
	if wxRsp.Response.SuccessTime != "" {
		refundTime = wxRsp.Response.SuccessTime
	} else if wxRsp.Response.CreateTime != "" {
		refundTime = wxRsp.Response.CreateTime
	}

	return &types.QueryRefundResponse{
		Success:      true,
		RefundID:     wxRsp.Response.RefundId,
		OutRefundNo:  wxRsp.Response.OutRefundNo,
		RefundAmount: int64(wxRsp.Response.Amount.Refund),
		Status:       status,
		RefundTime:   refundTime,
		Message:      "查询成功",
	}, nil
}

// HandleNotify 处理支付通知
func (wp *WechatProvider) HandleNotify(params *types.HandleNotifyParams) (*types.HandleNotifyResponse, error) {
	// 检查是否传入了HTTP请求对象
	if params.Request == nil {
		return &types.HandleNotifyResponse{
			Success: false,
			Error:   "HTTP请求对象不能为空",
		}, fmt.Errorf("HTTP request is required")
	}

	// 解析并验证微信通知签名
	notifyReq, err := wechat.V3ParseNotify(params.Request)
	if err != nil {
		return &types.HandleNotifyResponse{
			Success: false,
			Error:   fmt.Sprintf("解析通知数据失败: %v", err),
		}, err
	}

	// 从客户端获取微信平台证书公钥映射进行验签
	if err = notifyReq.VerifySignByPKMap(wp.client.WxPublicKeyMap()); err != nil {
		return &types.HandleNotifyResponse{
			Success: false,
			Error:   fmt.Sprintf("签名验证失败: %v", err),
		}, err
	}

	// 根据通知类型处理
	switch notifyReq.EventType {
	case "TRANSACTION.SUCCESS": // 支付成功通知
		return wp.handlePaymentNotify(notifyReq)
	case "REFUND.SUCCESS", "REFUND.ABNORMAL", "REFUND.CLOSED": // 退款通知
		return wp.handleRefundNotify(notifyReq)
	default:
		return &types.HandleNotifyResponse{
			Success: false,
			Error:   fmt.Sprintf("不支持的通知类型: %s", notifyReq.EventType),
		}, fmt.Errorf("unsupported event type: %s", notifyReq.EventType)
	}
}

// handlePaymentNotify 处理支付成功通知
func (wp *WechatProvider) handlePaymentNotify(notifyReq *wechat.V3NotifyReq) (*types.HandleNotifyResponse, error) {
	// 解密支付通知数据
	payResult, err := notifyReq.DecryptPayCipherText(wp.config.APIv3Key)
	if err != nil {
		return &types.HandleNotifyResponse{
			Success: false,
			Error:   fmt.Sprintf("解密支付通知数据失败: %v", err),
		}, err
	}

	// 构建通知数据
	notifyData := map[string]interface{}{
		"appid":            payResult.Appid,
		"mchid":            payResult.Mchid,
		"out_trade_no":     payResult.OutTradeNo,
		"transaction_id":   payResult.TransactionId,
		"trade_type":       payResult.TradeType,
		"trade_state":      payResult.TradeState,
		"trade_state_desc": payResult.TradeStateDesc,
		"bank_type":        payResult.BankType,
		"success_time":     payResult.SuccessTime,
	}

	// 提取金额信息
	var amount int64
	if payResult.Amount != nil {
		amount = int64(payResult.Amount.Total)
		notifyData["amount"] = map[string]interface{}{
			"total":       payResult.Amount.Total,
			"payer_total": payResult.Amount.PayerTotal,
			"currency":    payResult.Amount.Currency,
		}
	}

	return &types.HandleNotifyResponse{
		Success:    true,
		OutTradeNo: payResult.OutTradeNo,
		TradeNo:    payResult.TransactionId,
		Status:     convertWechatOrderStatus(payResult.TradeState),
		Amount:     amount,
		PayTime:    payResult.SuccessTime,
		NotifyData: notifyData,
		Message:    "支付通知处理成功",
	}, nil
}

// handleRefundNotify 处理退款通知
func (wp *WechatProvider) handleRefundNotify(notifyReq *wechat.V3NotifyReq) (*types.HandleNotifyResponse, error) {
	// 解密退款通知数据
	refundResult, err := notifyReq.DecryptRefundCipherText(wp.config.APIv3Key)
	if err != nil {
		return &types.HandleNotifyResponse{
			Success: false,
			Error:   fmt.Sprintf("解密退款通知数据失败: %v", err),
		}, err
	}

	// 构建通知数据
	notifyData := map[string]interface{}{
		"mchid":                 refundResult.Mchid,
		"out_trade_no":          refundResult.OutTradeNo,
		"transaction_id":        refundResult.TransactionId,
		"out_refund_no":         refundResult.OutRefundNo,
		"refund_id":             refundResult.RefundId,
		"refund_status":         refundResult.RefundStatus,
		"success_time":          refundResult.SuccessTime,
		"user_received_account": refundResult.UserReceivedAccount,
	}

	// 提取金额信息
	var refundAmount int64
	if refundResult.Amount != nil {
		refundAmount = int64(refundResult.Amount.Refund)
		notifyData["amount"] = map[string]interface{}{
			"total":        refundResult.Amount.Total,
			"refund":       refundResult.Amount.Refund,
			"payer_total":  refundResult.Amount.PayerTotal,
			"payer_refund": refundResult.Amount.PayerRefund,
		}
	}

	return &types.HandleNotifyResponse{
		Success:    true,
		OutTradeNo: refundResult.OutTradeNo,
		Status:     types.OrderStatus(refundResult.RefundStatus),
		Amount:     refundAmount,
		PayTime:    refundResult.SuccessTime,
		NotifyData: notifyData,
		Extra: map[string]interface{}{
			"out_refund_no": refundResult.OutRefundNo,
			"refund_id":     refundResult.RefundId,
			"refund_status": refundResult.RefundStatus,
		},
		Message: "退款通知处理成功",
	}, nil
}

// DownloadBill 下载对账单
func (wp *WechatProvider) DownloadBill(params *types.DownloadBillParams) (*types.DownloadBillResponse, error) {
	ctx := context.Background()

	// 构建请求参数
	bm := make(gopay.BodyMap)
	bm.Set("bill_date", params.BillDate) // 格式：YYYY-MM-DD

	// 设置账单类型
	if params.BillType != "" {
		bm.Set("bill_type", params.BillType) // ALL（默认值）、SUCCESS、REFUND
	} else {
		bm.Set("bill_type", "ALL")
	}

	// 调用微信获取对账单接口
	wxRsp, err := wp.client.V3BillTradeBill(ctx, bm)
	if err != nil {
		return &types.DownloadBillResponse{
			Success: false,
			Error:   fmt.Sprintf("调用微信对账单接口失败: %v", err),
		}, err
	}

	if wxRsp.Code != wechat.Success {
		return &types.DownloadBillResponse{
			Success: false,
			Error:   fmt.Sprintf("微信获取对账单失败: %s", wxRsp.Error),
		}, fmt.Errorf("wechat download bill failed: %s", wxRsp.Error)
	}

	// 获取下载链接
	downloadUrl := wxRsp.Response.DownloadUrl
	if downloadUrl == "" {
		return &types.DownloadBillResponse{
			Success: false,
			Error:   "未获取到下载链接",
		}, fmt.Errorf("download url is empty")
	}

	// 下载对账单文件
	billData, err := wp.client.V3BillDownLoadBill(ctx, downloadUrl)
	if err != nil {
		// 如果下载失败，返回下载链接
		return &types.DownloadBillResponse{
			Success:  true,
			BillData: downloadUrl, // 返回下载链接
			Message:  "获取对账单下载链接成功",
		}, nil
	}

	// 返回对账单内容
	return &types.DownloadBillResponse{
		Success:  true,
		BillData: string(billData), // 返回对账单数据
		Message:  "对账单下载成功",
	}, nil
}
