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

	numPages, err := pager.NumPages()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Current pages:", numPages)

	page, err := pager.AllocatePage()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Allocated page:", page.ID)

	page.Data[0] = 200

	err = pager.WritePage(page)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Page written successfully")

	numPages, err = pager.NumPages()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Pages after write:", numPages)
}
