package nodes

type PhysicalPlanNode interface {
	DisplayName() string
	Children() []PhysicalPlanNode
}
