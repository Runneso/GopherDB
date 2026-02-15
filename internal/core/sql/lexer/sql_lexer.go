package lexer

import (
	"strings"
)

var keywords = map[string]TokenType{
	"CREATE":  CREATE,
	"TABLE":   TABLE,
	"INDEX":   INDEX,
	"ON":      ON,
	"USING":   USING,
	"HASH":    HASH,
	"BTREE":   BTREE,
	"INSERT":  INSERT,
	"INTO":    INTO,
	"VALUES":  VALUES,
	"SELECT":  SELECT,
	"FROM":    FROM,
	"WHERE":   WHERE,
	"AND":     AND,
	"OR":      OR,
	"EXPLAIN": EXPLAIN,
	"INT64":   INT64,
	"VARCHAR": VARCHAR,
}

type handlerFn func(*SqlLexer, int, int, int) (*Token, error)

func handleSingle(tokenType TokenType, text string) handlerFn {
	return func(lexer *SqlLexer, startPos, startLine, startCol int) (*Token, error) {
		lexer.advance()
		return NewToken(tokenType, text, startPos, lexer.pos, startLine, startCol), nil
	}
}

func handleLT(lexer *SqlLexer, startPos, startLine, startCol int) (*Token, error) {
	lexer.advance()

	switch lexer.peek() {
	case '=':
		lexer.advance()
		return NewToken(LE, "<=", startPos, lexer.pos, startLine, startCol), nil
	case '>':
		lexer.advance()
		return NewToken(NE, "<>", startPos, lexer.pos, startLine, startCol), nil
	default:
		return NewToken(LT, "<", startPos, lexer.pos, startLine, startCol), nil
	}
}

func handleGT(lexer *SqlLexer, startPos, startLine, startCol int) (*Token, error) {
	lexer.advance()

	if lexer.peek() == '=' {
		lexer.advance()
		return NewToken(GE, ">=", startPos, lexer.pos, startLine, startCol), nil
	}
	return NewToken(GT, ">", startPos, lexer.pos, startLine, startCol), nil
}

func handleBang(lexer *SqlLexer, startPos, startLine, startCol int) (*Token, error) {
	lexer.advance()

	if lexer.peek() == '=' {
		lexer.advance()
		return NewToken(NE, "!=", startPos, lexer.pos, startLine, startCol), nil
	}
	return nil, NewSqlSyntaxError("Unexpected character: '!'", startPos, startLine, startCol)
}

type SqlLexer struct {
	input   []rune
	handler map[rune]handlerFn
	pos     int
	line    int
	col     int
}

func NewSqlLexer() *SqlLexer {
	lexer := &SqlLexer{}
	lexer.handler = map[rune]handlerFn{
		'(': handleSingle(LPAREN, "("),
		')': handleSingle(RPAREN, ")"),
		',': handleSingle(COMMA, ","),
		';': handleSingle(SEMICOLON, ";"),
		'*': handleSingle(ASTERISK, "*"),
		'=': handleSingle(EQ, "="),

		'<': handleLT,
		'>': handleGT,
		'!': handleBang,
	}
	return lexer
}

func (lexer *SqlLexer) Tokenize(sql string) ([]*Token, error) {
	lexer.input = []rune(sql)
	lexer.pos = 0
	lexer.line = 1
	lexer.col = 1

	var out []*Token

	for !lexer.eof() {
		char := lexer.peek()

		if lexer.isWhitespace(char) {
			lexer.consumeWhitespace()
			continue
		}

		if lexer.isIdentStart(char) {
			token := lexer.readIdentOrKeyword()
			out = append(out, token)
			continue
		}

		if lexer.isDigit(char) || (char == '-' && lexer.isDigit(lexer.peekNext())) {
			token := lexer.readNumber()
			out = append(out, token)
			continue
		}

		if char == '\'' {
			token, err := lexer.readString()
			if err != nil {
				return nil, err
			}
			out = append(out, token)
			continue
		}

		startPos := lexer.pos
		startLine := lexer.line
		startCol := lexer.col
		handler, ok := lexer.handler[char]

		if !ok {
			return nil, NewSqlSyntaxError("Unexpected character: '"+string(char)+"'", startPos, startLine, startCol)
		}

		token, err := handler(lexer, startPos, startLine, startCol)

		if err != nil {
			return nil, err
		}

		out = append(out, token)
	}

	out = append(out, NewToken(EOF, "", lexer.pos, lexer.pos, lexer.line, lexer.col))
	return out, nil
}

func (lexer *SqlLexer) readIdentOrKeyword() *Token {
	startPos := lexer.pos
	startLine := lexer.line
	startCol := lexer.col

	var builder strings.Builder
	for !lexer.eof() && lexer.isIdentPart(lexer.peek()) {
		builder.WriteRune(lexer.peek())
		lexer.advance()
	}

	text := builder.String()
	tokenType := IDENT
	if keyWord, ok := keywords[strings.ToUpper(text)]; ok {
		tokenType = keyWord
	}
	return NewToken(tokenType, text, startPos, lexer.pos, startLine, startCol)
}

func (lexer *SqlLexer) readNumber() *Token {
	startPos := lexer.pos
	startLine := lexer.line
	startCol := lexer.col

	var builder strings.Builder
	if lexer.peek() == '-' {
		builder.WriteRune('-')
		lexer.advance()
	}
	for !lexer.eof() && lexer.isDigit(lexer.peek()) {
		builder.WriteRune(lexer.peek())
		lexer.advance()
	}
	return NewToken(NUMBER, builder.String(), startPos, lexer.pos, startLine, startCol)
}

func (lexer *SqlLexer) readString() (*Token, error) {
	startPos := lexer.pos
	startLine := lexer.line
	startCol := lexer.col

	lexer.advance()
	var builder strings.Builder

	for !lexer.eof() {
		char := lexer.peek()
		if char == '\'' {
			if lexer.peekNext() == '\'' {
				builder.WriteRune('\'')
				lexer.advance()
				lexer.advance()
				continue
			}
			break
		}
		builder.WriteRune(char)
		lexer.advance()
	}

	if lexer.eof() {
		return nil, NewSqlSyntaxError("Unterminated string literal", startPos, startLine, startCol)
	}

	lexer.advance()
	return NewToken(STRING, builder.String(), startPos, lexer.pos, startLine, startCol), nil
}

func (lexer *SqlLexer) consumeWhitespace() {
	for !lexer.eof() && lexer.isWhitespace(lexer.peek()) {
		lexer.advance()
	}
}

func (lexer *SqlLexer) eof() bool {
	return lexer.pos >= len(lexer.input)
}

func (lexer *SqlLexer) peek() rune {
	if lexer.eof() {
		return 0
	}
	return lexer.input[lexer.pos]
}

func (lexer *SqlLexer) peekNext() rune {
	if lexer.pos+1 < len(lexer.input) {
		return lexer.input[lexer.pos+1]
	}
	return 0
}

func (lexer *SqlLexer) advance() {
	if lexer.eof() {
		return
	}
	char := lexer.input[lexer.pos]
	lexer.pos++
	if char == '\n' {
		lexer.line++
		lexer.col = 1
	} else {
		lexer.col++
	}
}

func (lexer *SqlLexer) isWhitespace(char rune) bool {
	return char == ' ' || char == '\t' || char == '\r' || char == '\n'
}

func (lexer *SqlLexer) isDigit(char rune) bool {
	return char >= '0' && char <= '9'
}

func (lexer *SqlLexer) isIdentStart(char rune) bool {
	return (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') || char == '_'
}

func (lexer *SqlLexer) isIdentPart(char rune) bool {
	return lexer.isIdentStart(char) || lexer.isDigit(char)
}
