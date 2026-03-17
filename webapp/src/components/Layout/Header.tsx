import { useAppStore } from '../../stores/appStore';
import { useSpellCheck, IS_WASM_MODE } from '../../hooks/useSpellCheck';
import { useServerReady } from '../../hooks/useServerReady';
import { ProfileManager } from '../ProfileManager/ProfileManager';
import { Button } from '../ui';
import { Settings, Undo2, AlertCircle, RefreshCw, Loader2 } from 'lucide-react';

// Tooltip message for disabled features in WASM mode
const WASM_DISABLED_MESSAGE = 'Disabled in headless mode';

export function Header() {
  const {
    undoStack,
    popUndo,
    setText,
    setCheckResult,
    setIsProfileModalOpen,
    currentProfileId,
  } = useAppStore();

  const { mode, wasmLoading, wasmError, retryWASM } = useSpellCheck();
  const { status } = useServerReady();
  const isWasmMode = mode === 'wasm' || IS_WASM_MODE;

  const handleUndo = () => {
    const entry = popUndo();
    if (entry) {
      setText(entry.originalText);
      setCheckResult(null);
    }
  };

  return (
    <header className="bg-white border-b border-gray-200 px-6 py-4">
      <div className="flex items-center justify-between max-w-7xl mx-auto">
        <div className="flex items-center gap-4">
          <h1 className="text-2xl font-bold text-gray-900">Spell Checker</h1>
          <span className="text-sm text-gray-500">
            Check your text for spelling mistakes
          </span>
        </div>

        <div className="flex items-center gap-4">
          {/* Server loading indicator for API mode */}
          {!isWasmMode && !status.ready && (
            <span className="text-xs text-blue-500 flex items-center gap-1 animate-pulse">
              <Loader2 className="w-3 h-3 animate-spin" />
              Loading dictionary...
            </span>
          )}

          {/* Profile Selector - Shows ProfileManager in API mode, disabled dropdown in WASM mode */}
          {isWasmMode ? (
            // WASM Mode: Simple dropdown with just default, disabled
            <div className="flex items-center gap-2">
              <select
                value={currentProfileId}
                disabled
                className="block w-32 px-3 py-2 bg-gray-100 border border-gray-300 rounded-md text-sm text-gray-600 cursor-not-allowed"
                title={WASM_DISABLED_MESSAGE}
              >
                <option value="default">default</option>
              </select>
              <span className="text-xs text-gray-400">(headless mode)</span>
            </div>
          ) : (
            // API Mode: Full profile management
            <ProfileManager />
          )}

          {/* Undo button */}
          {undoStack.length > 0 && (
            <Button
              variant="ghost"
              size="sm"
              onClick={handleUndo}
              className="flex items-center gap-1"
            >
              <Undo2 className="w-4 h-4" />
              Undo ({undoStack.length})
            </Button>
          )}

          {/* Profiles button - only show in API mode */}
          {!isWasmMode && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setIsProfileModalOpen(true)}
            >
              <Settings className="w-4 h-4 mr-1" />
              Profiles
            </Button>
          )}

          {/* WASM loading/error indicator */}
          {wasmLoading && (
            <span className="text-xs text-gray-400 animate-pulse">
              Initializing...
            </span>
          )}
          
          {wasmError && isWasmMode && (
            <div className="flex items-center gap-2">
              <span className="text-xs text-red-500 flex items-center gap-1" title={wasmError}>
                <AlertCircle className="w-3 h-3" />
                WASM Error
              </span>
              <Button
                variant="ghost"
                size="sm"
                onClick={retryWASM}
                className="text-xs p-1"
                title="Retry WASM initialization"
              >
                <RefreshCw className="w-3 h-3" />
              </Button>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}
