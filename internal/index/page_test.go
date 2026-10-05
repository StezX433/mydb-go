package index

import (
	"testing"

	"mydb-go/internal/storage"
)

func TestNodeSerialization(t *testing.T) {
	node := &Node{
		Type:   LeafNode,
		PageID: 10,
		Keys:   []Key{30, 40, 50},
		Values: []storage.RecordID{
			{PageID: 1, SlotID: 2},
			{PageID: 3, SlotID: 4},
			{PageID: 5, SlotID: 6},
		},
	}

	page := SerializeNode(node)

	decoded, err := DeserializeNode(page)
	if err != nil {
		t.Fatal(err)
	}

	if decoded.Type != LeafNode {
		t.Fatalf("expected leaf node")
	}

	if decoded.PageID != 10 {
		t.Fatalf(
			"expected page ID 10, got %d",
			decoded.PageID,
		)
	}

	if len(decoded.Keys) != 3 {
		t.Fatalf(
			"expected 3 keys, got %d",
			len(decoded.Keys),
		)
	}

	for i := range node.Keys {
		if decoded.Keys[i] != node.Keys[i] {
			t.Fatalf(
				"key mismatch at index %d: expected %d, got %d",
				i,
				node.Keys[i],
				decoded.Keys[i],
			)
		}

		if decoded.Values[i] != node.Values[i] {
			t.Fatalf(
				"RecordID mismatch at index %d",
				i,
			)
		}
	}
}

func TestInternalNodeSerialization(t *testing.T) {
	node := &Node{
		Type: InternalNode,
		PageID: 10,
		Keys: []Key{
			70,
			130,
			190,
		},
		ChildrenPageID: []storage.PageID{
			2,
			5,
			8,
			12,
		},
	}

	page := SerializeNode(node)

	decoded, err := DeserializeNode(page)
	if err != nil {
		t.Fatal(err)
	}

	if decoded.Type != InternalNode {
		t.Fatalf("expected internal node")
	}

	if decoded.PageID != 10 {
		t.Fatalf(
			"expected page ID 10, got %d",
			decoded.PageID,
		)
	}

	if len(decoded.Keys) != 3 {
		t.Fatalf(
			"expected 3 keys, got %d",
			len(decoded.Keys),
		)
	}

	if len(decoded.ChildrenPageID) != 4 {
		t.Fatalf(
			"expected 4 children, got %d",
			len(decoded.ChildrenPageID),
		)
	}

	for i := range node.Keys {
		if decoded.Keys[i] != node.Keys[i] {
			t.Fatalf(
				"key mismatch at index %d: expected %d, got %d",
				i,
				node.Keys[i],
				decoded.Keys[i],
			)
		}
	}

	for i := range node.ChildrenPageID {
		if decoded.ChildrenPageID[i] != node.ChildrenPageID[i] {
			t.Fatalf(
				"child page ID mismatch at index %d: expected %d, got %d",
				i,
				node.ChildrenPageID[i],
				decoded.ChildrenPageID[i],
			)
		}
	}
}