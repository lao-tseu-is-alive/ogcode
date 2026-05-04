import { useParams, useNavigate } from '@solidjs/router';
import { createEffect, on, For, Show, createMemo, createSignal } from 'solid-js';
import { usePlan } from '../context/plan';
import Breadcrumb from '../components/breadcrumb';
import MarkdownContent from '../components/markdown-content';
import type { Task } from '../api/client';

// ─── constants ────────────────────────────────────────────────────────────────

const STATUS_META: Record<string, { dot: string; badge: string; col: string; label: string }> = {
  pending:     { dot: 'bg-zinc-500',              badge: 'text-zinc-400 bg-zinc-500/10 border-zinc-500/20',          col: 'border-zinc-700/40',    label: 'Pending' },
  in_progress: { dot: 'bg-blue-400 animate-pulse', badge: 'text-blue-400 bg-blue-500/10 border-blue-500/20',          col: 'border-blue-500/40',    label: 'In Progress' },
  completed:   { dot: 'bg-emerald-400',            badge: 'text-emerald-400 bg-emerald-500/10 border-emerald-500/20', col: 'border-emerald-500/40', label: 'Completed' },
  failed:      { dot: 'bg-red-400',                badge: 'text-red-400 bg-red-500/10 border-red-500/20',             col: 'border-red-500/40',     label: 'Failed' },
};

const EFFORT_META: Record<string, { cls: string; label: string }> = {
  S:  { cls: 'text-emerald-400 bg-emerald-500/10 border-emerald-500/25', label: 'Small' },
  M:  { cls: 'text-blue-400 bg-blue-500/10 border-blue-500/25',          label: 'Medium' },
  L:  { cls: 'text-amber-400 bg-amber-500/10 border-amber-500/25',       label: 'Large' },
  XL: { cls: 'text-red-400 bg-red-500/10 border-red-500/25',             label: 'XL' },
};

const COMPLEXITY_CLS: Record<string, string> = {
  low:    'text-emerald-400',
  medium: 'text-amber-400',
  high:   'text-red-400',
};

const COLUMNS: Array<{ status: Task['status']; label: string }> = [
  { status: 'pending',     label: 'Pending' },
  { status: 'in_progress', label: 'In Progress' },
  { status: 'completed',   label: 'Completed' },
  { status: 'failed',      label: 'Failed' },
];

// ─── task detail drawer ───────────────────────────────────────────────────────

function TaskDrawer(props: { task: Task | null; onClose: () => void }) {
  const plan = usePlan();
  const navigate = useNavigate();
  const [starting, setStarting] = createSignal(false);
  const [completing, setCompleting] = createSignal(false);
  const [failing, setFailing] = createSignal(false);
  const [retrying, setRetrying] = createSignal(false);

  const t = () => props.task;

  const depTasks = () => {
    if (!t()) return [];
    return t()!.dependencies.map((id) => plan.tasks().find((x) => x.id === id)).filter(Boolean) as Task[];
  };

  const completedIds = () => new Set(plan.tasks().filter((x) => x.status === 'completed').map((x) => x.id));
  const canStart = () => plan.activePlan()?.status === 'locked' && t()?.status === 'pending' && (t()?.dependencies ?? []).every((d) => completedIds().has(d));
  const blockedBy = () => depTasks().filter((d) => d.status !== 'completed');

  const handleStart = async () => {
    if (!t()) return;
    setStarting(true);
    try {
      await plan.startTaskById(t()!.id);
    } finally {
      setStarting(false);
    }
  };

  const handleComplete = async () => {
    if (!t()) return;
    setCompleting(true);
    try {
      await plan.completeTaskById(t()!.id);
    } finally {
      setCompleting(false);
    }
  };

  const handleFail = async () => {
    if (!t()) return;
    setFailing(true);
    try {
      await plan.failTaskById(t()!.id);
    } finally {
      setFailing(false);
    }
  };

  const handleRetry = async () => {
    if (!t()) return;
    setRetrying(true);
    try {
      await plan.retryTaskById(t()!.id);
    } finally {
      setRetrying(false);
    }
  };

  return (
    <>
      {/* Backdrop */}
      <Show when={!!t()}>
        <div
          class="fixed inset-0 z-40 bg-black/40 backdrop-blur-[1px]"
          onClick={props.onClose}
        />
      </Show>

      {/* Drawer */}
      <div
        class={`fixed top-0 right-0 h-full z-50 w-[420px] max-w-[95vw] flex flex-col
                border-l border-[color:var(--border-subtle)] shadow-2xl
                transition-transform duration-250 ease-out
                ${t() ? 'translate-x-0' : 'translate-x-full'}`}
        style={{ background: 'var(--bg-surface)' }}
      >
        <Show when={!!t()}>
          {/* Drawer header */}
          <div class="h-12 shrink-0 border-b border-[color:var(--border-subtle)] flex items-center px-4 gap-3">
            <div class="flex items-center gap-2 flex-1 min-w-0">
              <span class={`w-2.5 h-2.5 rounded-full shrink-0 ${STATUS_META[t()!.status]?.dot || 'bg-zinc-500'}`} />
              <span class="text-[13px] font-semibold text-zinc-100 truncate">{t()!.title}</span>
            </div>
            <button
              type="button"
              onClick={props.onClose}
              class="w-7 h-7 rounded-md text-zinc-500 hover:text-zinc-200 hover:bg-[color:var(--bg-hover)] flex items-center justify-center transition shrink-0"
            >
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          {/* Drawer body */}
          <div class="flex-1 overflow-y-auto p-4 space-y-5">

            {/* Status + badges */}
            <div class="flex items-center gap-2 flex-wrap">
              <span class={`text-[11px] font-semibold px-2 py-1 rounded-lg border ${STATUS_META[t()!.status]?.badge}`}>
                {STATUS_META[t()!.status]?.label}
              </span>
              <span class={`text-[11px] font-bold px-2 py-1 rounded-lg border ${EFFORT_META[t()!.effort]?.cls || EFFORT_META.M.cls}`}>
                {EFFORT_META[t()!.effort]?.label || t()!.effort} effort
              </span>
              <span class={`text-[11px] font-medium capitalize ${COMPLEXITY_CLS[t()!.complexity] || 'text-zinc-400'}`}>
                {t()!.complexity} complexity
              </span>
            </div>

            {/* Description */}
            <Show when={t()!.description}>
              <div>
                <p class="text-[11px] font-semibold text-zinc-500 uppercase tracking-wider mb-1.5">Description</p>
                <MarkdownContent text={t()!.description} />
              </div>
            </Show>

            {/* Branch */}
            <Show when={t()!.branchName}>
              <div>
                <p class="text-[11px] font-semibold text-zinc-500 uppercase tracking-wider mb-1.5">Branch</p>
                <code class="text-[12px] text-zinc-300 bg-[color:var(--bg-elevated)] px-2.5 py-1.5 rounded-lg border border-[color:var(--border-subtle)] font-mono block w-fit">
                  {t()!.branchName}
                </code>
              </div>
            </Show>

            {/* PR link */}
            <Show when={t()!.prUrl}>
              <div>
                <p class="text-[11px] font-semibold text-zinc-500 uppercase tracking-wider mb-1.5">Pull Request</p>
                <a
                  href={t()!.prUrl!}
                  target="_blank"
                  rel="noopener noreferrer"
                  class="inline-flex items-center gap-1.5 text-[12px] text-blue-400 hover:text-blue-300 transition-colors"
                >
                  <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                  </svg>
                  PR #{t()!.prNumber}
                </a>
              </div>
            </Show>

            {/* Dependencies */}
            <Show when={depTasks().length > 0}>
              <div>
                <p class="text-[11px] font-semibold text-zinc-500 uppercase tracking-wider mb-2">
                  Dependencies ({depTasks().length})
                </p>
                <div class="space-y-1.5">
                  <For each={depTasks()}>
                    {(dep) => (
                      <div
                        onClick={() => { /* select dep task */ }}
                        class="flex items-center gap-2.5 px-3 py-2 rounded-lg border border-[color:var(--border-subtle)] bg-[color:var(--bg-elevated)] cursor-pointer hover:border-[color:var(--border-default)] transition-colors"
                      >
                        <span class={`w-2 h-2 rounded-full shrink-0 ${STATUS_META[dep.status]?.dot || 'bg-zinc-500'}`} />
                        <span class="text-[12px] text-zinc-300 flex-1 truncate">{dep.title}</span>
                        <span class={`text-[10px] font-medium px-1.5 py-0.5 rounded border ${STATUS_META[dep.status]?.badge}`}>
                          {STATUS_META[dep.status]?.label}
                        </span>
                      </div>
                    )}
                  </For>
                </div>
              </div>
            </Show>

            {/* Blocked notice */}
            <Show when={t()!.status === 'pending' && blockedBy().length > 0}>
              <div class="flex items-start gap-2.5 px-3 py-2.5 rounded-lg bg-amber-500/8 border border-amber-500/20">
                <svg class="w-4 h-4 text-amber-400 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01M12 3a9 9 0 100 18A9 9 0 0012 3z" />
                </svg>
                <p class="text-[12px] text-amber-300">
                  Blocked by {blockedBy().length} incomplete {blockedBy().length === 1 ? 'dependency' : 'dependencies'}
                </p>
              </div>
            </Show>
          </div>

          {/* Drawer footer actions */}
          <div class="shrink-0 border-t border-[color:var(--border-subtle)] p-3 space-y-2">
            {/* Primary action row */}
            <div class="flex gap-2">
              <Show when={t()!.status === 'in_progress'}>
                <button
                  type="button"
                  onClick={() => navigate(`/task/${t()!.id}`)}
                  class="flex-1 h-9 rounded-lg bg-blue-500/15 hover:bg-blue-500/25 border border-blue-500/30 text-blue-400 text-[13px] font-medium flex items-center justify-center gap-2 transition"
                >
                  <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
                  </svg>
                  Open Agent Session
                </button>
              </Show>
              <Show when={canStart()}>
                <button
                  type="button"
                  onClick={handleStart}
                  disabled={starting()}
                  class="flex-1 h-9 rounded-lg bg-[color:var(--accent)] hover:bg-[color:var(--accent-hover)] text-[color:var(--on-primary)] text-[13px] font-semibold flex items-center justify-center gap-2 transition shadow-sm disabled:opacity-60"
                >
                  <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M5 3l14 9-14 9V3z" />
                  </svg>
                  {starting() ? 'Starting…' : 'Start Task'}
                </button>
              </Show>
              <Show when={t()!.status === 'failed'}>
                <button
                  type="button"
                  onClick={handleRetry}
                  disabled={retrying()}
                  class="flex-1 h-9 rounded-lg bg-red-500/15 hover:bg-red-500/25 border border-red-500/30 text-red-400 text-[13px] font-semibold flex items-center justify-center gap-2 transition disabled:opacity-60"
                >
                  <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                  </svg>
                  {retrying() ? 'Retrying…' : 'Retry Task'}
                </button>
              </Show>
              <Show when={t()!.status === 'completed' || (t()!.status === 'pending' && !canStart() && blockedBy().length === 0)}>
                <div class="flex-1 h-9 rounded-lg border border-[color:var(--border-subtle)] flex items-center justify-center">
                  <span class="text-[12px] text-zinc-600">No actions available</span>
                </div>
              </Show>
            </div>

            {/* Manual override row for in-progress tasks */}
            <Show when={t()!.status === 'in_progress'}>
              <div class="flex gap-2">
                <button
                  type="button"
                  onClick={handleComplete}
                  disabled={completing()}
                  class="flex-1 h-8 rounded-lg bg-emerald-500/12 hover:bg-emerald-500/20 border border-emerald-500/25 text-emerald-400 text-[12px] font-medium flex items-center justify-center gap-1.5 transition disabled:opacity-50"
                >
                  <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                  </svg>
                  {completing() ? 'Completing…' : 'Mark Complete'}
                </button>
                <button
                  type="button"
                  onClick={handleFail}
                  disabled={failing()}
                  class="flex-1 h-8 rounded-lg bg-red-500/12 hover:bg-red-500/20 border border-red-500/25 text-red-400 text-[12px] font-medium flex items-center justify-center gap-1.5 transition disabled:opacity-50"
                >
                  <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                  {failing() ? 'Failing…' : 'Mark Failed'}
                </button>
              </div>
            </Show>
          </div>
        </Show>
      </div>
    </>
  );
}

// ─── kanban card ──────────────────────────────────────────────────────────────

function KanbanCard(props: { task: Task; onSelect: (t: Task) => void }) {
  const plan = usePlan();
  const [starting, setStarting] = createSignal(false);

  const t = () => props.task;
  const completedIds = () => new Set(plan.tasks().filter((x) => x.status === 'completed').map((x) => x.id));
  const canStart = () => plan.activePlan()?.status === 'locked' && t().status === 'pending' && t().dependencies.every((d) => completedIds().has(d));
  const isBlocked = () => t().status === 'pending' && !canStart();

  const depTasks = () =>
    t().dependencies.map((id) => plan.tasks().find((x) => x.id === id)).filter(Boolean) as Task[];

  const handleStart = async (e: MouseEvent) => {
    e.stopPropagation();
    setStarting(true);
    try { await plan.startTaskById(t().id); } finally { setStarting(false); }
  };

  const sm = STATUS_META[t().status] || STATUS_META.pending;

  return (
    <div
      onClick={() => props.onSelect(t())}
      class={`group rounded-xl border bg-[color:var(--bg-surface)] p-3.5 cursor-pointer
              transition-all duration-150 hover:shadow-lg hover:shadow-black/25
              hover:border-[color:var(--border-strong)] hover:-translate-y-px
              ${isBlocked() ? 'opacity-60' : ''}
              ${sm.col}`}
    >
      {/* Status dot + title */}
      <div class="flex items-start gap-2.5 mb-2.5">
        <span class={`mt-[5px] w-2 h-2 rounded-full shrink-0 ${sm.dot}`} />
        <p class="text-[13px] font-semibold text-zinc-100 leading-snug flex-1">{t().title}</p>
      </div>

      {/* Description */}
      <Show when={t().description}>
        <MarkdownContent text={t().description} class="prose-chat-preview line-clamp-2 mb-2.5 text-zinc-500" />
      </Show>

      {/* Dependency pills */}
      <Show when={depTasks().length > 0}>
        <div class="flex flex-wrap gap-1 mb-2.5">
          <For each={depTasks().slice(0, 3)}>
            {(dep) => (
              <span class={`inline-flex items-center gap-1 text-[9px] px-1.5 py-0.5 rounded-full border ${STATUS_META[dep.status]?.badge}`}>
                <span class={`w-1.5 h-1.5 rounded-full ${STATUS_META[dep.status]?.dot}`} />
                {dep.title.length > 20 ? dep.title.slice(0, 20) + '…' : dep.title}
              </span>
            )}
          </For>
          <Show when={depTasks().length > 3}>
            <span class="text-[9px] text-zinc-600 px-1 py-0.5">+{depTasks().length - 3} more</span>
          </Show>
        </div>
      </Show>

      {/* Footer */}
      <div class="flex items-center gap-1.5">
        <span class={`text-[9px] font-bold px-1.5 py-0.5 rounded border ${EFFORT_META[t().effort]?.cls || EFFORT_META.M.cls}`}>
          {t().effort}
        </span>
        <span class={`text-[10px] capitalize ${COMPLEXITY_CLS[t().complexity] || 'text-zinc-500'}`}>
          {t().complexity}
        </span>

        <div class="flex-1" />

        <Show when={t().prUrl}>
          <a
            href={t().prUrl!}
            target="_blank"
            rel="noopener noreferrer"
            onClick={(e) => e.stopPropagation()}
            class="text-[10px] text-blue-400 hover:text-blue-300 transition-colors"
          >
            PR #{t().prNumber}
          </a>
        </Show>

        <Show when={canStart()}>
          <button
            type="button"
            onClick={handleStart}
            disabled={starting()}
            class="text-[10px] px-2 py-0.5 rounded-md border border-[color:var(--accent)]/50
                   bg-[color:var(--accent-soft)] text-[color:var(--accent)] hover:bg-[color:var(--accent)]/25
                   transition-colors disabled:opacity-50 font-medium"
          >
            {starting() ? '…' : 'Start'}
          </button>
        </Show>

        <Show when={isBlocked()}>
          <span class="text-[10px] text-amber-500/70 font-medium">Blocked</span>
        </Show>

        {/* Expand hint */}
        <svg
          class="w-3 h-3 text-zinc-700 group-hover:text-zinc-500 transition-colors shrink-0"
          fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"
        >
          <path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
        </svg>
      </div>
    </div>
  );
}

// ─── page ─────────────────────────────────────────────────────────────────────

export default function PlanTasksPage() {
  const plan = usePlan();
  const params = useParams();
  const navigate = useNavigate();
  const [startingAll, setStartingAll] = createSignal(false);
  const [startError, setStartError] = createSignal('');
  const [selectedTask, setSelectedTask] = createSignal<Task | null>(null);

  createEffect(on(() => params.id, (id) => {
    if (id) plan.selectPlan(id);
  }));

  // Keep drawer in sync with live task data
  createEffect(() => {
    const sel = selectedTask();
    if (!sel) return;
    const fresh = plan.tasks().find((t) => t.id === sel.id);
    if (fresh) setSelectedTask(fresh);
  });

  const breadcrumbs = () => {
    const p = plan.activePlan();
    return [
      { label: 'Plans', href: '/plan' },
      { label: p?.title || 'Plan', href: `/plan/${p?.id}` },
      { label: 'Tasks' },
    ];
  };

  const byStatus = (status: Task['status']) =>
    plan.tasks().filter((t) => t.status === status).sort((a, b) => a.orderIndex - b.orderIndex);

  const completedIds = () => new Set(plan.tasks().filter((t) => t.status === 'completed').map((t) => t.id));
  const eligibleCount = () =>
    plan.tasks().filter((t) => t.status === 'pending' && t.dependencies.every((d) => completedIds().has(d))).length;
  const planLocked = () => plan.activePlan()?.status === 'locked';

  const counts = createMemo(() => ({
    total:       plan.tasks().length,
    done:        plan.tasks().filter((t) => t.status === 'completed').length,
    running:     plan.tasks().filter((t) => t.status === 'in_progress').length,
    failed:      plan.tasks().filter((t) => t.status === 'failed').length,
    pending:     plan.tasks().filter((t) => t.status === 'pending').length,
  }));

  const progress = () => counts().total > 0 ? Math.round((counts().done / counts().total) * 100) : 0;

  const handleStartAll = async () => {
    setStartingAll(true);
    setStartError('');
    try {
      await plan.startAllTasks();
    } catch (e: any) {
      setStartError(e?.message || String(e));
    } finally {
      setStartingAll(false);
    }
  };

  return (
    <div class="flex flex-col h-screen bg-[color:var(--bg-base)]">

      {/* ── header ── */}
      <header
        class="h-13 shrink-0 border-b border-[color:var(--border-subtle)] flex items-center px-4 gap-3"
        style={{ background: 'linear-gradient(var(--tint),var(--tint)) rgba(17,17,20,0.9)', 'z-index': 100 }}
      >
        <Breadcrumb items={breadcrumbs()} />
        <div class="flex-1" />

        {/* Progress pill */}
        <Show when={counts().total > 0}>
          <div class="hidden sm:flex items-center gap-2.5 px-3 py-1.5 rounded-lg bg-[color:var(--bg-elevated)] border border-[color:var(--border-subtle)]">
            <div class="w-20 h-1.5 rounded-full bg-zinc-800 overflow-hidden">
              <div
                class="h-full rounded-full bg-emerald-400 transition-all duration-500"
                style={{ width: `${progress()}%` }}
              />
            </div>
            <span class="text-[11px] text-zinc-300 font-medium tabular-nums">
              {counts().done}/{counts().total}
            </span>
            <Show when={counts().running > 0}>
              <span class="text-[10px] text-blue-400 font-medium flex items-center gap-1">
                <span class="w-1.5 h-1.5 rounded-full bg-blue-400 animate-pulse inline-block" />
                {counts().running}
              </span>
            </Show>
          </div>
        </Show>

        {/* Start All */}
        <Show when={planLocked() && eligibleCount() > 0}>
          <button
            type="button"
            onClick={handleStartAll}
            disabled={startingAll()}
            class="h-8 px-3.5 rounded-lg bg-[color:var(--accent)] hover:bg-[color:var(--accent-hover)]
                   text-[color:var(--on-primary)] text-[12px] font-semibold flex items-center gap-1.5
                   transition shadow-sm disabled:opacity-60"
          >
            <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M5 3l14 9-14 9V3z" />
            </svg>
            {startingAll() ? 'Starting…' : `Start ${eligibleCount()}`}
          </button>
        </Show>
      </header>

      {/* ── error banner ── */}
      <Show when={startError()}>
        <div class="px-4 py-2.5 flex items-start gap-2 text-[12px] text-red-300 bg-red-500/10 border-b border-red-500/20 shrink-0">
          <svg class="w-4 h-4 text-red-400 shrink-0 mt-px" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <div>
            <p class="font-semibold mb-0.5">Failed to start some tasks:</p>
            <p class="whitespace-pre-wrap text-red-300/80">{startError()}</p>
          </div>
          <button
            type="button"
            onClick={() => setStartError('')}
            class="ml-auto w-5 h-5 rounded text-red-400 hover:text-red-300 flex items-center justify-center shrink-0"
          >
            <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </Show>

      {/* ── breakdown banner ── */}
      <Show when={plan.activePlan()?.breakdownStatus === 'in_progress'}>
        <div class="px-4 py-2.5 flex items-center gap-2 text-[12px] text-amber-300 bg-amber-400/8 border-b border-amber-400/20 shrink-0">
          <svg class="w-3.5 h-3.5 animate-spin shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v4m0 8v4m8-8h-4M8 12H4m13.657-5.657l-2.829 2.829M9.172 14.828l-2.829 2.829m11.314 0l-2.829-2.829M9.172 9.172L6.343 6.343" />
          </svg>
          AI is analysing the plan and generating tasks…
        </div>
      </Show>

      {/* ── kanban ── */}
      <div class="flex-1 overflow-x-auto overflow-y-hidden">
        <div class="flex h-full gap-4 p-5 w-full">
          <For each={COLUMNS}>
            {(col) => {
              const colTasks = () => byStatus(col.status);
              const sm = STATUS_META[col.status];
              return (
                <div class="flex-1 min-w-[260px] flex flex-col gap-3">
                  {/* Column header */}
                  <div class="flex items-center gap-2 px-1">
                    <span class={`w-2 h-2 rounded-full ${sm.dot}`} />
                    <span class="text-[12px] font-semibold text-zinc-300">{col.label}</span>
                    <span class={`ml-0.5 text-[10px] font-semibold px-1.5 py-0.5 rounded-full border ${sm.badge}`}>
                      {colTasks().length}
                    </span>
                  </div>

                  {/* Drop zone / cards */}
                  <div
                    class={`flex-1 overflow-y-auto rounded-xl border p-2 space-y-2
                            ${colTasks().length === 0
                              ? 'border-dashed border-[color:var(--border-subtle)] bg-transparent'
                              : 'border-[color:var(--border-subtle)] bg-[color:var(--bg-elevated)]/30'}`}
                  >
                    <For each={colTasks()}>
                      {(task) => (
                        <KanbanCard
                          task={task}
                          onSelect={(t) => setSelectedTask(t)}
                        />
                      )}
                    </For>
                    <Show when={colTasks().length === 0}>
                      <div class="h-28 flex items-center justify-center">
                        <span class="text-[11px] text-zinc-700">No tasks</span>
                      </div>
                    </Show>
                  </div>
                </div>
              );
            }}
          </For>
        </div>
      </div>

      {/* ── task detail drawer ── */}
      <TaskDrawer
        task={selectedTask()}
        onClose={() => setSelectedTask(null)}
      />
    </div>
  );
}
