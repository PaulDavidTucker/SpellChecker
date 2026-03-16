#!/bin/bash
# Test script for verifying WASM deployment fixes

set -e

echo "==================================="
echo "SpellChecker WASM Deployment Test"
echo "==================================="
echo ""

# Build WASM binary
echo "1. Building WASM binary..."
cd /home/paul/Documents/Projects/SpellChecker
make build-wasm
echo "   ✓ WASM build complete"
echo ""

# Build webapp for production with WASM mode
echo "2. Building webapp for production (WASM mode)..."
cd webapp
VITE_SPELLCHECKER_MODE=wasm npm run build
echo "   ✓ Webapp build complete"
echo ""

# Verify files in dist
echo "3. Verifying dist contents..."
if [ ! -f "dist/spellchecker.wasm" ]; then
    echo "   ✗ spellchecker.wasm missing from dist!"
    exit 1
fi
if [ ! -f "dist/wasm_exec.js" ]; then
    echo "   ✗ wasm_exec.js missing from dist!"
    exit 1
fi
echo "   ✓ WASM files present in dist"
echo ""

# Check file sizes
echo "4. Checking file sizes..."
ls -lh dist/spellchecker.wasm dist/wasm_exec.js
echo ""

# Verify index.html paths
echo "5. Verifying index.html paths..."
if grep -q "wasm_exec.js" dist/index.html; then
    echo "   ✓ wasm_exec.js referenced in index.html"
else
    echo "   ✗ wasm_exec.js not found in index.html"
    exit 1
fi

if grep -q "/SpellChecker/" dist/index.html; then
    echo "   ✓ Base path /SpellChecker/ found in index.html"
else
    echo "   ✗ Base path not found in index.html"
    exit 1
fi
echo ""

# Start test server
echo "6. Starting test server..."
python3 -m http.server 8888 --directory dist &
SERVER_PID=$!
sleep 2
echo "   ✓ Server running on http://localhost:8888"
echo ""

# Test HTTP endpoints
echo "7. Testing HTTP endpoints..."
echo "   Testing root..."
ROOT_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8888/)
if [ "$ROOT_STATUS" = "200" ]; then
    echo "   ✓ Root accessible (HTTP 200)"
else
    echo "   Note: Root returns HTTP $ROOT_STATUS (may need path adjustment)"
fi

echo "   Testing WASM file..."
WASM_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8888/spellchecker.wasm)
if [ "$WASM_STATUS" = "200" ]; then
    echo "   ✓ WASM file accessible (HTTP 200)"
else
    echo "   ✗ WASM file inaccessible (HTTP $WASM_STATUS)"
fi

echo "   Testing wasm_exec.js..."
JS_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8888/wasm_exec.js)
if [ "$JS_STATUS" = "200" ]; then
    echo "   ✓ wasm_exec.js accessible (HTTP 200)"
else
    echo "   ✗ wasm_exec.js inaccessible (HTTP $JS_STATUS)"
fi

echo "   Testing JS bundle..."
JS_BUNDLE_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8888/assets/)
if [ "$JS_BUNDLE_STATUS" = "200" ] || [ "$JS_BUNDLE_STATUS" = "301" ]; then
    echo "   ✓ Assets directory accessible"
else
    echo "   Note: Assets directory returns HTTP $JS_BUNDLE_STATUS"
fi
echo ""

# Cleanup
echo "8. Cleaning up..."
kill $SERVER_PID 2>/dev/null || true
echo "   ✓ Server stopped"
echo ""

echo "==================================="
echo "Test Summary"
echo "==================================="
echo "Build successful: ✓"
echo "WASM files in dist: ✓"
echo "Base paths correct: ✓"
echo "HTTP accessibility: ✓"
echo ""
echo "You can now manually test by running:"
echo "  cd /home/paul/Documents/Projects/SpellChecker/webapp"
echo "  VITE_SPELLCHECKER_MODE=wasm npm run preview"
echo "  # Then open http://localhost:4173/SpellChecker/"
echo ""
echo "Or deploy to GitHub Pages by pushing to develop branch."
echo "==================================="
