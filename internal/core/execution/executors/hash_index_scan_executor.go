package executors

import (
	"GopherDB/internal/core/index"
	"GopherDB/internal/core/storage"
)

type HashIndexScanExecutor struct {
	idx       index.Index
	searchKey any
	tableHeap *storage.TableHeap
	tids      []index.TID
	cursor    int
	isOpen    bool
}

func NewHashIndexScanExecutor(idx index.Index, searchKey any, tableHeap *storage.TableHeap) *HashIndexScanExecutor {
	return &HashIndexScanExecutor{idx: idx, searchKey: searchKey, tableHeap: tableHeap}
}

func (executor *HashIndexScanExecutor) Open() error {
	tids, err := executor.idx.Search(executor.searchKey)
	if err != nil {
		return err
	}
	executor.tids = tids
	executor.cursor = 0
	executor.isOpen = true
	return nil
}

func (executor *HashIndexScanExecutor) Next() ([]any, error) {
	if !executor.isOpen {
		return nil, ErrNotOpen
	}
	if executor.cursor >= len(executor.tids) {
		return nil, nil
	}
	tid := executor.tids[executor.cursor]
	executor.cursor++
	return executor.tableHeap.ReadRow(tid)
}

func (executor *HashIndexScanExecutor) Close() error {
	executor.isOpen = false
	executor.tids = nil
	return nil
}
