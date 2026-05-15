import { createMemo, createEffect } from 'solid-js';
import { marked } from 'marked';
import { markedHighlight } from 'marked-highlight';
import hljs from 'highlight.js';
import DOMPurify from 'dompurify';
import mermaid from 'mermaid';

mermaid.initialize({
  startOnLoad: false,
  theme: 'dark',
  securityLevel: 'strict',
});

marked.use(
  markedHighlight({
    emptyLangClass: 'hljs',
    langPrefix: 'hljs language-',
    highlight(code, lang) {
      const language = hljs.getLanguage(lang) ? lang : 'plaintext';
      return hljs.highlight(code, { language }).value;
    },
  })
);

marked.setOptions({
  breaks: true,
  gfm: true,
});

export default function MarkdownContent(props: { text: string; class?: string }) {
  let containerRef: HTMLDivElement | undefined;

  const html = createMemo(() => {
    const raw = marked.parse(props.text, { async: false }) as string;
    const wrapped = raw
      .replace(/<table\b([^>]*)>/g, '<div class="overflow-auto max-w-full"><table$1>')
      .replace(/<\/table>/g, '</table></div>')
      .replace(
        /<pre><code class="hljs language-mermaid">([\s\S]*?)<\/code><\/pre>/g,
        '<pre class="mermaid">$1</pre>'
      );
    return DOMPurify.sanitize(wrapped, { USE_PROFILES: { html: true, svg: true } });
  });

  createEffect(() => {
    const _ = html();
    if (!containerRef) return;
    const nodes = containerRef.querySelectorAll('.mermaid');
    if (nodes.length === 0) return;
    requestAnimationFrame(() => {
      mermaid.run({ nodes }).catch(() => {});
    });
  });

  return (
    <div
      ref={containerRef}
      class={`prose-chat break-words min-w-0 ${props.class ?? ''}`}
      innerHTML={html()}
    />
  );
}
