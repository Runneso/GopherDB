package engine

type SessionContext struct {
	sessionID string
	requestID string
	trace     bool
}

func NewSessionContext(sessionID, requestID string, trace bool) *SessionContext {
	return &SessionContext{
		sessionID: sessionID,
		requestID: requestID,
		trace:     trace,
	}
}

func (ctx *SessionContext) SessionID() string {
	return ctx.sessionID
}

func (ctx *SessionContext) RequestID() string {
	return ctx.requestID
}

func (ctx *SessionContext) Trace() bool {
	return ctx.trace
}
