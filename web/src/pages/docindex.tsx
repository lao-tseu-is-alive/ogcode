import { createSignal, Show, For } from 'solid-js';
import { useServer } from '../context/server';
import { useDocIndex } from '../context/docindex';
import SessionSidebar from '../components/session-sidebar';
import PlanSidebar from '../components/plan-sidebar';

function Sidebar() {
  const server = useServer();
  return (
    <Show when={server.mode() === 'plan'} fallback={<SessionSidebar />}>
      <PlanSidebar />
    </Show>
  );
}

function docBasename(docPath: string): string {
  return docPath.split('/').pop() || docPath;
}

function docExt(docPath: string): string {
  const name = docBasename(docPath);
  const dot = name.lastIndexOf('.');
  return dot >= 0 ? name.slice(dot + 1).toLowerCase() : '';
}

function formatDate(ms: number): string {
  if (!ms) return '';
  return new Date(ms).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
}

function DocTypeTag(props: { ext: string }) {
  const colors: Record<string, string> = {
    pdf: 'text-red-400 bg-red-400/10 border-red-400/20',
    docx: 'text-blue-400 bg-blue-400/10 border-blue-400/20',
    doc: 'text-blue-400 bg-blue-400/10 border-blue-400/20',
    html: 'text-orange-400 bg-orange-400/10 border-orange-400/20',
    md: 'text-zinc-300 bg-zinc-300/10 border-zinc-300/20',
  };
  const cls = colors[props.ext] || 'text-zinc-400 bg-zinc-400/10 border-zinc-400/20';
  return (
    <span class={`text-[9px] font-bold uppercase tracking-wider px-1.5 py-0.5 rounded border ${cls}`}>
      {props.ext || 'doc'}
    </span>
  );
}

export default function DocIndexPage() {
  const docIndex = useDocIndex();
  const [showBuildModal, setShowBuildModal] = createSignal(false);
  const [isRebuild, setIsRebuild] = createSignal(false);

  const openBuildModal = (rebuild: boolean) => {
    setIsRebuild(rebuild);
    setShowBuildModal(true);
  };

  const handleConfirmBuild = () => {
    setShowBuildModal(false);
    docIndex.build(isRebuild());
  };

  const totalPages = () => docIndex.docs().reduce((s, d) => s + d.pageCount, 0);

  return (
    <div class="flex h-screen w-full">
      <Sidebar />

      <div class="flex-1 flex flex-col overflow-hidden bg-[color:var(--bg-base)]">
        {/* Header */}
        <div class="shrink-0 border-b border-[color:var(--border-subtle)] bg-[color:var(--bg-surface)] px-4 py-2.5 flex items-center gap-3">
          <div class="flex items-center gap-2 shrink-0">
            <svg class="w-4 h-4 text-[color:var(--accent)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 6.042A8.967 8.967 0 006 3.75c-1.052 0-2.062.18-3 .512v14.25A8.987 8.987 0 016 18c2.305 0 4.408.867 6 2.292m0-14.25a8.966 8.966 0 016-2.292c1.052 0 2.062.18 3 .512v14.25A8.987 8.987 0 0018 18a8.967 8.967 0 00-6 2.292m0-14.25v14.25" />
            </svg>
            <h1 class="text-[14px] font-semibold text-zinc-100">Doc Index</h1>
          </div>

          <Show when={docIndex.docs().length > 0}>
            <span class="text-[11px] text-zinc-500 bg-[color:var(--bg-elevated)] px-2 py-0.5 rounded border border-[color:var(--border-subtle)]">
              {docIndex.docs().length} docs · {totalPages()} pages
            </span>
          </Show>

          <Show when={docIndex.building()}>
            <div class="flex items-center gap-1.5 text-[11px] text-zinc-400">
              <div class="w-2 h-2 rounded-full bg-[color:var(--accent)] animate-pulse" />
              Indexing…
            </div>
          </Show>

          <div class="flex-1" />

          <Show when={docIndex.docs().length > 0}>
            <button
              onClick={() => openBuildModal(true)}
              disabled={docIndex.building() || docIndex.loading()}
              class="h-7 px-2.5 rounded-md text-[12px] bg-[color:var(--bg-elevated)] border border-[color:var(--border-subtle)] text-zinc-400 hover:text-zinc-200 hover:border-[color:var(--border-default)] disabled:opacity-50 transition flex items-center gap-1.5"
            >
              <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              Rebuild
            </button>
          </Show>

          <button
            onClick={() => docIndex.refresh()}
            disabled={docIndex.loading() || docIndex.building()}
            class="h-7 px-2.5 rounded-md text-[12px] bg-[color:var(--bg-elevated)] border border-[color:var(--border-subtle)] text-zinc-400 hover:text-zinc-200 hover:border-[color:var(--border-default)] disabled:opacity-50 transition flex items-center gap-1.5"
          >
            <Show when={docIndex.loading()} fallback={
              <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
            }>
              <div class="w-3 h-3 border-2 border-[color:var(--accent)] border-t-transparent rounded-full animate-spin" />
            </Show>
            Refresh
          </button>

          <button
            onClick={() => openBuildModal(false)}
            disabled={docIndex.building() || docIndex.loading()}
            class="h-7 px-3 rounded-md text-[12px] font-medium bg-[color:var(--accent)] text-[color:var(--on-primary)] hover:bg-[color:var(--accent-hover)] disabled:opacity-50 disabled:cursor-not-allowed transition flex items-center gap-1.5"
          >
            <Show when={docIndex.building()} fallback={
              <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M5 3l14 9-14 9V3z" />
              </svg>
            }>
              <div class="w-3 h-3 border-2 border-white border-t-transparent rounded-full animate-spin" />
            </Show>
            {docIndex.building() ? 'Indexing…' : 'Index Docs'}
          </button>
        </div>

        {/* Content */}
        <div class="flex-1 overflow-y-auto">
          {/* Loading */}
          <Show when={docIndex.loading() && docIndex.docs().length === 0}>
            <div class="flex items-center justify-center h-full">
              <div class="flex flex-col items-center gap-3">
                <div class="w-5 h-5 border-2 border-[color:var(--accent)] border-t-transparent rounded-full animate-spin" />
                <p class="text-[12px] text-zinc-500">Loading…</p>
              </div>
            </div>
          </Show>

          {/* Empty state */}
          <Show when={docIndex.docs().length === 0 && !docIndex.loading()}>
            <div class="flex flex-col items-center justify-center h-full text-center px-8">
              <svg class="w-10 h-10 text-zinc-700 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M12 6.042A8.967 8.967 0 006 3.75c-1.052 0-2.062.18-3 .512v14.25A8.987 8.987 0 016 18c2.305 0 4.408.867 6 2.292m0-14.25a8.966 8.966 0 016-2.292c1.052 0 2.062.18 3 .512v14.25A8.987 8.987 0 0018 18a8.967 8.967 0 00-6 2.292m0-14.25v14.25" />
              </svg>
              <p class="text-[14px] font-semibold text-zinc-300">No documents indexed</p>
              <p class="text-[12px] text-zinc-500 mt-1.5 max-w-[280px] leading-relaxed">
                Index your documents so agents can search and navigate them intelligently.
              </p>
              <Show when={!docIndex.building()}>
                <button
                  onClick={() => openBuildModal(false)}
                  class="mt-4 h-8 px-4 rounded-lg text-[12px] font-medium bg-[color:var(--accent)] text-[color:var(--on-primary)] hover:bg-[color:var(--accent-hover)] transition"
                >
                  Index Docs
                </button>
              </Show>
              <Show when={docIndex.building()}>
                <div class="mt-4 flex items-center gap-2 text-[12px] text-zinc-400">
                  <div class="w-3.5 h-3.5 border-2 border-[color:var(--accent)] border-t-transparent rounded-full animate-spin" />
                  Indexing in progress…
                </div>
              </Show>
            </div>
          </Show>

          {/* Doc table */}
          <Show when={docIndex.docs().length > 0}>
            <table class="w-full text-[12px]">
              <thead>
                <tr class="border-b border-[color:var(--border-subtle)] text-left">
                  <th class="px-5 py-2.5 text-[10px] font-semibold text-zinc-500 uppercase tracking-wider w-8" />
                  <th class="px-3 py-2.5 text-[10px] font-semibold text-zinc-500 uppercase tracking-wider">Name</th>
                  <th class="px-3 py-2.5 text-[10px] font-semibold text-zinc-500 uppercase tracking-wider text-right">Pages</th>
                  <th class="px-3 py-2.5 text-[10px] font-semibold text-zinc-500 uppercase tracking-wider text-right">Indexed</th>
                  <th class="px-5 py-2.5 text-[10px] font-semibold text-zinc-500 uppercase tracking-wider text-right">Status</th>
                </tr>
              </thead>
              <tbody>
                <For each={docIndex.docs()}>
                  {(doc) => {
                    const ext = docExt(doc.docPath);
                    return (
                      <tr class="border-b border-[color:var(--border-subtle)]/50 hover:bg-[color:var(--bg-surface)] transition-colors">
                        <td class="pl-5 pr-2 py-3">
                          <DocTypeTag ext={ext} />
                        </td>
                        <td class="px-3 py-3">
                          <span class="text-zinc-200 font-medium truncate max-w-[360px] block" title={doc.docPath}>
                            {docBasename(doc.docPath)}
                          </span>
                          <span class="text-[10px] text-zinc-600 truncate max-w-[360px] block" title={doc.docPath}>
                            {doc.docPath}
                          </span>
                        </td>
                        <td class="px-3 py-3 text-right text-zinc-400 font-mono tabular-nums">
                          {doc.pageCount}
                        </td>
                        <td class="px-3 py-3 text-right text-zinc-500 whitespace-nowrap">
                          {formatDate(doc.indexedAt)}
                        </td>
                        <td class="pl-3 pr-5 py-3 text-right">
                          <span class="inline-flex items-center gap-1 text-[11px] text-emerald-500">
                            <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                              <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                            </svg>
                            Indexed
                          </span>
                        </td>
                      </tr>
                    );
                  }}
                </For>
              </tbody>
            </table>
          </Show>
        </div>
      </div>

      {/* Build modal */}
      <Show when={showBuildModal()}>
        <div
          class="fixed inset-0 z-50 bg-black/60 backdrop-blur-[2px] flex items-center justify-center p-4"
          onClick={(e) => { if (e.target === e.currentTarget) setShowBuildModal(false); }}
        >
          <div class="w-full max-w-[440px] bg-[color:var(--bg-surface)] border border-[color:var(--border-default)] rounded-2xl shadow-[0_24px_64px_rgba(0,0,0,0.6)] flex flex-col overflow-hidden">
            <div class="px-6 pt-6 pb-4 border-b border-[color:var(--border-subtle)]">
              <div class="flex items-start justify-between gap-3">
                <div>
                  <h2 class="text-[15px] font-semibold text-zinc-100">
                    {isRebuild() ? 'Rebuild Index' : 'Index Documents'}
                  </h2>
                  <p class="text-[12px] text-zinc-500 mt-1 leading-relaxed">
                    {isRebuild()
                      ? 'Re-analyze all documents from scratch.'
                      : 'Scan the workspace and extract semantic labels per page.'}
                  </p>
                </div>
                <button
                  onClick={() => setShowBuildModal(false)}
                  class="w-7 h-7 rounded-lg flex items-center justify-center text-zinc-500 hover:text-zinc-200 hover:bg-[color:var(--bg-elevated)] transition shrink-0"
                >
                  <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            </div>

            <div class="px-3 py-2 max-h-[320px] overflow-y-auto">
              <Show when={docIndex.models().filter(m => m.enabled).length === 0}>
                <div class="px-3 py-8 text-center text-[12px] text-zinc-500">
                  No models available. Configure a provider in Settings.
                </div>
              </Show>
              <For each={[...new Set(docIndex.models().filter(m => m.enabled).map(m => m.providerId))]}>
                {(providerId) => {
                  const pModels = () => docIndex.models().filter(m => m.enabled && m.providerId === providerId);
                  const providerLabel: Record<string, string> = {
                    anthropic: 'Anthropic', openai: 'OpenAI', openrouter: 'OpenRouter',
                    google: 'Google', mistral: 'Mistral', ollama: 'Ollama',
                  };
                  const providerColor: Record<string, string> = {
                    anthropic: '#fb923c', openai: '#34d399', openrouter: '#a78bfa',
                    google: '#60a5fa', mistral: '#f43f5e', ollama: '#14b8a6',
                  };
                  return (
                    <div class="mb-1">
                      <div class="flex items-center gap-2 px-2 py-1.5">
                        <span class="w-1.5 h-1.5 rounded-full" style={{ background: providerColor[providerId] || '#71717a' }} />
                        <span class="text-[10px] font-semibold uppercase tracking-wider" style={{ color: providerColor[providerId] || '#71717a' }}>
                          {providerLabel[providerId] || providerId}
                        </span>
                      </div>
                      <For each={pModels()}>
                        {(model) => {
                          const isSel = () => docIndex.selectedModel() === model.id;
                          return (
                            <button
                              onClick={() => docIndex.selectModel(model.id)}
                              class={`w-full text-left px-3 py-2.5 rounded-lg mb-0.5 flex items-center justify-between gap-3 transition-colors
                                ${isSel() ? 'bg-[color:var(--accent-soft)] text-[color:var(--accent)]' : 'text-zinc-200 hover:bg-[color:var(--bg-elevated)]'}`}
                            >
                              <div class="flex items-center gap-2.5 min-w-0">
                                <div class={`w-4 h-4 rounded-full border-2 flex items-center justify-center shrink-0
                                  ${isSel() ? 'border-[color:var(--accent)] bg-[color:var(--accent)]' : 'border-zinc-600'}`}>
                                  <Show when={isSel()}>
                                    <svg class="w-2.5 h-2.5 text-white" fill="currentColor" viewBox="0 0 24 24">
                                      <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41L9 16.17z" />
                                    </svg>
                                  </Show>
                                </div>
                                <span class="text-[13px] font-medium truncate">{model.name}</span>
                                <Show when={model.default}>
                                  <span class="text-[9px] text-zinc-500 uppercase tracking-wider">default</span>
                                </Show>
                              </div>
                              <Show when={model.inputPricePerM > 0}>
                                <span class="text-[10px] text-zinc-500 font-mono">
                                  ${model.inputPricePerM % 1 === 0 ? model.inputPricePerM : model.inputPricePerM.toFixed(2)}
                                  <span class="text-zinc-700">/</span>
                                  ${model.outputPricePerM % 1 === 0 ? model.outputPricePerM : model.outputPricePerM.toFixed(2)}
                                </span>
                              </Show>
                            </button>
                          );
                        }}
                      </For>
                    </div>
                  );
                }}
              </For>
            </div>

            <div class="px-6 py-4 border-t border-[color:var(--border-subtle)] flex items-center justify-end gap-3">
              <button
                onClick={() => setShowBuildModal(false)}
                class="h-8 px-4 rounded-lg text-[12px] text-zinc-400 hover:text-zinc-200 hover:bg-[color:var(--bg-elevated)] transition"
              >
                Cancel
              </button>
              <button
                onClick={handleConfirmBuild}
                disabled={!docIndex.selectedModel()}
                class="h-8 px-4 rounded-lg text-[12px] font-medium bg-[color:var(--accent)] text-[color:var(--on-primary)] hover:bg-[color:var(--accent-hover)] disabled:opacity-50 disabled:cursor-not-allowed transition flex items-center gap-1.5"
              >
                <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M5 3l14 9-14 9V3z" />
                </svg>
                {isRebuild() ? 'Rebuild' : 'Index Docs'}
              </button>
            </div>
          </div>
        </div>
      </Show>
    </div>
  );
}
