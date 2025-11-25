package match

import (
	"context"
	"errors"
	"strings"

	"github.com/Another0Noob/mangadex-import/internal/mangadexapi"
	"github.com/lithammer/fuzzysearch/fuzzy"
)

//TODO: Refactor for other import formats

type MatchInfo struct {
	MangaDexTitle string
	ImportTitle   string
	MatchType     string // "exact" or "fuzzy"
}

// ImportEntry bundles original manga with its normalized title
type ImportEntry struct {
	Original   string
	Normalized string
}

type FollowedIndexes struct {
	MainTitleIndex map[string]string   // normalized main title -> mangaID
	AltTitleIndex  map[string]string   // normalized alt title  -> mangaID
	IDToTitles     map[string][]string // mangaID -> all normalized titles (debug)
	AllTitles      []string            // deduped list of all normalized titles (for fuzzy scan)
}

type Unmatched struct {
	MD        []mangadexapi.Manga
	Import    []ImportEntry // bundled to prevent index misalignment
	MDIndexes FollowedIndexes
}

type MatchResult struct {
	Matches   map[string]MatchInfo // key: MangaDex ID
	Unmatched Unmatched
}

func isEnglishOrRomanized(lang string) bool {
	return lang == "en" || strings.HasSuffix(lang, "-ro")
}

// pickOriginalTitle prefers human-friendly MangaDex title for logging
func pickOriginalTitle(m mangadexapi.Manga) string {
	attrs := m.Attributes

	// Prefer English primary title
	if t, ok := attrs.Title["en"]; ok && t != "" {
		return t
	}

	// Try romanized primary titles
	for lang, t := range attrs.Title {
		if strings.HasSuffix(lang, "-ro") && t != "" {
			return t
		}
	}

	// Try English alt titles
	for _, mp := range attrs.AltTitles {
		if t, ok := mp["en"]; ok && t != "" {
			return t
		}
	}

	// Try romanized alt titles
	for _, mp := range attrs.AltTitles {
		for lang, t := range mp {
			if strings.HasSuffix(lang, "-ro") && t != "" {
				return t
			}
		}
	}

	// Fallback: any primary title
	for _, t := range attrs.Title {
		if t != "" {
			return t
		}
	}

	// Fallback: any alt title
	for _, mp := range attrs.AltTitles {
		for _, t := range mp {
			if t != "" {
				return t
			}
		}
	}

	return ""
}

// BuildFollowedIndexes creates searchable indexes from MangaDex manga
func BuildFollowedIndexes(followed []mangadexapi.Manga) FollowedIndexes {
	mainIdx := make(map[string]string, len(followed))
	altIdx := make(map[string]string)
	idToTitles := make(map[string][]string, len(followed))
	seen := make(map[string]struct{})

	for _, m := range followed {
		collected := make([]string, 0, 4)

		// Main titles
		for lang, title := range m.Attributes.Title {
			if !isEnglishOrRomanized(lang) || title == "" {
				continue
			}
			n := normalizeTitle(title)
			if n == "" {
				continue
			}
			mainIdx[n] = m.ID
			collected = append(collected, n)
			seen[n] = struct{}{}
		}

		// Alt titles
		for _, altMap := range m.Attributes.AltTitles {
			for lang, title := range altMap {
				if !isEnglishOrRomanized(lang) || title == "" {
					continue
				}
				n := normalizeTitle(title)
				if n == "" {
					continue
				}
				altIdx[n] = m.ID
				collected = append(collected, n)
				seen[n] = struct{}{}
			}
		}

		if len(collected) > 0 {
			idToTitles[m.ID] = collected
		}
	}

	allTitles := make([]string, 0, len(seen))
	for t := range seen {
		allTitles = append(allTitles, t)
	}

	return FollowedIndexes{
		MainTitleIndex: mainIdx,
		AltTitleIndex:  altIdx,
		IDToTitles:     idToTitles,
		AllTitles:      allTitles,
	}
}

// buildOwnerSets inverts IDToTitles to normalized-title -> []MangaDexID
func buildOwnerSets(idToTitles map[string][]string) map[string][]string {
	owners := make(map[string][]string)
	for id, titles := range idToTitles {
		for _, t := range titles {
			owners[t] = append(owners[t], id)
		}
	}
	return owners
}

// rebuildIndexes creates new indexes excluding matched IDs
func rebuildIndexes(oldIdx FollowedIndexes, matchedIDs map[string]struct{}) FollowedIndexes {
	newMain := make(map[string]string)
	newAlt := make(map[string]string)
	newIDToTitles := make(map[string][]string)
	seen := make(map[string]struct{})

	// Filter MainTitleIndex
	for normTitle, id := range oldIdx.MainTitleIndex {
		if _, matched := matchedIDs[id]; !matched {
			newMain[normTitle] = id
			seen[normTitle] = struct{}{}
		}
	}

	// Filter AltTitleIndex
	for normTitle, id := range oldIdx.AltTitleIndex {
		if _, matched := matchedIDs[id]; !matched {
			newAlt[normTitle] = id
			seen[normTitle] = struct{}{}
		}
	}

	// Filter IDToTitles
	for id, titles := range oldIdx.IDToTitles {
		if _, matched := matchedIDs[id]; !matched {
			newIDToTitles[id] = titles
			for _, t := range titles {
				seen[t] = struct{}{}
			}
		}
	}

	// Rebuild AllTitles
	newAll := make([]string, 0, len(seen))
	for t := range seen {
		newAll = append(newAll, t)
	}

	return FollowedIndexes{
		MainTitleIndex: newMain,
		AltTitleIndex:  newAlt,
		IDToTitles:     newIDToTitles,
		AllTitles:      newAll,
	}
}

// MatchDirect performs exact normalized title matching
func MatchDirect(followed []mangadexapi.Manga, importManga []string) MatchResult {
	if len(followed) == 0 || len(importManga) == 0 {
		return MatchResult{
			Matches: make(map[string]MatchInfo),
			Unmatched: Unmatched{
				MD:        followed,
				Import:    normalizeImportEntries(importManga),
				MDIndexes: BuildFollowedIndexes(followed),
			},
		}
	}

	idx := BuildFollowedIndexes(followed)

	// Build lookup for full MangaDex objects
	mdByID := make(map[string]mangadexapi.Manga, len(followed))
	for _, m := range followed {
		mdByID[m.ID] = m
	}

	// Invert to handle title collisions
	owners := buildOwnerSets(idx.IDToTitles)

	matches := make(map[string]MatchInfo)
	matchedIDs := make(map[string]struct{})
	matchedImportIdx := make(map[int]struct{})

	// Find exact matches (only when unambiguous)
	for i, mm := range importManga {
		n := normalizeTitle(mm)
		if n == "" {
			continue
		}

		ids := owners[n]
		if len(ids) == 1 {
			id := ids[0]
			if _, seen := matches[id]; !seen {
				matches[id] = MatchInfo{
					MangaDexTitle: pickOriginalTitle(mdByID[id]),
					ImportTitle:   mm,
					MatchType:     "exact",
				}
				matchedIDs[id] = struct{}{}
				matchedImportIdx[i] = struct{}{}
			}
		}
		// Skip ambiguous (len>1) or no match (len==0)
	}

	// Build unmatched sets
	unmatchedMD := make([]mangadexapi.Manga, 0, len(followed)-len(matchedIDs))
	for _, m := range followed {
		if _, matched := matchedIDs[m.ID]; !matched {
			unmatchedMD = append(unmatchedMD, m)
		}
	}

	unmatchedImport := make([]ImportEntry, 0, len(importManga)-len(matchedImportIdx))
	for i, mm := range importManga {
		if _, matched := matchedImportIdx[i]; !matched {
			unmatchedImport = append(unmatchedImport, ImportEntry{
				Original:   mm,
				Normalized: normalizeTitle(mm),
			})
		}
	}

	return MatchResult{
		Matches: matches,
		Unmatched: Unmatched{
			MD:        unmatchedMD,
			Import:    unmatchedImport,
			MDIndexes: rebuildIndexes(idx, matchedIDs),
		},
	}
}

// normalizeImportEntries converts MAL manga to MALEntry format
func normalizeImportEntries(importManga []string) []ImportEntry {
	entries := make([]ImportEntry, len(importManga))
	for i, mm := range importManga {
		entries[i] = ImportEntry{
			Original:   mm,
			Normalized: normalizeTitle(mm),
		}
	}
	return entries
}

// FuzzyMatch adds fuzzy matches to existing MatchResult
func FuzzyMatch(res MatchResult) MatchResult {
	remaining := res.Unmatched.MDIndexes
	unmatchedMD := res.Unmatched.MD
	unmatchedImport := res.Unmatched.Import

	// Quick exits
	if len(remaining.AllTitles) == 0 || len(unmatchedImport) == 0 {
		return res
	}

	// Build MD lookup
	mdByID := make(map[string]*mangadexapi.Manga, len(unmatchedMD))
	for i := range unmatchedMD {
		mdByID[unmatchedMD[i].ID] = &unmatchedMD[i]
	}

	owners := buildOwnerSets(remaining.IDToTitles)

	newMatches := make(map[string]MatchInfo)
	matchedIDs := make(map[string]struct{})
	matchedImportIdx := make(map[int]struct{})

	for i, entry := range unmatchedImport {
		pat := entry.Normalized
		if pat == "" {
			continue
		}

		thr := distanceThreshold(len(pat))
		candidates := filterCandidates(remaining.AllTitles, pat, thr)
		if len(candidates) == 0 {
			continue
		}

		// Find best fuzzy match
		ranks := fuzzy.RankFind(pat, candidates)
		if len(ranks) == 0 || ranks[0].Distance > thr {
			continue
		}

		// Map back to MD ID (only if unambiguous)
		candNorm := ranks[0].Target
		ids := owners[candNorm]
		if len(ids) != 1 {
			continue
		}

		id := ids[0]
		if _, already := matchedIDs[id]; already {
			continue
		}

		// Record fuzzy match
		md := mdByID[id]
		newMatches[id] = MatchInfo{
			MangaDexTitle: pickOriginalTitle(*md),
			ImportTitle:   entry.Original,
			MatchType:     "fuzzy",
		}
		matchedIDs[id] = struct{}{}
		matchedImportIdx[i] = struct{}{}
	}

	// Merge new matches
	for id, mi := range newMatches {
		if _, exists := res.Matches[id]; !exists {
			res.Matches[id] = mi
		}
	}

	// Update remaining sets
	res.Unmatched.MDIndexes = rebuildIndexes(remaining, matchedIDs)

	newUnmatchedMD := make([]mangadexapi.Manga, 0, len(unmatchedMD))
	for _, m := range unmatchedMD {
		if _, matched := matchedIDs[m.ID]; !matched {
			newUnmatchedMD = append(newUnmatchedMD, m)
		}
	}
	res.Unmatched.MD = newUnmatchedMD

	newUnmatchedMAL := make([]ImportEntry, 0, len(unmatchedImport))
	for i, entry := range unmatchedImport {
		if _, matched := matchedImportIdx[i]; !matched {
			newUnmatchedMAL = append(newUnmatchedMAL, entry)
		}
	}
	res.Unmatched.Import = newUnmatchedMAL

	return res
}

// distanceThreshold calculates acceptable edit distance (~20% of length)
func distanceThreshold(n int) int {
	th := n / 5
	if th < 1 {
		return 1
	}
	if th > 3 {
		return 3
	}
	return th
}

// filterCandidates pre-filters candidates by length and first rune
func filterCandidates(allTitles []string, pattern string, threshold int) []string {
	if len(allTitles) == 0 {
		return nil
	}

	firstRune := func(s string) rune {
		for _, r := range s {
			return r
		}
		return 0
	}

	fr := firstRune(pattern)
	patLen := len(pattern)

	candidates := make([]string, 0, len(allTitles)/4)
	for _, t := range allTitles {
		// Filter by length window
		if abs(len(t)-patLen) > threshold {
			continue
		}
		// Filter by first character
		if firstRune(t) != fr {
			continue
		}
		candidates = append(candidates, t)
	}

	return candidates
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func SearchAndMatch(ctx context.Context, client *mangadexapi.Client, importEntry ImportEntry, limit int) (*MatchInfo, string, error) {
	if importEntry.Normalized == "" {
		return nil, "", errors.New("No title")
	}

	params := mangadexapi.QueryParams{
		Title: importEntry.Normalized,
		Limit: limit,
		Order: mangadexapi.OrderParams{"relevance": "desc"},
	}
	mangas, err := client.GetMangaList(ctx, params)
	if err != nil {
		return nil, "", err
	}

	if len(mangas) == 0 {
		return nil, "", errors.New("No search results")
	}

	// Exact match
	for _, manga := range mangas {
		for lang, title := range manga.Attributes.Title {
			if isEnglishOrRomanized(lang) && normalizeTitle(title) == importEntry.Normalized {
				return &MatchInfo{
					MangaDexTitle: pickOriginalTitle(manga),
					ImportTitle:   importEntry.Original,
					MatchType:     "exact",
				}, manga.ID, nil
			}
		}
		for _, altTitle := range manga.Attributes.AltTitles {
			for lang, title := range altTitle {
				if isEnglishOrRomanized(lang) && normalizeTitle(title) == importEntry.Normalized {
					return &MatchInfo{
						MangaDexTitle: pickOriginalTitle(manga),
						ImportTitle:   importEntry.Original,
						MatchType:     "exact",
					}, manga.ID, nil
				}
			}
		}
	}

	res, err := fuzzyMatchSingle(importEntry.Normalized, mangas)
	if err == nil {
		return &MatchInfo{
			MangaDexTitle: pickOriginalTitle(*res),
			ImportTitle:   importEntry.Original,
			MatchType:     "fuzzy",
		}, res.ID, nil
	}

	return nil, "", nil
}

func fuzzyMatchSingle(input string, mdList []mangadexapi.Manga) (*mangadexapi.Manga, error) {

	// Build: candidates = []string, owner = map[normalizedTitle][]index
	candidates := []string{}
	owners := make(map[string][]int) // normalized title -> manga indexes

	for i, manga := range mdList {
		// main titles
		for lang, t := range manga.Attributes.Title {
			if !isEnglishOrRomanized(lang) {
				continue
			}
			norm := normalizeTitle(t)
			if norm == "" {
				continue
			}
			candidates = append(candidates, norm)
			owners[norm] = append(owners[norm], i)
		}

		// alt titles
		for _, alt := range manga.Attributes.AltTitles {
			for lang, t := range alt {
				if !isEnglishOrRomanized(lang) {
					continue
				}
				norm := normalizeTitle(t)
				if norm == "" {
					continue
				}
				candidates = append(candidates, norm)
				owners[norm] = append(owners[norm], i)
			}
		}
	}

	thr := distanceThreshold(len(input))

	candidates = filterCandidates(candidates, input, thr)
	if len(candidates) == 0 {
		return nil, nil
	}

	ranks := fuzzy.RankFind(input, candidates)

	best := ranks[0]
	if best.Distance > thr {
		return nil, nil
	}

	idxList := owners[best.Target]

	idx := idxList[0]

	return &mdList[idx], nil
}
