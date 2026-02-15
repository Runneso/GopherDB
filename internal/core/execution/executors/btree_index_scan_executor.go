package executors

import (
	"GopherDB/internal/core/index"
	"GopherDB/internal/core/storage"
)

type BTreeIndexScanExecutor struct {
	idx           index.Index
	from          any
	fromInclusive bool
	to            any
	toInclusive   bool
	tableHeap     *storage.TableHeap
	tids          []index.TID
	cursor        int
	isOpen        bool
}

func NewBTreeIndexScanExecutor(idx index.Index, from any, fromInclusive bool, to any, toInclusive bool, tableHeap *storage.TableHeap) *BTreeIndexScanExecutor {
	return &BTreeIndexScanExecutor{
		idx:           idx,
		from:          from,
		fromInclusive: fromInclusive,
		to:            to,
		toInclusive:   toInclusive,
		tableHeap:     tableHeap,
	}
}

func (executor *BTreeIndexScanExecutor) Open() error {
	tids, err := executor.idx.RangeSearch(executor.from, executor.fromInclusive, executor.to, executor.toInclusive)
	if err != nil {
		return err
	}
	executor.tids = tids
	executor.cursor = 0
	executor.isOpen = true
	return nil
}

func (executor *BTreeIndexScanExecutor) Next() ([]any, error) {
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

func (executor *BTreeIndexScanExecutor) Close() error {
	executor.isOpen = false
	executor.tids = nil
	return nil
}
