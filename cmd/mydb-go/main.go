package main

import (
	"fmt"
	"log"

	"mydb-go/internal/storage"
)

func main() {
	pager, err := storage.NewPager("data/database.db")
	if err != nil {
		log.Fatal(err)
	}
	defer pager.Close()

	// Read the page we previously wrote.
	page, err := pager.ReadPage(5)
	if err != nil {
		log.Fatal(err)
	}

	// Read the record from the page.
	r, err := page.GetRecord(0)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Record read successfully")
	fmt.Println("ID:", r.ID)
	fmt.Println("Age:", r.Age)
	fmt.Println("Name:", string(r.Name[:]))
}
