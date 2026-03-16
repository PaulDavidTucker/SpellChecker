package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PaulDavidTucker/SpellChecker/internal/allowlist"
	"github.com/PaulDavidTucker/SpellChecker/internal/checker"
)

// buildTestDict creates a small dictionary file for testing.
func buildTestDict(t *testing.T, dir string) string {
	t.Helper()

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
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test dict: %v", err)
	}

	return path
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

// setupTestHandler creates a Handler with a test pool.
func setupTestHandler(t *testing.T) (*Handler, string) {
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

	pool, err := checker.NewPool(dictPath, store)
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}

	return NewHandler(pool, store), dir
}

func TestHealthEndpoint(t *testing.T) {
	handler, _ := setupTestHandler(t)
	server := httptest.NewServer(handler.Routes())
	defer server.Close()

	resp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %s", result["status"])
	}
}

func TestCheckEndpoint_ValidText(t *testing.T) {
	handler, _ := setupTestHandler(t)
	server := httptest.NewServer(handler.Routes())
	defer server.Close()

	reqBody := SpellCheckRequest{
		Text:      "the government announced a new partnership",
		ProfileID: "test-profile",
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := http.Post(
		server.URL+"/check",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result SpellCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result.Misspellings) != 0 {
		t.Errorf("Expected 0 misspellings, got %d", len(result.Misspellings))
	}

	if result.TokenCount == 0 {
		t.Error("Expected token_count > 0")
	}

	if result.CheckedCount == 0 {
		t.Error("Expected checked_count > 0")
	}

	if result.ElapsedMs <= 0 {
		t.Error("Expected elapsed_ms > 0")
	}
}

func TestCheckEndpoint_Misspellings(t *testing.T) {
	handler, _ := setupTestHandler(t)
	server := httptest.NewServer(handler.Routes())
	defer server.Close()

	reqBody := SpellCheckRequest{
		Text:      "the goverment announced a new partnership",
		ProfileID: "test-profile",
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := http.Post(
		server.URL+"/check",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result SpellCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result.Misspellings) != 1 {
		t.Fatalf("Expected 1 misspelling, got %d", len(result.Misspellings))
	}

	ms := result.Misspellings[0]
	if ms.Word != "goverment" {
		t.Errorf("Expected word 'goverment', got %s", ms.Word)
	}

	if len(ms.Suggestions) == 0 {
		t.Fatal("Expected at least 1 suggestion")
	}

	if ms.Suggestions[0].Word != "government" {
		t.Errorf("Expected suggestion 'government', got %s", ms.Suggestions[0].Word)
	}
}

func TestCheckEndpoint_IgnoreTerms(t *testing.T) {
	handler, _ := setupTestHandler(t)
	server := httptest.NewServer(handler.Routes())
	defer server.Close()

	reqBody := SpellCheckRequest{
		Text:        "the goverment announced a new partnership",
		ProfileID:   "test-profile",
		IgnoreTerms: []string{"goverment"},
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := http.Post(
		server.URL+"/check",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	var result SpellCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result.Misspellings) != 0 {
		t.Errorf("Expected 0 misspellings (ignored), got %d", len(result.Misspellings))
	}
}

func TestCheckEndpoint_MaxSuggestions(t *testing.T) {
	handler, _ := setupTestHandler(t)
	server := httptest.NewServer(handler.Routes())
	defer server.Close()

	reqBody := SpellCheckRequest{
		Text:           "the goverment announced a new partnership",
		ProfileID:      "test-profile",
		MaxSuggestions: 2,
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := http.Post(
		server.URL+"/check",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	var result SpellCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(result.Misspellings) != 1 {
		t.Fatalf("Expected 1 misspelling, got %d", len(result.Misspellings))
	}

	// Should have at most 2 suggestions
	if len(result.Misspellings[0].Suggestions) > 2 {
		t.Errorf("Expected at most 2 suggestions, got %d", len(result.Misspellings[0].Suggestions))
	}
}

func TestCheckEndpoint_InvalidJSON(t *testing.T) {
	handler, _ := setupTestHandler(t)
	server := httptest.NewServer(handler.Routes())
	defer server.Close()

	resp, err := http.Post(
		server.URL+"/check",
		"application/json",
		bytes.NewBufferString("not valid json"),
	)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestCheckEndpoint_UnescapedQuotesInJSON(t *testing.T) {
	handler, _ := setupTestHandler(t)
	server := httptest.NewServer(handler.Routes())
	defer server.Close()

	// This is what happens when user types quotes into Swagger UI without escaping
	// The JSON is invalid because quotes aren't escaped
	invalidJSON := `{"text": "He said "hello world" to me"}`

	resp, err := http.Post(
		server.URL+"/check",
		"application/json",
		bytes.NewBufferString(invalidJSON),
	)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	// Check that the error message is helpful
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte("escape")) {
		t.Errorf("Expected error message to mention escaping quotes, got: %s", string(body))
	}
}

func TestCheckEndpoint_EscapedQuotesInJSON(t *testing.T) {
	handler, _ := setupTestHandler(t)
	server := httptest.NewServer(handler.Routes())
	defer server.Close()

	// This is the correct way to send quotes in JSON
	validJSON := `{"text": "He said \"hello world\" to me"}`

	resp, err := http.Post(
		server.URL+"/check",
		"application/json",
		bytes.NewBufferString(validJSON),
	)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result SpellCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should process successfully
	if result.TokenCount == 0 {
		t.Error("Expected token_count > 0")
	}
}

func TestCheckEndpoint_Base64Text(t *testing.T) {
	handler, _ := setupTestHandler(t)
	server := httptest.NewServer(handler.Routes())
	defer server.Close()

	// Base64 encode: "the goverment announced a new partnership"
	encodedText := base64.StdEncoding.EncodeToString([]byte("the goverment announced a new partnership"))

	reqBody := map[string]string{
		"text_base64": encodedText,
		"profile_id":  "test-profile",
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := http.Post(
		server.URL+"/check",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result SpellCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should find the misspelling
	if len(result.Misspellings) != 1 {
		t.Fatalf("Expected 1 misspelling, got %d", len(result.Misspellings))
	}

	if result.Misspellings[0].Word != "goverment" {
		t.Errorf("Expected word 'goverment', got %s", result.Misspellings[0].Word)
	}
}

func TestCheckEndpoint_Base64WithQuotes(t *testing.T) {
	handler, _ := setupTestHandler(t)
	server := httptest.NewServer(handler.Routes())
	defer server.Close()

	// Base64 encode text with quotes: text outside quotes should be checked,
	// but text inside quotes should be skipped (treated as quoted string)
	textWithQuotes := `the goverment announced "a new test" partnership`
	encodedText := base64.StdEncoding.EncodeToString([]byte(textWithQuotes))

	reqBody := map[string]string{
		"text_base64": encodedText,
		"profile_id":  "test-profile",
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := http.Post(
		server.URL+"/check",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result SpellCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should find the misspelling "goverment" but skip "a new test" inside quotes
	if len(result.Misspellings) != 1 {
		t.Fatalf("Expected 1 misspelling (goverment outside quotes), got %d", len(result.Misspellings))
	}

	if result.Misspellings[0].Word != "goverment" {
		t.Errorf("Expected word 'goverment', got %s", result.Misspellings[0].Word)
	}
}

func TestCheckEndpoint_InvalidBase64(t *testing.T) {
	handler, _ := setupTestHandler(t)
	server := httptest.NewServer(handler.Routes())
	defer server.Close()

	reqBody := map[string]string{
		"text_base64": "!!!not-valid-base64!!!",
		"profile_id":  "test-profile",
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := http.Post(
		server.URL+"/check",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte("base64")) {
		t.Errorf("Expected error message to mention base64, got: %s", string(body))
	}
}

func TestCheckEndpoint_MissingText(t *testing.T) {
	handler, _ := setupTestHandler(t)
	server := httptest.NewServer(handler.Routes())
	defer server.Close()

	// Test with both text and text_base64 missing
	reqBody := map[string]string{
		"profile_id": "test-profile",
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := http.Post(
		server.URL+"/check",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestCheckEndpoint_MultipleMisspellings(t *testing.T) {
	handler, _ := setupTestHandler(t)
	server := httptest.NewServer(handler.Routes())
	defer server.Close()

	reqBody := SpellCheckRequest{
		Text:      "the goverment announcd a new partnarship",
		ProfileID: "test-profile",
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := http.Post(
		server.URL+"/check",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	var result SpellCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should find at least 2 misspellings
	if len(result.Misspellings) < 2 {
		t.Errorf("Expected at least 2 misspellings, got %d", len(result.Misspellings))
	}
}

func TestDocsEndpoint(t *testing.T) {
	handler, _ := setupTestHandler(t)
	server := httptest.NewServer(handler.Routes())
	defer server.Close()

	resp, err := http.Get(server.URL + "/docs")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "text/html" {
		t.Errorf("Expected Content-Type text/html, got %s", resp.Header.Get("Content-Type"))
	}

	// Check that it contains Swagger UI
	body := make([]byte, 1000)
	n, _ := resp.Body.Read(body)
	if n > 0 && !bytes.Contains(body[:n], []byte("swagger-ui")) {
		t.Error("Response does not contain Swagger UI")
	}
}

func TestSwaggerJSONEndpoint(t *testing.T) {
	handler, _ := setupTestHandler(t)
	server := httptest.NewServer(handler.Routes())
	defer server.Close()

	resp, err := http.Get(server.URL + "/swagger.json")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", resp.Header.Get("Content-Type"))
	}

	// Check that it's valid JSON with expected fields
	var swagger map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&swagger); err != nil {
		t.Fatalf("Failed to decode swagger JSON: %v", err)
	}

	if swagger["openapi"] != "3.0.0" {
		t.Error("Swagger JSON missing or incorrect openapi version")
	}
}

func TestCheckEndpoint_RepeatedWords(t *testing.T) {
	handler, _ := setupTestHandler(t)
	server := httptest.NewServer(handler.Routes())
	defer server.Close()

	reqBody := SpellCheckRequest{
		Text:      "the the goverment announced a new partnership",
		ProfileID: "test-profile",
	}

	jsonBody, _ := json.Marshal(reqBody)
	resp, err := http.Post(
		server.URL+"/check",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result SpellCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should find 1 misspelling (goverment) and 1 repeated word (the)
	if len(result.Misspellings) != 1 {
		t.Errorf("Expected 1 misspelling, got %d", len(result.Misspellings))
	}

	if len(result.RepeatedWords) != 1 {
		t.Fatalf("Expected 1 repeated word, got %d", len(result.RepeatedWords))
	}

	if result.RepeatedWords[0].Word != "the" {
		t.Errorf("Expected repeated word 'the', got %s", result.RepeatedWords[0].Word)
	}
}
