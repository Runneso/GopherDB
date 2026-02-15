package semantic

import (
	"fmt"
	"reflect"
	"strings"

	"GopherDB/internal/core/catalog/manager"
	"GopherDB/internal/core/catalog/model"
	"GopherDB/internal/core/sql/ast"
)

type stmtAnalyzer func(ast.Statement) (QueryTree, error)
type exprResolver func(ast.Expr, *model.TableDefinition) (ResolvedExpr, error)

type Analyzer struct {
	catalog       manager.CatalogManager
	stmtAnalyzers map[reflect.Type]stmtAnalyzer
	exprResolvers map[reflect.Type]exprResolver
	comparisonOps map[string]bool
	logicalOps    map[string]bool
	typeNames     map[string]string
	exprTypes     map[string]ExprType
}

func NewAnalyzer(catalog manager.CatalogManager) *Analyzer {
	analyzer := &Analyzer{
		catalog: catalog,
		comparisonOps: map[string]bool{
			"=": true, "<>": true, "<": true, "<=": true, ">": true, ">=": true,
		},
		logicalOps: map[string]bool{
			"AND": true, "OR": true,
		},
		typeNames: map[string]string{
			"INT": "INT64", "INTEGER": "INT64",
		},
		exprTypes: map[string]ExprType{
			"INT64": ExprTypeInt64, "VARCHAR": ExprTypeVarchar,
		},
	}
	analyzer.stmtAnalyzers = map[reflect.Type]stmtAnalyzer{
		reflect.TypeOf((*ast.CreateTableStmt)(nil)): analyzer.analyzeCreateTable,
		reflect.TypeOf((*ast.InsertStmt)(nil)):      analyzer.analyzeInsert,
		reflect.TypeOf((*ast.SelectStmt)(nil)):      analyzer.analyzeSelect,
		reflect.TypeOf((*ast.CreateIndexStmt)(nil)): analyzer.analyzeCreateIndex,
	}
	analyzer.exprResolvers = map[reflect.Type]exprResolver{
		reflect.TypeOf((*ast.ColumnRefExpr)(nil)):     analyzer.resolveColumnRef,
		reflect.TypeOf((*ast.LiteralInt64Expr)(nil)):  analyzer.resolveLiteralInt64,
		reflect.TypeOf((*ast.LiteralStringExpr)(nil)): analyzer.resolveLiteralString,
		reflect.TypeOf((*ast.BinaryExpr)(nil)):        analyzer.resolveBinary,
	}
	return analyzer
}

func (analyzer *Analyzer) Analyze(stmt ast.Statement) (QueryTree, error) {
	if ex, ok := stmt.(*ast.ExplainStmt); ok {
		inner, err := analyzer.Analyze(ex.Inner())
		if err != nil {
			return nil, err
		}
		return NewExplainQueryTree(inner), nil
	}
	if analyzer, ok := analyzer.stmtAnalyzers[reflect.TypeOf(stmt)]; ok {
		return analyzer(stmt)
	}
	return nil, analyzer.errorf("Unsupported statement type")
}

func (analyzer *Analyzer) analyzeCreateTable(stmt ast.Statement) (QueryTree, error) {
	s := stmt.(*ast.CreateTableStmt)
	tableName := s.TableName()
	table, _ := analyzer.catalog.GetTable(tableName.Text())
	if table != nil {
		return nil, analyzer.errorAt("Table already exists: "+tableName.Text(), tableName)
	}

	seen := make(map[string]bool)
	var cols []*ResolvedCreateColumn

	for _, cd := range s.Columns() {
		colName := cd.Name().Text()
		if seen[colName] {
			return nil, analyzer.errorAt("Duplicate column: "+colName, cd.Name())
		}
		seen[colName] = true

		typeName := analyzer.normalizeTypeName(cd.TypeName())
		typeDef, err := analyzer.catalog.GetTypeByName(typeName)
		if err != nil || typeDef == nil {
			return nil, analyzer.errorAt("Unknown type: "+cd.TypeName(), cd.Name())
		}

		cols = append(cols, NewResolvedCreateColumn(cd.Name(), typeDef))
	}

	return NewCreateTableQueryTree(tableName, cols), nil
}

func (analyzer *Analyzer) analyzeInsert(stmt ast.Statement) (QueryTree, error) {
	s := stmt.(*ast.InsertStmt)
	tableName := s.TableName()
	table, err := analyzer.catalog.GetTable(tableName.Text())
	if err != nil || table == nil {
		return nil, analyzer.errorAt("Table not found: "+tableName.Text(), tableName)
	}

	columns, err := analyzer.catalog.GetColumns(table)
	if err != nil {
		return nil, analyzer.errorf("Failed to get columns: %v", err)
	}

	if len(s.Values()) != len(columns) {
		return nil, analyzer.errorAt(fmt.Sprintf("Values count mismatch: expected %d got %d", len(columns), len(s.Values())), tableName)
	}

	var values []any
	for i, col := range columns {
		typeDef, err := analyzer.catalog.GetTypeByOid(col.TypeOid())
		if err != nil || typeDef == nil {
			return nil, analyzer.errorf("Unknown type oid: %d", col.TypeOid())
		}

		expr := s.Values()[i]
		val, valType, err := analyzer.resolveLiteral(expr)
		if err != nil {
			return nil, err
		}

		colType := analyzer.toExprType(typeDef)
		if valType != colType {
			return nil, analyzer.errorf("Type mismatch for column %s: expected %v got %v", col.Name(), colType, valType)
		}

		values = append(values, val)
	}

	return NewInsertQueryTree(table, columns, values), nil
}

func (analyzer *Analyzer) analyzeSelect(stmt ast.Statement) (QueryTree, error) {
	s := stmt.(*ast.SelectStmt)
	tableName := s.TableName()
	table, err := analyzer.catalog.GetTable(tableName.Text())
	if err != nil || table == nil {
		return nil, analyzer.errorAt("Table not found: "+tableName.Text(), tableName)
	}

	var outCols []*model.ColumnDefinition
	if s.SelectAll() {
		outCols, err = analyzer.catalog.GetColumns(table)
		if err != nil {
			return nil, analyzer.errorf("Failed to get columns: %v", err)
		}
	} else {
		for _, colName := range s.Columns() {
			col, err := analyzer.catalog.GetColumn(table, colName.Text())
			if err != nil || col == nil {
				return nil, analyzer.errorAt("Column not found: "+tableName.Text()+"."+colName.Text(), colName)
			}
			outCols = append(outCols, col)
		}
	}

	var filter ResolvedExpr
	if s.Where() != nil {
		filter, err = analyzer.resolveExpr(s.Where(), table)
		if err != nil {
			return nil, err
		}
		if filter.ExprType() != ExprTypeBool {
			return nil, analyzer.errorf("WHERE clause must be boolean")
		}
	}

	return NewSelectQueryTree(table, outCols, filter), nil
}

func (analyzer *Analyzer) analyzeCreateIndex(stmt ast.Statement) (QueryTree, error) {
	s := stmt.(*ast.CreateIndexStmt)
	indexName := s.IndexName()
	existing, _ := analyzer.catalog.GetIndex(indexName.Text())
	if existing != nil {
		return nil, analyzer.errorAt("Index already exists: "+indexName.Text(), indexName)
	}

	tableName := s.TableName()
	table, err := analyzer.catalog.GetTable(tableName.Text())
	if err != nil || table == nil {
		return nil, analyzer.errorAt("Table not found: "+tableName.Text(), tableName)
	}

	columnName := s.ColumnName()
	col, err := analyzer.catalog.GetColumn(table, columnName.Text())
	if err != nil || col == nil {
		return nil, analyzer.errorAt("Column not found: "+tableName.Text()+"."+columnName.Text(), columnName)
	}

	return NewCreateIndexQueryTree(indexName, table, col, s.IndexType()), nil
}

func (analyzer *Analyzer) resolveExpr(expr ast.Expr, table *model.TableDefinition) (ResolvedExpr, error) {
	if resolver, ok := analyzer.exprResolvers[reflect.TypeOf(expr)]; ok {
		return resolver(expr, table)
	}
	return nil, analyzer.errorf("Unsupported expression type")
}

func (analyzer *Analyzer) resolveColumnRef(expr ast.Expr, table *model.TableDefinition) (ResolvedExpr, error) {
	e := expr.(*ast.ColumnRefExpr)
	name := e.Name()
	col, err := analyzer.catalog.GetColumn(table, name.Text())
	if err != nil || col == nil {
		return nil, analyzer.errorAt("Column not found: "+table.Name()+"."+name.Text(), name)
	}
	typeDef, err := analyzer.catalog.GetTypeByOid(col.TypeOid())
	if err != nil || typeDef == nil {
		return nil, analyzer.errorf("Unknown type oid: %d", col.TypeOid())
	}
	return NewResolvedColumnRef(col, analyzer.toExprType(typeDef)), nil
}

func (analyzer *Analyzer) resolveLiteralInt64(expr ast.Expr, _ *model.TableDefinition) (ResolvedExpr, error) {
	e := expr.(*ast.LiteralInt64Expr)
	return NewResolvedConst(e.Value(), ExprTypeInt64), nil
}

func (analyzer *Analyzer) resolveLiteralString(expr ast.Expr, _ *model.TableDefinition) (ResolvedExpr, error) {
	e := expr.(*ast.LiteralStringExpr)
	return NewResolvedConst(e.Value(), ExprTypeVarchar), nil
}

func (analyzer *Analyzer) resolveBinary(expr ast.Expr, table *model.TableDefinition) (ResolvedExpr, error) {
	e := expr.(*ast.BinaryExpr)
	left, err := analyzer.resolveExpr(e.Left(), table)
	if err != nil {
		return nil, err
	}
	right, err := analyzer.resolveExpr(e.Right(), table)
	if err != nil {
		return nil, err
	}
	op := e.Op()

	if analyzer.logicalOps[op] {
		if left.ExprType() != ExprTypeBool {
			return nil, analyzer.errorf("Left operand of %s must be BOOL", op)
		}
		if right.ExprType() != ExprTypeBool {
			return nil, analyzer.errorf("Right operand of %s must be BOOL", op)
		}
		return NewResolvedBinaryExpr(op, left, right, ExprTypeBool), nil
	}

	if analyzer.comparisonOps[op] {
		if left.ExprType() != right.ExprType() {
			return nil, analyzer.errorf("Type mismatch in comparison: %v vs %v", left.ExprType(), right.ExprType())
		}
		if left.ExprType() == ExprTypeBool {
			return nil, analyzer.errorf("Cannot compare BOOL values")
		}
		return NewResolvedBinaryExpr(op, left, right, ExprTypeBool), nil
	}

	return nil, analyzer.errorf("Unsupported operator: %s", op)
}

func (analyzer *Analyzer) resolveLiteral(expr ast.Expr) (any, ExprType, error) {
	if expr, ok := expr.(*ast.LiteralInt64Expr); ok {
		return expr.Value(), ExprTypeInt64, nil
	}
	if expr, ok := expr.(*ast.LiteralStringExpr); ok {
		return expr.Value(), ExprTypeVarchar, nil
	}
	return nil, 0, analyzer.errorf("Only literal values are supported in INSERT")
}

func (analyzer *Analyzer) toExprType(typeDef *model.TypeDefinition) ExprType {
	if typ, ok := analyzer.exprTypes[typeDef.Name()]; ok {
		return typ
	}
	return ExprTypeInt64
}

func (analyzer *Analyzer) normalizeTypeName(typeName string) string {
	upper := strings.ToUpper(typeName)
	if normalized, ok := analyzer.typeNames[upper]; ok {
		return normalized
	}
	return upper
}

func (analyzer *Analyzer) errorf(format string, args ...any) error {
	return NewSemanticError(fmt.Sprintf(format, args...), nil, nil, nil)
}

func (analyzer *Analyzer) errorAt(message string, ident *ast.SqlIdent) error {
	return NewSemanticError(message, new(ident.Offset()), new(ident.Line()), new(ident.Column()))
}
