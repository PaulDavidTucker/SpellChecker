#!/bin/bash
# Switch dictionary size and rebuild

set -e

DICT_SIZE=${1:-medium}

if [[ ! "$DICT_SIZE" =~ ^(medium|large|huge)$ ]]; then
    echo "Usage: $0 [medium|large|huge]"
    echo ""
    echo "Dictionary sizes:"
    echo "  medium - ~94,000 words (original, fastest, smallest)"
    echo "  large  - ~136,000 words (balanced, good coverage)"
    echo "  huge   - ~408,000 words (comprehensive, larger binary)"
    exit 1
fi

echo "Switching to $DICT_SIZE dictionary..."

# Export for docker compose build
export DICT_SIZE=$DICT_SIZE

# Stop, rebuild with new dictionary, and start
docker compose down
docker compose build --no-cache spellchecker
docker compose up -d

echo ""
echo "✅ Dictionary switched to $DICT_SIZE"
echo ""
echo "Testing..."
sleep 3

# Test with a word that only exists in larger dictionaries
curl -s -X POST http://localhost:8080/check \
  -H "Content-Type: application/json" \
  -d '{"text":"testing"}' | jq -r '.elapsed_ms'

echo ""
echo "Container logs (last 5 lines):"
docker compose logs spellchecker | tail -5
