# Application flow

```
POST /check arrives
│
▼
api/handler.go
Parses JSON body into SpellCheckRequest
Calls pool.Get("bbc-news") to get the right Checker
│
▼
checker/pool.go
Returns the pre-built Checker for "bbc-news"
(built at startup with bbc-news allowlist terms
baked into its SymSpell index)
│
▼
checker/checker.go — Check()

1. Builds ignore set from request ignore_terms
2. Creates tokenizer.NewScanner(text)
3. Calls scanner.Tokenize()
   │
   ▼
   tokenizer/scanner.go
   Pre-scans for URLs/emails (skip zones)
   Runs the state machine character by character
   Returns []Token, each classified by kind
   │
   ▼
   checker/checker.go — back in Check()
4. Iterates tokens, skips non-checkable kinds
5. For each checkable token:
   ├── In ignore_terms? → skip
   ├── In allowlist? → skip
   └── SymSpell lookup:
   ├── Distance 0 (exact match) → word is correct, skip
   └── Distance > 0 or no match → misspelling
   └── Collect top N suggestions ranked by
   edit distance then frequency
6. Returns CheckResult
   │
   ▼
   api/handler.go
   Maps CheckResult to JSON response
   Returns to caller
```

The cycle of tokens is quite complex:

```
main.go
  │
  ├── dictionary.Load(paths)
  │     └── returns *Dictionary
  │
  ├── allowlist.NewStore(paths)
  │     └── returns *Store (holds profiles, watches for changes)
  │
  ├── checker.NewPool(dict, store)
  │     ├── For each profile:
  │     │     ├── buildIndex(dict + profile terms) → *SymSpell
  │     │     └── creates Checker{symspell, store, profileID}
  │     └── returns *Pool
  │
  └── api.NewHandler(pool)
        └── handler.Routes() → http.Handler
              │
              POST /check
              │  ├── Parse JSON → SpellCheckRequest
              │  ├── pool.Get(profileID) → *Checker
              │  ├── checker.Check(req)
              │  │     ├── Build ignore set
              │  │     ├── tokenizer.NewScanner(text).Tokenize()
              │  │     ├── For each checkable token:
              │  │     │     ├── Check ignore set
              │  │     │     ├── Check allowlist store
              │  │     │     ├── Check SymSpell index
              │  │     │     └── If miss → collect misspelling
              │  │     └── return CheckResult
              │  └── Marshal → JSON response
              │
              GET /health → {"status":"ok"}
```
