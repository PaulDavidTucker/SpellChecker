// API Response Types matching Go backend

export interface Suggestion {
  word: string;
  edit_distance: number;
}

export interface Misspelling {
  word: string;
  offset: number;
  line: number;
  column: number;
  suggestions: Suggestion[];
}

export interface RepeatedWord {
  word: string;
  offset: number;
  line: number;
  column: number;
}

export interface CapitalisationIssue {
  word: string;
  offset: number;
  line: number;
  column: number;
}

export interface SpellCheckResponse {
  misspellings: Misspelling[];
  repeated_words: RepeatedWord[];
  capitalisation_issues: CapitalisationIssue[];
  token_count: number;
  checked_count: number;
  elapsed_ms: number;
}

export interface SpellCheckRequest {
  text?: string;
  text_base64?: string;
  profile_id?: string;
  ignore_terms?: string[];
  max_suggestions?: number;
}

export interface Profile {
  id: string;
  description?: string;
  terms: string[];
}

// Frontend-specific types

export type IssueType = 'misspelling' | 'repeated' | 'capitalisation';

export interface Issue {
  id: string;
  type: IssueType;
  word: string;
  offset: number;
  line: number;
  column: number;
  suggestions?: string[];
}

export interface TextSegment {
  type: 'normal' | 'misspelling' | 'repeated' | 'capitalisation';
  text: string;
  issue?: Issue;
}

export interface UndoEntry {
  id: string;
  timestamp: number;
  originalText: string;
  newText: string;
  issue: Issue;
  appliedSuggestion: string;
}
