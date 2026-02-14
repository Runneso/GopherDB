package model

import (
	"encoding/binary"

	"GopherDB/internal/core/index"
)

type IndexDefinition struct {
	oid        int32
	name       string
	tableOid   int32
	columnOid  int32
	keyTypeOid int32
	indexType  index.IndexType
	fileNode   string
	metaPageId int32
	rootPageId int32
}

func NewIndexDefinition(
	oid int32,
	name string,
	tableOid int32,
	columnOid int32,
	keyTypeOid int32,
	indexType index.IndexType,
	fileNode string,
	metaPageId int32,
	rootPageId int32,
) (*IndexDefinition, error) {
	if name == "" {
		return nil, ErrNameRequired
	}
	if fileNode == "" {
		return nil, ErrFileNodeRequired
	}
	return &IndexDefinition{
		oid:        oid,
		name:       name,
		tableOid:   tableOid,
		columnOid:  columnOid,
		keyTypeOid: keyTypeOid,
		indexType:  indexType,
		fileNode:   fileNode,
		metaPageId: metaPageId,
		rootPageId: rootPageId,
	}, nil
}

func (idx *IndexDefinition) Oid() int32 {
	return idx.oid
}

func (idx *IndexDefinition) Name() string {
	return idx.name
}

func (idx *IndexDefinition) TableOid() int32 {
	return idx.tableOid
}

func (idx *IndexDefinition) ColumnOid() int32 {
	return idx.columnOid
}

func (idx *IndexDefinition) KeyTypeOid() int32 {
	return idx.keyTypeOid
}

func (idx *IndexDefinition) IndexType() index.IndexType {
	return idx.indexType
}

func (idx *IndexDefinition) FileNode() string {
	return idx.fileNode
}

func (idx *IndexDefinition) MetaPageId() int32 {
	return idx.metaPageId
}

func (idx *IndexDefinition) RootPageId() int32 {
	return idx.rootPageId
}

func (idx *IndexDefinition) ToBytes() ([]byte, error) {
	nameBytes := []byte(idx.name)
	fileBytes := []byte(idx.fileNode)

	if len(nameBytes) > maxUint16 || len(fileBytes) > maxUint16 {
		return nil, ErrStringTooLong
	}

	buf := make([]byte, intSize*7+shortSize+len(nameBytes)+shortSize+len(fileBytes))
	offset := 0

	binary.BigEndian.PutUint32(buf[offset:], uint32(idx.oid))
	offset += intSize

	binary.BigEndian.PutUint32(buf[offset:], uint32(idx.tableOid))
	offset += intSize

	binary.BigEndian.PutUint32(buf[offset:], uint32(idx.columnOid))
	offset += intSize

	binary.BigEndian.PutUint32(buf[offset:], uint32(idx.keyTypeOid))
	offset += intSize

	binary.BigEndian.PutUint32(buf[offset:], uint32(idx.indexType.Ordinal()))
	offset += intSize

	binary.BigEndian.PutUint32(buf[offset:], uint32(idx.metaPageId))
	offset += intSize

	binary.BigEndian.PutUint32(buf[offset:], uint32(idx.rootPageId))
	offset += intSize

	binary.BigEndian.PutUint16(buf[offset:], uint16(len(nameBytes)))
	offset += shortSize

	copy(buf[offset:], nameBytes)
	offset += len(nameBytes)

	binary.BigEndian.PutUint16(buf[offset:], uint16(len(fileBytes)))
	offset += shortSize
	copy(buf[offset:], fileBytes)

	return buf, nil
}

func IndexDefinitionFromBytes(data []byte) (*IndexDefinition, error) {
	minSize := intSize*7 + shortSize*2
	if len(data) < minSize {
		return nil, ErrDataTooShort
	}

	offset := 0

	oid := int32(binary.BigEndian.Uint32(data[offset:]))
	offset += intSize

	tableOid := int32(binary.BigEndian.Uint32(data[offset:]))
	offset += intSize

	columnOid := int32(binary.BigEndian.Uint32(data[offset:]))
	offset += intSize

	keyTypeOid := int32(binary.BigEndian.Uint32(data[offset:]))
	offset += intSize

	indexTypeOrd := int32(binary.BigEndian.Uint32(data[offset:]))
	offset += intSize

	metaPageId := int32(binary.BigEndian.Uint32(data[offset:]))
	offset += intSize

	rootPageId := int32(binary.BigEndian.Uint32(data[offset:]))
	offset += intSize

	nameLen := int(binary.BigEndian.Uint16(data[offset:]))
	offset += shortSize

	name := string(data[offset : offset+nameLen])
	offset += nameLen

	fileLen := int(binary.BigEndian.Uint16(data[offset:]))
	offset += shortSize

	fileNode := string(data[offset : offset+fileLen])

	return &IndexDefinition{
		oid:        oid,
		name:       name,
		tableOid:   tableOid,
		columnOid:  columnOid,
		keyTypeOid: keyTypeOid,
		indexType:  index.IndexTypeFromOrdinal(int(indexTypeOrd)),
		fileNode:   fileNode,
		metaPageId: metaPageId,
		rootPageId: rootPageId,
	}, nil
}
