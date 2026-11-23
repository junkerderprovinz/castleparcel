import { test, before } from "node:test";
import assert from "node:assert/strict";
import { createSSRApp, h, type Component } from "vue";
import { renderToString } from "vue/server-renderer";
import { account } from "../src/signin/state";
import { send } from "./upload-stub";
import { loadComponent } from "./vue-sfc";

let TopBar: Component;
let config: { uploadProtected: boolean };

before(async () => {
  const listen = () => {};
  Object.assign(globalThis, {
    document: { cookie: "fd_lang=en", documentElement: { lang: "", dir: "" }, addEventListener: listen, removeEventListener: listen },
    location: { pathname: "/", hash: "" },
    addEventListener: listen,
    window: { isSecureContext: true },
  });
  ({ config } = await import("../src/config"));
  TopBar = await loadComponent("src/components/TopBar.vue", {
    "@glimstone/vue": "test/glimstone-stub.ts",
    "@/src/upload": "test/upload-stub.ts",
    "@/src/steps": "test/steps-stub.ts",
    "@/src/signin/state": "src/signin/state.ts",
    "@/src/config": "src/config.ts",
  });
});

const render = (screen: string) => renderToString(createSSRApp({ render: () => h(TopBar, { screen }) }));
const backLink = /<a href="\/"[^>]*data-cp="back"[^>]*>Back<\/a>/;

test("the start page offers Settings and no way back", async () => {
  const html = await render("home");
  assert.doesNotMatch(html, /data-cp="back"/);
  assert.match(html, /data-cp="settings"/);
  assert.doesNotMatch(html, /CastleParcel/);
});

test("every other page leads back to the start page from the same place", async () => {
  for (const screen of ["note", "request", "r", "c", "m", "d", "missing"]) {
    const html = await render(screen);
    assert.match(html, backLink, screen);
    assert.ok(html.indexOf('data-cp="back"') < html.indexOf('data-cp="settings"'), `${screen}: back comes first`);
  }
});

test("Settings leads back without offering Settings again", async () => {
  const html = await render("settings");
  assert.match(html, /data-cp="back"[^>]*>Back</);
  assert.doesNotMatch(html, /data-cp="settings"/);
});

test("the options of a parcel lead back to the start page from the same place", async () => {
  Object.assign(send, { stage: "options", sent: null });
  try {
    const html = await render("home");
    assert.match(html, /data-cp="back"[^>]*>Back</);
    assert.ok(html.indexOf('data-cp="back"') < html.indexOf('data-cp="settings"'), "back comes first");
  } finally {
    Object.assign(send, { stage: "home", sent: null });
  }
});

test("a finished link offers Leave where Back sits on other pages", async () => {
  Object.assign(send, { stage: "done", sent: {} });
  try {
    const html = await render("home");
    assert.match(html, /data-cp="leave"[^>]*>Leave</);
    assert.doesNotMatch(html, /data-cp="back"/);
    assert.ok(html.indexOf('data-cp="leave"') < html.indexOf('data-cp="settings"'), "leave comes first");
  } finally {
    Object.assign(send, { stage: "home", sent: null });
  }
});

test("once the owner has a password, the pages that send offer the way in beside the guests' gate", async () => {
  Object.assign(account, { open: false, signedIn: false });
  config.uploadProtected = true;
  try {
    for (const screen of ["home", "note", "request"]) assert.match(await render(screen), /data-cp="signin-open"/, screen);
    for (const screen of ["d", "m", "r", "c", "settings"]) assert.doesNotMatch(await render(screen), /data-cp="signin-open"/, screen);
    config.uploadProtected = false;
    assert.doesNotMatch(await render("home"), /data-cp="signin-open"/, "the page shows the card itself");
  } finally {
    Object.assign(account, { open: true, signedIn: false });
    config.uploadProtected = false;
  }
});

test("the signed-in owner can sign out, and an open castle offers neither door", async () => {
  Object.assign(account, { open: false, signedIn: true });
  try {
    const html = await render("home");
    assert.match(html, /data-cp="signout"/);
    assert.doesNotMatch(html, /data-cp="signin-open"/);
    assert.doesNotMatch(await render("d"), /data-cp="signout"/);
  } finally {
    Object.assign(account, { open: true, signedIn: false });
  }
  const html = await render("home");
  assert.doesNotMatch(html, /data-cp="signin-open"|data-cp="signout"/);
});
