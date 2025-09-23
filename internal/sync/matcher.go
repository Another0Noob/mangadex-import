package sync

import (
	"strings"

	"github.com/Another0Noob/mangadex-import/internal/malparser"
	"github.com/Another0Noob/mangadex-import/internal/mangadexapi"
)

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

// MatchMALTitles compares MAL manga titles against the followed indexes
func MatchMALTitles(followed []mangadexapi.Manga, malManga []malparser.Manga) TitleMatchResult {
	idx := BuildFollowedIndexes(followed)

	matches := make([]malparser.Manga, 0, len(malManga))
	nonMatches := make([]malparser.Manga, 0, len(malManga))

	for _, mm := range malManga {
		n := normalizeTitle(mm.Title)
		if _, ok := idx.MainTitleIndex[n]; ok {
			matches = append(matches, mm)
			continue
		}
		if _, ok := idx.AltTitleIndex[n]; ok {
			matches = append(matches, mm)
			continue
		}
		nonMatches = append(nonMatches, mm)
	}

	return TitleMatchResult{
		Matched:   matches,
		Unmatched: nonMatches,
	}
}
