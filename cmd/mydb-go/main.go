package main

import (
	"fmt"

	"mydb-go/internal/index"
	"mydb-go/internal/storage"
)

func printTree(node *index.Node, level int) {
	fmt.Printf("Level %d: %v\n", level, node.Keys)

	for _, child := range node.Children {
		printTree(child, level+1)
	}
}

func main() {
	tree := index.NewBTree()

	for i := 1; i <= 30; i++ {
		tree.Insert(
			index.Key(i*10),
			storage.RecordID{
				PageID: 1,
				SlotID: uint16(i),
			},
		)
	}

	printTree(tree.Root, 0)

	rid, found := tree.Search(250)

	fmt.Println("Search 250:", found)
	fmt.Println("RecordID:", rid)

	_, found = tree.Search(999)

	fmt.Println("Search 999:", found)
}
