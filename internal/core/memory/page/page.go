package page

type Page interface {
	Bytes() []byte
	PageID() int
	Size() int
	IsValid() bool

	Read(int) ([]byte, error)

	Write([]byte) error
}
