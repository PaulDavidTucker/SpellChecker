import { useAppStore } from '../../stores/appStore';
import { Button } from '../ui';
import { X } from 'lucide-react';

export function SettingsPanel() {
  const {
    autoCheck,
    setAutoCheck,
    debounceMs,
    setDebounceMs,
    showSidebar,
    setShowSidebar,
    ignoreTerms,
    clearIgnoreTerms,
    removeIgnoreTerm,
  } = useAppStore();

  return (
    <div className="p-6 bg-gray-50 border-t border-gray-200">
      <div className="max-w-7xl mx-auto">
        <h3 className="text-sm font-medium text-gray-700 mb-4">Settings</h3>
        
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {/* Auto-check toggle */}
          <div className="flex items-center justify-between bg-white p-4 rounded-lg border border-gray-200">
            <div>
              <div className="font-medium text-sm">Auto-check</div>
              <div className="text-xs text-gray-500">
                Check as you type
              </div>
            </div>
            <button
              onClick={() => setAutoCheck(!autoCheck)}
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                autoCheck ? 'bg-blue-600' : 'bg-gray-200'
              }`}
            >
              <span
                className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                  autoCheck ? 'translate-x-6' : 'translate-x-1'
                }`}
              />
            </button>
          </div>

          {/* Debounce setting */}
          {autoCheck && (
            <div className="bg-white p-4 rounded-lg border border-gray-200">
              <div className="font-medium text-sm mb-3">Debounce delay</div>
              <div className="flex items-center gap-3">
                <input
                  type="range"
                  min="500"
                  max="3000"
                  step="100"
                  value={debounceMs}
                  onChange={(e) => setDebounceMs(Number(e.target.value))}
                  className="flex-1"
                />
                <span className="text-sm text-gray-600 w-16">{debounceMs}ms</span>
              </div>
            </div>
          )}

          {/* Sidebar toggle */}
          <div className="flex items-center justify-between bg-white p-4 rounded-lg border border-gray-200">
            <div>
              <div className="font-medium text-sm">Show sidebar</div>
              <div className="text-xs text-gray-500">
                Display issues panel
              </div>
            </div>
            <button
              onClick={() => setShowSidebar(!showSidebar)}
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                showSidebar ? 'bg-blue-600' : 'bg-gray-200'
              }`}
            >
              <span
                className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                  showSidebar ? 'translate-x-6' : 'translate-x-1'
                }`}
              />
            </button>
          </div>
        </div>

        {/* Ignore terms */}
        {ignoreTerms.length > 0 && (
          <div className="mt-6">
            <div className="flex items-center justify-between mb-3">
              <span className="text-sm font-medium text-gray-700">
                Ignored terms (this session)
              </span>
              <Button variant="ghost" size="sm" onClick={clearIgnoreTerms}>
                Clear all
              </Button>
            </div>
            <div className="flex flex-wrap gap-2">
              {ignoreTerms.map((term) => (
                <span
                  key={term}
                  className="inline-flex items-center px-3 py-1.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800"
                >
                  {term}
                  <button
                    onClick={() => removeIgnoreTerm(term)}
                    className="ml-2 text-gray-500 hover:text-red-500"
                  >
                    <X className="w-3 h-3" />
                  </button>
                </span>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
