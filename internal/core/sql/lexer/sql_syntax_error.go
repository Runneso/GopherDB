package lexer

import "fmt"

type SqlSyntaxError struct {
	message string
	offset  int
	line    int
	column  int
}

func NewSqlSyntaxError(message string, offset, line, column int) *SqlSyntaxError {
	return &SqlSyntaxError{
		message: message,
		offset:  offset,
		line:    line,
		column:  column,
	}
}

func (error *SqlSyntaxError) Error() string {
	return fmt.Sprintf("%s at line %d, column %d", error.message, error.line, error.column)
}

func (error *SqlSyntaxError) Message() string {
	return error.message
}

func (error *SqlSyntaxError) Offset() int {
	return error.offset
}

func (error *SqlSyntaxError) Line() int {
	return error.line
}

func (error *SqlSyntaxError) Column() int {
	return error.column
}
