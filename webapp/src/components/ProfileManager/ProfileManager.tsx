import { useState, useEffect, useCallback } from 'react';
import { useAppStore } from '../../stores/appStore';
import { useProfiles, useSpellCheck, IS_WASM_MODE } from '../../hooks/useSpellCheck';
import { Button, Input, Card } from '../ui';
import { Plus, Trash2, Save, X, ChevronDown, ChevronUp } from 'lucide-react';
import type { Profile } from '../../types/api';

// Tooltip message for disabled features in WASM mode
const WASM_DISABLED_MESSAGE = 'Disabled in headless mode';

export function ProfileManager() {
  const {
    profiles,
    setProfiles,
    currentProfileId,
    setCurrentProfileId,
    isProfileModalOpen,
    setIsProfileModalOpen,
  } = useAppStore();

  const { mode } = useSpellCheck();
  const isWasmMode = mode === 'wasm' || IS_WASM_MODE;

  const {
    fetchProfiles,
    createProfile,
    deleteProfile,
    addTermToProfile,
    removeTermFromProfile,
    isLoading,
  } = useProfiles();

  const [expandedProfile, setExpandedProfile] = useState<string | null>(null);
  const [newProfileId, setNewProfileId] = useState('');
  const [newTerm, setNewTerm] = useState('');
  const [activeProfileForTerm, setActiveProfileForTerm] = useState<string | null>(null);

  // Load profiles on mount (only in API mode)
  useEffect(() => {
    if (isWasmMode) return; // Skip API calls in WASM mode
    
    const loadProfiles = async () => {
      const loaded = await fetchProfiles();
      setProfiles(loaded);
    };
    loadProfiles();
  }, [fetchProfiles, setProfiles, isWasmMode]);

  const handleCreateProfile = useCallback(async () => {
    if (!newProfileId.trim()) return;
    if (isWasmMode) return; // Should not reach here due to disabled button, but safety check

    const profile: Profile = {
      id: newProfileId.trim(),
      terms: [],
    };

    const success = await createProfile(profile);
    if (success) {
      setProfiles([...profiles, profile]);
      setNewProfileId('');
      setCurrentProfileId(profile.id);
    }
  }, [newProfileId, profiles, createProfile, setProfiles, setCurrentProfileId, isWasmMode]);

  const handleDeleteProfile = useCallback(async (profileId: string) => {
    if (isWasmMode) return; // Should not reach here due to disabled button, but safety check

    const success = await deleteProfile(profileId);
    if (success) {
      setProfiles(profiles.filter(p => p.id !== profileId));
      if (currentProfileId === profileId) {
        setCurrentProfileId('default');
      }
    }
  }, [profiles, currentProfileId, deleteProfile, setProfiles, setCurrentProfileId, isWasmMode]);

  const handleAddTerm = useCallback(async (profileId: string) => {
    if (!newTerm.trim()) return;
    if (isWasmMode) return; // Should not reach here due to disabled button, but safety check

    const success = await addTermToProfile(profileId, newTerm.trim());
    if (success) {
      setProfiles(profiles.map(p => 
        p.id === profileId 
          ? { ...p, terms: [...p.terms, newTerm.trim()] }
          : p
      ));
      setNewTerm('');
    }
  }, [newTerm, profiles, addTermToProfile, setProfiles, isWasmMode]);

  const handleRemoveTerm = useCallback(async (profileId: string, term: string) => {
    if (isWasmMode) return; // Should not reach here due to disabled button, but safety check

    const success = await removeTermFromProfile(profileId, term);
    if (success) {
      setProfiles(profiles.map(p => 
        p.id === profileId 
          ? { ...p, terms: p.terms.filter(t => t !== term) }
          : p
      ));
    }
  }, [profiles, removeTermFromProfile, setProfiles, isWasmMode]);

  if (!isProfileModalOpen) {
    return (
      <div className="flex items-center gap-2">
        <select
          value={currentProfileId}
          onChange={(e) => setCurrentProfileId(e.target.value)}
          disabled={isWasmMode}
          className="block w-48 px-3 py-2 bg-white border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 text-sm disabled:bg-gray-100 disabled:text-gray-500 disabled:cursor-not-allowed"
          title={isWasmMode ? WASM_DISABLED_MESSAGE : undefined}
        >
          {isWasmMode ? (
            <option value="default">default</option>
          ) : (
            profiles.map((profile) => (
              <option key={profile.id} value={profile.id}>
                {profile.id}
              </option>
            ))
          )}
        </select>
        <Button 
          variant="ghost" 
          size="sm"
          onClick={() => !isWasmMode && setIsProfileModalOpen(true)}
          disabled={isWasmMode}
          title={isWasmMode ? WASM_DISABLED_MESSAGE : "Manage profiles"}
        >
          <Plus className="w-4 h-4" />
        </Button>
      </div>
    );
  }

  // Don't render modal in WASM mode (button is disabled, so this shouldn't happen)
  if (isWasmMode) {
    return null;
  }

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-2xl max-h-[85vh] flex flex-col">
        <div className="p-6 border-b border-gray-200 flex items-center justify-between">
          <h2 className="text-xl font-semibold">Profile Manager</h2>
          <button
            onClick={() => setIsProfileModalOpen(false)}
            className="text-gray-400 hover:text-gray-600 p-1"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-6 overflow-auto flex-1 space-y-6">
          {/* Create new profile */}
          <div>
            <h3 className="text-sm font-medium text-gray-700 mb-3">Create New Profile</h3>
            <div className="flex gap-3">
              <Input
                value={newProfileId}
                onChange={setNewProfileId}
                placeholder="Profile ID (e.g., my-custom-profile)"
                className="flex-1"
                disabled={isWasmMode}
              />
              <Button
                onClick={handleCreateProfile}
                disabled={isLoading || !newProfileId.trim() || isWasmMode}
                title={isWasmMode ? WASM_DISABLED_MESSAGE : "Create new profile"}
              >
                <Plus className="w-4 h-4 mr-1" />
                Create
              </Button>
            </div>
          </div>

          {/* Profile list */}
          <div className="space-y-4">
            <h3 className="text-sm font-medium text-gray-700">Existing Profiles</h3>
            
            {profiles.length === 0 ? (
              <p className="text-gray-500 text-sm">No profiles found</p>
            ) : (
              profiles.map((profile) => (
                <Card key={profile.id} className="p-5">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <button
                        onClick={() => setExpandedProfile(
                          expandedProfile === profile.id ? null : profile.id
                        )}
                        className="text-gray-500 hover:text-gray-700 p-1"
                        disabled={isWasmMode}
                        title={isWasmMode ? WASM_DISABLED_MESSAGE : undefined}
                      >
                        {expandedProfile === profile.id ? (
                          <ChevronUp className="w-4 h-4" />
                        ) : (
                          <ChevronDown className="w-4 h-4" />
                        )}
                      </button>
                      <span className="font-medium">{profile.id}</span>
                      <span className="text-sm text-gray-500">
                        ({profile.terms.length} terms)
                      </span>
                      {currentProfileId === profile.id && (
                        <span className="text-xs bg-blue-100 text-blue-800 px-2 py-0.5 rounded">
                          Active
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-2">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setCurrentProfileId(profile.id)}
                        disabled={currentProfileId === profile.id}
                      >
                        <Save className="w-4 h-4 mr-1" />
                        Use
                      </Button>
                      {profile.id !== 'default' && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleDeleteProfile(profile.id)}
                          disabled={isLoading || isWasmMode}
                          title={isWasmMode ? WASM_DISABLED_MESSAGE : "Delete profile"}
                        >
                          <Trash2 className="w-4 h-4 text-red-500" />
                        </Button>
                      )}
                    </div>
                  </div>

                  {/* Expanded: Show terms */}
                  {expandedProfile === profile.id && (
                    <div className="mt-4 pl-8 border-l-2 border-gray-200">
                      <div className="flex gap-3 mb-4">
                        <Input
                          value={activeProfileForTerm === profile.id ? newTerm : ''}
                          onChange={(val) => {
                            setNewTerm(val);
                            setActiveProfileForTerm(profile.id);
                          }}
                          placeholder="Add a term..."
                          className="flex-1"
                          disabled={isWasmMode}
                        />
                        <Button
                          size="sm"
                          onClick={() => handleAddTerm(profile.id)}
                          disabled={isLoading || !newTerm.trim() || isWasmMode}
                          title={isWasmMode ? WASM_DISABLED_MESSAGE : "Add term to profile"}
                        >
                          <Plus className="w-4 h-4" />
                        </Button>
                      </div>

                      <div className="flex flex-wrap gap-2">
                        {profile.terms.map((term) => (
                          <span
                            key={term}
                            className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800"
                          >
                            {term}
                            <button
                              onClick={() => handleRemoveTerm(profile.id, term)}
                              disabled={isWasmMode}
                              className="ml-1 text-gray-500 hover:text-red-500 disabled:opacity-50 disabled:cursor-not-allowed"
                              title={isWasmMode ? WASM_DISABLED_MESSAGE : "Remove term"}
                            >
                              <X className="w-3 h-3" />
                            </button>
                          </span>
                        ))}
                        {profile.terms.length === 0 && (
                          <span className="text-sm text-gray-400 italic">
                            No custom terms
                          </span>
                        )}
                      </div>
                    </div>
                  )}
                </Card>
              ))
            )}
          </div>
        </div>

        <div className="p-6 border-t border-gray-200 bg-gray-50 flex justify-end">
          <Button onClick={() => setIsProfileModalOpen(false)}>
            Close
          </Button>
        </div>
      </div>
    </div>
  );
}
