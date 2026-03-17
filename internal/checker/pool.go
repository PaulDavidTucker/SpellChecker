package checker

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/PaulDavidTucker/SpellChecker/internal/allowlist"

	symspell "github.com/snapp-incubator/go-symspell"
)

const (
	maxEditDistance = 2
	prefixLength    = 15
	allowlistFreq   = 1000
)

// Pool holds pre-built Checker instances keyed by profile ID.
// It provides the right Checker for each incoming request based
// on the profile_id in the request body.
type Pool struct {
	mu       sync.RWMutex
	checkers map[string]*Checker
	fallback *Checker

	dictPath string
	store    *allowlist.Store
	cache    *BinaryCache
}

// NewPool builds a Checker for each profile in the store, plus
// a fallback checker with only the base dictionary.
func NewPool(
	dictPath string,
	store *allowlist.Store,
) (*Pool, error) {
	// Initialize cache
	cacheDir := os.Getenv("CACHE_DIR")
	if cacheDir == "" {
		cacheDir = "/tmp/spellcheck-cache"
	}

	p := &Pool{
		checkers: make(map[string]*Checker),
		dictPath: dictPath,
		store:    store,
		cache:    NewBinaryCache(cacheDir),
	}

	if err := p.build(); err != nil {
		return nil, err
	}

	// Register for hot-reload notifications
	store.OnReload(func() {
		log.Println("Allowlist changed, rebuilding checker pool")
		if err := p.rebuild(); err != nil {
			log.Printf("Error rebuilding pool: %v", err)
		}
	})

	return p, nil
}

// Get returns the Checker for a given profile, falling back to
// the base checker if the profile isn't found.
func (p *Pool) Get(profileID string) *Checker {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if c, ok := p.checkers[profileID]; ok {
		return c
	}
	return p.fallback
}

// build constructs all checkers. Called at startup.
func (p *Pool) build() error {
	// Build fallback checker (base dictionary only)
	fallbackSS, err := p.buildIndex(nil)
	if err != nil {
		return fmt.Errorf("building fallback index: %w", err)
	}
	p.fallback = NewChecker(fallbackSS, p.store, "")

	// Build one checker per profile
	for _, profileID := range p.store.ProfileIDs() {
		extraTerms := p.store.AllProfileTerms(profileID)
		ss, err := p.buildIndex(extraTerms)
		if err != nil {
			return fmt.Errorf(
				"building index for %q: %w", profileID, err,
			)
		}
		p.checkers[profileID] = NewChecker(ss, p.store, profileID)
		log.Printf(
			"Built checker for profile %q (%d extra terms)",
			profileID, len(extraTerms),
		)
	}

	return nil
}

// rebuild reconstructs all checkers after an allowlist change.
// Builds new checkers first, then swaps them in atomically.
func (p *Pool) rebuild() error {
	// Build everything into temporary maps
	newCheckers := make(map[string]*Checker)

	fallbackSS, err := p.buildIndex(nil)
	if err != nil {
		return fmt.Errorf("rebuilding fallback index: %w", err)
	}
	newFallback := NewChecker(fallbackSS, p.store, "")

	for _, profileID := range p.store.ProfileIDs() {
		extraTerms := p.store.AllProfileTerms(profileID)
		ss, err := p.buildIndex(extraTerms)
		if err != nil {
			return fmt.Errorf(
				"rebuilding index for %q: %w", profileID, err,
			)
		}
		newCheckers[profileID] = NewChecker(
			ss, p.store, profileID,
		)
	}

	// Atomic swap — only hold the write lock for the pointer swap
	p.mu.Lock()
	p.checkers = newCheckers
	p.fallback = newFallback
	p.mu.Unlock()

	log.Printf("Rebuilt checker pool with %d profiles",
		len(newCheckers))
	return nil
}

// buildIndex creates a SymSpell instance loaded with the base
// dictionary and optional extra terms (from allowlists).
// Uses cache if available to speed up loading.
func (p *Pool) buildIndex(
	extraTerms []string,
) (symspell.SymSpell, error) {
	// Use cache to build index
	return p.cache.BuildIndex(p.dictPath)
}
