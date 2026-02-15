package model

import (
	"encoding/binary"
	"errors"
)

const (
	maxUint16 = 0xFFFF
	intSize   = 4
	shortSize = 2
)

var (
	ErrNameRequired = errors.New("name is required")
	ErrNameTooLong  = errors.New("name too long")
	ErrDataTooShort = errors.New("data too short")
)

type TypeDefinition struct {
	oid        int32
	name       string
	byteLength int32
}

func NewTypeDefinition(oid int32, name string, byteLength int32) (*TypeDefinition, error) {
	if name == "" {
		return nil, ErrNameRequired
	}
	return &TypeDefinition{
		oid:        oid,
		name:       name,
		byteLength: byteLength,
	}, nil
}

func (typ *TypeDefinition) Oid() int32 {
	return typ.oid
}

func (typ *TypeDefinition) Name() string {
	return typ.name
}

func (typ *TypeDefinition) ByteLength() int32 {
	return typ.byteLength
}

func (typ *TypeDefinition) ToBytes() ([]byte, error) {
	nameBytes := []byte(typ.name)
	if len(nameBytes) > maxUint16 {
		return nil, ErrNameTooLong
	}

	buf := make([]byte, intSize+shortSize+len(nameBytes)+intSize)
	offset := 0

	binary.BigEndian.PutUint32(buf[offset:], uint32(typ.oid))
	offset += intSize

	binary.BigEndian.PutUint16(buf[offset:], uint16(len(nameBytes)))
	offset += shortSize

	copy(buf[offset:], nameBytes)
	offset += len(nameBytes)

	binary.BigEndian.PutUint32(buf[offset:], uint32(typ.byteLength))

	return buf, nil
}

func TypeDefinitionFromBytes(data []byte) (*TypeDefinition, error) {
	minSize := intSize + shortSize + intSize
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

	byteLength := int32(binary.BigEndian.Uint32(data[offset:]))

	return &TypeDefinition{
		oid:        oid,
		name:       name,
		byteLength: byteLength,
	}, nil
}
