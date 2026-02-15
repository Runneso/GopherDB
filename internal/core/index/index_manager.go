package index

import (
	"errors"
	"sync"

	"GopherDB/internal/core/catalog/manager"
	"GopherDB/internal/core/catalog/model"
	"GopherDB/internal/core/memory/buffer"
	"GopherDB/internal/core/types"
)

var ErrIndexNotLoaded = errors.New("index not loaded")

type indexFactory func(string, buffer.BufferPoolManager, manager.CatalogManager, *model.IndexDefinition) (Index, error)

type IndexManager struct {
	mu         sync.Mutex
	root       string
	bufferPool buffer.BufferPoolManager
	catalog    manager.CatalogManager
	byName     map[string]Index
	factories  map[types.IndexType]indexFactory
}

func NewIndexManager(root string, bufferPool buffer.BufferPoolManager, catalog manager.CatalogManager) *IndexManager {
	m := &IndexManager{
		root:       root,
		bufferPool: bufferPool,
		catalog:    catalog,
		byName:     make(map[string]Index),
	}
	m.factories = map[types.IndexType]indexFactory{
		types.IndexTypeHash: func(r string, bp buffer.BufferPoolManager, c manager.CatalogManager, d *model.IndexDefinition) (Index, error) {
			return NewDiskHashIndex(r, bp, c, d)
		},
		types.IndexTypeBTree: func(r string, bp buffer.BufferPoolManager, c manager.CatalogManager, d *model.IndexDefinition) (Index, error) {
			return NewDiskBTreeIndex(r, bp, c, d)
		},
	}
	return m
}

func (m *IndexManager) GetOrCreate(def *model.IndexDefinition) (Index, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if idx, ok := m.byName[def.Name()]; ok {
		return idx, nil
	}

	factory, ok := m.factories[def.IndexType()]
	if !ok {
		return nil, ErrNotHashIndex
	}

	idx, err := factory(m.root, m.bufferPool, m.catalog, def)
	if err != nil {
		return nil, err
	}

	m.byName[def.Name()] = idx
	return idx, nil
}

func (m *IndexManager) Get(indexName string) (Index, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx, ok := m.byName[indexName]
	if !ok {
		return nil, ErrIndexNotLoaded
	}
	return idx, nil
}
