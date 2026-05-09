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
  memoryTokensSaved?: number;
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

// Memory config API
export interface MemoryConfig {
  enabled: boolean;
  embedProviderId: string;
  embedModel: string;
  embedApiKey: string;
  chatProviderId: string;
  chatModel: string;
  chatApiKey: string;
  updatedAt: number;
}

export function getMemoryConfig(): Promise<MemoryConfig> {
  return fetchAPI('/memory/config');
}

export function setMemoryConfig(cfg: Omit<MemoryConfig, 'updatedAt'>): Promise<MemoryConfig> {
  return fetchAPI('/memory/config', {
    method: 'POST',
    body: JSON.stringify(cfg),
  });
}

export function fetchMemoryModels(provider: string, type: 'embed' | 'chat', apiKey?: string): Promise<string[]> {
  const params = new URLSearchParams({ provider, type });
  if (apiKey) params.set('apiKey', apiKey);
  return fetchAPI(`/memory/models?${params}`);
}

// Provider config API
export interface ProviderConfig {
  providerId: string;
  apiKey: string;
  baseUrl: string;
  updatedAt: number;
}

export function getProviderConfigs(): Promise<ProviderConfig[]> {
  return fetchAPI('/providers/config');
}

export function setProviderConfig(id: string, cfg: Omit<ProviderConfig, 'providerId' | 'updatedAt'>): Promise<ProviderConfig> {
  return fetchAPI(`/providers/config/${id}`, {
    method: 'POST',
    body: JSON.stringify(cfg),
  });
}

// Pricing API — returns model ID → USD per 1 million input tokens
export function getProviderPricing(provider: string): Promise<Record<string, number>> {
  return fetchAPI(`/pricing?provider=${encodeURIComponent(provider)}`);
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

// Mode API
export interface ModeInfo {
  mode: string;
}

export function getMode(): Promise<ModeInfo> {
  return fetchAPI('/mode');
}

// Plan API
export interface Plan {
  id: string;
  sessionId: string;
  projectId: string;
  directory: string;
  title: string;
  status: 'open' | 'locked';
  model?: string;
  compactionSummary?: string;
  breakdownStatus?: '' | 'in_progress' | 'completed' | 'failed';
  breakdownWarnings?: string;
  allTasksCompleted?: boolean;
  createdAt: number;
  updatedAt: number;
}

export interface Task {
  id: string;
  planId: string;
  sessionId?: string;
  parentTaskId?: string;
  title: string;
  description: string;
  effort: 'S' | 'M' | 'L' | 'XL';
  complexity: 'low' | 'medium' | 'high';
  status: 'pending' | 'in_progress' | 'completed' | 'failed';
  dependencies: string[];
  branchName: string;
  worktreePath?: string;
  prUrl?: string;
  prNumber?: number;
  prError?: string;
  orderIndex: number;
  createdAt: number;
  updatedAt: number;
}

export function listPlans(directory?: string): Promise<Plan[]> {
  const dir = directory ? `?directory=${encodeURIComponent(directory)}` : '';
  return fetchAPI(`/plans${dir}`);
}

export function createPlan(directory?: string, title?: string, model?: string): Promise<Plan> {
  return fetchAPI('/plans', {
    method: 'POST',
    body: JSON.stringify({ directory, title, model }),
  });
}

export function getPlan(id: string): Promise<Plan> {
  return fetchAPI(`/plans/${id}`);
}

export function updatePlan(id: string, updates: { title?: string; model?: string }): Promise<Plan> {
  return fetchAPI(`/plans/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(updates),
  });
}

export function deletePlan(id: string): Promise<void> {
  return fetchAPI(`/plans/${id}`, { method: 'DELETE' });
}

export function lockPlan(id: string): Promise<Plan> {
  return fetchAPI(`/plans/${id}/lock`, { method: 'POST' });
}

export function sendPlanPrompt(id: string, content: string, model?: string): Promise<void> {
  return fetchAPI(`/plans/${id}/prompt`, {
    method: 'POST',
    body: JSON.stringify({ content, model }),
  });
}

export function getPlanMessages(id: string, before?: string): Promise<MessageWithParts[]> {
  const params = before ? `?before=${encodeURIComponent(before)}` : '';
  return fetchAPI(`/plans/${id}/message${params}`);
}

export function abortPlan(id: string): Promise<void> {
  return fetchAPI(`/plans/${id}/abort`, { method: 'POST' });
}

export async function downloadPlanExport(id: string): Promise<void> {
  const res = await fetch(`${API}/plans/${id}/export`);
  if (!res.ok) throw new Error(`Export failed: ${res.status}`);
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  const disp = res.headers.get('Content-Disposition') || '';
  const match = disp.match(/filename="(.+)"/);
  a.download = match?.[1] || 'plan.md';
  a.click();
  URL.revokeObjectURL(url);
}

// Task API
export function listTasks(planId: string): Promise<Task[]> {
  return fetchAPI(`/plans/${planId}/tasks`);
}

export function createTasks(planId: string, tasks: Array<{
  title: string;
  description?: string;
  effort?: string;
  complexity?: string;
  dependencies?: string[];
  orderIndex?: number;
}>): Promise<Task[]> {
  return fetchAPI(`/plans/${planId}/tasks`, {
    method: 'POST',
    body: JSON.stringify({ tasks }),
  });
}

export function getTask(id: string): Promise<Task> {
  return fetchAPI(`/tasks/${id}`);
}

export function updateTask(id: string, updates: {
  title?: string;
  description?: string;
  effort?: string;
  complexity?: string;
  status?: string;
  branchName?: string;
}): Promise<Task> {
  return fetchAPI(`/tasks/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(updates),
  });
}

export function startTask(id: string): Promise<Task> {
  return fetchAPI(`/tasks/${id}/start`, { method: 'POST' });
}

export function completeTask(id: string): Promise<Task> {
  return fetchAPI(`/tasks/${id}/complete`, { method: 'POST' });
}

export function failTask(id: string): Promise<Task> {
  return fetchAPI(`/tasks/${id}/fail`, { method: 'POST' });
}

export function retryTask(id: string): Promise<Task> {
  return fetchAPI(`/tasks/${id}/retry`, { method: 'POST' });
}

// Version API
export interface VersionInfo {
  version: string;
  commit: string;
  date: string;
  goVersion: string;
}

export interface UpdateInfo {
  latestVersion: string;
  updateAvailable: boolean;
  releaseUrl: string;
  publishedAt: string;
  releaseNotes: string;
  installCommand: string;
}

export interface VersionResponse {
  version: string;
  commit: string;
  date: string;
  goVersion: string;
  latestVersion: string;
  updateAvailable: boolean;
  releaseUrl: string;
  publishedAt: string;
  releaseNotes: string;
  installCommand: string;
}

export function getVersion(): Promise<VersionResponse> {
  return fetchAPI('/version');
}

export function checkForUpdate(): Promise<UpdateInfo> {
  return fetchAPI('/version/check', { method: 'POST' });
}