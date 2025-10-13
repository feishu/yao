package payment

import (
	"context"
	"fmt"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay/v3"
	"github.com/yaoapp/kun/log"
)

// AliPayV3Client 支付宝V3客户端
type AliPayV3Client struct {
	client *alipay.ClientV3
	config *AliPayConfig
}

// AliPayConfig 支付宝配置
type AliPayConfig struct {
	AppID              string `json:"app_id"`               // 应用ID
	PrivateKey         string `json:"private_key"`          // 应用私钥
	IsProd             bool   `json:"is_prod"`              // 是否生产环境
	AppAuthToken       string `json:"app_auth_token"`       // 授权token
	AppPublicCert      string `json:"app_public_cert"`      // 应用公钥证书
	AlipayRootCert     string `json:"alipay_root_cert"`     // 支付宝根证书
	AlipayPublicCert   string `json:"alipay_public_cert"`   // 支付宝公钥证书
	AESKey             string `json:"aes_key"`              // AES加密密钥
	NotifyURL          string `json:"notify_url"`           // 异步通知地址
	ReturnURL          string `json:"return_url"`           // 同步跳转地址
	SignType           string `json:"sign_type"`            // 签名类型，默认RSA2
	Charset            string `json:"charset"`              // 字符集，默认utf-8
	Format             string `json:"format"`               // 数据格式，默认JSON
	Version            string `json:"version"`              // 接口版本，默认1.0
	DebugSwitch        bool   `json:"debug_switch"`         // 调试开关
}

// NewAliPayV3Client 创建支付宝V3客户端
func NewAliPayV3Client(config *AliPayConfig) (*AliPayV3Client, error) {
	if config == nil {
		return nil, fmt.Errorf("支付宝配置不能为空")
	}

	if config.AppID == "" {
		return nil, fmt.Errorf("应用ID不能为空")
	}

	if config.PrivateKey == "" {
		return nil, fmt.Errorf("应用私钥不能为空")
	}

	// 初始化支付宝客户端
	client, err := alipay.NewClientV3(config.AppID, config.PrivateKey, config.IsProd)
	if err != nil {
		return nil, fmt.Errorf("初始化支付宝客户端失败: %v", err)
	}

	// 设置自定义配置
	if config.AppAuthToken != "" {
		client.SetAppAuthToken(config.AppAuthToken)
	}

	if config.AESKey != "" {
		client.SetAESKey(config.AESKey)
	}

	// 设置调试开关
	if config.DebugSwitch {
		client.DebugSwitch = gopay.DebugOn
	} else {
		client.DebugSwitch = gopay.DebugOff
	}

	// 设置证书（如果提供）
	if config.AppPublicCert != "" && config.AlipayRootCert != "" && config.AlipayPublicCert != "" {
		err = client.SetCert([]byte(config.AppPublicCert), []byte(config.AlipayRootCert), []byte(config.AlipayPublicCert))
		if err != nil {
			return nil, fmt.Errorf("设置支付宝证书失败: %v", err)
		}
	}

	return &AliPayV3Client{
		client: client,
		config: config,
	}, nil
}

// TradePayParams 统一收单交易支付接口参数
type TradePayParams struct {
	OutTradeNo    string                 `json:"out_trade_no"`    // 商户订单号
	Scene         string                 `json:"scene"`           // 支付场景
	AuthCode      string                 `json:"auth_code"`       // 支付授权码
	Subject       string                 `json:"subject"`         // 订单标题
	TotalAmount   string                 `json:"total_amount"`    // 订单总金额
	Body          string                 `json:"body"`            // 订单描述
	TimeoutExpress string                `json:"timeout_express"` // 超时时间
	ExtendParams  map[string]interface{} `json:"extend_params"`   // 业务扩展参数
}

// TradePayResponse 统一收单交易支付接口响应
type TradePayResponse struct {
	Code           string `json:"code"`
	Msg            string `json:"msg"`
	SubCode        string `json:"sub_code"`
	SubMsg         string `json:"sub_msg"`
	TradeNo        string `json:"trade_no"`
	OutTradeNo     string `json:"out_trade_no"`
	BuyerLogonID   string `json:"buyer_logon_id"`
	TotalAmount    string `json:"total_amount"`
	ReceiptAmount  string `json:"receipt_amount"`
	BuyerPayAmount string `json:"buyer_pay_amount"`
	PointAmount    string `json:"point_amount"`
	InvoiceAmount  string `json:"invoice_amount"`
	GMTPayment     string `json:"gmt_payment"`
	FundBillList   string `json:"fund_bill_list"`
	CardBalance    string `json:"card_balance"`
	StoreName      string `json:"store_name"`
	BuyerUserID    string `json:"buyer_user_id"`
	DiscountGoodsDetail string `json:"discount_goods_detail"`
	VoucherDetailList   string `json:"voucher_detail_list"`
	AdvanceAmount       string `json:"advance_amount"`
	AuthTradePayMode    string `json:"auth_trade_pay_mode"`
	ChargeAmount        string `json:"charge_amount"`
	ChargeFlags         string `json:"charge_flags"`
	SettlementID        string `json:"settlement_id"`
	BusinessParams      string `json:"business_params"`
	BuyerUserType       string `json:"buyer_user_type"`
	MdiscountAmount     string `json:"mdiscount_amount"`
	DiscountAmount      string `json:"discount_amount"`
	BuyerUserName       string `json:"buyer_user_name"`
}

// TradePay 统一收单交易支付接口
func (a *AliPayV3Client) TradePay(ctx context.Context, params *TradePayParams) (*TradePayResponse, error) {
	if params == nil {
		return nil, fmt.Errorf("支付参数不能为空")
	}

	// 构建请求参数
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", params.OutTradeNo).
		Set("scene", params.Scene).
		Set("auth_code", params.AuthCode).
		Set("subject", params.Subject).
		Set("total_amount", params.TotalAmount)

	if params.Body != "" {
		bm.Set("body", params.Body)
	}

	if params.TimeoutExpress != "" {
		bm.Set("timeout_express", params.TimeoutExpress)
	}

	if params.ExtendParams != nil {
		bm.Set("extend_params", params.ExtendParams)
	}

	// 调用支付接口
	aliRsp, err := a.client.TradePay(ctx, bm)
	if err != nil {
		log.Error("支付宝统一收单交易支付失败: %v", err)
		return nil, err
	}

	// 转换响应
	response := &TradePayResponse{
		Code:           "10000", // 成功状态码
		Msg:            "Success",
		SubCode:        "",
		SubMsg:         "",
		TradeNo:        aliRsp.TradeNo,
		OutTradeNo:     aliRsp.OutTradeNo,
		BuyerLogonID:   aliRsp.BuyerLogonId,
		TotalAmount:    aliRsp.TotalAmount,
		ReceiptAmount:  aliRsp.ReceiptAmount,
		BuyerPayAmount: aliRsp.BuyerPayAmount,
		PointAmount:    aliRsp.PointAmount,
		InvoiceAmount:  aliRsp.InvoiceAmount,
		GMTPayment:     aliRsp.GmtPayment,
		FundBillList:   "", // gopay v3 返回的是结构体数组，这里暂时设为空字符串
		CardBalance:    "", // gopay v3 中没有此字段
		StoreName:      aliRsp.StoreName,
		BuyerUserID:    aliRsp.BuyerUserId,
	}

	log.Info("支付宝统一收单交易支付成功，订单号: %s, 支付宝交易号: %s", 
		response.OutTradeNo, response.TradeNo)

	return response, nil
}

// TradePrecreateParams 统一收单线下交易预创建参数
type TradePrecreateParams struct {
	OutTradeNo     string                 `json:"out_trade_no"`     // 商户订单号
	Subject        string                 `json:"subject"`          // 订单标题
	TotalAmount    string                 `json:"total_amount"`     // 订单总金额
	Body           string                 `json:"body"`             // 订单描述
	TimeoutExpress string                 `json:"timeout_express"`  // 超时时间
	StoreID        string                 `json:"store_id"`         // 商户门店编号
	OperatorID     string                 `json:"operator_id"`      // 商户操作员编号
	TerminalID     string                 `json:"terminal_id"`      // 商户机具终端编号
	ExtendParams   map[string]interface{} `json:"extend_params"`    // 业务扩展参数
	GoodsDetail    []interface{}          `json:"goods_detail"`     // 订单包含的商品列表信息
	DiscountableAmount string             `json:"discountable_amount"` // 可打折金额
	UndiscountableAmount string           `json:"undiscountable_amount"` // 不可打折金额
}

// TradePrecreateResponse 统一收单线下交易预创建响应
type TradePrecreateResponse struct {
	Code       string `json:"code"`
	Msg        string `json:"msg"`
	SubCode    string `json:"sub_code"`
	SubMsg     string `json:"sub_msg"`
	OutTradeNo string `json:"out_trade_no"`
	QrCode     string `json:"qr_code"`
}

// TradePrecreate 统一收单线下交易预创建
func (a *AliPayV3Client) TradePrecreate(ctx context.Context, params *TradePrecreateParams) (*TradePrecreateResponse, error) {
	if params == nil {
		return nil, fmt.Errorf("预创建参数不能为空")
	}

	// 构建请求参数
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", params.OutTradeNo).
		Set("subject", params.Subject).
		Set("total_amount", params.TotalAmount)

	if params.Body != "" {
		bm.Set("body", params.Body)
	}

	if params.TimeoutExpress != "" {
		bm.Set("timeout_express", params.TimeoutExpress)
	}

	if params.StoreID != "" {
		bm.Set("store_id", params.StoreID)
	}

	if params.OperatorID != "" {
		bm.Set("operator_id", params.OperatorID)
	}

	if params.TerminalID != "" {
		bm.Set("terminal_id", params.TerminalID)
	}

	if params.ExtendParams != nil {
		bm.Set("extend_params", params.ExtendParams)
	}

	if params.GoodsDetail != nil {
		bm.Set("goods_detail", params.GoodsDetail)
	}

	if params.DiscountableAmount != "" {
		bm.Set("discountable_amount", params.DiscountableAmount)
	}

	if params.UndiscountableAmount != "" {
		bm.Set("undiscountable_amount", params.UndiscountableAmount)
	}

	// 调用预创建接口
	aliRsp, err := a.client.TradePrecreate(ctx, bm)
	if err != nil {
		log.Error("支付宝统一收单线下交易预创建失败: %v", err)
		return nil, err
	}

	// 转换响应
	response := &TradePrecreateResponse{
		Code:       "10000", // 成功状态码
		Msg:        "Success",
		SubCode:    "",
		SubMsg:     "",
		OutTradeNo: aliRsp.OutTradeNo,
		QrCode:     aliRsp.QrCode,
	}

	log.Info("支付宝统一收单线下交易预创建成功，订单号: %s, 二维码: %s", 
		response.OutTradeNo, response.QrCode)

	return response, nil
}

// TradeQueryParams 统一收单交易查询参数
type TradeQueryParams struct {
	OutTradeNo string `json:"out_trade_no"` // 商户订单号
	TradeNo    string `json:"trade_no"`     // 支付宝交易号
}

// TradeQueryResponse 统一收单交易查询响应
type TradeQueryResponse struct {
	Code           string `json:"code"`
	Msg            string `json:"msg"`
	SubCode        string `json:"sub_code"`
	SubMsg         string `json:"sub_msg"`
	TradeNo        string `json:"trade_no"`
	OutTradeNo     string `json:"out_trade_no"`
	BuyerLogonID   string `json:"buyer_logon_id"`
	TradeStatus    string `json:"trade_status"`
	TotalAmount    string `json:"total_amount"`
	TransCurrency  string `json:"trans_currency"`
	SettleCurrency string `json:"settle_currency"`
	SettleAmount   string `json:"settle_amount"`
	PayCurrency    string `json:"pay_currency"`
	PayAmount      string `json:"pay_amount"`
	SettleTransRate string `json:"settle_trans_rate"`
	TransPayRate    string `json:"trans_pay_rate"`
	BuyerPayAmount  string `json:"buyer_pay_amount"`
	PointAmount     string `json:"point_amount"`
	InvoiceAmount   string `json:"invoice_amount"`
	SendPayDate     string `json:"send_pay_date"`
	ReceiptAmount   string `json:"receipt_amount"`
	StoreID         string `json:"store_id"`
	TerminalID      string `json:"terminal_id"`
	FundBillList    string `json:"fund_bill_list"`
	StoreName       string `json:"store_name"`
	BuyerUserID     string `json:"buyer_user_id"`
	ChargeAmount    string `json:"charge_amount"`
	ChargeFlags     string `json:"charge_flags"`
	SettlementID    string `json:"settlement_id"`
	TradeSettleInfo string `json:"trade_settle_info"`
	AuthTradePayMode string `json:"auth_trade_pay_mode"`
	BuyerUserType    string `json:"buyer_user_type"`
	MdiscountAmount  string `json:"mdiscount_amount"`
	DiscountAmount   string `json:"discount_amount"`
	Subject          string `json:"subject"`
	Body             string `json:"body"`
	AlipaySubMerchantID string `json:"alipay_sub_merchant_id"`
	ExtInfos            string `json:"ext_infos"`
}

// TradeQuery 统一收单交易查询
func (a *AliPayV3Client) TradeQuery(ctx context.Context, params *TradeQueryParams) (*TradeQueryResponse, error) {
	if params == nil {
		return nil, fmt.Errorf("查询参数不能为空")
	}

	if params.OutTradeNo == "" && params.TradeNo == "" {
		return nil, fmt.Errorf("商户订单号和支付宝交易号不能同时为空")
	}

	// 构建请求参数
	bm := make(gopay.BodyMap)
	if params.OutTradeNo != "" {
		bm.Set("out_trade_no", params.OutTradeNo)
	}
	if params.TradeNo != "" {
		bm.Set("trade_no", params.TradeNo)
	}

	// 调用查询接口
	aliRsp, err := a.client.TradeQuery(ctx, bm)
	if err != nil {
		log.Error("支付宝统一收单交易查询失败: %v", err)
		return nil, err
	}

	// 转换响应
	response := &TradeQueryResponse{
		Code:           "10000", // 成功状态码
		Msg:            "Success",
		SubCode:        "",
		SubMsg:         "",
		TradeNo:        aliRsp.TradeNo,
		OutTradeNo:     aliRsp.OutTradeNo,
		BuyerLogonID:   aliRsp.BuyerLogonId,
		TradeStatus:    aliRsp.TradeStatus,
		TotalAmount:    aliRsp.TotalAmount,
		TransCurrency:  "",
		SettleCurrency: "",
		SettleAmount:   "",
		PayCurrency:    "",
		PayAmount:      "",
		SettleTransRate: "",
		TransPayRate:    "",
		BuyerPayAmount:  aliRsp.BuyerPayAmount,
		PointAmount:     aliRsp.PointAmount,
		InvoiceAmount:   aliRsp.InvoiceAmount,
		SendPayDate:     aliRsp.SendPayDate,
		ReceiptAmount:   aliRsp.ReceiptAmount,
		StoreID:         aliRsp.StoreId,
		TerminalID:      aliRsp.TerminalId,
		FundBillList:    "",
		StoreName:       aliRsp.StoreName,
		BuyerUserID:     aliRsp.BuyerUserId,
		ChargeAmount:    "",
		ChargeFlags:     "",
		SettlementID:    "",
		TradeSettleInfo: "",
		AuthTradePayMode: "",
		BuyerUserType:    "",
		MdiscountAmount:  "",
		DiscountAmount:   "",
		Subject:          "",
		Body:             "",
		AlipaySubMerchantID: "",
		ExtInfos:            "",
	}

	log.Info("支付宝统一收单交易查询成功，订单号: %s, 交易状态: %s", 
		response.OutTradeNo, response.TradeStatus)

	return response, nil
}

// TradeRefundParams 统一收单交易退款参数
type TradeRefundParams struct {
	OutTradeNo   string                 `json:"out_trade_no"`   // 商户订单号
	TradeNo      string                 `json:"trade_no"`       // 支付宝交易号
	RefundAmount string                 `json:"refund_amount"`  // 退款金额
	RefundReason string                 `json:"refund_reason"`  // 退款原因
	OutRequestNo string                 `json:"out_request_no"` // 退款请求号
	OperatorID   string                 `json:"operator_id"`    // 商户操作员编号
	StoreID      string                 `json:"store_id"`       // 商户门店编号
	TerminalID   string                 `json:"terminal_id"`    // 商户机具终端编号
	GoodsDetail  []interface{}          `json:"goods_detail"`   // 退款包含的商品列表信息
	RefundRoyaltyParameters []interface{} `json:"refund_royalty_parameters"` // 退分账明细信息
	OrgPid       string                 `json:"org_pid"`        // 银行间联模式下有用，其它场景请不要使用
}

// TradeRefundResponse 统一收单交易退款响应
type TradeRefundResponse struct {
	Code                    string `json:"code"`
	Msg                     string `json:"msg"`
	SubCode                 string `json:"sub_code"`
	SubMsg                  string `json:"sub_msg"`
	TradeNo                 string `json:"trade_no"`
	OutTradeNo              string `json:"out_trade_no"`
	BuyerLogonID            string `json:"buyer_logon_id"`
	FundChange              string `json:"fund_change"`
	RefundFee               string `json:"refund_fee"`
	RefundCurrency          string `json:"refund_currency"`
	GMTRefundPay            string `json:"gmt_refund_pay"`
	RefundDetailItemList    string `json:"refund_detail_item_list"`
	StoreName               string `json:"store_name"`
	BuyerUserID             string `json:"buyer_user_id"`
	RefundPresetPaytoolList string `json:"refund_preset_paytool_list"`
	RefundSettlementID      string `json:"refund_settlement_id"`
	PresentRefundBuyerAmount string `json:"present_refund_buyer_amount"`
	PresentRefundDiscountAmount string `json:"present_refund_discount_amount"`
	PresentRefundMdiscountAmount string `json:"present_refund_mdiscount_amount"`
}

// TradeRefund 统一收单交易退款
func (a *AliPayV3Client) TradeRefund(ctx context.Context, params *TradeRefundParams) (*TradeRefundResponse, error) {
	if params == nil {
		return nil, fmt.Errorf("退款参数不能为空")
	}

	if params.OutTradeNo == "" && params.TradeNo == "" {
		return nil, fmt.Errorf("商户订单号和支付宝交易号不能同时为空")
	}

	if params.RefundAmount == "" {
		return nil, fmt.Errorf("退款金额不能为空")
	}

	// 构建请求参数
	bm := make(gopay.BodyMap)
	if params.OutTradeNo != "" {
		bm.Set("out_trade_no", params.OutTradeNo)
	}
	if params.TradeNo != "" {
		bm.Set("trade_no", params.TradeNo)
	}
	bm.Set("refund_amount", params.RefundAmount)

	if params.RefundReason != "" {
		bm.Set("refund_reason", params.RefundReason)
	}

	if params.OutRequestNo != "" {
		bm.Set("out_request_no", params.OutRequestNo)
	}

	if params.OperatorID != "" {
		bm.Set("operator_id", params.OperatorID)
	}

	if params.StoreID != "" {
		bm.Set("store_id", params.StoreID)
	}

	if params.TerminalID != "" {
		bm.Set("terminal_id", params.TerminalID)
	}

	if params.GoodsDetail != nil {
		bm.Set("goods_detail", params.GoodsDetail)
	}

	if params.RefundRoyaltyParameters != nil {
		bm.Set("refund_royalty_parameters", params.RefundRoyaltyParameters)
	}

	if params.OrgPid != "" {
		bm.Set("org_pid", params.OrgPid)
	}

	// 调用退款接口
	aliRsp, err := a.client.TradeRefund(ctx, bm)
	if err != nil {
		log.Error("支付宝统一收单交易退款失败: %v", err)
		return nil, err
	}

	// 转换响应
	response := &TradeRefundResponse{
		Code:                    "10000", // 成功状态码
		Msg:                     "Success",
		SubCode:                 "",
		SubMsg:                  "",
		TradeNo:                 aliRsp.TradeNo,
		OutTradeNo:              aliRsp.OutTradeNo,
		BuyerLogonID:            aliRsp.BuyerLogonId,
		FundChange:              aliRsp.FundChange,
		RefundFee:               aliRsp.RefundFee,
		RefundCurrency:          "",
		GMTRefundPay:            "",
		RefundDetailItemList:    "",
		StoreName:               aliRsp.StoreName,
		BuyerUserID:             aliRsp.BuyerUserId,
		RefundPresetPaytoolList: "",
		RefundSettlementID:      "",
		PresentRefundBuyerAmount: "",
		PresentRefundDiscountAmount: "",
		PresentRefundMdiscountAmount: "",
	}

	log.Info("支付宝统一收单交易退款成功，订单号: %s, 退款金额: %s", 
		response.OutTradeNo, response.RefundFee)

	return response, nil
}

// TradeCloseParams 统一收单交易关闭参数
type TradeCloseParams struct {
	OutTradeNo string `json:"out_trade_no"` // 商户订单号
	TradeNo    string `json:"trade_no"`     // 支付宝交易号
	OperatorID string `json:"operator_id"`  // 商户操作员编号
}

// TradeCloseResponse 统一收单交易关闭响应
type TradeCloseResponse struct {
	Code       string `json:"code"`
	Msg        string `json:"msg"`
	SubCode    string `json:"sub_code"`
	SubMsg     string `json:"sub_msg"`
	TradeNo    string `json:"trade_no"`
	OutTradeNo string `json:"out_trade_no"`
}

// TradeClose 统一收单交易关闭
func (a *AliPayV3Client) TradeClose(ctx context.Context, params *TradeCloseParams) (*TradeCloseResponse, error) {
	if params == nil {
		return nil, fmt.Errorf("关闭参数不能为空")
	}

	if params.OutTradeNo == "" && params.TradeNo == "" {
		return nil, fmt.Errorf("商户订单号和支付宝交易号不能同时为空")
	}

	// 构建请求参数
	bm := make(gopay.BodyMap)
	if params.OutTradeNo != "" {
		bm.Set("out_trade_no", params.OutTradeNo)
	}
	if params.TradeNo != "" {
		bm.Set("trade_no", params.TradeNo)
	}
	if params.OperatorID != "" {
		bm.Set("operator_id", params.OperatorID)
	}

	// 调用关闭接口
	aliRsp, err := a.client.TradeClose(ctx, bm)
	if err != nil {
		log.Error("支付宝统一收单交易关闭失败: %v", err)
		return nil, err
	}

	// 转换响应
	response := &TradeCloseResponse{
		Code:       "10000", // 成功状态码
		Msg:        "Success",
		SubCode:    "",
		SubMsg:     "",
		TradeNo:    aliRsp.TradeNo,
		OutTradeNo: aliRsp.OutTradeNo,
	}

	log.Info("支付宝统一收单交易关闭成功，订单号: %s", response.OutTradeNo)

	return response, nil
}

// GetClient 获取支付宝客户端
func (a *AliPayV3Client) GetClient() *alipay.ClientV3 {
	return a.client
}

// GetConfig 获取支付宝配置
func (a *AliPayV3Client) GetConfig() *AliPayConfig {
	return a.config
}