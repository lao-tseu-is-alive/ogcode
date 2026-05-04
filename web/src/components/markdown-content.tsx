import { createMemo } from 'solid-js';
import { marked } from 'marked';
import { markedHighlight } from 'marked-highlight';
import hljs from 'highlight.js';
import DOMPurify from 'dompurify';

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
  const html = createMemo(() => {
    const raw = marked.parse(props.text, { async: false }) as string;
    return DOMPurify.sanitize(raw);
  });

  return <div class={`prose-chat break-words min-w-0 ${props.class ?? ''}`} innerHTML={html()} />;
}
