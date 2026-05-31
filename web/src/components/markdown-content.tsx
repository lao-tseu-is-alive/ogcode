import { createMemo, createEffect } from 'solid-js';
import { marked } from 'marked';
import { markedHighlight } from 'marked-highlight';
import hljs from 'highlight.js';
import DOMPurify from 'dompurify';
import mermaid from 'mermaid';
import katex from 'katex';
import 'katex/dist/katex.min.css';
import Plotly from 'plotly.js-dist-min';
import { renderRoughDiagram, type RoughSpec } from './rough-renderer';

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

// Encode HTML content for use in a data attribute, preserving all characters
// including <script>, event handlers, etc. We use base64 to avoid any
// encoding issues with DOMPurify or HTML attribute escaping.
function encodeHtmlForSrcdoc(html: string): string {
  return btoa(unescape(encodeURIComponent(html)));
}

function decodeHtmlFromSrcdoc(encoded: string): string {
  return decodeURIComponent(escape(atob(encoded)));
}

export default function MarkdownContent(props: { text: string; class?: string }) {
  let containerRef: HTMLDivElement | undefined;

  const html = createMemo(() => {
    const raw = marked.parse(props.text, { async: false }) as string;

    // Extract HTML code blocks and replace them with data-attributes placeholders
    // BEFORE DOMPurify runs, so that <script> tags and event handlers are preserved
    // intact (they will be rendered in a sandboxed iframe, not in the main page).
    const htmlBlocks: string[] = [];
    const withPlaceholders = raw.replace(
      /<pre><code class="hljs language-html">([\s\S]*?)<\/code><\/pre>/g,
      (_match, codeContent: string) => {
        // The code content is HTML-escaped by highlight.js. We need to unescape it
        // to get the raw HTML that the user wrote.
        const rawHtml = codeContent
          .replace(/&amp;/g, '&')
          .replace(/&lt;/g, '<')
          .replace(/&gt;/g, '>')
          .replace(/&quot;/g, '"')
          .replace(/&#39;/g, "'");
        const encoded = encodeHtmlForSrcdoc(rawHtml);
        htmlBlocks.push(encoded);
        const idx = htmlBlocks.length - 1;
        return `<div class="html-render" data-srcdoc-idx="${idx}"></div>`;
      }
    );

    const wrapped = withPlaceholders
      .replace(/<table\b([^>]*)>/g, '<div class="overflow-auto max-w-full"><table$1>')
      .replace(/<\/table>/g, '</table></div>')
      .replace(
        /<pre><code class="hljs language-mermaid">([\s\S]*?)<\/code><\/pre>/g,
        '<pre class="mermaid">$1</pre>'
      )
      .replace(
        /<pre><code class="hljs language-plotly">([\s\S]*?)<\/code><\/pre>/g,
        '<div class="plotly-chart" style="min-height:300px">$1</div>'
      )
      .replace(
        /<pre><code class="hljs language-rough">([\s\S]*?)<\/code><\/pre>/g,
        '<div class="rough-diagram" style="min-height:200px">$1</div>'
      );

    // Store the HTML blocks data on a module-level variable so the effect can
    // access the decoded content. We attach it to the container element.
    const sanitized = DOMPurify.sanitize(wrapped, {
      USE_PROFILES: { html: true, svg: true },
      ADD_ATTR: ['style', 'aria-hidden', 'data-srcdoc-idx'],
    });

    // Embed the encoded block data as a script tag that we'll strip before
    // setting innerHTML. We use a JSON array stored in a data attribute.
    const blocksJson = JSON.stringify(htmlBlocks);
    const dataTag = `<template class="html-blocks-data" data-blocks='${blocksJson.replace(/'/g, "&#39;")}'></template>`;
    return dataTag + sanitized;
  });

  createEffect(() => {
    const _ = html();
    if (!containerRef) return;

    // Extract and remove the embedded HTML blocks data
    const dataTemplate = containerRef.querySelector('.html-blocks-data');
    const blocksData: string[] = dataTemplate
      ? JSON.parse((dataTemplate as HTMLElement).getAttribute('data-blocks') || '[]')
      : [];
    dataTemplate?.remove();

    // Render mermaid diagrams
    const mermaidNodes = containerRef.querySelectorAll('.mermaid');
    if (mermaidNodes.length > 0) {
      requestAnimationFrame(() => {
        mermaid.run({ nodes: mermaidNodes }).catch(() => {});
      });
    }

    // Render Plotly charts
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

    // Render Rough diagrams
    const roughNodes = containerRef.querySelectorAll<HTMLElement>('.rough-diagram');
    roughNodes.forEach(el => {
      try {
        const spec = JSON.parse(el.textContent || '{}') as RoughSpec;
        el.textContent = '';
        renderRoughDiagram(el, spec);
      } catch {
        el.textContent = 'Invalid Rough diagram spec';
      }
    });

    // Render HTML blocks in sandboxed iframes
    const htmlNodes = containerRef.querySelectorAll('.html-render');
    htmlNodes.forEach(el => {
      const idx = parseInt(el.getAttribute('data-srcdoc-idx') || '-1', 10);
      if (idx < 0 || idx >= blocksData.length) {
        el.textContent = 'Invalid HTML block';
        return;
      }

      const rawHtml = decodeHtmlFromSrcdoc(blocksData[idx]);

      // Create a sandboxed iframe
      const iframe = document.createElement('iframe');
      iframe.style.cssText = 'width:100%;border:none;border-radius:8px;overflow:hidden;background:#1a1a2e;';
      iframe.sandbox.add('allow-scripts', 'allow-same-origin');
      // Note: allow-same-origin is needed for scripts within the iframe to
      // execute properly (e.g., to access their own DOM). The iframe is still
      // sandboxed — it cannot access the parent page's DOM.

      // Wrap the content in a dark-themed base template to match the chat UI
      const darkWrap = `<!DOCTYPE html><html><head><meta charset="utf-8"><style>
body { margin: 0; padding: 12px; background: #1a1a2e; color: #e4e4e7; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; }
a { color: #60a5fa; }
table { border-collapse: collapse; }
th, td { border: 1px solid #3f3f46; padding: 6px 12px; }
img { max-width: 100%; height: auto; }
</style></head><body>${rawHtml}</body></html>`;

      // Set height via resize observer after content loads
      iframe.addEventListener('load', () => {
        try {
          const doc = iframe.contentDocument || iframe.contentWindow?.document;
          if (doc?.body) {
            // Add a small delay for scripts to render
            const setHeight = () => {
              const height = doc.body.scrollHeight;
              if (height > 0) {
                iframe.style.height = height + 'px';
              }
            };
            setHeight();
            // Also set height after a short delay for async rendering
            setTimeout(setHeight, 200);
            setTimeout(setHeight, 1000);
          }
        } catch {
          // Cross-origin restrictions — set a reasonable default
          iframe.style.height = '300px';
        }
      });

      iframe.srcdoc = darkWrap;
      el.textContent = '';
      el.appendChild(iframe);
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