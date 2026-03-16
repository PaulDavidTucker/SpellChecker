package tokenizer

import "regexp"

// SkipZone marks a byte range in the input that should be emitted
// as a single token (URL or email) without further analysis.
type SkipZone struct {
	Start int
	End   int
	Kind  TokenKind
}

var (
	// Matches http://, https://, ftp://, or www. followed by
	// non-whitespace. Deliberately greedy — we want the whole URL
	// including query strings and fragments.
	urlPattern = regexp.MustCompile(
		`(?i)(?:https?://|ftp://|www\.)\S+`,
	)

	// Matches standard email addresses. Not RFC-5322 complete, but
	// covers real-world usage without pathological backtracking.
	emailPattern = regexp.MustCompile(
		`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`,
	)
)

// findSkipZones pre-scans the input for URLs, emails, and quoted strings.
// Returns zones sorted by start offset (inherent from regex
// FindAllStringIndex which scans left to right).
func findSkipZones(text string) []SkipZone {
	var zones []SkipZone

	for _, loc := range urlPattern.FindAllStringIndex(text, -1) {
		zones = append(zones, SkipZone{
			Start: loc[0],
			End:   loc[1],
			Kind:  TokenURL,
		})
	}

	for _, loc := range emailPattern.FindAllStringIndex(text, -1) {
		// Avoid double-matching a URL that contains an @ (rare
		// but possible with mailto: style URLs)
		if !overlapsAny(zones, loc[0], loc[1]) {
			zones = append(zones, SkipZone{
				Start: loc[0],
				End:   loc[1],
				Kind:  TokenEmail,
			})
		}
	}

	// Find quoted strings
	quoteZones := findQuotedStrings(text)
	for _, zone := range quoteZones {
		if !overlapsAny(zones, zone.Start, zone.End) {
			zones = append(zones, zone)
		}
	}

	return zones
}

// findQuotedStrings finds text within matching quote pairs.
// Supports straight quotes: "..." and '...'
// And curly/smart quotes: "..." and '...'
func findQuotedStrings(text string) []SkipZone {
	var zones []SkipZone

	quotePairs := []struct {
		open  rune
		close rune
	}{
		{'"', '"'},           // Straight double quotes
		{'\'', '\''},         // Straight single quotes
		{'\u201C', '\u201D'}, // Curly double quotes: " "
		{'\u2018', '\u2019'}, // Curly single quotes: ' '
	}

	for _, pair := range quotePairs {
		zones = append(zones, findQuoteZones(text, pair.open, pair.close)...)
	}

	return zones
}

// findQuoteZones finds all zones for a specific quote pair
func findQuoteZones(text string, openQuote, closeQuote rune) []SkipZone {
	var zones []SkipZone
	runes := []rune(text)

	for i := 0; i < len(runes); i++ {
		if runes[i] == openQuote {
			// For straight single quotes, be careful not to mistake
			// apostrophes in contractions (like "can't") for quote marks
			if openQuote == '\'' {
				// Check if previous character is a letter (apostrophe in contraction)
				if i > 0 && isLetterOrDigit(runes[i-1]) {
					// This is likely an apostrophe in a contraction, not a quote
					continue
				}
			}

			// Calculate byte offset for start
			startByte := len(string(runes[:i]))

			// Look for the closing quote
			found := false

			for j := i + 1; j < len(runes); j++ {
				// Check for line boundaries (for straight quotes)
				if (openQuote == '"' || openQuote == '\'') &&
					(runes[j] == '\n' || runes[j] == '\r') {
					// Don't cross line boundaries with straight quotes
					break
				}

				// Check if this is the closing quote
				if runes[j] == closeQuote {
					// For straight single quotes, be careful not to mistake
					// apostrophes at the end (like "users'") for quote marks
					if closeQuote == '\'' {
						// Check if next character is a letter (apostrophe in contraction/possessive)
						if j+1 < len(runes) && isLetterOrDigit(runes[j+1]) {
							// This is likely an apostrophe, not a closing quote
							continue
						}
					}

					endByte := len(string(runes[:j+1]))
					zones = append(zones, SkipZone{
						Start: startByte,
						End:   endByte,
						Kind:  TokenQuoted,
					})
					i = j // Skip past this quote
					found = true
					break
				}
			}

			if !found {
				// Unclosed quote - skip it and continue
				continue
			}
		}
	}

	return zones
}

// isLetterOrDigit checks if a rune is a letter or digit
func isLetterOrDigit(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func overlapsAny(zones []SkipZone, start, end int) bool {
	for _, z := range zones {
		if start < z.End && end > z.Start {
			return true
		}
	}
	return false
}
