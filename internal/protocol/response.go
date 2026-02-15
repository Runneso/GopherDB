package protocol

type Response struct {
	Type      string   `json:"type"`
	RequestID string   `json:"requestId"`
	Status    string   `json:"status"`
	Columns   []string `json:"columns,omitempty"`
	Rows      [][]any  `json:"rows,omitempty"`
	Affected  int      `json:"affected,omitempty"`
	Explain   string   `json:"explain,omitempty"`
	Error     *Error   `json:"error,omitempty"`
}

func NewOkResponse(requestID string, columns []string, rows [][]any, affected int, explain string) *Response {
	return &Response{
		Type:      "response",
		RequestID: requestID,
		Status:    "ok",
		Columns:   columns,
		Rows:      rows,
		Affected:  affected,
		Explain:   explain,
	}
}

func NewErrorResponse(requestID string, err *Error) *Response {
	return &Response{
		Type:      "response",
		RequestID: requestID,
		Status:    "error",
		Error:     err,
	}
}
