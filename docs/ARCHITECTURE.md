# Architecture Overview

## System Architecture

The SpellChecker service is built with a modular, layered architecture designed for performance and maintainability.

```
┌─────────────────────────────────────────────────────────────┐
│                        HTTP API Layer                        │
│                    (internal/api/handler.go)                 │
├─────────────────────────────────────────────────────────────┤
│                     Request/Response Types                   │
│                    (internal/api/request.go)                 │
├─────────────────────────────────────────────────────────────┤
│                      Spell Check Core                        │
│                   (internal/checker/*.go)                    │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │  Tokenizer  │  │  Allowlist  │  │     SymSpell        │ │
│  │  (scanner)  │  │   (store)   │  │   (dictionary)      │ │
│  └─────────────┘  └─────────────┘  └─────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## Component Breakdown

### 1. HTTP API Layer

**File:** `internal/api/handler.go`

Handles HTTP requests and responses:
- Request validation (JSON parsing, field validation)
- Response formatting (JSON encoding)
- Logging middleware (request logging with timing)
- Swagger documentation endpoints

**Key Features:**
- Supports both plain `text` and `text_base64` for complex content
- Provides helpful error messages for common mistakes
- Request logging with method, path, status code, and duration

### 2. Spell Check Core

**Files:**
- `internal/checker/checker.go` - Main orchestration
- `internal/checker/result.go` - Result types
- `internal/checker/pool.go` - Profile management

The heart of the system. Manages the spell-checking pipeline:
- Tokenizes input text
- Applies multi-layer filtering (ignore terms, allowlists, dictionary)
- Generates suggestions using SymSpell
- Detects repeated words

**Pipeline Flow:**

```
1. Tokenize input
        ↓
2. For each token:
   a. Check if checkable (skip URLs, emails, etc.)
   b. Check for repeated words (compare with previous token)
   c. Check ignore terms list
   d. Check allowlist (profile + base)
   e. Look up in SymSpell dictionary
   f. If not found, collect suggestions
        ↓
3. Return results (misspellings + repeated words)
```

### 3. Tokenizer

**Files:**
- `internal/tokenizer/scanner.go` - State machine scanner
- `internal/tokenizer/token.go` - Token types
- `internal/tokenizer/skipzones.go` - URL/email detection

Single-pass tokenizer that classifies text into tokens:

**Token Types:**
- `TokenWord` - Standard words (spell-checked)
- `TokenContraction` - "don't", "it's" (checked as units)
- `TokenHyphenated` - "well-known" (components checked separately)
- `TokenAcronym` - "BBC", "HTML" (skipped)
- `TokenNumber` - "42", "3.14" (skipped)
- `TokenAlphaNumeric` - "MP3", "h264" (skipped)
- `TokenURL` - URLs (skipped)
- `TokenEmail` - Email addresses (skipped)
- `TokenQuoted` - Quoted strings (skipped)
- `TokenPunctuation` - Punctuation (skipped)

**Special Handling:**
- Apostrophes in contractions vs possessives
- Hyphenated compounds
- Acronym detection (all caps, 2+ chars)
- Pluralised acronyms ("MPs", "CEOs")

### 4. Allowlist System

**Files:**
- `internal/allowlist/store.go` - Store management
- `internal/allowlist/profile.go` - Profile types
- `internal/allowlist/watcher.go` - File watching

Manages allowed terms per profile:
- YAML-based configuration
- Base allowlist + profile-specific terms
- Hot-reloading via fsnotify
- Thread-safe access with RWMutex

**Profile Structure:**
```yaml
profile_id: bbc-news
description: "BBC News style profile"
terms:
  - Brexit
  - Westminster
  - Downing Street
```

### 5. Dictionary Loader

**File:** `internal/dictionary/loader.go`

Loads word lists with frequency data:
- Space-separated format: `word frequency`
- Case normalization (lowercase)
- Default frequency of 1 for words without frequency
- Supports 94,000+ word English dictionary

### 6. SymSpell Integration

**Library:** `github.com/snapp-incubator/go-symspell`

Provides fuzzy string matching:
- Levenshtein edit distance (max 2)
- Prefix indexing for fast lookups
- Frequency-based ranking
- Returns top N suggestions

**Configuration:**
- `MaxDictionaryEditDistance: 2`
- `PrefixLength: 15` (increased for longer words)
- `CountThreshold: 10`

## Data Flow

```
┌──────────┐     ┌──────────┐     ┌──────────┐
│  HTTP    │────→│  Token   │────→│  Check   │
│ Request  │     │  Scanner │     │ Pipeline │
└──────────┘     └──────────┘     └────┬─────┘
                                        │
                      ┌────────────────┼────────────────┐
                      ↓                ↓                ↓
                 ┌────────┐      ┌──────────┐    ┌─────────┐
                 │ Ignore │      │ Allowlist│    │SymSpell │
                 │ Terms  │      │  Store   │    │  Lookup │
                 └────────┘      └──────────┘    └─────────┘
                      │                │                │
                      └────────────────┼────────────────┘
                                       ↓
                                 ┌──────────┐
                                 │ Suggest  │
                                 │  Engine  │
                                 └────┬─────┘
                                      ↓
                              ┌───────────────┐
                              │  HTTP Response │
                              │  (JSON)       │
                              └───────────────┘
```

## Performance Optimizations

### 1. Single-Pass Tokenization

The scanner processes text in one pass, avoiding multiple iterations:
- Time: O(n) where n = text length
- Memory: O(m) where m = number of tokens

### 2. Pre-computed Indices

Dictionary and allowlists are loaded once at startup:
- Checker pool builds indices per profile
- SymSpell creates deletion indices for fast lookup
- No disk I/O during request processing

### 3. Concurrent Safety

- RWMutex for allowlist store (many readers, few writers)
- Read-only checker instances (safe for concurrent use)
- Profile pool provides isolated checkers

### 4. Fast Set Lookups

Ignore terms and allowlists use Go maps:
- O(1) average lookup time
- Pre-allocated capacity to avoid resizing

### 5. Efficient String Operations

- Rune-based iteration for UTF-8 safety
- String interning where appropriate
- Minimal allocations in hot paths

## Scalability

### Vertical Scaling

- CPU-bound: dictionary lookups
- Memory-bound: loaded dictionaries
- Profile count limited by memory

### Horizontal Scaling

Stateless design allows multiple instances:
- No shared state between requests
- Each instance loads its own dictionary
- Can run behind load balancer

## Error Handling

### Graceful Degradation

- Dictionary load failures: Fatal (can't run without dictionary)
- Profile load failures: Log warning, continue
- SymSpell errors: Skip word, don't fail entire request
- Invalid JSON: Return 400 with helpful message

### Validation Layers

1. **HTTP Layer**: JSON parsing, required fields
2. **Checker Layer**: Token validation
3. **SymSpell Layer**: Word lookup (tolerates errors)

## Testing Strategy

### Unit Tests

- Tokenizer: 40+ test cases covering edge cases
- Checker: 20+ test cases for pipeline
- Dictionary loader: 11 test cases
- Allowlist store: Hot-reload tests

### Integration Tests

- HTTP handler: 12 test cases
- End-to-end: Full request/response cycle
- Base64 encoding: Complex text handling

### Performance Tests

- Benchmarks for tokenization
- Memory profiling
- Real-world text samples (harness)

## Future Enhancements

See [TODO.md](TODO.md) for planned features:
- Metrics endpoint (Prometheus)
- Batch processing endpoint
- Caching layer
- WebSocket support
- Dictionary management API
