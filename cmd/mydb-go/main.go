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

	// Check current number of pages
	numPages, err := pager.NumPages()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Current pages:", numPages)

	// Allocate a new page
	page, err := pager.AllocatePage()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Allocated page:", page.ID)

	// Put some data into it
	page.Data[0] = 200

	// Write it to disk
	err = pager.WritePage(page)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Page written successfully")

	// Check number of pages again
	numPages, err = pager.NumPages()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Pages after write:", numPages)
}
