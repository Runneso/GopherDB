package model

type DataType int

type HeapTuple struct {
	data []byte
	dt   DataType
}

func NewHeapTuple(data []byte, dt DataType) HeapTuple {
	return HeapTuple{
		data: data,
		dt:   dt,
	}
}

func (key *HeapTuple) Type() DataType {
	return key.dt
}

func (key *HeapTuple) Data() []byte {
	return key.data
}
