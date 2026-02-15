package protocol

type Request struct {
	Type      string `json:"type"`
	RequestID string `json:"requestId"`
	SQL       string `json:"sql"`
	Trace     bool   `json:"trace"`
}

func NewRequest(requestID, sql string, trace bool) *Request {
	return &Request{
		Type:      "query",
		RequestID: requestID,
		SQL:       sql,
		Trace:     trace,
	}
}
