package nodes

import "GopherDB/internal/core/sql/semantic"

type InsertNode struct {
	query *semantic.InsertQueryTree
}

func NewInsertNode(query *semantic.InsertQueryTree) *InsertNode {
	return &InsertNode{
		query: query,
	}
}

func (node *InsertNode) Query() *semantic.InsertQueryTree {
	return node.query
}

func (node *InsertNode) DisplayName() string {
	return "Insert(" + node.query.Table().Name() + ")"
}

func (node *InsertNode) Children() []LogicalPlanNode {
	return nil
}
