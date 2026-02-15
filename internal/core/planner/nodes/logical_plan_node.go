package nodes

type LogicalPlanNode interface {
	DisplayName() string
	Children() []LogicalPlanNode
}
