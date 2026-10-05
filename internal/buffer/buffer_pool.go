package buffer

import (
	"container/list"
	"fmt"

	"mydb-go/internal/storage"
)

type frame struct {
	page     *storage.Page
	dirty    bool
	pinCount int
}

type BufferPool struct {
	pager    *storage.Pager
	capacity int
	pages    map[storage.PageID]*list.Element
	lru      *list.List
}

func NewBufferPool(pager *storage.Pager, capacity int) *BufferPool {
	return &BufferPool{
		pager:    pager,
		capacity: capacity,
		pages:    make(map[storage.PageID]*list.Element),
		lru:      list.New(),
	}

}


func (b *BufferPool) GetPage(id storage.PageID) (*storage.Page, error) {
	if element, ok := b.pages[id]; ok {
		b.lru.MoveToFront(element)

		frame := element.Value.(*frame)
		frame.pinCount++

		return frame.page, nil
	}

	if len(b.pages) >= b.capacity {
		if err := b.evict(); err != nil {
			return nil, err
		}
	}

	page, err := b.pager.ReadPage(id)
	if err != nil {
		return nil, err
	}

	frame := &frame{
		page:     page,
		dirty:    false,
		pinCount: 1,
	}

	element := b.lru.PushFront(frame)
	b.pages[id] = element

	return page, nil
}

func (b *BufferPool) UnpinPage(id storage.PageID) error {
	element, ok := b.pages[id]
	if !ok {
		return fmt.Errorf("page %d is not in buffer pool", id)
	}

	frame := element.Value.(*frame)

	if frame.pinCount == 0 {
		return fmt.Errorf("page %d is already unpinned", id)
	}

	frame.pinCount--

	return nil
}



func (b *BufferPool) evict() error {
	for element := b.lru.Back(); element != nil; element = element.Prev() {
		frame := element.Value.(*frame)

		if frame.pinCount > 0 {
			continue
		}

		if frame.dirty {
			if err := b.pager.WritePage(frame.page); err != nil {
				return err
			}
		}

		delete(b.pages, frame.page.ID)
		b.lru.Remove(element)

		return nil
	}

	return fmt.Errorf("no unpinned pages available for eviction")
}








func (b *BufferPool) MarkDirty(id storage.PageID) error {
	element, ok := b.pages[id]

	if !ok {
		return fmt.Errorf("page %d is not in buffer pool", id)
	}

	frame := element.Value.(*frame)
	frame.dirty = true

	return nil
}



func (b *BufferPool) FlushPage(id storage.PageID) error {
	element, ok := b.pages[id]

	if !ok {
		return fmt.Errorf("page %d is not in buffer pool", id)
	}

	frame := element.Value.(*frame)

	if err := b.pager.WritePage(frame.page); err != nil {
		return err
	}

	frame.dirty = false

	return nil
}



func (b *BufferPool) FlushAll() error {
	for id := range b.pages {
		if err := b.FlushPage(id); err != nil {
			return err
		}
	}

	return nil
}

func (b *BufferPool) Size() int {
	return len(b.pages)
}
