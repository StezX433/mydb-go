package database

import (
	"fmt"

	"mydb-go/internal/buffer"
	"mydb-go/internal/record"
	"mydb-go/internal/storage"
)

type Table struct {
	pager *storage.Pager
	pool  *buffer.BufferPool
}

func NewTable(filename string) (*Table, error) {
	pager, err := storage.NewPager(filename)
	if err != nil {
		return nil, err
	}

	pool := buffer.NewBufferPool(pager, 3)

	return &Table{
		pager: pager,
		pool:  pool,
	}, nil
}

func (t *Table) Close() error {
	if err := t.pool.FlushAll(); err != nil {
		return err
	}

	return t.pager.Close()
}

func (t *Table) Insert(r record.Record) (storage.RecordID, error) {
	numPages, err := t.pager.NumPages()
	if err != nil {
		return storage.RecordID{}, err
	}

	for pageID := uint32(0); pageID < numPages; pageID++ {
		id := storage.PageID(pageID)

		page, err := t.pool.GetPage(id)
		if err != nil {
			return storage.RecordID{}, err
		}

		rid, err := page.InsertRecord(r)

		if err == nil {
			if err := t.pool.MarkDirty(id); err != nil {
				t.pool.UnpinPage(id)
				return storage.RecordID{}, err
			}

			if err := t.pool.UnpinPage(id); err != nil {
				return storage.RecordID{}, err
			}

			return rid, nil
		}

		if err := t.pool.UnpinPage(id); err != nil {
			return storage.RecordID{}, err
		}
	}

	page, err := t.pager.AllocatePage()
	if err != nil {
		return storage.RecordID{}, err
	}

	rid, err := page.InsertRecord(r)
	if err != nil {
		return storage.RecordID{}, err
	}

	if err := t.pager.WritePage(page); err != nil {
		return storage.RecordID{}, err
	}

	return rid, nil
}

func (t *Table) Get(rid storage.RecordID) (record.Record, error) {
	page, err := t.pool.GetPage(rid.PageID)
	if err != nil {
		return record.Record{}, err
	}

	defer t.pool.UnpinPage(rid.PageID)

	return page.GetRecord(rid.SlotID)
}

func (t *Table) PageCount() (uint32, error) {
	return t.pager.NumPages()
}

func (t *Table) String() string {
	count, err := t.PageCount()
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	return fmt.Sprintf("pages: %d", count)
}