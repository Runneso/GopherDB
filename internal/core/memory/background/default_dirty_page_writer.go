package background

import (
	"GopherDB/internal/core/memory/buffer"
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

type DefaultDirtyPageWriter struct {
	manager              buffer.BufferPoolManager
	intervalMs           int
	checkPointIntervalMs int
	batchSize            int
	backgroundFlag       atomic.Bool
	checkPointerFlag     atomic.Bool
}

func NewDefaultDirtyPageWriter(
	manager buffer.BufferPoolManager,
	intervalMs int,
	checkPointIntervalMs int,
	batchSize int,
) (*DefaultDirtyPageWriter, error) {
	if intervalMs <= 0 || checkPointIntervalMs <= 0 {
		return nil, fmt.Errorf("intervalMs or checkPointIntervalMs: negative or zero")
	}
	if batchSize <= 0 {
		return nil, fmt.Errorf("batchSize: negative or zero")
	}
	if manager == nil {
		return nil, fmt.Errorf("manager: nil")
	}

	return &DefaultDirtyPageWriter{
		manager:              manager,
		intervalMs:           intervalMs,
		checkPointIntervalMs: checkPointIntervalMs,
		batchSize:            batchSize,
	}, nil
}

func (defaultDirtyPageWriter *DefaultDirtyPageWriter) StartBackgroundWriter() (context.CancelFunc, error) {
	if !defaultDirtyPageWriter.backgroundFlag.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("backgroundFlag already started")
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func(ctx context.Context) {
		ticker := time.NewTicker(time.Duration(defaultDirtyPageWriter.intervalMs) * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				defaultDirtyPageWriter.flushBatch()
			case <-ctx.Done():
				return
			}
		}
	}(ctx)
	return cancel, nil
}

func (defaultDirtyPageWriter *DefaultDirtyPageWriter) StartCheckPointer() (context.CancelFunc, error) {
	if !defaultDirtyPageWriter.checkPointerFlag.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("checkPointerFlag already started")
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func(ctx context.Context) {
		ticker := time.NewTicker(time.Duration(defaultDirtyPageWriter.checkPointIntervalMs) * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = defaultDirtyPageWriter.manager.FlushAllPages()
			case <-ctx.Done():
				return
			}
		}
	}(ctx)
	return cancel, nil
}

func (defaultDirtyPageWriter *DefaultDirtyPageWriter) flushBatch() {
	dirty := defaultDirtyPageWriter.manager.DirtyPages()
	flushed := 0
	for _, p := range dirty {
		if flushed >= defaultDirtyPageWriter.batchSize {
			return
		}
		err := defaultDirtyPageWriter.manager.FlushPage(p.Key())
		if err != nil {
			continue
		}
		flushed++
	}
}
