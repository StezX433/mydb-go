package index

import "mydb-go/internal/storage"

type Key uint32

type NodeType uint8

const (
	LeafNode NodeType = iota
	InternalNode
)

const MaxKeys = 4

type Node struct {
	Type     NodeType
	Keys     []Key
	Values   []storage.RecordID
	Children []*Node
}
