package parser

import (
	"fmt"
	"strconv"
	"strings"

	"GopherDB/internal/core/sql/ast"
	"GopherDB/internal/core/sql/lexer"
	"GopherDB/internal/core/types"
)

type stmtParser func() (ast.Statement, error)

type SqlParser struct {
	tokens        []*lexer.Token
	pos           int
	stmtParsers   map[lexer.TokenType]stmtParser
	createParsers map[lexer.TokenType]stmtParser
	comparisonOps map[lexer.TokenType]string
	indexTypes    map[lexer.TokenType]types.IndexType
	typeTokens    map[lexer.TokenType]bool
	literalTokens map[lexer.TokenType]bool
}

func NewSqlParser() *SqlParser {
	parser := &SqlParser{
		comparisonOps: map[lexer.TokenType]string{
			lexer.EQ: "=", lexer.NE: "<>", lexer.LT: "<",
			lexer.LE: "<=", lexer.GT: ">", lexer.GE: ">=",
		},
		indexTypes: map[lexer.TokenType]types.IndexType{
			lexer.HASH: types.IndexTypeHash, lexer.BTREE: types.IndexTypeBTree,
		},
		typeTokens: map[lexer.TokenType]bool{
			lexer.INT64: true, lexer.VARCHAR: true, lexer.IDENT: true,
		},
		literalTokens: map[lexer.TokenType]bool{
			lexer.NUMBER: true, lexer.STRING: true,
		},
	}
	parser.stmtParsers = map[lexer.TokenType]stmtParser{
		lexer.EXPLAIN: parser.parseExplain,
		lexer.CREATE:  parser.parseCreate,
		lexer.INSERT:  parser.parseInsert,
		lexer.SELECT:  parser.parseSelect,
	}
	parser.createParsers = map[lexer.TokenType]stmtParser{
		lexer.TABLE: parser.parseCreateTable,
		lexer.INDEX: parser.parseCreateIndex,
	}
	return parser
}

func (parser *SqlParser) Parse(tokens []*lexer.Token) (ast.Statement, error) {
	parser.tokens, parser.pos = tokens, 0
	stmt, err := parser.parseStatement()
	if err != nil {
		return nil, err
	}
	return stmt, parser.expect(lexer.EOF)
}

func (parser *SqlParser) parseStatement() (ast.Statement, error) {
	if parser, ok := parser.stmtParsers[parser.current().Type()]; ok {
		return parser()
	}
	return nil, parser.errorf("Unexpected token: %s", parser.current().Type())
}

func (parser *SqlParser) parseExplain() (ast.Statement, error) {
	parser.advance()
	inner, err := parser.parseStatement()
	if err != nil {
		return nil, err
	}
	parser.match(lexer.SEMICOLON)
	return ast.NewExplainStmt(inner), nil
}

func (parser *SqlParser) parseCreate() (ast.Statement, error) {
	parser.advance()
	if parser, ok := parser.createParsers[parser.current().Type()]; ok {
		return parser()
	}
	return nil, parser.errorf("Expected TABLE or INDEX, got %s", parser.current().Type())
}

func (parser *SqlParser) parseCreateTable() (ast.Statement, error) {
	parser.advance()
	tableName, err := parser.expectIdent()
	if err != nil {
		return nil, err
	}
	columns, err := parseParenList(parser, parser.parseColumnDef)
	if err != nil {
		return nil, err
	}
	parser.match(lexer.SEMICOLON)
	return ast.NewCreateTableStmt(tableName, columns), nil
}

func (parser *SqlParser) parseColumnDef() (*ast.ColumnDef, error) {
	name, err := parser.expectIdent()
	if err != nil {
		return nil, err
	}
	if !parser.typeTokens[parser.current().Type()] {
		return nil, parser.errorf("Expected type name, got %s", parser.current().Type())
	}
	typeName := strings.ToUpper(parser.current().Text())
	parser.advance()
	return ast.NewColumnDef(name, typeName), nil
}

func (parser *SqlParser) parseCreateIndex() (ast.Statement, error) {
	parser.advance()
	indexName, err := parser.expectIdent()
	if err != nil {
		return nil, err
	}
	if err := parser.expect(lexer.ON); err != nil {
		return nil, err
	}
	tableName, err := parser.expectIdent()
	if err != nil {
		return nil, err
	}
	if err := parser.expect(lexer.LPAREN); err != nil {
		return nil, err
	}
	columnName, err := parser.expectIdent()
	if err != nil {
		return nil, err
	}
	if err := parser.expect(lexer.RPAREN); err != nil {
		return nil, err
	}
	if err := parser.expect(lexer.USING); err != nil {
		return nil, err
	}
	idxType, ok := parser.indexTypes[parser.current().Type()]
	if !ok {
		return nil, parser.errorf("Expected HASH or BTREE, got %s", parser.current().Type())
	}
	parser.advance()
	parser.match(lexer.SEMICOLON)
	return ast.NewCreateIndexStmt(indexName, tableName, columnName, idxType), nil
}

func (parser *SqlParser) parseInsert() (ast.Statement, error) {
	parser.advance()
	if err := parser.expect(lexer.INTO); err != nil {
		return nil, err
	}
	tableName, err := parser.expectIdent()
	if err != nil {
		return nil, err
	}
	if err := parser.expect(lexer.VALUES); err != nil {
		return nil, err
	}
	values, err := parseParenList(parser, parser.parseLiteral)
	if err != nil {
		return nil, err
	}
	parser.match(lexer.SEMICOLON)
	return ast.NewInsertStmt(tableName, values), nil
}

func (parser *SqlParser) parseSelect() (ast.Statement, error) {
	parser.advance()
	selectAll, columns := false, []*ast.SqlIdent(nil)
	if parser.match(lexer.ASTERISK) {
		selectAll = true
	} else {
		cols, err := parseCommaList(parser, parser.expectIdent)
		if err != nil {
			return nil, err
		}
		columns = cols
	}
	if err := parser.expect(lexer.FROM); err != nil {
		return nil, err
	}
	tableName, err := parser.expectIdent()
	if err != nil {
		return nil, err
	}
	var where ast.Expr
	if parser.match(lexer.WHERE) {
		where, err = parser.parseExpr()
		if err != nil {
			return nil, err
		}
	}
	parser.match(lexer.SEMICOLON)
	return ast.NewSelectStmt(selectAll, columns, tableName, where), nil
}

func (parser *SqlParser) parseExpr() (ast.Expr, error) {
	return parser.parseBinaryLeft(func() (ast.Expr, error) {
		return parser.parseBinaryLeft(parser.parseComparison, lexer.AND, "AND")
	}, lexer.OR, "OR")
}

func (parser *SqlParser) parseBinaryLeft(next func() (ast.Expr, error), op lexer.TokenType, opStr string) (ast.Expr, error) {
	left, err := next()
	if err != nil {
		return nil, err
	}
	for parser.match(op) {
		right, err := next()
		if err != nil {
			return nil, err
		}
		left = ast.NewBinaryExpr(opStr, left, right)
	}
	return left, nil
}

func (parser *SqlParser) parseComparison() (ast.Expr, error) {
	left, err := parser.parsePrimary()
	if err != nil {
		return nil, err
	}
	if op, ok := parser.comparisonOps[parser.current().Type()]; ok {
		parser.advance()
		right, err := parser.parsePrimary()
		if err != nil {
			return nil, err
		}
		return ast.NewBinaryExpr(op, left, right), nil
	}
	return left, nil
}

func (parser *SqlParser) parsePrimary() (ast.Expr, error) {
	tokenType := parser.current().Type()
	if tokenType == lexer.IDENT {
		ident, _ := parser.expectIdent()
		return ast.NewColumnRefExpr(ident), nil
	}
	if parser.literalTokens[tokenType] {
		return parser.parseLiteral()
	}
	if tokenType == lexer.LPAREN {
		parser.advance()
		expr, err := parser.parseExpr()
		if err != nil {
			return nil, err
		}
		return expr, parser.expect(lexer.RPAREN)
	}
	return nil, parser.errorf("Unexpected token: %s", tokenType)
}

func (parser *SqlParser) parseLiteral() (ast.Expr, error) {
	token := parser.current()
	parser.advance()
	if token.Type() == lexer.NUMBER {
		val, err := strconv.ParseInt(token.Text(), 10, 64)
		if err != nil {
			return nil, parser.errorf("Invalid number: %s", token.Text())
		}
		return ast.NewLiteralInt64Expr(val), nil
	}
	return ast.NewLiteralStringExpr(token.Text()), nil
}

func (parser *SqlParser) current() *lexer.Token {
	return parser.tokens[min(parser.pos, len(parser.tokens)-1)]
}

func (parser *SqlParser) advance() {
	parser.pos++
}

func (parser *SqlParser) currentType() lexer.TokenType {
	return parser.current().Type()
}

func (parser *SqlParser) match(tokenType lexer.TokenType) bool {
	if parser.currentType() == tokenType {
		parser.advance()
		return true
	}
	return false
}

func (parser *SqlParser) expect(tokenType lexer.TokenType) error {
	if parser.currentType() != tokenType {
		return parser.errorf("Expected %s, got %s", tokenType, parser.currentType())
	}
	parser.advance()
	return nil
}

func (parser *SqlParser) expectIdent() (*ast.SqlIdent, error) {
	token := parser.current()
	if err := parser.expect(lexer.IDENT); err != nil {
		return nil, err
	}
	return ast.NewSqlIdent(token.Text(), token.StartOffset(), token.Line(), token.Column()), nil
}

func (parser *SqlParser) errorf(format string, args ...any) error {
	t := parser.current()
	return lexer.NewSqlSyntaxError(fmt.Sprintf(format, args...), t.StartOffset(), t.Line(), t.Column())
}

func parseParenList[T any](parser *SqlParser, parse func() (T, error)) ([]T, error) {
	if err := parser.expect(lexer.LPAREN); err != nil {
		return nil, err
	}
	items, err := parseCommaList(parser, parse)
	if err != nil {
		return nil, err
	}
	return items, parser.expect(lexer.RPAREN)
}

func parseCommaList[T any](parser *SqlParser, parse func() (T, error)) ([]T, error) {
	item, err := parse()
	if err != nil {
		return nil, err
	}
	items := []T{item}
	for parser.match(lexer.COMMA) {
		item, err := parse()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}
