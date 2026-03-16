# API Documentation

## Overview

The SpellChecker service provides a REST API for spell checking text with profile-based allowlists. It supports detecting misspellings, repeated words, and provides suggestions for corrections.

## Base URL

```
http://localhost:8080
```

## Endpoints

### 1. Health Check

Check if the service is running and healthy.

**Endpoint:** `GET /health`

**Response:**
```json
{
  "status": "ok"
}
```

**Status Codes:**
- `200 OK` - Service is healthy

---

### 2. Spell Check

Check text for misspellings and repeated words.

**Endpoint:** `POST /check`

**Content-Type:** `application/json`

**Request Body:**

```json
{
  "text": "The goverment announced a new partnership",
  "text_base64": "SGUgc2FpZCAiaGVsbG8gd29ybGQiIHRvIG1l",
  "profile_id": "bbc-news",
  "ignore_terms": ["technical-term", "acronym"],
  "max_suggestions": 3
}
```

**Parameters:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `text` | string | No* | The text to check for misspellings |
| `text_base64` | string | No* | Base64-encoded text (use for complex text with quotes) |
| `profile_id` | string | No | Profile to use for allowlist checking (default: "default") |
| `ignore_terms` | array | No | Words to ignore during checking |
| `max_suggestions` | integer | No | Maximum suggestions per misspelling (default: 3) |

*Either `text` or `text_base64` must be provided, not both.

**Response:**

```json
{
  "misspellings": [
    {
      "word": "goverment",
      "offset": 4,
      "line": 1,
      "column": 5,
      "suggestions": [
        {
          "word": "government",
          "edit_distance": 1
        }
      ]
    }
  ],
  "repeated_words": [
    {
      "word": "the",
      "offset": 4,
      "line": 1,
      "column": 5
    }
  ],
  "token_count": 7,
  "checked_count": 6,
  "elapsed_ms": 0.5
}
```

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `misspellings` | array | List of misspelled words found |
| `repeated_words` | array | List of words appearing twice consecutively |
| `token_count` | integer | Total number of tokens in the text |
| `checked_count` | integer | Number of tokens that were spell-checked |
| `elapsed_ms` | float | Time taken to process the request in milliseconds |

**Status Codes:**
- `200 OK` - Request successful
- `400 Bad Request` - Invalid JSON or missing required fields

---

### 3. API Documentation (Swagger UI)

Interactive API documentation with Swagger UI.

**Endpoint:** `GET /docs`

Opens a web interface where you can explore and test the API endpoints.

---

### 4. Swagger Specification

Raw OpenAPI/Swagger specification in JSON format.

**Endpoint:** `GET /swagger.json`

## Examples

### Example 1: Simple Spell Check

```bash
curl -X POST http://localhost:8080/check \
  -H "Content-Type: application/json" \
  -d '{
    "text": "The goverment announced a new partnership"
  }'
```

### Example 2: Text with Quotes (using Base64)

```bash
# Encode text with quotes
echo -n 'He said "hello world" to me' | base64
# Result: SGUgc2FpZCAiaGVsbG8gd29ybGQiIHRvIG1l

curl -X POST http://localhost:8080/check \
  -H "Content-Type: application/json" \
  -d '{
    "text_base64": "SGUgc2FpZCAiaGVsbG8gd29ybGQiIHRvIG1l",
    "profile_id": "bbc-news"
  }'
```

### Example 3: With Ignore Terms

```bash
curl -X POST http://localhost:8080/check \
  -H "Content-Type: application/json" \
  -d '{
    "text": "The BBC announcd a new partnership",
    "profile_id": "bbc-news",
    "ignore_terms": ["BBC"]
  }'
```

## Error Handling

### Invalid JSON

```json
{
  "error": "invalid JSON: make sure to escape quotes inside text"
}
```

### Missing Text Field

```json
{
  "error": "text or text_base64 field is required"
}
```

### Invalid Base64

```json
{
  "error": "invalid base64 encoding"
}
```

## Token Types

The spell checker handles various token types:

- **Words** - Standard words that are spell-checked
- **Contractions** - "don't", "can't", "it's" - checked as units
- **Hyphenated** - "well-known" - components checked separately
- **Acronyms** - "BBC", "HTML" - skipped
- **Numbers** - "42", "3.14" - skipped
- **AlphaNumeric** - "MP3", "h264" - skipped
- **URLs** - "https://..." - skipped
- **Emails** - "a@b.com" - skipped
- **Quoted Strings** - Text in quotes - skipped
- **Punctuation** - ".", "," - skipped

## Features

### 1. Profile-Based Allowlists

Different profiles can have different allowed terms. For example, the "bbc-news" profile allows terms like "Brexit", "Westminster", etc.

### 2. Repeated Word Detection

Automatically detects words that appear twice in a row (e.g., "the the").

### 3. Context-Aware Checking

- Contractions are handled properly ("don't" vs "don" + "t")
- Hyphenated words are split and checked
- URLs and emails are skipped
- Quoted text is skipped

### 4. Suggestions

Returns up to 3 suggestions per misspelling with Levenshtein edit distance.

### 5. Position Information

Each issue includes:
- `offset` - Byte offset in the original text
- `line` - Line number (1-indexed)
- `column` - Column number (1-indexed)

## Performance

- Single-pass tokenization
- Concurrent profile loading
- In-memory dictionary for fast lookups
- Average response time: < 1ms for short texts

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DICT_PATH` | `/dictionaries/en_gb.txt` | Path to dictionary file |
| `BASE_ALLOWLIST_PATH` | `/config/base-allowlist.yaml` | Base allowlist config |
| `PROFILES_DIR` | `/config/profiles` | Directory with profile configs |
| `LISTEN_ADDR` | `:8080` | HTTP server address |

## Docker

### Build

```bash
docker build -t spellchecker .
```

### Run

```bash
docker run -p 8080:8080 spellchecker
```

### Docker Compose

```bash
docker-compose up
```

The service will be available at `http://localhost:8080`.
