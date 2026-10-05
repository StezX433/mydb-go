package index

import (
	"fmt"

	"mydb-go/internal/storage"
)

type NodeStore struct {
	pager *storage.Pager
}

func NewNodeStore(pager *storage.Pager) *NodeStore {
	return &NodeStore{
		pager: pager,
	}
}

func (s *NodeStore) CreateNode(node *Node) (storage.PageID, error) {
	page, err := s.pager.AllocatePage()
	if err != nil {
		return 0, err
	}

	node.PageID = page.ID

	if err := s.WriteNode(node); err != nil {
		return 0, err
	}

	return page.ID, nil
}

func (s *NodeStore) WriteNode(node *Node) error {
	page := SerializeNode(node)

	if page == nil {
		return fmt.Errorf("failed to serialize node")
	}

	return s.pager.WritePage(page)
}

func (s *NodeStore) ReadNode(id storage.PageID) (*Node, error) {
	page, err := s.pager.ReadPage(id)
	if err != nil {
		return nil, err
	}

	node, err := DeserializeNode(page)
	if err != nil {
		return nil, err
	}

	node.PageID = id

	return node, nil
}