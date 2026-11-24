// Writes server-go/internal/notices/texts.json, the messages the server sends
// to notification channels, from the "notices" of every language file. Run it
// after changing those texts; test/notice-texts.test.ts fails until then.
import { readdirSync, readFileSync, writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const KINDS = { requestUpload: "request.upload", storageFull: "storage.full", passwordReset: "signin.reset", test: "test" };
const locales = new URL("../lib/i18n/locales/", import.meta.url);
const out = new URL("../server-go/internal/notices/texts.json", import.meta.url);

export function noticeTexts() {
  const texts = {};
  for (const file of readdirSync(locales).filter((f) => f.endsWith(".json")).sort()) {
    const { notices } = JSON.parse(readFileSync(new URL(file, locales), "utf8"));
    texts[file.slice(0, -5)] = Object.fromEntries(Object.entries(KINDS).map(([key, kind]) => [kind, notices[key]]));
  }
  return JSON.stringify(texts, null, 1) + "\n";
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  writeFileSync(out, noticeTexts());
  console.log("wrote", fileURLToPath(out));
}
