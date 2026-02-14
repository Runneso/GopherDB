package model

type PageKey struct {
	fileID string
	pageID int
}

func (key *PageKey) FileID() string {
	return key.fileID
}
func (key *PageKey) PageID() int {
	return key.pageID
}
