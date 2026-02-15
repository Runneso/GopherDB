package nodes

type ExplainNode struct {
	inner PhysicalPlanNode
}

func NewExplainNode(inner PhysicalPlanNode) *ExplainNode {
	return &ExplainNode{
		inner: inner,
	}
}

func (node *ExplainNode) Inner() PhysicalPlanNode {
	return node.inner
}

func (node *ExplainNode) DisplayName() string {
	return "Explain"
}

func (node *ExplainNode) Children() []PhysicalPlanNode {
	return []PhysicalPlanNode{node.inner}
}
