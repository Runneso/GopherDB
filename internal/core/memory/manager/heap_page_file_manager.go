package manager

import (
	"GopherDB/internal/core/memory/page"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var (
	ErrInvalidPageID    = errors.New("invalid page id")
	ErrInvalidSignature = errors.New("invalid page signature")
	ErrOutOfBounds      = errors.New("page out of file bounds")
)

type HeapPageFileManager struct{}

func (heapPageFileManager *HeapPageFileManager) Write(page_ page.Page, path string) error {
	if path == "" {
		return fmt.Errorf("path: empty")
	}
	if page_ == nil {
		return fmt.Errorf("page: nil")
	}
	if page_.PageID() < 0 {
		return fmt.Errorf("%w: %d", ErrInvalidPageID, page_.PageID())
	}
	if !page_.IsValid() {
		return ErrInvalidSignature
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer file.Close()

	offset := int64(page_.PageID()) * int64(page.PageSize)
	_, err = file.WriteAt(page_.Bytes(), offset)
	if err != nil {
		return fmt.Errorf("write page %d: %w", page_.PageID(), err)
	}
	return nil
}

func (heapPageFileManager *HeapPageFileManager) Read(pageID int, path string) (page.Page, error) {
	if path == "" {
		return nil, fmt.Errorf("path: empty")
	}
	if pageID < 0 {
		return nil, fmt.Errorf("%w: %d", ErrInvalidPageID, pageID)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer file.Close()

	stats, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}

	offset := int64(pageID) * int64(page.PageSize)
	end := offset + int64(page.PageSize)
	if end > stats.Size() {
		return nil, fmt.Errorf("%w: page=%d fileSize=%d", ErrOutOfBounds, pageID, stats.Size())
	}

	buffer := make([]byte, page.PageSize)
	reader := io.NewSectionReader(file, offset, int64(page.PageSize))
	if _, err := io.ReadFull(reader, buffer); err != nil {
		return nil, fmt.Errorf("read page %d: %w", pageID, err)
	}

	result, err := page.NewHeapPageFromBuffer(pageID, buffer)
	if err != nil {
		return nil, fmt.Errorf("%w: page=%d: %v", ErrInvalidSignature, pageID, err)
	}
	return result, nil
}
