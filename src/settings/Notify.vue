<script setup lang="ts">
// The services CastleParcel tells about files that arrive, a store that fills up and a reset through the log, as
// GlimStone's notification card draws them. Like the sign-in, they are the owner's, so a stranger meets the sign-in
// here once there is a password.
import { NotifyCard } from '@glimstone/vue';
import type { NotifyApi, NotifyConfig, NotifyKind } from '@glimstone/core';
import { notifyCardText } from '@glimstone/core/notifyCardTexts';
import { NOTIFY_PROVIDERS } from '@glimstone/core/notifyProviders';
import { computed } from 'vue';
import { config } from '@/src/config';
import { language, t } from '@/src/i18n';
import Gate from '@/src/screens/Gate.vue';
import SignInPage from '@/src/signin/SignInPage.vue';
import { account } from '@/src/signin/state';
import { toast } from '@/src/toast';
import { authHeaders, gate } from '@/src/upload';

// Unraid's notifications need the host, which a container does not reach, and every event pings no check.
const KINDS: NotifyKind[] = ['ntfy', 'gotify', 'discord', 'slack', 'matrix', 'smtp', 'apprise', 'webhook'];
const EVENTS = { 'request.upload': 'requestUpload', 'storage.full': 'storageFull', 'signin.reset': 'passwordReset' } as const;

async function call(method: string, path: string, body?: unknown) {
  const res = await fetch(path, {
    method,
    headers: { ...authHeaders(), ...(body === undefined ? {} : { 'Content-Type': 'application/json' }) },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  return { status: res.status, body: await res.json().catch(() => ({})) };
}

const api: NotifyApi = {
  async load() {
    const r = await call('GET', '/api/notify');
    if (r.status !== 200) throw new Error(String(r.status));
    return r.body as NotifyConfig;
  },
  async save(next) {
    const r = await call('PUT', '/api/notify', { ...next, lang: language() });
    return r.status === 200 ? { ok: true } : { ok: false, problems: r.body.problems };
  },
  async test(channel) {
    const r = await call('POST', '/api/notify/test', { channel });
    return r.body.ok === true ? { ok: true } : { ok: false, code: r.body.code };
  },
};

const text = computed(() => notifyCardText(language()));
const events = computed(() => Object.entries(EVENTS).map(([id, key]) => ({ id, label: t(`notices.${key}.label`) })));
const guarded = computed(() => account.open && config.uploadProtected && !gate.token);
</script>

<template>
  <div class="cp-notify-tab">
    <SignInPage v-if="!account.open && !account.signedIn" />
    <Gate v-else-if="guarded" />
    <NotifyCard
      v-else
      :api="api"
      :text="text"
      :app="config.branding.appName"
      :lang="language()"
      :providers="NOTIFY_PROVIDERS"
      :kinds="KINDS"
      :events="events"
      :notify="toast"
    />
  </div>
</template>
