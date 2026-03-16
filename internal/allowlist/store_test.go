package allowlist

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile is a test helper that writes content to a file.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(
		path, []byte(content), 0644,
	); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func TestLoadBaseAllowlist(t *testing.T) {
	dir := t.TempDir()
	profilesDir := filepath.Join(dir, "profiles")
	os.MkdirAll(profilesDir, 0755)

	basePath := filepath.Join(dir, "base.yaml")
	writeFile(t, basePath, `
terms:
  - Kubernetes
  - Docker
  - gRPC
`)

	store, err := NewStore(basePath, profilesDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	tests := []struct {
		word     string
		expected bool
	}{
		{"kubernetes", true}, // lowercased
		{"docker", true},
		{"grpc", true},
		{"unknown", false},
	}

	for _, tt := range tests {
		got := store.IsAllowed(tt.word, "")
		if got != tt.expected {
			t.Errorf(
				"IsAllowed(%q, \"\"): got %v, want %v",
				tt.word, got, tt.expected,
			)
		}
	}
}

func TestLoadProfile(t *testing.T) {
	dir := t.TempDir()
	profilesDir := filepath.Join(dir, "profiles")
	os.MkdirAll(profilesDir, 0755)

	basePath := filepath.Join(dir, "base.yaml")
	writeFile(t, basePath, `
terms:
  - SharedTerm
`)

	writeFile(t, filepath.Join(profilesDir, "bbc-news.yaml"), `
profile_id: bbc-news
description: "BBC News editorial content"
terms:
  - Starmer
  - Ofcom
  - Brexit
`)

	writeFile(t, filepath.Join(profilesDir, "bbc-sport.yaml"), `
profile_id: bbc-sport
description: "BBC Sport content"
terms:
  - Haaland
  - VAR
`)

	store, err := NewStore(basePath, profilesDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	// Profile count
	if store.ProfileCount() != 2 {
		t.Errorf(
			"expected 2 profiles, got %d",
			store.ProfileCount(),
		)
	}

	// bbc-news profile terms
	if !store.IsAllowed("starmer", "bbc-news") {
		t.Error("expected 'starmer' allowed in bbc-news")
	}
	if !store.IsAllowed("ofcom", "bbc-news") {
		t.Error("expected 'ofcom' allowed in bbc-news")
	}

	// bbc-sport terms NOT allowed in bbc-news
	if store.IsAllowed("haaland", "bbc-news") {
		t.Error("expected 'haaland' NOT allowed in bbc-news")
	}

	// bbc-sport profile terms
	if !store.IsAllowed("haaland", "bbc-sport") {
		t.Error("expected 'haaland' allowed in bbc-sport")
	}

	// Base terms allowed in any profile
	if !store.IsAllowed("sharedterm", "bbc-news") {
		t.Error("expected 'sharedterm' allowed in bbc-news")
	}
	if !store.IsAllowed("sharedterm", "bbc-sport") {
		t.Error("expected 'sharedterm' allowed in bbc-sport")
	}
	if !store.IsAllowed("sharedterm", "") {
		t.Error("expected 'sharedterm' allowed with no profile")
	}
}

func TestProfileTermsReturnsCopy(t *testing.T) {
	dir := t.TempDir()
	profilesDir := filepath.Join(dir, "profiles")
	os.MkdirAll(profilesDir, 0755)

	basePath := filepath.Join(dir, "base.yaml")
	writeFile(t, basePath, "terms: []")

	writeFile(t, filepath.Join(profilesDir, "test.yaml"), `
profile_id: test
terms:
  - Alpha
  - Beta
`)

	store, err := NewStore(basePath, profilesDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	terms := store.ProfileTerms("test")
	if len(terms) != 2 {
		t.Fatalf("expected 2 terms, got %d", len(terms))
	}

	// Mutate the returned slice — should NOT affect the store
	terms[0] = "MUTATED"

	fresh := store.ProfileTerms("test")
	if fresh[0] == "MUTATED" {
		t.Error("ProfileTerms returned a reference, not a copy")
	}
}

func TestProfileIDs(t *testing.T) {
	dir := t.TempDir()
	profilesDir := filepath.Join(dir, "profiles")
	os.MkdirAll(profilesDir, 0755)

	basePath := filepath.Join(dir, "base.yaml")
	writeFile(t, basePath, "terms: []")

	writeFile(t, filepath.Join(profilesDir, "a.yaml"), `
profile_id: alpha
terms: [one]
`)
	writeFile(t, filepath.Join(profilesDir, "b.yaml"), `
profile_id: beta
terms: [two]
`)

	store, err := NewStore(basePath, profilesDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	ids := store.ProfileIDs()
	if len(ids) != 2 {
		t.Fatalf("expected 2 profile IDs, got %d", len(ids))
	}

	// Order isn't guaranteed (map iteration), so check membership
	idSet := make(map[string]struct{})
	for _, id := range ids {
		idSet[id] = struct{}{}
	}
	if _, ok := idSet["alpha"]; !ok {
		t.Error("missing profile ID 'alpha'")
	}
	if _, ok := idSet["beta"]; !ok {
		t.Error("missing profile ID 'beta'")
	}
}

func TestMissingBaseAllowlist(t *testing.T) {
	dir := t.TempDir()
	profilesDir := filepath.Join(dir, "profiles")
	os.MkdirAll(profilesDir, 0755)

	// Base file doesn't exist — should warn but not fail
	store, err := NewStore(
		filepath.Join(dir, "nonexistent.yaml"),
		profilesDir,
	)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if store.IsAllowed("anything", "") {
		t.Error("expected nothing allowed with no base file")
	}
}

func TestSkipsInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	profilesDir := filepath.Join(dir, "profiles")
	os.MkdirAll(profilesDir, 0755)

	basePath := filepath.Join(dir, "base.yaml")
	writeFile(t, basePath, "terms: []")

	// Valid profile
	writeFile(t, filepath.Join(profilesDir, "good.yaml"), `
profile_id: good
terms: [valid]
`)

	// Invalid YAML — should be skipped, not crash
	writeFile(
		t,
		filepath.Join(profilesDir, "bad.yaml"),
		"{{{{not yaml at all",
	)

	store, err := NewStore(basePath, profilesDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	if store.ProfileCount() != 1 {
		t.Errorf(
			"expected 1 profile (bad skipped), got %d",
			store.ProfileCount(),
		)
	}

	if !store.IsAllowed("valid", "good") {
		t.Error("expected 'valid' allowed in 'good'")
	}
}

func TestSkipsProfileWithNoID(t *testing.T) {
	dir := t.TempDir()
	profilesDir := filepath.Join(dir, "profiles")
	os.MkdirAll(profilesDir, 0755)

	basePath := filepath.Join(dir, "base.yaml")
	writeFile(t, basePath, "terms: []")

	// Missing profile_id field
	writeFile(t, filepath.Join(profilesDir, "noid.yaml"), `
description: "No ID here"
terms: [orphan]
`)

	store, err := NewStore(basePath, profilesDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	if store.ProfileCount() != 0 {
		t.Errorf("expected 0 profiles, got %d", store.ProfileCount())
	}
}

func TestAllProfileTerms(t *testing.T) {
	dir := t.TempDir()
	profilesDir := filepath.Join(dir, "profiles")
	os.MkdirAll(profilesDir, 0755)

	basePath := filepath.Join(dir, "base.yaml")
	writeFile(t, basePath, `
terms:
  - SharedTerm
  - Docker
`)

	writeFile(t, filepath.Join(profilesDir, "news.yaml"), `
profile_id: bbc-news
terms:
  - Starmer
  - Ofcom
`)

	store, err := NewStore(basePath, profilesDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	terms := store.AllProfileTerms("bbc-news")

	// Should contain base + profile terms, all lowercased
	termSet := make(map[string]struct{})
	for _, term := range terms {
		termSet[term] = struct{}{}
	}

	expected := []string{
		"sharedterm", "docker", "starmer", "ofcom",
	}
	for _, exp := range expected {
		if _, ok := termSet[exp]; !ok {
			t.Errorf("expected term %q in AllProfileTerms", exp)
		}
	}
}

func TestNonYAMLFilesIgnored(t *testing.T) {
	dir := t.TempDir()
	profilesDir := filepath.Join(dir, "profiles")
	os.MkdirAll(profilesDir, 0755)

	basePath := filepath.Join(dir, "base.yaml")
	writeFile(t, basePath, "terms: []")

	// These should be silently ignored
	writeFile(
		t,
		filepath.Join(profilesDir, "readme.md"),
		"# Not a profile",
	)
	writeFile(
		t,
		filepath.Join(profilesDir, ".gitkeep"),
		"",
	)
	writeFile(t, filepath.Join(profilesDir, "valid.yaml"), `
profile_id: valid
terms: [word]
`)

	store, err := NewStore(basePath, profilesDir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	if store.ProfileCount() != 1 {
		t.Errorf(
			"expected 1 profile, got %d",
			store.ProfileCount(),
		)
	}
}
