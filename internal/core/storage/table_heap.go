package storage

import (
	"GopherDB/internal/core/catalog/manager"
	"GopherDB/internal/core/catalog/model"
	"GopherDB/internal/core/index"
	"GopherDB/internal/core/memory/buffer"
	memmodel "GopherDB/internal/core/memory/model"
	"GopherDB/internal/core/memory/page"
	"GopherDB/internal/core/memory/serializer"
	"fmt"
	"strings"
)

type TableHeap struct {
	root       string
	bufferPool buffer.BufferPoolManager
	catalog    manager.CatalogManager
	table      *model.TableDefinition
	columns    []*model.ColumnDefinition
	serializer *serializer.HeapTupleSerializer
}

func NewTableHeap(root string, bufferPool buffer.BufferPoolManager, catalog manager.CatalogManager, table *model.TableDefinition) (*TableHeap, error) {
	columns, err := catalog.GetColumns(table)
	if err != nil {
		return nil, err
	}
	return &TableHeap{
		root:       root,
		bufferPool: bufferPool,
		catalog:    catalog,
		table:      table,
		columns:    columns,
		serializer: serializer.NewHeapTupleSerializer(),
	}, nil
}

func (heap *TableHeap) Table() *model.TableDefinition {
	return heap.table
}

func (heap *TableHeap) ScanTids() ([]index.TID, error) {
	var tids []index.TID
	pagesCount := heap.table.PagesCount()

	for pageID := int32(0); pageID < pagesCount; pageID++ {
		slot, err := heap.bufferPool.GetPage(heap.pageKey(int(pageID)))
		if err != nil {
			continue
		}
		heapPage, err := page.NewHeapPageFromBuffer(int(pageID), slot.Page().Bytes())
		if err != nil {
			continue
		}
		slotCount := heapPage.Size()
		for slotID := 0; slotID < slotCount; slotID++ {
			tids = append(tids, index.NewTID(pageID, int16(slotID)))
		}
	}

	return tids, nil
}

func (heap *TableHeap) ReadRow(tid index.TID) ([]any, error) {
	slot, err := heap.bufferPool.GetPage(heap.pageKey(int(tid.PageID())))
	if err != nil {
		return nil, err
	}

	heapPage, err := page.NewHeapPageFromBuffer(int(tid.PageID()), slot.Page().Bytes())
	if err != nil {
		return nil, err
	}

	data, err := heapPage.Read(int(tid.SlotID()))
	if err != nil {
		return nil, err
	}

	return heap.deserializeRow(data)
}

func (heap *TableHeap) InsertRow(values []any) (index.TID, error) {
	data, err := heap.serializeRow(values)
	if err != nil {
		return index.TID{}, err
	}

	pagesCount := heap.table.PagesCount()
	for pageID := int32(0); pageID < pagesCount; pageID++ {
		tid, err := heap.tryInsertIntoPage(int(pageID), data)
		if err == nil {
			return tid, nil
		}
	}

	return heap.insertIntoNewPage(data)
}

func (heap *TableHeap) tryInsertIntoPage(pageID int, data []byte) (index.TID, error) {
	slot, err := heap.bufferPool.GetPage(heap.pageKey(pageID))
	if err != nil {
		return index.TID{}, err
	}

	heapPage, err := page.NewHeapPageFromBuffer(pageID, slot.Page().Bytes())
	if err != nil {
		return index.TID{}, err
	}

	slotID := heapPage.Size()
	if err := heapPage.Write(data); err != nil {
		return index.TID{}, err
	}

	if err := heap.bufferPool.UpdatePage(heap.pageKey(pageID), heapPage); err != nil {
		return index.TID{}, err
	}

	return index.NewTID(int32(pageID), int16(slotID)), nil
}

func (heap *TableHeap) insertIntoNewPage(data []byte) (index.TID, error) {
	pageID := int(heap.table.PagesCount())
	newPage := page.NewHeapPage(pageID)

	if err := newPage.Write(data); err != nil {
		return index.TID{}, err
	}

	if _, err := heap.bufferPool.NewPage(heap.pageKey(pageID), newPage); err != nil {
		return index.TID{}, err
	}

	if err := heap.catalog.UpdatePagesCount(heap.table, int32(pageID+1)); err != nil {
		return index.TID{}, err
	}

	return index.NewTID(int32(pageID), 0), nil
}

func (heap *TableHeap) serializeRow(values []any) ([]byte, error) {
	var data []byte
	for i, col := range heap.columns {
		typeDef, err := heap.catalog.GetTypeByOid(col.TypeOid())
		if err != nil {
			return nil, err
		}
		dt := heap.typeNameToDataType(typeDef.Name())
		if dt == 0 {
			return nil, fmt.Errorf("unknown type: name=%q oid=%d", typeDef.Name(), typeDef.Oid())
		}
		tuple, err := heap.serializer.Serialize(values[i], dt)
		if err != nil {
			return nil, err
		}
		data = append(data, tuple.Data()...)
	}
	return data, nil
}

func (heap *TableHeap) deserializeRow(data []byte) ([]any, error) {
	var row []any
	offset := 0

	for _, col := range heap.columns {
		typeDef, err := heap.catalog.GetTypeByOid(col.TypeOid())
		if err != nil {
			return nil, err
		}

		dt := heap.typeNameToDataType(typeDef.Name())
		if dt == 0 {
			return nil, fmt.Errorf("unknown type: name=%q oid=%d", typeDef.Name(), typeDef.Oid())
		}
		length := heap.getFieldLength(dt, data[offset:])
		fieldData := data[offset : offset+length]

		tuple := memmodel.NewHeapTuple(fieldData, dt)
		value, err := heap.serializer.Deserialize(tuple)
		if err != nil {
			return nil, err
		}

		row = append(row, value)
		offset += length
	}

	return row, nil
}

func (heap *TableHeap) getFieldLength(dt memmodel.DataType, data []byte) int {
	if dt == memmodel.INT64 {
		return 8
	}
	if dt == memmodel.VARCHAR {
		return 1 + int(data[0])
	}
	return 0
}

func (heap *TableHeap) typeNameToDataType(name string) memmodel.DataType {
	normalized := strings.ToUpper(strings.TrimSpace(strings.TrimRight(name, "\x00")))
	switch normalized {
	case "INT64", "INT", "INTEGER":
		return memmodel.INT64
	case "VARCHAR":
		return memmodel.VARCHAR
	default:
		return 0
	}
}

func (heap *TableHeap) pageKey(pageID int) memmodel.PageKey {
	return memmodel.NewPageKey(heap.table.FileNode(), pageID)
}
