import { For, Show, createEffect, on, onMount, onCleanup, createSignal } from 'solid-js';
import { usePlan } from '../context/plan';
import MessageItem from './message-item';
import { saveScroll, getScroll } from '../lib/scroll-memory';

function isToolResultMessage(msg: any): boolean {
  if (msg.info.role !== 'user') return false;
  const parts = msg.parts || [];
  return parts.length > 0 && parts.every((p: any) => p.type === 'tool');
}

function isEmptyInProgress(msg: any): boolean {
  if (msg.info.role !== 'assistant') return false;
  if (msg.info.finish || msg.info.error) return false;
  return (msg.parts || []).length === 0;
}

export default function PlanMessageList() {
  const plan = usePlan();
  let scrollRef: HTMLDivElement | undefined;
  let bottomAnchor: HTMLDivElement | undefined;
  let restored = false;
  const [isScrolledUp, setIsScrolledUp] = createSignal(false);
  const [unreadCount, setUnreadCount] = createSignal(0);
  const [stickToBottom, setStickToBottom] = createSignal(true);

  const scrollKey = () => {
    const id = plan.activePlan()?.id || '';
    return id ? `plan:${id}` : '';
  };

  const visibleMessages = () => {
    const activeId = plan.activePlan()?.sessionId;
    return plan.messages()
      .filter((msg: any) => msg.info.sessionId === activeId)
      .filter((msg: any) => !isToolResultMessage(msg) && !isEmptyInProgress(msg));
  };

  const checkNearBottom = () => {
    if (!scrollRef) return false;
    return scrollRef.scrollHeight - scrollRef.scrollTop - scrollRef.clientHeight < 80;
  };

  onMount(() => {
    if (!scrollRef) return;
    const handler = () => {
      if (!scrollRef) return;
      const key = scrollKey();
      if (key) saveScroll(key, scrollRef.scrollTop);
      const nearBottom = checkNearBottom();
      setIsScrolledUp(!nearBottom);
      setStickToBottom(nearBottom);
      if (nearBottom) setUnreadCount(0);
    };
    scrollRef.addEventListener('scroll', handler, { passive: true });
    onCleanup(() => scrollRef?.removeEventListener('scroll', handler));
  });

  createEffect(on(
    () => visibleMessages().length,
    (count) => {
      if (restored || !scrollRef || count === 0) return;
      const key = scrollKey();
      const saved = key ? getScroll(key) : 0;
      requestAnimationFrame(() => {
        if (!scrollRef) return;
        if (saved > 0) {
          scrollRef.scrollTop = saved;
        } else {
          bottomAnchor?.scrollIntoView({ behavior: 'instant' });
        }
        restored = true;
      });
    },
  ));

  createEffect(on(
    () => plan.activePlan()?.id,
    () => {
      restored = false;
      setStickToBottom(true);
      setIsScrolledUp(false);
      setUnreadCount(0);
    },
  ));

  createEffect(on(
    () => {
      const msgs = plan.messages();
      const last = msgs[msgs.length - 1];
      let tailMark = 0;
      if (last?.parts) {
        for (const p of last.parts) {
          if (p.updatedAt > tailMark) tailMark = p.updatedAt;
        }
      }
      const loadingKey = plan.loading() ? '1' : '0';
      return msgs.length + ':' + tailMark + ':' + loadingKey;
    },
    (_curr, prev) => {
      if (!scrollRef) return;
      if (prev === undefined && !restored) return;
      if (stickToBottom()) {
        requestAnimationFrame(() => {
          bottomAnchor?.scrollIntoView({ behavior: 'instant' });
        });
      } else {
        const count = visibleMessages().length;
        setUnreadCount(Math.max(0, count));
      }
    },
  ));

  createEffect(on(
    () => plan.loading(),
    (isStreaming) => {
      if (!isStreaming || !stickToBottom()) return;
      const id = setInterval(() => {
        if (stickToBottom() && bottomAnchor) {
          bottomAnchor.scrollIntoView({ behavior: 'smooth' });
        }
      }, 500);
      onCleanup(() => clearInterval(id));
    },
  ));

  const scrollToBottom = () => {
    if (bottomAnchor) bottomAnchor.scrollIntoView({ behavior: 'smooth' });
    setStickToBottom(true);
    setIsScrolledUp(false);
    setUnreadCount(0);
  };

  return (
    <div ref={scrollRef} class="flex-1 overflow-y-auto relative">
        <div class="max-w-3xl mx-auto px-4 md:px-6 py-6 space-y-6">
          <Show when={visibleMessages().length === 0 && !plan.loading()}>
            <div class="flex flex-col items-center justify-center py-24 text-center">
              <div class="w-14 h-14 rounded-2xl bg-[color:var(--accent-soft)] border border-[color:var(--border-subtle)] flex items-center justify-center mb-4">
                <svg class="w-6 h-6 text-[color:var(--accent)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.6">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
              </div>
              <p class="text-[14px] font-medium text-zinc-300 mb-1">Start planning</p>
              <p class="text-[12px] text-zinc-500">Describe your project or requirement to begin.</p>
            </div>
          </Show>

          <For each={visibleMessages()}>
            {(msg) => (
              <>
                <MessageItem msg={msg} />
                <Show when={msg.info.role === 'assistant' && msg.info.error}>
                  <div class="flex gap-3">
                    <div class="w-7 h-7 shrink-0 rounded-lg bg-red-500/20 border border-red-500/30 flex items-center justify-center">
                      <svg class="w-3.5 h-3.5 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.4">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v4m0 4h.01M12 3a9 9 0 100 18A9 9 0 0012 3z" />
                      </svg>
                    </div>
                    <div class="flex-1 min-w-0 py-1">
                      <p class="text-[13px] text-red-400 font-medium">Agent error</p>
                      <p class="text-[12px] text-red-400/70 mt-0.5 break-all">{msg.info.error}</p>
                    </div>
                  </div>
                </Show>
              </>
            )}
          </For>

          <Show when={plan.loading()}>
            <div class="flex gap-3 animate-fade-in">
              <div class="w-7 h-7 shrink-0 rounded-lg bg-[color:var(--accent)] flex items-center justify-center shadow-sm">
                <svg class="w-3.5 h-3.5 text-[color:var(--on-primary)]" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.4">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
              </div>
              <div class="flex items-center py-1.5">
                <div class="thinking-dots">
                  <span></span>
                  <span></span>
                  <span></span>
                </div>
              </div>
            </div>
          </Show>

          <div ref={bottomAnchor} />
        </div>

        <Show when={isScrolledUp()}>
          <button
            onClick={scrollToBottom}
            class="fixed bottom-24 left-1/2 -translate-x-1/2 flex items-center gap-2 px-4 py-2 rounded-lg bg-[color:var(--accent)] hover:bg-[color:var(--accent-hover)] text-[color:var(--on-primary)] text-sm font-medium shadow-lg transition-all animate-fade-in"
            title="Jump to latest message"
          >
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M19 14l-7 7m0 0l-7-7m7 7V3" />
            </svg>
            Jump to latest
            <Show when={unreadCount() > 0}>
              <span class="ml-1 px-2 py-0.5 bg-red-500 rounded-full text-xs font-bold">
                {unreadCount()}
              </span>
            </Show>
          </button>
        </Show>
      </div>
  );
}