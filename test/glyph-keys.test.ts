import { test } from "node:test";
import assert from "node:assert/strict";
import { NOTIFY_CARD_KEYS } from "@glimstone/core";
import { glyphNameFor } from "../src/glyphs";

test("every glyph the notification card asks for has one", () => {
  for (const [what, key] of Object.entries(NOTIFY_CARD_KEYS)) assert.ok(glyphNameFor(key), `${what} (${key}) has no glyph`);
  assert.equal(glyphNameFor("mail"), "IconMail");
  assert.equal(glyphNameFor("link"), "IconLink");
});

test("a glyph's own name and an unknown key resolve as before", () => {
  assert.equal(glyphNameFor("IconTrash"), "IconTrash");
  assert.equal(glyphNameFor("version.check"), "IconSync");
  assert.equal(glyphNameFor("noSuchThing"), undefined);
});
