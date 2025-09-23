package sync

type FollowedIndexes struct {
	MainTitleIndex map[string]string   // normalized main title -> mangaID
	AltTitleIndex  map[string]string   // normalized alt title  -> mangaID
	IDToTitles     map[string][]string // mangaID -> all normalized titles (debug)
	AllTitles      []string            // deduped list of all normalized titles (for fuzzy scan)
}
