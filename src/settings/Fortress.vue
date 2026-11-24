<script setup lang="ts">
// CastleParcel's place in the fortress. Inside one, the first card shows and changes the phrase behind a fresh
// sign-in and leaves; once a member says the phrase changed and nobody in reach hands over the new one, it asks
// for the words. The other apps follow with their permissions, and the network card picks the relay and whether
// to look on this network. CastleParcel runs as one instance, so the other apps always show.
import {
  fill, fortressText, openRefusalText, phraseFromText, phraseReady,
  type ChangeAnswer, type FortressAnswer, type FortressApp, type FortressInstance, type PhraseAnswer,
} from '@glimstone/core';
import { Button, Card, FieldRow, Fortress, FortressPairing, FortressStatusWindow, Selector, TextField, Toggle, Window } from '@glimstone/vue';
import { computed, ref, shallowRef, watch } from 'vue';
import { config } from '@/src/config';
import { FortressError, fortressApi } from '@/src/fortress/api';
import { fortressApps, fortressPermissions, permissionOf, SELF, statusLines, type FortressInfo, type MemberStatus } from '@/src/fortress/view';
import { language, locale, t } from '@/src/i18n';
import Gate from '@/src/screens/Gate.vue';
import SignInPage from '@/src/signin/SignInPage.vue';
import { account } from '@/src/signin/state';
import { ERROR_CODES, errorWords } from '@/src/signin/words';
import { toast } from '@/src/toast';
import { gate } from '@/src/upload';

const LABEL_KEYS = {
  join: 'IconSignIn', leave: 'IconSignOut', showPhrase: 'IconEye', changePhrase: 'IconRefresh',
  remove: 'IconTrash', unlock: 'IconKey', copy: 'IconCopy',
};

// Until the owner is signed in, the tab is the way in, as the Sign-in tab is. While CastleParcel is open with an
// upload password, that password comes first.
const out = computed(() => !account.open && !account.signedIn);
const guarded = computed(() => account.open && config.uploadProtected && !gate.token);
const info = shallowRef<FortressInfo | null>(null);
const text = computed(() => {
  const own = fortressText(language());
  return { ...own, pairing: own.fortress };
});
const lang = computed(() => locale());
const code = (e: unknown) => (e instanceof FortressError ? e.code : 'internal');

function refusal(e: unknown): string {
  const c = code(e);
  if (c === 'relayAddress') return t('fortress.relayAddressBad');
  if ((ERROR_CODES as readonly string[]).includes(c)) return errorWords()[c as keyof ReturnType<typeof errorWords>] ?? text.value.failed;
  return text.value.failed;
}

async function load() {
  try {
    info.value = await fortressApi.read();
    network.value = { relay: info.value.relay.mode, url: info.value.relay.url, lan: info.value.lan };
  } catch (e) {
    toast(refusal(e), 'fail');
  }
}
watch(() => !out.value && !guarded.value, (allowed) => allowed && void load(), { immediate: true });

const when = (iso: string) => new Date(iso).toLocaleString(lang.value, { dateStyle: 'medium', timeStyle: 'short' });
const ago = (unix: number) => when(new Date(unix * 1000).toISOString());
const apps = computed(() => fortressApps(info.value?.members ?? [], ago));
const permissions = computed(() => fortressPermissions(info.value?.permissions ?? [], info.value?.members ?? [], t));
const removed = computed(() => (info.value?.removed ?? []).map((r) => (r.app ? fill(text.value.memberOn, { app: r.app, device: r.device }) : r.device)));

// The server writes the QR code's text, which names an own relay after the words. GlimStone asks for it by them.
let shown: { words: string; code: string } | null = null;
const phraseQr = (words: string[]) => (shown?.words === words.join(' ') ? shown.code : words.join(' '));

async function answer(ask: () => Promise<FortressInfo>): Promise<FortressAnswer> {
  try {
    info.value = await ask();
    return { ok: true };
  } catch (e) {
    // GlimStone words the kit's own refusals of a join itself.
    const c = code(e);
    return c === 'phrase' || c === 'closedUntilNewPhrase' ? { code: c } : { error: refusal(e) };
  }
}
const onJoin = (words: string[]) => answer(() => fortressApi.join(words));
const onLeave = () => answer(fortressApi.leave);
const onRemove = (id: string) => answer(() => fortressApi.remove(id));
const onPermission = (id: string, on: boolean) => answer(() => fortressApi.permission(permissionOf(id, on)));

async function onShowPhrase(): Promise<PhraseAnswer> {
  try {
    const r = await fortressApi.showPhrase();
    shown = { words: r.words.join(' '), code: r.code };
    return { words: r.words };
  } catch (e) {
    return { error: refusal(e) };
  }
}
async function onChangePhrase(): Promise<ChangeAnswer> {
  try {
    const r = await fortressApi.changePhrase();
    shown = { words: r.words.join(' '), code: r.code };
    void load();
    return { words: r.words, left: r.left.map((l) => (l.app ? fill(text.value.memberOn, { app: l.app, device: l.device }) : l.device)) };
  } catch (e) {
    return { error: refusal(e) };
  }
}

// A member with pages opens them in a tab through the fortress. Any other, such as an Unraid plugin, shows its
// report in a window.
type Shown = { app: FortressApp; instance: FortressInstance; got?: MemberStatus; error?: string };
const status = ref<Shown | null>(null);
function onOpen(app: FortressApp, instance: FortressInstance) {
  const m = info.value?.members.find((x) => x.id === instance.id);
  if (m?.pages) {
    // Asked at once, since the browser lets a tab open only shortly after the click.
    fortressApi.open(instance.id).then(
      ({ path }) => window.open(path, '_blank', 'noopener'),
      (e) => toast(openRefusalText(text.value, code(e)), 'fail'),
    );
    return;
  }
  status.value = { app, instance };
  const keep = (change: Partial<Shown>) => {
    if (status.value?.instance.id === instance.id) status.value = { ...status.value, ...change };
  };
  fortressApi.status(instance.id, language()).then(
    (got) => keep({ got: statusLines(got, when) }),
    (e) => keep({ error: openRefusalText(text.value, code(e)) }),
  );
}

// The new words after a change this CastleParcel was away for.
const newWords = ref<string | null>(null);
const joining = ref(false);
const newError = ref('');
async function takeNewWords() {
  const words = phraseFromText(newWords.value ?? '');
  if (!phraseReady(words)) return;
  joining.value = true;
  const a = await onJoin(words);
  joining.value = false;
  if (a.ok) newWords.value = null;
  else newError.value = a.error ?? (a.code === 'phrase' ? text.value.joinRefused : text.value.failed);
}

const network = ref({ relay: 'project' as FortressInfo['relay']['mode'], url: '', lan: true });
const relays = computed(() => [
  { id: 'project' as const, label: t('fortress.relayProject') },
  { id: 'off' as const, label: t('fortress.relayOff') },
  { id: 'own' as const, label: t('fortress.relayOwn') },
]);
const saving = ref(false);
async function saveNetwork() {
  if (network.value.relay === 'own' && !network.value.url.trim()) return;
  saving.value = true;
  try {
    info.value = await fortressApi.network(network.value.relay, network.value.url, network.value.lan);
  } catch (e) {
    toast(refusal(e), 'fail');
  }
  saving.value = false;
}
function pickRelay(mode: FortressInfo['relay']['mode']) {
  network.value.relay = mode;
  if (mode !== 'own') void saveNetwork();
}
function pickLAN(on: boolean) {
  network.value.lan = on;
  void saveNetwork();
}
</script>

<template>
  <SignInPage v-if="out" />
  <Gate v-else-if="guarded" />
  <template v-else-if="info">
    <FortressPairing
      v-if="info.joined"
      :text="text"
      :lang="lang"
      :app="SELF"
      :on-show-phrase="onShowPhrase"
      :phrase-qr="phraseQr"
      :on-change-phrase="onChangePhrase"
      :on-leave="onLeave"
      :label-keys="LABEL_KEYS"
      :hue-index="0"
    >
      <div v-if="info.phraseChanged" class="glim-fortress-removed" data-fortress="moved" role="status">
        <span><strong>{{ text.changedTitle }}</strong> {{ fill(text.phraseMoved, { app: SELF.name }) }}</span>
        <Button :label="text.enterPhrase" label-key="IconSignIn" tone="subtle" data-fortress="open-new-words" @click="newWords = ''; newError = ''" />
      </div>
    </FortressPairing>
    <Fortress
      :text="text"
      :lang="lang"
      :app="SELF"
      :joined="info.joined"
      :apps="apps"
      :permissions="permissions"
      :on-permission="onPermission"
      :notify="toast"
      :on-join="onJoin"
      :on-remove="onRemove"
      :on-change-phrase="info.joined ? onChangePhrase : undefined"
      :removed="removed"
      :removed-here="info.removedHere"
      :show-others="true"
      hide-switch
      :on-open="onOpen"
      :label-keys="LABEL_KEYS"
      :hue-start="1"
    />
    <Card :title="t('fortress.network')" :hue-index="apps.length + 3">
      <FieldRow :label="t('fortress.relay')" :hint="t('fortress.relayHint')">
        <Selector :model-value="network.relay" :items="relays" inline activation="manual" @update:model-value="pickRelay" />
      </FieldRow>
      <FieldRow v-if="network.relay === 'own'" :label="t('fortress.relayAddress')">
        <TextField v-model="network.url" mono placeholder="https://relay.example" @keydown.enter="saveNetwork" />
        <Button :label="t('fortress.save')" label-key="IconSave" :busy="saving" @click="saveNetwork" />
      </FieldRow>
      <FieldRow :label="t('fortress.lan')" :hint="t('fortress.lanHint')">
        <Toggle :model-value="network.lan" @update:model-value="pickLAN" />
      </FieldRow>
    </Card>
    <FortressStatusWindow
      v-if="status"
      :app="status.app"
      :instance="status.instance"
      :report="status.got?.report"
      :log="status.got?.log"
      :loading="!status.got && !status.error"
      :error="status.error"
      :text="text"
      :lang="lang"
      :on-close="() => (status = null)"
    />
    <Window
      v-if="newWords !== null"
      :title="text.changedTitle"
      :text="{ cancel: text.cancel, close: text.close }"
      :close-on-backdrop="false"
      :actions="[{ label: text.join, labelKey: 'IconSignIn', primary: true, busy: joining, disabled: !phraseReady(phraseFromText(newWords)), onClick: () => { void takeNewWords(); return false; } }]"
      @close="newWords = null"
    >
      <p>{{ fill(text.phraseMoved, { app: SELF.name }) }}</p>
      <FieldRow :label="text.phraseTitle" :hint="text.phraseWhere">
        <TextField v-model="newWords" data-fortress="new-words" autocomplete="off" autocapitalize="none" spellcheck="false" @keydown.enter="takeNewWords" />
      </FieldRow>
      <p v-if="newError" class="glim-fortress-error" role="alert">{{ newError }}</p>
    </Window>
  </template>
</template>
