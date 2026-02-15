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
