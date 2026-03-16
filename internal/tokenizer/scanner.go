package tokenizer

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Scanner tokenizes input text using a single-pass state machine.
type Scanner struct {
	input     string // Original input (kept as string for byte offsets)
	runes     []rune // Rune slice for character-by-character iteration
	pos       int    // Current index into runes
	bytePos   int    // Current byte offset into input
	line      int    // Current line number (1-indexed)
	col       int    // Current column number (1-indexed)
	skipZones []SkipZone
}

func NewScanner(text string) *Scanner {
	return &Scanner{
		input:     text,
		runes:     []rune(text),
		pos:       0,
		bytePos:   0,
		line:      1,
		col:       1,
		skipZones: findSkipZones(text),
	}
}

// Tokenize returns all tokens in the input.
func (s *Scanner) Tokenize() []Token {
	// Pre-allocate — a rough estimate is one token per 5 characters
	// (word + space). Over-allocating slightly is cheaper than
	// repeated slice growth.
	tokens := make([]Token, 0, len(s.runes)/5+1)

	for {
		tok, ok := s.next()
		if !ok {
			break
		}
		tokens = append(tokens, tok)
	}

	return tokens
}

func (s *Scanner) next() (Token, bool) {
	if s.pos >= len(s.runes) {
		return Token{}, false
	}

	// Check if we're inside a skip zone (URL/email)
	if zone, ok := s.inSkipZone(); ok {
		return s.consumeSkipZone(zone), true
	}

	r := s.runes[s.pos]

	switch {
	case unicode.IsLetter(r):
		return s.scanWord(), true

	case unicode.IsDigit(r):
		return s.scanNumber(), true

	case unicode.IsSpace(r):
		return s.scanWhitespace(), true

	default:
		// Single punctuation character
		return s.scanPunctuation(), true
	}
}

// scanWord handles the most complex case: words, contractions,
// hyphenated compounds, acronyms, and alphanumeric tokens.
//
// It accumulates characters and makes classification decisions
// based on what it encounters.
func (s *Scanner) scanWord() Token {
	startPos := s.pos
	startByte := s.bytePos
	startLine := s.line
	startCol := s.col

	hasLower := false
	hasUpper := false
	hasDigit := false
	hasApostrophe := false
	hasHyphen := false

	// Accumulate the token
	for s.pos < len(s.runes) {
		r := s.runes[s.pos]

		if unicode.IsLetter(r) {
			if unicode.IsLower(r) {
				hasLower = true
			} else {
				hasUpper = true
			}
			s.advance()
			continue
		}

		if unicode.IsDigit(r) {
			hasDigit = true
			s.advance()
			continue
		}

		// Apostrophe handling: include only if letters follow
		if isApostrophe(r) && s.lettersFollow(s.pos+1) {
			hasApostrophe = true
			s.advance()
			continue
		}

		// Hyphen handling: include only if letters follow
		if r == '-' && s.lettersFollow(s.pos+1) {
			hasHyphen = true
			s.advance()
			continue
		}

		// Any other character ends the word
		break
	}

	raw := string(s.runes[startPos:s.pos])

	kind := classifyWord(raw, hasLower, hasUpper, hasDigit,
		hasApostrophe, hasHyphen)

	normalised := ""
	if kind == TokenWord || kind == TokenContraction ||
		kind == TokenHyphenated {
		normalised = strings.ToLower(raw)
	}

	return Token{
		Text:       raw,
		Normalised: normalised,
		Kind:       kind,
		Offset:     startByte,
		Line:       startLine,
		Column:     startCol,
	}
}

// classifyWord determines the TokenKind based on what characters
// were found during scanning.
func classifyWord(
	text string,
	hasLower, hasUpper, hasDigit, hasApostrophe, hasHyphen bool,
) TokenKind {
	// Alphanumeric: contains both letters and digits
	if hasDigit {
		return TokenAlphaNumeric
	}

	// Contraction: contains an apostrophe
	if hasApostrophe {
		return TokenContraction
	}

	// Hyphenated compound
	if hasHyphen {
		return TokenHyphenated
	}

	// Acronym detection: all uppercase and at least 2 characters.
	// Also catches pluralised acronyms like "MPs", "CEOs" — all
	// uppercase except a trailing lowercase 's'.
	if hasUpper && !hasLower && len([]rune(text)) >= 2 {
		return TokenAcronym
	}

	if hasUpper && hasLower && len([]rune(text)) >= 3 {
		// Check for pluralised acronym: "MPs", "URLs", "CEOs"
		runes := []rune(text)
		if runes[len(runes)-1] == 's' {
			allUpperExceptLast := true
			for _, r := range runes[:len(runes)-1] {
				if !unicode.IsUpper(r) {
					allUpperExceptLast = false
					break
				}
			}
			if allUpperExceptLast {
				return TokenAcronym
			}
		}
	}

	return TokenWord
}

func (s *Scanner) scanNumber() Token {
	startPos := s.pos
	startByte := s.bytePos
	startLine := s.line
	startCol := s.col

	for s.pos < len(s.runes) {
		r := s.runes[s.pos]
		// Allow digits, decimal points (if followed by digit),
		// and commas in numbers (e.g. "1,000")
		if unicode.IsDigit(r) {
			s.advance()
			continue
		}
		if (r == '.' || r == ',') && s.digitFollows(s.pos+1) {
			s.advance()
			continue
		}
		// If we hit a letter, this is actually alphanumeric — but
		// the letter-first case is handled by scanWord. This only
		// triggers for digit-first like "4K", "3D".
		if unicode.IsLetter(r) {
			// Consume remaining alphanumeric chars
			for s.pos < len(s.runes) &&
				(unicode.IsLetter(s.runes[s.pos]) ||
					unicode.IsDigit(s.runes[s.pos])) {
				s.advance()
			}
			return Token{
				Text:   string(s.runes[startPos:s.pos]),
				Kind:   TokenAlphaNumeric,
				Offset: startByte,
				Line:   startLine,
				Column: startCol,
			}
		}
		break
	}

	return Token{
		Text:   string(s.runes[startPos:s.pos]),
		Kind:   TokenNumber,
		Offset: startByte,
		Line:   startLine,
		Column: startCol,
	}
}

func (s *Scanner) scanWhitespace() Token {
	startPos := s.pos
	startByte := s.bytePos
	startLine := s.line
	startCol := s.col

	for s.pos < len(s.runes) && unicode.IsSpace(s.runes[s.pos]) {
		s.advance()
	}

	return Token{
		Text:   string(s.runes[startPos:s.pos]),
		Kind:   TokenWhitespace,
		Offset: startByte,
		Line:   startLine,
		Column: startCol,
	}
}

func (s *Scanner) scanPunctuation() Token {
	startByte := s.bytePos
	startLine := s.line
	startCol := s.col
	r := s.runes[s.pos]
	s.advance()

	return Token{
		Text:   string(r),
		Kind:   TokenPunctuation,
		Offset: startByte,
		Line:   startLine,
		Column: startCol,
	}
}

// consumeSkipZone emits the entire skip zone as a single token
// and advances the scanner past it.
func (s *Scanner) consumeSkipZone(zone SkipZone) Token {
	startLine := s.line
	startCol := s.col

	// Advance rune-by-rune through the zone to keep line/col
	// tracking accurate
	for s.bytePos < zone.End && s.pos < len(s.runes) {
		s.advance()
	}

	return Token{
		Text:   s.input[zone.Start:zone.End],
		Kind:   zone.Kind,
		Offset: zone.Start,
		Line:   startLine,
		Column: startCol,
	}
}

// --- Helper methods ---

// advance moves the scanner forward by one rune, tracking byte
// offset, line number, and column.
func (s *Scanner) advance() {
	if s.pos >= len(s.runes) {
		return
	}

	r := s.runes[s.pos]

	// Track line and column
	if r == '\n' {
		s.line++
		s.col = 1
	} else {
		s.col++
	}

	// Track byte offset — runes can be 1-4 bytes in UTF-8
	s.bytePos += utf8.RuneLen(r)
	s.pos++
}

// lettersFollow returns true if position i starts with at least
// one letter. Used for apostrophe/hyphen decisions.
func (s *Scanner) lettersFollow(i int) bool {
	return i < len(s.runes) && unicode.IsLetter(s.runes[i])
}

// digitFollows returns true if position i is a digit.
func (s *Scanner) digitFollows(i int) bool {
	return i < len(s.runes) && unicode.IsDigit(s.runes[i])
}

// inSkipZone checks if the current byte position falls within a
// pre-identified URL or email zone.
func (s *Scanner) inSkipZone() (SkipZone, bool) {
	for _, zone := range s.skipZones {
		if s.bytePos >= zone.Start && s.bytePos < zone.End {
			return zone, true
		}
	}
	return SkipZone{}, false
}

// isApostrophe matches ASCII apostrophe and Unicode right single
// quotation mark (commonly used as a typographic apostrophe in
// CMS content).
func isApostrophe(r rune) bool {
	return r == '\'' || r == '\u2019'
}
