package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PaulDavidTucker/SpellChecker/internal/allowlist"
	"github.com/PaulDavidTucker/SpellChecker/internal/checker"
)

type Handler struct {
	pool  *checker.Pool
	store *allowlist.Store
}

func NewHandler(pool *checker.Pool, store *allowlist.Store) *Handler {
	return &Handler{pool: pool, store: store}
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

// corsMiddleware adds CORS headers to allow browser requests
type corsHandler struct {
	handler http.Handler
}

func (ch *corsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Add CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Max-Age", "86400")

	// Handle preflight OPTIONS request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	ch.handler.ServeHTTP(w, r)
}

func corsMiddleware(next http.Handler) http.Handler {
	return &corsHandler{handler: next}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /check", h.handleCheck)
	mux.HandleFunc("GET /health", h.handleHealth)
	mux.HandleFunc("GET /docs", h.handleDocs)
	mux.HandleFunc("GET /swagger.json", h.handleSwaggerJSON)

	// Profile management endpoints
	mux.HandleFunc("GET /profiles", h.handleListProfiles)
	mux.HandleFunc("POST /profiles", h.handleCreateProfile)
	mux.HandleFunc("GET /profiles/{id}", h.handleGetProfile)
	mux.HandleFunc("PUT /profiles/{id}", h.handleUpdateProfile)
	mux.HandleFunc("DELETE /profiles/{id}", h.handleDeleteProfile)
	mux.HandleFunc("POST /profiles/{id}/terms", h.handleAddTerm)
	mux.HandleFunc("DELETE /profiles/{id}/terms/{term}", h.handleRemoveTerm)

	// Static files (for production) - API routes take precedence
	mux.HandleFunc("GET /", h.handleStaticFiles)

	// Wrap with CORS and logging middleware
	return corsMiddleware(loggingMiddleware(mux))
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
		TokenCount:           result.TokenCount,
		CheckedCount:         result.CheckedCount,
		ElapsedMs:            result.ElapsedMs,
		Misspellings:         make([]MisspellingResponse, 0, len(result.Misspellings)),
		RepeatedWords:        make([]RepeatedWordResponse, 0, len(result.RepeatedWords)),
		CapitalisationIssues: make([]CapitalisationIssueResponse, 0, len(result.CapitalisationIssues)),
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

	for _, ci := range result.CapitalisationIssues {
		resp.CapitalisationIssues = append(resp.CapitalisationIssues, CapitalisationIssueResponse{
			Word:   ci.Word,
			Offset: ci.Offset,
			Line:   ci.Line,
			Column: ci.Column,
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

// Profile handlers

func (h *Handler) handleListProfiles(w http.ResponseWriter, r *http.Request) {
	ids := h.store.ProfileIDs()
	profiles := make([]ProfileResponse, 0, len(ids))

	for _, id := range ids {
		terms := h.store.ProfileTerms(id)
		profiles = append(profiles, ProfileResponse{
			ID:    id,
			Terms: terms,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profiles)
}

func (h *Handler) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	profile, ok := h.store.GetProfile(id)
	if !ok {
		http.Error(w, `{"error":"profile not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ProfileResponse{
		ID:          profile.ID,
		Description: profile.Description,
		Terms:       profile.Terms,
	})
}

func (h *Handler) handleCreateProfile(w http.ResponseWriter, r *http.Request) {
	var profile allowlist.Profile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if profile.ID == "" {
		http.Error(w, `{"error":"profile_id is required"}`, http.StatusBadRequest)
		return
	}

	if err := h.store.CreateProfile(profile); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ProfileResponse{
		ID:          profile.ID,
		Description: profile.Description,
		Terms:       profile.Terms,
	})
}

func (h *Handler) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var profile allowlist.Profile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	profile.ID = id // Ensure ID matches URL

	if err := h.store.UpdateProfile(profile); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, `{"error":"profile not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ProfileResponse{
		ID:          profile.ID,
		Description: profile.Description,
		Terms:       profile.Terms,
	})
}

func (h *Handler) handleDeleteProfile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.store.DeleteProfile(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, `{"error":"profile not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleAddTerm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Term string `json:"term"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.Term == "" {
		http.Error(w, `{"error":"term is required"}`, http.StatusBadRequest)
		return
	}

	if err := h.store.AddTermToProfile(id, req.Term); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, `{"error":"profile not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "added"})
}

func (h *Handler) handleRemoveTerm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	term := r.PathValue("term")

	if err := h.store.RemoveTermFromProfile(id, term); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, `{"error":"profile or term not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ProfileResponse is the API response format for profiles
type ProfileResponse struct {
	ID          string   `json:"id"`
	Description string   `json:"description,omitempty"`
	Terms       []string `json:"terms"`
}

// handleStaticFiles serves the React app static files in production
func (h *Handler) handleStaticFiles(w http.ResponseWriter, r *http.Request) {
	// Try to serve from webapp/dist directory (production build)
	path := "/webapp/dist" + r.URL.Path
	if r.URL.Path == "/" {
		path = "/webapp/dist/index.html"
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// If static files not found, return API info
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"SpellChecker API","endpoints":["/health","/check","/profiles","/docs"],"webapp":"not built - run npm run build in webapp/"}`))
		return
	}

	// Serve the file
	http.ServeFile(w, r, path)
}
