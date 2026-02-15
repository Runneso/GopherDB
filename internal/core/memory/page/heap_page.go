package page

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	ErrNilBuffer        = errors.New("nil buffer")
	ErrBadBufferSize    = errors.New("bad buffer size")
	ErrInvalidSignature = errors.New("invalid page signature")
	ErrSlotIndex        = errors.New("slot index out of range")
	ErrCorruptedPage    = errors.New("corrupted page")
	ErrNilData          = errors.New("nil data")
	ErrNoSpace          = errors.New("not enough space")
)

const (
	PageSize  = 8 * 1024
	Signature = 0x00DBDB01

	headerSize = 10
	slotSize   = 4

	signatureOff  = 0
	slotCountOff  = 4
	lowerBoundOff = 6
	upperBoundOff = 8

	shortSize = 2
)

type HeapPage struct {
	pageID int
	buf    []byte
}

func NewHeapPage(pageID int) *HeapPage {
	p := &HeapPage{
		pageID: pageID,
		buf:    make([]byte, PageSize),
	}
	p.initHeaders()
	return p
}

func NewHeapPageFromBuffer(pageID int, buffer []byte) (*HeapPage, error) {
	if buffer == nil {
		return nil, ErrNilBuffer
	}
	if len(buffer) != PageSize {
		return nil, fmt.Errorf("%w: want=%d got=%d", ErrBadBufferSize, PageSize, len(buffer))
	}
	p := &HeapPage{pageID: pageID, buf: buffer}
	if !p.IsValid() {
		return nil, ErrInvalidSignature
	}
	return p, nil
}

func (heapPage *HeapPage) Bytes() []byte {
	return heapPage.buf
}
func (heapPage *HeapPage) PageID() int {
	return heapPage.pageID
}

func (heapPage *HeapPage) Size() int {
	return int(heapPage.readU16(slotCountOff))
}

func (heapPage *HeapPage) IsValid() bool {
	return heapPage.readU32(signatureOff) == Signature
}

func (heapPage *HeapPage) Read(index int) ([]byte, error) {
	if err := heapPage.verifySignature(); err != nil {
		return nil, err
	}

	slotCount := int(heapPage.readU16(slotCountOff))
	if index < 0 || index >= slotCount {
		return nil, fmt.Errorf("%w: index=%d slotCount=%d", ErrSlotIndex, index, slotCount)
	}

	slotPos := headerSize + index*slotSize
	offset := int(heapPage.readU16(slotPos))
	length := int(heapPage.readU16(slotPos + shortSize))

	upper := int(heapPage.readU16(upperBoundOff))
	if offset+length > PageSize || offset < upper {
		return nil, fmt.Errorf("%w: offset=%d length=%d upperBound=%d pageSize=%d",
			ErrCorruptedPage, offset, length, upper, PageSize,
		)
	}

	out := make([]byte, length)
	copy(out, heapPage.buf[offset:offset+length])
	return out, nil
}

func (heapPage *HeapPage) Write(data []byte) error {
	if err := heapPage.verifySignature(); err != nil {
		return err
	}
	if data == nil {
		return ErrNilData
	}

	length := len(data)
	slotCount := int(heapPage.readU16(slotCountOff))
	lower := int(heapPage.readU16(lowerBoundOff))
	upper := int(heapPage.readU16(upperBoundOff))

	required := slotSize + length
	freeSpace := upper - lower
	if required > freeSpace {
		return fmt.Errorf("%w: required=%d (data=%d + slot=%d) free=%d (gap=%d-%d)",
			ErrNoSpace, required, length, slotSize, freeSpace, upper, lower,
		)
	}

	newUpper := upper - length
	copy(heapPage.buf[newUpper:newUpper+length], data)

	slotPos := headerSize + slotCount*slotSize
	heapPage.writeU16(slotPos, uint16(newUpper))
	heapPage.writeU16(slotPos+shortSize, uint16(length))

	heapPage.writeU16(slotCountOff, uint16(slotCount+1))
	heapPage.writeU16(lowerBoundOff, uint16(lower+slotSize))
	heapPage.writeU16(upperBoundOff, uint16(newUpper))
	return nil
}

func (heapPage *HeapPage) verifySignature() error {
	if !heapPage.IsValid() {
		return ErrInvalidSignature
	}
	return nil
}

func (heapPage *HeapPage) initHeaders() {
	heapPage.writeU32(signatureOff, Signature)
	heapPage.writeU16(slotCountOff, 0)
	heapPage.writeU16(lowerBoundOff, headerSize)
	heapPage.writeU16(upperBoundOff, PageSize)
}

func (heapPage *HeapPage) readU32(pos int) uint32 {
	return binary.BigEndian.Uint32(heapPage.buf[pos : pos+4])
}

func (heapPage *HeapPage) readU16(pos int) uint16 {
	return binary.BigEndian.Uint16(heapPage.buf[pos : pos+2])
}

func (heapPage *HeapPage) writeU32(pos int, v uint32) {
	binary.BigEndian.PutUint32(heapPage.buf[pos:pos+4], v)
}

func (heapPage *HeapPage) writeU16(pos int, v uint16) {
	binary.BigEndian.PutUint16(heapPage.buf[pos:pos+2], v)
}
