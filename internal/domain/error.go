package domain

import "errors"

// 公共业务错误，带业务上下文，禁止吞错或空返回。
var (
	// ErrNotFound 资源不存在。
	ErrNotFound = errors.New("资源不存在")
	// ErrConflict 资源冲突，例如唯一约束违反。
	ErrConflict = errors.New("资源冲突")
	// ErrValidation 请求参数或业务校验失败。
	ErrValidation = errors.New("请求参数无效")
	// ErrUnauthorized 未认证或会话失效。
	ErrUnauthorized = errors.New("未认证或会话已过期")
	// ErrForbidden 已认证但缺少权限。
	ErrForbidden = errors.New("禁止访问：权限不足")
	// ErrRateLimited 触发登录限流。
	ErrRateLimited = errors.New("操作过于频繁，请稍后再试")
	// ErrIdempotentConflict 幂等键重复请求，但请求体不一致。
	ErrIdempotentConflict = errors.New("幂等键重复且请求不一致")
	// ErrStateTransition 非法状态流转。
	ErrStateTransition = errors.New("非法状态流转")
	// ErrMileageNotIncreasing 里程必须单调递增。
	ErrMileageNotIncreasing = errors.New("里程必须单调递增")
	// ErrInvariant 业务不变量被违反。
	ErrInvariant = errors.New("业务不变量被违反")
)

// CodedError 携带稳定错误码的业务错误，用于统一 JSON 错误响应。
type CodedError struct {
	Code    string
	Message string
	Err     error
}

// Error 返回具体业务消息；若未设置 Message 则回退到底层错误文本。
// 必须优先返回 Message，否则所有 NewCoded 包装的错误都会丢失具体原因，
// 例如 NewCoded("validation_error","幂等键不能为空",ErrValidation) 会错误地只返回"请求参数无效"。
func (e *CodedError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "未知错误"
}

func (e *CodedError) Unwrap() error {
	if e.Err != nil {
		return e.Err
	}
	return nil
}

// NewCoded 构造一个带码错误。
func NewCoded(code, message string, err error) *CodedError {
	return &CodedError{Code: code, Message: message, Err: err}
}

// Is 适配 errors.Is，按底层错误或码匹配。
func (e *CodedError) Is(target error) bool {
	if e == target {
		return true
	}
	if e.Err != nil && errors.Is(e.Err, target) {
		return true
	}
	if t, ok := target.(*CodedError); ok {
		return e.Code == t.Code
	}
	return false
}
