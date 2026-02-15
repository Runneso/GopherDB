package parser

import (
	"testing"

	"GopherDB/internal/core/sql/ast"
	"GopherDB/internal/core/sql/lexer"
	"GopherDB/internal/core/types"
)

func parse(t *testing.T, sql string) ast.Statement {
	tokens, err := lexer.NewSqlLexer().Tokenize(sql)
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}
	stmt, err := NewSqlParser().Parse(tokens)
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}
	return stmt
}

func TestCreateTable(t *testing.T) {
	stmt := parse(t, "CREATE TABLE users (id INT64, name VARCHAR)").(*ast.CreateTableStmt)
	if stmt.TableName().Text() != "users" || len(stmt.Columns()) != 2 {
		t.Error("CREATE TABLE parsing failed")
	}
}

func TestCreateIndex(t *testing.T) {
	stmt := parse(t, "CREATE INDEX idx ON users (id) USING BTREE").(*ast.CreateIndexStmt)
	if stmt.IndexType() != types.IndexTypeBTree {
		t.Error("expected BTREE")
	}
}

func TestInsert(t *testing.T) {
	stmt := parse(t, "INSERT INTO users VALUES (1, 'John')").(*ast.InsertStmt)
	if len(stmt.Values()) != 2 {
		t.Error("expected 2 values")
	}
}

func TestSelectAll(t *testing.T) {
	stmt := parse(t, "SELECT * FROM users").(*ast.SelectStmt)
	if !stmt.SelectAll() {
		t.Error("expected SelectAll")
	}
}

func TestSelectWhere(t *testing.T) {
	stmt := parse(t, "SELECT * FROM users WHERE id = 1").(*ast.SelectStmt)
	bin := stmt.Where().(*ast.BinaryExpr)
	if bin.Op() != "=" {
		t.Error("expected =")
	}
}

func TestPrecedence(t *testing.T) {
	stmt := parse(t, "SELECT * FROM t WHERE a = 1 OR b = 2 AND c = 3").(*ast.SelectStmt)
	or := stmt.Where().(*ast.BinaryExpr)
	if or.Op() != "OR" {
		t.Error("expected OR at top")
	}
}

func TestParentheses(t *testing.T) {
	stmt := parse(t, "SELECT * FROM t WHERE (a = 1 OR b = 2) AND c = 3").(*ast.SelectStmt)
	and := stmt.Where().(*ast.BinaryExpr)
	if and.Op() != "AND" {
		t.Error("expected AND at top")
	}
}

func TestExplain(t *testing.T) {
	stmt := parse(t, "EXPLAIN SELECT * FROM users").(*ast.ExplainStmt)
	if _, ok := stmt.Inner().(*ast.SelectStmt); !ok {
		t.Error("expected SelectStmt inside")
	}
}

func TestComparisonOps(t *testing.T) {
	ops := map[string]string{"=": "=", "<>": "<>", "!=": "<>", "<": "<", "<=": "<=", ">": ">", ">=": ">="}
	for sql, expected := range ops {
		stmt := parse(t, "SELECT * FROM t WHERE a "+sql+" 1").(*ast.SelectStmt)
		if stmt.Where().(*ast.BinaryExpr).Op() != expected {
			t.Errorf("expected %s", expected)
		}
	}
}
