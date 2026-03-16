import { Header } from './Header';
import { SettingsPanel } from './SettingsPanel';
import { TextEditor } from '../Editor/TextEditor';
import { IssuesPanel } from '../IssuesPanel/IssuesSidebar';
import { ProfileManager } from '../ProfileManager/ProfileManager';
import { useAppStore } from '../../stores/appStore';
import { useSpellCheck } from '../../hooks/useSpellCheck';

export function MainLayout() {
  const { showSidebar, isProfileModalOpen } = useAppStore();
  const { mode } = useSpellCheck();

  return (
    <div className="flex flex-col h-screen bg-gray-50">
      <Header />
      
      <div className="flex flex-1 overflow-hidden">
        <div className={`flex flex-col ${showSidebar ? 'flex-1' : 'w-full'}`}>
          <div className="flex-1 p-6 overflow-hidden">
            <div className="h-full max-w-7xl mx-auto bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
              <TextEditor />
            </div>
          </div>
          
          <SettingsPanel />
        </div>

        {showSidebar && <IssuesPanel />}
      </div>

      {/* Profile modal overlay - only show in API mode */}
      {mode === 'api' && isProfileModalOpen && <ProfileManager />}
    </div>
  );
}
