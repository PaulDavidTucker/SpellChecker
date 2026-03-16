package checker

// Misspelling represents a single spelling error found.
type Misspelling struct {
	Word        string       `json:"word"`
	Offset      int          `json:"offset"`
	Line        int          `json:"line"`
	Column      int          `json:"column"`
	Suggestions []Suggestion `json:"suggestions"`
}

// RepeatedWord represents a word that appears twice in a row (e.g., "the the").
type RepeatedWord struct {
	Word   string `json:"word"`
	Offset int    `json:"offset"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

// CapitalisationIssue represents a lowercase word that follows sentence-ending
// punctuation, indicating a potential capitalisation error.
type CapitalisationIssue struct {
	Word   string `json:"word"`
	Offset int    `json:"offset"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

// Suggestion is a candidate correction for a misspelling.
type Suggestion struct {
	Word         string `json:"word"`
	EditDistance int    `json:"edit_distance"`
}

// CheckResult is the complete output of a spell check run.
type CheckResult struct {
	Misspellings         []Misspelling         `json:"misspellings"`
	RepeatedWords        []RepeatedWord        `json:"repeated_words"`
	CapitalisationIssues []CapitalisationIssue `json:"capitalisation_issues"`
	TokenCount           int                   `json:"token_count"`
	CheckedCount         int                   `json:"checked_count"`
	ElapsedMs            float64               `json:"elapsed_ms"`
}

// CheckRequest holds everything needed to check a piece of text.
type CheckRequest struct {
	Text           string
	ProfileID      string
	IgnoreTerms    []string
	MaxSuggestions int
}
