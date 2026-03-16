package checker

import (
	"fmt"
	"log"
	"sync"

	"github.com/PaulDavidTucker/SpellChecker/internal/allowlist"

	symspell "github.com/snapp-incubator/go-symspell"
	"github.com/snapp-incubator/go-symspell/pkg/options"
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
}

// NewPool builds a Checker for each profile in the store, plus
// a fallback checker with only the base dictionary.
func NewPool(
	dictPath string,
	store *allowlist.Store,
) (*Pool, error) {
	p := &Pool{
		checkers: make(map[string]*Checker),
		dictPath: dictPath,
		store:    store,
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
func (p *Pool) buildIndex(
	extraTerms []string,
) (symspell.SymSpell, error) {

	ss := symspell.NewSymSpell(
		options.WithCountThreshold(1),
		options.WithMaxDictionaryEditDistance(maxEditDistance),
		options.WithPrefixLength(prefixLength),
	)

	// Load the main dictionary file
	// Column 0 = word, Column 1 = frequency
	ok, err := ss.LoadDictionary(p.dictPath, 0, 1, " ")
	if err != nil {
		return nil, fmt.Errorf(
			"loading dictionary %s: %w", p.dictPath, err,
		)
	}
	if !ok {
		return nil, fmt.Errorf(
			"failed to load dictionary %s", p.dictPath,
		)
	}

	// Note: This library version doesn't support adding individual
	// dictionary entries at runtime. Allowlist terms are checked
	// separately in the Checker.Check method.
	_ = extraTerms

	return ss, nil
}
