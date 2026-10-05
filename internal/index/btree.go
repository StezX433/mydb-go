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
	t.insertIntoLeaf(t.Root, key, rid)

	if len(t.Root.Keys) > MaxKeys {
		t.splitRoot()
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
