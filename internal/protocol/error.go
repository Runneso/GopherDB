package protocol

type Error struct {
	Code    string    `json:"code"`
	Message string    `json:"message"`
	Pos     *ErrorPos `json:"pos,omitempty"`
}

type ErrorPos struct {
	Offset int `json:"offset,omitempty"`
	Line   int `json:"line,omitempty"`
	Column int `json:"column,omitempty"`
}

func NewError(code, message string) *Error {
	return &Error{Code: code, Message: message}
}

func NewErrorWithPos(code, message string, offset, line, column int) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Pos:     &ErrorPos{Offset: offset, Line: line, Column: column},
	}
}
