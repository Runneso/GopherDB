package execution

import (
	"errors"
	"reflect"

	"GopherDB/internal/core/catalog/manager"
	"GopherDB/internal/core/execution/executors"
	"GopherDB/internal/core/index"
	"GopherDB/internal/core/memory/buffer"
	"GopherDB/internal/core/optimizer/nodes"
	"GopherDB/internal/core/storage"
)

var ErrUnsupportedNode = errors.New("unsupported physical plan node")

type executorCreator func(*DefaultExecutorFactory, nodes.PhysicalPlanNode) (executors.Executor, error)

type DefaultExecutorFactory struct {
	root         string
	bufferPool   buffer.BufferPoolManager
	catalog      manager.CatalogManager
	indexManager *index.IndexManager
	creators     map[reflect.Type]executorCreator
}

func NewDefaultExecutorFactory(root string, bufferPool buffer.BufferPoolManager, catalog manager.CatalogManager, indexManager *index.IndexManager) *DefaultExecutorFactory {
	factory := &DefaultExecutorFactory{
		root:         root,
		bufferPool:   bufferPool,
		catalog:      catalog,
		indexManager: indexManager,
	}
	factory.creators = map[reflect.Type]executorCreator{
		reflect.TypeOf((*nodes.ExplainNode)(nil)):        createExplain,
		reflect.TypeOf((*nodes.CreateTableNode)(nil)):    createCreateTable,
		reflect.TypeOf((*nodes.CreateIndexNode)(nil)):    createCreateIndex,
		reflect.TypeOf((*nodes.InsertNode)(nil)):         createInsert,
		reflect.TypeOf((*nodes.ProjectNode)(nil)):        createProject,
		reflect.TypeOf((*nodes.FilterNode)(nil)):         createFilter,
		reflect.TypeOf((*nodes.SeqScanNode)(nil)):        createSeqScan,
		reflect.TypeOf((*nodes.HashIndexScanNode)(nil)):  createHashIndexScan,
		reflect.TypeOf((*nodes.BTreeIndexScanNode)(nil)): createBTreeIndexScan,
	}
	return factory
}

func (factory *DefaultExecutorFactory) CreateExecutor(plan nodes.PhysicalPlanNode) (executors.Executor, error) {
	creator, ok := factory.creators[reflect.TypeOf(plan)]
	if !ok {
		return nil, ErrUnsupportedNode
	}
	return creator(factory, plan)
}

func createExplain(f *DefaultExecutorFactory, plan nodes.PhysicalPlanNode) (executors.Executor, error) {
	return f.CreateExecutor(plan.(*nodes.ExplainNode).Inner())
}

func createCreateTable(f *DefaultExecutorFactory, plan nodes.PhysicalPlanNode) (executors.Executor, error) {
	return executors.NewCreateTableExecutor(f.catalog, plan.(*nodes.CreateTableNode).Query()), nil
}

func createCreateIndex(f *DefaultExecutorFactory, plan nodes.PhysicalPlanNode) (executors.Executor, error) {
	return executors.NewCreateIndexExecutor(f.root, f.bufferPool, f.catalog, f.indexManager, plan.(*nodes.CreateIndexNode).Query()), nil
}

func createInsert(f *DefaultExecutorFactory, plan nodes.PhysicalPlanNode) (executors.Executor, error) {
	query := plan.(*nodes.InsertNode).Query()
	tableHeap, err := storage.NewTableHeap(f.root, f.bufferPool, f.catalog, query.Table())
	if err != nil {
		return nil, err
	}
	return executors.NewInsertExecutor(f.catalog, f.indexManager, tableHeap, query), nil
}

func createProject(f *DefaultExecutorFactory, plan nodes.PhysicalPlanNode) (executors.Executor, error) {
	p := plan.(*nodes.ProjectNode)
	child, err := f.CreateExecutor(p.Child())
	if err != nil {
		return nil, err
	}
	return executors.NewProjectExecutor(child, p.Columns()), nil
}

func createFilter(f *DefaultExecutorFactory, plan nodes.PhysicalPlanNode) (executors.Executor, error) {
	node := plan.(*nodes.FilterNode)
	child, err := f.CreateExecutor(node.Child())
	if err != nil {
		return nil, err
	}
	return executors.NewFilterExecutor(child, node.Predicate()), nil
}

func createSeqScan(f *DefaultExecutorFactory, plan nodes.PhysicalPlanNode) (executors.Executor, error) {
	scan := plan.(*nodes.SeqScanNode)
	tableHeap, err := storage.NewTableHeap(f.root, f.bufferPool, f.catalog, scan.Table())
	if err != nil {
		return nil, err
	}
	return executors.NewSeqScanExecutor(tableHeap), nil
}

func createHashIndexScan(f *DefaultExecutorFactory, plan nodes.PhysicalPlanNode) (executors.Executor, error) {
	scan := plan.(*nodes.HashIndexScanNode)
	tableHeap, err := storage.NewTableHeap(f.root, f.bufferPool, f.catalog, scan.Table())
	if err != nil {
		return nil, err
	}
	idx, err := f.indexManager.GetOrCreate(scan.Index())
	if err != nil {
		return nil, err
	}
	return executors.NewHashIndexScanExecutor(idx, scan.Value(), tableHeap), nil
}

func createBTreeIndexScan(f *DefaultExecutorFactory, plan nodes.PhysicalPlanNode) (executors.Executor, error) {
	scan := plan.(*nodes.BTreeIndexScanNode)
	tableHeap, err := storage.NewTableHeap(f.root, f.bufferPool, f.catalog, scan.Table())
	if err != nil {
		return nil, err
	}
	idx, err := f.indexManager.GetOrCreate(scan.Index())
	if err != nil {
		return nil, err
	}
	return executors.NewBTreeIndexScanExecutor(idx, scan.From(), scan.FromInclusive(), scan.To(), scan.ToInclusive(), tableHeap), nil
}
