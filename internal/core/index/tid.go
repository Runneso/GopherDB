package index

const TIDSize = 6

type TID struct {
	pageID int32
	slotID int16
}

func NewTID(pageID int32, slotID int16) TID {
	return TID{
		pageID: pageID,
		slotID: slotID,
	}
}

func (tid TID) PageID() int32 {
	return tid.pageID
}

func (tid TID) SlotID() int16 {
	return tid.slotID
}
