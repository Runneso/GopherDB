package nodes

type ExplainNode struct {
	inner LogicalPlanNode
}

func NewExplainNode(inner LogicalPlanNode) *ExplainNode {
	return &ExplainNode{
		inner: inner,
	}
}

func (node *ExplainNode) Inner() LogicalPlanNode {
	return node.inner
}

func (node *ExplainNode) DisplayName() string {
	return "Explain"
}

func (node *ExplainNode) Children() []LogicalPlanNode {
	return []LogicalPlanNode{node.inner}
}
