package sync

import (
	"strings"

	"github.com/Another0Noob/mangadex-import/internal/malparser"
	"github.com/Another0Noob/mangadex-import/internal/mangadexapi"
	"github.com/lithammer/fuzzysearch/fuzzy"
)

type MatchInfo struct {
	MangaDexTitle string
	MALTitle      string
}

// Bundle all remaining/original/normalized data
type Unmatched struct {
	MD        []mangadexapi.Manga // original MD objects
	MAL       []malparser.Manga   // original MAL objects
	MALNorm   []string            // normalized MAL titles (aligned with MAL)
	MDIndexes FollowedIndexes     // normalized MD title indexes for MD
}

type MatchResult struct {
	Matches   map[string]MatchInfo // key: MangaDex ID
	Unmatched Unmatched
}

func isEnglishOrRomanized(lang string) bool {
	if lang == "en" {
		return true
	}
	// romanized languages end with -ro (e.g. ja-ro, ko-ro, zh-ro)
	if strings.HasSuffix(lang, "-ro") {
		return true
	}
	return false
}

// Prefer an original, human-friendly MangaDex title for logging.
func pickOriginalTitle(m mangadexapi.Manga) string {
	attrs := m.Attributes

	// Prefer English primary title if available
	if t, ok := attrs.Title["en"]; ok && t != "" {
		return t
	}

	// Otherwise try romanized (primary titles)
	for lang, t := range attrs.Title {
		if strings.HasSuffix(lang, "-ro") && t != "" {
			return t
		}
	}

	// Then try alt titles: en first, then romanized
	for _, mp := range attrs.AltTitles {
		if t, ok := mp["en"]; ok && t != "" {
			return t
		}
	}
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

// BuildFollowedIndexes builds indexes for titles
func BuildFollowedIndexes(followed []mangadexapi.Manga) FollowedIndexes {
	mainIdx := make(map[string]string, len(followed))
	altIdx := make(map[string]string)
	idToTitles := make(map[string][]string, len(followed))
	seen := make(map[string]struct{})

	for _, m := range followed {
		collected := make([]string, 0, 4)

		// Main titles (map[lang]title)
		for lang, title := range m.Attributes.Title {
			if !isEnglishOrRomanized(lang) {
				continue
			}
			n := normalizeTitle(title)
			mainIdx[n] = m.ID
			collected = append(collected, n)
			seen[n] = struct{}{}
		}

		// Alt titles ([]map[lang]title)
		for _, altMap := range m.Attributes.AltTitles {
			for lang, title := range altMap {
				if !isEnglishOrRomanized(lang) {
					continue
				}
				n := normalizeTitle(title)
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

// buildOwnerSets inverts IDToTitles into normalized-title -> []MangaDexID.
// This avoids silent overwrites when multiple series normalize to the same string.
func buildOwnerSets(idToTitles map[string][]string) map[string][]string {
	owners := make(map[string][]string, len(idToTitles))
	for id, titles := range idToTitles {
		for _, t := range titles {
			owners[t] = append(owners[t], id)
		}
	}
	return owners
}

// MatchMALTitlesDirect returns a single MatchResult with matches and remaining pools.
func MatchDirect(followed []mangadexapi.Manga, malManga []malparser.Manga) MatchResult {
	idx := BuildFollowedIndexes(followed)

	// Fast lookup for full MangaDex object by ID
	mdByID := make(map[string]mangadexapi.Manga, len(followed))
	for _, m := range followed {
		mdByID[m.ID] = m
	}

	// Invert to normalized -> []IDs to handle collisions robustly
	owners := buildOwnerSets(idx.IDToTitles)

	matches := make(map[string]MatchInfo)
	matchedIDs := make(map[string]struct{})
	matchedMALIdx := make(map[int]struct{})

	// Collect exact-title matches (normalized), only when unambiguous
	for i, mm := range malManga {
		n := normalizeTitle(mm.Title)

		if ids := owners[n]; len(ids) == 1 {
			id := ids[0]
			if _, seen := matches[id]; !seen {
				matches[id] = MatchInfo{
					MangaDexTitle: pickOriginalTitle(mdByID[id]),
					MALTitle:      mm.Title,
				}
			}
			matchedIDs[id] = struct{}{}
			matchedMALIdx[i] = struct{}{}
			continue
		}
		// ambiguous (len>1) or no owner (len==0): skip
	}

	// Build remaining FollowedIndexes (remove all entries for matched IDs)
	remainingMain := make(map[string]string, len(idx.MainTitleIndex))
	remainingAlt := make(map[string]string, len(idx.AltTitleIndex))
	remainingIDToTitles := make(map[string][]string, len(idx.IDToTitles))
	seen := make(map[string]struct{})

	for normTitle, id := range idx.MainTitleIndex {
		if _, isMatched := matchedIDs[id]; isMatched {
			continue
		}
		remainingMain[normTitle] = id
		seen[normTitle] = struct{}{}
	}
	for normTitle, id := range idx.AltTitleIndex {
		if _, isMatched := matchedIDs[id]; isMatched {
			continue
		}
		remainingAlt[normTitle] = id
		seen[normTitle] = struct{}{}
	}
	for id, titles := range idx.IDToTitles {
		if _, isMatched := matchedIDs[id]; isMatched {
			continue
		}
		remainingIDToTitles[id] = titles
		for _, t := range titles {
			seen[t] = struct{}{}
		}
	}
	remainingAll := make([]string, 0, len(seen))
	for t := range seen {
		remainingAll = append(remainingAll, t)
	}
	remainingIdx := FollowedIndexes{
		MainTitleIndex: remainingMain,
		AltTitleIndex:  remainingAlt,
		IDToTitles:     remainingIDToTitles,
		AllTitles:      remainingAll,
	}

	// Unmatched MD (original objects)
	unmatchedMD := make([]mangadexapi.Manga, 0, len(followed))
	for _, m := range followed {
		if _, ok := matchedIDs[m.ID]; !ok {
			unmatchedMD = append(unmatchedMD, m)
		}
	}

	// Unmatched MAL (original + normalized titles)
	unmatchedMAL := make([]malparser.Manga, 0, len(malManga))
	unmatchedMALNorm := make([]string, 0, len(malManga))
	for i, mm := range malManga {
		if _, ok := matchedMALIdx[i]; ok {
			continue
		}
		unmatchedMAL = append(unmatchedMAL, mm)
		unmatchedMALNorm = append(unmatchedMALNorm, normalizeTitle(mm.Title))
	}

	return MatchResult{
		Matches: matches,
		Unmatched: Unmatched{
			MD:        unmatchedMD,
			MAL:       unmatchedMAL,
			MALNorm:   unmatchedMALNorm,
			MDIndexes: remainingIdx,
		},
	}
}

// FuzzyMatchRemaining consumes and returns the same MatchResult, adding new fuzzy matches.
func FuzzyMatch(res MatchResult) MatchResult {
	remaining := res.Unmatched.MDIndexes
	unmatchedMD := res.Unmatched.MD
	unmatchedMAL := res.Unmatched.MAL
	unmatchedMALNorm := res.Unmatched.MALNorm

	// Quick exits
	if len(remaining.AllTitles) == 0 || len(unmatchedMALNorm) == 0 {
		return res
	}

	// Build MD lookup by ID from current remaining MD set (use pointers to avoid copying)
	mdByID := make(map[string]*mangadexapi.Manga, len(unmatchedMD))
	for i := range unmatchedMD {
		mdByID[unmatchedMD[i].ID] = &unmatchedMD[i]
	}

	// Invert remaining to normalized -> []IDs to resolve ambiguity
	owners := buildOwnerSets(remaining.IDToTitles)

	// Distance threshold: allow small deviations relative to query length
	distThreshold := func(n int) int {
		// ~20% of length, min 1, max 3
		th := n / 5
		if th < 1 {
			th = 1
		}
		if th > 3 {
			th = 3
		}
		return th
	}
	// Cheap filters to reduce fuzzy candidate set
	firstRune := func(s string) rune {
		for _, r := range s {
			return r
		}
		return 0
	}
	abs := func(x int) int {
		if x < 0 {
			return -x
		}
		return x
	}

	newMatches := make(map[string]MatchInfo)
	matchedIDs := make(map[string]struct{})
	matchedMALIdx := make(map[int]struct{})

	for i, pat := range unmatchedMALNorm {
		if pat == "" {
			continue
		}
		thr := distThreshold(len(pat))
		fr := firstRune(pat)

		// Prefilter by length window and first rune
		cands := make([]string, 0, len(remaining.AllTitles)/4)
		for _, t := range remaining.AllTitles {
			if abs(len(t)-len(pat)) > thr {
				continue
			}
			if firstRune(t) != fr {
				continue
			}
			cands = append(cands, t)
		}
		if len(cands) == 0 {
			continue
		}

		// Ranked candidates among filtered normalized MD titles
		ranks := fuzzy.RankFind(pat, cands)
		if len(ranks) == 0 {
			continue
		}

		best := ranks[0] // sorted by increasing Distance
		if best.Distance > thr {
			continue
		}

		// Map back to MD ID(s) owning this normalized title via owners map
		candNorm := best.Target
		ids := owners[candNorm]
		if len(ids) != 1 {
			continue // ambiguous or none
		}
		id := ids[0]
		if _, already := matchedIDs[id]; already {
			continue
		}

		// Record match
		md := mdByID[id]
		newMatches[id] = MatchInfo{
			MangaDexTitle: pickOriginalTitle(*md),
			MALTitle:      unmatchedMAL[i].Title, // original MAL title
		}
		matchedIDs[id] = struct{}{}
		matchedMALIdx[i] = struct{}{}
	}

	// Merge matches into res
	for id, mi := range newMatches {
		// keep first win if already matched
		if _, exists := res.Matches[id]; !exists {
			res.Matches[id] = mi
		}
	}

	// Remove matched IDs from remaining FollowedIndexes
	remainingMain := make(map[string]string, len(remaining.MainTitleIndex))
	remainingAlt := make(map[string]string, len(remaining.AltTitleIndex))
	remainingIDToTitles := make(map[string][]string, len(remaining.IDToTitles))
	seen := make(map[string]struct{})

	for normTitle, id := range remaining.MainTitleIndex {
		if _, isMatched := matchedIDs[id]; isMatched {
			continue
		}
		remainingMain[normTitle] = id
		seen[normTitle] = struct{}{}
	}
	for normTitle, id := range remaining.AltTitleIndex {
		if _, isMatched := matchedIDs[id]; isMatched {
			continue
		}
		remainingAlt[normTitle] = id
		seen[normTitle] = struct{}{}
	}
	for id, titles := range remaining.IDToTitles {
		if _, isMatched := matchedIDs[id]; isMatched {
			continue
		}
		remainingIDToTitles[id] = titles
		for _, t := range titles {
			seen[t] = struct{}{}
		}
	}
	remainingAll := make([]string, 0, len(seen))
	for t := range seen {
		remainingAll = append(remainingAll, t)
	}

	// Update remaining MD indexes
	res.Unmatched.MDIndexes = FollowedIndexes{
		MainTitleIndex: remainingMain,
		AltTitleIndex:  remainingAlt,
		IDToTitles:     remainingIDToTitles,
		AllTitles:      remainingAll,
	}

	// Filter unmatched MD by matched IDs
	newUnmatchedMD := make([]mangadexapi.Manga, 0, len(unmatchedMD))
	for _, m := range unmatchedMD {
		if _, ok := matchedIDs[m.ID]; !ok {
			newUnmatchedMD = append(newUnmatchedMD, m)
		}
	}
	res.Unmatched.MD = newUnmatchedMD

	// Filter unmatched MAL by matched indices
	newUnmatchedMAL := make([]malparser.Manga, 0, len(unmatchedMAL))
	newUnmatchedMALNorm := make([]string, 0, len(unmatchedMALNorm))
	for i := range unmatchedMAL {
		if _, ok := matchedMALIdx[i]; ok {
			continue
		}
		newUnmatchedMAL = append(newUnmatchedMAL, unmatchedMAL[i])
		newUnmatchedMALNorm = append(newUnmatchedMALNorm, unmatchedMALNorm[i])
	}
	res.Unmatched.MAL = newUnmatchedMAL
	res.Unmatched.MALNorm = newUnmatchedMALNorm

	return res
}
