import { useState, useCallback, useEffect } from 'react';
import type { SpellCheckRequest, SpellCheckResponse, Profile } from '../types/api';
import { spellCheckerWASM } from '../wasm/spellchecker';
import { BUILD_MODE, IS_WASM_MODE, IS_API_MODE, getEffectiveMode } from '../config/mode';

// Re-export mode constants for components to use
export { BUILD_MODE, IS_WASM_MODE, IS_API_MODE };

// Mode detection - can be 'wasm', 'api', or 'auto'
const MODE = BUILD_MODE;
const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';

/**
 * Normalize spell check response to ensure all arrays are present
 * Go WASM returns nil slices as null in JSON, but TypeScript expects arrays
 */
function normalizeResponse(response: SpellCheckResponse): SpellCheckResponse {
  return {
    ...response,
    misspellings: response.misspellings || [],
    repeated_words: response.repeated_words || [],
    capitalisation_issues: response.capitalisation_issues || [],
  };
}

export function useSpellCheck() {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [wasmReady, setWasmReady] = useState(false);
  const [wasmLoading, setWasmLoading] = useState(false);
  const [wasmError, setWasmError] = useState<string | null>(null);

  // Initialize WASM on mount if in WASM mode or auto mode
  useEffect(() => {
    if (MODE === 'wasm' || MODE === 'auto') {
      initializeWASM();
    }
  }, []);

  const initializeWASM = async () => {
    if (wasmLoading || wasmReady) return;
    
    setWasmLoading(true);
    setWasmError(null);
    
    try {
      // Pass undefined to use automatic path detection
      await spellCheckerWASM.initialize(undefined, 3);
      setWasmReady(true);
      console.log('WASM spell checker ready');
    } catch (err) {
      console.warn('WASM initialization failed after retries:', err);
      setWasmReady(false);
      setWasmError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setWasmLoading(false);
    }
  };

  const checkText = useCallback(async (
    request: SpellCheckRequest
  ): Promise<SpellCheckResponse | null> => {
    setIsLoading(true);
    setError(null);

    try {
      // Try WASM first if available
      if ((MODE === 'wasm' || MODE === 'auto') && wasmReady) {
        // Use setTimeout to yield to the UI thread so loading state is shown
        await new Promise(resolve => setTimeout(resolve, 0));
        const result = spellCheckerWASM.checkText(
          request.text || '', 
          request.profile_id || 'default'
        );
        // Normalize response to ensure arrays are never null
        const normalizedResult = normalizeResponse(result as SpellCheckResponse);
        return normalizedResult;
      }

      // Fall back to API
      if (MODE === 'api' || MODE === 'auto') {
        const response = await fetch(`${API_BASE}/check`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(request),
        });

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw new Error(errorData.error || `HTTP ${response.status}`);
        }

        const data: SpellCheckResponse = await response.json();
        return normalizeResponse(data);
      }

      throw new Error('No spell checking backend available');
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error';
      setError(errorMessage);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, [wasmReady]);

  const effectiveMode = getEffectiveMode(wasmReady);

  return { 
    checkText, 
    isLoading, 
    error,
    wasmReady,
    wasmLoading,
    wasmError,
    mode: effectiveMode,
    retryWASM: initializeWASM,
  };
}

/**
 * Hook for profile management
 * In WASM mode, all functions return errors immediately since profiles require a backend
 */
export function useProfiles() {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Profiles are only supported in API mode
  const profilesSupported = IS_API_MODE;

  const fetchProfiles = useCallback(async (): Promise<Profile[]> => {
    // In WASM mode, profiles are not supported
    if (IS_WASM_MODE) {
      console.log('Profiles not supported in WASM mode');
      return [];
    }

    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(`${API_BASE}/profiles`);

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      const data: Profile[] = await response.json();
      return data;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return [];
    } finally {
      setIsLoading(false);
    }
  }, []);

  const createProfile = useCallback(async (
    profile: Profile
  ): Promise<boolean> => {
    // In WASM mode, profiles are not supported
    if (IS_WASM_MODE) {
      setError('Creating profiles is disabled in headless (WASM) mode');
      return false;
    }

    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(`${API_BASE}/profiles`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          profile_id: profile.id,
          description: profile.description,
          terms: profile.terms,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `HTTP ${response.status}`);
      }

      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const updateProfile = useCallback(async (
    profile: Profile
  ): Promise<boolean> => {
    // In WASM mode, profiles are not supported
    if (IS_WASM_MODE) {
      setError('Updating profiles is disabled in headless (WASM) mode');
      return false;
    }

    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(`${API_BASE}/profiles/${profile.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          profile_id: profile.id,
          description: profile.description,
          terms: profile.terms,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `HTTP ${response.status}`);
      }

      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const deleteProfile = useCallback(async (
    profileId: string
  ): Promise<boolean> => {
    // In WASM mode, profiles are not supported
    if (IS_WASM_MODE) {
      setError('Deleting profiles is disabled in headless (WASM) mode');
      return false;
    }

    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(`${API_BASE}/profiles/${profileId}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const addTermToProfile = useCallback(async (
    profileId: string,
    term: string
  ): Promise<boolean> => {
    // In WASM mode, profiles are not supported
    if (IS_WASM_MODE) {
      setError('Adding terms to profiles is disabled in headless (WASM) mode');
      return false;
    }

    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(`${API_BASE}/profiles/${profileId}/terms`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ term }),
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const removeTermFromProfile = useCallback(async (
    profileId: string,
    term: string
  ): Promise<boolean> => {
    // In WASM mode, profiles are not supported
    if (IS_WASM_MODE) {
      setError('Removing terms from profiles is disabled in headless (WASM) mode');
      return false;
    }

    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(
        `${API_BASE}/profiles/${profileId}/terms/${encodeURIComponent(term)}`,
        { method: 'DELETE' }
      );

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  return {
    fetchProfiles,
    createProfile,
    updateProfile,
    deleteProfile,
    addTermToProfile,
    removeTermFromProfile,
    isLoading,
    error,
    profilesSupported,
  };
}
