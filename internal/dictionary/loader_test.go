package dictionary

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestLoadBasic tests loading a basic dictionary file
func TestLoadBasic(t *testing.T) {
	dir := t.TempDir()
	dictPath := filepath.Join(dir, "test_dict.txt")

	// Create a test dictionary with word frequency format
	content := "the 100000\ngovernment 50000\ntest 25000\n"
	if err := os.WriteFile(dictPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	dict, err := Load(dictPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if dict.WordCount() != 3 {
		t.Errorf("Expected 3 words, got %d", dict.WordCount())
	}

	// Check that entries exist
	entries := dict.Entries()
	if _, ok := entries["the"]; !ok {
		t.Error("Expected 'the' in dictionary")
	}
	if _, ok := entries["government"]; !ok {
		t.Error("Expected 'government' in dictionary")
	}
	if _, ok := entries["test"]; !ok {
		t.Error("Expected 'test' in dictionary")
	}
}

// TestLoadWordsOnly tests loading a dictionary with just words (no frequencies)
func TestLoadWordsOnly(t *testing.T) {
	dir := t.TempDir()
	dictPath := filepath.Join(dir, "words_only.txt")

	// Create dictionary with just words, no frequencies
	content := "hello\nworld\ntest\n"
	if err := os.WriteFile(dictPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	dict, err := Load(dictPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if dict.WordCount() != 3 {
		t.Errorf("Expected 3 words, got %d", dict.WordCount())
	}

	// Check default frequency is 1
	entries := dict.Entries()
	if freq := entries["hello"]; freq != 1 {
		t.Errorf("Expected default frequency 1, got %d", freq)
	}
}

// TestLoadMixedContent tests loading a dictionary with mixed content
func TestLoadMixedContent(t *testing.T) {
	dir := t.TempDir()
	dictPath := filepath.Join(dir, "mixed.txt")

	// Mix of words with frequencies and words without
	content := "highfreq 1000000\nnofreq\nlowfreq 100\n"
	if err := os.WriteFile(dictPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	dict, err := Load(dictPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	entries := dict.Entries()

	// Check frequencies
	if freq := entries["highfreq"]; freq != 1000000 {
		t.Errorf("Expected frequency 1000000, got %d", freq)
	}
	if freq := entries["nofreq"]; freq != 1 {
		t.Errorf("Expected default frequency 1, got %d", freq)
	}
	if freq := entries["lowfreq"]; freq != 100 {
		t.Errorf("Expected frequency 100, got %d", freq)
	}
}

// TestLoadCaseInsensitive tests that words are lowercased
func TestLoadCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	dictPath := filepath.Join(dir, "mixed_case.txt")

	content := "Hello 100\nWORLD 200\nTest 300\n"
	if err := os.WriteFile(dictPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	dict, err := Load(dictPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	entries := dict.Entries()

	// All should be lowercase
	if _, ok := entries["hello"]; !ok {
		t.Error("Expected 'hello' (lowercased) in dictionary")
	}
	if _, ok := entries["world"]; !ok {
		t.Error("Expected 'world' (lowercased) in dictionary")
	}
	if _, ok := entries["test"]; !ok {
		t.Error("Expected 'test' (lowercased) in dictionary")
	}

	// Check original case is not present
	if _, ok := entries["Hello"]; ok {
		t.Error("Did not expect 'Hello' (original case) in dictionary")
	}
}

// TestLoadEmptyLines tests that empty lines are skipped
func TestLoadEmptyLines(t *testing.T) {
	dir := t.TempDir()
	dictPath := filepath.Join(dir, "with_empty.txt")

	content := "word1 100\n\n\nword2 200\n\nword3 300\n"
	if err := os.WriteFile(dictPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	dict, err := Load(dictPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if dict.WordCount() != 3 {
		t.Errorf("Expected 3 words, got %d", dict.WordCount())
	}
}

// TestLoadInvalidFrequency tests handling of invalid frequency values
func TestLoadInvalidFrequency(t *testing.T) {
	dir := t.TempDir()
	dictPath := filepath.Join(dir, "invalid_freq.txt")

	// Mix of valid and invalid frequencies
	content := "valid 100\ninvalid abc\nalsovalid 200\n"
	if err := os.WriteFile(dictPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	dict, err := Load(dictPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	entries := dict.Entries()

	// Valid frequencies should be loaded
	if freq := entries["valid"]; freq != 100 {
		t.Errorf("Expected frequency 100, got %d", freq)
	}
	if freq := entries["alsovalid"]; freq != 200 {
		t.Errorf("Expected frequency 200, got %d", freq)
	}

	// Invalid frequency should default to 1
	if freq := entries["invalid"]; freq != 1 {
		t.Errorf("Expected default frequency 1 for invalid, got %d", freq)
	}
}

// TestLoadNonExistentFile tests loading a non-existent file
func TestLoadNonExistentFile(t *testing.T) {
	dictPath := "/nonexistent/path/dictionary.txt"

	_, err := Load(dictPath)
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

// TestLoadLargeFile tests loading a larger dictionary
func TestLoadLargeFile(t *testing.T) {
	dir := t.TempDir()
	dictPath := filepath.Join(dir, "large_dict.txt")

	// Generate a larger dictionary with unique words
	var content string
	for i := 0; i < 100; i++ {
		word := fmt.Sprintf("word%c%d", 'a'+rune(i%26), i)
		freq := (i%9 + 1) * 1000
		content += fmt.Sprintf("%s %d\n", word, freq)
	}

	if err := os.WriteFile(dictPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	dict, err := Load(dictPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// We should get 100 unique words
	if dict.WordCount() != 100 {
		t.Errorf("Expected 100 words, got %d", dict.WordCount())
	}
}

// TestEntries tests the Entries() method
func TestEntries(t *testing.T) {
	dir := t.TempDir()
	dictPath := filepath.Join(dir, "test.txt")

	content := "word1 100\nword2 200\n"
	if err := os.WriteFile(dictPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	dict, err := Load(dictPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	entries := dict.Entries()
	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}

	// Verify we can read frequencies
	if entries["word1"] != 100 {
		t.Errorf("Expected word1 frequency 100, got %d", entries["word1"])
	}
	if entries["word2"] != 200 {
		t.Errorf("Expected word2 frequency 200, got %d", entries["word2"])
	}
}
