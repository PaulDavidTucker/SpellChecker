import { useMemo } from 'react';
import { useAppStore } from '../../stores/appStore';
import { useSpellCheck } from '../../hooks/useSpellCheck';
import { convertResponseToIssues, applySuggestion } from '../../utils/textSegmentation';
import { Badge, Card } from '../ui';
import { 
  AlertCircle, 
  Repeat, 
  Type, 
  X,
  RotateCcw,
  EyeOff,
  CheckCircle2,
  Space
} from 'lucide-react';
import type { Issue } from '../../types/api';

export function IssuesPanel() {
  const {
    checkResult,
    text,
    setText,
    setCheckResult,
    pushUndo,
    addIgnoreTerm,
    popUndo,
    undoStack,
    showSidebar,
    setShowSidebar,
    currentProfileId,
    ignoreTerms,
  } = useAppStore();

  const { checkText } = useSpellCheck();

  const issues = useMemo(() => {
    if (!checkResult) return [];
    return convertResponseToIssues(checkResult);
  }, [checkResult]);

  const handleApplySuggestion = async (issue: Issue, suggestion: string) => {
    const originalText = text;
    const newText = applySuggestion(text, issue, suggestion);

    pushUndo({
      id: Date.now().toString(),
      timestamp: Date.now(),
      originalText,
      newText,
      issue,
      appliedSuggestion: suggestion,
    });

    setText(newText);
    
    // Re-check to show remaining issues with updated positions
    const result = await checkText({
      text: newText,
      profile_id: currentProfileId,
      ignore_terms: ignoreTerms,
      max_suggestions: 3,
    });
    
    if (result) {
      setCheckResult(result);
    }
  };

  const handleIgnore = async (issue: Issue) => {
    addIgnoreTerm(issue.word);
    // Re-check with new ignore term
    const result = await checkText({
      text,
      profile_id: currentProfileId,
      ignore_terms: [...ignoreTerms, issue.word.toLowerCase()],
      max_suggestions: 3,
    });
    
    if (result) {
      setCheckResult(result);
    }
  };

  const handleApplyAll = async () => {
    if (!text || issues.length === 0) return;

    const originalText = text;
    let newText = text;
    let appliedCount = 0;

    // Sort issues by offset in descending order (end to start)
    // This prevents offset shifts from affecting subsequent replacements
    const sortedIssues = [...issues]
      .filter(issue => issue.suggestions && issue.suggestions.length > 0)
      .sort((a, b) => b.offset - a.offset);

    for (const issue of sortedIssues) {
      const suggestion = issue.suggestions![0]; // Use first suggestion
      newText = applySuggestion(newText, issue, suggestion);
      appliedCount++;
    }

    if (appliedCount > 0) {
      pushUndo({
        id: Date.now().toString(),
        timestamp: Date.now(),
        originalText,
        newText,
        issue: issues[0], // Reference first issue for undo metadata
        appliedSuggestion: `Applied ${appliedCount} fixes`,
      });

      setText(newText);
      
      // Re-check to show any remaining issues
      const result = await checkText({
        text: newText,
        profile_id: currentProfileId,
        ignore_terms: ignoreTerms,
        max_suggestions: 3,
      });
      
      if (result) {
        setCheckResult(result);
      }
    }
  };

  const handleUndo = () => {
    const entry = popUndo();
    if (entry) {
      setText(entry.originalText);
      setCheckResult(null);
    }
  };

  const getIssueIcon = (type: Issue['type']) => {
    switch (type) {
      case 'misspelling':
        return <AlertCircle className="w-4 h-4 text-red-500" />;
      case 'repeated':
        return <Repeat className="w-4 h-4 text-yellow-500" />;
      case 'capitalisation':
        return <Type className="w-4 h-4 text-blue-500" />;
      case 'missing_space':
        return <Space className="w-4 h-4 text-purple-500" />;
    }
  };

  const getIssueBadge = (type: Issue['type']) => {
    switch (type) {
      case 'misspelling':
        return <Badge variant="error">Spelling</Badge>;
      case 'repeated':
        return <Badge variant="warning">Repeated</Badge>;
      case 'capitalisation':
        return <Badge variant="default">Capitalisation</Badge>;
      case 'missing_space':
        return <Badge variant="default" className="bg-purple-100 text-purple-800">Missing Space</Badge>;
    }
  };

  if (!showSidebar) {
    return (
      <button
        onClick={() => setShowSidebar(true)}
        className="fixed right-4 top-20 bg-white border border-gray-300 rounded-lg p-2 shadow-md hover:bg-gray-50"
      >
        Show Issues
      </button>
    );
  }

  return (
    <div className="w-80 bg-gray-50 border-l border-gray-200 flex flex-col h-full">
      <div className="p-5 border-b border-gray-200 bg-white flex items-center justify-between">
        <h3 className="font-semibold text-gray-900">Issues</h3>
        <div className="flex items-center gap-2">
          {undoStack.length > 0 && (
            <button
              onClick={handleUndo}
              className="p-2 text-gray-600 hover:text-blue-600 hover:bg-blue-50 rounded"
              title="Undo last change"
            >
              <RotateCcw className="w-4 h-4" />
            </button>
          )}
          <button
            onClick={() => setShowSidebar(false)}
            className="p-2 text-gray-400 hover:text-gray-600"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-auto p-5 space-y-4">
        {!checkResult ? (
          <div className="text-center text-gray-500 py-8">
            <p>Click "Check Spelling" to see issues</p>
          </div>
        ) : issues.length === 0 ? (
          <div className="text-center py-8">
            <div className="text-green-500 mb-2">✓</div>
            <p className="text-green-600 font-medium">No issues found!</p>
            <p className="text-sm text-gray-500 mt-1">
              {checkResult.token_count} tokens checked in {checkResult.elapsed_ms.toFixed(1)}ms
            </p>
          </div>
        ) : (
          <>
            <div className="flex items-center justify-between mb-4">
              <span className="text-sm text-gray-600">
                Found {issues.length} issue{issues.length !== 1 ? 's' : ''}
              </span>
              {issues.some(i => i.suggestions && i.suggestions.length > 0) && (
                <button
                  onClick={handleApplyAll}
                  className="text-xs flex items-center gap-1 px-3 py-1.5 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
                >
                  <CheckCircle2 className="w-3 h-3" />
                  Apply All
                </button>
              )}
            </div>

            {issues.map((issue) => (
              <Card key={issue.id} className="p-4">
                <div className="flex items-start gap-3">
                  {getIssueIcon(issue.type)}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="font-medium text-gray-900">"{issue.word}"</span>
                      {getIssueBadge(issue.type)}
                    </div>
                    <p className="text-xs text-gray-500 mb-3">
                      Line {issue.line}, Column {issue.column}
                    </p>

                    {issue.suggestions && issue.suggestions.length > 0 && (
                      <div className="space-y-1.5">
                        {issue.suggestions.slice(0, 3).map((suggestion, idx) => (
                          <button
                            key={idx}
                            onClick={() => handleApplySuggestion(issue, suggestion)}
                            className="block w-full text-left px-3 py-1.5 text-sm text-blue-600 hover:bg-blue-50 rounded"
                          >
                            → {suggestion}
                          </button>
                        ))}
                      </div>
                    )}

                    <div className="flex items-center gap-2 mt-3 pt-3 border-t border-gray-100">
                      <button
                        onClick={() => handleIgnore(issue)}
                        className="text-xs text-gray-500 hover:text-gray-700 flex items-center gap-1 px-2 py-1"
                      >
                        <EyeOff className="w-3 h-3" />
                        Ignore
                      </button>
                    </div>
                  </div>
                </div>
              </Card>
            ))}
          </>
        )}
      </div>

      {checkResult && (
        <div className="p-5 border-t border-gray-200 bg-white text-xs text-gray-500">
          <div className="flex justify-between">
            <span>Tokens: {checkResult.token_count}</span>
            <span>Checked: {checkResult.checked_count}</span>
          </div>
          <div className="mt-1">
            Time: {checkResult.elapsed_ms.toFixed(1)}ms
          </div>
        </div>
      )}
    </div>
  );
}
