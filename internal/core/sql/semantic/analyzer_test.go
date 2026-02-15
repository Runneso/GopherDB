package semantic

import (
	"testing"

	"GopherDB/internal/core/catalog/model"
	"GopherDB/internal/core/sql/ast"
	"GopherDB/internal/core/sql/lexer"
	"GopherDB/internal/core/sql/parser"
	"GopherDB/internal/core/types"
)

type colDef struct {
	name    string
	typeOid int32
}

type mockCatalog struct {
	tables  map[string]*model.TableDefinition
	columns map[int32][]*model.ColumnDefinition
	types   map[string]*model.TypeDefinition
	indexes map[string]*model.IndexDefinition
}

func newMockCatalog() *mockCatalog {
	int64Type, _ := model.NewTypeDefinition(1, "INT64", 8)
	varcharType, _ := model.NewTypeDefinition(2, "VARCHAR", -1)

	return &mockCatalog{
		tables:  make(map[string]*model.TableDefinition),
		columns: make(map[int32][]*model.ColumnDefinition),
		types: map[string]*model.TypeDefinition{
			"INT64":   int64Type,
			"VARCHAR": varcharType,
		},
		indexes: make(map[string]*model.IndexDefinition),
	}
}

func (m *mockCatalog) addTable(name string, cols []colDef) {
	table, _ := model.NewTableDefinition(int32(len(m.tables)+1), name, "table", name+".dat", 0)
	m.tables[name] = table
	for i, c := range cols {
		col, _ := model.NewColumnDefinition(int32(i+1), table.Oid(), c.typeOid, c.name, int32(i))
		m.columns[table.Oid()] = append(m.columns[table.Oid()], col)
	}
}

func (m *mockCatalog) CreateTable(_ string, _ []*model.ColumnDefinition) (*model.TableDefinition, error) {
	return nil, nil
}

func (m *mockCatalog) GetTable(name string) (*model.TableDefinition, error) {
	return m.tables[name], nil
}

func (m *mockCatalog) ListTables() ([]*model.TableDefinition, error) {
	return nil, nil
}

func (m *mockCatalog) GetColumns(table *model.TableDefinition) ([]*model.ColumnDefinition, error) {
	return m.columns[table.Oid()], nil
}

func (m *mockCatalog) GetColumn(table *model.TableDefinition, name string) (*model.ColumnDefinition, error) {
	for _, col := range m.columns[table.Oid()] {
		if col.Name() == name {
			return col, nil
		}
	}
	return nil, nil
}

func (m *mockCatalog) GetTypeByOid(oid int32) (*model.TypeDefinition, error) {
	for _, t := range m.types {
		if t.Oid() == oid {
			return t, nil
		}
	}
	return nil, nil
}

func (m *mockCatalog) GetTypeByName(name string) (*model.TypeDefinition, error) {
	return m.types[name], nil
}

func (m *mockCatalog) UpdatePagesCount(_ *model.TableDefinition, _ int32) error {
	return nil
}

func (m *mockCatalog) CreateIndex(_, _, _ string, _ types.IndexType) (*model.IndexDefinition, error) {
	return nil, nil
}

func (m *mockCatalog) GetIndex(name string) (*model.IndexDefinition, error) {
	return m.indexes[name], nil
}

func (m *mockCatalog) ListIndexes(_ *model.TableDefinition) ([]*model.IndexDefinition, error) {
	return nil, nil
}

func parse(t *testing.T, sql string) ast.Statement {
	tokens, err := lexer.NewSqlLexer().Tokenize(sql)
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}
	stmt, err := parser.NewSqlParser().Parse(tokens)
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}
	return stmt
}

func TestAnalyzeCreateTable(t *testing.T) {
	catalog := newMockCatalog()
	analyzer := NewAnalyzer(catalog)

	stmt := parse(t, "CREATE TABLE users (id INT64, name VARCHAR)")
	result, err := analyzer.Analyze(stmt)
	if err != nil {
		t.Fatalf("analyze error: %v", err)
	}

	ct := result.(*CreateTableQueryTree)
	if ct.TableName().Text() != "users" {
		t.Errorf("expected 'users', got '%s'", ct.TableName().Text())
	}
	if len(ct.Columns()) != 2 {
		t.Errorf("expected 2 columns, got %d", len(ct.Columns()))
	}
}

func TestAnalyzeCreateTableDuplicateColumn(t *testing.T) {
	catalog := newMockCatalog()
	analyzer := NewAnalyzer(catalog)

	stmt := parse(t, "CREATE TABLE users (id INT64, id VARCHAR)")
	_, err := analyzer.Analyze(stmt)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAnalyzeCreateTableTableExists(t *testing.T) {
	catalog := newMockCatalog()
	catalog.addTable("users", []colDef{{"id", 1}})
	analyzer := NewAnalyzer(catalog)

	stmt := parse(t, "CREATE TABLE users (id INT64)")
	_, err := analyzer.Analyze(stmt)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAnalyzeInsert(t *testing.T) {
	catalog := newMockCatalog()
	catalog.addTable("users", []colDef{{"id", 1}, {"name", 2}})
	analyzer := NewAnalyzer(catalog)

	stmt := parse(t, "INSERT INTO users VALUES (1, 'John')")
	result, err := analyzer.Analyze(stmt)
	if err != nil {
		t.Fatalf("analyze error: %v", err)
	}

	ins := result.(*InsertQueryTree)
	if ins.Table().Name() != "users" {
		t.Errorf("expected 'users', got '%s'", ins.Table().Name())
	}
	if len(ins.Values()) != 2 {
		t.Errorf("expected 2 values, got %d", len(ins.Values()))
	}
}

func TestAnalyzeInsertTypeMismatch(t *testing.T) {
	catalog := newMockCatalog()
	catalog.addTable("users", []colDef{{"id", 1}})
	analyzer := NewAnalyzer(catalog)

	stmt := parse(t, "INSERT INTO users VALUES ('not a number')")
	_, err := analyzer.Analyze(stmt)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAnalyzeSelect(t *testing.T) {
	catalog := newMockCatalog()
	catalog.addTable("users", []colDef{{"id", 1}, {"name", 2}})
	analyzer := NewAnalyzer(catalog)

	stmt := parse(t, "SELECT * FROM users")
	result, err := analyzer.Analyze(stmt)
	if err != nil {
		t.Fatalf("analyze error: %v", err)
	}

	sel := result.(*SelectQueryTree)
	if sel.Table().Name() != "users" {
		t.Errorf("expected 'users', got '%s'", sel.Table().Name())
	}
	if len(sel.TargetColumns()) != 2 {
		t.Errorf("expected 2 columns, got %d", len(sel.TargetColumns()))
	}
}

func TestAnalyzeSelectColumns(t *testing.T) {
	catalog := newMockCatalog()
	catalog.addTable("users", []colDef{{"id", 1}, {"name", 2}})
	analyzer := NewAnalyzer(catalog)

	stmt := parse(t, "SELECT id FROM users")
	result, err := analyzer.Analyze(stmt)
	if err != nil {
		t.Fatalf("analyze error: %v", err)
	}

	sel := result.(*SelectQueryTree)
	if len(sel.TargetColumns()) != 1 {
		t.Errorf("expected 1 column, got %d", len(sel.TargetColumns()))
	}
}

func TestAnalyzeSelectWhere(t *testing.T) {
	catalog := newMockCatalog()
	catalog.addTable("users", []colDef{{"id", 1}})
	analyzer := NewAnalyzer(catalog)

	stmt := parse(t, "SELECT * FROM users WHERE id = 1")
	result, err := analyzer.Analyze(stmt)
	if err != nil {
		t.Fatalf("analyze error: %v", err)
	}

	sel := result.(*SelectQueryTree)
	if sel.Filter() == nil {
		t.Error("expected filter")
	}
	if sel.Filter().ExprType() != ExprTypeBool {
		t.Errorf("expected BOOL, got %v", sel.Filter().ExprType())
	}
}

func TestAnalyzeSelectWhereAnd(t *testing.T) {
	catalog := newMockCatalog()
	catalog.addTable("users", []colDef{{"id", 1}, {"name", 2}})
	analyzer := NewAnalyzer(catalog)

	stmt := parse(t, "SELECT * FROM users WHERE id = 1 AND name = 'John'")
	result, err := analyzer.Analyze(stmt)
	if err != nil {
		t.Fatalf("analyze error: %v", err)
	}

	sel := result.(*SelectQueryTree)
	bin := sel.Filter().(*ResolvedBinaryExpr)
	if bin.Op() != "AND" {
		t.Errorf("expected AND, got %s", bin.Op())
	}
}

func TestAnalyzeCreateIndex(t *testing.T) {
	catalog := newMockCatalog()
	catalog.addTable("users", []colDef{{"id", 1}})
	analyzer := NewAnalyzer(catalog)

	stmt := parse(t, "CREATE INDEX idx ON users (id) USING BTREE")
	result, err := analyzer.Analyze(stmt)
	if err != nil {
		t.Fatalf("analyze error: %v", err)
	}

	ci := result.(*CreateIndexQueryTree)
	if ci.IndexName().Text() != "idx" {
		t.Errorf("expected 'idx', got '%s'", ci.IndexName().Text())
	}
	if ci.IndexType() != types.IndexTypeBTree {
		t.Errorf("expected BTREE, got %v", ci.IndexType())
	}
}

func TestAnalyzeExplain(t *testing.T) {
	catalog := newMockCatalog()
	catalog.addTable("users", []colDef{{"id", 1}})
	analyzer := NewAnalyzer(catalog)

	stmt := parse(t, "EXPLAIN SELECT * FROM users")
	result, err := analyzer.Analyze(stmt)
	if err != nil {
		t.Fatalf("analyze error: %v", err)
	}

	ex := result.(*ExplainQueryTree)
	if _, ok := ex.Inner().(*SelectQueryTree); !ok {
		t.Error("expected SelectQueryTree inside")
	}
}

func TestAnalyzeTableNotFound(t *testing.T) {
	catalog := newMockCatalog()
	analyzer := NewAnalyzer(catalog)

	stmt := parse(t, "SELECT * FROM nonexistent")
	_, err := analyzer.Analyze(stmt)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAnalyzeColumnNotFound(t *testing.T) {
	catalog := newMockCatalog()
	catalog.addTable("users", []colDef{{"id", 1}})
	analyzer := NewAnalyzer(catalog)

	stmt := parse(t, "SELECT nonexistent FROM users")
	_, err := analyzer.Analyze(stmt)
	if err == nil {
		t.Fatal("expected error")
	}
}
