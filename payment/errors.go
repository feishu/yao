package payment

import "fmt"

// ErrorCode 支付错误码
type ErrorCode string

const (
	// 参数错误
	ErrCodeInvalidParams ErrorCode = "INVALID_PARAMS"
	ErrCodeMissingParams ErrorCode = "MISSING_PARAMS"

	// 配置错误
	ErrCodeConfigNotFound     ErrorCode = "CONFIG_NOT_FOUND"
	ErrCodeConfigInvalid      ErrorCode = "CONFIG_INVALID"
	ErrCodeProviderNotFound   ErrorCode = "PROVIDER_NOT_FOUND"
	ErrCodeProviderNotEnabled ErrorCode = "PROVIDER_NOT_ENABLED"

	// 业务错误
	ErrCodeCreateOrderFailed  ErrorCode = "CREATE_ORDER_FAILED"
	ErrCodeQueryOrderFailed   ErrorCode = "QUERY_ORDER_FAILED"
	ErrCodeRefundFailed       ErrorCode = "REFUND_FAILED"
	ErrCodeQueryRefundFailed  ErrorCode = "QUERY_REFUND_FAILED"
	ErrCodeDownloadBillFailed ErrorCode = "DOWNLOAD_BILL_FAILED"

	// 通知错误
	ErrCodeNotifyVerifyFailed ErrorCode = "NOTIFY_VERIFY_FAILED"
	ErrCodeNotifyParseFailed  ErrorCode = "NOTIFY_PARSE_FAILED"
	ErrCodeNotifyHandleFailed ErrorCode = "NOTIFY_HANDLE_FAILED"

	// 系统错误
	ErrCodeInternalError ErrorCode = "INTERNAL_ERROR"
	ErrCodeNetworkError  ErrorCode = "NETWORK_ERROR"
	ErrCodeTimeoutError  ErrorCode = "TIMEOUT_ERROR"
)

// PaymentError 支付错误类型
type PaymentError struct {
	Code    ErrorCode              // 错误码
	Message string                 // 错误消息
	Cause   error                  // 原始错误
	Details map[string]interface{} // 错误详情
}

// Error 实现 error 接口
func (e *PaymentError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 实现 errors.Unwrap 接口
func (e *PaymentError) Unwrap() error {
	return e.Cause
}

// NewPaymentError 创建支付错误
func NewPaymentError(code ErrorCode, message string, cause error) *PaymentError {
	return &PaymentError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Details: make(map[string]interface{}),
	}
}

// WithDetail 添加错误详情
func (e *PaymentError) WithDetail(key string, value interface{}) *PaymentError {
	e.Details[key] = value
	return e
}

// WithDetails 批量添加错误详情
func (e *PaymentError) WithDetails(details map[string]interface{}) *PaymentError {
	for k, v := range details {
		e.Details[k] = v
	}
	return e
}

// IsPaymentError 判断是否为支付错误
func IsPaymentError(err error) bool {
	_, ok := err.(*PaymentError)
	return ok
}

// GetPaymentError 从 error 中提取 PaymentError
func GetPaymentError(err error) *PaymentError {
	if err == nil {
		return nil
	}
	if pe, ok := err.(*PaymentError); ok {
		return pe
	}
	return nil
}

// 错误消息映射（用于国际化）
var errorMessages = map[ErrorCode]string{
	ErrCodeInvalidParams:      "参数无效",
	ErrCodeMissingParams:      "缺少必需参数",
	ErrCodeConfigNotFound:     "配置未找到",
	ErrCodeConfigInvalid:      "配置无效",
	ErrCodeProviderNotFound:   "支付渠道未找到",
	ErrCodeProviderNotEnabled: "支付渠道未启用",
	ErrCodeCreateOrderFailed:  "创建订单失败",
	ErrCodeQueryOrderFailed:   "查询订单失败",
	ErrCodeRefundFailed:       "退款失败",
	ErrCodeQueryRefundFailed:  "查询退款失败",
	ErrCodeDownloadBillFailed: "下载对账单失败",
	ErrCodeNotifyVerifyFailed: "通知验证失败",
	ErrCodeNotifyParseFailed:  "通知解析失败",
	ErrCodeNotifyHandleFailed: "通知处理失败",
	ErrCodeInternalError:      "内部错误",
	ErrCodeNetworkError:       "网络错误",
	ErrCodeTimeoutError:       "请求超时",
}

// GetErrorMessage 获取错误消息
func GetErrorMessage(code ErrorCode) string {
	if msg, ok := errorMessages[code]; ok {
		return msg
	}
	return "未知错误"
}
