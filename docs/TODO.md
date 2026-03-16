TODO:

```
✅ Token types and classification
✅ Scanner state machine (single-pass, UTF-8 safe)
✅ Skip zones for URLs and emails
✅ Edge cases: contractions, hyphens, acronyms, pluralised
acronyms, numbers, alphanumeric
✅ Line/column/byte offset tracking
✅ Comprehensive test suite
✅ Allowlist store with YAML profiles
✅ Hot reload via fsnotify
✅ RWMutex for concurrent safety
✅ Dockerfile

Phase 1
  ✅ Token types
  ✅ Scanner state machine
  ✅ Skip zones
  ✅ Test suite
  → Run tests, fix any failures, iterate

Phase 2 — Allowlist system
  ✅ allowlist/profile.go
  ✅ allowlist/store.go
  ✅ allowlist/watcher.go
  ✅ Write profile YAML files for bbc-news, default
  ✅ Test store loading and IsAllowed

Phase 3 — SymSpell integration
  ✅ go get github.com/snapp-incubator/go-symspell
  ✅ Obtain en_GB dictionary in word\tfrequency format
  ✅ checker/pool.go — build indices per profile
  ✅ checker/checker.go — the pipeline orchestrator
  ✅ checker/result.go — result types (done)

Phase 4 — HTTP layer
  ✅ Handler in api/ package
  ✅ JSON request/response types
  ✅ Wire everything in main.go
  ✅ Integration test: POST real text, check response

Phase 5 — Docker and deployment
  ✅ Bake dictionary + config into image
  ✅ Health check endpoint
  ✅ Integrate into docker-compose
  ✅ Build a test scaffold for validation
  ✅ Put together a test dataset
  ✅ Add a CLI
```

Nice to have in the future:

High value, low effort:

- ✅ Repeated word detection — "the the" is trivial to catch during the token iteration loop. Compare each word token to the previous one. Cheap and catches a very common typo.

- Capitalisation after full stops — if a word follows . and is lowercase, flag it. Not spelling per se, but cheap to check and useful for editorial content.

- Metrics endpoint — expose GET /metrics with Prometheus-compatible counters: requests served, average latency, cache hit rate. Useful for monitoring in a docker-compose setup.

Medium value, medium effort:

- Batch endpoint — POST /check-batch that takes an array of texts. Fan out with goroutines, one per text, collect results. This is where Go's concurrency model would genuinely shine.

- Confidence scoring — instead of just returning suggestions, return a confidence that the word is actually misspelt. A word 1 edit distance from a very common word is more likely a typo than a word 2 edits from a rare word. The frequency data gives you this.

- Context-aware possessive handling — "BBC's" is currently a contraction token. The checker should strip 's and check "BBC" — which then gets caught by the acronym path in the allowlist. This needs a small amount of logic in the checker, not the tokenizer.

Lower priority but interesting:

- Caching layer — if the same article gets checked multiple times (e.g. during editing), cache the result keyed by a hash of the text + profile. An in-memory LRU cache with a short TTL.

- WebSocket endpoint — for real-time checking as someone types in an editor. Each message is a paragraph or sentence, responses stream back.

- Dictionary management API — POST /profiles/{id}/terms to add words to a profile without editing YAML files. Persists to the YAML files on disk so the change survives restarts.
