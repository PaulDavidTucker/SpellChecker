package checker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PaulDavidTucker/SpellChecker/internal/allowlist"
)

// buildTestDict creates a small dictionary file for testing.
// Returns the path to the file.
func buildTestDict(t *testing.T, dir string) string {
	t.Helper()

	// word frequency format — one per line, space-separated
	words := []string{
		"the 100000",
		"government 50000",
		"announced 40000",
		"a 90000",
		"new 60000",
		"partnership 30000",
		"with 80000",
		"france 20000",
		"receive 35000",
		"their 70000",
		"there 65000",
		"they 68000",
		"programme 25000",
		"well 45000",
		"known 42000",
		"state 38000",
		"art 15000",
		"of 95000",
		"don't 55000",
		"it's 52000",
		"can't 48000",
		"colour 10000",
		"favour 9000",
		"hello 20000",
		"world 30000",
		"test 25000",
	}

	path := filepath.Join(dir, "test_dict.txt")
	content := strings.Join(words, "\n") + "\n"
	if err := os.WriteFile(
		path, []byte(content), 0644,
	); err != nil {
		t.Fatalf("failed to write test dict: %v", err)
	}

	return path
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(
		path, []byte(content), 0644,
	); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

// setupTestPool creates a Pool with a small dictionary and
// optional profiles for testing.
func setupTestPool(t *testing.T) (*Pool, string) {
	t.Helper()
	dir := t.TempDir()

	dictPath := buildTestDict(t, dir)

	profilesDir := filepath.Join(dir, "profiles")
	os.MkdirAll(profilesDir, 0755)

	basePath := filepath.Join(dir, "base.yaml")
	writeTestFile(t, basePath, "terms: []")

	writeTestFile(
		t,
		filepath.Join(profilesDir, "test-profile.yaml"),
		`
profile_id: test-profile
description: "Test profile"
terms:
  - Starmer
  - Ofcom
`,
	)

	store, err := allowlist.NewStore(basePath, profilesDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	pool, err := NewPool(dictPath, store)
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}

	return pool, dir
}

func TestCorrectWordsProduceNoMisspellings(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	result := checker.Check(CheckRequest{
		Text:      "the government announced a new partnership",
		ProfileID: "test-profile",
	})

	if len(result.Misspellings) != 0 {
		t.Errorf(
			"expected 0 misspellings, got %d: %+v",
			len(result.Misspellings),
			result.Misspellings,
		)
	}
}

func TestMisspelledWordDetected(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	result := checker.Check(CheckRequest{
		Text:      "the goverment announced a new partnership",
		ProfileID: "test-profile",
	})

	if len(result.Misspellings) != 1 {
		t.Fatalf(
			"expected 1 misspelling, got %d: %+v",
			len(result.Misspellings),
			result.Misspellings,
		)
	}

	ms := result.Misspellings[0]
	if ms.Word != "goverment" {
		t.Errorf("expected 'goverment', got %q", ms.Word)
	}

	// Should suggest "government"
	if len(ms.Suggestions) == 0 {
		t.Fatal("expected at least 1 suggestion")
	}
	if ms.Suggestions[0].Word != "government" {
		t.Errorf(
			"expected suggestion 'government', got %q",
			ms.Suggestions[0].Word,
		)
	}
}

func TestMultipleMisspellings(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	result := checker.Check(CheckRequest{
		Text:      "the goverment anounced a new partnrship",
		ProfileID: "test-profile",
	})

	// "goverment", "anounced", "partnrship" should all be caught
	if len(result.Misspellings) < 2 {
		t.Errorf(
			"expected at least 2 misspellings, got %d: %+v",
			len(result.Misspellings),
			result.Misspellings,
		)
	}

	misspelled := make(map[string]struct{})
	for _, ms := range result.Misspellings {
		misspelled[ms.Word] = struct{}{}
	}

	if _, ok := misspelled["goverment"]; !ok {
		t.Error("expected 'goverment' to be flagged")
	}
	if _, ok := misspelled["anounced"]; !ok {
		t.Error("expected 'anounced' to be flagged")
	}
}

func TestIgnoreTermsSkipsWords(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	// "buildx" isn't in the dictionary, but it's in ignore_terms
	result := checker.Check(CheckRequest{
		Text:        "use buildx to build images",
		ProfileID:   "test-profile",
		IgnoreTerms: []string{"buildx"},
	})

	for _, ms := range result.Misspellings {
		if ms.Word == "buildx" {
			t.Error("expected 'buildx' to be ignored")
		}
	}
}

func TestIgnoreTermsCaseInsensitive(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	result := checker.Check(CheckRequest{
		Text:        "use Buildx to build images",
		ProfileID:   "test-profile",
		IgnoreTerms: []string{"buildx"},
	})

	for _, ms := range result.Misspellings {
		if strings.EqualFold(ms.Word, "buildx") {
			t.Error("expected 'Buildx' to be ignored")
		}
	}
}

func TestAllowlistTermsNotFlagged(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	// "Starmer" is in the test-profile allowlist
	result := checker.Check(CheckRequest{
		Text:      "Starmer announced a new programme",
		ProfileID: "test-profile",
	})

	for _, ms := range result.Misspellings {
		if ms.Word == "Starmer" {
			t.Error("expected 'Starmer' to be allowed by profile")
		}
	}
}

func TestFallbackCheckerUsedForUnknownProfile(t *testing.T) {
	pool, _ := setupTestPool(t)

	// "nonexistent" profile should fall back to base checker
	checker := pool.Get("nonexistent")

	result := checker.Check(CheckRequest{
		Text:      "the government announced a new partnership",
		ProfileID: "nonexistent",
	})

	if len(result.Misspellings) != 0 {
		t.Errorf(
			"expected 0 misspellings with fallback, got %d",
			len(result.Misspellings),
		)
	}
}

func TestMaxSuggestionsRespected(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	result := checker.Check(CheckRequest{
		Text:           "the goverment announced",
		ProfileID:      "test-profile",
		MaxSuggestions: 1,
	})

	for _, ms := range result.Misspellings {
		if len(ms.Suggestions) > 1 {
			t.Errorf(
				"expected max 1 suggestion, got %d",
				len(ms.Suggestions),
			)
		}
	}
}

func TestDefaultMaxSuggestions(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	// MaxSuggestions=0 should default to 3
	result := checker.Check(CheckRequest{
		Text:           "the goverment announced",
		ProfileID:      "test-profile",
		MaxSuggestions: 0,
	})

	for _, ms := range result.Misspellings {
		if len(ms.Suggestions) > 3 {
			t.Errorf(
				"expected max 3 suggestions (default), got %d",
				len(ms.Suggestions),
			)
		}
	}
}

func TestPositionTracking(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	result := checker.Check(CheckRequest{
		Text:      "the goverment",
		ProfileID: "test-profile",
	})

	if len(result.Misspellings) != 1 {
		t.Fatalf(
			"expected 1 misspelling, got %d",
			len(result.Misspellings),
		)
	}

	ms := result.Misspellings[0]

	// "goverment" starts at byte offset 4 ("the " = 4 bytes)
	if ms.Offset != 4 {
		t.Errorf("expected offset 4, got %d", ms.Offset)
	}
	if ms.Line != 1 {
		t.Errorf("expected line 1, got %d", ms.Line)
	}
	if ms.Column != 5 {
		t.Errorf("expected column 5, got %d", ms.Column)
	}
}

func TestMultilinePositionTracking(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	result := checker.Check(CheckRequest{
		Text:      "the government\ngoverment announced",
		ProfileID: "test-profile",
	})

	if len(result.Misspellings) != 1 {
		t.Fatalf(
			"expected 1 misspelling, got %d",
			len(result.Misspellings),
		)
	}

	ms := result.Misspellings[0]
	if ms.Line != 2 {
		t.Errorf("expected line 2, got %d", ms.Line)
	}
	if ms.Column != 1 {
		t.Errorf("expected column 1, got %d", ms.Column)
	}
}

func TestEmptyTextReturnsNoResults(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	result := checker.Check(CheckRequest{
		Text:      "",
		ProfileID: "test-profile",
	})

	if len(result.Misspellings) != 0 {
		t.Errorf("expected 0 misspellings for empty text")
	}
	if result.TokenCount != 0 {
		t.Errorf("expected 0 tokens for empty text")
	}
}

func TestTokenAndCheckedCounts(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	result := checker.Check(CheckRequest{
		Text:      "the government announced",
		ProfileID: "test-profile",
	})

	// 3 words + 2 whitespace = 5 tokens
	if result.TokenCount < 3 {
		t.Errorf(
			"expected at least 3 tokens, got %d",
			result.TokenCount,
		)
	}

	// 3 words checked
	if result.CheckedCount != 3 {
		t.Errorf(
			"expected 3 checked, got %d",
			result.CheckedCount,
		)
	}
}

func TestElapsedMsPopulated(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	result := checker.Check(CheckRequest{
		Text:      "the government announced",
		ProfileID: "test-profile",
	})

	if result.ElapsedMs <= 0 {
		t.Error("expected ElapsedMs > 0")
	}
}

func TestContractionBase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"don't", "don"},
		{"it's", "it"},
		{"they're", "they"},
		{"we've", "we"},
		{"i'll", "i"},
		{"he'd", "he"},
		{"i'm", "i"},
		{"hello", "hello"}, // no contraction suffix
	}

	for _, tt := range tests {
		got := contractionBase(tt.input)
		if got != tt.expected {
			t.Errorf(
				"contractionBase(%q): got %q, want %q",
				tt.input, got, tt.expected,
			)
		}
	}
}

func TestRepeatedWords(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	result := checker.Check(CheckRequest{
		Text:      "the the government announced a new partnership",
		ProfileID: "test-profile",
	})

	if len(result.RepeatedWords) != 1 {
		t.Fatalf("Expected 1 repeated word, got %d", len(result.RepeatedWords))
	}

	rw := result.RepeatedWords[0]
	if rw.Word != "the" {
		t.Errorf("Expected repeated word 'the', got %s", rw.Word)
	}

	// Should be at the second "the" (second occurrence)
	if rw.Line != 1 {
		t.Errorf("Expected line 1, got %d", rw.Line)
	}
}

func TestMultipleRepeatedWords(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	result := checker.Check(CheckRequest{
		Text:      "the the quick quick brown fox",
		ProfileID: "test-profile",
	})

	if len(result.RepeatedWords) != 2 {
		t.Fatalf("Expected 2 repeated words, got %d", len(result.RepeatedWords))
	}

	// Check first repeated word
	if result.RepeatedWords[0].Word != "the" {
		t.Errorf("Expected first repeated word 'the', got %s", result.RepeatedWords[0].Word)
	}

	// Check second repeated word
	if result.RepeatedWords[1].Word != "quick" {
		t.Errorf("Expected second repeated word 'quick', got %s", result.RepeatedWords[1].Word)
	}
}

func TestNoRepeatedWords(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	result := checker.Check(CheckRequest{
		Text:      "the government announced a new partnership",
		ProfileID: "test-profile",
	})

	if len(result.RepeatedWords) != 0 {
		t.Errorf("Expected 0 repeated words, got %d", len(result.RepeatedWords))
	}
}

func TestCaseInsensitiveRepeatedWords(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	result := checker.Check(CheckRequest{
		Text:      "The the government announced a new partnership",
		ProfileID: "test-profile",
	})

	if len(result.RepeatedWords) != 1 {
		t.Fatalf("Expected 1 repeated word (case-insensitive), got %d", len(result.RepeatedWords))
	}

	if result.RepeatedWords[0].Word != "the" {
		t.Errorf("Expected repeated word 'the', got %s", result.RepeatedWords[0].Word)
	}
}

func TestRepeatedWordWithPunctuation(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	result := checker.Check(CheckRequest{
		Text:      "Hello hello world",
		ProfileID: "test-profile",
	})

	// Should still detect despite different case and being at sentence start
	if len(result.RepeatedWords) != 1 {
		t.Fatalf("Expected 1 repeated word, got %d", len(result.RepeatedWords))
	}
}

func TestCapitalisationAfterPeriod(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	result := checker.Check(CheckRequest{
		Text:      "The government announced. it was a partnership",
		ProfileID: "test-profile",
	})

	if len(result.CapitalisationIssues) != 1 {
		t.Fatalf("Expected 1 capitalisation issue, got %d", len(result.CapitalisationIssues))
	}

	ci := result.CapitalisationIssues[0]
	if ci.Word != "it" {
		t.Errorf("Expected capitalisation issue for 'it', got %s", ci.Word)
	}
	if ci.Line != 1 {
		t.Errorf("Expected line 1, got %d", ci.Line)
	}
}

func TestCapitalisationAfterExclamation(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	result := checker.Check(CheckRequest{
		Text:      "The government announced! it was great",
		ProfileID: "test-profile",
	})

	if len(result.CapitalisationIssues) != 1 {
		t.Fatalf("Expected 1 capitalisation issue, got %d", len(result.CapitalisationIssues))
	}

	if result.CapitalisationIssues[0].Word != "it" {
		t.Errorf("Expected capitalisation issue for 'it', got %s", result.CapitalisationIssues[0].Word)
	}
}

func TestCapitalisationAfterQuestion(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	result := checker.Check(CheckRequest{
		Text:      "Did the government announce? yes it did",
		ProfileID: "test-profile",
	})

	if len(result.CapitalisationIssues) != 1 {
		t.Fatalf("Expected 1 capitalisation issue, got %d", len(result.CapitalisationIssues))
	}

	if result.CapitalisationIssues[0].Word != "yes" {
		t.Errorf("Expected capitalisation issue for 'yes', got %s", result.CapitalisationIssues[0].Word)
	}
}

func TestNoCapitalisationIssuesWhenCorrect(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	result := checker.Check(CheckRequest{
		Text:      "The government announced. It was a partnership",
		ProfileID: "test-profile",
	})

	if len(result.CapitalisationIssues) != 0 {
		t.Errorf("Expected 0 capitalisation issues, got %d: %+v", len(result.CapitalisationIssues), result.CapitalisationIssues)
	}
}

func TestCapitalisationWithMultipleSentences(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	result := checker.Check(CheckRequest{
		Text:      "First sentence. second sentence! third sentence? fourth",
		ProfileID: "test-profile",
	})

	if len(result.CapitalisationIssues) != 3 {
		t.Fatalf("Expected 3 capitalisation issues (second, third, fourth), got %d", len(result.CapitalisationIssues))
	}

	words := make([]string, 3)
	for i, ci := range result.CapitalisationIssues {
		words[i] = ci.Word
	}

	if words[0] != "second" || words[1] != "third" || words[2] != "fourth" {
		t.Errorf("Expected 'second', 'third', 'fourth', got %v", words)
	}
}

func TestCapitalisationInAllowlist(t *testing.T) {
	_, dir := setupTestPool(t)

	// Add a lowercase term to profile that should be allowed even after period
	basePath := filepath.Join(dir, "base.yaml")
	writeTestFile(t, basePath, `
terms:
  - iPhone
`)

	// Reload store with updated base allowlist
	profilesDir := filepath.Join(dir, "profiles")
	store, err := allowlist.NewStore(basePath, profilesDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	// Create new pool with updated store
	dictPath := filepath.Join(dir, "test_dict.txt")
	newPool, err := NewPool(dictPath, store)
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}

	checker := newPool.Get("test-profile")

	result := checker.Check(CheckRequest{
		Text:      "New phone. iphone is here",
		ProfileID: "test-profile",
	})

	// "iphone" is in allowlist, should not be flagged
	for _, ci := range result.CapitalisationIssues {
		if strings.EqualFold(ci.Word, "iphone") {
			t.Error("Expected 'iphone' to be ignored due to allowlist")
		}
	}
}

func TestCapitalisationWithIgnoreTerms(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	result := checker.Check(CheckRequest{
		Text:        "The government announced. k8s is great",
		ProfileID:   "test-profile",
		IgnoreTerms: []string{"k8s"},
	})

	// "k8s" is in ignore_terms, should not be flagged
	for _, ci := range result.CapitalisationIssues {
		if strings.EqualFold(ci.Word, "k8s") {
			t.Error("Expected 'k8s' to be ignored due to ignore_terms")
		}
	}
}

func TestCapitalisationPositionTracking(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	result := checker.Check(CheckRequest{
		Text:      "Hello. world",
		ProfileID: "test-profile",
	})

	if len(result.CapitalisationIssues) != 1 {
		t.Fatalf("Expected 1 capitalisation issue, got %d", len(result.CapitalisationIssues))
	}

	ci := result.CapitalisationIssues[0]
	// "world" starts at byte offset 7 ("Hello. " = 7 bytes)
	if ci.Offset != 7 {
		t.Errorf("Expected offset 7, got %d", ci.Offset)
	}
	if ci.Line != 1 {
		t.Errorf("Expected line 1, got %d", ci.Line)
	}
	if ci.Column != 8 {
		t.Errorf("Expected column 8, got %d", ci.Column)
	}
}

func TestNoCapitalisationForNonCheckableTokens(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	// Numbers after period should not trigger capitalisation issue
	result := checker.Check(CheckRequest{
		Text:      "Version 1.0 is out",
		ProfileID: "test-profile",
	})

	// The "1.0" is a number token, shouldn't cause capitalisation issues
	// and "is" follows "0" which is a number, not a sentence end
	for _, ci := range result.CapitalisationIssues {
		t.Errorf("Unexpected capitalisation issue: %+v", ci)
	}
}

func TestCapitalisationAfterNewline(t *testing.T) {
	pool, _ := setupTestPool(t)
	checker := pool.Get("test-profile")

	result := checker.Check(CheckRequest{
		Text:      "First sentence.\nsecond sentence",
		ProfileID: "test-profile",
	})

	if len(result.CapitalisationIssues) != 1 {
		t.Fatalf("Expected 1 capitalisation issue after newline, got %d", len(result.CapitalisationIssues))
	}

	if result.CapitalisationIssues[0].Word != "second" {
		t.Errorf("Expected capitalisation issue for 'second', got %s", result.CapitalisationIssues[0].Word)
	}
}
