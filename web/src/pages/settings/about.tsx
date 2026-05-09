import { For, createSignal, createEffect, onMount, Show, createMemo } from 'solid-js';
import { getVersion, checkForUpdate, type VersionResponse } from '../../api/client';

const APP_VERSION = '0.1.5';

interface Highlight {
  title: string;
  body: string;
  icon: string;
}

const HIGHLIGHTS: Highlight[] = [
  {
    title: 'Local-first',
    body: 'The agent runs on your machine with full read & write access to your project.',
    icon: 'M3.75 9.75l7.5-6 7.5 6m-13.5 0v9a1.5 1.5 0 001.5 1.5h3.75v-6h4.5v6h3.75a1.5 1.5 0 001.5-1.5v-9m-13.5 0H3m18 0h-1.5',
  },
  {
    title: 'Bring your own model',
    body: 'Connect Anthropic, OpenAI, OpenRouter, Google, Mistral, or any custom endpoint.',
    icon: 'M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.847.813a4.5 4.5 0 00-3.09 3.091z',
  },
  {
    title: 'Persistent sessions',
    body: 'Every conversation is saved and resumable. Pick up where you left off, anytime.',
    icon: 'M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z',
  },
  {
    title: 'Keyboard-first',
    body: 'Built for the terminal mindset. Send, abort, and switch sessions without leaving home row.',
    icon: 'M6.75 3.75h.008v.008H6.75v-.008zM6.75 7.5h.008v.008H6.75V7.5zm0 3.75h.008v.008H6.75v-.008zM10.5 3.75h.008v.008H10.5v-.008zM10.5 7.5h.008v.008H10.5V7.5zm0 3.75h.008v.008H10.5v-.008zM14.25 3.75h.008v.008h-.008v-.008zM14.25 7.5h.008v.008h-.008V7.5zm0 3.75h.008v.008h-.008v-.008zM17.25 3.75h.008v.008h-.008v-.008zM17.25 7.5h.008v.008h-.008V7.5zm0 3.75h.008v.008h-.008v-.008zM4.5 18.75h15a.75.75 0 00.75-.75v-1.5a.75.75 0 00-.75-.75h-15a.75.75 0 00-.75.75v1.5a.75.75 0 00.75.75z',
  },
];

interface BuildRow {
  label: string;
  value: string;
  mono?: boolean;
  update?: {
    available: boolean;
    latestVersion: string;
    installCommand: string;
  };
}

const BUILD: BuildRow[] = [
  { label: 'Frontend', value: 'SolidJS · Vite',   mono: false },
  { label: 'Engine',   value: 'Go',               mono: false },
  { label: 'License',  value: 'MIT',              mono: false },
];

export default function AboutSettings() {
  const [versionInfo, setVersionInfo] = createSignal<VersionResponse | null>(null);
  const [isChecking, setIsChecking] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);

  onMount(async () => {
    try {
      const info = await getVersion();
      setVersionInfo(info);
      setError(null);
    } catch (err) {
      console.error('Failed to fetch version:', err);
      setError('Failed to check for updates');
    }
  });

  const handleCheckUpdate = async () => {
    setIsChecking(true);
    setError(null);
    try {
      const info = await checkForUpdate();
      const fullInfo = await getVersion();
      setVersionInfo(fullInfo);
    } catch (err) {
      console.error('Failed to check for updates:', err);
      setError('Failed to check for updates');
    } finally {
      setIsChecking(false);
    }
  };

  const serverVersion = createMemo(() => versionInfo()?.version ?? 'unknown');
  const latestVersion = createMemo(() => {
    const v = versionInfo()?.latestVersion;
    return v ? v.replace(/^v/, '') : null;
  });
  const hasUpdate = createMemo(() => versionInfo()?.updateAvailable ?? false);

  const copyInstallCommand = () => {
    if (versionInfo()?.installCommand) {
      navigator.clipboard.writeText(versionInfo()!.installCommand);
    }
  };
  return (
    <div class="max-w-3xl mx-auto px-8 py-12">
      {/* Hero */}
      <header class="relative mb-12">
        <div
          class="absolute -inset-x-12 -top-8 h-56 pointer-events-none -z-0"
          style={{
            background:
              'radial-gradient(ellipse 60% 60% at 50% 0%, var(--glow), transparent 70%)',
          }}
        />
        <div class="relative flex flex-col items-center text-center">
          <div class="relative mb-5">
            <div class="absolute inset-0 rounded-2xl bg-[color:var(--accent)] blur-xl opacity-40" />
            <div class="relative w-16 h-16 rounded-2xl bg-[color:var(--accent)] flex items-center justify-center shadow-lg ring-1 ring-white/10">
              <svg class="w-8 h-8 text-[color:var(--on-primary)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.4">
                <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
            </div>
          </div>
          <h1 class="text-3xl font-semibold tracking-tight text-zinc-50">ogcode</h1>
          <p class="text-[14px] text-zinc-400 mt-2 max-w-md leading-relaxed">
            A coding agent at home in your terminal — with a fast, modern web UI to drive it.
          </p>
          <div class="mt-5 inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full border border-[color:var(--border-subtle)] bg-[color:var(--bg-surface)] text-[11px] font-mono text-zinc-300">
            <span class="w-1.5 h-1.5 rounded-full bg-emerald-400" />
            v{APP_VERSION}
          </div>
        </div>
      </header>

      {/* Highlights */}
      <section class="mb-8">
        <div class="flex items-baseline justify-between mb-4">
          <h2 class="text-[12px] uppercase tracking-wider text-zinc-500 font-semibold">
            Highlights
          </h2>
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <For each={HIGHLIGHTS}>
            {(h) => (
              <div class="group rounded-xl border border-[color:var(--border-subtle)] bg-[color:var(--bg-surface)] p-4 hover:border-[color:var(--border-default)] transition">
                <div class="w-9 h-9 rounded-lg bg-[color:var(--bg-elevated)] border border-[color:var(--border-subtle)] flex items-center justify-center text-[color:var(--accent)] mb-3 transition">
                  <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.7">
                    <path stroke-linecap="round" stroke-linejoin="round" d={h.icon} />
                  </svg>
                </div>
                <div class="text-[13.5px] font-semibold text-zinc-100">{h.title}</div>
                <p class="text-[12px] text-zinc-500 mt-1 leading-relaxed">{h.body}</p>
              </div>
            )}
          </For>
        </div>
      </section>

      {/* Version info */}
      <section class="mb-8">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-[12px] uppercase tracking-wider text-zinc-500 font-semibold">
            Version
          </h2>
          <button
            onClick={handleCheckUpdate}
            disabled={isChecking()}
            class="text-[11px] text-zinc-400 hover:text-zinc-200 disabled:text-zinc-600 disabled:cursor-not-allowed flex items-center gap-1 transition-colors"
          >
            <Show when={isChecking()}>
              <svg class="w-3 h-3 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
            </Show>
            <Show when={!isChecking()}>
              <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
            </Show>
            {isChecking() ? 'Checking...' : 'Check for updates'}
          </button>
        </div>

        <div class="rounded-xl border border-[color:var(--border-subtle)] bg-[color:var(--bg-surface)] overflow-hidden">
          <div class="divide-y divide-[color:var(--border-subtle)]">
            {/* Current Version */}
            <div class="flex items-center justify-between px-5 py-3">
              <span class="text-[12px] text-zinc-500">Current</span>
              <div class="flex items-center gap-2">
                <Show when={hasUpdate()}>
                  <span class="w-1.5 h-1.5 rounded-full bg-amber-400 animate-pulse" />
                </Show>
                <span class="text-[12.5px] text-zinc-200 font-mono">{serverVersion()}</span>
              </div>
            </div>

            {/* Latest Version with Update Button */}
            <div class="flex items-center justify-between px-5 py-3">
              <span class="text-[12px] text-zinc-500">Latest</span>
              <div class="flex items-center gap-3">
                <Show when={hasUpdate()}
                  fallback={<span class="text-[12.5px] text-zinc-400 font-mono">{latestVersion()}</span>}
                >
                  <span class="text-[12.5px] text-amber-400 font-mono font-semibold">v{latestVersion()}</span>
                </Show>
              </div>
            </div>

            {/* Install Command - only show when update available */}
            <Show when={hasUpdate() && versionInfo()?.installCommand}>
              <div class="px-5 py-3 bg-zinc-800/50">
                <div class="flex items-center justify-between mb-2">
                  <span class="text-[11px] text-zinc-500 uppercase tracking-wider">Update Command</span>
                  <button
                    onClick={copyInstallCommand}
                    class="text-[10px] text-zinc-400 hover:text-zinc-300 flex items-center gap-1 transition-colors"
                  >
                    <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                    </svg>
                    Copy
                  </button>
                </div>
                <code class="block text-xs text-green-400 font-mono bg-zinc-900/50 rounded px-2 py-1.5">
                  {versionInfo()?.installCommand}
                </code>
                <a
                  href={versionInfo()?.releaseUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  class="inline-flex items-center gap-1 text-[10px] text-blue-400 hover:text-blue-300 mt-2"
                >
                  View full release notes
                  <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                  </svg>
                </a>
              </div>
            </Show>

            {/* Error */}
            <Show when={error()}>
              <div class="px-5 py-3 bg-red-900/20 border border-red-800/30">
                <span class="text-[12px] text-red-400">{error()}</span>
              </div>
            </Show>

            {/* No update message */}
            <Show when={!hasUpdate() && versionInfo() && !error()}>
              <div class="px-5 py-3 bg-emerald-900/20 border-t border-emerald-800/30">
                <div class="flex items-center gap-2">
                  <svg class="w-4 h-4 text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                  </svg>
                  <span class="text-[12px] text-emerald-400">You're on the latest version!</span>
                </div>
              </div>
            </Show>
          </div>
        </div>
      </section>

      {/* Build info */}
      <section>
        <div class="flex items-baseline justify-between mb-4">
          <h2 class="text-[12px] uppercase tracking-wider text-zinc-500 font-semibold">
            Build
          </h2>
        </div>
        <div class="rounded-xl border border-[color:var(--border-subtle)] bg-[color:var(--bg-surface)] overflow-hidden">
          <div class="divide-y divide-[color:var(--border-subtle)]">
            <For each={BUILD}>
              {(row) => (
                <div class="flex items-center justify-between px-5 py-3">
                  <span class="text-[12px] text-zinc-500">{row.label}</span>
                  <span class={`text-[12.5px] text-zinc-200 ${row.mono ? 'font-mono' : ''}`}>
                    {row.value}
                  </span>
                </div>
              )}
            </For>
          </div>
        </div>
      </section>

      {/* Footer signature */}
      <p class="mt-10 text-center text-[11px] text-zinc-600">
        Made for builders who live in the terminal.
      </p>
    </div>
  );
}
