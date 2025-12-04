package comickparser

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
)

// Manga represents a single row from the comick CSV export.
// Fields are exported so callers can read them.
type Manga struct {
	// HID          string   // hid column
	Title string // title column
	// Type         string   // type column (Manga/Manhwa/etc)
	// Rating       string   // rating column (kept as string to preserve whatever format)
	// Origination  string   // origination column
	// Read         string   // read column (kept as string to preserve whatever format)
	// LastRead     string   // last_read column
	// Synonyms []string // parsed synonyms (split on comma/semicolon/pipe)
	// MAL          string   // myanimelist url/id column
	// AniList      string   // anilist url/id column
	// MangaUpdates string   // mangaupdates url/id column
}

// ParseComickFile reads the CSV at filePath and returns a slice of Manga.
// The parser is header-aware: it maps columns by header name (case-insensitive).
// If a header is missing, a sensible default index is used based on the typical
// comick export layout:
//
//	hid,title,type,rating,origination,read,last_read,synonyms,mal,anilist,mangaupdates
//
// Synonyms are split on ',', ';' or '|' and trimmed. Rows without a title are
// skipped.

// ParseComickFile parses a Comick CSV file from disk (original)
func ParseComickFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	return ParseComickReader(file)
}

// ParseComickReader parses Comick CSV data from any io.Reader
func ParseComickReader(reader io.Reader) ([]string, error) {
	r := csv.NewReader(reader)
	// Read header row (required for mapping). If EOF, return empty slice.
	header, err := r.Read()
	if err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, err
	}

	// Build header map with normalized names
	headerMap := make(map[string]int, len(header))
	for i, h := range header {
		headerMap[h] = i
	}

	// Read remaining records
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	// Helper: get index for a canonical name, fallback to default indices
	defaults := map[string]int{
		// "hid":          0,
		"title": 1,
		// "type":         2,
		// "rating":       3,
		// "origination":  4,
		// "read":         5,
		// "last_read":    6,
		// "synonyms": 7,
		// "mal":          8,
		// "anilist":      9,
		// "mangaupdates": 10,
	}

	getIndex := func(name string) int {
		if i, ok := headerMap[name]; ok {
			return i
		}
		if d, ok := defaults[name]; ok {
			return d
		}
		return -1
	}

	// hidIdx := getIndex("hid")
	titleIdx := getIndex("title")
	// typeIdx := getIndex("type")
	// ratingIdx := getIndex("rating")
	// origIdx := getIndex("origination")
	// readIdx := getIndex("read")
	// lastReadIdx := getIndex("last_read")
	// synIdx := getIndex("synonyms")
	// malIdx := getIndex("mal")
	// aniIdx := getIndex("anilist")
	// muIdx := getIndex("mangaupdates")

	/*
		splitSynonyms := func(s string) []string {
			s = strings.TrimSpace(s)
			if s == "" {
				return nil
			}
			var parts []string
			parts = strings.Split(s, ",")
			out := make([]string, 0, len(parts))
			for _, p := range parts {
				if t := strings.TrimSpace(p); t != "" {
					out = append(out, t)
				}
			}
			return out
		}
	*/

	get := func(rec []string, idx int) string {
		if idx < 0 || idx >= len(rec) {
			return ""
		}
		return strings.TrimSpace(rec[idx])
	}

	var out []string
	for _, rec := range records {
		// Skip completely empty records
		if len(rec) == 0 {
			continue
		}

		title := get(rec, titleIdx)
		// If title is missing, skip row
		if title == "" {
			continue
		}

		/*
			m := Manga{
				// HID:          get(rec, hidIdx),
				Title: title,
				// Type:         get(rec, typeIdx),
				// Rating:       get(rec, ratingIdx),
				// Origination:  get(rec, origIdx),
				// Read:         get(rec, readIdx),
				// LastRead:     get(rec, lastReadIdx),
				// Synonyms: splitSynonyms(get(rec, synIdx)),
				// MAL:          get(rec, malIdx),
				// AniList:      get(rec, aniIdx),
				// MangaUpdates: get(rec, muIdx),
			}
		*/
		out = append(out, title)
	}

	return out, nil
}

// normalizeHeader converts header string to a normalized canonical form used
// for comparison: lowercased, trimmed, spaces -> underscore, and common
// punctuation removed. This helps match headers like "Last Read" and "last_read".
func normalizeHeader(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	h = strings.ReplaceAll(h, " ", "_")
	h = strings.ReplaceAll(h, "-", "_")
	h = strings.ReplaceAll(h, ".", "")
	h = strings.ReplaceAll(h, "\"", "")
	h = strings.ReplaceAll(h, "`", "")
	// collapse multiple underscores
	for strings.Contains(h, "__") {
		h = strings.ReplaceAll(h, "__", "_")
	}
	return h
}
