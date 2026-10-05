package index

import (
	"fmt"
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

func (t *PersistentBTree) Delete(key Key) error {
	root, err := t.Root()
	if err != nil {
		return err
	}

	if root.Type == LeafNode {
		deleted, err := deleteFromLeaf(root, key)
		if err != nil {
			return err
		}

		if !deleted {
			return nil
		}

		return t.store.WriteNode(root)
	}

	deleted, _, err := t.deleteNode(root, key, true)
	if err != nil {
		return err
	}

	if !deleted {
		return nil
	}

	root, err = t.Root()
	if err != nil {
		return err
	}

	if root.Type == InternalNode && len(root.ChildrenPageID) == 1 {
		t.rootPageID = root.ChildrenPageID[0]
	}

	return nil
}

func (t *PersistentBTree) deleteNode(
	node *Node,
	key Key,
	isRoot bool,
) (bool, bool, error) {
	if node.Type == LeafNode {
		deleted, err := deleteFromLeaf(node, key)
		if err != nil {
			return false, false, err
		}

		if !deleted {
			return false, false, nil
		}

		if err := t.store.WriteNode(node); err != nil {
			return false, false, err
		}

		underflow := !isRoot && len(node.Keys) < MinKeys

		return true, underflow, nil
	}

	pos := sort.Search(len(node.Keys), func(i int) bool {
		return key < node.Keys[i]
	})

	child, err := t.store.ReadNode(node.ChildrenPageID[pos])
	if err != nil {
		return false, false, err
	}

	deleted, underflow, err := t.deleteNode(
		child,
		key,
		false,
	)
	if err != nil {
		return false, false, err
	}

	if !deleted {
		return false, false, nil
	}

	if underflow {
		if err := t.rebalanceChild(node, pos); err != nil {
			return false, false, err
		}
	}

	if err := t.refreshSeparators(node); err != nil {
		return false, false, err
	}

	if err := t.store.WriteNode(node); err != nil {
		return false, false, err
	}

	parentUnderflow := !isRoot && len(node.Keys) < MinKeys

	return true, parentUnderflow, nil
}

func deleteFromLeaf(
	node *Node,
	key Key,
) (bool, error) {
	pos := sort.Search(len(node.Keys), func(i int) bool {
		return node.Keys[i] >= key
	})

	if pos >= len(node.Keys) || node.Keys[pos] != key {
		return false, nil
	}

	node.Keys = append(
		node.Keys[:pos],
		node.Keys[pos+1:]...,
	)

	node.Values = append(
		node.Values[:pos],
		node.Values[pos+1:]...,
	)

	return true, nil
}

func (t *PersistentBTree) rebalanceChild(
	parent *Node,
	childIndex int,
) error {
	child, err := t.store.ReadNode(
		parent.ChildrenPageID[childIndex],
	)
	if err != nil {
		return err
	}

	if child.Type == LeafNode {
		return t.rebalanceLeaf(parent, childIndex)
	}

	return t.rebalanceInternal(parent, childIndex)
}

func (t *PersistentBTree) rebalanceLeaf(
	parent *Node,
	childIndex int,
) error {
	child, err := t.store.ReadNode(
		parent.ChildrenPageID[childIndex],
	)
	if err != nil {
		return err
	}

	if childIndex > 0 {
		left, err := t.store.ReadNode(
			parent.ChildrenPageID[childIndex-1],
		)
		if err != nil {
			return err
		}

		if len(left.Keys) > MinKeys {
			key := left.Keys[len(left.Keys)-1]
			value := left.Values[len(left.Values)-1]

			left.Keys = left.Keys[:len(left.Keys)-1]
			left.Values = left.Values[:len(left.Values)-1]

			child.Keys = append(child.Keys, 0)
			copy(child.Keys[1:], child.Keys[:len(child.Keys)-1])
			child.Keys[0] = key

			child.Values = append(child.Values, storage.RecordID{})
			copy(child.Values[1:], child.Values[:len(child.Values)-1])
			child.Values[0] = value

			if err := t.store.WriteNode(left); err != nil {
				return err
			}

			return t.store.WriteNode(child)
		}
	}

	if childIndex+1 < len(parent.ChildrenPageID) {
		right, err := t.store.ReadNode(
			parent.ChildrenPageID[childIndex+1],
		)
		if err != nil {
			return err
		}

		if len(right.Keys) > MinKeys {
			key := right.Keys[0]
			value := right.Values[0]

			right.Keys = right.Keys[1:]
			right.Values = right.Values[1:]

			child.Keys = append(child.Keys, key)
			child.Values = append(child.Values, value)

			if err := t.store.WriteNode(right); err != nil {
				return err
			}

			return t.store.WriteNode(child)
		}
	}

	if childIndex > 0 {
		left, err := t.store.ReadNode(
			parent.ChildrenPageID[childIndex-1],
		)
		if err != nil {
			return err
		}

		left.Keys = append(left.Keys, child.Keys...)
		left.Values = append(left.Values, child.Values...)
		left.NextLeaf = child.NextLeaf

		parent.ChildrenPageID = append(
			parent.ChildrenPageID[:childIndex],
			parent.ChildrenPageID[childIndex+1:]...,
		)

		parent.Keys = append(
			parent.Keys[:childIndex-1],
			parent.Keys[childIndex:]...,
		)

		return t.store.WriteNode(left)
	}

	right, err := t.store.ReadNode(
		parent.ChildrenPageID[childIndex+1],
	)
	if err != nil {
		return err
	}

	child.Keys = append(child.Keys, right.Keys...)
	child.Values = append(child.Values, right.Values...)
	child.NextLeaf = right.NextLeaf

	parent.ChildrenPageID = append(
		parent.ChildrenPageID[:childIndex+1],
		parent.ChildrenPageID[childIndex+2:]...,
	)

	parent.Keys = append(
		parent.Keys[:childIndex],
		parent.Keys[childIndex+1:]...,
	)

	return t.store.WriteNode(child)
}

func (t *PersistentBTree) rebalanceInternal(
	parent *Node,
	childIndex int,
) error {
	child, err := t.store.ReadNode(
		parent.ChildrenPageID[childIndex],
	)
	if err != nil {
		return err
	}

	if childIndex > 0 {
		left, err := t.store.ReadNode(
			parent.ChildrenPageID[childIndex-1],
		)
		if err != nil {
			return err
		}

		if len(left.Keys) > MinKeys {
			movedPageID := left.ChildrenPageID[len(left.ChildrenPageID)-1]

			movedNode, err := t.store.ReadNode(movedPageID)
			if err != nil {
				return err
			}

			movedFirstKey, err := t.firstKey(movedNode)
			if err != nil {
				return err
			}

			left.ChildrenPageID = left.ChildrenPageID[:len(left.ChildrenPageID)-1]

			left.Keys = left.Keys[:len(left.Keys)-1]

			child.ChildrenPageID = append(
				child.ChildrenPageID,
				0,
			)

			copy(
				child.ChildrenPageID[1:],
				child.ChildrenPageID[:len(child.ChildrenPageID)-1],
			)

			child.ChildrenPageID[0] = movedPageID

			child.Keys = append(child.Keys, 0)

			copy(
				child.Keys[1:],
				child.Keys[:len(child.Keys)-1],
			)

			child.Keys[0] = movedFirstKey

			if err := t.store.WriteNode(left); err != nil {
				return err
			}

			return t.store.WriteNode(child)
		}
	}

	if childIndex+1 < len(parent.ChildrenPageID) {
		right, err := t.store.ReadNode(
			parent.ChildrenPageID[childIndex+1],
		)
		if err != nil {
			return err
		}

		if len(right.Keys) > MinKeys {
			movedPageID := right.ChildrenPageID[0]

			movedNode, err := t.store.ReadNode(movedPageID)
			if err != nil {
				return err
			}

			movedFirstKey, err := t.firstKey(movedNode)
			if err != nil {
				return err
			}

			right.ChildrenPageID = right.ChildrenPageID[1:]
			right.Keys = right.Keys[1:]

			child.ChildrenPageID = append(
				child.ChildrenPageID,
				movedPageID,
			)

			child.Keys = append(
				child.Keys,
				movedFirstKey,
			)

			if err := t.store.WriteNode(right); err != nil {
				return err
			}

			return t.store.WriteNode(child)
		}
	}

	if childIndex > 0 {
		left, err := t.store.ReadNode(
			parent.ChildrenPageID[childIndex-1],
		)
		if err != nil {
			return err
		}

		childFirstKey, err := t.firstKey(child)
		if err != nil {
			return err
		}

		left.Keys = append(left.Keys, childFirstKey)
		left.Keys = append(left.Keys, child.Keys...)
		left.ChildrenPageID = append(
			left.ChildrenPageID,
			child.ChildrenPageID...,
		)

		parent.ChildrenPageID = append(
			parent.ChildrenPageID[:childIndex],
			parent.ChildrenPageID[childIndex+1:]...,
		)

		parent.Keys = append(
			parent.Keys[:childIndex-1],
			parent.Keys[childIndex:]...,
		)

		return t.store.WriteNode(left)
	}

	right, err := t.store.ReadNode(
		parent.ChildrenPageID[childIndex+1],
	)
	if err != nil {
		return err
	}

	rightFirstKey, err := t.firstKey(right)
	if err != nil {
		return err
	}

	child.Keys = append(child.Keys, rightFirstKey)
	child.Keys = append(child.Keys, right.Keys...)
	child.ChildrenPageID = append(
		child.ChildrenPageID,
		right.ChildrenPageID...,
	)

	parent.ChildrenPageID = append(
		parent.ChildrenPageID[:childIndex+1],
		parent.ChildrenPageID[childIndex+2:]...,
	)

	parent.Keys = append(
		parent.Keys[:childIndex],
		parent.Keys[childIndex+1:]...,
	)

	return t.store.WriteNode(child)
}

func (t *PersistentBTree) refreshSeparators(
	node *Node,
) error {
	if node.Type != InternalNode {
		return nil
	}

	node.Keys = make([]Key, len(node.ChildrenPageID)-1)

	for i := 1; i < len(node.ChildrenPageID); i++ {
		child, err := t.store.ReadNode(
			node.ChildrenPageID[i],
		)
		if err != nil {
			return err
		}

		firstKey, err := t.firstKey(child)
		if err != nil {
			return err
		}

		node.Keys[i-1] = firstKey
	}

	return nil
}

func (t *PersistentBTree) firstKey(
	node *Node,
) (Key, error) {
	for node.Type == InternalNode {
		if len(node.ChildrenPageID) == 0 {
			return 0, fmt.Errorf(
				"internal node %d has no children",
				node.PageID,
			)
		}

		child, err := t.store.ReadNode(
			node.ChildrenPageID[0],
		)
		if err != nil {
			return 0, err
		}

		node = child
	}

	if len(node.Keys) == 0 {
		return 0, fmt.Errorf(
			"leaf node %d has no keys",
			node.PageID,
		)
	}

	return node.Keys[0], nil
}
