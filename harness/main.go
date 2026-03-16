package main

import (
	"encoding/json"
	"html/template"
	"log"
	"os"
	"path/filepath"

	"github.com/PaulDavidTucker/SpellChecker/internal/allowlist"
	"github.com/PaulDavidTucker/SpellChecker/internal/checker"
)

// TestCase matches our JSON structure
type TestCase struct {
	ID    string `json:"id"`
	Focus string `json:"focus"`
	Text  string `json:"text"`
}

// TestResult holds the test case and our checker's output
type TestResult struct {
	TestCase
	Misspellings  []string
	RepeatedWords []string
	Changed       bool
}

// Keep the HTMLTemplate string exactly the same as the previous message here
const HTMLTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Spell Checker v1 - Demo Results</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto,
                         Helvetica, Arial, sans-serif;
            background-color: #f4f4f9;
            color: #333;
            max-width: 1000px;
            margin: 0 auto;
            padding: 2rem;
        }
        h1 { color: #111; border-bottom: 2px solid #ddd; padding-bottom: 0.5rem; }
        .summary { background: #fff; padding: 1rem; border-radius: 8px;
                   box-shadow: 0 2px 4px rgba(0,0,0,0.1); margin-bottom: 2rem; }
        .card { background: #fff; border-radius: 8px; margin-bottom: 1.5rem;
                box-shadow: 0 2px 4px rgba(0,0,0,0.1); overflow: hidden; }
        .card-header { background: #2c3e50; color: #fff; padding: 0.75rem 1rem;
                       font-weight: bold; display: flex;
                       justify-content: space-between; }
        .card-body { padding: 1rem; }
        .text-block { margin-bottom: 1rem; }
        .label { font-size: 0.85rem; font-weight: bold; color: #666;
                 text-transform: uppercase; margin-bottom: 0.25rem; }
        .original { background: #ffebee; border-left: 4px solid #f44336;
                    padding: 0.75rem; border-radius: 0 4px 4px 0; }
        .issues { background: #fff3e0; border-left: 4px solid #ff9800;
                  padding: 0.75rem; border-radius: 0 4px 4px 0; }
        .unchanged { background: #e8f5e9; border-left: 4px solid #4caf50;
                     padding: 0.75rem; border-radius: 0 4px 4px 0; }
        .misspelling { color: #d32f2f; font-weight: bold; }
        .repeated { color: #f57c00; font-weight: bold; }
    </style>
</head>
<body>
    <h1>Spell Checker v1 - Test Results</h1>

    <div class="summary">
        <p><strong>Total Tests Run:</strong> {{len .}}</p>
        <p><em>Review the automated corrections below. Highlighted sections
           indicate text processing differences.</em></p>
    </div>

    {{range .}}
    {{$result := .}}
    <div class="card">
        <div class="card-header">
            <span>{{.ID}}</span>
            <span>Focus: {{.Focus}}</span>
        </div>
        <div class="card-body">
            <div class="text-block">
                <div class="label">Original Text</div>
                <div class="original">{{.Text}}</div>
            </div>
            <div class="text-block">
                <div class="label">Issues Found</div>
                {{if .Changed}}
                    <div class="issues">
                        {{if .Misspellings}}
                            <div><strong>Misspellings:</strong>
                                {{range $i, $word := .Misspellings}}
                                    <span class="misspelling">{{$word}}</span>{{if not (eq $i (sub (len $result.Misspellings) 1))}}, {{end}}
                                {{end}}
                            </div>
                        {{end}}
                        {{if .RepeatedWords}}
                            <div><strong>Repeated Words:</strong>
                                {{range $i, $word := .RepeatedWords}}
                                    <span class="repeated">{{$word}}</span>{{if not (eq $i (sub (len $result.RepeatedWords) 1))}}, {{end}}
                                {{end}}
                            </div>
                        {{end}}
                    </div>
                {{else}}
                    <div class="unchanged">No issues detected</div>
                {{end}}
            </div>
        </div>
    </div>
    {{end}}
</body>
</html>
`

func main() {
	// 1. Set up paths - use the actual project paths
	dictPath := filepath.Join("..", "dictionaries", "en_gb.txt")
	baseAllowlistPath := filepath.Join("..", "config", "base-allowlist.yaml")
	profilesDir := filepath.Join("..", "config", "profiles")

	// 2. Load allowlist store
	store, err := allowlist.NewStore(baseAllowlistPath, profilesDir)
	if err != nil {
		log.Fatalf("Failed to load allowlist store: %v", err)
	}
	log.Printf("Loaded %d profiles", store.ProfileCount())

	// 3. Build the checker pool
	pool, err := checker.NewPool(dictPath, store)
	if err != nil {
		log.Fatalf("Failed to build checker pool: %v", err)
	}

	// 4. Get a checker instance (use "default" profile or empty string for fallback)
	spellChecker := pool.Get("default")

	// 5. Read the JSON fixtures
	fixturePath := "./fixtures.json"
	file, err := os.ReadFile(fixturePath)
	if err != nil {
		log.Fatalf("Failed to read test cases from %s: %v", fixturePath, err)
	}

	var testCases []TestCase
	if err := json.Unmarshal(file, &testCases); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	log.Printf("Loaded %d test cases", len(testCases))

	// 6. Process each test case through the checker
	var results []TestResult
	for _, tc := range testCases {
		// Run the spell check
		result := spellChecker.Check(checker.CheckRequest{
			Text:           tc.Text,
			ProfileID:      "default",
			MaxSuggestions: 3,
		})

		// Extract misspelled words and repeated words
		var misspellings []string
		for _, ms := range result.Misspellings {
			misspellings = append(misspellings, ms.Word)
		}

		var repeatedWords []string
		for _, rw := range result.RepeatedWords {
			repeatedWords = append(repeatedWords, rw.Word)
		}

		results = append(results, TestResult{
			TestCase:      tc,
			Misspellings:  misspellings,
			RepeatedWords: repeatedWords,
			Changed:       len(result.Misspellings) > 0 || len(result.RepeatedWords) > 0,
		})
	}

	// 7. Generate the HTML report
	outputFile, err := os.Create("demo_results.html")
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer outputFile.Close()

	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"sub": func(a, b int) int { return a - b },
	}).Parse(HTMLTemplate)
	if err != nil {
		log.Fatalf("Failed to parse HTML template: %v", err)
	}

	if err := tmpl.Execute(outputFile, results); err != nil {
		log.Fatalf("Failed to write HTML: %v", err)
	}

	log.Printf("Demo successfully generated at demo_results.html! Processed %d test cases.", len(results))
}
