package malparser

import (
	"encoding/xml"
	"io"
	"os"
)

type MALData struct {
	XMLName xml.Name `xml:"myanimelist"`
	Entries []Manga  `xml:"manga"`
}

type Manga struct {
	ID       int    `xml:"manga_mangadb_id"`
	Title    string `xml:"manga_title"`
	MyStatus string `xml:"my_status"`
}

func ParseMALFile(filePath string) ([]Manga, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var malData MALData
	if err := xml.Unmarshal(data, &malData); err != nil {
		return nil, err
	}

	return malData.Entries, nil
}
