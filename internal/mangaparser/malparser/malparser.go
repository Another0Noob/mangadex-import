package malparser

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
)

type MALData struct {
	// XMLName xml.Name `xml:"myanimelist"`
	Entries []Manga `xml:"manga"`
}

type Manga struct {
	ID       int    `xml:"manga_mangadb_id"`
	Title    string `xml:"manga_title"`
	MyStatus string `xml:"my_status"`
}

func ParseMALFile(path string) (*MALData, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	return ParseMALReader(file)
}

// ParseMALReader parses MAL XML data from any io.Reader
func ParseMALReader(reader io.Reader) (*MALData, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read data: %w", err)
	}

	var malData MALData
	if err := xml.Unmarshal(data, &malData); err != nil {
		return nil, err
	}

	return &malData, nil
}

func ReturnMALTitles(manga *MALData) []string {
	titles := make([]string, len(manga.Entries))
	for i, m := range manga.Entries {
		titles[i] = m.Title
	}
	return titles
}
