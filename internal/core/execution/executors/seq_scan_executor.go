package executors

import (
	"GopherDB/internal/core/index"
	"GopherDB/internal/core/storage"
)

type SeqScanExecutor struct {
	tableHeap *storage.TableHeap
	tids      []index.TID
	cursor    int
	isOpen    bool
}

func NewSeqScanExecutor(tableHeap *storage.TableHeap) *SeqScanExecutor {
	return &SeqScanExecutor{tableHeap: tableHeap}
}

func (executor *SeqScanExecutor) Open() error {
	tids, err := executor.tableHeap.ScanTids()
	if err != nil {
		return err
	}
	executor.tids = tids
	executor.cursor = 0
	executor.isOpen = true
	return nil
}

func (executor *SeqScanExecutor) Next() ([]any, error) {
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

func (executor *SeqScanExecutor) Close() error {
	executor.isOpen = false
	executor.tids = nil
	return nil
}
