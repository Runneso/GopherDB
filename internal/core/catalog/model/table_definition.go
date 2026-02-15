package model

import (
	"encoding/binary"
	"errors"
	"strings"
)

var (
	ErrTypeRequired     = errors.New("type is required")
	ErrFileNodeRequired = errors.New("fileNode is required")
	ErrStringTooLong    = errors.New("string too long")
)

type TableDefinition struct {
	oid        int32
	name       string
	typ        string
	fileNode   string
	pagesCount int32
}

func NewTableDefinition(oid int32, name, typ, fileNode string, pagesCount int32) (*TableDefinition, error) {
	if strings.TrimSpace(name) == "" {
		return nil, ErrNameRequired
	}
	if strings.TrimSpace(typ) == "" {
		return nil, ErrTypeRequired
	}
	if strings.TrimSpace(fileNode) == "" {
		return nil, ErrFileNodeRequired
	}
	return &TableDefinition{
		oid:        oid,
		name:       name,
		typ:        typ,
		fileNode:   fileNode,
		pagesCount: pagesCount,
	}, nil
}

func (table *TableDefinition) Oid() int32 {
	return table.oid
}

func (table *TableDefinition) Name() string {
	return table.name
}

func (table *TableDefinition) Type() string {
	return table.typ
}

func (table *TableDefinition) FileNode() string {
	return table.fileNode
}

func (table *TableDefinition) PagesCount() int32 {
	return table.pagesCount
}

func (table *TableDefinition) SetPagesCount(v int32) {
	table.pagesCount = v
}

func (table *TableDefinition) ToBytes() ([]byte, error) {
	nameBytes := []byte(table.name)
	typeBytes := []byte(table.typ)
	fileBytes := []byte(table.fileNode)

	if len(nameBytes) > maxUint16 || len(typeBytes) > maxUint16 || len(fileBytes) > maxUint16 {
		return nil, ErrStringTooLong
	}

	buf := make([]byte, intSize+shortSize+len(nameBytes)+shortSize+len(typeBytes)+shortSize+len(fileBytes)+intSize)
	offset := 0

	binary.BigEndian.PutUint32(buf[offset:], uint32(table.oid))
	offset += intSize

	binary.BigEndian.PutUint16(buf[offset:], uint16(len(nameBytes)))
	offset += shortSize

	copy(buf[offset:], nameBytes)
	offset += len(nameBytes)

	binary.BigEndian.PutUint16(buf[offset:], uint16(len(typeBytes)))
	offset += shortSize

	copy(buf[offset:], typeBytes)
	offset += len(typeBytes)

	binary.BigEndian.PutUint16(buf[offset:], uint16(len(fileBytes)))
	offset += shortSize

	copy(buf[offset:], fileBytes)
	offset += len(fileBytes)

	binary.BigEndian.PutUint32(buf[offset:], uint32(table.pagesCount))

	return buf, nil
}

func TableDefinitionFromBytes(data []byte) (*TableDefinition, error) {
	minSize := intSize + shortSize*3 + intSize
	if len(data) < minSize {
		return nil, ErrDataTooShort
	}

	offset := 0

	oid := int32(binary.BigEndian.Uint32(data[offset:]))
	offset += intSize

	nameLen := int(binary.BigEndian.Uint16(data[offset:]))
	offset += shortSize

	name := string(data[offset : offset+nameLen])
	offset += nameLen

	typeLen := int(binary.BigEndian.Uint16(data[offset:]))
	offset += shortSize

	typ := string(data[offset : offset+typeLen])
	offset += typeLen

	fileLen := int(binary.BigEndian.Uint16(data[offset:]))
	offset += shortSize

	fileNode := string(data[offset : offset+fileLen])
	offset += fileLen

	pagesCount := int32(binary.BigEndian.Uint32(data[offset:]))

	return &TableDefinition{
		oid:        oid,
		name:       name,
		typ:        typ,
		fileNode:   fileNode,
		pagesCount: pagesCount,
	}, nil
}
