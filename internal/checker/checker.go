package checker

import (
	"strings"
	"time"
	"unicode"

	"github.com/PaulDavidTucker/SpellChecker/internal/allowlist"
	"github.com/PaulDavidTucker/SpellChecker/internal/tokenizer"

	symspell "github.com/snapp-incubator/go-symspell"
	"github.com/snapp-incubator/go-symspell/pkg/verbosity"
)

// Checker runs the spell-checking pipeline for a specific
// profile. It is safe for concurrent use — all state is
// read-only after construction.
type Checker struct {
	ss      symspell.SymSpell
	store   *allowlist.Store
	profile string
}

// NewChecker creates a Checker bound to a specific profile
// and SymSpell index.
func NewChecker(
	ss symspell.SymSpell,
	store *allowlist.Store,
	profileID string,
) *Checker {
	return &Checker{
		ss:      ss,
		store:   store,
		profile: profileID,
	}
}

// Check runs the full spell-checking pipeline.
func (c *Checker) Check(req CheckRequest) CheckResult {
	start := time.Now()

	// If no SymSpell index (during startup), return empty results
	if c.ss == nil {
		return CheckResult{
			Misspellings:         []Misspelling{},
			RepeatedWords:        []RepeatedWord{},
			CapitalisationIssues: []CapitalisationIssue{},
			TokenCount:           0,
			CheckedCount:         0,
			ElapsedMs:            0,
		}
	}

	// Build a fast lookup set for request-level ignore terms
	ignoreSet := make(map[string]struct{}, len(req.IgnoreTerms))
	for _, term := range req.IgnoreTerms {
		ignoreSet[strings.ToLower(term)] = struct{}{}
	}

	maxSuggestions := req.MaxSuggestions
	if maxSuggestions <= 0 {
		maxSuggestions = 3
	}

	// Step 1: Tokenize
	scanner := tokenizer.NewScanner(req.Text)
	tokens := scanner.Tokenize()

	// Step 2: Check each token through the pipeline
	var misspellings []Misspelling
	var repeatedWords []RepeatedWord
	var capitalisationIssues []CapitalisationIssue
	checkedCount := 0
	var prevToken *tokenizer.Token
	sentenceJustEnded := true // Start of text is also a sentence start

	for _, tok := range tokens {
		// Check for capitalisation issues - if we're at a sentence start and
		// the word begins with lowercase, flag it (unless it's in allowlist or ignore set)
		if sentenceJustEnded && isCheckable(tok) {
			if isLowercaseWord(tok.Text) {
				// Check if this is a domain-like pattern (e.g., .com) or allowlisted
				if !c.shouldIgnoreCapitalisation(tok, ignoreSet) {
					capitalisationIssues = append(capitalisationIssues, CapitalisationIssue{
						Word:       tok.Text,
						Offset:     tok.Offset,
						Line:       tok.Line,
						Column:     tok.Column,
						Suggestion: capitalizeFirst(tok.Text),
					})
				}
			}
			sentenceJustEnded = false
		}

		// Check for random capital letters in the middle of words (e.g., "camelCase", "teSt")
		if isCheckable(tok) && tok.Kind != tokenizer.TokenHyphenated {
			if hasRandomCapitalLetters(tok.Text) {
				// Suggest the all-lowercase version
				suggestion := strings.ToLower(tok.Text)
				capitalisationIssues = append(capitalisationIssues, CapitalisationIssue{
					Word:       tok.Text,
					Offset:     tok.Offset,
					Line:       tok.Line,
					Column:     tok.Column,
					Suggestion: suggestion,
				})
			}
		}

		// Check for repeated words - compare with previous checkable token
		if isCheckable(tok) && tok.Kind != tokenizer.TokenHyphenated {
			if prevToken != nil && strings.EqualFold(tok.Normalised, prevToken.Normalised) {
				// Found a repeated word
				repeatedWords = append(repeatedWords, RepeatedWord{
					Word:   tok.Text,
					Offset: tok.Offset,
					Line:   tok.Line,
					Column: tok.Column,
				})
			}
			prevToken = &tok
		}

		// Track sentence-ending punctuation
		if tok.Kind == tokenizer.TokenPunctuation && isSentenceEndingPunct(tok.Text) {
			sentenceJustEnded = true
		} else if tok.Kind != tokenizer.TokenWhitespace {
			// Any non-whitespace, non-sentence-ending token resets the flag
			sentenceJustEnded = false
		}

		if !isCheckable(tok) {
			continue
		}

		// Hyphenated tokens: check each component separately
		if tok.Kind == tokenizer.TokenHyphenated {
			results := c.checkHyphenated(
				tok, ignoreSet, maxSuggestions,
			)
			misspellings = append(misspellings, results...)
			checkedCount++
			continue
		}

		// Contractions: try the full form first. If not found,
		// strip the suffix and check the base word.
		lookupWord := tok.Normalised
		if tok.Kind == tokenizer.TokenContraction {
			lookupWord = contractionBase(tok.Normalised)
		}

		checkedCount++

		// Layer 1: Request-level ignore terms
		if _, ignored := ignoreSet[lookupWord]; ignored {
			continue
		}
		// Also check the full contraction form
		if tok.Kind == tokenizer.TokenContraction {
			if _, ignored := ignoreSet[tok.Normalised]; ignored {
				continue
			}
		}

		// Layer 2: Allowlist (profile + base)
		if c.store.IsAllowed(lookupWord, c.profile) {
			continue
		}
		if tok.Kind == tokenizer.TokenContraction {
			if c.store.IsAllowed(tok.Normalised, c.profile) {
				continue
			}
		}

		// Layer 3: SymSpell lookup
		suggestions, err := c.ss.Lookup(
			lookupWord,
			verbosity.All,
			2,
		)
		if err != nil {
			// If SymSpell errors, skip this word rather than
			// returning a false positive
			continue
		}

		// Distance 0 = exact match = word is correct
		if len(suggestions) > 0 && suggestions[0].Distance == 0 {
			continue
		}

		// Check if this could be two words missing a space
		// Try splitting the word at each position and check if both parts are valid
		if splitSuggestion := c.trySplitWord(lookupWord, sentenceJustEnded); splitSuggestion != "" {
			// This is a missing space issue
			ms := Misspelling{
				Word:   tok.Text,
				Offset: tok.Offset,
				Line:   tok.Line,
				Column: tok.Column,
				Type:   "missing_space",
				Suggestions: []Suggestion{
					{
						Word:         splitSuggestion,
						EditDistance: 0,
					},
				},
			}
			misspellings = append(misspellings, ms)
			continue
		}

		// Word is misspelt — collect suggestions
		ms := Misspelling{
			Word:   tok.Text,
			Offset: tok.Offset,
			Line:   tok.Line,
			Column: tok.Column,
		}

		limit := maxSuggestions
		if len(suggestions) < limit {
			limit = len(suggestions)
		}

		for _, s := range suggestions[:limit] {
			ms.Suggestions = append(ms.Suggestions, Suggestion{
				Word:         s.Term,
				EditDistance: s.Distance,
			})
		}

		misspellings = append(misspellings, ms)
	}

	return CheckResult{
		Misspellings:         misspellings,
		RepeatedWords:        repeatedWords,
		CapitalisationIssues: capitalisationIssues,
		TokenCount:           len(tokens),
		CheckedCount:         checkedCount,
		ElapsedMs: float64(
			time.Since(start).Microseconds(),
		) / 1000.0,
	}
}

// checkHyphenated splits a hyphenated token on '-' and checks
// each component independently.
func (c *Checker) checkHyphenated(
	tok tokenizer.Token,
	ignoreSet map[string]struct{},
	maxSuggestions int,
) []Misspelling {
	parts := strings.Split(tok.Text, "-")
	var results []Misspelling

	// Track byte offset within the hyphenated token
	offset := tok.Offset

	for _, part := range parts {
		normalised := strings.ToLower(part)

		if _, ignored := ignoreSet[normalised]; ignored {
			offset += len(part) + 1
			continue
		}

		if c.store.IsAllowed(normalised, c.profile) {
			offset += len(part) + 1
			continue
		}

		suggestions, err := c.ss.Lookup(
			normalised,
			verbosity.All,
			2,
		)
		if err != nil {
			offset += len(part) + 1
			continue
		}

		if len(suggestions) > 0 && suggestions[0].Distance == 0 {
			offset += len(part) + 1
			continue
		}

		ms := Misspelling{
			Word:   part,
			Offset: offset,
			Line:   tok.Line,
			Column: tok.Column + (offset - tok.Offset),
		}

		limit := maxSuggestions
		if len(suggestions) < limit {
			limit = len(suggestions)
		}

		for _, s := range suggestions[:limit] {
			ms.Suggestions = append(ms.Suggestions, Suggestion{
				Word:         s.Term,
				EditDistance: s.Distance,
			})
		}

		results = append(results, ms)
		offset += len(part) + 1
	}

	return results
}

// contractionBase strips common contraction suffixes to get the
// base word for dictionary lookup.
//
// "don't" → "don" (but "don't" itself should be in the dictionary)
// "paul's" → "paul"
// "they're" → "they"
//
// The caller should try the full contraction first, then fall
// back to this.
func contractionBase(normalised string) string {
	suffixes := []string{
		"\u2019s", "'s", // possessive / "is"
		"\u2019t", "'t", // "not"
		"\u2019re", "'re", // "are"
		"\u2019ve", "'ve", // "have"
		"\u2019ll", "'ll", // "will"
		"\u2019d", "'d", // "had" / "would"
		"\u2019m", "'m", // "am"
	}

	for _, suffix := range suffixes {
		if strings.HasSuffix(normalised, suffix) {
			base := strings.TrimSuffix(normalised, suffix)
			if len(base) > 0 {
				return base
			}
		}
	}

	return normalised
}

func isCheckable(tok tokenizer.Token) bool {
	switch tok.Kind {
	case tokenizer.TokenWord,
		tokenizer.TokenContraction,
		tokenizer.TokenHyphenated:
		return true
	default:
		return false
	}
}

// isSentenceEndingPunct returns true if the punctuation marks the end
// of a sentence (. ! ?)
func isSentenceEndingPunct(punct string) bool {
	return punct == "." || punct == "!" || punct == "?"
}

// isLowercaseWord returns true if the word starts with a lowercase letter.
// This handles Unicode properly by checking the first rune.
func isLowercaseWord(word string) bool {
	if word == "" {
		return false
	}
	// Get first rune (handles multi-byte UTF-8)
	for _, r := range word {
		return unicode.IsLower(r)
	}
	return false
}

// hasRandomCapitalLetters checks if a word has capital letters in the middle
// (e.g., "camelCase", "PascalCase", "teSt")
// Returns false for all-caps words like "NASA", "BBC"
func hasRandomCapitalLetters(word string) bool {
	if len(word) <= 1 {
		return false
	}

	// Check for all-caps words
	allCaps := true
	for _, r := range word {
		if unicode.IsLetter(r) && !unicode.IsUpper(r) {
			allCaps = false
			break
		}
	}
	if allCaps {
		return false
	}

	// Check for capitals after the first character
	for i, r := range word {
		if i > 0 && unicode.IsUpper(r) {
			return true
		}
	}

	return false
}

// shouldIgnoreCapitalisation checks if a word should be ignored for
// capitalisation checking. This includes:
// - Words in the ignore set
// - Words in the allowlist
// - Common domain extensions when they follow a dot
func (c *Checker) shouldIgnoreCapitalisation(tok tokenizer.Token, ignoreSet map[string]struct{}) bool {
	// Check ignore set
	if _, ignored := ignoreSet[strings.ToLower(tok.Text)]; ignored {
		return true
	}

	// Check allowlist
	if c.store.IsAllowed(strings.ToLower(tok.Text), c.profile) {
		return true
	}

	return false
}

// trySplitWord attempts to split a word into two valid words (missing space detection)
// Returns the split suggestion with a space if valid, otherwise empty string
// If atSentenceStart is true, capitalizes the first word in the suggestion
// Examples: "helloworld" → "hello world", "spellingmistake" → "spelling mistake"
func (c *Checker) trySplitWord(word string, atSentenceStart bool) string {
	// Need at least 3 characters to split into two words (1+2 or 2+1 minimum)
	if len(word) < 3 {
		return ""
	}

	// Try splitting at every possible position
	for i := 2; i < len(word)-2; i++ {
		first := word[:i]
		second := word[i:]

		// Check if both parts are valid words using symspell
		firstValid := c.isValidWord(first)
		secondValid := c.isValidWord(second)

		if firstValid && secondValid {
			// If at sentence start, capitalize the first word
			if atSentenceStart {
				first = capitalizeFirst(first)
			}
			// Return the split version with a space
			return first + " " + second
		}
	}

	return ""
}

// isValidWord checks if a word exists in the dictionary using symspell
func (c *Checker) isValidWord(word string) bool {
	suggestions, err := c.ss.Lookup(word, verbosity.All, 0)
	if err != nil {
		return false
	}
	// If first suggestion is exact match (distance 0), word is valid
	return len(suggestions) > 0 && suggestions[0].Distance == 0
}

// capitalizeFirst capitalizes the first letter of a word
func capitalizeFirst(word string) string {
	if word == "" {
		return word
	}
	runes := []rune(word)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
