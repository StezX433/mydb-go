package storage

import (
	"fmt"
	"os"
)

type Pager struct {
	file *os.File
}

func NewPager(filename string) (*Pager, error) {
	file, err := os.OpenFile(
		filename,
		os.O_RDWR|os.O_CREATE,
		0644,
	)

	if err != nil {
		return nil, err
	}

	return &Pager{
		file: file,
	}, nil
}

func (p *Pager) ReadPage(id PageID) (*Page, error) {
	page := &Page{
		ID: id,
	}

	offset := int64(id) * int64(PageSize)

	n, err := p.file.ReadAt(page.Data[:], offset)
	if err != nil {
		return nil, err
	}

	if n != PageSize {
		return nil, fmt.Errorf(
			"could not read complete page: got %d bytes, expected %d",
			n,
			PageSize,
		)
	}

	return page, nil
}

func (p *Pager) WritePage(page *Page) error {
	offset := int64(page.ID) * int64(PageSize)

	n, err := p.file.WriteAt(page.Data[:], offset)
	if err != nil {
		return err
	}

	if n != PageSize {
		return fmt.Errorf(
			"could not write complete page: wrote %d bytes, expected %d",
			n,
			PageSize,
		)
	}

	return nil
}

func (p *Pager) NumPages() (uint32, error) {
	info, err := p.file.Stat()
	if err != nil {
		return 0, err
	}

	fileSize := info.Size()

	if fileSize%int64(PageSize) != 0 {
		return 0, fmt.Errorf(
			"database file is corrupted: size %d is not a multiple of page size %d",
			fileSize,
			PageSize,
		)
	}

	return uint32(fileSize / int64(PageSize)), nil
}

func (p *Pager) AllocatePage() (*Page, error) {
	numPages, err := p.NumPages()
	if err != nil {
		return nil, err
	}

	return NewPage(PageID(numPages)), nil
}

func (p *Pager) Close() error {
	return p.file.Close()
}
