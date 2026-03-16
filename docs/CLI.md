# CLI Documentation

The SpellCheck CLI provides command-line spell checking capabilities with support for profiles, multiple output formats, and configuration management.

## Installation

### Build from Source

```bash
# Clone the repository
git clone <repository-url>
cd SpellChecker

# Build the CLI
go build -o spellcheck-cli ./cmd/cli

# Optional: Install to $GOPATH/bin
go install ./cmd/cli
```

## Quick Start

```bash
# Check text from command line
./spellcheck-cli check "The goverment announced a new partnership"

# Check text from file
./spellcheck-cli check --file article.txt

# Check with specific profile
./spellcheck-cli check --profile bbc-news "The MPs debated Brexit"

# Pipe text via stdin
echo "The goverment announced" | ./spellcheck-cli check
```

## Commands

### `check` - Spell Check Text

Check text for misspellings and repeated words.

**Usage:**
```bash
spellcheck-cli check [text] [flags]
```

**Examples:**
```bash
# Check text from argument
spellcheck-cli check "The goverment announced a new partnership"

# Check text from file
spellcheck-cli check --file article.txt
spellcheck-cli check -f article.txt

# Check with specific profile
spellcheck-cli check --profile bbc-news "The MPs debated Brexit"
spellcheck-cli check -p bbc-news "The MPs debated Brexit"

# Pipe from stdin
echo "The goverment announced" | spellcheck-cli check
cat article.txt | spellcheck-cli check

# Ignore specific terms
spellcheck-cli check --ignore "BBC,MPs" "The BBC announced"
spellcheck-cli check -i "BBC,MPs" "The BBC announced"

# Limit suggestions
spellcheck-cli check --max-suggestions 5 "The goverment announced"
spellcheck-cli check -n 5 "The goverment announced"

# JSON output
spellcheck-cli check --output json "The goverment announced"
spellcheck-cli check -o json "The goverment announced"

# Simple output
spellcheck-cli check --output simple "The goverment announced"
```

**Flags:**

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | `""` | Read text from file |
| `--profile` | `-p` | `"default"` | Profile to use for checking |
| `--output` | `-o` | `"pretty"` | Output format: pretty, simple, json |
| `--ignore` | `-i` | `[]` | Terms to ignore (comma-separated) |
| `--max-suggestions` | `-n` | `3` | Maximum suggestions per misspelling |

**Output Formats:**

**Pretty (default):**
```
╔════════════════════════════════════════════════════════╗
║           Spell Check Results                          ║
╚════════════════════════════════════════════════════════╝

✗ Misspellings Found: 2
──────────────────────────────────────────────────

Word:    goverment
Location: Line 1, Column 5
Suggestions:
  1. government (distance: 1)
  2. movement (distance: 2)
  3. averment (distance: 2)

══════════════════════════════════════════════════
Total tokens: 11 | Checked: 6 | Time: 1.25ms
```

**Simple:**
```
✗ Found 2 misspelling(s):
  Line 1, Col 5: 'goverment'
    Suggestions: government, movement, averment
  Line 1, Col 15: 'announced'
    Suggestions: announce, unannounced

Checked 6 tokens in 1.25ms
```

**JSON:**
```json
{
  "misspellings": [
    {
      "word": "goverment",
      "offset": 4,
      "line": 1,
      "column": 5,
      "suggestions": [
        {"word": "government", "edit_distance": 1},
        {"word": "movement", "edit_distance": 2},
        {"word": "averment", "edit_distance": 2}
      ]
    }
  ],
  "repeated_words": [],
  "token_count": 11,
  "checked_count": 6,
  "elapsed_ms": 1.25
}
```

---

### `profiles` - List Available Profiles

List all available profiles that can be used with the `--profile` flag.

**Usage:**
```bash
spellcheck-cli profiles
```

**Example Output:**
```
Available profiles (2 total):

• default
  Default profile with base allowlist terms

• bbc-news
  Terms: 13
```

---

### `validate` - Validate Configuration

Validate that all configuration files are correct and loadable.

**Usage:**
```bash
spellcheck-cli validate
```

**Checks:**
- Dictionary file exists and is readable
- Base allowlist YAML is valid
- All profile YAML files are valid
- Profile IDs match filenames

**Example Output:**
```
Validating configuration...

✓ Dictionary: dictionaries/en_gb.txt
  Loaded 94231 words

✓ Base allowlist: config/base-allowlist.yaml

✓ Profiles directory: config/profiles
  Found 2 profiles

✓ All configurations are valid!
```

---

### `config` - Show Configuration

Display the current configuration settings.

**Usage:**
```bash
spellcheck-cli config
```

**Example Output:**
```
Current Configuration:

Dictionary path:     dictionaries/en_gb.txt
Base allowlist:      config/base-allowlist.yaml
Profiles directory:  config/profiles
Default profile:     default
```

---

### `version` - Show Version

Display version information.

**Usage:**
```bash
spellcheck-cli version
```

**Example Output:**
```
spellcheck-cli version 1.0.0
  commit: abc1234
  built:  2024-03-16T01:30:00Z
```

---

## Global Flags

These flags are available for all commands:

| Flag | Default | Description |
|------|---------|-------------|
| `--dict` | `dictionaries/en_gb.txt` | Path to dictionary file |
| `--base-allowlist` | `config/base-allowlist.yaml` | Path to base allowlist config |
| `--profiles-dir` | `config/profiles` | Directory containing profile configs |

**Example with Global Flags:**
```bash
# Use custom dictionary
spellcheck-cli --dict=/path/to/custom.txt check "some text"

# Use custom profiles directory
spellcheck-cli --profiles-dir=/custom/profiles profiles
```

---

## Exit Codes

| Exit Code | Meaning |
|-----------|---------|
| `0` | Success (no misspellings found, or command completed successfully) |
| `1` | Misspellings or repeated words found (when running `check` command) |
| `1` | General error (invalid config, file not found, etc.) |

**Usage in Scripts:**
```bash
#!/bin/bash

# Check file and capture exit code
spellcheck-cli check --file article.txt --output simple
EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
    echo "✓ No spelling errors found"
elif [ $EXIT_CODE -eq 1 ]; then
    echo "✗ Spelling errors detected"
    # Prevent commit, fail CI, etc.
    exit 1
else
    echo "? Error occurred"
    exit 1
fi
```

---

## Shell Completion

Generate shell completion scripts for bash, zsh, fish, and PowerShell.

### Bash

```bash
# Add to ~/.bashrc
source <(spellcheck-cli completion bash)

# Or save to file
spellcheck-cli completion bash > /etc/bash_completion.d/spellcheck-cli
```

### Zsh

```bash
# Add to ~/.zshrc
source <(spellcheck-cli completion zsh)

# Or save to file
spellcheck-cli completion zsh > "${fpath[1]}/_spellcheck-cli"
```

### Fish

```bash
spellcheck-cli completion fish > ~/.config/fish/completions/spellcheck-cli.fish
```

---

## Examples

### CI/CD Integration

```yaml
# .github/workflows/spellcheck.yml
name: Spell Check
on: [push, pull_request]

jobs:
  spellcheck:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Build CLI
        run: go build -o spellcheck-cli ./cmd/cli
      
      - name: Check documentation
        run: |
          spellcheck-cli check --file README.md --output simple
          spellcheck-cli check --file docs/API.md --output simple
      
      - name: Check all markdown files
        run: |
          for file in $(find . -name "*.md" -type f); do
            echo "Checking $file..."
            spellcheck-cli check --file "$file" --output simple || true
          done
```

### Git Hook

```bash
#!/bin/bash
# .git/hooks/pre-commit

# Check staged files
for file in $(git diff --cached --name-only --diff-filter=ACM | grep '\.md$'); do
    if ! spellcheck-cli check --file "$file" --output simple 2>/dev/null; then
        echo "Spelling errors in $file. Commit aborted."
        exit 1
    fi
done
```

### Batch Processing

```bash
#!/bin/bash

# Process all text files in a directory
for file in articles/*.txt; do
    echo "Processing $file..."
    spellcheck-cli check --file "$file" --output json > "results/$(basename $file).json"
done
```

---

## Advanced Usage

### Custom Configuration

Create a custom configuration for your project:

```bash
# Create project-specific dictionary
cat > project-dict.txt << 'EOF'
mycompany 1000
myproduct 500
technicalterm 100
EOF

# Create project profile
cat > project-profile.yaml << 'EOF'
profile_id: myproject
description: "My project profile"
terms:
  - mycompany
  - myproduct
  - technicalterm
EOF

# Use custom configuration
spellcheck-cli \
  --dict=./project-dict.txt \
  --profiles-dir=./profiles \
  check --profile myproject "Some text"
```

### Integration with Editors

**Vim:**
```vim
" Add to .vimrc
function! SpellCheck()
    execute '!spellcheck-cli check --file % --output simple'
endfunction

command! SpellCheck call SpellCheck()
```

**VS Code:**
Create a task in `.vscode/tasks.json`:
```json
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "Spell Check",
            "type": "shell",
            "command": "spellcheck-cli",
            "args": ["check", "--file", "${file}", "--output", "simple"],
            "group": "build",
            "presentation": {
                "reveal": "always"
            }
        }
    ]
}
```

---

## Troubleshooting

### Dictionary Not Found

```bash
# Verify dictionary path
spellcheck-cli validate

# Use absolute path
spellcheck-cli --dict=/absolute/path/to/dict.txt check "text"
```

### Profile Not Loading

```bash
# List available profiles
spellcheck-cli profiles

# Validate configuration
spellcheck-cli validate
```

### Large Files

For very large files, consider:
```bash
# Split file into chunks
split -l 1000 large-file.txt chunk-

# Check each chunk
for chunk in chunk-*; do
    spellcheck-cli check --file "$chunk" --output simple
done
```

---

## See Also

- [API Documentation](API.md) - HTTP API reference
- [Architecture Overview](ARCHITECTURE.md) - System design
- [Deployment Guide](DEPLOYMENT.md) - Production deployment
- [Developer Guide](DEVELOPER.md) - Contributing guidelines
