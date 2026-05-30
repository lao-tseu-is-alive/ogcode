import { Show, For, createMemo, createSignal, createEffect, onMount } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import { useServer } from '../../context/server';
import { useSession } from '../../context/session';
import { useTheme } from '../../context/theme';
import { getMemoryConfig, setMemoryConfig, fetchMemoryModels, getCallGraphAgentConfig, setCallGraphAgentConfig, getSearchConfig, setSearchConfig } from '../../api/client';
import { EMBED_PROVIDERS, CHAT_PROVIDERS } from '../../lib/providers';

const PROVIDER_LABELS: Record<string, string> = {
  anthropic: 'Anthropic',
  openai: 'OpenAI',
  openrouter: 'OpenRouter',
  ollama: 'Ollama',
  google: 'Google',
  mistral: 'Mistral',
};

export default function GeneralSettings() {
  const server = useServer();
  const session = useSession();
  const theme = useTheme();
  const navigate = useNavigate();

  const stats = createMemo(() => {
    const all = session.models();
    const enabled = all.filter((m) => m.enabled);
    const providers = new Set(all.map((m) => m.providerId));
    const customs = all.filter((m) => m.isCustom);
    return {
      total: all.length,
      enabled: enabled.length,
      providers: providers.size,
      customs: customs.length,
    };
  });

  const defaultModel = createMemo(() => {
    const all = session.models();
    return all.find((m) => m.default && m.enabled) || all.find((m) => m.enabled);
  });

  return (
    <div class="max-w-3xl mx-auto px-8 py-10">
      {/* Page header */}
      <header class="mb-10">
        <h1 class="text-2xl font-semibold tracking-tight text-zinc-50">General</h1>
        <p class="text-[13px] text-zinc-500 mt-1.5">
          Workspace information and high-level defaults for this session.
        </p>
      </header>

      <div class="space-y-6">
        {/* Workspace card */}
        <Card title="Workspace" description="The directory ogcode is operating on.">
          <Row label="Directory">
            <span class="font-mono text-[12px] text-zinc-200 break-all">
              {server.directory() || '—'}
            </span>
          </Row>
          <Row label="Connection">
            <div class="flex items-center gap-2">
              <span class={`w-1.5 h-1.5 rounded-full ${server.connected() ? 'bg-emerald-400' : 'bg-zinc-600'}`} />
              <span class="text-[12px] text-zinc-200">
                {server.connected() ? 'Live' : 'Disconnected'}
              </span>
            </div>
          </Row>
          <Show when={server.branch()}>
            <Row label="Branch">
              <span class="font-mono text-[12px] text-zinc-200">{server.branch()}</span>
            </Row>
          </Show>
        </Card>

        {/* Memory card */}
        <Card
          title="Agentic Session Memory"
          description="Helps ogcode remember your past work within this session and bring back what's relevant when you need it."
        >
          <MemoryConfigForm />
        </Card>

        {/* Call graph card */}
        <Card
          title="Call Graph Agent Instructions"
          description="When enabled, build and plan agents proactively map every symbol and relationship they encounter into a persistent code knowledge graph."
        >
          <CallGraphConfigForm />
        </Card>

        {/* Search card */}
        <Card
          title="Web Search Agent"
          description="Enables parallel web research via headless Chrome. Build and note agents can call deep_search to fetch and synthesise live information. Requires a server restart to take effect."
        >
          <SearchConfigForm />
        </Card>

        {/* Theme card */}
        <Card
          title="Theme"
          description="Set a primary accent color for this project. The full palette is derived automatically and persisted per directory."
        >
          <ThemePicker />
        </Card>

        {/* Models summary */}
        <Card
          title="Models"
          description="Manage which AI models are available across your sessions."
          action={
            <button
              type="button"
              onClick={() => navigate('/settings/models')}
              class="h-8 px-3 text-[12px] font-medium text-zinc-200 border border-[color:var(--border-default)]
                     hover:border-[color:var(--border-strong)] hover:bg-[color:var(--bg-hover)]
                     rounded-lg transition flex items-center gap-1.5"
            >
              Manage
              <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
              </svg>
            </button>
          }
        >
          <div class="grid grid-cols-4 gap-px rounded-lg overflow-hidden bg-[color:var(--border-subtle)]">
            <Stat label="Enabled" value={stats().enabled} />
            <Stat label="Available" value={stats().total} />
            <Stat label="Providers" value={stats().providers} />
            <Stat label="Custom" value={stats().customs} />
          </div>
          <Show when={defaultModel()}>
            <div class="pt-3 mt-3 border-t border-[color:var(--border-subtle)]">
              <div class="text-[11px] text-zinc-500 mb-1">Default model</div>
              <div class="flex items-center gap-2">
                <span class="text-[13px] text-zinc-100 font-medium">{defaultModel()!.name}</span>
                <span class="text-[11px] text-zinc-500">
                  {PROVIDER_LABELS[defaultModel()!.providerId] || defaultModel()!.providerId}
                </span>
              </div>
            </div>
          </Show>
        </Card>

        {/* Keyboard shortcuts */}
        <Card title="Keyboard" description="Shortcuts available throughout the app.">
          <div class="space-y-2">
            <Shortcut keys={['Enter']} description="Send message" />
            <Shortcut keys={['Shift', 'Enter']} description="Insert newline" />
            <Shortcut keys={['⌘', 'N']} description="Start new session" />
          </div>
        </Card>
      </div>
    </div>
  );
}

function Card(props: { title: string; description?: string; action?: any; children: any }) {
  return (
    <section class="rounded-xl border border-[color:var(--border-subtle)] bg-[color:var(--bg-surface)] overflow-hidden">
      <header class="px-5 pt-4 pb-3 flex items-start justify-between gap-4 border-b border-[color:var(--border-subtle)]">
        <div class="min-w-0">
          <h2 class="text-[14px] font-semibold text-zinc-100">{props.title}</h2>
          <Show when={props.description}>
            <p class="text-[12px] text-zinc-500 mt-0.5">{props.description}</p>
          </Show>
        </div>
        {props.action}
      </header>
      <div class="px-5 py-4">{props.children}</div>
    </section>
  );
}

function Row(props: { label: string; children: any }) {
  return (
    <div class="flex items-start justify-between gap-4 py-2 first:pt-0 last:pb-0">
      <div class="text-[12px] text-zinc-500 shrink-0 pt-0.5 w-24">{props.label}</div>
      <div class="flex-1 min-w-0 text-right">{props.children}</div>
    </div>
  );
}

function Stat(props: { label: string; value: number | string }) {
  return (
    <div class="bg-[color:var(--bg-surface)] px-3 py-3 text-center">
      <div class="text-[18px] font-semibold text-zinc-100 tabular-nums leading-none">{props.value}</div>
      <div class="text-[10.5px] text-zinc-500 mt-1.5 uppercase tracking-wider">{props.label}</div>
    </div>
  );
}

function Shortcut(props: { keys: string[]; description: string }) {
  return (
    <div class="flex items-center justify-between py-1">
      <span class="text-[12.5px] text-zinc-300">{props.description}</span>
      <div class="flex items-center gap-1">
        <For each={props.keys}>
          {(k, i) => (
            <span class="flex items-center gap-1">
              <Show when={i() > 0}>
                <span class="text-zinc-600 text-[10px]">+</span>
              </Show>
              <kbd class="px-1.5 py-0.5 rounded border border-[color:var(--border-default)] bg-[color:var(--bg-elevated)] font-mono text-[10.5px] text-zinc-300">
                {k}
              </kbd>
            </span>
          )}
        </For>
      </div>
    </div>
  );
}


const EMBED_MODEL_HINTS: Record<string, string> = {
  openai: 'text-embedding-3-small',
  openrouter: 'text-embedding-3-small',
  ollama: 'nomic-embed-text',
};

const CHAT_MODEL_HINTS: Record<string, string> = {
  anthropic: 'claude-sonnet-4-6',
  openai: 'gpt-4o',
  openrouter: 'anthropic/claude-sonnet-4.6',
  ollama: 'qwen3',
};

function MemoryConfigForm() {
  const [enabled, setEnabled] = createSignal(false);
  const [embedProvider, setEmbedProvider] = createSignal('openai');
  const [embedModel, setEmbedModel] = createSignal('');
  const [embedApiKey, setEmbedApiKey] = createSignal('');
  const [chatProvider, setChatProvider] = createSignal('');
  const [chatModel, setChatModel] = createSignal('');
  const [chatApiKey, setChatApiKey] = createSignal('');
  const [loading, setLoading] = createSignal(true);
  const [saving, setSaving] = createSignal(false);
  const [error, setError] = createSignal('');
  const [saved, setSaved] = createSignal(false);

  onMount(async () => {
    try {
      const cfg = await getMemoryConfig();
      setEnabled(cfg.enabled);
      setEmbedProvider(cfg.embedProviderId || 'openai');
      setEmbedModel(cfg.embedModel || '');
      setEmbedApiKey(cfg.embedApiKey || '');
      setChatProvider(cfg.chatProviderId || '');
      setChatModel(cfg.chatModel || '');
      setChatApiKey(cfg.chatApiKey || '');
    } catch {
      setError('Failed to load memory config');
    } finally {
      setLoading(false);
    }
  });

  const handleSave = async () => {
    setError('');
    setSaved(false);
    if (enabled() && !embedProvider()) {
      setError('Select a memory provider');
      return;
    }
    setSaving(true);
    try {
      await setMemoryConfig({
        enabled: enabled(),
        embedProviderId: embedProvider(),
        embedModel: embedModel(),
        embedApiKey: embedApiKey(),
        chatProviderId: chatProvider(),
        chatModel: chatModel(),
        chatApiKey: chatApiKey(),
      });
      setSaved(true);
      setTimeout(() => setSaved(false), 3000);
    } catch {
      setError('Failed to save memory config');
    } finally {
      setSaving(false);
    }
  };

  const inputClass = 'w-full h-8 px-2.5 rounded-md border border-[color:var(--border-default)] bg-[color:var(--bg-elevated)] text-[12px] text-zinc-200 font-mono focus:outline-none focus:border-[color:var(--accent)] transition disabled:opacity-40';
  const selectClass = 'w-full h-8 pl-2.5 pr-7 rounded-md border border-[color:var(--border-default)] bg-[color:var(--bg-elevated)] text-[12px] text-zinc-200 focus:outline-none focus:border-[color:var(--accent)] transition appearance-none cursor-pointer disabled:opacity-40';

  return (
    <Show when={!loading()} fallback={
      <div class="py-4 flex items-center gap-2 text-[12px] text-zinc-500">
        <svg class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v4m0 8v4m8-8h-4M8 12H4" />
        </svg>
        Loading…
      </div>
    }>
      <div class="space-y-5">

        {/* ── Enable toggle ── */}
        <div class="flex items-center justify-between gap-4">
          <div class="min-w-0">
            <div class="text-[13px] text-zinc-100 font-medium">Enable agentic memory</div>
            <div class="text-[11.5px] text-zinc-500 mt-0.5 leading-snug">
              ogcode recalls relevant work from past sessions to give better, context-aware answers.
            </div>
          </div>
          <button
            type="button"
            role="switch"
            aria-checked={enabled()}
            onClick={() => setEnabled(v => !v)}
            class={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent
              transition-colors duration-200 focus:outline-none
              ${enabled() ? 'bg-[color:var(--accent)]' : 'bg-zinc-700'}`}
          >
            <span class={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow
              transition duration-200 ${enabled() ? 'translate-x-4' : 'translate-x-0'}`} />
          </button>
        </div>

        <Show when={enabled()}>
          <div class="space-y-4">

            {/* ── Section 1: Vector Store ── */}
            <div class="rounded-lg border border-[color:var(--border-subtle)] overflow-hidden">
              <div class="px-4 py-2.5 bg-[color:var(--bg-elevated)] border-b border-[color:var(--border-subtle)] flex items-center gap-2">
                <svg class="w-3.5 h-3.5 text-[color:var(--accent)] shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M4 7v10c0 2 1 3 3 3h10c2 0 3-1 3-3V7c0-2-1-3-3-3H7C5 4 4 5 4 7z" />
                  <path stroke-linecap="round" stroke-linejoin="round" d="M4 10h16M4 14h16" />
                </svg>
                <span class="text-[11.5px] font-semibold text-zinc-300">Vector Store</span>
                <span class="ml-auto text-[10.5px] text-zinc-500">Stores and retrieves memory embeddings</span>
              </div>
              <div class="px-4 py-3 space-y-3">
                {/* Provider + Model side by side */}
                <div class="grid grid-cols-2 gap-3">
                  <div>
                    <label class="block text-[11px] text-zinc-500 mb-1.5">Provider</label>
                    <div class="relative">
                      <select
                        value={embedProvider()}
                        onChange={e => {
                          const p = e.currentTarget.value;
                          setEmbedProvider(p);
                          if (!embedModel()) setEmbedModel(EMBED_MODEL_HINTS[p] || '');
                        }}
                        class={selectClass}
                      >
                        <For each={EMBED_PROVIDERS}>
                          {(p) => <option value={p.id}>{p.label}</option>}
                        </For>
                      </select>
                      <svg class="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-zinc-500"
                        fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
                      </svg>
                    </div>
                  </div>
                  <div>
                    <label class="block text-[11px] text-zinc-500 mb-1.5">Embedding model</label>
                    <input
                      type="text"
                      value={embedModel()}
                      onInput={e => setEmbedModel(e.currentTarget.value)}
                      placeholder={EMBED_MODEL_HINTS[embedProvider()] || 'e.g. text-embedding-3-small'}
                      class={inputClass}
                    />
                  </div>
                </div>
                {/* API key */}
                <div>
                  <label class="block text-[11px] text-zinc-500 mb-1.5 flex items-center gap-1.5">
                    API key
                    <Show when={embedApiKey() === '__SET__'}>
                      <span class="text-emerald-500 font-medium">● set</span>
                    </Show>
                  </label>
                  <input
                    type="password"
                    value={embedApiKey() === '__SET__' ? '' : embedApiKey()}
                    onInput={e => setEmbedApiKey(e.currentTarget.value)}
                    placeholder={embedApiKey() === '__SET__' ? 'leave blank to keep existing key' : 'sk-… or leave blank to use env var'}
                    class={inputClass}
                  />
                </div>
              </div>
            </div>

            {/* ── Section 2: AI Understanding ── */}
            <div class="rounded-lg border border-[color:var(--border-subtle)] overflow-hidden">
              <div class="px-4 py-2.5 bg-[color:var(--bg-elevated)] border-b border-[color:var(--border-subtle)] flex items-center gap-2">
                <svg class="w-3.5 h-3.5 text-[color:var(--accent)] shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
                </svg>
                <span class="text-[11.5px] font-semibold text-zinc-300">AI Understanding</span>
                <span class="ml-auto text-[10.5px] text-zinc-500">Summarises and understands your history</span>
              </div>
              <div class="px-4 py-3 space-y-3">
                {/* Provider + Model side by side */}
                <div class="grid grid-cols-2 gap-3">
                  <div>
                    <label class="block text-[11px] text-zinc-500 mb-1.5">Provider</label>
                    <div class="relative">
                      <select
                        value={chatProvider()}
                        onChange={e => {
                          const p = e.currentTarget.value;
                          setChatProvider(p);
                          if (p && !chatModel()) setChatModel(CHAT_MODEL_HINTS[p] || '');
                          if (!p) setChatModel('');
                        }}
                        class={selectClass}
                      >
                        <For each={CHAT_PROVIDERS}>
                          {(p) => <option value={p.id}>{p.label}</option>}
                        </For>
                      </select>
                      <svg class="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-zinc-500"
                        fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
                      </svg>
                    </div>
                  </div>
                  <div>
                    <label class="block text-[11px] text-zinc-500 mb-1.5">Model</label>
                    <Show
                      when={chatProvider()}
                      fallback={
                        <input
                          type="text"
                          disabled
                          placeholder="Select a provider first"
                          class={inputClass}
                        />
                      }
                    >
                      <ModelPicker
                        provider={chatProvider()}
                        apiKey={chatApiKey()}
                        type="chat"
                        value={chatModel()}
                        onSelect={setChatModel}
                        placeholder={CHAT_MODEL_HINTS[chatProvider()] || 'e.g. claude-sonnet-4-6'}
                        inputClass={inputClass}
                        selectClass={selectClass}
                      />
                    </Show>
                  </div>
                </div>
                {/* API key — only when a provider is chosen */}
                <Show when={chatProvider()}>
                  <div>
                    <label class="block text-[11px] text-zinc-500 mb-1.5 flex items-center gap-1.5">
                      API key
                      <Show when={chatApiKey() === '__SET__'}>
                        <span class="text-emerald-500 font-medium">● set</span>
                      </Show>
                    </label>
                    <input
                      type="password"
                      value={chatApiKey() === '__SET__' ? '' : chatApiKey()}
                      onInput={e => setChatApiKey(e.currentTarget.value)}
                      placeholder={chatApiKey() === '__SET__' ? 'leave blank to keep existing key' : 'sk-… or leave blank to use env var'}
                      class={inputClass}
                    />
                  </div>
                </Show>
              </div>
            </div>

          </div>
        </Show>

        {/* ── Footer ── */}
        <div class="pt-3 border-t border-[color:var(--border-subtle)] flex items-center justify-between gap-3">
          <Show when={error()}>
            <p class="text-[11px] text-red-400">{error()}</p>
          </Show>
          <Show when={saved() && !error()}>
            <p class="text-[11px] text-emerald-400">Saved — restart ogcode to apply.</p>
          </Show>
          <Show when={!error() && !saved()}>
            <p class="text-[11px] text-zinc-600">Changes apply after restarting ogcode.</p>
          </Show>
          <button
            type="button"
            onClick={handleSave}
            disabled={saving()}
            class="shrink-0 h-8 px-4 text-[12px] font-medium rounded-lg transition
              bg-[color:var(--accent)] text-[color:var(--on-primary)] hover:bg-[color:var(--accent-hover)]
              disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1.5"
          >
            <Show when={saving()}>
              <svg class="w-3 h-3 animate-spin" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v4m0 8v4m8-8h-4M8 12H4" />
              </svg>
            </Show>
            {saving() ? 'Saving…' : 'Save'}
          </button>
        </div>

      </div>
    </Show>
  );
}

function CallGraphConfigForm() {
  const [enabled, setEnabled] = createSignal(true);
  const [loading, setLoading] = createSignal(true);
  const [saving, setSaving] = createSignal(false);

  onMount(async () => {
    try {
      const cfg = await getCallGraphAgentConfig();
      setEnabled(cfg.enabled);
    } catch {
      // default stays true
    } finally {
      setLoading(false);
    }
  });

  const handleToggle = async () => {
    const next = !enabled();
    setEnabled(next);
    setSaving(true);
    try {
      await setCallGraphAgentConfig({ enabled: next });
    } catch {
      setEnabled(!next); // revert on error
    } finally {
      setSaving(false);
    }
  };

  return (
    <Show when={!loading()} fallback={
      <div class="py-4 flex items-center gap-2 text-[12px] text-zinc-500">
        <svg class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v4m0 8v4m8-8h-4M8 12H4" />
        </svg>
        Loading…
      </div>
    }>
      <div class="flex items-center justify-between gap-4">
        <div class="min-w-0">
          <div class="text-[13px] text-zinc-100 font-medium">Enable call graph instructions</div>
          <div class="text-[11.5px] text-zinc-500 mt-0.5 leading-snug">
            Agents read and update the call graph as they explore and modify code. Takes effect immediately for new sessions.
          </div>
        </div>
        <button
          type="button"
          role="switch"
          aria-checked={enabled()}
          disabled={saving()}
          onClick={handleToggle}
          class={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent
            transition-colors duration-200 focus:outline-none disabled:opacity-50
            ${enabled() ? 'bg-[color:var(--accent)]' : 'bg-zinc-700'}`}
        >
          <span class={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow
            transition duration-200 ${enabled() ? 'translate-x-4' : 'translate-x-0'}`} />
        </button>
      </div>
    </Show>
  );
}

function SearchConfigForm() {
  const server = useServer();
  const [enabled, setEnabled] = createSignal(false);
  const [useRealProfile, setUseRealProfile] = createSignal(false);
  const [loading, setLoading] = createSignal(true);
  const [saving, setSaving] = createSignal(false);
  const [restartNeeded, setRestartNeeded] = createSignal(false);

  onMount(async () => {
    try {
      const cfg = await getSearchConfig();
      setEnabled(cfg.enabled);
      setUseRealProfile(cfg.useRealProfile);
    } catch {
      // defaults stay false
    } finally {
      setLoading(false);
    }
  });

  const save = async (next: { enabled: boolean; useRealProfile: boolean }) => {
    setSaving(true);
    try {
      await setSearchConfig(next);
      setRestartNeeded(true);
    } finally {
      setSaving(false);
    }
  };

  const handleToggle = async () => {
    const next = !enabled();
    setEnabled(next);
    try {
      await save({ enabled: next, useRealProfile: useRealProfile() });
    } catch {
      setEnabled(!next);
    }
  };

  const handleProfileToggle = async () => {
    const next = !useRealProfile();
    setUseRealProfile(next);
    try {
      await save({ enabled: enabled(), useRealProfile: next });
    } catch {
      setUseRealProfile(!next);
    }
  };

  return (
    <Show when={!loading()} fallback={
      <div class="py-4 flex items-center gap-2 text-[12px] text-zinc-500">
        <svg class="w-3.5 h-3.5 animate-spin" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v4m0 8v4m8-8h-4M8 12H4" />
        </svg>
        Loading…
      </div>
    }>
      <div class="space-y-3">
        <div class="flex items-center justify-between gap-4">
          <div class="min-w-0">
            <div class="text-[13px] text-zinc-100 font-medium">Enable web search</div>
            <div class="text-[11.5px] text-zinc-500 mt-0.5 leading-snug">
              Starts a headless Chrome bridge at server launch. Build and note agents gain <code class="font-mono bg-zinc-800 px-1 rounded">deep_search</code> and <code class="font-mono bg-zinc-800 px-1 rounded">web_search</code> tools.
            </div>
          </div>
          <button
            type="button"
            role="switch"
            aria-checked={enabled()}
            disabled={saving()}
            onClick={handleToggle}
            class={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent
              transition-colors duration-200 focus:outline-none disabled:opacity-50
              ${enabled() ? 'bg-[color:var(--accent)]' : 'bg-zinc-700'}`}
          >
            <span class={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow
              transition duration-200 ${enabled() ? 'translate-x-4' : 'translate-x-0'}`} />
          </button>
        </div>
        {/* Bridge live status */}
        <Show when={enabled() && !restartNeeded()}>
          <Show
            when={server.searchRunning()}
            fallback={
              <div class="flex items-center gap-2 text-[11.5px] text-red-400 bg-red-400/10 rounded-md px-3 py-2">
                <span class="w-1.5 h-1.5 rounded-full bg-red-400 shrink-0" />
                Bridge failed to start. Check that Node.js is installed and run <code class="font-mono bg-zinc-800 px-1 rounded">npx playwright install chromium</code> in <code class="font-mono bg-zinc-800 px-1 rounded">~/.local/share/ogcode/search-bridge/</code>, then restart.
              </div>
            }
          >
            <div class="flex items-center gap-2 text-[11.5px] text-emerald-400">
              <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 shrink-0" />
              Bridge running — web_search, fetch_page and deep_search tools are active.
            </div>
          </Show>
        </Show>

        <Show when={restartNeeded()}>
          <div class="flex items-center gap-2 text-[11.5px] text-amber-400 bg-amber-400/10 rounded-md px-3 py-2">
            <svg class="w-3.5 h-3.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z" />
            </svg>
            Restart the server for this change to take effect.
          </div>
        </Show>
        <Show when={enabled()}>
          <div class="border-t border-zinc-800 pt-3 mt-1 space-y-3">
            {/* Real Chrome profile toggle */}
            <div class="flex items-center justify-between gap-4">
              <div class="min-w-0">
                <div class="text-[13px] text-zinc-100 font-medium">Use real Chrome profile</div>
                <div class="text-[11.5px] text-zinc-500 mt-0.5 leading-snug">
                  Uses your Chrome cookies and logins for better search results. Chrome must be <strong class="text-zinc-400">fully closed</strong> before starting ogcode.
                </div>
              </div>
              <button
                type="button"
                role="switch"
                aria-checked={useRealProfile()}
                disabled={saving()}
                onClick={handleProfileToggle}
                class={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent
                  transition-colors duration-200 focus:outline-none disabled:opacity-50
                  ${useRealProfile() ? 'bg-[color:var(--accent)]' : 'bg-zinc-700'}`}
              >
                <span class={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow
                  transition duration-200 ${useRealProfile() ? 'translate-x-4' : 'translate-x-0'}`} />
              </button>
            </div>

            {/* Real profile warning */}
            <Show when={useRealProfile()}>
              <div class="flex items-center gap-2 text-[11.5px] text-amber-400 bg-amber-400/10 rounded-md px-3 py-2">
                <svg class="w-3.5 h-3.5 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z" />
                </svg>
                Close Chrome completely before restarting ogcode. Chrome locks its profile — two processes cannot share it simultaneously.
              </div>
            </Show>
          </div>
        </Show>
      </div>
    </Show>
  );
}

interface ModelPickerProps {
  provider: string;
  apiKey: string;
  type: 'embed' | 'chat';
  value: string;
  onSelect: (v: string) => void;
  placeholder: string;
  inputClass: string;
  selectClass: string;
}

function ModelPicker(props: ModelPickerProps) {
  const [models, setModels] = createSignal<string[]>([]);
  const [fetching, setFetching] = createSignal(false);
  const [fetchError, setFetchError] = createSignal('');

  const doFetch = async (provider: string, apiKey: string) => {
    if (!provider) { setModels([]); return; }
    setFetching(true);
    setFetchError('');
    try {
      // Don't forward the sentinel — backend uses the stored key.
      const key = (apiKey && apiKey !== '__SET__') ? apiKey : undefined;
      const list = await fetchMemoryModels(provider, props.type, key);
      setModels(list ?? []);
      // Auto-select first model if nothing is selected yet.
      if (list?.length && !props.value) props.onSelect(list[0]);
    } catch (e: any) {
      setFetchError(e?.message ?? 'Failed to fetch models');
      setModels([]);
    } finally {
      setFetching(false);
    }
  };

  // Re-fetch whenever provider changes.
  createEffect(() => {
    const p = props.provider;
    const k = props.apiKey;
    doFetch(p, k);
  });

  const chevron = (
    <svg class="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-zinc-500"
      fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
      <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
    </svg>
  );

  return (
    <div class="space-y-1">
      <Show when={fetching()}>
        <div class="flex items-center gap-2 h-8 px-2.5 rounded-md border border-[color:var(--border-default)] bg-[color:var(--bg-elevated)]">
          <svg class="w-3 h-3 animate-spin text-zinc-500 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v4m0 8v4m8-8h-4M8 12H4" />
          </svg>
          <span class="text-[12px] text-zinc-500">Fetching models…</span>
        </div>
      </Show>

      <Show when={!fetching()}>
        <Show when={models().length > 0}
          fallback={
            <input
              type="text"
              value={props.value}
              onInput={e => props.onSelect(e.currentTarget.value)}
              placeholder={props.placeholder}
              class={props.inputClass}
            />
          }
        >
          <div class="relative">
            <select
              value={props.value}
              onChange={e => props.onSelect(e.currentTarget.value)}
              class={props.selectClass}
            >
              <option value="">— select model —</option>
              <For each={models()}>
                {(m) => <option value={m}>{m}</option>}
              </For>
            </select>
            {chevron}
          </div>
        </Show>
      </Show>

      <div class="flex items-center justify-between">
        <Show when={fetchError()}>
          <p class="text-[10.5px] text-amber-500">{fetchError()} — type a model name above.</p>
        </Show>
        <Show when={!fetching() && props.provider}>
          <button
            type="button"
            onClick={() => doFetch(props.provider, props.apiKey)}
            class="ml-auto text-[10.5px] text-zinc-500 hover:text-zinc-300 transition flex items-center gap-1"
          >
            <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            Refresh
          </button>
        </Show>
      </div>
    </div>
  );
}

function ThemePicker() {
  const themeCtx = useTheme();
  const [saving, setSaving] = createSignal(false);
  const [error, setError] = createSignal('');

  const isValidHex = (s: string) => /^#[0-9a-fA-F]{6}$/.test(s) || /^#[0-9a-fA-F]{3}$/.test(s);

  const handleInput = async (hex: string) => {
    setError('');
    if (!isValidHex(hex)) {
      setError('Invalid hex color (use #RRGGBB)');
      return;
    }
    setSaving(true);
    try {
      await themeCtx.setPrimaryColor(hex);
    } catch {
      setError('Failed to save theme');
    } finally {
      setSaving(false);
    }
  };

  const presets = [
    { label: 'Blue', hex: '#3b82f6' },
    { label: 'Indigo', hex: '#6366f1' },
    { label: 'Violet', hex: '#8b5cf6' },
    { label: 'Rose', hex: '#f43f5e' },
    { label: 'Amber', hex: '#f59e0b' },
    { label: 'Emerald', hex: '#10b981' },
    { label: 'Cyan', hex: '#06b6d4' },
    { label: 'Pink', hex: '#ec4899' },
  ];

  return (
    <div class="space-y-4">
      <div class="flex items-center gap-3">
        {/* Native color picker */}
        <div class="relative">
          <input
            type="color"
            value={themeCtx.primaryColor()}
            onInput={(e) => handleInput(e.currentTarget.value)}
            disabled={saving()}
            class="w-10 h-10 rounded-lg border-2 border-[color:var(--border-default)] cursor-pointer
                   bg-transparent appearance-none [&::-webkit-color-swatch-wrapper]:p-0
                   [&::-webkit-color-swatch]:rounded-md [&::-webkit-color-swatch]:border-none"
          />
        </div>
        <div class="flex-1">
          <div class="flex items-center gap-2">
            <input
              type="text"
              value={themeCtx.primaryColor()}
              onChange={(e) => handleInput(e.currentTarget.value)}
              disabled={saving()}
              placeholder="#3b82f6"
              class="w-28 h-8 px-2.5 rounded-md border border-[color:var(--border-default)]
                     bg-[color:var(--bg-elevated)] text-[12px] font-mono text-zinc-200
                     focus:outline-none focus:border-[color:var(--accent)] transition"
            />
            <Show when={saving()}>
              <svg class="w-3.5 h-3.5 animate-spin text-zinc-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v4m0 8v4m8-8h-4M8 12H4" />
              </svg>
            </Show>
          </div>
          <Show when={error()}>
            <p class="text-[11px] text-red-400 mt-1">{error()}</p>
          </Show>
        </div>
        {/* Live preview swatch with derived accent */}
        <div class="flex items-center gap-1.5">
          <span class="text-[11px] text-zinc-500 mr-1">Preview</span>
          <span class="w-6 h-6 rounded-md border border-[color:var(--border-default)]" style={{ background: 'var(--accent)' }} />
          <span class="w-6 h-6 rounded-md border border-[color:var(--border-default)]" style={{ background: 'var(--accent-hover)' }} />
          <span class="w-6 h-6 rounded-md border border-[color:var(--border-default)]" style={{ background: 'var(--accent-soft)' }} />
          <span class="w-6 h-6 rounded-md border border-[color:var(--border-default)]" style={{ background: 'linear-gradient(var(--tint), var(--tint)) var(--bg-surface)' }} title="Sidebar tint" />
        </div>
      </div>

      {/* Preset swatches */}
      <div>
        <div class="text-[11px] text-zinc-500 mb-2">Presets</div>
        <div class="flex flex-wrap gap-2">
          <For each={presets}>
            {(p) => (
              <button
                type="button"
                onClick={() => handleInput(p.hex)}
                disabled={saving()}
                title={p.label}
                class={`w-7 h-7 rounded-lg border-2 transition hover:scale-110
                  ${themeCtx.primaryColor() === p.hex
                    ? 'border-white ring-2 ring-white/20'
                    : 'border-[color:var(--border-default)] hover:border-[color:var(--border-strong)]'
                  }`}
                style={{ background: p.hex }}
              />
            )}
          </For>
        </div>
      </div>

      <div class="pt-3 mt-1 border-t border-[color:var(--border-subtle)]">
        <p class="text-[11px] text-zinc-500 leading-relaxed">
          Theme is saved per project directory. Reopening ogcode from this path restores your colors automatically.
        </p>
      </div>
    </div>
  );
}
