// Package payment 提供支付功能的适配器
package payment

import (
	"encoding/json"

	"github.com/yaoapp/yao/payment/providers"
)

// ProviderAdapter 支付提供商适配器，用于适配不同的类型定义
type ProviderAdapter struct {
	provider providers.PaymentProvider
}

// CreateOrder 创建支付订单
func (pa *ProviderAdapter) CreateOrder(params *CreateOrderParams) (*CreateOrderResponse, error) {
	providerParams := &providers.CreateOrderParams{
		OutTradeNo: params.OutTradeNo,
		Amount:     params.Amount,
		Subject:    params.Subject,
		Body:       params.Body,
		TradeType:  providers.TradeType(params.TradeType),
		NotifyURL:  params.NotifyURL,
		ReturnURL:  params.ReturnURL,
		ExpireTime: params.ExpireTime,
		MerchantID: params.MerchantNo,
		Channel:    providers.PaymentChannel(params.Channel),
		Extra:      params.ExtendParams,
	}

	// 设置支付宝特有参数
	if params.AlipayParams != nil {
		providerParams.AlipayParams = &providers.AlipayOrderParams{
			TimeoutExpress:    params.AlipayParams.TimeoutExpress,
			ExtendParams:      params.AlipayParams.ExtendParams,
			BuyerID:           params.AlipayParams.BuyerID,
			BuyerLogonID:      params.AlipayParams.BuyerLogonID,
			ProductCode:       params.AlipayParams.ProductCode,
			QuitURL:           params.AlipayParams.QuitURL,
			EnablePayChannels: params.AlipayParams.EnablePayChannels,
		}
	}

	// 设置微信特有参数
	if params.WechatParams != nil {
		wechatParams := &providers.WechatOrderParams{
			Detail:    params.WechatParams.Detail,
			AppID:     params.WechatParams.AppID,
			Attach:    params.WechatParams.Attach,
			GoodsTag:  params.WechatParams.GoodsTag,
			LimitPay:  params.WechatParams.LimitPay,
			StoreInfo: params.WechatParams.StoreInfo,
		}
		
		// 处理 SceneInfo 字段类型转换
		if params.WechatParams.SceneInfo != "" {
			// 如果 SceneInfo 是字符串，尝试解析为 JSON
			sceneInfo := make(map[string]interface{})
			if err := json.Unmarshal([]byte(params.WechatParams.SceneInfo), &sceneInfo); err == nil {
				wechatParams.SceneInfo = sceneInfo
			} else {
				// 如果解析失败，创建默认的 SceneInfo
				wechatParams.SceneInfo = map[string]interface{}{
					"h5_info": map[string]interface{}{
						"type": "Wap",
						"wap_url": params.WechatParams.SceneInfo,
						"wap_name": "支付",
					},
				}
			}
		}
		
		providerParams.WechatParams = wechatParams
	}

	// 调用提供商方法
	response, err := pa.provider.CreateOrder(providerParams)
	if err != nil {
		return nil, err
	}

	// 转换响应类型
	return &CreateOrderResponse{
		Success:    response.Success,
		OrderID:    response.OrderID,
		OutTradeNo: response.OutTradeNo,
		PayInfo:    response.PayInfo,
		QRCode:     response.QRCode,
		PayURL:     response.PayURL,
		Message:    response.Message,
		Error:      response.Error,
	}, nil
}

// QueryOrder 查询支付订单
func (pa *ProviderAdapter) QueryOrder(params *QueryOrderParams) (*QueryOrderResponse, error) {
	// 转换参数类型
	providerParams := &providers.QueryOrderParams{
		OutTradeNo:    params.OutTradeNo,
		TradeNo:       "",
		TransactionID: params.TransactionID,
		MerchantID:    params.MerchantNo,
		Channel:       providers.PaymentChannel(params.Channel),
	}

	// 调用提供商方法
	response, err := pa.provider.QueryOrder(providerParams)
	if err != nil {
		return nil, err
	}

	// 转换响应类型
	return &QueryOrderResponse{
		Success:       response.Success,
		OrderID:       response.OrderID,
		OutTradeNo:    response.OutTradeNo,
		TransactionID: response.TransactionID,
		Channel:       string(response.Channel),
		TradeType:     string(response.TradeType),
		Amount:        response.Amount,
		Status:        string(response.Status),
		PaidAt:        &response.PaidAt,
		CreatedAt:     response.CreatedAt,
		Message:       response.Message,
		Error:         response.Error,
	}, nil
}

// CreateRefund 创建退款
func (pa *ProviderAdapter) CreateRefund(params *CreateRefundParams) (*CreateRefundResponse, error) {
	// 转换参数类型
	providerParams := &providers.CreateRefundParams{
		OutTradeNo:    params.OutTradeNo,
		TransactionID: params.TransactionID,
		OutRefundNo:   params.OutRefundNo,
		RefundAmount:  params.RefundAmount,
		TotalAmount:   params.TotalAmount,
		Reason:        params.Reason,
		NotifyURL:     params.NotifyURL,
		MerchantID:    params.MerchantNo,
		MerchantNo:    params.MerchantNo,
		Channel:       providers.PaymentChannel(params.Channel),
	}

	// 调用提供商方法
	response, err := pa.provider.CreateRefund(providerParams)
	if err != nil {
		return nil, err
	}

	// 转换响应类型
	return &CreateRefundResponse{
		Success:      response.Success,
		RefundID:     response.RefundID,
		OutRefundNo:  response.OutRefundNo,
		RefundNo:     response.RefundID,
		RefundAmount: response.RefundAmount,
		Status:       string(response.Status),
		Message:      response.Message,
		Error:        response.Error,
	}, nil
}

// QueryRefund 查询退款状态
func (pa *ProviderAdapter) QueryRefund(params *QueryRefundParams) (*QueryRefundResponse, error) {
	// 转换参数类型
	providerParams := &providers.QueryRefundParams{
		OutTradeNo:  "",
		OutRefundNo: params.OutRefundNo,
		RefundID:    params.RefundID,
		MerchantID:  params.MerchantNo,
		Channel:     providers.PaymentChannel(params.Channel),
	}

	// 调用提供商方法
	response, err := pa.provider.QueryRefund(providerParams)
	if err != nil {
		return nil, err
	}

	// 转换响应类型
	return &QueryRefundResponse{
		Success:      response.Success,
		RefundID:     response.RefundID,
		OutRefundNo:  response.OutRefundNo,
		RefundNo:     response.RefundID,
		RefundAmount: response.RefundAmount,
		Status:       string(response.Status),
		RefundAt:     &response.RefundTime,
		CreatedAt:    "",
		Message:      response.Message,
		Error:        response.Error,
	}, nil
}

// HandleNotify 处理异步通知
func (pa *ProviderAdapter) HandleNotify(params *HandleNotifyParams) (*HandleNotifyResponse, error) {
	// 转换参数类型
	providerParams := &providers.HandleNotifyParams{
		Channel:     providers.PaymentChannel(params.Channel),
		RequestBody: params.RequestBody,
		NotifyData:  make(map[string]interface{}),
		MerchantID:  params.MerchantID,
	}

	// 调用提供商方法
	response, err := pa.provider.HandleNotify(providerParams)
	if err != nil {
		return nil, err
	}

	// 转换响应类型
	return &HandleNotifyResponse{
		Success:       response.Success,
		OutTradeNo:    response.OutTradeNo,
		TransactionID: response.TradeNo,
		Amount:        response.Amount,
		Status:        string(response.Status),
		PaidAt:        nil,
		NotifyData:    response.NotifyData,
		Message:       response.Message,
		Error:         response.Error,
	}, nil
}

// DownloadBill 下载对账单
func (pa *ProviderAdapter) DownloadBill(params *DownloadBillParams) (*DownloadBillResponse, error) {
	// 转换参数类型
	providerParams := &providers.DownloadBillParams{
		Channel:    providers.PaymentChannel(params.Channel),
		BillDate:   params.BillDate,
		BillType:   params.BillType,
		MerchantNo: params.MerchantNo,
	}

	// 调用提供商方法
	response, err := pa.provider.DownloadBill(providerParams)
	if err != nil {
		return nil, err
	}

	// 转换响应类型
	return &DownloadBillResponse{
		Success:  response.Success,
		BillData: response.BillData,
		Message:  response.Message,
		Error:    response.Error,
	}, nil
}