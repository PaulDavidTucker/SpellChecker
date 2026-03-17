package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/PaulDavidTucker/SpellChecker/internal/checker"
)

// This tool pre-builds SymSpell indices and saves cache markers
// Run during Docker build to warm up the dictionary cache
func main() {
	var (
		dictPath = flag.String("dict", "", "Path to dictionary file")
		cacheDir = flag.String("cache-dir", "/tmp/spellcheck-cache", "Cache directory")
	)
	flag.Parse()

	if *dictPath == "" {
		log.Fatal("Usage: warmup-cache -dict=<path> [-cache-dir=<dir>]")
	}

	// Ensure dictionary exists
	if _, err := os.Stat(*dictPath); err != nil {
		log.Fatalf("Dictionary not found: %v", err)
	}

	fmt.Printf("Warming up cache for %s...\n", *dictPath)
	start := time.Now()

	// Create cache
	cache := checker.NewBinaryCache(*cacheDir)

	// Build index (this is the slow part we're caching)
	_, err := cache.BuildIndex(*dictPath)
	if err != nil {
		log.Fatalf("Failed to build index: %v", err)
	}

	elapsed := time.Since(start)
	fmt.Printf("✓ Cache warmed up in %v\n", elapsed)

	// List cache files
	base := filepath.Base(*dictPath)
	pattern := filepath.Join(*cacheDir, base+"_*")
	matches, _ := filepath.Glob(pattern)
	for _, match := range matches {
		info, _ := os.Stat(match)
		fmt.Printf("  Cache file: %s (%d bytes)\n", filepath.Base(match), info.Size())
	}
}
