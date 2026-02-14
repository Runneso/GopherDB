package manager

import "GopherDB/internal/core/memory/page"

type PageFileManager interface {
	Write(page.Page, string) error
	Read(int, string) (page.Page, error)
}
