package providers

import (
	"context"
	"fmt"

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
func NewWechatProvider(config map[string]interface{}) (PaymentProvider, error) {
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
func (wp *WechatProvider) CreateOrder(params *CreateOrderParams) (*CreateOrderResponse, error) {
	ctx := context.Background()

	// 根据交易类型调用不同的创建方法
	switch TradeType(params.TradeType) {
	case TradeTypeJSAPI:
		return wp.createJSAPIOrder(ctx, params)
	case TradeTypeNative:
		return wp.createNativeOrder(ctx, params)
	case TradeTypeApp:
		return wp.createAppOrder(ctx, params)
	case TradeTypeH5:
		return wp.createH5Order(ctx, params)
	default:
		return &CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("不支持的交易类型: %s", params.TradeType),
		}, fmt.Errorf("unsupported trade type: %s", params.TradeType)
	}
}

// createJSAPIOrder 创建JSAPI支付订单
func (wp *WechatProvider) createJSAPIOrder(ctx context.Context, params *CreateOrderParams) (*CreateOrderResponse, error) {

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

	// 调用微信JSAPI支付接口
	wxRsp, err := wp.client.V3TransactionJsapi(ctx, bm)
	if err != nil {
		return &CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("调用微信JSAPI支付接口失败: %v", err),
		}, err
	}

	if wxRsp.Code != wechat.Success {
		return &CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("微信JSAPI支付失败: %s", wxRsp.Error),
		}, fmt.Errorf("wechat jsapi payment failed: %s", wxRsp.Error)
	}

	// 生成JSAPI支付参数
	jsapiParams, err := wp.client.PaySignOfJSAPI(wp.config.AppID, wxRsp.Response.PrepayId)
	if err != nil {
		return &CreateOrderResponse{
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

	return &CreateOrderResponse{
		Success:    true,
		OrderID:    params.OutTradeNo,
		OutTradeNo: params.OutTradeNo,
		PayInfo:    payInfo,
		Message:    "订单创建成功",
	}, nil
}

// createNativeOrder 创建Native支付订单
func (wp *WechatProvider) createNativeOrder(ctx context.Context, params *CreateOrderParams) (*CreateOrderResponse, error) {
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
		return &CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("调用微信Native支付接口失败: %v", err),
		}, err
	}

	if wxRsp.Code != wechat.Success {
		return &CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("微信Native支付失败: %s", wxRsp.Error),
		}, fmt.Errorf("wechat native payment failed: %s", wxRsp.Error)
	}

	return &CreateOrderResponse{
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
func (wp *WechatProvider) createAppOrder(ctx context.Context, params *CreateOrderParams) (*CreateOrderResponse, error) {
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
		return &CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("调用微信APP支付接口失败: %v", err),
		}, err
	}

	if wxRsp.Code != wechat.Success {
		return &CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("微信APP支付失败: %s", wxRsp.Error),
		}, fmt.Errorf("wechat app payment failed: %s", wxRsp.Error)
	}

	// 生成APP支付参数
	appParams, err := wp.client.PaySignOfApp(wp.config.AppID, wxRsp.Response.PrepayId)
	if err != nil {
		return &CreateOrderResponse{
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

	return &CreateOrderResponse{
		Success:    true,
		OrderID:    params.OutTradeNo,
		OutTradeNo: params.OutTradeNo,
		PayInfo:    payInfo,
		Message:    "订单创建成功",
	}, nil
}

// createH5Order 创建H5支付订单
func (wp *WechatProvider) createH5Order(ctx context.Context, params *CreateOrderParams) (*CreateOrderResponse, error) {
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
		return &CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("调用微信H5支付接口失败: %v", err),
		}, err
	}

	if wxRsp.Code != wechat.Success {
		return &CreateOrderResponse{
			Success: false,
			Error:   fmt.Sprintf("微信H5支付失败: %s", wxRsp.Error),
		}, fmt.Errorf("wechat h5 payment failed: %s", wxRsp.Error)
	}

	return &CreateOrderResponse{
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
func (wp *WechatProvider) QueryOrder(params *QueryOrderParams) (*QueryOrderResponse, error) {
	_ = context.Background()

	// 暂时返回基本响应，实际实现需要调用微信查询订单接口
	return &QueryOrderResponse{
		Success:    true,
		OrderID:    params.OutTradeNo,
		OutTradeNo: params.OutTradeNo,
		Status:     OrderStatusPending,
		Message:    "查询订单功能暂未实现",
	}, nil
}

// CreateRefund 创建退款
func (wp *WechatProvider) CreateRefund(params *CreateRefundParams) (*CreateRefundResponse, error) {
	_ = context.Background()

	// 构建请求参数
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", params.OutTradeNo)
	bm.Set("out_refund_no", params.OutRefundNo)
	bm.Set("reason", params.Reason)
	bm.Set("notify_url", params.NotifyURL)

	// 设置金额信息
	bm.SetBodyMap("amount", func(bm gopay.BodyMap) {
		bm.Set("refund", params.RefundAmount)
		bm.Set("total", params.TotalAmount)
		bm.Set("currency", "CNY")
	})

	// 暂时返回基本响应，实际实现需要调用微信退款接口
	return &CreateRefundResponse{
		Success:      true,
		RefundID:     params.OutRefundNo,
		OutRefundNo:  params.OutRefundNo,
		Status:       RefundStatusProcessing,
		RefundAmount: params.RefundAmount,
		Message:      "退款申请功能暂未实现",
	}, nil
}

// QueryRefund 查询退款
func (wp *WechatProvider) QueryRefund(params *QueryRefundParams) (*QueryRefundResponse, error) {
	// 暂时返回基本响应，实际实现需要调用微信查询退款接口
	return &QueryRefundResponse{
		Success:     true,
		OutRefundNo: params.OutRefundNo,
		Status:      RefundStatusProcessing,
		Message:     "查询退款功能暂未实现",
	}, nil
}

// HandleNotify 处理支付通知
func (wp *WechatProvider) HandleNotify(params *HandleNotifyParams) (*HandleNotifyResponse, error) {
	// 暂时返回基本响应，实际实现需要验证签名和解析通知内容
	return &HandleNotifyResponse{
		Success: true,
		Message: "通知处理功能暂未实现",
	}, nil
}

// DownloadBill 下载对账单
func (wp *WechatProvider) DownloadBill(params *DownloadBillParams) (*DownloadBillResponse, error) {
	// 暂时返回基本响应，实际实现需要调用微信对账单接口
	return &DownloadBillResponse{
		Success: true,
		Message: "对账单下载功能暂未实现",
	}, nil
}
