package storage

const PageSize = 4096

type PageID uint32

type Page struct {
	ID   PageID
	Data [PageSize]byte
}
