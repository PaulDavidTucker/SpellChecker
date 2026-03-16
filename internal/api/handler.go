package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PaulDavidTucker/SpellChecker/internal/checker"
)

type Handler struct {
	pool *checker.Pool
}

func NewHandler(pool *checker.Pool) *Handler {
	return &Handler{pool: pool}
}

// loggingMiddleware wraps an http.Handler and logs each request
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the response writer to capture the status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /check", h.handleCheck)
	mux.HandleFunc("GET /health", h.handleHealth)
	mux.HandleFunc("GET /docs", h.handleDocs)
	mux.HandleFunc("GET /swagger.json", h.handleSwaggerJSON)

	// Wrap with logging middleware
	return loggingMiddleware(mux)
}

func (h *Handler) handleCheck(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req SpellCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Provide a more helpful error message for common JSON issues
		errMsg := "invalid JSON"
		if strings.Contains(err.Error(), "invalid character") && strings.Contains(err.Error(), "after object key:value pair") {
			errMsg = "invalid JSON: make sure to escape quotes inside text (e.g., use \\\"hello\\\" instead of \"hello\")"
		}
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, errMsg), http.StatusBadRequest)
		return
	}

	if req.Text == "" && req.TextBase64 == "" {
		http.Error(
			w,
			`{"error":"text or text_base64 field is required"}`,
			http.StatusBadRequest,
		)
		return
	}

	// Determine the text to check
	var text string
	if req.Text != "" {
		text = req.Text
	} else if req.TextBase64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(req.TextBase64)
		if err != nil {
			http.Error(
				w,
				`{"error":"invalid base64 encoding"}`,
				http.StatusBadRequest,
			)
			return
		}
		text = string(decoded)
	}

	// Select the right checker for this profile
	c := h.pool.Get(req.ProfileID)

	// Run the pipeline
	result := c.Check(checker.CheckRequest{
		Text:           text,
		ProfileID:      req.ProfileID,
		IgnoreTerms:    req.IgnoreTerms,
		MaxSuggestions: req.MaxSuggestions,
	})

	// Map internal types to response types
	resp := SpellCheckResponse{
		TokenCount:    result.TokenCount,
		CheckedCount:  result.CheckedCount,
		ElapsedMs:     result.ElapsedMs,
		Misspellings:  make([]MisspellingResponse, 0, len(result.Misspellings)),
		RepeatedWords: make([]RepeatedWordResponse, 0, len(result.RepeatedWords)),
	}

	for _, ms := range result.Misspellings {
		msResp := MisspellingResponse{
			Word:   ms.Word,
			Offset: ms.Offset,
			Line:   ms.Line,
			Column: ms.Column,
		}
		for _, s := range ms.Suggestions {
			msResp.Suggestions = append(
				msResp.Suggestions,
				SuggestionResponse{
					Word:         s.Word,
					EditDistance: s.EditDistance,
				},
			)
		}
		resp.Misspellings = append(resp.Misspellings, msResp)
	}

	for _, rw := range result.RepeatedWords {
		resp.RepeatedWords = append(resp.RepeatedWords, RepeatedWordResponse{
			Word:   rw.Word,
			Offset: rw.Offset,
			Line:   rw.Line,
			Column: rw.Column,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleHealth(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func (h *Handler) handleDocs(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(swaggerUIHTML))
}

func (h *Handler) handleSwaggerJSON(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(swaggerJSON))
}
