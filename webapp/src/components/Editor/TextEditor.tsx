import { useState, useCallback, useEffect, useRef } from "react";
import { useAppStore } from "../../stores/appStore";
import { useSpellCheck } from "../../hooks/useSpellCheck";
import {
  convertResponseToIssues,
  segmentText,
  applySuggestion,
} from "../../utils/textSegmentation";
import type { Issue, TextSegment } from "../../types/api";
import { Button, Badge } from "../ui";
import { Loader2, Check, RotateCcw, Zap } from "lucide-react";

export function TextEditor() {
  const {
    text,
    setText,
    setCheckResult,
    currentProfileId,
    ignoreTerms,
    checkResult,
    pushUndo,
    undoStack,
    popUndo,
    addIgnoreTerm,
    autoCheck,
    debounceMs,
    setAutoCheck,
  } = useAppStore();

  const { checkText, isLoading } = useSpellCheck();
  const [segments, setSegments] = useState<TextSegment[]>([
    { type: "normal", text: text || "" },
  ]);
  const [selectedIssue, setSelectedIssue] = useState<Issue | null>(null);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const handleCheck = useCallback(async () => {
    if (!text.trim()) return;

    const result = await checkText({
      text,
      profile_id: currentProfileId,
      ignore_terms: ignoreTerms,
      max_suggestions: 3,
    });

    if (result) {
      setCheckResult(result);
    }
  }, [text, currentProfileId, ignoreTerms, checkText, setCheckResult]);

  // Debounced auto-check effect
  useEffect(() => {
    if (!autoCheck || !text.trim()) {
      return;
    }

    // Clear existing timer
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }

    // Set new timer
    debounceTimerRef.current = setTimeout(() => {
      handleCheck();
    }, debounceMs);

    // Cleanup on unmount or when dependencies change
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, [text, autoCheck, debounceMs, handleCheck]);

  const handleTextChange = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      setText(e.target.value);
      // Clear check results when user edits
      if (checkResult) {
        setCheckResult(null);
      }
    },
    [setText, checkResult, setCheckResult],
  );

  const handleApplySuggestion = useCallback(
    async (issue: Issue, suggestion: string) => {
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
      setSelectedIssue(null);

      // Re-check to update highlights
      const result = await checkText({
        text: newText,
        profile_id: currentProfileId,
        ignore_terms: ignoreTerms,
        max_suggestions: 3,
      });

      if (result) {
        setCheckResult(result);
      }
    },
    [
      text,
      setText,
      setCheckResult,
      pushUndo,
      checkText,
      currentProfileId,
      ignoreTerms,
    ],
  );

  const handleIgnore = useCallback(
    (issue: Issue) => {
      addIgnoreTerm(issue.word);
      // Re-check without this issue
      handleCheck();
    },
    [addIgnoreTerm, handleCheck],
  );

  const handleUndo = useCallback(() => {
    const entry = popUndo();
    if (entry) {
      setText(entry.originalText);
      setCheckResult(null);
      setSelectedIssue(null);
    }
  }, [popUndo, setText, setCheckResult]);

  // Update segments when check result or text changes
  useEffect(() => {
    if (checkResult && text) {
      const issues = convertResponseToIssues(checkResult);
      setSegments(segmentText(text, issues));
    } else {
      setSegments([{ type: "normal", text: text || "" }]);
    }
  }, [checkResult, text]);

  const issueCount = checkResult
    ? (checkResult.misspellings?.length || 0) +
      (checkResult.repeated_words?.length || 0) +
      (checkResult.capitalisation_issues?.length || 0)
    : 0;

  return (
    <div className="flex flex-col h-full p-6">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold text-gray-900">Text Editor</h2>
        <div className="flex items-center gap-3">
          {undoStack.length > 0 && (
            <Button variant="ghost" size="sm" onClick={handleUndo}>
              <RotateCcw className="w-4 h-4 mr-1" />
              Undo
            </Button>
          )}
          <Button
            variant={autoCheck ? "secondary" : "ghost"}
            size="sm"
            onClick={() => setAutoCheck(!autoCheck)}
          >
            <Zap
              className={`w-4 h-4 mr-1 ${autoCheck ? "fill-yellow-400 text-yellow-600" : ""}`}
            />
            Auto {autoCheck && "ON"}
          </Button>
          <Button
            onClick={handleCheck}
            disabled={isLoading || !text.trim()}
            size="sm"
          >
            {isLoading ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                Checking...
              </>
            ) : (
              <>
                Check Spelling{" "}
                {issueCount > 0 && (
                  <Badge variant="error" className="ml-2">
                    {issueCount}
                  </Badge>
                )}
              </>
            )}
          </Button>
        </div>
      </div>

      {/* Text Input */}
      <div className="mb-6">
        <textarea
          value={text}
          onChange={handleTextChange}
          className="w-full p-4 border border-gray-300 rounded-lg resize-none focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent font-sans text-base leading-relaxed"
          style={{ minHeight: "200px" }}
          placeholder="Type or paste your text here..."
        />
      </div>

      {/* Highlighted View with Suggestions */}
      {checkResult && issueCount > 0 && (
        <div className="flex-1 border border-gray-200 rounded-lg overflow-hidden bg-white">
          <div className="bg-gray-50 px-6 py-3 border-b border-gray-200 flex items-center justify-between">
            <span className="text-sm font-medium text-gray-700">
              Issues Found
            </span>
            <div className="flex items-center gap-3 text-xs">
              <span className="flex items-center gap-1">
                <span className="w-2 h-2 bg-red-500 rounded-full"></span>{" "}
                Spelling
              </span>
              <span className="flex items-center gap-1">
                <span className="w-2 h-2 bg-yellow-500 rounded-full"></span>{" "}
                Repeated
              </span>
              <span className="flex items-center gap-1">
                <span className="w-2 h-2 bg-blue-500 rounded-full"></span>{" "}
                Capital
              </span>
              <span className="flex items-center gap-1">
                <span className="w-2 h-2 bg-purple-500 rounded-full"></span>{" "}
                Space
              </span>
            </div>
          </div>

          <div className="p-6" style={{ maxHeight: "300px" }}>
            <div className="whitespace-pre-wrap font-sans text-base leading-relaxed">
              {segments.map((segment, index) => {
                if (segment.type === "normal") {
                  return <span key={index}>{segment.text}</span>;
                }

                const isSelected = selectedIssue?.id === segment.issue?.id;
                let bgClass = "";
                let borderClass = "";

                switch (segment.type) {
                  case "misspelling":
                    bgClass = "bg-red-50";
                    borderClass = "border-b-2 border-red-400";
                    break;
                  case "repeated":
                    bgClass = "bg-yellow-50";
                    borderClass = "border-b-2 border-yellow-400";
                    break;
                  case "capitalisation":
                    bgClass = "bg-blue-50";
                    borderClass = "border-b-2 border-blue-400";
                    break;
                  case "missing_space":
                    bgClass = "bg-purple-50";
                    borderClass = "border-b-2 border-purple-400";
                    break;
                }

                return (
                  <span key={index} className="relative inline">
                    <span
                      className={`${bgClass} ${borderClass} cursor-pointer hover:opacity-80 rounded px-0.5`}
                      onClick={() =>
                        setSelectedIssue(
                          isSelected ? null : segment.issue || null,
                        )
                      }
                    >
                      {segment.text}
                    </span>

                    {/* Inline Suggestions Popup */}
                    {isSelected && segment.issue && (
                      <span className="absolute left-0 top-full mt-2 z-10 bg-white border border-gray-200 rounded-lg shadow-lg p-3 inline-block whitespace-nowrap">
                        <div className="text-xs text-gray-500 mb-2">
                          {segment.issue.word}:
                        </div>
                        <div className="flex gap-2 flex-wrap">
                          {segment.issue.suggestions &&
                          segment.issue.suggestions.length > 0 ? (
                            segment.issue.suggestions.map(
                              (suggestion, sIdx) => (
                                <button
                                  key={sIdx}
                                  className="px-3 py-1.5 text-sm bg-blue-100 hover:bg-blue-200 text-blue-800 rounded"
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    handleApplySuggestion(
                                      segment.issue!,
                                      suggestion,
                                    );
                                  }}
                                >
                                  {suggestion}
                                </button>
                              ),
                            )
                          ) : (
                            <span className="text-sm text-gray-400 italic">
                              No suggestions
                            </span>
                          )}
                          <button
                            className="px-3 py-1.5 text-sm bg-gray-100 hover:bg-gray-200 text-gray-600 rounded"
                            onClick={(e) => {
                              e.stopPropagation();
                              handleIgnore(segment.issue!);
                            }}
                          >
                            Ignore
                          </button>
                          <button
                            className="px-3 py-1.5 text-sm text-gray-400 hover:text-gray-600"
                            onClick={(e) => {
                              e.stopPropagation();
                              setSelectedIssue(null);
                            }}
                          >
                            ✕
                          </button>
                        </div>
                      </span>
                    )}
                  </span>
                );
              })}
            </div>
          </div>
        </div>
      )}

      {checkResult && issueCount === 0 && (
        <div className="flex-1 flex items-center justify-center p-8">
          <div className="text-center">
            <Check className="w-12 h-12 text-green-500 mx-auto mb-3" />
            <p className="text-green-600 font-medium text-lg">
              No issues found!
            </p>
            <p className="text-sm text-gray-500 mt-2">
              {checkResult.token_count} words checked in{" "}
              {checkResult.elapsed_ms.toFixed(1)}ms
            </p>
          </div>
        </div>
      )}
    </div>
  );
}
