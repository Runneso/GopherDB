package replacer

import (
	"GopherDB/internal/core/memory/model"
)

type Replacer interface {
	Push(*model.BufferSlot)
	Delete(model.PageKey)
	PickVictim() *model.BufferSlot
}
