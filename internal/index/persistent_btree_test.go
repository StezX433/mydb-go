package index

import (
	"os"
	"testing"

	"mydb-go/internal/storage"
)

func TestPersistentBTreeLeafSplit(t *testing.T) {
	filename := "test_persistent_btree_split.db"

	defer os.Remove(filename)

	pager, err := storage.NewPager(filename)
	if err != nil {
		t.Fatal(err)
	}

	tree, err := NewPersistentBTree(pager)
	if err != nil {
		pager.Close()
		t.Fatal(err)
	}

	records := []struct {
		key Key
		rid storage.RecordID
	}{
		{
			key: 10,
			rid: storage.RecordID{
				PageID: 1,
				SlotID: 0,
			},
		},
		{
			key: 20,
			rid: storage.RecordID{
				PageID: 2,
				SlotID: 1,
			},
		},
		{
			key: 30,
			rid: storage.RecordID{
				PageID: 3,
				SlotID: 2,
			},
		},
		{
			key: 40,
			rid: storage.RecordID{
				PageID: 4,
				SlotID: 3,
			},
		},
		{
			key: 50,
			rid: storage.RecordID{
				PageID: 5,
				SlotID: 4,
			},
		},
	}

	for _, record := range records {
		if err := tree.Insert(record.key, record.rid); err != nil {
			pager.Close()
			t.Fatal(err)
		}
	}

	rootID := tree.RootPageID()

	root, err := tree.Root()
	if err != nil {
		pager.Close()
		t.Fatal(err)
	}

	if root.Type != InternalNode {
		pager.Close()
		t.Fatalf(
			"expected root to be internal after split, got %v",
			root.Type,
		)
	}

	if len(root.Keys) != 1 {
		pager.Close()
		t.Fatalf(
			"expected root to have 1 separator key, got %d",
			len(root.Keys),
		)
	}

	if root.Keys[0] != 30 {
		pager.Close()
		t.Fatalf(
			"expected root separator to be 30, got %d",
			root.Keys[0],
		)
	}

	if len(root.ChildrenPageID) != 2 {
		pager.Close()
		t.Fatalf(
			"expected 2 root children, got %d",
			len(root.ChildrenPageID),
		)
	}

	leftPageID := root.ChildrenPageID[0]
	rightPageID := root.ChildrenPageID[1]

	left, err := tree.store.ReadNode(leftPageID)
	if err != nil {
		pager.Close()
		t.Fatal(err)
	}

	right, err := tree.store.ReadNode(rightPageID)
	if err != nil {
		pager.Close()
		t.Fatal(err)
	}

	if left.Type != LeafNode {
		pager.Close()
		t.Fatalf("expected left child to be a leaf")
	}

	if right.Type != LeafNode {
		pager.Close()
		t.Fatalf("expected right child to be a leaf")
	}

	expectedLeftKeys := []Key{10, 20}

	if len(left.Keys) != len(expectedLeftKeys) {
		pager.Close()
		t.Fatalf(
			"expected left leaf to have %d keys, got %d",
			len(expectedLeftKeys),
			len(left.Keys),
		)
	}

	for i, expected := range expectedLeftKeys {
		if left.Keys[i] != expected {
			pager.Close()
			t.Fatalf(
				"expected left key %d at index %d, got %d",
				expected,
				i,
				left.Keys[i],
			)
		}
	}

	expectedRightKeys := []Key{30, 40, 50}

	if len(right.Keys) != len(expectedRightKeys) {
		pager.Close()
		t.Fatalf(
			"expected right leaf to have %d keys, got %d",
			len(expectedRightKeys),
			len(right.Keys),
		)
	}

	for i, expected := range expectedRightKeys {
		if right.Keys[i] != expected {
			pager.Close()
			t.Fatalf(
				"expected right key %d at index %d, got %d",
				expected,
				i,
				right.Keys[i],
			)
		}
	}

	if left.NextLeaf != rightPageID {
		pager.Close()
		t.Fatalf(
			"expected left.NextLeaf to be %d, got %d",
			rightPageID,
			left.NextLeaf,
		)
	}

	if right.NextLeaf != 0 {
		pager.Close()
		t.Fatalf(
			"expected right.NextLeaf to be 0, got %d",
			right.NextLeaf,
		)
	}

	for _, record := range records {
		rid, found, err := tree.Search(record.key)
		if err != nil {
			pager.Close()
			t.Fatal(err)
		}

		if !found {
			pager.Close()
			t.Fatalf(
				"expected key %d to be found",
				record.key,
			)
		}

		if rid != record.rid {
			pager.Close()
			t.Fatalf(
				"key %d: expected RecordID %+v, got %+v",
				record.key,
				record.rid,
				rid,
			)
		}
	}

	if err := pager.Close(); err != nil {
		t.Fatal(err)
	}

	pager, err = storage.NewPager(filename)
	if err != nil {
		t.Fatal(err)
	}

	defer pager.Close()

	reopenedTree, err := OpenPersistentBTree(
		pager,
		rootID,
	)
	if err != nil {
		t.Fatal(err)
	}

	reopenedRoot, err := reopenedTree.Root()
	if err != nil {
		t.Fatal(err)
	}

	if reopenedRoot.Type != InternalNode {
		t.Fatalf(
			"after reopening: expected internal root, got %v",
			reopenedRoot.Type,
		)
	}

	if len(reopenedRoot.Keys) != 1 {
		t.Fatalf(
			"after reopening: expected 1 root key, got %d",
			len(reopenedRoot.Keys),
		)
	}

	if reopenedRoot.Keys[0] != 30 {
		t.Fatalf(
			"after reopening: expected root separator 30, got %d",
			reopenedRoot.Keys[0],
		)
	}

	reopenedLeft, err := reopenedTree.store.ReadNode(
		reopenedRoot.ChildrenPageID[0],
	)
	if err != nil {
		t.Fatal(err)
	}

	reopenedRight, err := reopenedTree.store.ReadNode(
		reopenedRoot.ChildrenPageID[1],
	)
	if err != nil {
		t.Fatal(err)
	}

	if reopenedLeft.NextLeaf != reopenedRight.PageID {
		t.Fatalf(
			"after reopening: expected left NextLeaf %d, got %d",
			reopenedRight.PageID,
			reopenedLeft.NextLeaf,
		)
	}

	expectedLeftKeys = []Key{10, 20}

	if len(reopenedLeft.Keys) != len(expectedLeftKeys) {
		t.Fatalf(
			"after reopening: expected left leaf to have %d keys, got %d",
			len(expectedLeftKeys),
			len(reopenedLeft.Keys),
		)
	}

	for i, expected := range expectedLeftKeys {
		if reopenedLeft.Keys[i] != expected {
			t.Fatalf(
				"after reopening: expected left key %d at index %d, got %d",
				expected,
				i,
				reopenedLeft.Keys[i],
			)
		}
	}

	expectedRightKeys = []Key{30, 40, 50}

	if len(reopenedRight.Keys) != len(expectedRightKeys) {
		t.Fatalf(
			"after reopening: expected right leaf to have %d keys, got %d",
			len(expectedRightKeys),
			len(reopenedRight.Keys),
		)
	}

	for i, expected := range expectedRightKeys {
		if reopenedRight.Keys[i] != expected {
			t.Fatalf(
				"after reopening: expected right key %d at index %d, got %d",
				expected,
				i,
				reopenedRight.Keys[i],
			)
		}
	}

	for _, record := range records {
		rid, found, err := reopenedTree.Search(record.key)
		if err != nil {
			t.Fatal(err)
		}

		if !found {
			t.Fatalf(
				"after reopening: expected key %d to be found",
				record.key,
			)
		}

		if rid != record.rid {
			t.Fatalf(
				"after reopening: key %d: expected RecordID %+v, got %+v",
				record.key,
				record.rid,
				rid,
			)
		}
	}
}

func TestPersistentBTreeRangeScan(t *testing.T) {
	filename := "test_persistent_btree_range.db"

	defer os.Remove(filename)

	pager, err := storage.NewPager(filename)
	if err != nil {
		t.Fatal(err)
	}

	tree, err := NewPersistentBTree(pager)
	if err != nil {
		pager.Close()
		t.Fatal(err)
	}

	records := []struct {
		key Key
		rid storage.RecordID
	}{
		{10, storage.RecordID{PageID: 1, SlotID: 0}},
		{20, storage.RecordID{PageID: 2, SlotID: 1}},
		{30, storage.RecordID{PageID: 3, SlotID: 2}},
		{40, storage.RecordID{PageID: 4, SlotID: 3}},
		{50, storage.RecordID{PageID: 5, SlotID: 4}},
		{60, storage.RecordID{PageID: 6, SlotID: 5}},
		{70, storage.RecordID{PageID: 7, SlotID: 6}},
	}

	for _, record := range records {
		if err := tree.Insert(record.key, record.rid); err != nil {
			pager.Close()
			t.Fatal(err)
		}
	}

	results, err := tree.RangeScan(20, 60)
	if err != nil {
		pager.Close()
		t.Fatal(err)
	}

	expected := []storage.RecordID{
		{PageID: 2, SlotID: 1},
		{PageID: 3, SlotID: 2},
		{PageID: 4, SlotID: 3},
		{PageID: 5, SlotID: 4},
		{PageID: 6, SlotID: 5},
	}

	if len(results) != len(expected) {
		pager.Close()
		t.Fatalf(
			"expected %d results, got %d",
			len(expected),
			len(results),
		)
	}

	for i := range expected {
		if results[i] != expected[i] {
			pager.Close()
			t.Fatalf(
				"result %d: expected %+v, got %+v",
				i,
				expected[i],
				results[i],
			)
		}
	}

	if err := pager.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPersistentBTreeValidation(t *testing.T) {
	filename := "test_persistent_btree_validation.db"

	defer os.Remove(filename)

	pager, err := storage.NewPager(filename)
	if err != nil {
		t.Fatal(err)
	}

	tree, err := NewPersistentBTree(pager)
	if err != nil {
		pager.Close()
		t.Fatal(err)
	}

	records := []struct {
		key Key
		rid storage.RecordID
	}{
		{10, storage.RecordID{PageID: 1, SlotID: 0}},
		{20, storage.RecordID{PageID: 2, SlotID: 1}},
		{30, storage.RecordID{PageID: 3, SlotID: 2}},
		{40, storage.RecordID{PageID: 4, SlotID: 3}},
		{50, storage.RecordID{PageID: 5, SlotID: 4}},
		{60, storage.RecordID{PageID: 6, SlotID: 5}},
		{70, storage.RecordID{PageID: 7, SlotID: 6}},
		{80, storage.RecordID{PageID: 8, SlotID: 7}},
		{90, storage.RecordID{PageID: 9, SlotID: 8}},
	}

	for _, record := range records {
		if err := tree.Insert(record.key, record.rid); err != nil {
			pager.Close()
			t.Fatal(err)
		}
	}

	if err := tree.Validate(); err != nil {
		pager.Close()
		t.Fatalf("tree validation failed: %v", err)
	}

	if err := pager.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPersistentBTreeDeleteWithoutRebalance(t *testing.T) {
	filename := "test_persistent_btree_delete_simple.db"

	defer os.Remove(filename)

	pager, err := storage.NewPager(filename)
	if err != nil {
		t.Fatal(err)
	}

	tree, err := NewPersistentBTree(pager)
	if err != nil {
		pager.Close()
		t.Fatal(err)
	}

	for i := 1; i <= 5; i++ {
		key := Key(i * 10)

		if err := tree.Insert(
			key,
			storage.RecordID{
				PageID: storage.PageID(i),
				SlotID: uint16(i),
			},
		); err != nil {
			pager.Close()
			t.Fatal(err)
		}
	}

	if err := tree.Delete(40); err != nil {
		pager.Close()
		t.Fatal(err)
	}

	_, found, err := tree.Search(40)
	if err != nil {
		pager.Close()
		t.Fatal(err)
	}

	if found {
		pager.Close()
		t.Fatalf("expected key 40 to be deleted")
	}

	for _, key := range []Key{10, 20, 30, 50} {
		_, found, err := tree.Search(key)
		if err != nil {
			pager.Close()
			t.Fatal(err)
		}

		if !found {
			pager.Close()
			t.Fatalf(
				"expected key %d to remain",
				key,
			)
		}
	}

	if err := tree.Validate(); err != nil {
		pager.Close()
		t.Fatalf(
			"tree validation failed after deletion: %v",
			err,
		)
	}

	if err := pager.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPersistentBTreeDeleteBorrowFromRight(t *testing.T) {
	filename := "test_persistent_btree_delete_borrow_right.db"

	defer os.Remove(filename)

	pager, err := storage.NewPager(filename)
	if err != nil {
		t.Fatal(err)
	}

	tree, err := NewPersistentBTree(pager)
	if err != nil {
		pager.Close()
		t.Fatal(err)
	}

	for i := 1; i <= 5; i++ {
		key := Key(i * 10)

		if err := tree.Insert(
			key,
			storage.RecordID{
				PageID: storage.PageID(i),
				SlotID: uint16(i),
			},
		); err != nil {
			pager.Close()
			t.Fatal(err)
		}
	}

	if err := tree.Delete(20); err != nil {
		pager.Close()
		t.Fatal(err)
	}

	root, err := tree.Root()
	if err != nil {
		pager.Close()
		t.Fatal(err)
	}

	if root.Type != InternalNode {
		pager.Close()
		t.Fatalf("expected internal root")
	}

	if len(root.Keys) != 1 {
		pager.Close()
		t.Fatalf(
			"expected 1 separator key, got %d",
			len(root.Keys),
		)
	}

	if root.Keys[0] != 40 {
		pager.Close()
		t.Fatalf(
			"expected separator 40 after redistribution, got %d",
			root.Keys[0],
		)
	}

	left, err := tree.store.ReadNode(
		root.ChildrenPageID[0],
	)
	if err != nil {
		pager.Close()
		t.Fatal(err)
	}

	right, err := tree.store.ReadNode(
		root.ChildrenPageID[1],
	)
	if err != nil {
		pager.Close()
		t.Fatal(err)
	}

	expectedLeft := []Key{10, 30}
	expectedRight := []Key{40, 50}

	if len(left.Keys) != len(expectedLeft) {
		pager.Close()
		t.Fatalf(
			"expected left leaf to have %d keys, got %d",
			len(expectedLeft),
			len(left.Keys),
		)
	}

	if len(right.Keys) != len(expectedRight) {
		pager.Close()
		t.Fatalf(
			"expected right leaf to have %d keys, got %d",
			len(expectedRight),
			len(right.Keys),
		)
	}

	for i := range expectedLeft {
		if left.Keys[i] != expectedLeft[i] {
			pager.Close()
			t.Fatalf(
				"left key %d: expected %d, got %d",
				i,
				expectedLeft[i],
				left.Keys[i],
			)
		}
	}

	for i := range expectedRight {
		if right.Keys[i] != expectedRight[i] {
			pager.Close()
			t.Fatalf(
				"right key %d: expected %d, got %d",
				i,
				expectedRight[i],
				right.Keys[i],
			)
		}
	}

	if err := tree.Validate(); err != nil {
		pager.Close()
		t.Fatalf(
			"tree validation failed after redistribution: %v",
			err,
		)
	}

	if err := pager.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPersistentBTreeDeleteBorrowFromLeft(t *testing.T) {
	filename := "test_persistent_btree_delete_borrow_left.db"

	defer os.Remove(filename)

	pager, err := storage.NewPager(filename)
	if err != nil {
		t.Fatal(err)
	}

	tree, err := NewPersistentBTree(pager)
	if err != nil {
		pager.Close()
		t.Fatal(err)
	}

	keys := []Key{10, 20, 25, 30, 40, 50, 60, 70}

	for i, key := range keys {
		if err := tree.Insert(
			key,
			storage.RecordID{
				PageID: storage.PageID(i + 1),
				SlotID: uint16(i),
			},
		); err != nil {
			pager.Close()
			t.Fatal(err)
		}
	}

	if err := tree.Delete(30); err != nil {
		pager.Close()
		t.Fatal(err)
	}

	_, found, err := tree.Search(30)
	if err != nil {
		pager.Close()
		t.Fatal(err)
	}

	if found {
		pager.Close()
		t.Fatalf("expected key 30 to be deleted")
	}

	if err := tree.Validate(); err != nil {
		pager.Close()
		t.Fatalf(
			"tree validation failed after left redistribution: %v",
			err,
		)
	}

	if err := pager.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPersistentBTreeDeleteMerge(t *testing.T) {
	filename := "test_persistent_btree_delete_merge.db"

	defer os.Remove(filename)

	pager, err := storage.NewPager(filename)
	if err != nil {
		t.Fatal(err)
	}

	tree, err := NewPersistentBTree(pager)
	if err != nil {
		pager.Close()
		t.Fatal(err)
	}

	for i := 1; i <= 4; i++ {
		key := Key(i * 10)

		if err := tree.Insert(
			key,
			storage.RecordID{
				PageID: storage.PageID(i),
				SlotID: uint16(i),
			},
		); err != nil {
			pager.Close()
			t.Fatal(err)
		}
	}

	if err := tree.Delete(20); err != nil {
		pager.Close()
		t.Fatal(err)
	}

	root, err := tree.Root()
	if err != nil {
		pager.Close()
		t.Fatal(err)
	}

	if root.Type != LeafNode {
		pager.Close()
		t.Fatalf(
			"expected root to collapse into a leaf, got %v",
			root.Type,
		)
	}

	expected := []Key{10, 30, 40}

	if len(root.Keys) != len(expected) {
		pager.Close()
		t.Fatalf(
			"expected %d keys, got %d",
			len(expected),
			len(root.Keys),
		)
	}

	for i := range expected {
		if root.Keys[i] != expected[i] {
			pager.Close()
			t.Fatalf(
				"expected key %d at index %d, got %d",
				expected[i],
				i,
				root.Keys[i],
			)
		}
	}

	if err := tree.Validate(); err != nil {
		pager.Close()
		t.Fatalf(
			"tree validation failed after merge: %v",
			err,
		)
	}

	if err := pager.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPersistentBTreeDeleteAll(t *testing.T) {
	filename := "test_persistent_btree_delete_all.db"

	defer os.Remove(filename)

	pager, err := storage.NewPager(filename)
	if err != nil {
		t.Fatal(err)
	}

	tree, err := NewPersistentBTree(pager)
	if err != nil {
		pager.Close()
		t.Fatal(err)
	}

	const count = 50

	for i := 1; i <= count; i++ {
		key := Key(i)

		if err := tree.Insert(
			key,
			storage.RecordID{
				PageID: storage.PageID(i),
				SlotID: uint16(i),
			},
		); err != nil {
			pager.Close()
			t.Fatal(err)
		}
	}

	for i := 1; i <= count; i++ {
		key := Key(i)

		if err := tree.Delete(key); err != nil {
			pager.Close()
			t.Fatalf(
				"failed to delete key %d: %v",
				key,
				err,
			)
		}

		_, found, err := tree.Search(key)
		if err != nil {
			pager.Close()
			t.Fatal(err)
		}

		if found {
			pager.Close()
			t.Fatalf(
				"key %d still found after deletion",
				key,
			)
		}

		if err := tree.Validate(); err != nil {
			pager.Close()
			t.Fatalf(
				"tree validation failed after deleting %d: %v",
				key,
				err,
			)
		}
	}

	root, err := tree.Root()
	if err != nil {
		pager.Close()
		t.Fatal(err)
	}

	if root.Type != LeafNode {
		pager.Close()
		t.Fatalf(
			"expected final root to be a leaf, got %v",
			root.Type,
		)
	}

	if len(root.Keys) != 0 {
		pager.Close()
		t.Fatalf(
			"expected final root to be empty, got %d keys",
			len(root.Keys),
		)
	}

	if err := pager.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPersistentBTreeDeleteMissingKey(t *testing.T) {
	filename := "test_persistent_btree_delete_missing.db"

	defer os.Remove(filename)

	pager, err := storage.NewPager(filename)
	if err != nil {
		t.Fatal(err)
	}

	tree, err := NewPersistentBTree(pager)
	if err != nil {
		pager.Close()
		t.Fatal(err)
	}

	for i := 1; i <= 5; i++ {
		key := Key(i * 10)

		if err := tree.Insert(
			key,
			storage.RecordID{
				PageID: storage.PageID(i),
				SlotID: uint16(i),
			},
		); err != nil {
			pager.Close()
			t.Fatal(err)
		}
	}

	if err := tree.Delete(999); err != nil {
		pager.Close()
		t.Fatal(err)
	}

	for _, key := range []Key{10, 20, 30, 40, 50} {
		_, found, err := tree.Search(key)
		if err != nil {
			pager.Close()
			t.Fatal(err)
		}

		if !found {
			pager.Close()
			t.Fatalf(
				"key %d disappeared after deleting missing key",
				key,
			)
		}
	}

	if err := tree.Validate(); err != nil {
		pager.Close()
		t.Fatalf(
			"tree validation failed: %v",
			err,
		)
	}

	if err := pager.Close(); err != nil {
		t.Fatal(err)
	}
}
