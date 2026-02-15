package buffer

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"GopherDB/internal/core/memory/manager"
	"GopherDB/internal/core/memory/model"
	"GopherDB/internal/core/memory/page"
	"GopherDB/internal/core/memory/replacer"
)

var (
	ErrPoolSize         = errors.New("poolSize must be > 0")
	ErrPageAlreadyInBuf = errors.New("page already exists in buffer")
	ErrNoVictim         = errors.New("no victim available (all pages pinned)")
)

type HeapBufferPoolManager struct {
	mu sync.Mutex

	poolSize        int
	pageFileManager manager.PageFileManager
	replacer        replacer.Replacer
	root            string

	pageTable map[model.PageKey]*model.BufferSlot
}

func NewHeapBufferPoolManager(
	poolSize int,
	pageFileManager manager.PageFileManager,
	replacer replacer.Replacer,
	storageRoot string,
) (*HeapBufferPoolManager, error) {
	if poolSize <= 0 {
		return nil, ErrPoolSize
	}
	if pageFileManager == nil {
		return nil, fmt.Errorf("pageFileManager: nil")
	}
	if replacer == nil {
		return nil, fmt.Errorf("replacer: nil")
	}
	if storageRoot == "" {
		return nil, fmt.Errorf("storageRoot: empty")
	}

	return &HeapBufferPoolManager{
		poolSize:        poolSize,
		pageFileManager: pageFileManager,
		replacer:        replacer,
		root:            storageRoot,
		pageTable:       make(map[model.PageKey]*model.BufferSlot, poolSize),
	}, nil
}

func (heapBufferPoolManager *HeapBufferPoolManager) GetPage(key model.PageKey) (*model.BufferSlot, error) {
	heapBufferPoolManager.mu.Lock()
	defer heapBufferPoolManager.mu.Unlock()

	if slot, ok := heapBufferPoolManager.pageTable[key]; ok && slot != nil {
		slot.IncrementUsage()
		heapBufferPoolManager.touch(slot)
		return slot, nil
	}

	if err := heapBufferPoolManager.ensureSpaceLocked(); err != nil {
		return nil, err
	}

	pg, err := heapBufferPoolManager.pageFileManager.Read(key.PageID(), filepath.Join(heapBufferPoolManager.root, key.FileID()))
	if err != nil {
		return nil, err
	}

	slot := model.NewBufferSlot(key, pg)
	heapBufferPoolManager.pageTable[key] = slot
	heapBufferPoolManager.touch(slot)
	return slot, nil
}

func (heapBufferPoolManager *HeapBufferPoolManager) NewPage(key model.PageKey, pg page.Page) (*model.BufferSlot, error) {
	heapBufferPoolManager.mu.Lock()
	defer heapBufferPoolManager.mu.Unlock()

	if pg == nil {
		return nil, fmt.Errorf("page: nil")
	}
	if _, exists := heapBufferPoolManager.pageTable[key]; exists {
		return nil, fmt.Errorf("%w: %v", ErrPageAlreadyInBuf, key)
	}

	if err := heapBufferPoolManager.ensureSpaceLocked(); err != nil {
		return nil, err
	}

	slot := model.NewBufferSlot(key, pg)
	heapBufferPoolManager.pageTable[key] = slot
	heapBufferPoolManager.touch(slot)
	return slot, nil
}

func (heapBufferPoolManager *HeapBufferPoolManager) UpdatePage(key model.PageKey, pg page.Page) error {
	if pg == nil {
		return fmt.Errorf("page: nil")
	}

	heapBufferPoolManager.mu.Lock()
	defer heapBufferPoolManager.mu.Unlock()

	slot := heapBufferPoolManager.pageTable[key]
	if slot == nil {
		var err error
		slot, err = heapBufferPoolManager.getPageLocked(key)
		if err != nil {
			return err
		}
	}

	slot.SetPage(pg)
	slot.SetDirty(true)
	heapBufferPoolManager.touch(slot)
	return nil
}

func (heapBufferPoolManager *HeapBufferPoolManager) PinPage(key model.PageKey) error {
	heapBufferPoolManager.mu.Lock()
	defer heapBufferPoolManager.mu.Unlock()

	slot := heapBufferPoolManager.pageTable[key]
	if slot == nil {
		var err error
		slot, err = heapBufferPoolManager.getPageLocked(key)
		if err != nil {
			return err
		}
	}

	slot.SetPinned(true)
	heapBufferPoolManager.replacer.Delete(key)
	return nil
}

func (heapBufferPoolManager *HeapBufferPoolManager) UnpinPage(key model.PageKey) error {
	heapBufferPoolManager.mu.Lock()
	defer heapBufferPoolManager.mu.Unlock()

	slot := heapBufferPoolManager.pageTable[key]
	if slot == nil || !slot.Pinned() {
		return nil
	}
	slot.SetPinned(false)
	heapBufferPoolManager.replacer.Push(slot)
	return nil
}

func (heapBufferPoolManager *HeapBufferPoolManager) FlushPage(key model.PageKey) error {
	heapBufferPoolManager.mu.Lock()
	defer heapBufferPoolManager.mu.Unlock()

	slot := heapBufferPoolManager.pageTable[key]
	if slot == nil {
		return nil
	}
	return heapBufferPoolManager.flushSlotLocked(key, slot)
}

func (heapBufferPoolManager *HeapBufferPoolManager) FlushAllPages() error {
	heapBufferPoolManager.mu.Lock()
	defer heapBufferPoolManager.mu.Unlock()

	for k, slot := range heapBufferPoolManager.pageTable {
		if slot == nil {
			continue
		}
		if err := heapBufferPoolManager.flushSlotLocked(k, slot); err != nil {
			return err
		}
	}
	return nil
}

func (heapBufferPoolManager *HeapBufferPoolManager) DirtyPages() []*model.BufferSlot {
	heapBufferPoolManager.mu.Lock()
	defer heapBufferPoolManager.mu.Unlock()

	res := make([]*model.BufferSlot, 0)
	for _, slot := range heapBufferPoolManager.pageTable {
		if slot != nil && slot.Dirty() {
			res = append(res, slot)
		}
	}
	return res
}

func (heapBufferPoolManager *HeapBufferPoolManager) touch(slot *model.BufferSlot) {
	if slot != nil && !slot.Pinned() {
		heapBufferPoolManager.replacer.Push(slot)
	}
}

func (heapBufferPoolManager *HeapBufferPoolManager) getPageLocked(key model.PageKey) (*model.BufferSlot, error) {
	if slot, ok := heapBufferPoolManager.pageTable[key]; ok && slot != nil {
		slot.IncrementUsage()
		heapBufferPoolManager.touch(slot)
		return slot, nil
	}

	if err := heapBufferPoolManager.ensureSpaceLocked(); err != nil {
		return nil, err
	}

	pg, err := heapBufferPoolManager.pageFileManager.Read(key.PageID(), filepath.Join(heapBufferPoolManager.root, key.FileID()))
	if err != nil {
		return nil, err
	}

	slot := model.NewBufferSlot(key, pg)
	heapBufferPoolManager.pageTable[key] = slot
	heapBufferPoolManager.touch(slot)
	return slot, nil
}

func (heapBufferPoolManager *HeapBufferPoolManager) ensureSpaceLocked() error {
	if len(heapBufferPoolManager.pageTable) < heapBufferPoolManager.poolSize {
		return nil
	}

	victim := heapBufferPoolManager.replacer.PickVictim()
	if victim == nil {
		return ErrNoVictim
	}

	vKey := victim.Key()
	if victim.Dirty() {
		if err := heapBufferPoolManager.flushSlotLocked(vKey, victim); err != nil {
			return err
		}
	}

	delete(heapBufferPoolManager.pageTable, vKey)
	return nil
}

func (heapBufferPoolManager *HeapBufferPoolManager) flushSlotLocked(key model.PageKey, slot *model.BufferSlot) error {
	if slot.Dirty() {
		if err := heapBufferPoolManager.pageFileManager.Write(slot.Page(), filepath.Join(heapBufferPoolManager.root, key.FileID())); err != nil {
			return err
		}
		slot.SetDirty(false)
	}
	return nil
}
