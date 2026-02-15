package lexer

import (
	"testing"
)

func TestTokenizeSimpleSelect(t *testing.T) {
	lexer := NewSqlLexer()
	tokens, err := lexer.Tokenize("SELECT * FROM users")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []struct {
		tokenType TokenType
		text      string
	}{
		{SELECT, "SELECT"},
		{ASTERISK, "*"},
		{FROM, "FROM"},
		{IDENT, "users"},
		{EOF, ""},
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, exp := range expected {
		if tokens[i].Type() != exp.tokenType {
			t.Errorf("token %d: expected type %v, got %v", i, exp.tokenType, tokens[i].Type())
		}
		if tokens[i].Text() != exp.text {
			t.Errorf("token %d: expected text %q, got %q", i, exp.text, tokens[i].Text())
		}
	}
}

func TestTokenizeCreateTable(t *testing.T) {
	lexer := NewSqlLexer()
	tokens, err := lexer.Tokenize("CREATE TABLE users (id INT64, name VARCHAR)")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []struct {
		tokenType TokenType
		text      string
	}{
		{CREATE, "CREATE"},
		{TABLE, "TABLE"},
		{IDENT, "users"},
		{LPAREN, "("},
		{IDENT, "id"},
		{INT64, "INT64"},
		{COMMA, ","},
		{IDENT, "name"},
		{VARCHAR, "VARCHAR"},
		{RPAREN, ")"},
		{EOF, ""},
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, exp := range expected {
		if tokens[i].Type() != exp.tokenType {
			t.Errorf("token %d: expected type %v, got %v", i, exp.tokenType, tokens[i].Type())
		}
		if tokens[i].Text() != exp.text {
			t.Errorf("token %d: expected text %q, got %q", i, exp.text, tokens[i].Text())
		}
	}
}

func TestTokenizeInsert(t *testing.T) {
	lexer := NewSqlLexer()
	tokens, err := lexer.Tokenize("INSERT INTO users VALUES (1, 'John')")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []struct {
		tokenType TokenType
		text      string
	}{
		{INSERT, "INSERT"},
		{INTO, "INTO"},
		{IDENT, "users"},
		{VALUES, "VALUES"},
		{LPAREN, "("},
		{NUMBER, "1"},
		{COMMA, ","},
		{STRING, "John"},
		{RPAREN, ")"},
		{EOF, ""},
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, exp := range expected {
		if tokens[i].Type() != exp.tokenType {
			t.Errorf("token %d: expected type %v, got %v", i, exp.tokenType, tokens[i].Type())
		}
		if tokens[i].Text() != exp.text {
			t.Errorf("token %d: expected text %q, got %q", i, exp.text, tokens[i].Text())
		}
	}
}

func TestTokenizeSelectWithWhere(t *testing.T) {
	lexer := NewSqlLexer()
	tokens, err := lexer.Tokenize("SELECT * FROM users WHERE id = 1 AND name <> 'test'")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []struct {
		tokenType TokenType
		text      string
	}{
		{SELECT, "SELECT"},
		{ASTERISK, "*"},
		{FROM, "FROM"},
		{IDENT, "users"},
		{WHERE, "WHERE"},
		{IDENT, "id"},
		{EQ, "="},
		{NUMBER, "1"},
		{AND, "AND"},
		{IDENT, "name"},
		{NE, "<>"},
		{STRING, "test"},
		{EOF, ""},
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, exp := range expected {
		if tokens[i].Type() != exp.tokenType {
			t.Errorf("token %d: expected type %v, got %v", i, exp.tokenType, tokens[i].Type())
		}
		if tokens[i].Text() != exp.text {
			t.Errorf("token %d: expected text %q, got %q", i, exp.text, tokens[i].Text())
		}
	}
}

func TestTokenizeComparisonOperators(t *testing.T) {
	lexer := NewSqlLexer()
	tokens, err := lexer.Tokenize("a = b < c <= d > e >= f <> g != h")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []TokenType{
		IDENT, EQ, IDENT, LT, IDENT, LE, IDENT, GT, IDENT, GE, IDENT, NE, IDENT, NE, IDENT, EOF,
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, exp := range expected {
		if tokens[i].Type() != exp {
			t.Errorf("token %d: expected type %v, got %v", i, exp, tokens[i].Type())
		}
	}
}

func TestTokenizeNegativeNumber(t *testing.T) {
	lexer := NewSqlLexer()
	tokens, err := lexer.Tokenize("SELECT -123")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(tokens))
	}

	if tokens[1].Type() != NUMBER || tokens[1].Text() != "-123" {
		t.Errorf("expected NUMBER(-123), got %v(%s)", tokens[1].Type(), tokens[1].Text())
	}
}

func TestTokenizeEscapedString(t *testing.T) {
	lexer := NewSqlLexer()
	tokens, err := lexer.Tokenize("'It''s a test'")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(tokens))
	}

	if tokens[0].Type() != STRING || tokens[0].Text() != "It's a test" {
		t.Errorf("expected STRING(It's a test), got %v(%s)", tokens[0].Type(), tokens[0].Text())
	}
}

func TestTokenizeUnterminatedString(t *testing.T) {
	lexer := NewSqlLexer()
	_, err := lexer.Tokenize("'unterminated")

	if err == nil {
		t.Fatal("expected error for unterminated string")
	}

	syntaxErr, ok := err.(*SqlSyntaxError)
	if !ok {
		t.Fatalf("expected SqlSyntaxError, got %T", err)
	}

	if syntaxErr.Line() != 1 {
		t.Errorf("expected line 1, got %d", syntaxErr.Line())
	}
}

func TestTokenizeUnexpectedCharacter(t *testing.T) {
	lexer := NewSqlLexer()
	_, err := lexer.Tokenize("SELECT @")

	if err == nil {
		t.Fatal("expected error for unexpected character")
	}

	_, ok := err.(*SqlSyntaxError)
	if !ok {
		t.Fatalf("expected SqlSyntaxError, got %T", err)
	}
}

func TestTokenizeMultiline(t *testing.T) {
	lexer := NewSqlLexer()
	tokens, err := lexer.Tokenize("SELECT\n*\nFROM\nusers")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tokens[0].Line() != 1 {
		t.Errorf("SELECT should be on line 1, got %d", tokens[0].Line())
	}
	if tokens[1].Line() != 2 {
		t.Errorf("* should be on line 2, got %d", tokens[1].Line())
	}
	if tokens[2].Line() != 3 {
		t.Errorf("FROM should be on line 3, got %d", tokens[2].Line())
	}
	if tokens[3].Line() != 4 {
		t.Errorf("users should be on line 4, got %d", tokens[3].Line())
	}
}

func TestTokenizeCreateIndex(t *testing.T) {
	lexer := NewSqlLexer()
	tokens, err := lexer.Tokenize("CREATE INDEX idx ON users USING BTREE (id)")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []struct {
		tokenType TokenType
		text      string
	}{
		{CREATE, "CREATE"},
		{INDEX, "INDEX"},
		{IDENT, "idx"},
		{ON, "ON"},
		{IDENT, "users"},
		{USING, "USING"},
		{BTREE, "BTREE"},
		{LPAREN, "("},
		{IDENT, "id"},
		{RPAREN, ")"},
		{EOF, ""},
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, exp := range expected {
		if tokens[i].Type() != exp.tokenType {
			t.Errorf("token %d: expected type %v, got %v", i, exp.tokenType, tokens[i].Type())
		}
		if tokens[i].Text() != exp.text {
			t.Errorf("token %d: expected text %q, got %q", i, exp.text, tokens[i].Text())
		}
	}
}

func TestTokenizeCaseInsensitiveKeywords(t *testing.T) {
	lexer := NewSqlLexer()
	tokens, err := lexer.Tokenize("select FROM Select")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tokens[0].Type() != SELECT || tokens[0].Text() != "select" {
		t.Errorf("expected SELECT(select), got %v(%s)", tokens[0].Type(), tokens[0].Text())
	}
	if tokens[1].Type() != FROM || tokens[1].Text() != "FROM" {
		t.Errorf("expected FROM(FROM), got %v(%s)", tokens[1].Type(), tokens[1].Text())
	}
	if tokens[2].Type() != SELECT || tokens[2].Text() != "Select" {
		t.Errorf("expected SELECT(Select), got %v(%s)", tokens[2].Type(), tokens[2].Text())
	}
}

func TestTokenizeEmptyInput(t *testing.T) {
	lexer := NewSqlLexer()
	tokens, err := lexer.Tokenize("")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tokens) != 1 || tokens[0].Type() != EOF {
		t.Errorf("expected single EOF token")
	}
}

func TestTokenizeSemicolon(t *testing.T) {
	lexer := NewSqlLexer()
	tokens, err := lexer.Tokenize("SELECT 1;")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []TokenType{SELECT, NUMBER, SEMICOLON, EOF}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, exp := range expected {
		if tokens[i].Type() != exp {
			t.Errorf("token %d: expected type %v, got %v", i, exp, tokens[i].Type())
		}
	}
}
