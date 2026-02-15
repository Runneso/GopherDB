package semantic

import "fmt"

type SemanticError struct {
	message string
	offset  *int
	line    *int
	column  *int
}

func NewSemanticError(message string, offset, line, column *int) *SemanticError {
	return &SemanticError{
		message: message,
		offset:  offset,
		line:    line,
		column:  column,
	}
}

func (error *SemanticError) Error() string {
	if error.line != nil && error.column != nil {
		return fmt.Sprintf("%s at line %d, column %d", error.message, *error.line, *error.column)
	}
	return error.message
}

func (error *SemanticError) Message() string {
	return error.message
}

func (error *SemanticError) Offset() *int {
	return error.offset
}

func (error *SemanticError) Line() *int {
	return error.line
}

func (error *SemanticError) Column() *int {
	return error.column
}
