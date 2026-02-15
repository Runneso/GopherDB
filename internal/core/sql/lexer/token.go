package lexer

type Token struct {
	tokenType   TokenType
	text        string
	startOffset int
	endOffset   int
	line        int
	column      int
}

func NewToken(tokenType TokenType, text string, startOffset, endOffset, line, column int) *Token {
	return &Token{
		tokenType:   tokenType,
		text:        text,
		startOffset: startOffset,
		endOffset:   endOffset,
		line:        line,
		column:      column,
	}
}

func (token *Token) Type() TokenType {
	return token.tokenType
}

func (token *Token) Text() string {
	return token.text
}

func (token *Token) StartOffset() int {
	return token.startOffset
}

func (token *Token) EndOffset() int {
	return token.endOffset
}

func (token *Token) Line() int {
	return token.line
}

func (token *Token) Column() int {
	return token.column
}

func (token *Token) String() string {
	return token.tokenType.String() + "(" + token.text + ")"
}
