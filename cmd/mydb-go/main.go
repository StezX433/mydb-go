package main

import (
	"fmt"

	"mydb-go/internal/index"
	"mydb-go/internal/storage"
)

func main() {
	tree := index.NewBTree()

	tree.Insert(10, storage.RecordID{PageID: 1, SlotID: 0})
	tree.Insert(20, storage.RecordID{PageID: 1, SlotID: 1})
	tree.Insert(30, storage.RecordID{PageID: 1, SlotID: 2})
	tree.Insert(40, storage.RecordID{PageID: 1, SlotID: 3})
	tree.Insert(50, storage.RecordID{PageID: 1, SlotID: 4})

	fmt.Println("Root type:", tree.Root.Type)
	fmt.Println("Root keys:", tree.Root.Keys)
	fmt.Println("Number of children:", len(tree.Root.Children))

	for i, child := range tree.Root.Children {
		fmt.Println("Child", i, "keys:", child.Keys)
	}

	rid, found := tree.Search(40)

	fmt.Println("Search 40:", found)
	fmt.Println("RecordID:", rid)

	_, found = tree.Search(100)

	fmt.Println("Search 100:", found)
}
