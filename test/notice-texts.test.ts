import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { noticeTexts } from "../scripts/gen-notice-texts.mjs";

test("the server sends the notification texts the language files hold", () => {
  const embedded = readFileSync(new URL("../server-go/internal/notices/texts.json", import.meta.url), "utf8");
  assert.equal(embedded, noticeTexts(), "run node scripts/gen-notice-texts.mjs");
});
