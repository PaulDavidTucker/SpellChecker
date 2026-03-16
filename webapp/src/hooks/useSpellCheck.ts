import { useState, useCallback, useEffect } from 'react';
import type { SpellCheckRequest, SpellCheckResponse, Profile } from '../types/api';
import { spellCheckerWASM } from '../wasm/spellchecker';

// Mode detection - can be 'wasm', 'api', or 'auto'
const MODE = import.meta.env.VITE_SPELLCHECKER_MODE || 'auto';
const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export function useSpellCheck() {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [wasmReady, setWasmReady] = useState(false);
  const [wasmLoading, setWasmLoading] = useState(false);

  // Initialize WASM on mount if in WASM mode
  useEffect(() => {
    if (MODE === 'wasm' || MODE === 'auto') {
      initializeWASM();
    }
  }, []);

  const initializeWASM = async () => {
    if (wasmLoading || wasmReady) return;
    
    setWasmLoading(true);
    try {
      await spellCheckerWASM.initialize('./spellchecker.wasm');
      setWasmReady(true);
      console.log('WASM spell checker ready');
    } catch (err) {
      console.warn('WASM initialization failed, falling back to API:', err);
      setWasmReady(false);
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
        const result = spellCheckerWASM.checkText(
          request.text || '', 
          request.profile_id || 'default'
        );
        return result as SpellCheckResponse;
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
        return data;
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

  return { 
    checkText, 
    isLoading, 
    error,
    wasmReady,
    wasmLoading,
    mode: wasmReady ? 'wasm' : 'api'
  };
}

export function useProfiles() {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Profiles are always fetched from API (not supported in WASM yet)
  const fetchProfiles = useCallback(async (): Promise<Profile[]> => {
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
  };
}
