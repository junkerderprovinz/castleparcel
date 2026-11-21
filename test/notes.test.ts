import { test } from "node:test";
import assert from "node:assert/strict";
import { notesBlocks } from "../lib/notes";

test("release notes become headings, lists and paragraphs", () => {
  const blocks = notesBlocks("Short intro\nover two lines.\n\n## ✨ Added\n\n- **Faster** uploads\n- A [guide](https://example.org/g)\n\n---\n\nThe end.");
  assert.deepEqual(blocks, [
    { kind: "paragraph", parts: [{ text: "Short intro over two lines." }] },
    { kind: "heading", parts: [{ text: "✨ Added" }] },
    {
      kind: "list",
      items: [
        [{ text: "Faster", bold: true }, { text: " uploads" }],
        [{ text: "A " }, { text: "guide", href: "https://example.org/g" }],
      ],
    },
    { kind: "paragraph", parts: [{ text: "The end." }] },
  ]);
});

test("backticks mark code", () => {
  assert.deepEqual(notesBlocks("Run `verify-client.mjs` first."), [
    { kind: "paragraph", parts: [{ text: "Run " }, { text: "verify-client.mjs", code: true }, { text: " first." }] },
  ]);
});

test("markup in the notes stays words and a script link is no link", () => {
  const [block] = notesBlocks('<img src=x onerror="alert(1)"> [click](javascript:void)');
  assert.deepEqual(block, {
    kind: "paragraph",
    parts: [{ text: '<img src=x onerror="alert(1)"> ' }, { text: "click" }],
  });
});
