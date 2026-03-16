# Build WASM binary for browser
build-wasm:
	@echo "Building WASM Spell Checker..."
	GOOS=js GOARCH=wasm go build -o webapp/public/spellchecker.wasm ./cmd/wasm
	@echo "WASM build complete: webapp/public/spellchecker.wasm"

# Build webapp for production (WASM mode)
build-webapp: build-wasm
	cd webapp && npm run build
	@echo "Webapp build complete: webapp/dist/"

# Copy wasm_exec.js (one-time setup)
copy-wasm-exec:
	@if [ -f /usr/share/go-1.22/misc/wasm/wasm_exec.js ]; then \
		cp /usr/share/go-1.22/misc/wasm/wasm_exec.js webapp/public/; \
		echo "✓ Copied wasm_exec.js"; \
	elif [ -f /usr/local/go/misc/wasm/wasm_exec.js ]; then \
		cp /usr/local/go/misc/wasm/wasm_exec.js webapp/public/; \
		echo "✓ Copied wasm_exec.js"; \
	else \
		echo "⚠ wasm_exec.js not found. Please copy manually from your Go installation."; \
	fi

# Clean build artifacts
clean:
	rm -f webapp/public/spellchecker.wasm
	rm -rf webapp/dist
	@echo "Cleaned build artifacts"

# Deploy to GitHub Pages (requires gh CLI)
deploy-github:
	@echo "Building for GitHub Pages..."
	$(MAKE) build-wasm
	cd webapp && npm run build
	@echo "Deploying to GitHub Pages..."
	gh pages deploy webapp/dist --source-dir=webapp/dist

# Full development setup
dev-setup: copy-wasm-exec
	cd webapp && npm install
	@echo "✓ Development setup complete"

# Run development server with API backend
dev:
	docker-compose up -d
	cd webapp && npm run dev

.PHONY: build-wasm build-webapp copy-wasm-exec clean deploy-github dev-setup dev
