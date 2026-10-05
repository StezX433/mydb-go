package index

import (
	"sort"

	"mydb-go/internal/storage"
)

type BTree struct {
	Root *Node
}

func NewBTree() *BTree {
	return &BTree{
		Root: &Node{
			Type:     LeafNode,
			Keys:     []Key{},
			Values:   []storage.RecordID{},
			Children: []*Node{},
		},
	}
}

func (t *BTree) Insert(key Key, rid storage.RecordID) {
	if t.Root.Type == LeafNode {
		t.insertIntoLeaf(t.Root, key, rid)

		if len(t.Root.Keys) > MaxKeys {
			t.splitRoot()
		}

		return
	}

	t.insertIntoInternal(t.Root, key, rid)

	if len(t.Root.Keys) > MaxKeys {
		t.splitInternalRoot()
	}
}

func (t *BTree) insertIntoLeaf(
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

func (t *BTree) insertIntoInternal(
	node *Node,
	key Key,
	rid storage.RecordID,
) {
	pos := sort.Search(len(node.Keys), func(i int) bool {
		return key < node.Keys[i]
	})

	child := node.Children[pos]

	if child.Type == LeafNode {
		t.insertIntoLeaf(child, key, rid)

		if len(child.Keys) > MaxKeys {
			t.splitChild(node, pos)
		}

		return
	}

	t.insertIntoInternal(child, key, rid)

	if len(child.Keys) > MaxKeys {
		t.splitInternalChild(node, pos)
	}
}

func (t *BTree) Search(key Key) (storage.RecordID, bool) {
	return searchNode(t.Root, key)
}

func searchNode(node *Node, key Key) (storage.RecordID, bool) {
	if node.Type == LeafNode {
		pos := sort.Search(len(node.Keys), func(i int) bool {
			return node.Keys[i] >= key
		})

		if pos < len(node.Keys) && node.Keys[pos] == key {
			return node.Values[pos], true
		}

		return storage.RecordID{}, false
	}

	pos := sort.Search(len(node.Keys), func(i int) bool {
		return key < node.Keys[i]
	})

	return searchNode(node.Children[pos], key)
}

func (t *BTree) splitRoot() {
	oldRoot := t.Root

	mid := len(oldRoot.Keys) / 2

	left := &Node{
		Type:   LeafNode,
		Keys:   append([]Key{}, oldRoot.Keys[:mid]...),
		Values: append([]storage.RecordID{}, oldRoot.Values[:mid]...),
	}

	right := &Node{
		Type:   LeafNode,
		Keys:   append([]Key{}, oldRoot.Keys[mid:]...),
		Values: append([]storage.RecordID{}, oldRoot.Values[mid:]...),
	}

	t.Root = &Node{
		Type:     InternalNode,
		Keys:     []Key{right.Keys[0]},
		Children: []*Node{left, right},
	}
}

func (t *BTree) splitChild(parent *Node, childIndex int) {
	child := parent.Children[childIndex]

	mid := len(child.Keys) / 2

	left := &Node{
		Type:   LeafNode,
		Keys:   append([]Key{}, child.Keys[:mid]...),
		Values: append([]storage.RecordID{}, child.Values[:mid]...),
	}

	right := &Node{
		Type:   LeafNode,
		Keys:   append([]Key{}, child.Keys[mid:]...),
		Values: append([]storage.RecordID{}, child.Values[mid:]...),
	}

	separator := right.Keys[0]

	parent.Keys = append(parent.Keys, 0)
	copy(
		parent.Keys[childIndex+1:],
		parent.Keys[childIndex:],
	)
	parent.Keys[childIndex] = separator

	parent.Children = append(parent.Children, nil)
	copy(
		parent.Children[childIndex+2:],
		parent.Children[childIndex+1:],
	)

	parent.Children[childIndex] = left
	parent.Children[childIndex+1] = right
}

func (t *BTree) splitInternalChild(parent *Node, childIndex int) {
	child := parent.Children[childIndex]

	mid := len(child.Keys) / 2
	separator := child.Keys[mid]

	left := &Node{
		Type:     InternalNode,
		Keys:     append([]Key{}, child.Keys[:mid]...),
		Children: append([]*Node{}, child.Children[:mid+1]...),
	}

	right := &Node{
		Type:     InternalNode,
		Keys:     append([]Key{}, child.Keys[mid+1:]...),
		Children: append([]*Node{}, child.Children[mid+1:]...),
	}

	parent.Keys = append(parent.Keys, 0)
	copy(
		parent.Keys[childIndex+1:],
		parent.Keys[childIndex:],
	)
	parent.Keys[childIndex] = separator

	parent.Children = append(parent.Children, nil)
	copy(
		parent.Children[childIndex+2:],
		parent.Children[childIndex+1:],
	)

	parent.Children[childIndex] = left
	parent.Children[childIndex+1] = right
}

func (t *BTree) splitInternalRoot() {
	oldRoot := t.Root

	mid := len(oldRoot.Keys) / 2
	separator := oldRoot.Keys[mid]

	left := &Node{
		Type:     InternalNode,
		Keys:     append([]Key{}, oldRoot.Keys[:mid]...),
		Children: append([]*Node{}, oldRoot.Children[:mid+1]...),
	}

	right := &Node{
		Type:     InternalNode,
		Keys:     append([]Key{}, oldRoot.Keys[mid+1:]...),
		Children: append([]*Node{}, oldRoot.Children[mid+1:]...),
	}

	t.Root = &Node{
		Type:     InternalNode,
		Keys:     []Key{separator},
		Children: []*Node{left, right},
	}
}
