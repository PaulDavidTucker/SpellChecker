/**
 * Build-time mode detection for headless/WASM vs API mode
 * 
 * This module provides compile-time constants for determining the spell checker mode,
 * enabling tree-shaking and dead code elimination in WASM mode.
 */

// Mode from build-time environment variable
export const BUILD_MODE = import.meta.env.VITE_SPELLCHECKER_MODE || 'auto';

// Compile-time constants (these will be tree-shaken by the bundler)
export const IS_WASM_MODE = BUILD_MODE === 'wasm';
export const IS_API_MODE = BUILD_MODE === 'api';
export const IS_AUTO_MODE = BUILD_MODE === 'auto';

/**
 * Get the effective mode considering runtime state
 * In auto mode, we check if WASM is ready
 */
export function getEffectiveMode(wasmReady: boolean): 'wasm' | 'api' {
  if (IS_WASM_MODE) return 'wasm';
  if (IS_API_MODE) return 'api';
  // auto mode
  return wasmReady ? 'wasm' : 'api';
}

/**
 * Check if profiles are supported
 * Profiles only work in API mode
 */
export function areProfilesSupported(): boolean {
  // In WASM mode, profiles are never supported
  if (IS_WASM_MODE) return false;
  // In API mode, always supported
  if (IS_API_MODE) return true;
  // In auto mode, we don't know until runtime - assume not supported if WASM is ready
  return false;
}
