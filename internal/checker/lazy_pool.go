package checker

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/PaulDavidTucker/SpellChecker/internal/allowlist"
)

// LazyPool loads dictionaries asynchronously to enable fast server startup
type LazyPool struct {
	mu       sync.RWMutex
	checkers map[string]*Checker
	fallback *Checker
	ready    bool
	loading  bool
	loadTime time.Duration

	dictPath string
	store    *allowlist.Store
	cache    *BinaryCache
}

// NewLazyPool creates a pool that loads dictionaries asynchronously
func NewLazyPool(dictPath string, store *allowlist.Store) (*LazyPool, error) {
	cacheDir := os.Getenv("CACHE_DIR")
	if cacheDir == "" {
		cacheDir = "/tmp/spellcheck-cache"
	}

	p := &LazyPool{
		checkers: make(map[string]*Checker),
		dictPath: dictPath,
		store:    store,
		cache:    NewBinaryCache(cacheDir),
		ready:    false,
		loading:  false,
	}

	// Start with a nil checker - will return empty results until loaded
	p.fallback = NewChecker(nil, store, "")

	// Begin background loading
	go p.loadAsync()

	// Register for hot-reload notifications
	store.OnReload(func() {
		fmt.Println("Allowlist changed, triggering async rebuild...")
		go p.rebuildAsync()
	})

	return p, nil
}

// IsReady returns true when the dictionary has finished loading
func (p *LazyPool) IsReady() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.ready
}

// GetLoadTime returns how long the dictionary took to load
func (p *LazyPool) GetLoadTime() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.loadTime
}

// WaitForReady blocks until the dictionary is loaded (with timeout)
func (p *LazyPool) WaitForReady(timeout time.Duration) bool {
	start := time.Now()
	for !p.IsReady() {
		if time.Since(start) > timeout {
			return false
		}
		time.Sleep(100 * time.Millisecond)
	}
	return true
}

// loadAsync loads dictionaries in the background
func (p *LazyPool) loadAsync() {
	p.mu.Lock()
	if p.loading {
		p.mu.Unlock()
		return
	}
	p.loading = true
	p.mu.Unlock()

	fmt.Printf("🔄 Background loading dictionary from %s...\n", p.dictPath)
	start := time.Now()

	// Build the index using cache
	ss, err := p.cache.BuildIndex(p.dictPath)
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to load dictionary: %v\n", err)
		// Keep using nil checker - server stays up but returns empty
		return
	}

	// Build fallback checker
	fallback := NewChecker(ss, p.store, "")

	// Build profile checkers
	checkers := make(map[string]*Checker)
	for _, profileID := range p.store.ProfileIDs() {
		checkers[profileID] = NewChecker(ss, p.store, profileID)
	}

	elapsed := time.Since(start)
	fmt.Printf("✅ Dictionary loaded in %v (%d profiles ready)\n",
		elapsed, len(checkers))

	// Swap in the loaded checkers
	p.mu.Lock()
	p.fallback = fallback
	p.checkers = checkers
	p.loadTime = elapsed
	p.ready = true
	p.loading = false
	p.mu.Unlock()
}

// rebuildAsync rebuilds checkers after allowlist changes
func (p *LazyPool) rebuildAsync() {
	p.mu.RLock()
	if !p.ready {
		p.mu.RUnlock()
		fmt.Println("Cannot rebuild: dictionary not yet loaded")
		return
	}

	// Get reference to current index
	currentFallback := p.fallback
	p.mu.RUnlock()

	if currentFallback == nil || currentFallback.ss == nil {
		fmt.Println("Cannot rebuild: no dictionary index available")
		return
	}

	fmt.Println("🔄 Rebuilding profile checkers...")

	// Rebuild profile checkers with same index
	checkers := make(map[string]*Checker)
	for _, profileID := range p.store.ProfileIDs() {
		checkers[profileID] = NewChecker(currentFallback.ss, p.store, profileID)
	}

	p.mu.Lock()
	p.checkers = checkers
	p.mu.Unlock()

	fmt.Printf("✅ Rebuilt %d profile checkers\n", len(checkers))
}

// Get returns the Checker for a given profile
// Falls back to empty checker if not yet loaded
func (p *LazyPool) Get(profileID string) *Checker {
	p.mu.RLock()
	if p.ready {
		defer p.mu.RUnlock()
		if c, ok := p.checkers[profileID]; ok {
			return c
		}
		return p.fallback
	}
	p.mu.RUnlock()

	// Not ready yet - return nil checker
	// This will produce empty results but won't crash
	return p.fallback
}

// ReadyMiddleware returns HTTP middleware that checks if dictionary is loaded
func (p *LazyPool) ReadyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Health check always works, but includes ready status
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		// Check if ready
		if !p.IsReady() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error":"Dictionary still loading, please wait","ready":false,"status":"loading"}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}
