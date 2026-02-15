package execution

import (
	"GopherDB/internal/core/execution/executors"
	"GopherDB/internal/core/optimizer/nodes"
)

type ExecutorFactory interface {
	CreateExecutor(plan nodes.PhysicalPlanNode) (executors.Executor, error)
}
