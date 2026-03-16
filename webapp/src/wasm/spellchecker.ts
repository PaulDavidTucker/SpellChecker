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

class SpellCheckerWASM {
  private state: SpellCheckerState = {
    isReady: false,
    isLoading: false,
    error: null,
    version: null,
  };
  private loadPromise: Promise<void> | null = null;

  /**
   * Initialize the WASM spell checker
   */
  async initialize(wasmPath: string = '/spellchecker.wasm'): Promise<void> {
    if (this.loadPromise) {
      return this.loadPromise;
    }

    this.loadPromise = this.loadWasm(wasmPath);
    return this.loadPromise;
  }

  private async loadWasm(wasmPath: string): Promise<void> {
    this.state.isLoading = true;
    this.state.error = null;

    try {
      // Load the Go WASM runtime
      const go = new (window as any).Go();

      // Fetch and instantiate the WASM module
      const response = await fetch(wasmPath);
      if (!response.ok) {
        throw new Error(`Failed to load WASM: ${response.status} ${response.statusText}`);
      }

      const wasmBytes = await response.arrayBuffer();
      const result = await WebAssembly.instantiate(wasmBytes, go.importObject);

      // Run the Go program
      go.run(result.instance);

      // Wait for initialization
      await this.waitForReady();

      this.state.isReady = true;
      this.state.version = (window as any).spellchecker?.version || 'unknown';
      
      console.log('WASM SpellChecker initialized:', this.state.version);
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
          reject(new Error('WASM initialization timeout'));
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
      throw new Error('SpellChecker API not available');
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
