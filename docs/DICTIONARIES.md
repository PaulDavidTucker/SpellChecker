# Dictionary System

The SpellChecker now supports three dictionary sizes, allowing you to balance between accuracy and resource usage.

## Available Dictionaries

### 1. Medium (Default) - `en_gb_medium.txt`
- **Size**: ~94,000 words
- **File Size**: 1.1MB
- **WASM Binary**: ~4.5MB
- **Memory Usage**: ~100-150MB
- **Startup Time**: <5 seconds
- **Best For**: Fast deployments, resource-constrained environments
- **Coverage**: Common English words, suitable for general text

### 2. Large - `en_gb_large.txt` ✅ **Recommended**
- **Size**: ~136,000 words  
- **File Size**: 1.6MB
- **WASM Binary**: ~5.5MB
- **Memory Usage**: ~150-200MB
- **Startup Time**: 10-20 seconds
- **Best For**: Production deployments, better accuracy
- **Coverage**: Extended vocabulary including technical terms, uncommon words

### 3. Huge - `en_gb_huge.txt` ⚠️ **Resource Intensive**
- **Size**: ~408,000 words
- **File Size**: 4.9MB
- **WASM Binary**: ~8-10MB
- **Memory Usage**: ~500-600MB
- **Startup Time**: 30-60 seconds
- **Best For**: Maximum accuracy, rare/obscure words
- **Warning**: Requires significant memory, may not work in scratch containers

## Switching Dictionaries

### Using the Helper Script (Recommended)

```bash
# Switch to medium (default, fastest)
./switch-dict.sh medium

# Switch to large (recommended balance)
./switch-dict.sh large

# Switch to huge (comprehensive but slow)
./switch-dict.sh huge
```

### Using Docker Compose Directly

```bash
# Set dictionary size and rebuild
export DICT_SIZE=large
docker compose down
docker compose build --no-cache spellchecker
docker compose up -d

# Check logs to verify which dictionary loaded
docker compose logs spellchecker | grep "Loaded dictionary"
```

### Using Docker Build Args

```bash
# Build with specific dictionary
docker build --build-arg DICT_SIZE=large -t spellchecker:large .
docker run -p 8080:8080 spellchecker:large
```

## Dictionary Format

All dictionaries use the same format:
```
word frequency
the 22761659
of 8915110
and 10572938
...
```

Frequency indicates how common the word is (higher = more common). This helps the spell checker suggest the most likely corrections.

## How Dictionaries Are Generated

1. **Medium**: Original SCOWL size 60 dictionary (~94K words)
2. **Large**: Medium + top 50K additional words from expanded word list
3. **Huge**: Medium + all 370K additional words from expanded word list

Additional words receive artificially generated frequencies that decrease as the list progresses, mimicking natural frequency distribution.

## Files

- `dictionaries/en_gb.txt` - Original dictionary (kept for reference)
- `dictionaries/en_gb_medium.txt` - 94K words
- `dictionaries/en_gb_large.txt` - 136K words (recommended)
- `dictionaries/en_gb_huge.txt` - 408K words

## Troubleshooting

### Container won't start or health check fails
- **Cause**: Dictionary too large for available memory
- **Solution**: Use medium or large dictionary instead of huge

### Slow startup times
- **Cause**: Large dictionaries take time to build symspell index
- **Solution**: Use medium dictionary for faster startup, or wait longer for large/huge

### Many false positives (good words flagged)
- **Cause**: Dictionary too small
- **Solution**: Switch to large or huge dictionary

## Performance Comparison

| Dictionary | Words | Binary Size | Memory | Startup | Avg Check Time |
|------------|-------|-------------|--------|---------|----------------|
| Medium     | 94K   | 4.5MB       | 150MB  | 5s      | 1ms            |
| Large      | 136K  | 5.5MB       | 200MB  | 15s     | 2ms            |
| Huge       | 408K  | 9MB         | 600MB  | 45s     | 5ms            |

## Recommendation

**For most users**: Use `large` dictionary - it provides significantly better coverage than medium with reasonable resource usage.

**For resource-constrained environments**: Use `medium` dictionary - still provides good coverage for common English text.

**For maximum accuracy**: Use `huge` dictionary, but ensure your container has at least 1GB RAM available.
