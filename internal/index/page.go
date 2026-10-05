package index

import (
	"encoding/binary"
	"fmt"

	"mydb-go/internal/storage"
)

const (
	indexMagic uint32 = 0x42505431

	indexHeaderSize = 12
	keySize         = 4
	pageIDSize      = 4
)

func SerializeNode(node *Node) *storage.Page {
	page := storage.NewPage(node.PageID)

	data := page.Data[:]

	binary.LittleEndian.PutUint32(
		data[0:4],
		indexMagic,
	)

	data[4] = byte(node.Type)

	binary.LittleEndian.PutUint16(
		data[6:8],
		uint16(len(node.Keys)),
	)

	if node.Type == LeafNode {
		binary.LittleEndian.PutUint32(
			data[8:12],
			uint32(node.NextLeaf),
		)
	}

	offset := indexHeaderSize

	for _, key := range node.Keys {
		binary.LittleEndian.PutUint32(
			data[offset:offset+keySize],
			uint32(key),
		)

		offset += keySize
	}

	if node.Type == LeafNode {
		for _, rid := range node.Values {
			binary.LittleEndian.PutUint32(
				data[offset:offset+4],
				uint32(rid.PageID),
			)

			offset += 4

			binary.LittleEndian.PutUint16(
				data[offset:offset+2],
				rid.SlotID,
			)

			offset += 2
		}
	}

	if node.Type == InternalNode {
		for _, childPageID := range node.ChildrenPageID {
			binary.LittleEndian.PutUint32(
				data[offset:offset+pageIDSize],
				uint32(childPageID),
			)

			offset += pageIDSize
		}
	}

	return page
}

func DeserializeNode(page *storage.Page) (*Node, error) {
	data := page.Data[:]

	magic := binary.LittleEndian.Uint32(data[0:4])

	if magic != indexMagic {
		return nil, fmt.Errorf("invalid B+ tree node")
	}

	nodeType := NodeType(data[4])

	if nodeType != LeafNode && nodeType != InternalNode {
		return nil, fmt.Errorf("invalid node type")
	}

	keyCount := binary.LittleEndian.Uint16(data[6:8])

	node := &Node{
		Type:   nodeType,
		Keys:   make([]Key, keyCount),
		PageID: page.ID,
	}

	if nodeType == LeafNode {
		node.NextLeaf = storage.PageID(
			binary.LittleEndian.Uint32(data[8:12]),
		)
	}

	offset := indexHeaderSize

	for i := 0; i < int(keyCount); i++ {
		node.Keys[i] = Key(
			binary.LittleEndian.Uint32(
				data[offset : offset+keySize],
			),
		)

		offset += keySize
	}

	if nodeType == LeafNode {
		node.Values = make([]storage.RecordID, keyCount)

		for i := 0; i < int(keyCount); i++ {
			pageID := binary.LittleEndian.Uint32(
				data[offset : offset+4],
			)

			offset += 4

			slotID := binary.LittleEndian.Uint16(
				data[offset : offset+2],
			)

			offset += 2

			node.Values[i] = storage.RecordID{
				PageID: storage.PageID(pageID),
				SlotID: slotID,
			}
		}
	}

	if nodeType == InternalNode {
		node.ChildrenPageID = make([]storage.PageID, keyCount+1)

		for i := 0; i < int(keyCount)+1; i++ {
			childPageID := binary.LittleEndian.Uint32(
				data[offset : offset+pageIDSize],
			)

			offset += pageIDSize

			node.ChildrenPageID[i] = storage.PageID(childPageID)
		}
	}

	return node, nil
}
