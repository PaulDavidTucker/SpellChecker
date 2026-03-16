#!/bin/bash

# Build WASM spell checker for browser deployment
set -e

echo "Building WASM Spell Checker..."

# Create directories
mkdir -p webapp/public
mkdir -p dist

# Copy WASM exec helper (if available)
if [ -f /usr/share/go-1.22/misc/wasm/wasm_exec.js ]; then
    cp /usr/share/go-1.22/misc/wasm/wasm_exec.js webapp/public/
    echo "✓ Copied wasm_exec.js"
elif [ -f /usr/local/go/misc/wasm/wasm_exec.js ]; then
    cp /usr/local/go/misc/wasm/wasm_exec.js webapp/public/
    echo "✓ Copied wasm_exec.js"
else
    echo "⚠ wasm_exec.js not found in standard locations"
fi

# Build Go WASM binary
echo "Compiling Go to WASM..."
GOOS=js GOARCH=wasm go build -o webapp/public/spellchecker.wasm ./cmd/wasm

if [ $? -eq 0 ]; then
    echo "✓ WASM build successful"
    
    # Get file size
    SIZE=$(du -h webapp/public/spellchecker.wasm | cut -f1)
    echo "  Size: $SIZE"
else
    echo "✗ WASM build failed"
    exit 1
fi

echo ""
echo "WASM files ready in webapp/public/"
echo "  - spellchecker.wasm"
echo "  - wasm_exec.js"
echo ""
echo "Next steps:"
echo "  1. cd webapp && npm run build"
echo "  2. Deploy dist/ folder to GitHub Pages"
