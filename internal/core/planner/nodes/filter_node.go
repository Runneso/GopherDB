package nodes

import "GopherDB/internal/core/sql/semantic"

type FilterNode struct {
	child     LogicalPlanNode
	predicate semantic.ResolvedExpr
}

func NewFilterNode(child LogicalPlanNode, predicate semantic.ResolvedExpr) *FilterNode {
	return &FilterNode{
		child:     child,
		predicate: predicate,
	}
}

func (node *FilterNode) Child() LogicalPlanNode {
	return node.child
}

func (node *FilterNode) Predicate() semantic.ResolvedExpr {
	return node.predicate
}

func (node *FilterNode) DisplayName() string {
	return "Filter"
}

func (node *FilterNode) Children() []LogicalPlanNode {
	return []LogicalPlanNode{node.child}
}
