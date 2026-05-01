const BASE_URL = import.meta.env.VITE_API_URL || '';
const API = `${BASE_URL}/api`;

export async function fetchAPI<T>(path: string, opts?: RequestInit): Promise<T> {
  const res = await fetch(`${API}${path}`, {
    headers: { 'Content-Type': 'application/json', ...opts?.headers },
    ...opts,
  });
  if (res.status === 204) return undefined as T;
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`API error ${res.status}: ${text}`);
  }
  return res.json();
}

// Session API
export interface Session {
  id: string;
  projectId: string;
  directory: string;
  title: string;
  model?: string;
  permission?: string;
  compactionSummary?: string;
  createdAt: number;
  updatedAt: number;
}

export function listSessions(directory?: string): Promise<Session[]> {
  const dir = directory ? `?directory=${encodeURIComponent(directory)}` : '';
  return fetchAPI(`/session${dir}`);
}

export function createSession(directory?: string, model?: string): Promise<Session> {
  return fetchAPI('/session', {
    method: 'POST',
    body: JSON.stringify({ directory, model }),
  });
}

export function updateSession(id: string, updates: { title?: string; model?: string; permission?: string }): Promise<Session> {
  return fetchAPI(`/session/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(updates),
  });
}

export function getSession(id: string): Promise<Session> {
  return fetchAPI(`/session/${id}`);
}

export function deleteSession(id: string): Promise<void> {
  return fetchAPI(`/session/${id}`, { method: 'DELETE' });
}

// Message API
export interface TokenCounts {
  total?: number;
  input?: number;
  output?: number;
  reasoning?: number;
  cacheRead?: number;
  cacheWrite?: number;
}

export interface MessageInfo {
  id: string;
  sessionId: string;
  role: 'user' | 'assistant';
  agent?: string;
  parentId?: string;
  finish?: string;
  error?: string;
  cost?: number;
  tokens?: TokenCounts;
  createdAt: number;
}

export interface Part {
  id: string;
  messageId: string;
  sessionId: string;
  type: 'text' | 'tool' | 'reasoning';
  data: TextPartData | ToolPartData | ReasoningPartData;
  createdAt: number;
  updatedAt: number;
}

export interface TextPartData {
  text: string;
}

export interface ToolPartData {
  tool: string;
  callId: string;
  state: ToolState;
}

export interface ToolState {
  status: 'pending' | 'running' | 'completed' | 'error';
  input: any;
  output?: string;
  error?: string;
  title?: string;
  metadata?: any;
}

export interface ReasoningPartData {
  text: string;
}

export interface MessageWithParts {
  info: MessageInfo;
  parts: Part[];
}

export function getMessages(sessionId: string): Promise<MessageWithParts[]> {
  return fetchAPI(`/session/${sessionId}/message`);
}

export function sendPrompt(sessionId: string, content: string, model?: string): Promise<void> {
  return fetchAPI(`/session/${sessionId}/prompt`, {
    method: 'POST',
    body: JSON.stringify({ content, model }),
  });
}

export function replyPermission(sessionId: string, permissionId: string, response: string): Promise<void> {
  return fetchAPI(`/session/${sessionId}/permission/${permissionId}`, {
    method: 'POST',
    body: JSON.stringify({ response }),
  });
}

export function abortSession(sessionId: string): Promise<void> {
  return fetchAPI(`/session/${sessionId}/abort`, { method: 'POST' });
}

// Config API
export interface ConfigInfo {
  directory: string;
  port: number;
  memoryEnabled: boolean;
  memoryProvider: string;
}

export function getConfig(): Promise<ConfigInfo> {
  return fetchAPI('/config');
}

// Path API
export interface PathInfo {
  home: string;
  directory: string;
  state: string;
}

export function getPath(): Promise<PathInfo> {
  return fetchAPI('/path');
}

// VCS API
export interface VCSInfo {
  branch: string;
}

export function getVCS(): Promise<VCSInfo> {
  return fetchAPI('/vcs');
}

// Models API
export interface ModelInfo {
  id: string;
  name: string;
  providerId: string;
  default: boolean;
  enabled: boolean;
  isCustom: boolean;
}

export function getModels(): Promise<ModelInfo[]> {
  return fetchAPI('/models');
}

export interface ModelPreference {
  id: string;
  providerId: string;
  displayName: string;
  enabled: boolean;
  isCustom: boolean;
}

export function setModelPreference(pref: ModelPreference): Promise<ModelInfo[]> {
  return fetchAPI('/models/preference', {
    method: 'POST',
    body: JSON.stringify(pref),
  });
}

export function deleteModelPreference(id: string): Promise<void> {
  return fetchAPI(`/models/preference/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

// Theme API
export interface Theme {
  directory: string;
  primaryColor: string;
  accent: string;
  accentHover: string;
  accentSoft: string;
  accentRing: string;
  onPrimary: string;
  glow: string;
  tint: string;
}

export function getTheme(directory?: string): Promise<Theme> {
  const dir = directory ? `?directory=${encodeURIComponent(directory)}` : '';
  return fetchAPI(`/theme${dir}`);
}

export function setTheme(primaryColor: string, directory?: string): Promise<Theme> {
  return fetchAPI('/theme', {
    method: 'POST',
    body: JSON.stringify({ primaryColor, directory }),
  });
}

export function deleteTheme(directory: string): Promise<void> {
  return fetchAPI(`/theme/${encodeURIComponent(directory)}`, { method: 'DELETE' });
}