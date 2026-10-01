package storage

import (
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

	_, err := p.file.ReadAt(page.Data[:], offset)
	if err != nil {
		return nil, err
	}

	return page, nil
}

func (p *Pager) WritePage(page *Page) error {
	offset := int64(page.ID) * int64(PageSize)

	_, err := p.file.WriteAt(page.Data[:], offset)

	return err
}

func (p *Pager) NumPages() (uint32, error) {
	info, err := p.file.Stat()
	if err != nil {
		return 0, err
	}

	fileSize := info.Size()

	return uint32(fileSize / int64(PageSize)), nil
}

func (p *Pager) AllocatePage() (*Page, error) {
	numPages, err := p.NumPages()
	if err != nil {
		return nil, err
	}

	page := &Page{
		ID: PageID(numPages),
	}

	return page, nil
}

func (p *Pager) Close() error {
	return p.file.Close()
}
