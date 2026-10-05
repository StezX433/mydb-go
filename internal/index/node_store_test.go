package index

import (
	"os"
	"testing"

	"mydb-go/internal/storage"
)

func TestNodeStore(t *testing.T) {
	filename := "test_node_store.db"

	defer os.Remove(filename)

	pager, err := storage.NewPager(filename)
	if err != nil {
		t.Fatal(err)
	}

	defer pager.Close()

	store := NewNodeStore(pager)

	node := &Node{
		Type: LeafNode,
		Keys: []Key{10, 20, 30},
		Values: []storage.RecordID{
			{PageID: 1, SlotID: 0},
			{PageID: 2, SlotID: 1},
			{PageID: 3, SlotID: 2},
		},
	}

	pageID, err := store.CreateNode(node)
	if err != nil {
		t.Fatal(err)
	}

	if node.PageID != pageID {
		t.Fatalf(
			"expected node page ID %d, got %d",
			pageID,
			node.PageID,
		)
	}

	loaded, err := store.ReadNode(pageID)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.PageID != pageID {
		t.Fatalf(
			"expected loaded page ID %d, got %d",
			pageID,
			loaded.PageID,
		)
	}

	if len(loaded.Keys) != 3 {
		t.Fatalf(
			"expected 3 keys, got %d",
			len(loaded.Keys),
		)
	}

	for i := range node.Keys {
		if loaded.Keys[i] != node.Keys[i] {
			t.Fatalf("key mismatch at index %d", i)
		}

		if loaded.Values[i] != node.Values[i] {
			t.Fatalf("RecordID mismatch at index %d", i)
		}
	}
}
