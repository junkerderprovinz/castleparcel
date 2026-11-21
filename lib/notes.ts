// Release notes as blocks the page draws itself, so nothing from the notes is
// ever parsed as HTML: headings, bullet lists and paragraphs, with **bold**,
// `code` and [links](https://...) inside them.

export type Inline = { text: string; bold?: boolean; code?: boolean; href?: string };
export type Block = { kind: "heading" | "paragraph"; parts: Inline[] } | { kind: "list"; items: Inline[][] };

const INLINE = /\[([^\]]+)\]\(([^)\s]+)\)|\*\*([^*]+)\*\*|`([^`]+)`/g;

function inline(text: string): Inline[] {
  const parts: Inline[] = [];
  let last = 0;
  for (const m of text.matchAll(INLINE)) {
    if (m.index > last) parts.push({ text: text.slice(last, m.index) });
    if (m[1] !== undefined) {
      // Only web and mail links become links; a javascript: one stays words.
      parts.push(/^(https?:|mailto:)/i.test(m[2]!) ? { text: m[1], href: m[2] } : { text: m[1] });
    } else if (m[3] !== undefined) {
      parts.push({ text: m[3], bold: true });
    } else {
      parts.push({ text: m[4]!, code: true });
    }
    last = m.index + m[0].length;
  }
  if (last < text.length) parts.push({ text: text.slice(last) });
  return parts;
}

export function notesBlocks(md: string): Block[] {
  const blocks: Block[] = [];
  let list: Inline[][] | null = null;
  let paragraph: string[] = [];
  const flush = () => {
    if (paragraph.length) blocks.push({ kind: "paragraph", parts: inline(paragraph.join(" ")) });
    paragraph = [];
    list = null;
  };
  for (const raw of md.replace(/\r\n/g, "\n").split("\n")) {
    const line = raw.trim();
    const heading = /^#{1,6}\s+(.*)$/.exec(line);
    const item = /^[-*]\s+(.*)$/.exec(line);
    if (!line || /^(-{3,}|\*{3,})$/.test(line)) {
      flush();
    } else if (heading) {
      flush();
      blocks.push({ kind: "heading", parts: inline(heading[1]!) });
    } else if (item) {
      if (paragraph.length) flush();
      if (!list) {
        list = [];
        blocks.push({ kind: "list", items: list });
      }
      list.push(inline(item[1]!));
    } else {
      list = null;
      paragraph.push(line);
    }
  }
  flush();
  return blocks;
}
