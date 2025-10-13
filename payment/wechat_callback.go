package payment

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/yaoapp/kun/log"
)

// CallbackHandler 微信支付V3回调处理器
type CallbackHandler struct {
	client   *WeChatV3Client
	apiV3Key string
}

// NewCallbackHandler 创建回调处理器
func NewCallbackHandler(client *WeChatV3Client, apiV3Key string) *CallbackHandler {
	return &CallbackHandler{
		client:   client,
		apiV3Key: apiV3Key,
	}
}

// HandlePayNotify 处理支付通知回调
func (h *CallbackHandler) HandlePayNotify(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// 解析异步通知
	notifyReq, err := h.client.ParseNotify(r)
	if err != nil {
		log.Error("解析微信支付通知失败: %v", err)
		h.writeErrorResponse(w, "解析通知失败")
		return
	}

	// 验证签名
	err = h.client.VerifyNotifySign(r)
	if err != nil {
		log.Error("验证微信支付通知签名失败: %v", err)
		h.writeErrorResponse(w, "签名验证失败")
		return
	}

	// 解密支付通知
	payNotify, err := h.client.DecryptPayNotify(notifyReq, h.apiV3Key)
	if err != nil {
		log.Error("解密微信支付通知失败: %v", err)
		h.writeErrorResponse(w, "解密通知失败")
		return
	}

	// 处理支付通知业务逻辑
	err = h.processPayNotify(ctx, payNotify)
	if err != nil {
		log.Error("处理微信支付通知业务逻辑失败: %v", err)
		h.writeErrorResponse(w, "处理业务逻辑失败")
		return
	}

	// 返回成功响应
	h.writeSuccessResponse(w)
	log.Info("微信支付通知处理成功，订单号: %s, 交易状态: %s", payNotify.OutTradeNo, payNotify.TradeState)
}

// HandleRefundNotify 处理退款通知回调
func (h *CallbackHandler) HandleRefundNotify(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// 解析异步通知
	notifyReq, err := h.client.ParseNotify(r)
	if err != nil {
		log.Error("解析微信退款通知失败: %v", err)
		h.writeErrorResponse(w, "解析通知失败")
		return
	}

	// 验证签名
	err = h.client.VerifyNotifySign(r)
	if err != nil {
		log.Error("验证微信退款通知签名失败: %v", err)
		h.writeErrorResponse(w, "签名验证失败")
		return
	}

	// 解密退款通知
	refundNotify, err := h.client.DecryptRefundNotify(notifyReq, h.apiV3Key)
	if err != nil {
		log.Error("解密微信退款通知失败: %v", err)
		h.writeErrorResponse(w, "解密通知失败")
		return
	}

	// 处理退款通知业务逻辑
	err = h.processRefundNotify(ctx, refundNotify)
	if err != nil {
		log.Error("处理微信退款通知业务逻辑失败: %v", err)
		h.writeErrorResponse(w, "处理业务逻辑失败")
		return
	}

	// 返回成功响应
	h.writeSuccessResponse(w)
	log.Info("微信退款通知处理成功，订单号: %s, 退款状态: %s", refundNotify.OutTradeNo, refundNotify.RefundStatus)
}

// processPayNotify 处理支付通知业务逻辑
func (h *CallbackHandler) processPayNotify(ctx context.Context, payNotify *PayNotification) error {
	// 根据交易状态处理业务逻辑
	switch payNotify.TradeState {
	case "SUCCESS":
		// 支付成功，更新订单状态
		return h.handlePaymentSuccess(ctx, payNotify)
	case "REFUND":
		// 转入退款，更新订单状态
		return h.handlePaymentRefund(ctx, payNotify)
	case "NOTPAY":
		// 未支付，记录日志
		log.Info("订单未支付，订单号: %s", payNotify.OutTradeNo)
		return nil
	case "CLOSED":
		// 已关闭，更新订单状态
		return h.handlePaymentClosed(ctx, payNotify)
	case "REVOKED":
		// 已撤销，更新订单状态
		return h.handlePaymentRevoked(ctx, payNotify)
	case "USERPAYING":
		// 用户支付中，记录日志
		log.Info("用户支付中，订单号: %s", payNotify.OutTradeNo)
		return nil
	case "PAYERROR":
		// 支付失败，更新订单状态
		return h.handlePaymentError(ctx, payNotify)
	default:
		log.Warn("未知的交易状态: %s, 订单号: %s", payNotify.TradeState, payNotify.OutTradeNo)
		return nil
	}
}

// processRefundNotify 处理退款通知业务逻辑
func (h *CallbackHandler) processRefundNotify(ctx context.Context, refundNotify *RefundNotification) error {
	// 根据退款状态处理业务逻辑
	switch refundNotify.RefundStatus {
	case "SUCCESS":
		// 退款成功，更新订单状态
		return h.handleRefundSuccess(ctx, refundNotify)
	case "CLOSED":
		// 退款关闭，更新订单状态
		return h.handleRefundClosed(ctx, refundNotify)
	case "PROCESSING":
		// 退款处理中，记录日志
		log.Info("退款处理中，订单号: %s, 退款单号: %s", refundNotify.OutTradeNo, refundNotify.OutRefundNo)
		return nil
	case "ABNORMAL":
		// 退款异常，记录日志并处理
		return h.handleRefundAbnormal(ctx, refundNotify)
	default:
		log.Warn("未知的退款状态: %s, 订单号: %s", refundNotify.RefundStatus, refundNotify.OutTradeNo)
		return nil
	}
}

// handlePaymentSuccess 处理支付成功
func (h *CallbackHandler) handlePaymentSuccess(ctx context.Context, payNotify *PayNotification) error {
	log.Info("处理支付成功通知，订单号: %s, 微信订单号: %s, 金额: %d分", 
		payNotify.OutTradeNo, payNotify.TransactionID, payNotify.Amount.Total)
	
	// TODO: 实现具体的业务逻辑
	// 1. 更新订单状态为已支付
	// 2. 记录支付流水
	// 3. 触发后续业务流程（如发货、积分奖励等）
	// 4. 发送支付成功通知给用户
	
	return nil
}

// handlePaymentRefund 处理支付转入退款
func (h *CallbackHandler) handlePaymentRefund(ctx context.Context, payNotify *PayNotification) error {
	log.Info("处理支付转入退款通知，订单号: %s, 微信订单号: %s", 
		payNotify.OutTradeNo, payNotify.TransactionID)
	
	// TODO: 实现具体的业务逻辑
	// 1. 更新订单状态为退款中
	// 2. 记录退款流水
	// 3. 触发退款相关业务流程
	
	return nil
}

// handlePaymentClosed 处理支付关闭
func (h *CallbackHandler) handlePaymentClosed(ctx context.Context, payNotify *PayNotification) error {
	log.Info("处理支付关闭通知，订单号: %s, 微信订单号: %s", 
		payNotify.OutTradeNo, payNotify.TransactionID)
	
	// TODO: 实现具体的业务逻辑
	// 1. 更新订单状态为已关闭
	// 2. 释放库存
	// 3. 记录关闭原因
	
	return nil
}

// handlePaymentRevoked 处理支付撤销
func (h *CallbackHandler) handlePaymentRevoked(ctx context.Context, payNotify *PayNotification) error {
	log.Info("处理支付撤销通知，订单号: %s, 微信订单号: %s", 
		payNotify.OutTradeNo, payNotify.TransactionID)
	
	// TODO: 实现具体的业务逻辑
	// 1. 更新订单状态为已撤销
	// 2. 释放库存
	// 3. 记录撤销原因
	
	return nil
}

// handlePaymentError 处理支付失败
func (h *CallbackHandler) handlePaymentError(ctx context.Context, payNotify *PayNotification) error {
	log.Info("处理支付失败通知，订单号: %s, 微信订单号: %s", 
		payNotify.OutTradeNo, payNotify.TransactionID)
	
	// TODO: 实现具体的业务逻辑
	// 1. 更新订单状态为支付失败
	// 2. 释放库存
	// 3. 记录失败原因
	// 4. 通知用户支付失败
	
	return nil
}

// handleRefundSuccess 处理退款成功
func (h *CallbackHandler) handleRefundSuccess(ctx context.Context, refundNotify *RefundNotification) error {
	log.Info("处理退款成功通知，订单号: %s, 退款单号: %s, 退款金额: %d分", 
		refundNotify.OutTradeNo, refundNotify.OutRefundNo, refundNotify.Amount.Refund)
	
	// TODO: 实现具体的业务逻辑
	// 1. 更新订单状态为已退款
	// 2. 记录退款流水
	// 3. 触发退款后续业务流程
	// 4. 发送退款成功通知给用户
	
	return nil
}

// handleRefundClosed 处理退款关闭
func (h *CallbackHandler) handleRefundClosed(ctx context.Context, refundNotify *RefundNotification) error {
	log.Info("处理退款关闭通知，订单号: %s, 退款单号: %s", 
		refundNotify.OutTradeNo, refundNotify.OutRefundNo)
	
	// TODO: 实现具体的业务逻辑
	// 1. 更新退款状态为已关闭
	// 2. 记录关闭原因
	// 3. 通知相关人员
	
	return nil
}

// handleRefundAbnormal 处理退款异常
func (h *CallbackHandler) handleRefundAbnormal(ctx context.Context, refundNotify *RefundNotification) error {
	log.Error("处理退款异常通知，订单号: %s, 退款单号: %s", 
		refundNotify.OutTradeNo, refundNotify.OutRefundNo)
	
	// TODO: 实现具体的业务逻辑
	// 1. 更新退款状态为异常
	// 2. 记录异常信息
	// 3. 发送异常告警
	// 4. 人工介入处理
	
	return nil
}

// writeSuccessResponse 写入成功响应
func (h *CallbackHandler) writeSuccessResponse(w http.ResponseWriter) {
	response := h.client.NotifyResponse()
	h.writeJSONResponse(w, http.StatusOK, response)
}

// writeErrorResponse 写入错误响应
func (h *CallbackHandler) writeErrorResponse(w http.ResponseWriter, message string) {
	response := h.client.NotifyErrorResponse(message)
	h.writeJSONResponse(w, http.StatusBadRequest, response)
}

// writeJSONResponse 写入JSON响应
func (h *CallbackHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Error("写入响应失败: %v", err)
	}
}

// CallbackConfig 回调配置
type CallbackConfig struct {
	PayNotifyURL    string `json:"pay_notify_url"`    // 支付通知URL
	RefundNotifyURL string `json:"refund_notify_url"` // 退款通知URL
}

// RegisterCallbackRoutes 注册回调路由
func (h *CallbackHandler) RegisterCallbackRoutes(mux *http.ServeMux, config *CallbackConfig) {
	if config.PayNotifyURL != "" {
		mux.HandleFunc(config.PayNotifyURL, h.HandlePayNotify)
		log.Info("注册微信支付通知回调路由: %s", config.PayNotifyURL)
	}
	
	if config.RefundNotifyURL != "" {
		mux.HandleFunc(config.RefundNotifyURL, h.HandleRefundNotify)
		log.Info("注册微信退款通知回调路由: %s", config.RefundNotifyURL)
	}
}