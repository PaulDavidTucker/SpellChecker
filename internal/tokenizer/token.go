package tokenizer

import "fmt"

// TokenKind classifies a token for downstream processing.
type TokenKind int

const (
	TokenWord         TokenKind = iota // Standard word — spell check this
	TokenContraction                   // "don't", "it's" — check as unit
	TokenHyphenated                    // "well-known" — check components
	TokenAcronym                       // "BBC", "HTML" — skip
	TokenNumber                        // "42", "3.14" — skip
	TokenAlphaNumeric                  // "MP3", "h264" — skip
	TokenURL                           // "https://..." — skip
	TokenEmail                         // "a@b.com" — skip
	TokenQuoted                        // "..." or '...' — skip entire quote
	TokenPunctuation                   // ".", "," — skip
	TokenWhitespace                    // " ", "\n" — skip
)

// Token represents a single unit extracted from the input text.
type Token struct {
	Text       string    // Raw text exactly as it appeared
	Normalised string    // Lowercased form for dictionary lookup
	Kind       TokenKind // Classification
	Offset     int       // Byte offset from start of input
	Line       int       // 1-indexed line number
	Column     int       // 1-indexed column number
}

func (k TokenKind) String() string {
	names := [...]string{
		"Word",
		"Contraction",
		"Hyphenated",
		"Acronym",
		"Number",
		"AlphaNumeric",
		"URL",
		"Email",
		"Quoted",
		"Punctuation",
		"Whitespace",
	}
	if int(k) < len(names) {
		return names[k]
	}
	return fmt.Sprintf("Unknown(%d)", k)
}
