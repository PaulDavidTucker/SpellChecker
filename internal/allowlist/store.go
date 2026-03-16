package allowlist

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Store holds the base allowlist and all profile-specific
// allowlists in memory.
//
// Concurrent reads are safe. Writes (hot reload) acquire an
// exclusive lock via sync.RWMutex.
type Store struct {
	mu       sync.RWMutex
	base     map[string]struct{}            // lowercased
	profiles map[string]map[string]struct{} // profile_id → terms
	raw      map[string][]string            // original case

	profilesDir string
	onReload    func() // called after successful reload
}

type baseAllowlistFile struct {
	Terms []string `yaml:"terms"`
}

// NewStore loads the base allowlist and all profile YAML files.
func NewStore(
	basePath string,
	profilesDir string,
) (*Store, error) {
	s := &Store{
		base:        make(map[string]struct{}),
		profiles:    make(map[string]map[string]struct{}),
		raw:         make(map[string][]string),
		profilesDir: profilesDir,
	}

	if err := s.loadBase(basePath); err != nil {
		return nil, fmt.Errorf("loading base allowlist: %w", err)
	}

	if err := s.loadProfiles(profilesDir); err != nil {
		return nil, fmt.Errorf("loading profiles: %w", err)
	}

	return s, nil
}

// OnReload registers a callback that fires after profiles are
// successfully reloaded. Used by the checker pool to rebuild
// SymSpell indices.
func (s *Store) OnReload(fn func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onReload = fn
}

func (s *Store) loadBase(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf(
			"Warning: no base allowlist at %s: %v",
			path, err,
		)
		return nil
	}

	var file baseAllowlistFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.base = make(map[string]struct{}, len(file.Terms))
	for _, term := range file.Terms {
		s.base[strings.ToLower(term)] = struct{}{}
	}

	log.Printf("Loaded %d base allowlist terms", len(file.Terms))
	return nil
}

func (s *Store) loadProfiles(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading profiles dir: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear existing profiles so removed files are reflected
	s.profiles = make(map[string]map[string]struct{})
	s.raw = make(map[string][]string)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("Warning: skipping %s: %v", path, err)
			continue
		}

		var profile Profile
		if err := yaml.Unmarshal(data, &profile); err != nil {
			log.Printf("Warning: skipping %s: %v", path, err)
			continue
		}

		if profile.ID == "" {
			log.Printf(
				"Warning: skipping %s: no profile_id", path,
			)
			continue
		}

		termSet := make(
			map[string]struct{},
			len(profile.Terms),
		)
		for _, term := range profile.Terms {
			termSet[strings.ToLower(term)] = struct{}{}
		}

		s.profiles[profile.ID] = termSet
		s.raw[profile.ID] = profile.Terms

		log.Printf(
			"Loaded profile %q with %d terms",
			profile.ID,
			len(profile.Terms),
		)
	}

	return nil
}

// IsAllowed checks whether a word is permitted by the profile
// or base allowlist. Word must be pre-lowercased.
func (s *Store) IsAllowed(word string, profileID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if profileID != "" {
		if terms, ok := s.profiles[profileID]; ok {
			if _, found := terms[word]; found {
				return true
			}
		}
	}

	_, found := s.base[word]
	return found
}

// ProfileIDs returns all loaded profile IDs.
func (s *Store) ProfileIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.profiles))
	for id := range s.profiles {
		ids = append(ids, id)
	}
	return ids
}

// ProfileTerms returns the original-case terms for a profile.
// Returns a copy so the caller can't mutate internal state.
func (s *Store) ProfileTerms(profileID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	terms, ok := s.raw[profileID]
	if !ok {
		return nil
	}

	result := make([]string, len(terms))
	copy(result, terms)
	return result
}

// AllProfileTerms returns every allowlist term across the base
// and all profiles, deduplicated, in original case. Used when
// building SymSpell indices.
func (s *Store) AllProfileTerms(profileID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := make(map[string]struct{})
	var result []string

	// Base terms (stored lowercased, which is fine for SymSpell)
	for term := range s.base {
		if _, exists := seen[term]; !exists {
			seen[term] = struct{}{}
			result = append(result, term)
		}
	}

	// Profile-specific terms (use original case from raw)
	if terms, ok := s.raw[profileID]; ok {
		for _, term := range terms {
			lower := strings.ToLower(term)
			if _, exists := seen[lower]; !exists {
				seen[lower] = struct{}{}
				result = append(result, lower)
			}
		}
	}

	return result
}

// ProfileCount returns the number of loaded profiles.
func (s *Store) ProfileCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.profiles)
}

// GetProfile returns a profile by ID
func (s *Store) GetProfile(profileID string) (*Profile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	terms, ok := s.raw[profileID]
	if !ok {
		return nil, false
	}

	return &Profile{
		ID:          profileID,
		Description: "", // Could store descriptions separately if needed
		Terms:       append([]string{}, terms...),
	}, true
}

// CreateProfile creates a new profile with the given ID and terms
func (s *Store) CreateProfile(profile Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if profile.ID == "" {
		return fmt.Errorf("profile_id is required")
	}

	// Check if profile already exists
	if _, exists := s.profiles[profile.ID]; exists {
		return fmt.Errorf("profile %q already exists", profile.ID)
	}

	// Add to memory
	termSet := make(map[string]struct{}, len(profile.Terms))
	for _, term := range profile.Terms {
		termSet[strings.ToLower(term)] = struct{}{}
	}
	s.profiles[profile.ID] = termSet
	s.raw[profile.ID] = profile.Terms

	// Persist to disk
	if err := s.saveProfile(profile); err != nil {
		// Rollback memory changes
		delete(s.profiles, profile.ID)
		delete(s.raw, profile.ID)
		return fmt.Errorf("saving profile: %w", err)
	}

	// Trigger reload callback
	if s.onReload != nil {
		s.onReload()
	}

	return nil
}

// UpdateProfile updates an existing profile
func (s *Store) UpdateProfile(profile Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if profile.ID == "" {
		return fmt.Errorf("profile_id is required")
	}

	// Check if profile exists
	if _, exists := s.profiles[profile.ID]; !exists {
		return fmt.Errorf("profile %q not found", profile.ID)
	}

	// Update memory
	termSet := make(map[string]struct{}, len(profile.Terms))
	for _, term := range profile.Terms {
		termSet[strings.ToLower(term)] = struct{}{}
	}
	s.profiles[profile.ID] = termSet
	s.raw[profile.ID] = profile.Terms

	// Persist to disk
	if err := s.saveProfile(profile); err != nil {
		return fmt.Errorf("saving profile: %w", err)
	}

	// Trigger reload callback
	if s.onReload != nil {
		s.onReload()
	}

	return nil
}

// DeleteProfile removes a profile
func (s *Store) DeleteProfile(profileID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if profile exists
	if _, exists := s.profiles[profileID]; !exists {
		return fmt.Errorf("profile %q not found", profileID)
	}

	// Remove from memory
	delete(s.profiles, profileID)
	delete(s.raw, profileID)

	// Remove file from disk
	path := filepath.Join(s.profilesDir, profileID+".yaml")
	if err := os.Remove(path); err != nil {
		// Try to rollback
		// (simplified - in production you'd reload from disk)
		return fmt.Errorf("removing profile file: %w", err)
	}

	// Trigger reload callback
	if s.onReload != nil {
		s.onReload()
	}

	return nil
}

// AddTermToProfile adds a single term to an existing profile
func (s *Store) AddTermToProfile(profileID, term string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	terms, ok := s.raw[profileID]
	if !ok {
		return fmt.Errorf("profile %q not found", profileID)
	}

	// Check if term already exists
	lowerTerm := strings.ToLower(term)
	for _, existing := range terms {
		if strings.ToLower(existing) == lowerTerm {
			return nil // Already exists, no error
		}
	}

	// Add to memory
	terms = append(terms, term)
	s.raw[profileID] = terms
	s.profiles[profileID][lowerTerm] = struct{}{}

	// Persist
	profile := Profile{
		ID:          profileID,
		Description: "",
		Terms:       terms,
	}
	if err := s.saveProfile(profile); err != nil {
		return fmt.Errorf("saving profile: %w", err)
	}

	// Trigger reload callback
	if s.onReload != nil {
		s.onReload()
	}

	return nil
}

// RemoveTermFromProfile removes a single term from a profile
func (s *Store) RemoveTermFromProfile(profileID, term string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	terms, ok := s.raw[profileID]
	if !ok {
		return fmt.Errorf("profile %q not found", profileID)
	}

	// Find and remove term
	lowerTerm := strings.ToLower(term)
	found := false
	newTerms := make([]string, 0, len(terms))
	for _, existing := range terms {
		if strings.ToLower(existing) != lowerTerm {
			newTerms = append(newTerms, existing)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("term %q not found in profile", term)
	}

	// Update memory
	s.raw[profileID] = newTerms
	delete(s.profiles[profileID], lowerTerm)

	// Persist
	profile := Profile{
		ID:          profileID,
		Description: "",
		Terms:       newTerms,
	}
	if err := s.saveProfile(profile); err != nil {
		return fmt.Errorf("saving profile: %w", err)
	}

	// Trigger reload callback
	if s.onReload != nil {
		s.onReload()
	}

	return nil
}

// saveProfile persists a profile to disk
func (s *Store) saveProfile(profile Profile) error {
	path := filepath.Join(s.profilesDir, profile.ID+".yaml")
	data, err := yaml.Marshal(profile)
	if err != nil {
		return fmt.Errorf("marshaling profile: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing profile file: %w", err)
	}

	return nil
}
