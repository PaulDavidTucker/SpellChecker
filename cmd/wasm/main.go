//go:build js && wasm
// +build js,wasm

package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"syscall/js"
	"time"
	"unicode"

	"github.com/PaulDavidTucker/SpellChecker/internal/tokenizer"
)

//go:embed dictionaries/en_gb.txt
var dictionaryData []byte

// SimpleSpellChecker implements a basic spell checker with O(1) word lookup
// and generates suggestions using edit distance
type SimpleSpellChecker struct {
	words    map[string]int // word -> frequency
	wordList []string       // sorted list for generating suggestions
}

var spellChecker *SimpleSpellChecker

func main() {
	// Initialize the spell checker with embedded dictionary
	if err := initChecker(); err != nil {
		fmt.Printf("Failed to initialize checker: %v\n", err)
		return
	}

	// Expose functions to JavaScript
	js.Global().Set("spellchecker", js.ValueOf(map[string]interface{}{
		"checkText": js.FuncOf(checkText),
		"isReady":   js.ValueOf(true),
		"version":   js.ValueOf("1.0.0-wasm"),
	}))

	fmt.Println("WASM SpellChecker initialized successfully")

	// Keep the program running
	select {}
}

func initChecker() error {
	spellChecker = &SimpleSpellChecker{
		words:    make(map[string]int),
		wordList: make([]string, 0),
	}

	// Load dictionary from embedded data
	if err := spellChecker.loadDictionary(); err != nil {
		return fmt.Errorf("loading dictionary: %w", err)
	}

	fmt.Printf("Dictionary loaded with %d entries\n", len(spellChecker.words))
	return nil
}

// loadDictionary parses the embedded dictionary data and populates the word map
func (sc *SimpleSpellChecker) loadDictionary() error {
	lines := strings.Split(string(dictionaryData), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		word := strings.ToLower(fields[0])
		count, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		sc.words[word] = count
		sc.wordList = append(sc.wordList, word)
	}

	// Sort word list for consistent suggestion generation
	sort.Strings(sc.wordList)

	if len(sc.words) == 0 {
		return fmt.Errorf("no valid dictionary entries found")
	}

	return nil
}

// isValidWord checks if a word exists in the dictionary
func (sc *SimpleSpellChecker) isValidWord(word string) bool {
	word = strings.ToLower(word)
	_, exists := sc.words[word]
	return exists
}

// getSuggestions returns up to maxSuggestions similar words using edit distance
func (sc *SimpleSpellChecker) getSuggestions(word string, maxEditDistance, maxSuggestions int) []Suggestion {
	word = strings.ToLower(word)

	if sc.isValidWord(word) {
		return nil // Word is correct, no suggestions needed
	}

	candidates := make([]Suggestion, 0)

	// Check all dictionary words
	for dictWord, freq := range sc.words {
		distance := levenshteinDistance(word, dictWord)
		if distance <= maxEditDistance {
			candidates = append(candidates, Suggestion{
				Word:     dictWord,
				Distance: distance,
				Freq:     freq,
			})
		}
	}

	// Sort by distance (ascending), then by frequency (descending)
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Distance != candidates[j].Distance {
			return candidates[i].Distance < candidates[j].Distance
		}
		return candidates[i].Freq > candidates[j].Freq
	})

	// Return top N suggestions
	if len(candidates) > maxSuggestions {
		candidates = candidates[:maxSuggestions]
	}

	return candidates
}

// trySplitWord attempts to split a word into two valid words (missing space detection)
// Returns the split suggestion with a space if valid, otherwise empty string
// If atSentenceStart is true, capitalizes the first word in the suggestion
// Examples: "helloworld" → "hello world", "spellingmistake" → "spelling mistake"
// Examples with sentence start: "Helloworld" → "Hello world"
func (sc *SimpleSpellChecker) trySplitWord(word string, atSentenceStart bool) string {
	word = strings.ToLower(word)

	// Need at least 3 characters to split into two words (1+2 or 2+1 minimum)
	if len(word) < 3 {
		return ""
	}

	// Try splitting at every possible position
	for i := 2; i < len(word)-2; i++ {
		first := word[:i]
		second := word[i:]

		// Both parts must be valid words
		if sc.isValidWord(first) && sc.isValidWord(second) {
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

type Suggestion struct {
	Word     string
	Distance int
	Freq     int
}

// levenshteinDistance calculates the edit distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Ensure s1 is the shorter string for space optimization
	if len(s1) > len(s2) {
		s1, s2 = s2, s1
	}

	previousRow := make([]int, len(s1)+1)
	currentRow := make([]int, len(s1)+1)

	for i := 0; i <= len(s1); i++ {
		previousRow[i] = i
	}

	for j := 1; j <= len(s2); j++ {
		currentRow[0] = j

		for i := 1; i <= len(s1); i++ {
			insertionCost := previousRow[i] + 1
			deletionCost := currentRow[i-1] + 1
			substitutionCost := previousRow[i-1]
			if s1[i-1] != s2[j-1] {
				substitutionCost += 1
			}

			currentRow[i] = min(insertionCost, min(deletionCost, substitutionCost))
		}

		previousRow, currentRow = currentRow, previousRow
	}

	return previousRow[len(s1)]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

// hasRandomCapitalLetters checks if a word has capital letters in the middle
// (e.g., "camelCase", "PascalCase", "teSt")
func hasRandomCapitalLetters(word string) bool {
	if len(word) <= 1 {
		return false
	}

	runes := []rune(word)

	// Skip all-caps words (e.g., "NASA", "BBC")
	allCaps := true
	for _, r := range runes {
		if unicode.IsLetter(r) && !unicode.IsUpper(r) {
			allCaps = false
			break
		}
	}
	if allCaps {
		return false
	}

	// Check for capitals after the first character
	for i := 1; i < len(runes); i++ {
		if unicode.IsUpper(runes[i]) {
			return true
		}
	}

	return false
}

// toAllLower converts a word to all lowercase
func toAllLower(word string) string {
	return strings.ToLower(word)
}

func checkText(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return createErrorResponse("text parameter required")
	}

	text := args[0].String()
	_ = args[1].String() // profileId is currently ignored in WASM mode

	start := time.Now()

	// Tokenize the text
	scanner := tokenizer.NewScanner(text)
	tokens := scanner.Tokenize()

	// Check each token
	var misspellings []map[string]interface{}
	var repeatedWords []map[string]interface{}
	var capitalisationIssues []map[string]interface{}

	var prevToken *tokenizer.Token
	sentenceJustEnded := true

	for _, tok := range tokens {
		// Check for repeated words
		if isCheckable(tok) && tok.Kind != tokenizer.TokenHyphenated {
			if prevToken != nil && strings.EqualFold(tok.Normalised, prevToken.Normalised) {
				repeatedWords = append(repeatedWords, map[string]interface{}{
					"word":   tok.Text,
					"offset": tok.Offset,
					"line":   tok.Line,
					"column": tok.Column,
				})
			}
			prevToken = &tok
		}

		// Check for capitalisation issues at sentence start
		if sentenceJustEnded && isCheckable(tok) {
			if isLowercaseWord(tok.Text) {
				// Suggest the capitalized version
				suggestion := capitalizeFirst(tok.Text)
				capitalisationIssues = append(capitalisationIssues, map[string]interface{}{
					"word":        tok.Text,
					"offset":      tok.Offset,
					"line":        tok.Line,
					"column":      tok.Column,
					"suggestions": []string{suggestion},
				})
			}
			sentenceJustEnded = false
		}

		// Check for random capital letters in the middle of words
		if isCheckable(tok) && tok.Kind != tokenizer.TokenHyphenated {
			if hasRandomCapitalLetters(tok.Text) {
				// Suggest the all-lowercase version
				suggestion := toAllLower(tok.Text)
				capitalisationIssues = append(capitalisationIssues, map[string]interface{}{
					"word":        tok.Text,
					"offset":      tok.Offset,
					"line":        tok.Line,
					"column":      tok.Column,
					"suggestions": []string{suggestion},
				})
			}
		}

		// Check spelling
		if isCheckable(tok) && tok.Kind != tokenizer.TokenHyphenated {
			normalizedWord := tok.Normalised

			if !spellChecker.isValidWord(normalizedWord) {
				// Get suggestions
				suggestions := spellChecker.getSuggestions(normalizedWord, 2, 3)

				suggestionList := make([]map[string]interface{}, 0, len(suggestions))
				for _, s := range suggestions {
					suggestionList = append(suggestionList, map[string]interface{}{
						"word":          s.Word,
						"edit_distance": s.Distance,
					})
				}

				// Check if this could be two words missing a space
				issueType := "misspelling"
				if splitSuggestion := spellChecker.trySplitWord(normalizedWord, sentenceJustEnded); splitSuggestion != "" {
					// This is a missing space issue, not a misspelling
					issueType = "missing_space"
					// Add the split suggestion at the beginning (higher priority)
					suggestionList = append([]map[string]interface{}{
						{
							"word":          splitSuggestion,
							"edit_distance": 0, // Perfect match when split
						},
					}, suggestionList...)
				}

				misspellings = append(misspellings, map[string]interface{}{
					"word":        tok.Text,
					"offset":      tok.Offset,
					"line":        tok.Line,
					"column":      tok.Column,
					"type":        issueType,
					"suggestions": suggestionList,
				})
			}
		}

		// Track sentence endings
		if tok.Kind == tokenizer.TokenPunctuation && isSentenceEnding(tok.Text) {
			sentenceJustEnded = true
		} else if tok.Kind != tokenizer.TokenWhitespace {
			sentenceJustEnded = false
		}
	}

	elapsedMs := float64(time.Since(start).Microseconds()) / 1000.0

	response := map[string]interface{}{
		"misspellings":          misspellings,
		"repeated_words":        repeatedWords,
		"capitalisation_issues": capitalisationIssues,
		"token_count":           len(tokens),
		"checked_count":         len(tokens),
		"elapsed_ms":            elapsedMs,
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return createErrorResponse("failed to encode response")
	}

	return js.ValueOf(string(jsonBytes))
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

func isLowercaseWord(word string) bool {
	if word == "" {
		return false
	}
	for _, r := range word {
		// Check if first character is lowercase
		return r >= 'a' && r <= 'z'
	}
	return false
}

func isSentenceEnding(punct string) bool {
	return punct == "." || punct == "!" || punct == "?"
}

func createErrorResponse(message string) js.Value {
	response := map[string]interface{}{
		"error": message,
	}
	jsonBytes, _ := json.Marshal(response)
	return js.ValueOf(string(jsonBytes))
}
