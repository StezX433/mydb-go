package index

import "mydb-go/internal/storage"

type Key uint32

type NodeType uint8

const (
	LeafNode NodeType = iota
	InternalNode
)

const (
	MaxKeys = 4
	MinKeys = 2
)

type Node struct {
	Type NodeType

	Keys   []Key
	Values []storage.RecordID

	Children       []*Node
	ChildrenPageID []storage.PageID

	PageID   storage.PageID
	NextLeaf storage.PageID
}
