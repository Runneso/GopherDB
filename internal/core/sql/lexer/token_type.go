package lexer

type TokenType int

const (
	EOF TokenType = iota

	// Literals
	IDENT
	NUMBER
	STRING

	// Keywords - DDL
	CREATE
	TABLE
	INDEX
	ON
	USING
	HASH
	BTREE

	// Keywords - DML
	INSERT
	INTO
	VALUES

	SELECT
	FROM
	WHERE

	// Logical operators
	AND
	OR

	// Other keywords
	EXPLAIN

	// Data types
	INT64
	VARCHAR

	// Punctuation
	LPAREN
	RPAREN
	COMMA
	SEMICOLON
	ASTERISK

	// Comparison operators
	EQ
	NE
	LT
	LE
	GT
	GE
)

var tokenTypeNames = [...]string{
	EOF:       "EOF",
	IDENT:     "IDENT",
	NUMBER:    "NUMBER",
	STRING:    "STRING",
	CREATE:    "CREATE",
	TABLE:     "TABLE",
	INDEX:     "INDEX",
	ON:        "ON",
	USING:     "USING",
	HASH:      "HASH",
	BTREE:     "BTREE",
	INSERT:    "INSERT",
	INTO:      "INTO",
	VALUES:    "VALUES",
	SELECT:    "SELECT",
	FROM:      "FROM",
	WHERE:     "WHERE",
	AND:       "AND",
	OR:        "OR",
	EXPLAIN:   "EXPLAIN",
	INT64:     "INT64",
	VARCHAR:   "VARCHAR",
	LPAREN:    "LPAREN",
	RPAREN:    "RPAREN",
	COMMA:     "COMMA",
	SEMICOLON: "SEMICOLON",
	ASTERISK:  "ASTERISK",
	EQ:        "EQ",
	NE:        "NE",
	LT:        "LT",
	LE:        "LE",
	GT:        "GT",
	GE:        "GE",
}

func (token TokenType) String() string {
	if int(token) < len(tokenTypeNames) {
		return tokenTypeNames[token]
	}
	return "UNKNOWN"
}
