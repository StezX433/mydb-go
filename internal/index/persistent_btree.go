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

	if root.Type == LeafNode {
		insertIntoLeaf(root, key, rid)

		if len(root.Keys) <= MaxKeys {
			return t.store.WriteNode(root)
		}

		return t.splitRootLeaf(root)
	}

	return t.insertIntoInternal(root, key, rid)
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

func (t *PersistentBTree) splitRootLeaf(root *Node) error {
	mid := len(root.Keys) / 2

	right := &Node{
		Type:     LeafNode,
		Keys:     append([]Key{}, root.Keys[mid:]...),
		Values:   append([]storage.RecordID{}, root.Values[mid:]...),
		NextLeaf: root.NextLeaf,
	}

	root.Keys = append([]Key{}, root.Keys[:mid]...)
	root.Values = append([]storage.RecordID{}, root.Values[:mid]...)

	rightPageID, err := t.store.CreateNode(right)
	if err != nil {
		return err
	}

	root.NextLeaf = rightPageID

	if err := t.store.WriteNode(root); err != nil {
		return err
	}

	newRoot := &Node{
		Type:           InternalNode,
		Keys:           []Key{right.Keys[0]},
		ChildrenPageID: []storage.PageID{root.PageID, rightPageID},
	}

	newRootPageID, err := t.store.CreateNode(newRoot)
	if err != nil {
		return err
	}

	t.rootPageID = newRootPageID

	return nil
}

func (t *PersistentBTree) insertIntoInternal(
	node *Node,
	key Key,
	rid storage.RecordID,
) error {
	pos := sort.Search(len(node.Keys), func(i int) bool {
		return key < node.Keys[i]
	})

	childPageID := node.ChildrenPageID[pos]

	child, err := t.store.ReadNode(childPageID)
	if err != nil {
		return err
	}

	if child.Type == LeafNode {
		insertIntoLeaf(child, key, rid)

		if len(child.Keys) <= MaxKeys {
			return t.store.WriteNode(child)
		}

		return t.splitLeafChild(node, pos, child)
	}

	return nil
}

func (t *PersistentBTree) splitLeafChild(
	parent *Node,
	childIndex int,
	child *Node,
) error {
	mid := len(child.Keys) / 2

	right := &Node{
		Type:     LeafNode,
		Keys:     append([]Key{}, child.Keys[mid:]...),
		Values:   append([]storage.RecordID{}, child.Values[mid:]...),
		NextLeaf: child.NextLeaf,
	}

	child.Keys = append([]Key{}, child.Keys[:mid]...)
	child.Values = append([]storage.RecordID{}, child.Values[:mid]...)

	rightPageID, err := t.store.CreateNode(right)
	if err != nil {
		return err
	}

	child.NextLeaf = rightPageID

	if err := t.store.WriteNode(child); err != nil {
		return err
	}

	separator := right.Keys[0]

	parent.Keys = append(parent.Keys, 0)
	copy(
		parent.Keys[childIndex+1:],
		parent.Keys[childIndex:],
	)
	parent.Keys[childIndex] = separator

	parent.ChildrenPageID = append(parent.ChildrenPageID, 0)
	copy(
		parent.ChildrenPageID[childIndex+2:],
		parent.ChildrenPageID[childIndex+1:],
	)
	parent.ChildrenPageID[childIndex+1] = rightPageID

	return t.store.WriteNode(parent)
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
