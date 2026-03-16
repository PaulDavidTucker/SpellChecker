import { useAppStore } from '../../stores/appStore';
import { ProfileManager } from '../ProfileManager/ProfileManager';
import { Button } from '../ui';
import { Settings, Undo2 } from 'lucide-react';

export function Header() {
  const {
    undoStack,
    popUndo,
    setText,
    setCheckResult,
    setIsProfileModalOpen,
  } = useAppStore();

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
          <ProfileManager />

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

          <Button
            variant="ghost"
            size="sm"
            onClick={() => setIsProfileModalOpen(true)}
          >
            <Settings className="w-4 h-4 mr-1" />
            Profiles
          </Button>
        </div>
      </div>
    </header>
  );
}
