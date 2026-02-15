package manager

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"GopherDB/internal/core/catalog/model"
	"GopherDB/internal/core/memory/buffer"
	memmodel "GopherDB/internal/core/memory/model"
	"GopherDB/internal/core/memory/page"
	"GopherDB/internal/core/types"
)

const (
	builtinInt64   = "INT64"
	builtinVarchar = "VARCHAR"

	tablesFile  = "table_definitions.dat"
	columnsFile = "column_definitions.dat"
	typesFile   = "types_definitions.dat"
	indexesFile = "index_definitions.dat"
)

var (
	ErrTableExists    = errors.New("table exists")
	ErrTableNotFound  = errors.New("table not found")
	ErrColumnNotFound = errors.New("column not found")
	ErrIndexExists    = errors.New("index exists")
	ErrCorruptedFile  = errors.New("corrupted catalog file")
)

type DefaultCatalogManager struct {
	mu         sync.Mutex
	root       string
	bufferPool buffer.BufferPoolManager

	tablesByOid  map[int32]*model.TableDefinition
	tablesByName map[string]*model.TableDefinition

	columnsByOid      map[int32]*model.ColumnDefinition
	columnsByTableOid map[int32][]*model.ColumnDefinition

	typesByOid  map[int32]*model.TypeDefinition
	typesByName map[string]*model.TypeDefinition

	indexesByOid      map[int32]*model.IndexDefinition
	indexesByName     map[string]*model.IndexDefinition
	indexesByTableOid map[int32][]*model.IndexDefinition

	nextTableOid  int32
	nextColumnOid int32
	nextTypeOid   int32
	nextIndexOid  int32
}

func NewDefaultCatalogManager(root string, bufferPool buffer.BufferPoolManager) (*DefaultCatalogManager, error) {
	cm := &DefaultCatalogManager{
		root:              root,
		bufferPool:        bufferPool,
		tablesByOid:       make(map[int32]*model.TableDefinition),
		tablesByName:      make(map[string]*model.TableDefinition),
		columnsByOid:      make(map[int32]*model.ColumnDefinition),
		columnsByTableOid: make(map[int32][]*model.ColumnDefinition),
		typesByOid:        make(map[int32]*model.TypeDefinition),
		typesByName:       make(map[string]*model.TypeDefinition),
		indexesByOid:      make(map[int32]*model.IndexDefinition),
		indexesByName:     make(map[string]*model.IndexDefinition),
		indexesByTableOid: make(map[int32][]*model.IndexDefinition),
		nextTableOid:      1,
		nextColumnOid:     1,
		nextTypeOid:       1,
		nextIndexOid:      1,
	}

	if err := cm.ensureCatalogDirectory(); err != nil {
		return nil, err
	}
	if err := cm.loadFromDisk(); err != nil {
		return nil, err
	}
	if err := cm.ensureBuiltinTypesPresent(); err != nil {
		return nil, err
	}

	return cm, nil
}

func (defaultCatalogManager *DefaultCatalogManager) CreateTable(name string, columns []*model.ColumnDefinition) (*model.TableDefinition, error) {
	defaultCatalogManager.mu.Lock()
	defer defaultCatalogManager.mu.Unlock()

	if _, ok := defaultCatalogManager.tablesByName[name]; ok {
		return nil, fmt.Errorf("%w: %s", ErrTableExists, name)
	}

	oid := defaultCatalogManager.nextTableOid
	defaultCatalogManager.nextTableOid++

	table, err := model.NewTableDefinition(oid, name, "TABLE", fmt.Sprintf("%d.dat", oid), 0)
	if err != nil {
		return nil, err
	}

	defaultCatalogManager.indexTable(table)
	data, err := table.ToBytes()
	if err != nil {
		return nil, err
	}
	if err := defaultCatalogManager.appendRecord(tablesFile, data); err != nil {
		return nil, err
	}

	for position, proto := range columns {
		colOid := defaultCatalogManager.nextColumnOid
		defaultCatalogManager.nextColumnOid++

		col, err := model.NewColumnDefinition(colOid, table.Oid(), proto.TypeOid(), proto.Name(), int32(position))
		if err != nil {
			return nil, err
		}

		defaultCatalogManager.indexColumn(col)
		colData, err := col.ToBytes()
		if err != nil {
			return nil, err
		}
		if err := defaultCatalogManager.appendRecord(columnsFile, colData); err != nil {
			return nil, err
		}
	}

	return table, nil
}

func (defaultCatalogManager *DefaultCatalogManager) GetTable(tableName string) (*model.TableDefinition, error) {
	defaultCatalogManager.mu.Lock()
	defer defaultCatalogManager.mu.Unlock()

	table, ok := defaultCatalogManager.tablesByName[tableName]
	if !ok {
		return nil, nil
	}
	return table, nil
}

func (defaultCatalogManager *DefaultCatalogManager) ListTables() ([]*model.TableDefinition, error) {
	defaultCatalogManager.mu.Lock()
	defer defaultCatalogManager.mu.Unlock()

	result := make([]*model.TableDefinition, 0, len(defaultCatalogManager.tablesByOid))
	for _, t := range defaultCatalogManager.tablesByOid {
		result = append(result, t)
	}
	return result, nil
}

func (defaultCatalogManager *DefaultCatalogManager) GetColumns(table *model.TableDefinition) ([]*model.ColumnDefinition, error) {
	defaultCatalogManager.mu.Lock()
	defer defaultCatalogManager.mu.Unlock()

	columns := defaultCatalogManager.columnsByTableOid[table.Oid()]
	if columns == nil {
		return []*model.ColumnDefinition{}, nil
	}
	return columns, nil
}

func (defaultCatalogManager *DefaultCatalogManager) GetColumn(table *model.TableDefinition, columnName string) (*model.ColumnDefinition, error) {
	defaultCatalogManager.mu.Lock()
	defer defaultCatalogManager.mu.Unlock()

	columns := defaultCatalogManager.columnsByTableOid[table.Oid()]
	for _, c := range columns {
		if c.Name() == columnName {
			return c, nil
		}
	}
	return nil, nil
}

func (defaultCatalogManager *DefaultCatalogManager) GetTypeByOid(oid int32) (*model.TypeDefinition, error) {
	defaultCatalogManager.mu.Lock()
	defer defaultCatalogManager.mu.Unlock()

	typ, ok := defaultCatalogManager.typesByOid[oid]
	if !ok {
		return nil, nil
	}
	return typ, nil
}

func (defaultCatalogManager *DefaultCatalogManager) GetTypeByName(name string) (*model.TypeDefinition, error) {
	defaultCatalogManager.mu.Lock()
	defer defaultCatalogManager.mu.Unlock()

	typ, ok := defaultCatalogManager.typesByName[name]
	if !ok {
		return nil, nil
	}
	return typ, nil
}

func (defaultCatalogManager *DefaultCatalogManager) UpdatePagesCount(table *model.TableDefinition, pagesCount int32) error {
	defaultCatalogManager.mu.Lock()
	defer defaultCatalogManager.mu.Unlock()

	updated, err := model.NewTableDefinition(table.Oid(), table.Name(), table.Type(), table.FileNode(), pagesCount)
	if err != nil {
		return err
	}

	defaultCatalogManager.indexTable(updated)
	data, err := updated.ToBytes()
	if err != nil {
		return err
	}
	return defaultCatalogManager.appendRecord(tablesFile, data)
}

func (defaultCatalogManager *DefaultCatalogManager) CreateIndex(indexName, tableName, columnName string, indexType types.IndexType) (*model.IndexDefinition, error) {
	defaultCatalogManager.mu.Lock()
	defer defaultCatalogManager.mu.Unlock()

	if _, ok := defaultCatalogManager.indexesByName[indexName]; ok {
		return nil, fmt.Errorf("%w: %s", ErrIndexExists, indexName)
	}

	table, ok := defaultCatalogManager.tablesByName[tableName]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrTableNotFound, tableName)
	}

	var column *model.ColumnDefinition
	for _, c := range defaultCatalogManager.columnsByTableOid[table.Oid()] {
		if c.Name() == columnName {
			column = c
			break
		}
	}
	if column == nil {
		return nil, fmt.Errorf("%w: %s.%s", ErrColumnNotFound, tableName, columnName)
	}

	oid := defaultCatalogManager.nextIndexOid
	defaultCatalogManager.nextIndexOid++
	fileNode := fmt.Sprintf("%d.idx", oid)

	definition, err := model.NewIndexDefinition(oid, indexName, table.Oid(), column.Oid(), column.TypeOid(), indexType, fileNode, 0, 0)
	if err != nil {
		return nil, err
	}

	defaultCatalogManager.indexIndex(definition)
	data, err := definition.ToBytes()
	if err != nil {
		return nil, err
	}
	if err := defaultCatalogManager.appendRecord(indexesFile, data); err != nil {
		return nil, err
	}

	return definition, nil
}

func (defaultCatalogManager *DefaultCatalogManager) GetIndex(indexName string) (*model.IndexDefinition, error) {
	defaultCatalogManager.mu.Lock()
	defer defaultCatalogManager.mu.Unlock()

	idx, ok := defaultCatalogManager.indexesByName[indexName]
	if !ok {
		return nil, nil
	}
	return idx, nil
}

func (defaultCatalogManager *DefaultCatalogManager) ListIndexes(table *model.TableDefinition) ([]*model.IndexDefinition, error) {
	defaultCatalogManager.mu.Lock()
	defer defaultCatalogManager.mu.Unlock()

	indexes := defaultCatalogManager.indexesByTableOid[table.Oid()]
	if indexes == nil {
		return []*model.IndexDefinition{}, nil
	}
	return indexes, nil
}

func (defaultCatalogManager *DefaultCatalogManager) ensureCatalogDirectory() error {
	return os.MkdirAll(defaultCatalogManager.root, 0755)
}

func (defaultCatalogManager *DefaultCatalogManager) loadFromDisk() error {
	if err := defaultCatalogManager.readAllRecords(typesFile, func(data []byte) error {
		t, err := model.TypeDefinitionFromBytes(data)
		if err != nil {
			return err
		}
		defaultCatalogManager.indexType(t)
		return nil
	}); err != nil {
		return err
	}

	if err := defaultCatalogManager.readAllRecords(tablesFile, func(data []byte) error {
		t, err := model.TableDefinitionFromBytes(data)
		if err != nil {
			return err
		}
		defaultCatalogManager.indexTable(t)
		return nil
	}); err != nil {
		return err
	}

	if err := defaultCatalogManager.readAllRecords(columnsFile, func(data []byte) error {
		c, err := model.ColumnDefinitionFromBytes(data)
		if err != nil {
			return err
		}
		defaultCatalogManager.indexColumn(c)
		return nil
	}); err != nil {
		return err
	}

	if err := defaultCatalogManager.readAllRecords(indexesFile, func(data []byte) error {
		idx, err := model.IndexDefinitionFromBytes(data)
		if err != nil {
			return err
		}
		defaultCatalogManager.indexIndex(idx)
		return nil
	}); err != nil {
		return err
	}

	defaultCatalogManager.recomputeNextOids()
	return nil
}

func (defaultCatalogManager *DefaultCatalogManager) readAllRecords(fileID string, consumer func([]byte) error) error {
	filePath := filepath.Join(defaultCatalogManager.root, fileID)

	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	size := info.Size()
	if size == 0 {
		return nil
	}
	if size%page.PageSize != 0 {
		return fmt.Errorf("%w: %s", ErrCorruptedFile, filePath)
	}

	pages := int(size / page.PageSize)
	for pageID := 0; pageID < pages; pageID++ {
		key := memmodel.NewPageKey(fileID, pageID)
		slot, err := defaultCatalogManager.bufferPool.GetPage(key)
		if err != nil {
			return err
		}
		p := slot.Page()
		for recordID := 0; recordID < p.Size(); recordID++ {
			data, err := p.Read(recordID)
			if err != nil {
				return err
			}
			if err := consumer(data); err != nil {
				return err
			}
		}
	}
	return nil
}

func (defaultCatalogManager *DefaultCatalogManager) pageCount(fileID string) (int, error) {
	filePath := filepath.Join(defaultCatalogManager.root, fileID)

	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	size := info.Size()
	if size == 0 {
		return 0, nil
	}
	if size%page.PageSize != 0 {
		return 0, fmt.Errorf("%w: %s", ErrCorruptedFile, filePath)
	}

	return int(size / page.PageSize), nil
}

func (defaultCatalogManager *DefaultCatalogManager) appendRecord(fileID string, data []byte) error {
	pages, err := defaultCatalogManager.pageCount(fileID)
	if err != nil {
		return err
	}

	if pages == 0 {
		key := memmodel.NewPageKey(fileID, 0)
		p := page.NewHeapPage(0)
		if _, err := defaultCatalogManager.bufferPool.NewPage(key, p); err != nil {
			return err
		}
		if err := p.Write(data); err != nil {
			return err
		}
		if err := defaultCatalogManager.bufferPool.UpdatePage(key, p); err != nil {
			return err
		}
		return defaultCatalogManager.bufferPool.FlushPage(key)
	}

	lastKey := memmodel.NewPageKey(fileID, pages-1)
	slot, err := defaultCatalogManager.bufferPool.GetPage(lastKey)
	if err != nil {
		return err
	}
	last := slot.Page()

	if err := last.Write(data); err != nil {
		newKey := memmodel.NewPageKey(fileID, pages)
		p := page.NewHeapPage(pages)
		if _, err := defaultCatalogManager.bufferPool.NewPage(newKey, p); err != nil {
			return err
		}
		if err := p.Write(data); err != nil {
			return err
		}
		if err := defaultCatalogManager.bufferPool.UpdatePage(newKey, p); err != nil {
			return err
		}
		return defaultCatalogManager.bufferPool.FlushPage(newKey)
	}

	if err := defaultCatalogManager.bufferPool.UpdatePage(lastKey, last); err != nil {
		return err
	}
	return defaultCatalogManager.bufferPool.FlushPage(lastKey)
}

func (defaultCatalogManager *DefaultCatalogManager) indexType(typ *model.TypeDefinition) {
	defaultCatalogManager.typesByOid[typ.Oid()] = typ
	defaultCatalogManager.typesByName[typ.Name()] = typ
}

func (defaultCatalogManager *DefaultCatalogManager) indexTable(table *model.TableDefinition) {
	defaultCatalogManager.tablesByOid[table.Oid()] = table
	defaultCatalogManager.tablesByName[table.Name()] = table
}

func (defaultCatalogManager *DefaultCatalogManager) indexColumn(column *model.ColumnDefinition) {
	defaultCatalogManager.columnsByOid[column.Oid()] = column

	list := defaultCatalogManager.columnsByTableOid[column.TableOid()]
	filtered := make([]*model.ColumnDefinition, 0, len(list))
	for _, existing := range list {
		if existing.Oid() != column.Oid() {
			filtered = append(filtered, existing)
		}
	}
	filtered = append(filtered, column)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Position() < filtered[j].Position()
	})
	defaultCatalogManager.columnsByTableOid[column.TableOid()] = filtered
}

func (defaultCatalogManager *DefaultCatalogManager) indexIndex(idx *model.IndexDefinition) {
	defaultCatalogManager.indexesByOid[idx.Oid()] = idx
	defaultCatalogManager.indexesByName[idx.Name()] = idx

	list := defaultCatalogManager.indexesByTableOid[idx.TableOid()]
	filtered := make([]*model.IndexDefinition, 0, len(list))
	for _, existing := range list {
		if existing.Oid() != idx.Oid() {
			filtered = append(filtered, existing)
		}
	}
	filtered = append(filtered, idx)
	defaultCatalogManager.indexesByTableOid[idx.TableOid()] = filtered
}

func (defaultCatalogManager *DefaultCatalogManager) recomputeNextOids() {
	defaultCatalogManager.nextTableOid = defaultCatalogManager.nextIdAfterMax(defaultCatalogManager.tablesByOid)
	defaultCatalogManager.nextColumnOid = defaultCatalogManager.nextIdAfterMax(defaultCatalogManager.columnsByOid)
	defaultCatalogManager.nextTypeOid = defaultCatalogManager.nextIdAfterMax(defaultCatalogManager.typesByOid)
	defaultCatalogManager.nextIndexOid = defaultCatalogManager.nextIdAfterMax(defaultCatalogManager.indexesByOid)
}

func (defaultCatalogManager *DefaultCatalogManager) nextIdAfterMax(m interface{}) int32 {
	var maxOid int32 = 0
	switch v := m.(type) {
	case map[int32]*model.TableDefinition:
		for oid := range v {
			if oid > maxOid {
				maxOid = oid
			}
		}
	case map[int32]*model.ColumnDefinition:
		for oid := range v {
			if oid > maxOid {
				maxOid = oid
			}
		}
	case map[int32]*model.TypeDefinition:
		for oid := range v {
			if oid > maxOid {
				maxOid = oid
			}
		}
	case map[int32]*model.IndexDefinition:
		for oid := range v {
			if oid > maxOid {
				maxOid = oid
			}
		}
	}
	return maxOid + 1
}

func (defaultCatalogManager *DefaultCatalogManager) ensureBuiltinTypesPresent() error {
	changed := false

	if _, ok := defaultCatalogManager.typesByName[builtinInt64]; !ok {
		typ, _ := model.NewTypeDefinition(defaultCatalogManager.nextTypeOid, builtinInt64, 8)
		defaultCatalogManager.nextTypeOid++
		defaultCatalogManager.indexType(typ)
		data, err := typ.ToBytes()
		if err != nil {
			return err
		}
		if err := defaultCatalogManager.appendRecord(typesFile, data); err != nil {
			return err
		}
		changed = true
	}

	if _, ok := defaultCatalogManager.typesByName[builtinVarchar]; !ok {
		typ, _ := model.NewTypeDefinition(defaultCatalogManager.nextTypeOid, builtinVarchar, -1)
		defaultCatalogManager.nextTypeOid++
		defaultCatalogManager.indexType(typ)
		data, err := typ.ToBytes()
		if err != nil {
			return err
		}
		if err := defaultCatalogManager.appendRecord(typesFile, data); err != nil {
			return err
		}
		changed = true
	}

	if changed {
		defaultCatalogManager.recomputeNextOids()
	}
	return nil
}
