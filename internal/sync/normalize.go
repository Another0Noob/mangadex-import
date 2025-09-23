package sync

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var (
	reNonAlnum   = regexp.MustCompile(`[^a-z0-9\s]+`)
	reMultiSpace = regexp.MustCompile(`\s+`)
)

// stripDiacritics removes combining marks after NFD decomposition.
func stripDiacritics(s string) string {
	decomp := norm.NFD.String(s)
	var b strings.Builder
	b.Grow(len(decomp))
	for _, r := range decomp {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func normalizeTitle(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	// Unicode normalization (NFKC) to fold width/compatibility forms (full‑width, etc.)
	s = norm.NFKC.String(s)

	// Remove diacritics (é -> e, ñ -> n, ō -> o)
	s = stripDiacritics(s)

	s = strings.ToLower(s)

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
	padded := " " + s + " "
	padded = strings.ReplaceAll(padded, " wo ", " o ")
	s = strings.TrimSpace(padded)

	// Token handling
	tokens := strings.Fields(s)
	out := tokens[:0]
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]

		// Collapse any variant of node to no
		if tok == "node" {
			out = append(out, "no")
			continue
		}
		if tok == "no" && i+1 < len(tokens) && tokens[i+1] == "de" {
			out = append(out, "no")
			i++
			continue
		}

		out = append(out, tok)
	}

	s = strings.Join(out, " ")
	s = reMultiSpace.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
