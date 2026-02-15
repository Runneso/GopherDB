package model

type PageKey struct {
	fileID string
	pageID int
}

func NewPageKey(fileID string, pageID int) PageKey {
	return PageKey{fileID: fileID, pageID: pageID}
}

func (key *PageKey) FileID() string {
	return key.fileID
}

func (key *PageKey) PageID() int {
	return key.pageID
}
