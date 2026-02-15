package ast

type SqlIdent struct {
	text   string
	offset int
	line   int
	column int
}

func NewSqlIdent(text string, offset, line, column int) *SqlIdent {
	return &SqlIdent{
		text:   text,
		offset: offset,
		line:   line,
		column: column,
	}
}

func (ident *SqlIdent) Text() string {
	return ident.text
}

func (ident *SqlIdent) Offset() int {
	return ident.offset
}

func (ident *SqlIdent) Line() int {
	return ident.line
}

func (ident *SqlIdent) Column() int {
	return ident.column
}
