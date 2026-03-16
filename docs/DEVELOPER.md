# Developer Guide

## Getting Started

### Prerequisites

- Go 1.26 or later
- Docker (optional, for containerized deployment)
- Make (optional, for convenience commands)

### Clone and Setup

```bash
git clone <repository-url>
cd SpellChecker
go mod download
```

## Project Structure

```
SpellChecker/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── allowlist/               # Profile/allowlist management
│   │   ├── profile.go
│   │   ├── store.go
│   │   ├── store_test.go
│   │   └── watcher.go
│   ├── api/                     # HTTP API layer
│   │   ├── handler.go
│   │   ├── handler_test.go
│   │   ├── request.go
│   │   └── swagger.go
│   ├── checker/                 # Spell check core
│   │   ├── checker.go
│   │   ├── checker_test.go
│   │   ├── pool.go
│   │   └── result.go
│   ├── dictionary/              # Dictionary loading
│   │   ├── loader.go
│   │   └── loader_test.go
│   └── tokenizer/               # Text tokenization
│       ├── scanner.go
│       ├── scanner_test.go
│       ├── skipzones.go
│       └── token.go
├── config/
│   ├── base-allowlist.yaml      # Global allowed terms
│   └── profiles/                # Profile configurations
│       ├── bbc-news.yaml
│       ├── bbc-sport.yaml
│       └── default.yaml
├── dictionaries/
│   └── en_gb.txt                # Dictionary file
├── docs/                        # Documentation
│   ├── API.md
│   ├── ARCHITECTURE.md
│   ├── DEPLOYMENT.md
│   └── TODO.md
├── harness/                     # Test harness
│   ├── fixtures.json
│   ├── main.go
│   └── demo_results.html
├── Dockerfile                   # Container build
├── docker-compose.yaml          # Docker Compose config
└── README.md                    # Project overview
```

## Running the Application

### Development Mode

```bash
# Run the server
go run cmd/server/main.go

# Server starts on :8080
# Logs will show loading progress
```

### With Custom Config

```bash
# Set environment variables
export DICT_PATH=./dictionaries/en_gb.txt
export PROFILES_DIR=./config/profiles
export BASE_ALLOWLIST_PATH=./config/base-allowlist.yaml

go run cmd/server/main.go
```

### Production Mode

```bash
# Build binary
go build -o server ./cmd/server

# Run binary
./server
```

## Running Tests

### All Tests

```bash
go test ./...
```

### Specific Package

```bash
go test ./internal/tokenizer -v
go test ./internal/checker -v
go test ./internal/api -v
```

### With Coverage

```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Benchmarks

```bash
go test -bench=. ./internal/tokenizer
go test -bench=. -benchmem ./internal/tokenizer
```

## Test Harness

The harness runs real-world test cases:

```bash
cd harness
go run .

# Generates demo_results.html
# View in browser: open demo_results.html
```

## Code Style

### Formatting

```bash
# Format all code
go fmt ./...

# Or use goimports
goimports -w .
```

### Linting

```bash
# Install linter
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run
```

### Naming Conventions

- **Packages:** lowercase, no underscores (`tokenizer`, `allowlist`)
- **Files:** lowercase with underscores (`scanner_test.go`)
- **Types:** PascalCase (`SpellCheckRequest`)
- **Functions:** PascalCase for exported, camelCase for internal
- **Variables:** camelCase
- **Constants:** UPPER_SNAKE_CASE for exported, camelCase for internal
- **Interfaces:** PascalCase with -er suffix (`Reader`, `Writer`)

### Comments

```go
// Checker runs the spell-checking pipeline for a specific profile.
// It is safe for concurrent use.
type Checker struct {
    // ss is the SymSpell instance for this profile
    ss      symspell.SymSpell
    store   *allowlist.Store
    profile string
}

// Check runs the full spell-checking pipeline.
// It tokenizes the text and checks each token.
func (c *Checker) Check(req CheckRequest) CheckResult {
    // ...
}
```

## Architecture Patterns

### Dependency Injection

Dependencies are injected at construction time:

```go
// NewChecker creates a Checker with its dependencies
func NewChecker(
    ss symspell.SymSpell,
    store *allowlist.Store,
    profileID string,
) *Checker {
    return &Checker{
        ss:      ss,
        store:   store,
        profile: profileID,
    }
}
```

### Interface Segregation

Keep interfaces small and focused:

```go
// Tokenizer interface for text tokenization
type Tokenizer interface {
    Tokenize(text string) []Token
}

// Suggester interface for spelling suggestions
type Suggester interface {
    Suggest(word string, max int) []Suggestion
}
```

### Error Handling

- Return errors, don't panic
- Wrap errors with context
- Handle gracefully, don't fail entire request

```go
suggestions, err := c.ss.Lookup(word, verbosity.All, 2)
if err != nil {
    // Log error but continue with other words
    log.Printf("SymSpell error for %q: %v", word, err)
    continue
}
```

## Adding New Features

### Example: Adding a New Token Type

1. **Update token.go:**

```go
const (
    TokenWord         TokenKind = iota
    // ... existing types
    TokenNewType                   // Add new type
)
```

2. **Update scanner.go:**

```go
func (s *Scanner) scanNewType() Token {
    // Implementation
}
```

3. **Add tests:**

```go
func TestNewType(t *testing.T) {
    tokens := NewScanner("test input").Tokenize()
    // Assertions
}
```

4. **Update documentation**

### Example: Adding a New API Endpoint

1. **Update handler.go:**

```go
func (h *Handler) Routes() http.Handler {
    mux := http.NewServeMux()
    mux.HandleFunc("POST /check", h.handleCheck)
    mux.HandleFunc("POST /new-endpoint", h.handleNewEndpoint)
    // ...
}

func (h *Handler) handleNewEndpoint(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```

2. **Add tests:**

```go
func TestNewEndpoint(t *testing.T) {
    // Test implementation
}
```

3. **Update swagger.go:**

Add endpoint documentation to OpenAPI spec.

## Debugging

### Enable Verbose Logging

```go
// In main.go or relevant file
log.SetFlags(log.LstdFlags | log.Lshortfile)
```

### Debug a Specific Test

```bash
go test -v -run TestSpecificFunction ./internal/checker
```

### Profile CPU/Memory

```bash
# CPU profile
go test -cpuprofile=cpu.prof -bench=. ./internal/checker
go tool pprof cpu.prof

# Memory profile
go test -memprofile=mem.prof -bench=. ./internal/checker
go tool pprof mem.prof
```

### Race Detection

```bash
go test -race ./...
```

## Common Issues

### Issue: Dictionary not loading

**Solution:**
```bash
# Verify dictionary file exists
ls dictionaries/en_gb.txt

# Check file format
cat dictionaries/en_gb.txt | head -5
```

### Issue: Tests failing with "dictionary not found"

**Solution:**
```bash
# Run from project root
cd /home/paul/Documents/Projects/SpellChecker
go test ./...

# Or use absolute paths in tests
```

### Issue: Hot reload not working

**Solution:**
- Check file permissions
- Verify fsnotify is working on your OS
- Look for "WatchAndReload" log messages

## Release Process

### Version Bump

1. Update version in code (if applicable)
2. Update CHANGELOG.md
3. Tag release:

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

### Build Release

```bash
# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o server-linux-amd64 ./cmd/server
GOOS=darwin GOARCH=amd64 go build -o server-darwin-amd64 ./cmd/server
GOOS=windows GOARCH=amd64 go build -o server-windows-amd64.exe ./cmd/server
```

### Docker Release

```bash
# Build with version tag
docker build -t spellchecker:v1.0.0 .
docker tag spellchecker:v1.0.0 spellchecker:latest

# Push to registry
docker push spellchecker:v1.0.0
docker push spellchecker:latest
```

## Contributing

### Before Submitting

1. Run all tests: `go test ./...`
2. Run linter: `golangci-lint run`
3. Format code: `go fmt ./...`
4. Check coverage: `go test -cover ./...`
5. Update documentation if needed

### Pull Request Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing performed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Tests pass
- [ ] Documentation updated
```

## Resources

- [Go Documentation](https://golang.org/doc/)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Project README](../README.md)
- [Architecture Overview](ARCHITECTURE.md)
- [API Documentation](API.md)
- [Deployment Guide](DEPLOYMENT.md)

## Questions?

If you have questions or need help:

1. Check existing documentation
2. Review test cases for examples
3. Look at similar implementations in the codebase
4. Open an issue with your question
