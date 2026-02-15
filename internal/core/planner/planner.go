package planner

import (
	"GopherDB/internal/core/planner/nodes"
	"GopherDB/internal/core/sql/semantic"
)

type Planner interface {
	Plan(queryTree semantic.QueryTree) nodes.LogicalPlanNode
}
