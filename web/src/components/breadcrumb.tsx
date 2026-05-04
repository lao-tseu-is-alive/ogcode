import { useNavigate } from '@solidjs/router';
import { For, Show } from 'solid-js';

interface BreadcrumbItem {
  label: string;
  href?: string;
}

interface BreadcrumbProps {
  items: BreadcrumbItem[];
}

export default function Breadcrumb(props: BreadcrumbProps) {
  const navigate = useNavigate();

  return (
    <nav class="flex items-center gap-1 text-[12px] min-w-0">
      <For each={props.items}>
        {(item, i) => (
          <>
            <Show when={i() > 0}>
              <svg class="w-3 h-3 text-zinc-600 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
              </svg>
            </Show>
            <Show
              when={item.href}
              fallback={<span class="text-zinc-400 truncate max-w-[200px]">{item.label}</span>}
            >
              <a
                href={item.href}
                onClick={(e) => { e.preventDefault(); navigate(item.href!); }}
                class="text-zinc-500 hover:text-zinc-200 transition truncate max-w-[200px]"
              >
                {item.label}
              </a>
            </Show>
          </>
        )}
      </For>
    </nav>
  );
}