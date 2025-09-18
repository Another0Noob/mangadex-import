package main

import (
	"fmt"

	"github.com/Another0Noob/mangadex-import/internal/malparser"
)

func main() {
	malPtr, err := malparser.ParseMALFile("file.xml")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	for _, manga := range malPtr.Entries {
		fmt.Println(manga.Title)
	}
}
