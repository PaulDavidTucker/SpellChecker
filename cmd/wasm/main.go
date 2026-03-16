//go:build js && wasm
// +build js,wasm

package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"syscall/js"
	"time"

	"github.com/PaulDavidTucker/SpellChecker/internal/tokenizer"
	symspell "github.com/snapp-incubator/go-symspell"
	"github.com/snapp-incubator/go-symspell/pkg/options"
	"github.com/snapp-incubator/go-symspell/pkg/verbosity"
)

//go:embed dictionaries/en_gb.txt
var dictionaryData []byte

var symSpellIndex symspell.SymSpell

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
	// Create SymSpell instance
	symSpellIndex = symspell.NewSymSpell(
		options.WithCountThreshold(1),
		options.WithMaxDictionaryEditDistance(2),
		options.WithPrefixLength(7),
	)

	// Write embedded dictionary to temp file in WASM filesystem
	tmpFile := "/tmp/dictionary.txt"
	if err := os.WriteFile(tmpFile, dictionaryData, 0644); err != nil {
		return fmt.Errorf("writing dictionary to temp file: %w", err)
	}

	// Load dictionary from temp file
	ok, err := symSpellIndex.LoadDictionary(tmpFile, 0, 1, " ")
	if err != nil {
		return fmt.Errorf("loading dictionary: %w", err)
	}
	if !ok {
		return fmt.Errorf("failed to load dictionary")
	}

	return nil
}

func checkText(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return createErrorResponse("text parameter required")
	}

	text := args[0].String()

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

		// Check for capitalisation issues
		if sentenceJustEnded && isCheckable(tok) {
			if isLowercaseWord(tok.Text) {
				capitalisationIssues = append(capitalisationIssues, map[string]interface{}{
					"word":   tok.Text,
					"offset": tok.Offset,
					"line":   tok.Line,
					"column": tok.Column,
				})
			}
			sentenceJustEnded = false
		}

		// Check spelling
		if isCheckable(tok) && tok.Kind != tokenizer.TokenHyphenated {
			// Lookup in SymSpell
			suggestions, _ := symSpellIndex.Lookup(tok.Normalised, verbosity.Top, 2)

			// If no suggestions found, it's likely misspelled
			if len(suggestions) == 0 {
				// Get top 3 suggestions
				allSuggestions, _ := symSpellIndex.Lookup(tok.Normalised, verbosity.Closest, 2)

				suggestionList := make([]map[string]interface{}, 0, 3)
				for i, s := range allSuggestions {
					if i >= 3 {
						break
					}
					suggestionList = append(suggestionList, map[string]interface{}{
						"word":          s.Term,
						"edit_distance": s.Distance,
					})
				}

				misspellings = append(misspellings, map[string]interface{}{
					"word":        tok.Text,
					"offset":      tok.Offset,
					"line":        tok.Line,
					"column":      tok.Column,
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
