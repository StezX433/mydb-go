package storage

import (
	"encoding/binary"
	"fmt"

	"mydb-go/internal/record"
)

const (
	PageSize   = 4096
	HeaderSize = 8
)

type PageID uint32

type PageHeader struct {
	RecordCount uint32
	FreeSpace   uint32
}

type Page struct {
	ID   PageID
	Data [PageSize]byte
}

func (p *Page) SetHeader(header PageHeader) {
	binary.LittleEndian.PutUint32(
		p.Data[0:4],
		header.RecordCount,
	)

	binary.LittleEndian.PutUint32(
		p.Data[4:8],
		header.FreeSpace,
	)
}

func (p *Page) GetHeader() PageHeader {
	return PageHeader{
		RecordCount: binary.LittleEndian.Uint32(p.Data[0:4]),
		FreeSpace:   binary.LittleEndian.Uint32(p.Data[4:8]),
	}
}

func NewPage(id PageID) *Page {
	page := &Page{
		ID: id,
	}

	header := PageHeader{
		RecordCount: 0,
		FreeSpace:   PageSize - HeaderSize,
	}

	page.SetHeader(header)

	return page
}

func (p *Page) InsertRecord(r record.Record) error {
	header := p.GetHeader()

	offset := HeaderSize + int(header.RecordCount)*record.RecordSize

	if offset+record.RecordSize > PageSize {
		return fmt.Errorf("page is full")
	}

	data := r.Serialize()

	copy(
		p.Data[offset:offset+record.RecordSize],
		data[:],
	)

	header.RecordCount++
	header.FreeSpace = uint32(PageSize - (HeaderSize + int(header.RecordCount)*record.RecordSize))

	p.SetHeader(header)

	return nil
}
func (p *Page) GetRecord(index uint32) (record.Record, error) {
	header := p.GetHeader()

	if index >= header.RecordCount {
		return record.Record{}, fmt.Errorf("record index out of range")
	}

	offset := HeaderSize + int(index)*record.RecordSize

	var data [record.RecordSize]byte

	copy(
		data[:],
		p.Data[offset:offset+record.RecordSize],
	)

	return record.Deserialize(data), nil
}

