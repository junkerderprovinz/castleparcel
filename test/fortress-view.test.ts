import { test } from "node:test";
import assert from "node:assert/strict";
import { fortressApps, fortressPermissions, permissionOf, statusLines, type FortressMember } from "../src/fortress/view";

const member = (over: Partial<FortressMember>): FortressMember => ({
  id: "0000000000000000000000000000000011112222",
  app: "bravo",
  name: "Bravo",
  version: "1.0.0",
  device: "Tower",
  deployment: "container",
  online: true,
  relay: false,
  pages: true,
  ...over,
});

test("the other apps come one card each, sorted by name, with their logo", () => {
  const apps = fortressApps(
    [
      member({ id: "a1", app: "zulu", name: "Zulu", icon: "data:image/svg+xml;base64,AA" }),
      member({ id: "b1" }),
      member({ id: "b2", device: "Laptop", deployment: "desktop", online: false, lastSeen: 1700000000 }),
    ],
    () => "yesterday",
  );
  assert.deepEqual(apps.map((a) => a.name), ["Bravo", "Zulu"]);
  assert.equal(apps[1]!.logo, "data:image/svg+xml;base64,AA");
  const away = apps[0]!.instances[1]!;
  assert.equal(away.presence, "away");
  assert.equal(away.lastSeen, "yesterday");
  assert.equal(away.deviceKind, "pc");
  assert.deepEqual(away.open, {});
});

test("CastleParcel itself is not among the other apps", () => {
  assert.deepEqual(fortressApps([member({ app: "castleparcel" })], () => ""), []);
});

test("a member that never said its app goes on a card of its own by its device, with nothing to open", () => {
  const [card] = fortressApps([member({ id: "x9", app: "", name: "", device: "Pi" })], () => "");
  assert.equal(card!.name, "Pi");
  assert.equal(card!.instances[0]!.open, undefined);
});

test("two members that claim one app on one device are in doubt", () => {
  const apps = fortressApps([member({ id: "b1" }), member({ id: "b2" })], () => "");
  assert.ok(apps[0]!.instances.every((i) => i.doubt));
});

test("a permission names the app it opens, and its id leads back to it", () => {
  const [p] = fortressPermissions(
    [{ from: "castleparcel", to: "bravo", action: "ui.open", on: false }],
    [member({})],
    (key, params) => `${key} ${params?.app}`,
  );
  assert.equal(p!.title, "fortress.uiOpen Bravo");
  assert.deepEqual(permissionOf(p!.id, true), { from: "castleparcel", to: "bravo", action: "ui.open", on: true });
});

test("a member's report and log are put in the reader's times", () => {
  const got = statusLines(
    {
      report: { state: "warning", needsYou: [{ severity: "warn", text: "Look" }], work: [{ title: "Upload" }], next: [] },
      log: [{ at: "2026-11-24T10:00:00Z", level: "info", text: "New parcel" }],
    },
    (iso) => `at ${iso}`,
  );
  assert.equal(got.report.state, "warning");
  assert.deepEqual(got.log, [{ at: "at 2026-11-24T10:00:00Z", level: "info", text: "New parcel" }]);
  assert.deepEqual(got.report.work, [{ title: "Upload" }]);
});
