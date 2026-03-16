package api

// SpellCheckRequest is the JSON body of POST /check.
// Either 'text' or 'text_base64' must be provided (not both).
type SpellCheckRequest struct {
	Text           string   `json:"text"`
	TextBase64     string   `json:"text_base64"`
	Locale         string   `json:"locale"`
	ProfileID      string   `json:"profile_id"`
	ContentType    string   `json:"content_type"`
	IgnoreTerms    []string `json:"ignore_terms"`
	MaxSuggestions int      `json:"max_suggestions"`
}

// SpellCheckResponse is the JSON response.
type SpellCheckResponse struct {
	Misspellings  []MisspellingResponse  `json:"misspellings"`
	RepeatedWords []RepeatedWordResponse `json:"repeated_words"`
	TokenCount    int                    `json:"token_count"`
	CheckedCount  int                    `json:"checked_count"`
	ElapsedMs     float64                `json:"elapsed_ms"`
}

type MisspellingResponse struct {
	Word        string               `json:"word"`
	Offset      int                  `json:"offset"`
	Line        int                  `json:"line"`
	Column      int                  `json:"column"`
	Suggestions []SuggestionResponse `json:"suggestions"`
}

type RepeatedWordResponse struct {
	Word   string `json:"word"`
	Offset int    `json:"offset"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

type SuggestionResponse struct {
	Word         string `json:"word"`
	EditDistance int    `json:"edit_distance"`
}
