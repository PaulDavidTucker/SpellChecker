import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { 
  SpellCheckResponse, 
  Profile,
  UndoEntry 
} from '../types/api';

interface AppState {
  // Editor state
  text: string;
  isChecking: boolean;
  checkResult: SpellCheckResponse | null;
  
  // Profile state
  profiles: Profile[];
  currentProfileId: string;
  
  // Session state
  ignoreTerms: string[];
  undoStack: UndoEntry[];
  
  // Settings
  autoCheck: boolean;
  debounceMs: number;
  showSidebar: boolean;
  
  // UI state
  selectedIssueId: string | null;
  isProfileModalOpen: boolean;
  
  // Actions
  setText: (text: string) => void;
  setCheckResult: (result: SpellCheckResponse | null) => void;
  setIsChecking: (isChecking: boolean) => void;
  
  setProfiles: (profiles: Profile[]) => void;
  setCurrentProfileId: (id: string) => void;
  addProfile: (profile: Profile) => void;
  removeProfile: (id: string) => void;
  
  addIgnoreTerm: (term: string) => void;
  removeIgnoreTerm: (term: string) => void;
  clearIgnoreTerms: () => void;
  
  pushUndo: (entry: UndoEntry) => void;
  popUndo: () => UndoEntry | undefined;
  clearUndo: () => void;
  
  setAutoCheck: (auto: boolean) => void;
  setDebounceMs: (ms: number) => void;
  setShowSidebar: (show: boolean) => void;
  
  setSelectedIssueId: (id: string | null) => void;
  setIsProfileModalOpen: (open: boolean) => void;
}

export const useAppStore = create<AppState>()(
  persist(
    (set, get) => ({
      // Initial state
      text: '',
      isChecking: false,
      checkResult: null,
      
      profiles: [],
      currentProfileId: 'default',
      
      ignoreTerms: [],
      undoStack: [],
      
      autoCheck: false,
      debounceMs: 1000,
      showSidebar: true,
      
      selectedIssueId: null,
      isProfileModalOpen: false,
      
      // Actions
      setText: (text) => set({ text }),
      setCheckResult: (result) => set({ checkResult: result }),
      setIsChecking: (isChecking) => set({ isChecking }),
      
      setProfiles: (profiles) => set({ profiles }),
      setCurrentProfileId: (id) => set({ currentProfileId: id }),
      addProfile: (profile) => set((state) => ({ 
        profiles: [...state.profiles, profile] 
      })),
      removeProfile: (id) => set((state) => ({ 
        profiles: state.profiles.filter(p => p.id !== id),
        currentProfileId: state.currentProfileId === id ? 'default' : state.currentProfileId
      })),
      
      addIgnoreTerm: (term) => set((state) => ({
        ignoreTerms: state.ignoreTerms.includes(term.toLowerCase()) 
          ? state.ignoreTerms 
          : [...state.ignoreTerms, term.toLowerCase()]
      })),
      removeIgnoreTerm: (term) => set((state) => ({
        ignoreTerms: state.ignoreTerms.filter(t => t !== term.toLowerCase())
      })),
      clearIgnoreTerms: () => set({ ignoreTerms: [] }),
      
      pushUndo: (entry) => set((state) => ({
        undoStack: [...state.undoStack, entry].slice(-50) // Keep last 50
      })),
      popUndo: () => {
        const state = get();
        const entry = state.undoStack[state.undoStack.length - 1];
        if (entry) {
          set({ undoStack: state.undoStack.slice(0, -1) });
        }
        return entry;
      },
      clearUndo: () => set({ undoStack: [] }),
      
      setAutoCheck: (auto) => set({ autoCheck: auto }),
      setDebounceMs: (ms) => set({ debounceMs: ms }),
      setShowSidebar: (show) => set({ showSidebar: show }),
      
      setSelectedIssueId: (id) => set({ selectedIssueId: id }),
      setIsProfileModalOpen: (open) => set({ isProfileModalOpen: open }),
    }),
    {
      name: 'spell-checker-storage',
      partialize: (state) => ({
        currentProfileId: state.currentProfileId,
        autoCheck: state.autoCheck,
        debounceMs: state.debounceMs,
        showSidebar: state.showSidebar,
      }),
    }
  )
);
