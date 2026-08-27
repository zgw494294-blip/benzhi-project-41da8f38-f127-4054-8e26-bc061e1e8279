package domain

import "fmt"

type ErrorCode string

const (
	CodeValidation ErrorCode = "validation"
	CodeNotFound   ErrorCode = "not_found"
	CodeConflict   ErrorCode = "conflict"
	CodeForbidden  ErrorCode = "forbidden"
	CodeState      ErrorCode = "invalid_state"
)

type BusinessError struct {
	Code    ErrorCode
	Message string
}

func (e *BusinessError) Error() string { return e.Message }

func NewError(code ErrorCode, format string, args ...any) error {
	return &BusinessError{Code: code, Message: fmt.Sprintf(format, args...)}
}

func ErrorCodeOf(err error) ErrorCode {
	if e, ok := err.(*BusinessError); ok {
		return e.Code
	}
	return "internal"
}
