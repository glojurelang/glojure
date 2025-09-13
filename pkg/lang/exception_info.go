package lang

import "fmt"

type ExceptionInfo struct {
	message string
	data    IPersistentMap
	cause   error
}

var _ IExceptionInfo = (*ExceptionInfo)(nil)
var _ error = (*ExceptionInfo)(nil)

func NewExceptionInfo(msg string, data IPersistentMap) *ExceptionInfo {
	return &ExceptionInfo{
		message: msg,
		data:    data,
	}
}

func NewExceptionInfoWithCause(msg string, data IPersistentMap, cause error) *ExceptionInfo {
	return &ExceptionInfo{
		message: msg,
		data:    data,
		cause:   cause,
	}
}

func (e *ExceptionInfo) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.message, e.cause)
	}
	return e.message
}

func (e *ExceptionInfo) GetData() IPersistentMap {
	return e.data
}

func (e *ExceptionInfo) Unwrap() error {
	return e.cause
}

func (e *ExceptionInfo) Message() string {
	return e.message
}

func (e *ExceptionInfo) Cause() error {
	return e.cause
}