import { test } from "node:test";
import assert from "node:assert/strict";
import { isNewer, releaseState } from "../lib/release";

test("a release counts as newer by its numbers, not as text", () => {
  assert.equal(isNewer("v1.10.0", "1.9.3"), true);
  assert.equal(isNewer("v1.9.3", "1.9.3"), false);
  assert.equal(isNewer("v1.9.2", "1.9.3"), false);
  assert.equal(isNewer("v2.0", "1.99.99"), true);
  assert.equal(isNewer("v1.9.3", "1.9.3+main.abc1234"), false);
});

test("the card says a newer release is available and names it without its v", () => {
  assert.deepEqual(releaseState({ latest: "v1.5.0", checkedAt: "2026-11-21T10:00:00Z" }, "1.4.2"), { kind: "available", version: "1.5.0" });
});

test("the card says it is up to date and when the server asked", () => {
  assert.deepEqual(releaseState({ latest: "v1.4.2", checkedAt: "2026-11-21T10:00:00Z" }, "1.4.2"), {
    kind: "current",
    checkedAt: new Date("2026-11-21T10:00:00Z"),
  });
});

test("the card says the check failed when the server has no tag", () => {
  assert.deepEqual(releaseState({}, "1.4.2"), { kind: "failed" });
});
