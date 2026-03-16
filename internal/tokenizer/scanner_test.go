package tokenizer

import (
	"strings"
	"testing"
)

func TestBasicWords(t *testing.T) {
	tokens := NewScanner("hello world").Tokenize()
	words := filterKind(tokens, TokenWord)

	assertTokenTexts(t, words, []string{"hello", "world"})
}

func TestContractions(t *testing.T) {
	tokens := NewScanner("don't can't it's").Tokenize()
	contractions := filterKind(tokens, TokenContraction)

	assertTokenTexts(t, contractions, []string{
		"don't", "can't", "it's",
	})
}

func TestSmartApostrophe(t *testing.T) {
	tokens := NewScanner("don\u2019t").Tokenize()
	contractions := filterKind(tokens, TokenContraction)

	assertTokenTexts(t, contractions, []string{"don\u2019t"})
}

func TestHyphenatedWords(t *testing.T) {
	tokens := NewScanner(
		"well-known state-of-the-art",
	).Tokenize()
	hyphenated := filterKind(tokens, TokenHyphenated)

	assertTokenTexts(t, hyphenated, []string{
		"well-known", "state-of-the-art",
	})
}

func TestDanglingHyphen(t *testing.T) {
	tokens := NewScanner("self- aware").Tokenize()
	words := filterKind(tokens, TokenWord)

	assertTokenTexts(t, words, []string{"self", "aware"})

	// The hyphen should be punctuation
	punct := filterKind(tokens, TokenPunctuation)
	assertTokenTexts(t, punct, []string{"-"})
}

func TestAcronyms(t *testing.T) {
	tokens := NewScanner("BBC HTML UK").Tokenize()
	acronyms := filterKind(tokens, TokenAcronym)

	assertTokenTexts(t, acronyms, []string{"BBC", "HTML", "UK"})
}

func TestPluralisedAcronyms(t *testing.T) {
	tokens := NewScanner("MPs CEOs URLs").Tokenize()
	acronyms := filterKind(tokens, TokenAcronym)

	assertTokenTexts(t, acronyms, []string{"MPs", "CEOs", "URLs"})
}

func TestSingleUppercaseLetterIsWord(t *testing.T) {
	tokens := NewScanner("I saw a dog").Tokenize()
	words := filterKind(tokens, TokenWord)

	assertTokenTexts(t, words, []string{
		"I", "saw", "a", "dog",
	})
}

func TestAlphaNumericLetterFirst(t *testing.T) {
	tokens := NewScanner("MP3 h264").Tokenize()
	alphanum := filterKind(tokens, TokenAlphaNumeric)

	assertTokenTexts(t, alphanum, []string{"MP3", "h264"})
}

func TestAlphaNumericDigitFirst(t *testing.T) {
	tokens := NewScanner("4K 3D").Tokenize()
	alphanum := filterKind(tokens, TokenAlphaNumeric)

	assertTokenTexts(t, alphanum, []string{"4K", "3D"})
}

func TestNumbers(t *testing.T) {
	tokens := NewScanner("42 3.14 1,000").Tokenize()
	nums := filterKind(tokens, TokenNumber)

	assertTokenTexts(t, nums, []string{"42", "3.14", "1,000"})
}

func TestURLs(t *testing.T) {
	input := "Visit https://www.bbc.co.uk/news for updates"
	tokens := NewScanner(input).Tokenize()
	urls := filterKind(tokens, TokenURL)

	assertTokenTexts(t, urls, []string{
		"https://www.bbc.co.uk/news",
	})

	// Words around the URL should still be captured
	words := filterKind(tokens, TokenWord)
	assertTokenTexts(t, words, []string{
		"Visit", "for", "updates",
	})
}

func TestURLWithQueryString(t *testing.T) {
	input := "See https://example.com/path?q=test&lang=en here"
	tokens := NewScanner(input).Tokenize()
	urls := filterKind(tokens, TokenURL)

	assertTokenTexts(t, urls, []string{
		"https://example.com/path?q=test&lang=en",
	})
}

func TestEmails(t *testing.T) {
	input := "Contact paul@bbc.co.uk for info"
	tokens := NewScanner(input).Tokenize()
	emails := filterKind(tokens, TokenEmail)

	assertTokenTexts(t, emails, []string{"paul@bbc.co.uk"})
}

func TestPossessives(t *testing.T) {
	tokens := NewScanner("Paul's book").Tokenize()
	contractions := filterKind(tokens, TokenContraction)

	assertTokenTexts(t, contractions, []string{"Paul's"})
}

func TestMixedContent(t *testing.T) {
	input := "The BBC's MP3 player at " +
		"https://bbc.co.uk costs £42."
	tokens := NewScanner(input).Tokenize()

	words := filterKind(tokens, TokenWord)
	assertTokenTexts(t, words, []string{
		"The", "player", "at", "costs",
	})

	contractions := filterKind(tokens, TokenContraction)
	assertTokenTexts(t, contractions, []string{"BBC's"})

	alphanum := filterKind(tokens, TokenAlphaNumeric)
	assertTokenTexts(t, alphanum, []string{"MP3"})

	urls := filterKind(tokens, TokenURL)
	assertTokenTexts(t, urls, []string{"https://bbc.co.uk"})
}

func TestRealisticArticle(t *testing.T) {
	input := `The government announced a new partnership with France.
MPs debated the bill's implications for the UK's
state-of-the-art NHS systems. Contact info@nhs.uk
or visit https://www.nhs.uk for details.`

	tokens := NewScanner(input).Tokenize()

	words := filterKind(tokens, TokenWord)
	expectedWords := []string{
		"The", "government", "announced", "a", "new",
		"partnership", "with", "France",
		"debated", "the", "implications", "for", "the",
		"systems", "Contact",
		"or", "visit", "for", "details",
	}
	assertTokenTexts(t, words, expectedWords)

	acronyms := filterKind(tokens, TokenAcronym)
	assertTokenTexts(t, acronyms, []string{"MPs", "NHS"})

	contractions := filterKind(tokens, TokenContraction)
	assertTokenTexts(t, contractions, []string{
		"bill's", "UK's",
	})

	hyphenated := filterKind(tokens, TokenHyphenated)
	assertTokenTexts(t, hyphenated, []string{
		"state-of-the-art",
	})

	emails := filterKind(tokens, TokenEmail)
	assertTokenTexts(t, emails, []string{"info@nhs.uk"})

	urls := filterKind(tokens, TokenURL)
	assertTokenTexts(t, urls, []string{"https://www.nhs.uk"})
}

func TestLineAndColumnTracking(t *testing.T) {
	input := "first line\nsecond line\nthird"
	tokens := NewScanner(input).Tokenize()
	words := filterKind(tokens, TokenWord)

	tests := []struct {
		text string
		line int
		col  int
	}{
		{"first", 1, 1},
		{"line", 1, 7},
		{"second", 2, 1},
		{"line", 2, 8},
		{"third", 3, 1},
	}

	if len(words) != len(tests) {
		t.Fatalf("expected %d words, got %d", len(tests), len(words))
	}

	for i, tt := range tests {
		tok := words[i]
		if tok.Text != tt.text {
			t.Errorf(
				"word[%d]: expected %q, got %q",
				i, tt.text, tok.Text,
			)
		}
		if tok.Line != tt.line {
			t.Errorf(
				"word[%d] %q: expected line %d, got %d",
				i, tt.text, tt.line, tok.Line,
			)
		}
		if tok.Column != tt.col {
			t.Errorf(
				"word[%d] %q: expected col %d, got %d",
				i, tt.text, tt.col, tok.Column,
			)
		}
	}
}

func TestByteOffsets(t *testing.T) {
	// "café world" — "é" is 2 bytes in UTF-8
	input := "café world"
	tokens := NewScanner(input).Tokenize()
	words := filterKind(tokens, TokenWord)

	if len(words) != 2 {
		t.Fatalf("expected 2 words, got %d", len(words))
	}

	// "café" starts at byte 0
	if words[0].Offset != 0 {
		t.Errorf(
			"café: expected offset 0, got %d", words[0].Offset,
		)
	}

	// "world" starts at byte 6 (c=1, a=1, f=1, é=2, space=1)
	if words[1].Offset != 6 {
		t.Errorf(
			"world: expected offset 6, got %d", words[1].Offset,
		)
	}
}

func TestNormalisedIsLowercase(t *testing.T) {
	tokens := NewScanner("Hello WORLD don't Well-Known").Tokenize()
	checkable := filterCheckable(tokens)

	for _, tok := range checkable {
		if tok.Normalised == "" {
			t.Errorf("token %q has empty Normalised", tok.Text)
		}
		if tok.Normalised != toLower(tok.Normalised) {
			t.Errorf(
				"token %q Normalised %q contains uppercase",
				tok.Text,
				tok.Normalised,
			)
		}
	}
}

func TestEmptyInput(t *testing.T) {
	tokens := NewScanner("").Tokenize()
	if len(tokens) != 0 {
		t.Errorf("expected 0 tokens, got %d", len(tokens))
	}
}

func TestWhitespaceOnly(t *testing.T) {
	tokens := NewScanner("   \n\t  ").Tokenize()
	for _, tok := range tokens {
		if tok.Kind != TokenWhitespace {
			t.Errorf(
				"expected whitespace, got %s: %q",
				tok.Kind, tok.Text,
			)
		}
	}
}

func TestPunctuationSurroundingWords(t *testing.T) {
	// Words in parentheses and brackets should be tokenized
	// Words in quotes are now treated as Quoted tokens (skipped)
	tokens := NewScanner(`"hello" (world) [test]`).Tokenize()
	words := filterKind(tokens, TokenWord)

	// Only "world" and "test" are words - "hello" is in quotes
	assertTokenTexts(t, words, []string{
		"world", "test",
	})

	// Verify "hello" is a Quoted token
	quoted := filterKind(tokens, TokenQuoted)
	if len(quoted) != 1 || quoted[0].Text != `"hello"` {
		t.Errorf("expected '\"hello\"' as Quoted token, got %v", quoted)
	}
}

func TestMultiplePunctuationBetweenWords(t *testing.T) {
	tokens := NewScanner("end... Start").Tokenize()
	words := filterKind(tokens, TokenWord)

	assertTokenTexts(t, words, []string{"end", "Start"})
}

func TestTrailingPossessive(t *testing.T) {
	// "workers' rights" — apostrophe at end, no letters follow
	tokens := NewScanner("workers' rights").Tokenize()
	words := filterKind(tokens, TokenWord)

	// "workers" should be a plain word, apostrophe is punctuation
	assertTokenTexts(t, words, []string{"workers", "rights"})
}

func TestDoubleQuotedStrings(t *testing.T) {
	// Simple quoted string
	tokens := NewScanner(`He said "hello world" to me`).Tokenize()

	// Should have a quoted token
	quoted := filterKind(tokens, TokenQuoted)
	if len(quoted) != 1 {
		t.Fatalf("expected 1 quoted token, got %d", len(quoted))
	}
	if quoted[0].Text != `"hello world"` {
		t.Errorf("expected quoted text to be '\"hello world\"', got %q", quoted[0].Text)
	}

	// Words outside quotes should be checkable
	words := filterCheckable(tokens)
	assertTokenTexts(t, words, []string{"He", "said", "to", "me"})
}

func TestSingleQuotedStrings(t *testing.T) {
	// Single-quoted string
	tokens := NewScanner("The word 'test' is quoted").Tokenize()

	quoted := filterKind(tokens, TokenQuoted)
	if len(quoted) != 1 {
		t.Fatalf("expected 1 quoted token, got %d", len(quoted))
	}
	if quoted[0].Text != "'test'" {
		t.Errorf("expected quoted text to be \"'test'\", got %q", quoted[0].Text)
	}
}

func TestCurlyQuotedStrings(t *testing.T) {
	// Curly/smart quotes (using actual Unicode characters)
	// \u201C = left double quotation mark ("), \u201D = right double quotation mark (")
	tokens := NewScanner("He said \u201Chello world\u201D to me").Tokenize()

	quoted := filterKind(tokens, TokenQuoted)
	if len(quoted) != 1 {
		t.Fatalf("expected 1 quoted token with curly quotes, got %d", len(quoted))
	}
	if quoted[0].Text != "\u201Chello world\u201D" {
		t.Errorf("expected quoted text with curly quotes, got %q", quoted[0].Text)
	}
}

func TestMultipleQuotedStrings(t *testing.T) {
	// Multiple quoted strings in one sentence
	tokens := NewScanner(`He said "hello" and "goodbye"`).Tokenize()

	quoted := filterKind(tokens, TokenQuoted)
	if len(quoted) != 2 {
		t.Fatalf("expected 2 quoted tokens, got %d", len(quoted))
	}
	assertTokenTexts(t, quoted, []string{`"hello"`, `"goodbye"`})
}

func TestQuotedStringWithMisspellings(t *testing.T) {
	// Misspellings inside quotes should be skipped
	tokens := NewScanner(`The "goverment announcd" is wrong`).Tokenize()

	// Should have one quoted token containing both misspellings
	quoted := filterKind(tokens, TokenQuoted)
	if len(quoted) != 1 {
		t.Fatalf("expected 1 quoted token, got %d", len(quoted))
	}
	if quoted[0].Text != `"goverment announcd"` {
		t.Errorf("expected '\"goverment announcd\"', got %q", quoted[0].Text)
	}

	// Words outside should be checked
	words := filterCheckable(tokens)
	assertTokenTexts(t, words, []string{"The", "is", "wrong"})
}

func TestEmptyQuotedString(t *testing.T) {
	// Empty quotes
	tokens := NewScanner(`He said ""`).Tokenize()

	quoted := filterKind(tokens, TokenQuoted)
	if len(quoted) != 1 {
		t.Fatalf("expected 1 quoted token, got %d", len(quoted))
	}
	if quoted[0].Text != `""` {
		t.Errorf("expected empty quotes '\"\"', got %q", quoted[0].Text)
	}
}

func TestQuoteNotClosingSameLine(t *testing.T) {
	// Straight quotes shouldn't cross line boundaries
	tokens := NewScanner("He said \"hello\nworld").Tokenize()

	// Should NOT have a quoted token since closing quote is on next line
	quoted := filterKind(tokens, TokenQuoted)
	if len(quoted) != 0 {
		t.Errorf("expected 0 quoted tokens (straight quotes don't cross lines), got %d", len(quoted))
	}
}

func TestQuoteWithPunctuation(t *testing.T) {
	// Quote containing punctuation
	tokens := NewScanner(`The "test, with commas!" is quoted`).Tokenize()

	quoted := filterKind(tokens, TokenQuoted)
	if len(quoted) != 1 {
		t.Fatalf("expected 1 quoted token, got %d", len(quoted))
	}
	if quoted[0].Text != `"test, with commas!"` {
		t.Errorf("expected '\"test, with commas!\"', got %q", quoted[0].Text)
	}
}

// --- Helpers ---

func filterKind(tokens []Token, kind TokenKind) []Token {
	var result []Token
	for _, t := range tokens {
		if t.Kind == kind {
			result = append(result, t)
		}
	}
	return result
}

func filterCheckable(tokens []Token) []Token {
	var result []Token
	for _, t := range tokens {
		switch t.Kind {
		case TokenWord, TokenContraction, TokenHyphenated:
			result = append(result, t)
		}
	}
	return result
}

func assertTokenTexts(
	t *testing.T,
	tokens []Token,
	expected []string,
) {
	t.Helper()
	if len(tokens) != len(expected) {
		texts := make([]string, len(tokens))
		for i, tok := range tokens {
			texts[i] = tok.Text
		}
		t.Fatalf(
			"expected %d tokens %v, got %d: %v",
			len(expected), expected, len(tokens), texts,
		)
	}
	for i, tok := range tokens {
		if tok.Text != expected[i] {
			t.Errorf(
				"token[%d]: expected %q, got %q",
				i, expected[i], tok.Text,
			)
		}
	}
}

func toLower(s string) string {
	result := make([]rune, len([]rune(s)))
	for i, r := range []rune(s) {
		if r >= 'A' && r <= 'Z' {
			result[i] = r + 32
		} else {
			result[i] = r
		}
	}
	return string(result)
}

func BenchmarkTokenize(b *testing.B) {
	input := `The government announced a new partnership with
France. MPs debated the bill's implications for the UK's
state-of-the-art NHS systems. Visit https://www.nhs.uk
or email info@nhs.uk for more information about the
well-established programme.`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewScanner(input).Tokenize()
	}
}

func BenchmarkTokenizeLong(b *testing.B) {
	// Simulate a ~1000 word article by repeating
	paragraph := `The government announced a new partnership with
France. MPs debated the bill's implications for the UK's
state-of-the-art NHS systems. `

	var builder strings.Builder
	for i := 0; i < 50; i++ {
		builder.WriteString(paragraph)
	}
	input := builder.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewScanner(input).Tokenize()
	}
}
