/**
 * WebAssembly SpellChecker Bridge
 * 
 * This module loads the Go WASM binary and provides a JavaScript interface
 * to the spell checking functions.
 */

export interface SpellCheckResult {
  misspellings: Array<{
    word: string;
    offset: number;
    line: number;
    column: number;
    suggestions: Array<{
      word: string;
      edit_distance: number;
    }>;
  }>;
  repeated_words: Array<{
    word: string;
    offset: number;
    line: number;
    column: number;
  }>;
  capitalisation_issues: Array<{
    word: string;
    offset: number;
    line: number;
    column: number;
  }>;
  token_count: number;
  checked_count: number;
  elapsed_ms: number;
}

export interface SpellCheckerState {
  isReady: boolean;
  isLoading: boolean;
  error: string | null;
  version: string | null;
}

interface LoadAttempt {
  attempt: number;
  maxAttempts: number;
  delayMs: number;
}

class SpellCheckerWASM {
  private state: SpellCheckerState = {
    isReady: false,
    isLoading: false,
    error: null,
    version: null,
  };
  private loadPromise: Promise<void> | null = null;

  /**
   * Get the correct WASM path based on current location
   * This handles GitHub Pages base paths correctly
   */
  private getWasmPath(providedPath?: string): string {
    if (providedPath) {
      return providedPath;
    }

    // Get the base URL from Vite's env (e.g., "/SpellChecker/" on GitHub Pages)
    const baseUrl = import.meta.env.BASE_URL || '/';
    
    // Construct the full path
    // Remove trailing slash from base, add leading slash to path if needed
    const cleanBase = baseUrl.endsWith('/') ? baseUrl.slice(0, -1) : baseUrl;
    const wasmPath = 'spellchecker.wasm';
    
    return cleanBase ? `${cleanBase}/${wasmPath}` : `/${wasmPath}`;
  }

  /**
   * Initialize the WASM spell checker with retry logic
   */
  async initialize(wasmPath?: string, maxRetries: number = 3): Promise<void> {
    if (this.loadPromise) {
      return this.loadPromise;
    }

    const resolvedPath = this.getWasmPath(wasmPath);
    this.loadPromise = this.loadWasmWithRetry(resolvedPath, maxRetries);
    return this.loadPromise;
  }

  private async loadWasmWithRetry(wasmPath: string, maxRetries: number): Promise<void> {
    const loadConfig: LoadAttempt = {
      attempt: 1,
      maxAttempts: maxRetries,
      delayMs: 1000,
    };

    while (loadConfig.attempt <= loadConfig.maxAttempts) {
      try {
        await this.loadWasm(wasmPath);
        return; // Success - exit retry loop
      } catch (error) {
        console.warn(`WASM load attempt ${loadConfig.attempt}/${loadConfig.maxAttempts} failed:`, error);
        
        if (loadConfig.attempt >= loadConfig.maxAttempts) {
          throw new Error(
            `Failed to load WASM after ${loadConfig.maxAttempts} attempts. ` +
            `Last error: ${error instanceof Error ? error.message : 'Unknown error'}`
          );
        }

        // Wait before retry with exponential backoff
        await this.delay(loadConfig.delayMs);
        loadConfig.delayMs *= 2; // Exponential backoff
        loadConfig.attempt++;
      }
    }
  }

  private delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  private async loadWasm(wasmPath: string): Promise<void> {
    this.state.isLoading = true;
    this.state.error = null;

    try {
      // Check if Go runtime is available
      if (typeof (window as any).Go !== 'function') {
        throw new Error('Go WASM runtime (wasm_exec.js) not loaded. Ensure the script is in index.html before main.js.');
      }

      // Load the Go WASM runtime
      const go = new (window as any).Go();

      // Fetch and instantiate the WASM module
      console.log(`Fetching WASM from: ${wasmPath}`);
      const response = await fetch(wasmPath);
      if (!response.ok) {
        throw new Error(`Failed to load WASM from ${wasmPath}: ${response.status} ${response.statusText}`);
      }

      const wasmBytes = await response.arrayBuffer();
      
      // Check WebAssembly support
      if (!WebAssembly.validate(wasmBytes)) {
        throw new Error('WASM binary validation failed - possibly corrupted file');
      }

      const result = await WebAssembly.instantiate(wasmBytes, go.importObject);

      // Run the Go program (this starts a goroutine that runs the main function)
      go.run(result.instance);

      // Wait for initialization
      await this.waitForReady(30000); // 30 second timeout for slower connections

      this.state.isReady = true;
      this.state.version = (window as any).spellchecker?.version || 'unknown';
      
      console.log('WASM SpellChecker initialized successfully:', this.state.version);
    } catch (error) {
      this.state.error = error instanceof Error ? error.message : 'Unknown error';
      this.state.isReady = false;
      console.error('Failed to load WASM SpellChecker:', error);
      throw error;
    } finally {
      this.state.isLoading = false;
    }
  }

  private waitForReady(timeoutMs: number = 10000): Promise<void> {
    return new Promise((resolve, reject) => {
      const startTime = Date.now();
      
      const checkReady = () => {
        const spellchecker = (window as any).spellchecker;
        
        if (spellchecker?.isReady) {
          resolve();
          return;
        }

        if (Date.now() - startTime > timeoutMs) {
          reject(new Error(`WASM initialization timeout after ${timeoutMs}ms`));
          return;
        }

        setTimeout(checkReady, 100);
      };

      checkReady();
    });
  }

  /**
   * Check text for spelling errors
   */
  checkText(text: string, profileId: string = 'default'): SpellCheckResult {
    if (!this.state.isReady) {
      throw new Error('SpellChecker not initialized. Call initialize() first.');
    }

    const spellchecker = (window as any).spellchecker;
    if (!spellchecker?.checkText) {
      throw new Error('SpellChecker API not available - wasm_exec.js may not be loaded');
    }

    const result = spellchecker.checkText(text, profileId);
    
    if (typeof result === 'string') {
      return JSON.parse(result);
    }
    
    return result;
  }

  /**
   * Get current state
   */
  getState(): SpellCheckerState {
    // Update state from window object if available
    const spellchecker = (window as any).spellchecker;
    if (spellchecker) {
      this.state.isReady = spellchecker.isReady || false;
      this.state.version = spellchecker.version || this.state.version;
    }
    return { ...this.state };
  }

  /**
   * Check if WASM is supported in this browser
   */
  static isSupported(): boolean {
    return typeof WebAssembly === 'object' && 
           typeof WebAssembly.instantiate === 'function' &&
           typeof (window as any).Go === 'function';
  }
}

// Export singleton instance
export const spellCheckerWASM = new SpellCheckerWASM();

// Export class for custom instances
export { SpellCheckerWASM };

// Default export
export default spellCheckerWASM;
