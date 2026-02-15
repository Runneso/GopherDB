package replacer

import (
	"GopherDB/internal/core/memory/model"
	"container/list"
	"sync"
)

type LRUReplacer struct {
	mu         sync.Mutex
	linkedList *list.List
	lru        map[model.PageKey]*list.Element
}

type lruEntry struct {
	key  model.PageKey
	slot *model.BufferSlot
}

func NewLRUReplacer() *LRUReplacer {
	return &LRUReplacer{
		linkedList: list.New(),
		lru:        make(map[model.PageKey]*list.Element, 16),
	}
}

func (lruReplacer *LRUReplacer) Push(slot *model.BufferSlot) {
	if slot == nil || slot.Pinned() {
		return
	}

	lruReplacer.mu.Lock()
	defer lruReplacer.mu.Unlock()

	key := slot.Key()
	if el, ok := lruReplacer.lru[key]; ok {
		el.Value.(*lruEntry).slot = slot
		lruReplacer.linkedList.MoveToBack(el)
		return
	}

	el := lruReplacer.linkedList.PushBack(&lruEntry{key: key, slot: slot})
	lruReplacer.lru[key] = el
}

func (lruReplacer *LRUReplacer) Delete(key model.PageKey) {
	lruReplacer.mu.Lock()
	defer lruReplacer.mu.Unlock()

	if el, ok := lruReplacer.lru[key]; ok {
		lruReplacer.linkedList.Remove(el)
		delete(lruReplacer.lru, key)
	}
}

func (lruReplacer *LRUReplacer) PickVictim() *model.BufferSlot {
	lruReplacer.mu.Lock()
	defer lruReplacer.mu.Unlock()

	el := lruReplacer.linkedList.Front()
	if el == nil {
		return nil
	}
	entry := el.Value.(*lruEntry)

	lruReplacer.linkedList.Remove(el)
	delete(lruReplacer.lru, entry.key)
	return entry.slot
}
