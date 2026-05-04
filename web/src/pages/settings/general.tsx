import { Show, For, createMemo, createSignal } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import { useServer } from '../../context/server';
import { useSession } from '../../context/session';
import { useTheme } from '../../context/theme';

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
          title="Memory"
          description="Agentic memory provides persistent context across sessions."
        >
          <Row label="Status">
            <div class="flex items-center gap-2">
              <span class={`w-1.5 h-1.5 rounded-full ${server.memoryEnabled() ? 'bg-emerald-400' : 'bg-zinc-600'}`} />
              <span class="text-[12px] text-zinc-200">
                {server.memoryEnabled() ? 'Enabled' : 'Disabled'}
              </span>
            </div>
          </Row>
          <Show when={server.memoryEnabled() && server.memoryProvider()}>
            <Row label="Source">
              <span class="text-[12px] text-zinc-200 font-mono">
                MCP ({server.memoryProvider()})
              </span>
            </Row>
          </Show>
          <div class="pt-3 mt-3 border-t border-[color:var(--border-subtle)]">
            <p class="text-[11px] text-zinc-500 leading-relaxed">
              Configure via <code class="px-1 py-0.5 rounded bg-[color:var(--bg-elevated)] border border-[color:var(--border-subtle)] text-zinc-400 font-mono">OGCODE_AGENTIC_MEMORY_MODE</code> environment variable.
            </p>
          </div>
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
