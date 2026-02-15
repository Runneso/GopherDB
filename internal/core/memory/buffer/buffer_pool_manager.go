package buffer

import (
	"GopherDB/internal/core/memory/model"
	"GopherDB/internal/core/memory/page"
)

type BufferPoolManager interface {
	GetPage(model.PageKey) (*model.BufferSlot, error)
	NewPage(model.PageKey, page.Page) (*model.BufferSlot, error)
	UpdatePage(model.PageKey, page.Page) error
	PinPage(model.PageKey) error
	UnpinPage(model.PageKey) error
	FlushPage(model.PageKey) error
	FlushAllPages() error
	DirtyPages() []*model.BufferSlot
}
