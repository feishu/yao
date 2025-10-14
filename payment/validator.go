package payment

import "fmt"

// Validator 验证器接口
type Validator interface {
	Validate() error
}

// Validate 验证创建订单参数
func (p *CreateOrderParams) Validate() error {
	if p.MerchantNo == "" {
		return NewPaymentError(ErrCodeMissingParams, "merchant_no is required", nil)
	}

	if p.OutTradeNo == "" {
		return NewPaymentError(ErrCodeMissingParams, "out_trade_no is required", nil)
	}

	if p.Channel == "" {
		return NewPaymentError(ErrCodeMissingParams, "channel is required", nil)
	}

	if p.TradeType == "" {
		return NewPaymentError(ErrCodeMissingParams, "trade_type is required", nil)
	}

	if p.Amount <= 0 {
		return NewPaymentError(ErrCodeInvalidParams, "amount must be greater than 0", nil).
			WithDetail("amount", p.Amount)
	}

	if p.Subject == "" {
		return NewPaymentError(ErrCodeMissingParams, "subject is required", nil)
	}

	if p.NotifyURL == "" {
		return NewPaymentError(ErrCodeMissingParams, "notify_url is required", nil)
	}

	// 验证支付渠道
	if !isValidPaymentChannel(p.Channel) {
		return NewPaymentError(ErrCodeInvalidParams, "invalid payment channel", nil).
			WithDetail("channel", p.Channel)
	}

	// 验证交易类型
	if !isValidTradeType(p.TradeType) {
		return NewPaymentError(ErrCodeInvalidParams, "invalid trade type", nil).
			WithDetail("trade_type", p.TradeType)
	}

	return nil
}

// Validate 验证查询订单参数
func (p *QueryOrderParams) Validate() error {
	if p.MerchantNo == "" {
		return NewPaymentError(ErrCodeMissingParams, "merchant_no is required", nil)
	}

	if p.Channel == "" {
		return NewPaymentError(ErrCodeMissingParams, "channel is required", nil)
	}

	if p.OutTradeNo == "" && p.TransactionID == "" {
		return NewPaymentError(ErrCodeMissingParams, "either out_trade_no or transaction_id is required", nil)
	}

	// 验证支付渠道
	if !isValidPaymentChannel(p.Channel) {
		return NewPaymentError(ErrCodeInvalidParams, "invalid payment channel", nil).
			WithDetail("channel", p.Channel)
	}

	return nil
}

// Validate 验证创建退款参数
func (p *CreateRefundParams) Validate() error {
	if p.MerchantNo == "" {
		return NewPaymentError(ErrCodeMissingParams, "merchant_no is required", nil)
	}

	if p.Channel == "" {
		return NewPaymentError(ErrCodeMissingParams, "channel is required", nil)
	}

	if p.OutRefundNo == "" {
		return NewPaymentError(ErrCodeMissingParams, "out_refund_no is required", nil)
	}

	if p.RefundAmount <= 0 {
		return NewPaymentError(ErrCodeInvalidParams, "refund_amount must be greater than 0", nil).
			WithDetail("refund_amount", p.RefundAmount)
	}

	if p.TotalAmount <= 0 {
		return NewPaymentError(ErrCodeInvalidParams, "total_amount must be greater than 0", nil).
			WithDetail("total_amount", p.TotalAmount)
	}

	if p.RefundAmount > p.TotalAmount {
		return NewPaymentError(ErrCodeInvalidParams, "refund_amount cannot exceed total_amount", nil).
			WithDetails(map[string]interface{}{
				"refund_amount": p.RefundAmount,
				"total_amount":  p.TotalAmount,
			})
	}

	if p.OutTradeNo == "" && p.TransactionID == "" {
		return NewPaymentError(ErrCodeMissingParams, "either out_trade_no or transaction_id is required", nil)
	}

	// 验证支付渠道
	if !isValidPaymentChannel(p.Channel) {
		return NewPaymentError(ErrCodeInvalidParams, "invalid payment channel", nil).
			WithDetail("channel", p.Channel)
	}

	return nil
}

// Validate 验证查询退款参数
func (p *QueryRefundParams) Validate() error {
	if p.MerchantNo == "" {
		return NewPaymentError(ErrCodeMissingParams, "merchant_no is required", nil)
	}

	if p.Channel == "" {
		return NewPaymentError(ErrCodeMissingParams, "channel is required", nil)
	}

	if p.OutRefundNo == "" && p.RefundID == "" {
		return NewPaymentError(ErrCodeMissingParams, "either out_refund_no or refund_id is required", nil)
	}

	// 验证支付渠道
	if !isValidPaymentChannel(p.Channel) {
		return NewPaymentError(ErrCodeInvalidParams, "invalid payment channel", nil).
			WithDetail("channel", p.Channel)
	}

	return nil
}

// Validate 验证下载对账单参数
func (p *DownloadBillParams) Validate() error {
	if p.MerchantNo == "" {
		return NewPaymentError(ErrCodeMissingParams, "merchant_no is required", nil)
	}

	if p.Channel == "" {
		return NewPaymentError(ErrCodeMissingParams, "channel is required", nil)
	}

	if p.BillDate == "" {
		return NewPaymentError(ErrCodeMissingParams, "bill_date is required", nil)
	}

	// 验证支付渠道
	if !isValidPaymentChannel(p.Channel) {
		return NewPaymentError(ErrCodeInvalidParams, "invalid payment channel", nil).
			WithDetail("channel", p.Channel)
	}

	// 验证对账单类型（可选）
	if p.BillType != "" && !isValidBillType(p.BillType) {
		return NewPaymentError(ErrCodeInvalidParams, "invalid bill type", nil).
			WithDetail("bill_type", p.BillType)
	}

	return nil
}

// Validate 验证对账参数
func (p *ReconcileParams) Validate() error {
	if p.MerchantNo == "" {
		return NewPaymentError(ErrCodeMissingParams, "merchant_no is required", nil)
	}

	if p.Channel == "" {
		return NewPaymentError(ErrCodeMissingParams, "channel is required", nil)
	}

	if p.BillDate == "" {
		return NewPaymentError(ErrCodeMissingParams, "bill_date is required", nil)
	}

	// 验证支付渠道
	if !isValidPaymentChannel(p.Channel) {
		return NewPaymentError(ErrCodeInvalidParams, "invalid payment channel", nil).
			WithDetail("channel", p.Channel)
	}

	return nil
}

// ValidateAlipayConfig 验证支付宝配置
func ValidateAlipayConfig(config map[string]interface{}) error {
	requiredFields := []string{"app_id", "private_key", "public_key"}

	for _, field := range requiredFields {
		value, exists := config[field]
		if !exists {
			return NewPaymentError(ErrCodeConfigInvalid, 
				fmt.Sprintf("alipay config field '%s' is required", field), nil).
				WithDetail("field", field)
		}

		if strValue, ok := value.(string); !ok || strValue == "" {
			return NewPaymentError(ErrCodeConfigInvalid,
				fmt.Sprintf("alipay config field '%s' must be a non-empty string", field), nil).
				WithDetail("field", field)
		}
	}

	return nil
}

// ValidateWechatConfig 验证微信配置
func ValidateWechatConfig(config map[string]interface{}) error {
	requiredFields := []string{"app_id", "mch_id", "apiv3_key", "private_key", "serial_no"}

	for _, field := range requiredFields {
		value, exists := config[field]
		if !exists {
			return NewPaymentError(ErrCodeConfigInvalid,
				fmt.Sprintf("wechat config field '%s' is required", field), nil).
				WithDetail("field", field)
		}

		if strValue, ok := value.(string); !ok || strValue == "" {
			return NewPaymentError(ErrCodeConfigInvalid,
				fmt.Sprintf("wechat config field '%s' must be a non-empty string", field), nil).
				WithDetail("field", field)
		}
	}

	return nil
}
