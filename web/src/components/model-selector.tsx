import { createSignal, For, Show, createMemo } from 'solid-js';
import { useSession } from '../context/session';
import type { ModelInfo } from '../api/client';

const PROVIDER_LABELS: Record<string, string> = {
  anthropic: 'Anthropic',
  openai: 'OpenAI',
  openrouter: 'OpenRouter',
  google: 'Google',
  mistral: 'Mistral',
};

const PROVIDER_DOT: Record<string, string> = {
  anthropic: 'bg-orange-400',
  openai: 'bg-emerald-400',
  openrouter: 'bg-violet-400',
  google: 'bg-blue-400',
  mistral: 'bg-rose-400',
};

const PROVIDER_TEXT: Record<string, string> = {
  anthropic: 'text-orange-400',
  openai: 'text-emerald-400',
  openrouter: 'text-violet-400',
  google: 'text-blue-400',
  mistral: 'text-rose-400',
};

interface ModelSelectorProps {
  selectedModel?: () => string;
  models?: () => ModelInfo[];
  onSelect?: (modelId: string) => void;
  placement?: 'top' | 'bottom';
}

export default function ModelSelector(props: ModelSelectorProps = {}) {
  const session = useSession();
  const [open, setOpen] = createSignal(false);

  const allModels = () => (props.models ? props.models() : session.models());
  const enabledModels = createMemo(() => allModels().filter((m) => m.enabled));

  const grouped = createMemo((): Map<string, ModelInfo[]> => {
    const map = new Map<string, ModelInfo[]>();
    for (const m of enabledModels()) {
      const list = map.get(m.providerId) || [];
      list.push(m);
      map.set(m.providerId, list);
    }
    return map;
  });

  const selectedModelInfo = (): ModelInfo | undefined => {
    const id = props.selectedModel ? props.selectedModel() : session.selectedModel();
    // Prefer enabled models, but fall back to all models so a session whose
    // model has since been disabled still shows the correct label in the trigger.
    return enabledModels().find((m) => m.id === id) ?? allModels().find((m) => m.id === id);
  };

  const handleSelect = (modelId: string) => {
    if (props.onSelect) {
      props.onSelect(modelId);
    } else {
      session.selectModel(modelId);
    }
    setOpen(false);
  };

  return (
    <div class="relative">
      <button
        type="button"
        onClick={() => setOpen(!open())}
        class="flex items-center gap-1.5 px-2 py-1 h-8 text-[12px] font-medium text-zinc-300
               hover:bg-[color:var(--bg-hover)] rounded-md
               transition-colors whitespace-nowrap max-w-[200px]"
      >
        <span class={`w-1.5 h-1.5 rounded-full ${
          PROVIDER_DOT[selectedModelInfo()?.providerId || ''] || 'bg-zinc-500'
        }`} />
        <span class="truncate">{selectedModelInfo()?.name || 'Select model'}</span>
        <svg class="w-3 h-3 text-zinc-500 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
        </svg>
      </button>
      <Show when={open()}>
        <div class="fixed inset-0 z-40" onClick={() => setOpen(false)} />
        <div
          class={`absolute left-0 w-72 bg-[color:var(--bg-overlay)]
                 border border-[color:var(--border-default)] rounded-xl shadow-[0_16px_40px_rgba(0,0,0,0.5)]
                 z-50 py-1 max-h-96 overflow-y-auto
                 ${(props.placement ?? 'top') === 'bottom' ? 'top-full mt-1.5' : 'bottom-full mb-1.5'}`}
        >
          <For each={[...grouped().entries()]}>
            {([providerId, models]) => (
              <div>
                <div class={`px-3 pt-2 pb-1 text-[10px] font-semibold uppercase tracking-wider flex items-center gap-1.5 ${
                  PROVIDER_TEXT[providerId] || 'text-zinc-500'
                }`}>
                  <span class={`w-1.5 h-1.5 rounded-full ${
                    PROVIDER_DOT[providerId] || 'bg-zinc-500'
                  }`} />
                  {PROVIDER_LABELS[providerId] || providerId}
                </div>
                <For each={models}>
                  {(model) => {
                    const isSelected = () => (props.selectedModel ? props.selectedModel() : session.selectedModel()) === model.id;
                    return (
                      <button
                        type="button"
                        onClick={() => handleSelect(model.id)}
                        class={`w-full text-left px-3 py-1.5 text-[13px] transition-colors
                                flex items-center justify-between gap-2
                                ${isSelected()
                                  ? 'bg-[color:var(--accent-soft)] text-[color:var(--accent)]'
                                  : 'text-zinc-200 hover:bg-[color:var(--bg-hover)]'
                                }`}
                      >
                        <span class="truncate flex items-center gap-2">
                          <Show when={isSelected()} fallback={<span class="w-3.5" />}>
                            <svg class="w-3.5 h-3.5 text-[color:var(--accent)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                              <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                            </svg>
                          </Show>
                          {model.name}
                        </span>
                        <div class="flex items-center gap-1 shrink-0">
                          <Show when={model.inputPricePerM > 0 || model.outputPricePerM > 0}>
                            <span class="text-[9px] text-zinc-500 bg-zinc-500/10 px-1 py-0.5 rounded font-mono tabular-nums">
                              ${fmtPrice(model.inputPricePerM)}/${fmtPrice(model.outputPricePerM)}
                            </span>
                          </Show>
                          <Show when={model.isCustom}>
                            <span class="text-[9px] text-violet-400 bg-violet-500/10 px-1 py-0.5 rounded">custom</span>
                          </Show>
                          <Show when={model.default}>
                            <span class="text-[9.5px] text-zinc-500 uppercase tracking-wider">default</span>
                          </Show>
                        </div>
                      </button>
                    );
                  }}
                </For>
              </div>
            )}
          </For>
          <Show when={enabledModels().length === 0}>
            <div class="px-3 py-4 text-[12px] text-zinc-500 text-center">
              No models available
            </div>
          </Show>
        </div>
      </Show>
    </div>
  );
}

function fmtPrice(n: number): string {
  if (n === 0) return '0';
  if (n < 0.01) return n.toFixed(2);
  if (n < 1) return n.toFixed(2).replace(/0+$/, '').replace(/\.$/, '');
  if (Number.isInteger(n)) return String(n);
  return n.toFixed(2).replace(/0+$/, '').replace(/\.$/, '');
}