package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/PaulDavidTucker/SpellChecker/internal/allowlist"
	"github.com/PaulDavidTucker/SpellChecker/internal/checker"
)

// LazyHandler is an API handler that uses LazyPool for async dictionary loading
type LazyHandler struct {
	pool  *checker.LazyPool
	store *allowlist.Store
}

// NewLazyHandler creates a handler with lazy pool loading
func NewLazyHandler(pool *checker.LazyPool, store *allowlist.Store) *LazyHandler {
	return &LazyHandler{pool: pool, store: store}
}

// Routes returns the HTTP handler with all routes
func (h *LazyHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /check", h.handleCheck)
	mux.HandleFunc("GET /health", h.handleHealth)
	mux.HandleFunc("GET /ready", h.handleReady)
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

	// Wrap with CORS, logging, and ready check middleware
	// Note: corsMiddleware and loggingMiddleware are defined in handler.go
	return h.pool.ReadyMiddleware(corsMiddleware(loggingMiddleware(mux)))
}

func (h *LazyHandler) handleCheck(w http.ResponseWriter, r *http.Request) {
	var req SpellCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if req.Text == "" && req.TextBase64 == "" {
		http.Error(w, `{"error":"text or text_base64 required"}`, http.StatusBadRequest)
		return
	}

	var text string
	if req.Text != "" {
		text = req.Text
	} else if req.TextBase64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(req.TextBase64)
		if err != nil {
			http.Error(w, `{"error":"invalid base64"}`, http.StatusBadRequest)
			return
		}
		text = string(decoded)
	}

	c := h.pool.Get(req.ProfileID)
	result := c.Check(checker.CheckRequest{
		Text:           text,
		ProfileID:      req.ProfileID,
		IgnoreTerms:    req.IgnoreTerms,
		MaxSuggestions: req.MaxSuggestions,
	})

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
			msResp.Suggestions = append(msResp.Suggestions, SuggestionResponse{
				Word:         s.Word,
				EditDistance: s.EditDistance,
			})
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
			Word:       ci.Word,
			Offset:     ci.Offset,
			Line:       ci.Line,
			Column:     ci.Column,
			Suggestion: ci.Suggestion,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *LazyHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ready := h.pool.IsReady()
	loadTime := h.pool.GetLoadTime()

	status := map[string]interface{}{
		"status":    "ok",
		"ready":     ready,
		"load_time": loadTime.Seconds(),
	}

	if ready {
		status["message"] = "Service is ready"
	} else {
		status["message"] = "Dictionary still loading"
	}

	json.NewEncoder(w).Encode(status)
}

func (h *LazyHandler) handleReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ready := h.pool.IsReady()
	loadTime := h.pool.GetLoadTime()

	if ready {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ready":     true,
			"load_time": loadTime.Seconds(),
		})
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ready":     false,
			"load_time": loadTime.Seconds(),
			"message":   "Dictionary still loading",
		})
	}
}

func (h *LazyHandler) handleListProfiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	profiles := make([]map[string]interface{}, 0)
	for _, id := range h.store.ProfileIDs() {
		terms := h.store.ProfileTerms(id)
		profiles = append(profiles, map[string]interface{}{
			"id":    id,
			"terms": terms,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"profiles": profiles,
	})
}

func (h *LazyHandler) handleCreateProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"error":"not implemented in lazy mode"}`))
}

func (h *LazyHandler) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	w.Header().Set("Content-Type", "application/json")

	terms := h.store.ProfileTerms(id)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":    id,
		"terms": terms,
	})
}

func (h *LazyHandler) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"error":"not implemented in lazy mode"}`))
}

func (h *LazyHandler) handleDeleteProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"error":"not implemented in lazy mode"}`))
}

func (h *LazyHandler) handleAddTerm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"error":"not implemented in lazy mode"}`))
}

func (h *LazyHandler) handleRemoveTerm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"error":"not implemented in lazy mode"}`))
}

func (h *LazyHandler) handleDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>SpellChecker API</title></head>
<body>
<h1>SpellChecker API</h1>
<p>Status: <a href="/health">/health</a></p>
<p>Ready: <a href="/ready">/ready</a></p>
</body>
</html>`))
}

func (h *LazyHandler) handleSwaggerJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"openapi":"3.0.0","info":{"title":"SpellChecker","version":"1.0.0"}}`))
}

func (h *LazyHandler) handleStaticFiles(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("Not found"))
}

// Reuse middleware from original handler
// ... (same as in handler.go)
