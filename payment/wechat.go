package payment

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/wechat/v3"
	"github.com/yaoapp/kun/log"
)

// WeChatV3Config 微信支付V3配置
type WeChatV3Config struct {
	MchID      string `json:"mch_id"`      // 商户ID
	SerialNo   string `json:"serial_no"`   // 商户证书序列号
	APIv3Key   string `json:"apiv3_key"`   // APIv3密钥
	PrivateKey string `json:"private_key"` // 私钥内容
	PublicKey  string `json:"public_key"`  // 微信支付公钥内容
	PublicKeyID string `json:"public_key_id"` // 微信支付公钥ID
	NotifyURL  string `json:"notify_url"`  // 异步通知地址
	IsProd     bool   `json:"is_prod"`     // 是否生产环境
}

// WeChatV3Client 微信支付V3客户端
type WeChatV3Client struct {
	client *wechat.ClientV3
	config *WeChatV3Config
}

// WeChatV3PayRequest 微信支付V3请求参数
type WeChatV3PayRequest struct {
	AppID       string  `json:"app_id"`       // 应用ID
	Description string  `json:"description"`  // 商品描述
	OutTradeNo  string  `json:"out_trade_no"` // 商户订单号
	TotalAmount int64   `json:"total_amount"` // 支付金额（分）
	Currency    string  `json:"currency"`     // 货币类型，默认CNY
	TimeExpire  string  `json:"time_expire"`  // 订单失效时间
	NotifyURL   string  `json:"notify_url"`   // 异步通知地址
	OpenID      string  `json:"openid"`       // 用户openid（JSAPI支付必填）
	SceneInfo   *SceneInfo `json:"scene_info"` // 场景信息（H5支付必填）
}

// SceneInfo H5支付场景信息
type SceneInfo struct {
	PayerClientIP string    `json:"payer_client_ip"` // 用户终端IP
	H5Info        *H5Info   `json:"h5_info"`         // H5场景信息
}

// H5Info H5场景详细信息
type H5Info struct {
	Type    string `json:"type"`    // 场景类型
	AppName string `json:"app_name"` // 应用名称
	AppURL  string `json:"app_url"`  // 网站URL
	BundleID string `json:"bundle_id"` // iOS平台BundleID
	PackageName string `json:"package_name"` // Android平台包名
}

// WeChatV3PayResponse 微信支付V3响应
type WeChatV3PayResponse struct {
	PrepayID string `json:"prepay_id"` // 预支付交易会话标识
	CodeURL  string `json:"code_url"`  // 二维码链接（Native支付）
	H5URL    string `json:"h5_url"`    // 支付跳转链接（H5支付）
	PaySign  *PaySign `json:"pay_sign"` // 调起支付签名信息
}

// PaySign 调起支付签名信息
type PaySign struct {
	AppID     string `json:"appId"`     // 应用ID
	TimeStamp string `json:"timeStamp"` // 时间戳
	NonceStr  string `json:"nonceStr"`  // 随机字符串
	Package   string `json:"package"`   // 订单详情扩展字符串
	SignType  string `json:"signType"`  // 签名方式
	PaySign   string `json:"paySign"`   // 签名
}

// NewWeChatV3Client 创建微信支付V3客户端
func NewWeChatV3Client(config *WeChatV3Config) (*WeChatV3Client, error) {
	if config == nil {
		return nil, fmt.Errorf("微信支付V3配置不能为空")
	}

	// 验证必要参数
	if config.MchID == "" || config.SerialNo == "" || config.APIv3Key == "" || config.PrivateKey == "" {
		return nil, fmt.Errorf("微信支付V3配置参数不完整")
	}

	// 初始化微信客户端V3
	client, err := wechat.NewClientV3(config.MchID, config.SerialNo, config.APIv3Key, config.PrivateKey)
	if err != nil {
		log.Error("初始化微信支付V3客户端失败: %v", err)
		return nil, fmt.Errorf("初始化微信支付V3客户端失败: %v", err)
	}

	// 设置微信支付公钥自动验签（推荐方式）
	if config.PublicKey != "" && config.PublicKeyID != "" {
		err = client.SetPlatformCert([]byte(config.PublicKey), config.PublicKeyID)
		if err != nil {
			log.Error("设置微信支付公钥验签失败: %v", err)
			return nil, fmt.Errorf("设置微信支付公钥验签失败: %v", err)
		}
	} else {
		// 使用微信平台证书自动获取证书+同步验签（并自动定时更新微信平台API证书）
		err = client.AutoVerifySign()
		if err != nil {
			log.Error("设置微信支付自动验签失败: %v", err)
			return nil, fmt.Errorf("设置微信支付自动验签失败: %v", err)
		}
	}

	// 设置Debug模式（开发环境）
	if !config.IsProd {
		client.DebugSwitch = gopay.DebugOn
	}

	log.Info("微信支付V3客户端初始化成功，商户ID: %s", config.MchID)

	return &WeChatV3Client{
		client: client,
		config: config,
	}, nil
}

// JSAPIPayment JSAPI支付（小程序、公众号）
func (w *WeChatV3Client) JSAPIPayment(ctx context.Context, req *WeChatV3PayRequest) (*WeChatV3PayResponse, error) {
	if req.OpenID == "" {
		return nil, fmt.Errorf("JSAPI支付必须提供用户openid")
	}

	// 设置订单失效时间（默认10分钟）
	if req.TimeExpire == "" {
		req.TimeExpire = time.Now().Add(10 * time.Minute).Format(time.RFC3339)
	}

	// 设置货币类型
	if req.Currency == "" {
		req.Currency = "CNY"
	}

	// 设置异步通知地址
	notifyURL := req.NotifyURL
	if notifyURL == "" {
		notifyURL = w.config.NotifyURL
	}

	// 构建请求参数
	bm := make(gopay.BodyMap)
	bm.Set("appid", req.AppID).
		Set("mchid", w.config.MchID).
		Set("description", req.Description).
		Set("out_trade_no", req.OutTradeNo).
		Set("time_expire", req.TimeExpire).
		Set("notify_url", notifyURL).
		SetBodyMap("amount", func(bm gopay.BodyMap) {
			bm.Set("total", req.TotalAmount).
				Set("currency", req.Currency)
		}).
		SetBodyMap("payer", func(bm gopay.BodyMap) {
			bm.Set("openid", req.OpenID)
		})

	// 调用微信支付JSAPI下单接口
	wxRsp, err := w.client.V3TransactionJsapi(ctx, bm)
	if err != nil {
		log.Error("微信JSAPI支付下单失败: %v", err)
		return nil, fmt.Errorf("微信JSAPI支付下单失败: %v", err)
	}

	if wxRsp.Code != wechat.Success {
		log.Error("微信JSAPI支付下单响应错误: %s", wxRsp.Error)
		return nil, fmt.Errorf("微信JSAPI支付下单失败: %s", wxRsp.Error)
	}

	// 生成调起支付签名
	paySign, err := w.client.PaySignOfJSAPI(req.AppID, wxRsp.Response.PrepayId)
	if err != nil {
		log.Error("生成JSAPI支付签名失败: %v", err)
		return nil, fmt.Errorf("生成JSAPI支付签名失败: %v", err)
	}

	log.Info("微信JSAPI支付下单成功，订单号: %s, PrepayID: %s", req.OutTradeNo, wxRsp.Response.PrepayId)

	return &WeChatV3PayResponse{
		PrepayID: wxRsp.Response.PrepayId,
		PaySign: &PaySign{
			AppID:     paySign.AppId,
			TimeStamp: paySign.TimeStamp,
			NonceStr:  paySign.NonceStr,
			Package:   paySign.Package,
			SignType:  "RSA",
			PaySign:   paySign.PaySign,
		},
	}, nil
}

// AppPayment APP支付
func (w *WeChatV3Client) AppPayment(ctx context.Context, req *WeChatV3PayRequest) (*WeChatV3PayResponse, error) {
	// 设置订单失效时间（默认10分钟）
	if req.TimeExpire == "" {
		req.TimeExpire = time.Now().Add(10 * time.Minute).Format(time.RFC3339)
	}

	// 设置货币类型
	if req.Currency == "" {
		req.Currency = "CNY"
	}

	// 设置异步通知地址
	notifyURL := req.NotifyURL
	if notifyURL == "" {
		notifyURL = w.config.NotifyURL
	}

	// 构建请求参数
	bm := make(gopay.BodyMap)
	bm.Set("appid", req.AppID).
		Set("mchid", w.config.MchID).
		Set("description", req.Description).
		Set("out_trade_no", req.OutTradeNo).
		Set("time_expire", req.TimeExpire).
		Set("notify_url", notifyURL).
		SetBodyMap("amount", func(bm gopay.BodyMap) {
			bm.Set("total", req.TotalAmount).
				Set("currency", req.Currency)
		})

	// 调用微信支付APP下单接口
	wxRsp, err := w.client.V3TransactionApp(ctx, bm)
	if err != nil {
		log.Error("微信APP支付下单失败: %v", err)
		return nil, fmt.Errorf("微信APP支付下单失败: %v", err)
	}

	if wxRsp.Code != wechat.Success {
		log.Error("微信APP支付下单响应错误: %s", wxRsp.Error)
		return nil, fmt.Errorf("微信APP支付下单失败: %s", wxRsp.Error)
	}

	// 生成调起支付签名
	paySign, err := w.client.PaySignOfApp(req.AppID, wxRsp.Response.PrepayId)
	if err != nil {
		log.Error("生成APP支付签名失败: %v", err)
		return nil, fmt.Errorf("生成APP支付签名失败: %v", err)
	}

	log.Info("微信APP支付下单成功，订单号: %s, PrepayID: %s", req.OutTradeNo, wxRsp.Response.PrepayId)

	return &WeChatV3PayResponse{
		PrepayID: wxRsp.Response.PrepayId,
		PaySign: &PaySign{
			AppID:     paySign.Appid,
			TimeStamp: paySign.Timestamp,
			NonceStr:  paySign.Noncestr,
			Package:   paySign.Package,
			SignType:  "RSA",
			PaySign:   paySign.Sign,
		},
	}, nil
}

// NativePayment Native支付（扫码支付）
func (w *WeChatV3Client) NativePayment(ctx context.Context, req *WeChatV3PayRequest) (*WeChatV3PayResponse, error) {
	// 设置订单失效时间（默认10分钟）
	if req.TimeExpire == "" {
		req.TimeExpire = time.Now().Add(10 * time.Minute).Format(time.RFC3339)
	}

	// 设置货币类型
	if req.Currency == "" {
		req.Currency = "CNY"
	}

	// 设置异步通知地址
	notifyURL := req.NotifyURL
	if notifyURL == "" {
		notifyURL = w.config.NotifyURL
	}

	// 构建请求参数
	bm := make(gopay.BodyMap)
	bm.Set("appid", req.AppID).
		Set("mchid", w.config.MchID).
		Set("description", req.Description).
		Set("out_trade_no", req.OutTradeNo).
		Set("time_expire", req.TimeExpire).
		Set("notify_url", notifyURL).
		SetBodyMap("amount", func(bm gopay.BodyMap) {
			bm.Set("total", req.TotalAmount).
				Set("currency", req.Currency)
		})

	// 调用微信支付Native下单接口
	wxRsp, err := w.client.V3TransactionNative(ctx, bm)
	if err != nil {
		log.Error("微信Native支付下单失败: %v", err)
		return nil, fmt.Errorf("微信Native支付下单失败: %v", err)
	}

	if wxRsp.Code != wechat.Success {
		log.Error("微信Native支付下单响应错误: %s", wxRsp.Error)
		return nil, fmt.Errorf("微信Native支付下单失败: %s", wxRsp.Error)
	}

	log.Info("微信Native支付下单成功，订单号: %s, 二维码: %s", req.OutTradeNo, wxRsp.Response.CodeUrl)

	return &WeChatV3PayResponse{
		CodeURL: wxRsp.Response.CodeUrl,
	}, nil
}

// H5Payment H5支付
func (w *WeChatV3Client) H5Payment(ctx context.Context, req *WeChatV3PayRequest) (*WeChatV3PayResponse, error) {
	if req.SceneInfo == nil {
		return nil, fmt.Errorf("H5支付必须提供场景信息")
	}

	// 设置订单失效时间（默认10分钟）
	if req.TimeExpire == "" {
		req.TimeExpire = time.Now().Add(10 * time.Minute).Format(time.RFC3339)
	}

	// 设置货币类型
	if req.Currency == "" {
		req.Currency = "CNY"
	}

	// 设置异步通知地址
	notifyURL := req.NotifyURL
	if notifyURL == "" {
		notifyURL = w.config.NotifyURL
	}

	// 构建请求参数
	bm := make(gopay.BodyMap)
	bm.Set("appid", req.AppID).
		Set("mchid", w.config.MchID).
		Set("description", req.Description).
		Set("out_trade_no", req.OutTradeNo).
		Set("time_expire", req.TimeExpire).
		Set("notify_url", notifyURL).
		SetBodyMap("amount", func(bm gopay.BodyMap) {
			bm.Set("total", req.TotalAmount).
				Set("currency", req.Currency)
		}).
		SetBodyMap("scene_info", func(bm gopay.BodyMap) {
			bm.Set("payer_client_ip", req.SceneInfo.PayerClientIP)
			if req.SceneInfo.H5Info != nil {
				bm.SetBodyMap("h5_info", func(bm gopay.BodyMap) {
					bm.Set("type", req.SceneInfo.H5Info.Type)
					if req.SceneInfo.H5Info.AppName != "" {
						bm.Set("app_name", req.SceneInfo.H5Info.AppName)
					}
					if req.SceneInfo.H5Info.AppURL != "" {
						bm.Set("app_url", req.SceneInfo.H5Info.AppURL)
					}
					if req.SceneInfo.H5Info.BundleID != "" {
						bm.Set("bundle_id", req.SceneInfo.H5Info.BundleID)
					}
					if req.SceneInfo.H5Info.PackageName != "" {
						bm.Set("package_name", req.SceneInfo.H5Info.PackageName)
					}
				})
			}
		})

	// 调用微信支付H5下单接口
	wxRsp, err := w.client.V3TransactionH5(ctx, bm)
	if err != nil {
		log.Error("微信H5支付下单失败: %v", err)
		return nil, fmt.Errorf("微信H5支付下单失败: %v", err)
	}

	if wxRsp.Code != wechat.Success {
		log.Error("微信H5支付下单响应错误: %s", wxRsp.Error)
		return nil, fmt.Errorf("微信H5支付下单失败: %s", wxRsp.Error)
	}

	log.Info("微信H5支付下单成功，订单号: %s, 支付链接: %s", req.OutTradeNo, wxRsp.Response.H5Url)

	return &WeChatV3PayResponse{
		H5URL: wxRsp.Response.H5Url,
	}, nil
}

// QueryOrder 查询订单
func (w *WeChatV3Client) QueryOrder(ctx context.Context, outTradeNo string) (*wechat.QueryOrderRsp, error) {
	if outTradeNo == "" {
		return nil, fmt.Errorf("商户订单号不能为空")
	}

	// 查询订单
	wxRsp, err := w.client.V3TransactionQueryOrder(ctx, wechat.OutTradeNo, outTradeNo)
	if err != nil {
		log.Error("查询微信支付订单失败: %v", err)
		return nil, fmt.Errorf("查询微信支付订单失败: %v", err)
	}

	if wxRsp.Code != wechat.Success {
		log.Error("查询微信支付订单响应错误: %s", wxRsp.Error)
		return nil, fmt.Errorf("查询微信支付订单失败: %s", wxRsp.Error)
	}

	log.Info("查询微信支付订单成功，订单号: %s, 状态: %s", outTradeNo, wxRsp.Response.TradeState)

	return wxRsp, nil
}

// CloseOrder 关闭订单
func (w *WeChatV3Client) CloseOrder(ctx context.Context, outTradeNo string) error {
	if outTradeNo == "" {
		return fmt.Errorf("商户订单号不能为空")
	}

	// 关闭订单
	wxRsp, err := w.client.V3TransactionCloseOrder(ctx, outTradeNo)
	if err != nil {
		log.Error("关闭微信支付订单失败: %v", err)
		return fmt.Errorf("关闭微信支付订单失败: %v", err)
	}

	if wxRsp.Code != wechat.Success {
		log.Error("关闭微信支付订单响应错误: %s", wxRsp.Error)
		return fmt.Errorf("关闭微信支付订单失败: %s", wxRsp.Error)
	}

	log.Info("关闭微信支付订单成功，订单号: %s", outTradeNo)

	return nil
}

// Refund 申请退款
func (w *WeChatV3Client) Refund(ctx context.Context, outTradeNo, outRefundNo string, refundAmount, totalAmount int64, reason string) (*wechat.RefundRsp, error) {
	if outTradeNo == "" || outRefundNo == "" {
		return nil, fmt.Errorf("商户订单号和退款单号不能为空")
	}

	if refundAmount <= 0 || totalAmount <= 0 {
		return nil, fmt.Errorf("退款金额和订单总金额必须大于0")
	}

	// 构建退款请求参数
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", outTradeNo).
		Set("out_refund_no", outRefundNo).
		Set("reason", reason).
		SetBodyMap("amount", func(bm gopay.BodyMap) {
			bm.Set("refund", refundAmount).
				Set("total", totalAmount).
				Set("currency", "CNY")
		})

	// 申请退款
	wxRsp, err := w.client.V3Refund(ctx, bm)
	if err != nil {
		log.Error("申请微信支付退款失败: %v", err)
		return nil, fmt.Errorf("申请微信支付退款失败: %v", err)
	}

	if wxRsp.Code != wechat.Success {
		log.Error("申请微信支付退款响应错误: %s", wxRsp.Error)
		return nil, fmt.Errorf("申请微信支付退款失败: %s", wxRsp.Error)
	}

	log.Info("申请微信支付退款成功，订单号: %s, 退款单号: %s", outTradeNo, outRefundNo)

	return wxRsp, nil
}

// QueryRefund 查询退款
func (w *WeChatV3Client) QueryRefund(ctx context.Context, outRefundNo string) (*wechat.RefundQueryRsp, error) {
	if outRefundNo == "" {
		return nil, fmt.Errorf("退款单号不能为空")
	}

	// 查询退款
	wxRsp, err := w.client.V3RefundQuery(ctx, outRefundNo, nil)
	if err != nil {
		log.Error("查询微信支付退款失败: %v", err)
		return nil, fmt.Errorf("查询微信支付退款失败: %v", err)
	}

	if wxRsp.Code != wechat.Success {
		log.Error("查询微信支付退款响应错误: %s", wxRsp.Error)
		return nil, fmt.Errorf("查询微信支付退款失败: %s", wxRsp.Error)
	}

	log.Info("查询微信支付退款成功，退款单号: %s, 状态: %s", outRefundNo, wxRsp.Response.Status)

	return wxRsp, nil
}