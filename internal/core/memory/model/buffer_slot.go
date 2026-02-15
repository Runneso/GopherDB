package model

import (
	"GopherDB/internal/core/memory/page"
)

type BufferSlot struct {
	key        PageKey
	page       page.Page
	dirty      bool
	pinned     bool
	usageCount int
}

func NewBufferSlot(key PageKey, p page.Page) *BufferSlot {
	return &BufferSlot{
		key:        key,
		page:       p,
		dirty:      false,
		pinned:     false,
		usageCount: 0,
	}
}

func (s *BufferSlot) Key() PageKey {
	return s.key
}

func (s *BufferSlot) Page() page.Page {
	return s.page
}

func (s *BufferSlot) SetPage(p page.Page) {
	s.page = p
}

func (s *BufferSlot) Dirty() bool {
	return s.dirty
}

func (s *BufferSlot) SetDirty(v bool) {
	s.dirty = v
}

func (s *BufferSlot) Pinned() bool {
	return s.pinned
}

func (s *BufferSlot) SetPinned(v bool) {
	s.pinned = v
}

func (s *BufferSlot) UsageCount() int {
	return s.usageCount
}

func (s *BufferSlot) IncrementUsage() {
	s.usageCount++
}
