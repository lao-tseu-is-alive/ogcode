import { createMemo, createEffect } from 'solid-js';
import { marked } from 'marked';
import { markedHighlight } from 'marked-highlight';
import hljs from 'highlight.js';
import DOMPurify from 'dompurify';
import mermaid from 'mermaid';
import katex from 'katex';
import 'katex/dist/katex.min.css';
import Plotly from 'plotly.js-dist-min';

mermaid.initialize({
  startOnLoad: false,
  theme: 'dark',
  securityLevel: 'strict',
});

marked.use({
  extensions: [
    {
      name: 'blockMath',
      level: 'block' as const,
      start(src: string) { return src.indexOf('$$'); },
      tokenizer(src: string) {
        const match = /^\$\$([\s\S]+?)\$\$/.exec(src);
        if (match) return { type: 'blockMath', raw: match[0], text: match[1].trim() };
      },
      renderer(token: any) {
        try {
          return `<div class="math-block">${katex.renderToString(token.text, { displayMode: true, throwOnError: false, output: 'html' })}</div>\n`;
        } catch {
          return `<div class="math-block">${token.text}</div>\n`;
        }
      },
    },
    {
      name: 'inlineMath',
      level: 'inline' as const,
      start(src: string) { return src.indexOf('$'); },
      tokenizer(src: string) {
        const match = /^\$(?!\$)((?:[^$\n]|\\.)+?)\$/.exec(src);
        if (match) return { type: 'inlineMath', raw: match[0], text: match[1].trim() };
      },
      renderer(token: any) {
        try {
          return katex.renderToString(token.text, { displayMode: false, throwOnError: false, output: 'html' });
        } catch {
          return `<span class="math-inline">${token.text}</span>`;
        }
      },
    },
  ],
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
      )
      .replace(
        /<pre><code class="hljs language-plotly">([\s\S]*?)<\/code><\/pre>/g,
        '<div class="plotly-chart" style="min-height:300px">$1</div>'
      );
    return DOMPurify.sanitize(wrapped, { USE_PROFILES: { html: true, svg: true }, ADD_ATTR: ['style', 'aria-hidden'] });
  });

  createEffect(() => {
    const _ = html();
    if (!containerRef) return;
    const mermaidNodes = containerRef.querySelectorAll('.mermaid');
    if (mermaidNodes.length > 0) {
      requestAnimationFrame(() => {
        mermaid.run({ nodes: mermaidNodes }).catch(() => {});
      });
    }
    const plotlyNodes = containerRef.querySelectorAll<HTMLElement>('.plotly-chart');
    plotlyNodes.forEach(el => {
      try {
        const spec = JSON.parse(el.textContent || '{}');
        el.textContent = '';
        Plotly.newPlot(el, spec.data ?? [], spec.layout ?? {}, { responsive: true, displayModeBar: false, ...spec.config });
      } catch {
        el.textContent = 'Invalid Plotly spec';
      }
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
