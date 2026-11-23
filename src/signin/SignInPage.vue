<script setup lang="ts">
// The owner's way in: GlimStone's sign-in card on the kit's sign-in. The sign-in service that is on stands under
// Sign in, and passkeys come through the browser's offer in the password field.
import { SignInCard, type SignInCardService } from '@glimstone/vue';
import { signInCardHandlers, signInServiceLabel, takeSignInCode, type SignInOffer, type SignInRemoved } from '@glimstone/core';
import { IDENTITY_PROVIDERS } from '@glimstone/core/identityProviders';
import { computed, onMounted, ref } from 'vue';
import AppMark from '@/src/components/AppMark.vue';
import { config } from '@/src/config';
import { t } from '@/src/i18n';
import { leaveNote, signIn } from './kit';
import { arrivedWords, cardText, errorWords, resetNote } from './words';

const props = defineProps<{ note?: string }>();
const s = (key: string) => t('signin.' + key);

const offer = ref<SignInOffer | null>(null);
const code = takeSignInCode();
const refused = code ? ({ ...errorWords(), ...arrivedWords() } as Record<string, string | undefined>)[code] ?? s('failed') : undefined;

const labelKeys = { signIn: 'IconSignIn', forgot: 'IconForgotPassword', back: 'IconBack', setPassword: 'IconKey', service: 'IconSignIn' };
const first = computed(() => offer.value?.services[0]);
const service = computed<SignInCardService | undefined>(() => {
  const f = first.value;
  if (!f) return undefined;
  const { name, mark } = signInServiceLabel(f, IDENTITY_PROVIDERS, { services: { unnamed: s('tile.services.unnamed') } });
  return mark ? { name, mark } : { name };
});

onMounted(async () => {
  const r = await signIn.options();
  offer.value = r.ok ? r.value : { open: false, owner: true, minPassword: 12, signedIn: false, passkeys: false, services: [] };
});

const handlers = computed(() =>
  signInCardHandlers(signIn, {
    words: { errors: errorWords(), failed: s('failed') },
    onSignedIn: () => location.reload(),
    onReset: (removed: SignInRemoved) => {
      leaveNote(resetNote(removed));
      location.reload();
    },
  }),
);

function toService() {
  if (first.value) location.assign(signIn.serviceAddress(first.value.id, location.pathname));
}
</script>

<template>
  <SignInCard
    v-if="offer"
    class="cp-signin"
    :app-name="config.branding.appName"
    :text="cardText()"
    reset
    :min-password="offer.minPassword"
    :note="props.note"
    :error="refused"
    :service="service"
    :on-service="toService"
    :label-keys="labelKeys"
    id-prefix="cp-signin"
    :on-sign-in="handlers.onSignIn"
    :on-passkey-offer="offer.passkeys ? handlers.onPasskeyOffer : undefined"
    :on-ask-code="handlers.onAskCode"
    :on-set-password="handlers.onSetPassword"
    data-cp="signin"
  >
    <template #logo><AppMark :size="120" /></template>
  </SignInCard>
</template>
