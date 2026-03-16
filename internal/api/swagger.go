package api

// swaggerUIHTML is the Swagger UI HTML page
const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SpellChecker API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css" />
    <style>
        html { box-sizing: border-box; overflow: -moz-scrollbars-vertical; overflow-y: scroll; }
        *, *:before, *:after { box-sizing: inherit; }
        body { margin: 0; background: #fafafa; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
        window.onload = function() {
            SwaggerUIBundle({
                url: '/swagger.json',
                dom_id: '#swagger-ui',
                presets: [
                    SwaggerUIBundle.presets.apis
                ],
                layout: 'BaseLayout'
            });
        };
    </script>
</body>
</html>`

// swaggerJSON is the OpenAPI/Swagger specification
const swaggerJSON = `{
  "openapi": "3.0.0",
  "info": {
    "title": "SpellChecker API",
    "description": "A spell checking service with profile-based allowlists",
    "version": "1.0.0"
  },
  "servers": [
    {
      "url": "/",
      "description": "Local server"
    }
  ],
  "paths": {
    "/health": {
      "get": {
        "summary": "Health check",
        "description": "Returns the health status of the service",
        "responses": {
          "200": {
            "description": "Service is healthy",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "status": {
                      "type": "string",
                      "example": "ok"
                    }
                  }
                }
              }
            }
          }
        }
      }
    },
    "/check": {
      "post": {
        "summary": "Check text for misspellings",
        "description": "Analyzes text and returns any misspelled words with suggestions. Note: When including quotes in the text field, they must be escaped with backslashes (e.g., \"He said \\\"hello\\\" to me\").",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "$ref": "#/components/schemas/SpellCheckRequest"
              },
              "examples": {
                "simple": {
                  "summary": "Simple spell check",
                  "value": {
                    "text": "The goverment announced a new partnership"
                  }
                },
                "with-quotes": {
                  "summary": "Text containing quotes (must be escaped)",
                  "value": {
                    "text": "He said \"hello world\" to me"
                  }
                },
                "with-base64": {
                  "summary": "Using base64 encoding (recommended for complex text)",
                  "value": {
                    "text_base64": "SGUgc2FpZCAiaGVsbG8gd29ybGQiIHRvIG1l",
                    "profile_id": "bbc-news"
                  }
                },
                "with-profile": {
                  "summary": "With profile and ignore terms",
                  "value": {
                    "text": "The goverment announced a new partnership with Starmer",
                    "profile_id": "bbc-news",
                    "ignore_terms": ["customword"],
                    "max_suggestions": 3
                  }
                },
                "with-repeated-words": {
                  "summary": "Text with repeated words (e.g., 'the the')",
                  "value": {
                    "text": "The the quick brown fox"
                  }
                }
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "Spell check results",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/SpellCheckResponse"
                }
              }
            }
          },
          "400": {
            "description": "Invalid request",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "error": {
                      "type": "string"
                    }
                  }
                }
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "SpellCheckRequest": {
        "type": "object",
        "properties": {
          "text": {
            "type": "string",
            "description": "The text to check for misspellings. Either 'text' or 'text_base64' must be provided.",
            "example": "The goverment announced a new partnership"
          },
          "text_base64": {
            "type": "string",
            "description": "Base64-encoded text. Use this for large text blocks or text with complex characters. Either 'text' or 'text_base64' must be provided.",
            "example": "SGUgc2FpZCAiaGVsbG8gd29ybGQiIHRvIG1l"
          },
          "profile_id": {
            "type": "string",
            "description": "Profile to use for allowlist checking",
            "example": "bbc-news"
          },
          "locale": {
            "type": "string",
            "description": "Locale code (reserved for future use)",
            "example": "en-GB"
          },
          "content_type": {
            "type": "string",
            "description": "Content type (reserved for future use)",
            "example": "article"
          },
          "ignore_terms": {
            "type": "array",
            "description": "Words to ignore during checking",
            "items": {
              "type": "string"
            },
            "example": ["technical-term", "acronym"]
          },
          "max_suggestions": {
            "type": "integer",
            "description": "Maximum number of suggestions per misspelling",
            "default": 3,
            "example": 5
          }
        }
      },
      "SpellCheckResponse": {
        "type": "object",
        "properties": {
          "misspellings": {
            "type": "array",
            "description": "List of misspelled words found",
            "items": {
              "$ref": "#/components/schemas/Misspelling"
            }
          },
          "repeated_words": {
            "type": "array",
            "description": "List of words that appear twice in a row (e.g., 'the the')",
            "items": {
              "$ref": "#/components/schemas/RepeatedWord"
            }
          },
          "token_count": {
            "type": "integer",
            "description": "Total number of tokens in the text",
            "example": 7
          },
          "checked_count": {
            "type": "integer",
            "description": "Number of tokens that were spell-checked",
            "example": 6
          },
          "elapsed_ms": {
            "type": "number",
            "description": "Time taken to process the request in milliseconds",
            "example": 0.5
          }
        }
      },
      "RepeatedWord": {
        "type": "object",
        "description": "A word that appears twice consecutively",
        "properties": {
          "word": {
            "type": "string",
            "description": "The repeated word",
            "example": "the"
          },
          "offset": {
            "type": "integer",
            "description": "Byte offset where the second occurrence starts",
            "example": 4
          },
          "line": {
            "type": "integer",
            "description": "Line number (1-based)",
            "example": 1
          },
          "column": {
            "type": "integer",
            "description": "Column number (1-based)",
            "example": 5
          }
        }
      },
      "Misspelling": {
        "type": "object",
        "properties": {
          "word": {
            "type": "string",
            "description": "The misspelled word",
            "example": "goverment"
          },
          "offset": {
            "type": "integer",
            "description": "Byte offset in the text where the word starts",
            "example": 4
          },
          "line": {
            "type": "integer",
            "description": "Line number (1-based)",
            "example": 1
          },
          "column": {
            "type": "integer",
            "description": "Column number (1-based)",
            "example": 5
          },
          "suggestions": {
            "type": "array",
            "description": "Suggested corrections",
            "items": {
              "$ref": "#/components/schemas/Suggestion"
            }
          }
        }
      },
      "Suggestion": {
        "type": "object",
        "properties": {
          "word": {
            "type": "string",
            "description": "The suggested correction",
            "example": "government"
          },
          "edit_distance": {
            "type": "integer",
            "description": "Levenshtein edit distance from the misspelling",
            "example": 1
          }
        }
      }
    }
  }
}`
