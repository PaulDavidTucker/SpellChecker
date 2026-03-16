package checker

import (
	"strings"
	"time"

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
	checkedCount := 0
	var prevToken *tokenizer.Token

	for _, tok := range tokens {
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
		Misspellings:  misspellings,
		RepeatedWords: repeatedWords,
		TokenCount:    len(tokens),
		CheckedCount:  checkedCount,
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
