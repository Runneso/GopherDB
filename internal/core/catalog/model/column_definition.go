package model

import (
	"encoding/binary"
)

type ColumnDefinition struct {
	oid      int32
	tableOid int32
	typeOid  int32
	name     string
	position int32
}

func NewColumnDefinition(oid, tableOid, typeOid int32, name string, position int32) (*ColumnDefinition, error) {
	if name == "" {
		return nil, ErrNameRequired
	}
	return &ColumnDefinition{
		oid:      oid,
		tableOid: tableOid,
		typeOid:  typeOid,
		name:     name,
		position: position,
	}, nil
}

func (column *ColumnDefinition) Oid() int32 {
	return column.oid
}

func (column *ColumnDefinition) TableOid() int32 {
	return column.tableOid
}

func (column *ColumnDefinition) TypeOid() int32 {
	return column.typeOid
}

func (column *ColumnDefinition) Name() string {
	return column.name
}

func (column *ColumnDefinition) Position() int32 {
	return column.position
}

func (column *ColumnDefinition) ToBytes() ([]byte, error) {
	nameBytes := []byte(column.name)
	if len(nameBytes) > maxUint16 {
		return nil, ErrNameTooLong
	}

	buf := make([]byte, intSize*3+shortSize+len(nameBytes)+intSize)
	offset := 0

	binary.BigEndian.PutUint32(buf[offset:], uint32(column.oid))
	offset += intSize

	binary.BigEndian.PutUint32(buf[offset:], uint32(column.tableOid))
	offset += intSize

	binary.BigEndian.PutUint32(buf[offset:], uint32(column.typeOid))
	offset += intSize

	binary.BigEndian.PutUint16(buf[offset:], uint16(len(nameBytes)))
	offset += shortSize

	copy(buf[offset:], nameBytes)
	offset += len(nameBytes)

	binary.BigEndian.PutUint32(buf[offset:], uint32(column.position))

	return buf, nil
}

func ColumnDefinitionFromBytes(data []byte) (*ColumnDefinition, error) {
	minSize := intSize*3 + shortSize + intSize
	if len(data) < minSize {
		return nil, ErrDataTooShort
	}

	offset := 0

	oid := int32(binary.BigEndian.Uint32(data[offset:]))
	offset += intSize

	tableOid := int32(binary.BigEndian.Uint32(data[offset:]))
	offset += intSize

	typeOid := int32(binary.BigEndian.Uint32(data[offset:]))
	offset += intSize

	nameLen := int(binary.BigEndian.Uint16(data[offset:]))
	offset += shortSize

	name := string(data[offset : offset+nameLen])
	offset += nameLen

	position := int32(binary.BigEndian.Uint32(data[offset:]))

	return &ColumnDefinition{
		oid:      oid,
		tableOid: tableOid,
		typeOid:  typeOid,
		name:     name,
		position: position,
	}, nil
}
