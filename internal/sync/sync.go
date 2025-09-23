package sync

import (
	"github.com/Another0Noob/mangadex-import/internal/malparser"

	"github.com/Another0Noob/mangadex-import/internal/mangadexapi"
)

func buildFollowedSets(followed []mangadexapi.Manga) (ids map[string]struct{}, titles map[string]struct{}) {
	ids = make(map[string]struct{}, len(followed))
	titles = make(map[string]struct{})
	for _, m := range followed {
		ids[m.ID] = struct{}{}
		for _, v := range m.Attributes.Title {
			titles[normalizeTitle(v)] = struct{}{}
		}
		for _, alt := range m.Attributes.AltTitles {
			for _, v := range alt {
				titles[normalizeTitle(v)] = struct{}{}
			}
		}
	}
	return
}

func FilterUnfollowedByTitle(followed []mangadexapi.Manga, candidates []malparser.Manga) []malparser.Manga {
	_, titleSet := buildFollowedSets(followed)
	out := make([]malparser.Manga, 0, len(candidates))
	for _, c := range candidates {
		if _, ok := titleSet[normalizeTitle(c.Title)]; !ok {
			out = append(out, c)
		}
	}
	return out
}

func BuildFollowedIndexes(followed []mangadexapi.Manga) FollowedIndexes {
	mainIdx := make(map[string]string, len(followed))
	altIdx := make(map[string]string)
	idToTitles := make(map[string][]string, len(followed))
	seen := make(map[string]struct{})

	for _, m := range followed {
		all := make([]string, 0, 8)

		for _, v := range m.Attributes.Title {
			n := normalizeTitle(v)
			mainIdx[n] = m.ID
			all = append(all, n)
			seen[n] = struct{}{}
		}
		for _, alt := range m.Attributes.AltTitles {
			for _, v := range alt {
				n := normalizeTitle(v)
				altIdx[n] = m.ID
				all = append(all, n)
				seen[n] = struct{}{}
			}
		}
		idToTitles[m.ID] = all
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
