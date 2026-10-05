package index

import (
	"sort"

	"mydb-go/internal/storage"
)

type PersistentBTree struct {
	store      *NodeStore
	rootPageID storage.PageID
}

func NewPersistentBTree(pager *storage.Pager) (*PersistentBTree, error) {
	store := NewNodeStore(pager)

	root := &Node{
		Type:   LeafNode,
		Keys:   []Key{},
		Values: []storage.RecordID{},
	}

	rootPageID, err := store.CreateNode(root)
	if err != nil {
		return nil, err
	}

	return &PersistentBTree{
		store:      store,
		rootPageID: rootPageID,
	}, nil
}

func OpenPersistentBTree(
	pager *storage.Pager,
	rootPageID storage.PageID,
) (*PersistentBTree, error) {
	store := NewNodeStore(pager)

	_, err := store.ReadNode(rootPageID)
	if err != nil {
		return nil, err
	}

	return &PersistentBTree{
		store:      store,
		rootPageID: rootPageID,
	}, nil
}

func (t *PersistentBTree) RootPageID() storage.PageID {
	return t.rootPageID
}

func (t *PersistentBTree) Root() (*Node, error) {
	return t.store.ReadNode(t.rootPageID)
}

func (t *PersistentBTree) Insert(
	key Key,
	rid storage.RecordID,
) error {
	root, err := t.Root()
	if err != nil {
		return err
	}

	split, err := t.insert(root, key, rid)
	if err != nil {
		return err
	}

	if split == nil {
		return nil
	}

	newRoot := &Node{
		Type:           InternalNode,
		Keys:           []Key{split.separator},
		ChildrenPageID: []storage.PageID{root.PageID, split.rightPageID},
	}

	newRootPageID, err := t.store.CreateNode(newRoot)
	if err != nil {
		return err
	}

	t.rootPageID = newRootPageID

	return nil
}

type splitResult struct {
	separator   Key
	rightPageID storage.PageID
}

func (t *PersistentBTree) insert(
	node *Node,
	key Key,
	rid storage.RecordID,
) (*splitResult, error) {
	if node.Type == LeafNode {
		insertIntoLeaf(node, key, rid)

		if len(node.Keys) <= MaxKeys {
			return nil, t.store.WriteNode(node)
		}

		return t.splitLeaf(node)
	}

	pos := sort.Search(len(node.Keys), func(i int) bool {
		return key < node.Keys[i]
	})

	childPageID := node.ChildrenPageID[pos]

	child, err := t.store.ReadNode(childPageID)
	if err != nil {
		return nil, err
	}

	split, err := t.insert(child, key, rid)
	if err != nil {
		return nil, err
	}

	if split == nil {
		return nil, nil
	}

	node.Keys = append(node.Keys, 0)
	copy(
		node.Keys[pos+1:],
		node.Keys[pos:],
	)
	node.Keys[pos] = split.separator

	node.ChildrenPageID = append(node.ChildrenPageID, 0)
	copy(
		node.ChildrenPageID[pos+2:],
		node.ChildrenPageID[pos+1:],
	)
	node.ChildrenPageID[pos+1] = split.rightPageID

	if len(node.Keys) <= MaxKeys {
		return nil, t.store.WriteNode(node)
	}

	return t.splitInternal(node)
}

func insertIntoLeaf(
	node *Node,
	key Key,
	rid storage.RecordID,
) {
	pos := sort.Search(len(node.Keys), func(i int) bool {
		return node.Keys[i] >= key
	})

	node.Keys = append(node.Keys, 0)
	copy(node.Keys[pos+1:], node.Keys[pos:])
	node.Keys[pos] = key

	node.Values = append(node.Values, storage.RecordID{})
	copy(node.Values[pos+1:], node.Values[pos:])
	node.Values[pos] = rid
}

func (t *PersistentBTree) splitLeaf(
	node *Node,
) (*splitResult, error) {
	mid := len(node.Keys) / 2

	right := &Node{
		Type:     LeafNode,
		Keys:     append([]Key{}, node.Keys[mid:]...),
		Values:   append([]storage.RecordID{}, node.Values[mid:]...),
		NextLeaf: node.NextLeaf,
	}

	node.Keys = append([]Key{}, node.Keys[:mid]...)
	node.Values = append([]storage.RecordID{}, node.Values[:mid]...)

	rightPageID, err := t.store.CreateNode(right)
	if err != nil {
		return nil, err
	}

	node.NextLeaf = rightPageID

	if err := t.store.WriteNode(node); err != nil {
		return nil, err
	}

	return &splitResult{
		separator:   right.Keys[0],
		rightPageID: rightPageID,
	}, nil
}

func (t *PersistentBTree) splitInternal(
	node *Node,
) (*splitResult, error) {
	mid := len(node.Keys) / 2
	separator := node.Keys[mid]

	right := &Node{
		Type:           InternalNode,
		Keys:           append([]Key{}, node.Keys[mid+1:]...),
		ChildrenPageID: append([]storage.PageID{}, node.ChildrenPageID[mid+1:]...),
	}

	node.Keys = append([]Key{}, node.Keys[:mid]...)
	node.ChildrenPageID = append(
		[]storage.PageID{},
		node.ChildrenPageID[:mid+1]...,
	)

	rightPageID, err := t.store.CreateNode(right)
	if err != nil {
		return nil, err
	}

	if err := t.store.WriteNode(node); err != nil {
		return nil, err
	}

	return &splitResult{
		separator:   separator,
		rightPageID: rightPageID,
	}, nil
}

func (t *PersistentBTree) Search(
	key Key,
) (storage.RecordID, bool, error) {
	node, err := t.Root()
	if err != nil {
		return storage.RecordID{}, false, err
	}

	for node.Type == InternalNode {
		pos := sort.Search(len(node.Keys), func(i int) bool {
			return key < node.Keys[i]
		})

		node, err = t.store.ReadNode(node.ChildrenPageID[pos])
		if err != nil {
			return storage.RecordID{}, false, err
		}
	}

	pos := sort.Search(len(node.Keys), func(i int) bool {
		return node.Keys[i] >= key
	})

	if pos < len(node.Keys) && node.Keys[pos] == key {
		return node.Values[pos], true, nil
	}

	return storage.RecordID{}, false, nil
}

func (t *PersistentBTree) RangeScan(
	startKey Key,
	endKey Key,
) ([]storage.RecordID, error) {
	if startKey > endKey {
		return []storage.RecordID{}, nil
	}

	node, err := t.Root()
	if err != nil {
		return nil, err
	}

	for node.Type == InternalNode {
		pos := sort.Search(len(node.Keys), func(i int) bool {
			return startKey < node.Keys[i]
		})

		node, err = t.store.ReadNode(node.ChildrenPageID[pos])
		if err != nil {
			return nil, err
		}
	}

	pos := sort.Search(len(node.Keys), func(i int) bool {
		return node.Keys[i] >= startKey
	})

	var results []storage.RecordID

	for {
		for i := pos; i < len(node.Keys); i++ {
			if node.Keys[i] > endKey {
				return results, nil
			}

			results = append(results, node.Values[i])
		}

		if node.NextLeaf == 0 {
			break
		}

		node, err = t.store.ReadNode(node.NextLeaf)
		if err != nil {
			return nil, err
		}

		pos = 0
	}

	return results, nil
}
