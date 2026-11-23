<script setup lang="ts">
import { COFFEE_BUTTON_SVG, componentText, type UpdateState } from '@glimstone/core';
import { BrandMark, InfoAboutCard, InfoComponentsCard, InfoHelpCard, InfoVersionCard } from '@glimstone/vue';
import { computed, h, ref, type FunctionalComponent } from 'vue';
import { releaseState } from '@/lib/release';
import { glyph } from '@/src/glyphs';
import GiveWindow from '@/src/gs-gaps/GiveWindow.vue';
import ReleaseNotesWindow from '@/src/settings/ReleaseNotesWindow.vue';
import { language, t } from '@/src/i18n';
import { toast } from '@/src/toast';
import { windowWords } from '@/src/ui';
import { version } from '@/package.json';

const REPO = 'https://github.com/junkerderprovinz/castleparcel';
const DISCORD = 'https://discord.gg/R99DBA8Mh7';
const UPDATE_GUIDE = `${REPO}#updating`;

const words = computed(() => componentText(language()));

// The page may only talk to its own server, which asks GitHub on its behalf.
const update = ref<UpdateState>({ kind: 'unchecked' });
async function check() {
  update.value = { kind: 'checking' };
  try {
    const res = await fetch('/api/latest-release');
    update.value = releaseState(await res.json(), version);
  } catch {
    update.value = { kind: 'failed' };
  }
}

// While a newer release waits, its notes are what is new.
const notes = ref<string | null>(null);
const whatsNew = () => {
  notes.value = update.value.kind === 'available' ? update.value.version : version;
};

const svgMark = (svg: string): FunctionalComponent => () => h(BrandMark, { svg });
const coffeeArt = svgMark(COFFEE_BUTTON_SVG);

type Route = 'coffee' | 'paypal' | 'crypto';
const giving = ref<Route | null>(null);
const ROUTES: Record<Route, { title: string; text: string; url: string; glyph: string }> = {
  coffee: { title: 'info.coffeeButton', text: 'info.coffeeText', url: 'https://buymeacoffee.com/junkerderprovinz', glyph: 'IconCoffee' },
  paypal: { title: 'info.paypal', text: 'info.paypalText', url: 'https://www.paypal.com/donate/?hosted_button_id=76FVV52TKXTUS', glyph: 'IconPayPal' },
  crypto: { title: 'info.crypto', text: 'info.cryptoText', url: 'https://junkerderprovinz.github.io/junkerderprovinz/', glyph: 'IconBitcoin' },
};

function bugReport() {
  const report = JSON.stringify({ app: 'CastleParcel', version, userAgent: navigator.userAgent, secureContext: isSecureContext }, null, 2);
  const a = Object.assign(document.createElement('a'), { href: URL.createObjectURL(new Blob([report], { type: 'application/json' })), download: 'castleparcel-bug-report.json' });
  a.click();
  URL.revokeObjectURL(a.href);
  toast(t('info.bugSaved'));
}
</script>

<template>
  <InfoVersionCard :text="words.versionCard" name="CastleParcel" :version="version" :repo-url="REPO" route="guide" :state="update" :on-check="check" :guide-url="UPDATE_GUIDE" :on-whats-new="whatsNew" />
  <InfoComponentsCard :text="words.componentsCard" name="CastleParcel" :rows="[]" />
  <InfoHelpCard :repo-url="REPO" :repo-glyph="glyph('IconGithub')" :discord-url="DISCORD" :discord-glyph="glyph('IconDiscord')" :bug-glyph="glyph('IconDiagnostics')" :on-bug-report="bugReport"
    :text="{ title: t('info.help'), report: t('info.report'), repoButton: t('info.repo'), discordButton: t('info.discord'), mailButton: '', mailSubject: '', bugReport: t('info.bug'), bugReportSub: t('info.bugSub') }" />
  <InfoAboutCard :coffee-art="coffeeArt" :paypal-glyph="glyph('IconPayPal')" :crypto-glyph="glyph('IconBitcoin')"
    :on-coffee="() => (giving = 'coffee')" :on-paypal="() => (giving = 'paypal')" :on-crypto="() => (giving = 'crypto')"
    :text="{ title: t('info.about'), body: t('info.body'), coffee: t('info.coffee'), coffeeButton: t('info.coffeeButton'), paypalButton: t('info.paypal'), cryptoButton: t('info.crypto') }" />
  <ReleaseNotesWindow v-if="notes" :version="notes" :repo="REPO" @close="notes = null" />
  <GiveWindow v-if="giving" :title="t(ROUTES[giving].title)" :appeal="t('info.appeal')" :text="t(ROUTES[giving].text)" :go="t('info.open')" :url="ROUTES[giving].url" :glyph-key="ROUTES[giving].glyph" :words="windowWords()" @close="giving = null" />
</template>
