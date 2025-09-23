package sync

import "github.com/Another0Noob/mangadex-import/internal/malparser"

type FollowedIndexes struct {
	MainTitleIndex map[string]string   // normalized main title -> mangaID
	AltTitleIndex  map[string]string   // normalized alt title  -> mangaID
	IDToTitles     map[string][]string // mangaID -> all normalized titles (debug)
	AllTitles      []string            // deduped list of all normalized titles (for fuzzy scan)
}

type TitleMatchResult struct {
	Matched   []malparser.Manga
	Unmatched []malparser.Manga
}
