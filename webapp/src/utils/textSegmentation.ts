import type { Issue, TextSegment, SpellCheckResponse } from '../types/api';

export function convertResponseToIssues(response: SpellCheckResponse): Issue[] {
  const issues: Issue[] = [];

  // Convert misspellings
  response.misspellings.forEach((m, index) => {
    issues.push({
      id: `misspelling-${index}`,
      type: 'misspelling',
      word: m.word,
      offset: m.offset,
      line: m.line,
      column: m.column,
      suggestions: m.suggestions?.map(s => s.word) || [],
    });
  });

  // Convert repeated words
  response.repeated_words.forEach((r, index) => {
    issues.push({
      id: `repeated-${index}`,
      type: 'repeated',
      word: r.word,
      offset: r.offset,
      line: r.line,
      column: r.column,
    });
  });

  // Convert capitalisation issues
  response.capitalisation_issues.forEach((c, index) => {
    issues.push({
      id: `capitalisation-${index}`,
      type: 'capitalisation',
      word: c.word,
      offset: c.offset,
      line: c.line,
      column: c.column,
    });
  });

  // Sort by offset to process in order
  return issues.sort((a, b) => a.offset - b.offset);
}

export function segmentText(text: string, issues: Issue[]): TextSegment[] {
  if (issues.length === 0) {
    return [{ type: 'normal', text }];
  }

  const segments: TextSegment[] = [];
  let currentOffset = 0;

  for (const issue of issues) {
    // Add normal text before this issue
    if (issue.offset > currentOffset) {
      segments.push({
        type: 'normal',
        text: text.slice(currentOffset, issue.offset),
      });
    }

    // Add the issue segment
    const issueText = text.slice(issue.offset, issue.offset + issue.word.length);
    segments.push({
      type: issue.type,
      text: issueText,
      issue,
    });

    currentOffset = issue.offset + issue.word.length;
  }

  // Add remaining text
  if (currentOffset < text.length) {
    segments.push({
      type: 'normal',
      text: text.slice(currentOffset),
    });
  }

  return segments;
}

export function applySuggestion(
  text: string,
  issue: Issue,
  suggestion: string
): string {
  return (
    text.slice(0, issue.offset) +
    suggestion +
    text.slice(issue.offset + issue.word.length)
  );
}

export function ignoreIssue(
  _text: string,
  issue: Issue,
  addIgnoreTerm: (term: string) => void
): void {
  // Just add to ignore list - don't modify text
  addIgnoreTerm(issue.word);
}
