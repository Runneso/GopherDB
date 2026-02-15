package replacer

import (
	"sync"

	"GopherDB/internal/core/memory/model"
)

type ClockReplacer struct {
	mu sync.Mutex

	ring []*model.BufferSlot
	head int

	inRing map[model.PageKey]struct{}
	ref    map[model.PageKey]struct{}
}

func NewClockReplacer() *ClockReplacer {
	return &ClockReplacer{
		ring:   make([]*model.BufferSlot, 0),
		head:   0,
		inRing: make(map[model.PageKey]struct{}),
		ref:    make(map[model.PageKey]struct{}),
	}
}

func (c *ClockReplacer) Push(slot *model.BufferSlot) {
	if slot == nil || slot.Pinned() {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	key := slot.Key()
	if _, ok := c.inRing[key]; !ok {
		c.inRing[key] = struct{}{}
		c.ring = append(c.ring, slot)
	}
	c.ref[key] = struct{}{}
}

func (c *ClockReplacer) Delete(key model.PageKey) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.inRing[key]; !ok {
		return
	}
	delete(c.inRing, key)
	delete(c.ref, key)

	for i, s := range c.ring {
		if s.Key() == key {
			copy(c.ring[i:], c.ring[i+1:])
			c.ring = c.ring[:len(c.ring)-1]

			if len(c.ring) == 0 {
				c.head = 0
			} else if i < c.head {
				c.head--
			} else if c.head >= len(c.ring) {
				c.head = 0
			}
			break
		}
	}
}

func (c *ClockReplacer) PickVictim() *model.BufferSlot {
	c.mu.Lock()
	defer c.mu.Unlock()

	for len(c.ring) > 0 {
		if c.head >= len(c.ring) {
			c.head = 0
		}

		slot := c.ring[c.head]
		key := slot.Key()

		if _, ok := c.ref[key]; ok {
			delete(c.ref, key)
			c.head++
			continue
		}

		delete(c.inRing, key)

		copy(c.ring[c.head:], c.ring[c.head+1:])
		c.ring = c.ring[:len(c.ring)-1]
		if c.head >= len(c.ring) {
			c.head = 0
		}

		return slot
	}

	return nil
}
