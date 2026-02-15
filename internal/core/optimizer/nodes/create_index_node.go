package nodes

import "GopherDB/internal/core/sql/semantic"

type CreateIndexNode struct {
	query *semantic.CreateIndexQueryTree
}

func NewCreateIndexNode(query *semantic.CreateIndexQueryTree) *CreateIndexNode {
	return &CreateIndexNode{
		query: query,
	}
}

func (node *CreateIndexNode) Query() *semantic.CreateIndexQueryTree {
	return node.query
}

func (node *CreateIndexNode) DisplayName() string {
	return "CreateIndex(" + node.query.IndexName().Text() + ")"
}

func (node *CreateIndexNode) Children() []PhysicalPlanNode {
	return nil
}
