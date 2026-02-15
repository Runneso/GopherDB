package nodes

import "GopherDB/internal/core/sql/semantic"

type CreateTableNode struct {
	query *semantic.CreateTableQueryTree
}

func NewCreateTableNode(query *semantic.CreateTableQueryTree) *CreateTableNode {
	return &CreateTableNode{
		query: query,
	}
}

func (node *CreateTableNode) Query() *semantic.CreateTableQueryTree {
	return node.query
}

func (node *CreateTableNode) DisplayName() string {
	return "CreateTable(" + node.query.TableName().Text() + ")"
}

func (node *CreateTableNode) Children() []LogicalPlanNode {
	return nil
}
