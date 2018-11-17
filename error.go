// 错误类型

package api

type Error struct {
	Code    string
	Message string
	Data    interface{}
}

func NewError(code string, message string) *Error {
	return &Error{code, message, nil}
}

func (e *Error) Error() string {
	return e.Code + ": " + e.Message
}

func (e *Error) SetMessage(message string) *Error {
	e.Message = message
	return e
}

func (e *Error) SetData(data interface{}) *Error {
	e.Data = data
	return e
}

func (e *Error) SetErrorData(err error) *Error {
	e.Data = map[string]string{"error": err.Error()}
	return e
}

func IsError(x interface{}) bool {
	if _, ok := x.(*Error); ok {
		return true
	}
	if _, ok := x.(Error); ok {
		return true
	}
	return false
}
