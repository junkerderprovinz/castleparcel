import { test, before } from "node:test";
import assert from "node:assert/strict";
import { createSSRApp, h, type Component } from "vue";
import { renderToString } from "vue/server-renderer";
import { loadComponent } from "./vue-sfc";

const opened: unknown[] = [];
let Info: Component;

before(async () => {
  const listen = () => {};
  Object.assign(globalThis, {
    document: { cookie: "fd_lang=en", documentElement: { lang: "", dir: "", dataset: {} }, addEventListener: listen, removeEventListener: listen, createElement: () => ({}), querySelectorAll: () => [], body: {} },
    window: { open: (...args: unknown[]) => opened.push(args[0]), addEventListener: listen, removeEventListener: listen, matchMedia: () => ({ matches: false, addEventListener: listen, removeEventListener: listen }) },
    location: { pathname: "/settings/info", hash: "" },
    addEventListener: listen,
  });
  Info = await loadComponent("src/settings/Info.vue");
});

const render = () => renderToString(createSSRApp({ render: () => h(Info) }));

test("showing the Info tab opens no other window", async () => {
  const html = await render();
  assert.match(html, /CastleParcel/);
  assert.deepEqual(opened, []);
});

test("the Help card offers GitHub and Discord and no mail", async () => {
  const html = await render();
  assert.match(html, /open an issue on GitHub, or come say hi on Discord\./);
  assert.match(html, /aria-label="GitHub"/);
  assert.match(html, /aria-label="Discord"[^>]*>[\s\S]*?glim-discord-mark/);
  assert.doesNotMatch(html, /aria-label="Email"/);
});

test("the GlimStone version opens its newest release", async () => {
  const html = await render();
  assert.match(html, /href="https:\/\/github\.com\/junkerderprovinz\/glimstone\/releases\/latest"/);
});

test("the tile holds Version, Components, Help and About, in that order", async () => {
  const html = await render();
  const cards = [...html.matchAll(/data-glim="(Info\w+Card)"/g)].map((m) => m[1]);
  assert.deepEqual(cards, ["InfoVersionCard", "InfoComponentsCard", "InfoHelpCard", "InfoAboutCard"]);
});

test("the Version card shows CastleParcel alone, its number opening the newest release", async () => {
  const html = await render();
  const version = html.slice(html.indexOf('data-glim="InfoVersionCard"'), html.indexOf('data-glim="InfoComponentsCard"'));
  assert.match(version, /CastleParcel/);
  assert.doesNotMatch(version, /GlimStone/);
  assert.match(version, /href="https:\/\/github\.com\/junkerderprovinz\/castleparcel\/releases\/latest"/);
  assert.match(version, /What&#39;s new/);
  assert.match(version, /Check for updates/);
  assert.doesNotMatch(version, /How to update/);
});

test("the Components card lists GlimStone and offers no switch or button", async () => {
  const html = await render();
  const components = html.slice(html.indexOf('data-glim="InfoComponentsCard"'), html.indexOf('data-glim="InfoHelpCard"'));
  assert.match(components, /GlimStone/);
  assert.doesNotMatch(components, /<button|role="switch"/);
});

test("the About card is called About alone", async () => {
  const html = await render();
  const about = html.slice(html.indexOf('data-glim="InfoAboutCard"'));
  assert.doesNotMatch(about.slice(0, 600), /About CastleParcel/);
});
