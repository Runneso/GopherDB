package optimizer

import (
	"GopherDB/internal/core/optimizer/nodes"
	plannernodes "GopherDB/internal/core/planner/nodes"
)

type Optimizer interface {
	Optimize(logicalPlan plannernodes.LogicalPlanNode) nodes.PhysicalPlanNode
}
