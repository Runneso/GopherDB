package engine

type ExecutionResult struct {
	columns  []string
	rows     [][]any
	affected int
	explain  string
}

func NewExecutionResult(columns []string, rows [][]any, affected int, explain string) *ExecutionResult {
	return &ExecutionResult{
		columns:  columns,
		rows:     rows,
		affected: affected,
		explain:  explain,
	}
}

func (result *ExecutionResult) Columns() []string {
	return result.columns
}

func (result *ExecutionResult) Rows() [][]any {
	return result.rows
}

func (result *ExecutionResult) Affected() int {
	return result.affected
}

func (result *ExecutionResult) Explain() string {
	return result.explain
}
