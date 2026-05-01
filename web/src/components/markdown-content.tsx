import { createMemo } from 'solid-js';
import { marked } from 'marked';
import DOMPurify from 'dompurify';

marked.setOptions({
  breaks: true,
  gfm: true,
});

export default function MarkdownContent(props: { text: string }) {
  const html = createMemo(() => {
    const raw = marked.parse(props.text, { async: false }) as string;
    return DOMPurify.sanitize(raw);
  });

  return <div class="prose-chat break-words min-w-0" innerHTML={html()} />;
}