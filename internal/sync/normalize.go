package sync

import (
	"regexp"
	"strings"
)

var (
	reNonAlnum   = regexp.MustCompile(`[^a-z0-9\s]+`)
	reMultiSpace = regexp.MustCompile(`\s+`)
)

func normalizeTitle(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))

	// Remove special suffixes before stripping punctuation
	for _, r := range []string{
		"@comic",
		"the comic",
		"(comic)",
		"(manga)",
	} {
		s = strings.ReplaceAll(s, r, "")
	}

	// Remove all non-alphanumeric (keep spaces)
	s = reNonAlnum.ReplaceAllString(s, " ")

	// Normalize 'wo' particle to 'o'
	// Pad to simplify whole-word replacement
	padded := " " + s + " "
	padded = strings.ReplaceAll(padded, " wo ", " o ")
	s = strings.TrimSpace(padded)

	// Token handling
	tokens := strings.Fields(s)
	out := tokens[:0]
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]

		// Collapse any variant of node to no:
		// 1) "node" => "no"
		// 2) "no" "de" => "no" (skip "de")
		if tok == "node" {
			out = append(out, "no")
			continue
		}
		if tok == "no" && i+1 < len(tokens) && tokens[i+1] == "de" {
			out = append(out, "no")
			i++ // skip "de"
			continue
		}

		out = append(out, tok)
	}

	s = strings.Join(out, " ")
	s = reMultiSpace.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
