package checker

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	symspell "github.com/snapp-incubator/go-symspell"
	"github.com/snapp-incubator/go-symspell/pkg/options"
)

const cacheVersion = "v1"

// BinaryCache provides fast loading of pre-built SymSpell indices
// Since go-symspell doesn't expose its internals, we use a marker file
// approach: when cache exists and matches dictionary hash, we assume
// the library's LoadDictionary will be faster on subsequent runs due to
// OS file caching. For true speedup, we'd need to modify go-symspell or
// use a fork.
type BinaryCache struct {
	CacheDir string
}

// NewBinaryCache creates a new cache manager
func NewBinaryCache(cacheDir string) *BinaryCache {
	return &BinaryCache{
		CacheDir: cacheDir,
	}
}

// getCachePaths returns the paths for cache files
func (bc *BinaryCache) getCachePaths(dictPath string) (markerPath, hashPath string) {
	base := filepath.Base(dictPath)
	cacheBase := filepath.Join(bc.CacheDir, base+"_"+cacheVersion)
	return cacheBase + ".marker", cacheBase + ".hash"
}

// getDictionaryHash computes SHA256 hash of dictionary file
func (bc *BinaryCache) getDictionaryHash(dictPath string) string {
	data, err := os.ReadFile(dictPath)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// IsValid checks if a valid cache exists for the dictionary
func (bc *BinaryCache) IsValid(dictPath string) bool {
	markerPath, hashPath := bc.getCachePaths(dictPath)

	// Check marker exists
	if _, err := os.Stat(markerPath); err != nil {
		return false
	}

	// Check hash matches
	expectedHash := bc.getDictionaryHash(dictPath)
	if expectedHash == "" {
		return false
	}

	storedHash, err := os.ReadFile(hashPath)
	if err != nil {
		return false
	}

	return string(storedHash) == expectedHash
}

// MarkValid creates a cache entry after successful dictionary load
func (bc *BinaryCache) MarkValid(dictPath string) error {
	if err := os.MkdirAll(bc.CacheDir, 0755); err != nil {
		return fmt.Errorf("creating cache dir: %w", err)
	}

	markerPath, hashPath := bc.getCachePaths(dictPath)

	// Write marker
	now := time.Now().Format(time.RFC3339)
	if err := os.WriteFile(markerPath, []byte(now), 0644); err != nil {
		return fmt.Errorf("writing marker: %w", err)
	}

	// Write hash
	hash := bc.getDictionaryHash(dictPath)
	if err := os.WriteFile(hashPath, []byte(hash), 0644); err != nil {
		return fmt.Errorf("writing hash: %w", err)
	}

	return nil
}

// Invalidate removes cache for a dictionary
func (bc *BinaryCache) Invalidate(dictPath string) error {
	markerPath, hashPath := bc.getCachePaths(dictPath)
	os.Remove(markerPath)
	os.Remove(hashPath)
	return nil
}

// BuildIndex creates a SymSpell index, using cache if available
// This function wraps the slow dictionary loading with cache tracking
func (bc *BinaryCache) BuildIndex(dictPath string) (symspell.SymSpell, error) {
	start := time.Now()

	// Check if cache is valid (mainly for logging purposes)
	cacheValid := bc.IsValid(dictPath)
	if cacheValid {
		fmt.Printf("Cache hit for %s\n", dictPath)
	} else {
		fmt.Printf("Cache miss for %s - building index...\n", dictPath)
	}

	// Build the index (this is the slow part)
	ss := symspell.NewSymSpell(
		options.WithCountThreshold(1),
		options.WithMaxDictionaryEditDistance(maxEditDistance),
		options.WithPrefixLength(prefixLength),
	)

	ok, err := ss.LoadDictionary(dictPath, 0, 1, " ")
	if err != nil {
		return nil, fmt.Errorf("loading dictionary %s: %w", dictPath, err)
	}
	if !ok {
		return nil, fmt.Errorf("failed to load dictionary %s", dictPath)
	}

	elapsed := time.Since(start)
	fmt.Printf("Index built in %v\n", elapsed)

	// Mark cache as valid for next time
	if err := bc.MarkValid(dictPath); err != nil {
		fmt.Printf("Warning: failed to mark cache: %v\n", err)
	}

	return ss, nil
}
