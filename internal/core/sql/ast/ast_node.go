package ast

type AstNode interface {
	astNode()
}

type Statement interface {
	AstNode
	statement()
}

type Expr interface {
	AstNode
	expr()
}
