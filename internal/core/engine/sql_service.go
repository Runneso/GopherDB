package engine

import (
	"GopherDB/internal/core/catalog/manager"
	"GopherDB/internal/core/execution"
	"GopherDB/internal/core/index"
	"GopherDB/internal/core/memory/buffer"
	"GopherDB/internal/core/optimizer"
	"GopherDB/internal/core/planner"
	"GopherDB/internal/core/sql/lexer"
	"GopherDB/internal/core/sql/parser"
	"GopherDB/internal/core/sql/semantic"
)

type SqlService struct {
	root         string
	bufferPool   buffer.BufferPoolManager
	catalog      manager.CatalogManager
	indexManager *index.IndexManager

	analyzer        *semantic.Analyzer
	planner         planner.Planner
	optimizer       optimizer.Optimizer
	executorFactory execution.ExecutorFactory
	engine          execution.QueryExecutionEngine
}

func NewSqlService(root string, bufferPool buffer.BufferPoolManager, catalog manager.CatalogManager, indexManager *index.IndexManager) *SqlService {
	return &SqlService{
		root:            root,
		bufferPool:      bufferPool,
		catalog:         catalog,
		indexManager:    indexManager,
		analyzer:        semantic.NewAnalyzer(catalog),
		planner:         planner.NewDefaultPlanner(),
		optimizer:       optimizer.NewDefaultOptimizer(catalog),
		executorFactory: execution.NewDefaultExecutorFactory(root, bufferPool, catalog, indexManager),
		engine:          execution.NewDefaultQueryExecutionEngine(),
	}
}

func (service *SqlService) Execute(sql string) (*ExecutionResult, error) {
	return service.ExecuteWithContext(nil, sql)
}

func (service *SqlService) ExecuteWithContext(ctx *SessionContext, sql string) (*ExecutionResult, error) {
	trace := ctx != nil && ctx.Trace()

	lex := lexer.NewSqlLexer()
	prs := parser.NewSqlParser()

	tokens, err := lex.Tokenize(sql)
	if err != nil {
		return nil, err
	}

	stmt, err := prs.Parse(tokens)
	if err != nil {
		return nil, err
	}

	queryTree, err := service.analyzer.Analyze(stmt)
	if err != nil {
		return nil, err
	}

	logicalPlan := service.planner.Plan(queryTree)
	physicalPlan := service.optimizer.Optimize(logicalPlan)

	var explainText string
	if trace || queryTree.Type() == semantic.QueryTypeExplain {
		explainText = FormatExplain(tokens, stmt, queryTree, logicalPlan, physicalPlan)
	}

	if queryTree.Type() == semantic.QueryTypeExplain {
		return NewExecutionResult(nil, nil, 0, explainText), nil
	}

	executor, err := service.executorFactory.CreateExecutor(physicalPlan)
	if err != nil {
		return nil, err
	}

	rows, err := service.engine.Execute(executor)
	if err != nil {
		return nil, err
	}

	if queryTree.Type() != semantic.QueryTypeSelect {
		service.bufferPool.FlushAllPages()
	}

	var columns []string
	if sel, ok := queryTree.(*semantic.SelectQueryTree); ok {
		for _, col := range sel.TargetColumns() {
			columns = append(columns, col.Name())
		}
	}

	affected := 0
	if queryTree.Type() == semantic.QueryTypeInsert {
		affected = 1
	}

	var explain string
	if trace {
		explain = explainText
	}

	return NewExecutionResult(columns, rows, affected, explain), nil
}
